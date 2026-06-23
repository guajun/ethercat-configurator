package pdo

import (
	"fmt"

	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/model"
)

func BuildLayout(device model.Device) (model.Layout, []diag.Diagnostic) {
	objects := map[uint32]model.ObjectEntry{}
	for _, object := range device.ObjectDictionary {
		objects[model.ObjectKey(object.Index, object.Subindex)] = object
	}

	rx, rxDiagnostics := buildDirection(model.DirectionRX, device.PDOs.RX, objects)
	tx, txDiagnostics := buildDirection(model.DirectionTX, device.PDOs.TX, objects)
	return model.Layout{RX: rx, TX: tx}, append(rxDiagnostics, txDiagnostics...)
}

func buildDirection(direction model.Direction, pdos []model.PDO, objects map[uint32]model.ObjectEntry) (model.LayoutDirection, []diag.Diagnostic) {
	layout := model.LayoutDirection{Direction: direction}
	var diagnostics []diag.Diagnostic
	seenMappings := map[uint32]struct{}{}

	for _, pdo := range pdos {
		for _, pdoEntry := range pdo.Entries {
			key := model.ObjectKey(pdoEntry.Index, pdoEntry.Subindex)
			object, ok := objects[key]
			if !ok {
				diagnostics = append(diagnostics, diag.Error(diag.CodePDOEntryNotFound, fmt.Sprintf("PDO entry 0x%04x:%02x not found", pdoEntry.Index, pdoEntry.Subindex)))
				continue
			}
			if _, ok := seenMappings[key]; ok {
				diagnostics = append(diagnostics, diag.Error(diag.CodePDODuplicateMapping, fmt.Sprintf("duplicate PDO mapping for 0x%04x:%02x", pdoEntry.Index, pdoEntry.Subindex)))
				continue
			}
			seenMappings[key] = struct{}{}
			if object.PDO != "" && object.PDO != direction {
				diagnostics = append(diagnostics, diag.Error(diag.CodePDODirectionMismatch, fmt.Sprintf("object 0x%04x:%02x belongs to %s PDO but is mapped in %s PDO", object.Index, object.Subindex, object.PDO, direction)))
				continue
			}
			if direction == model.DirectionRX && object.Access == "ro" {
				diagnostics = append(diagnostics, diag.Error(diag.CodePDODirectionMismatch, fmt.Sprintf("RX PDO object 0x%04x:%02x must be writable", object.Index, object.Subindex)))
				continue
			}
			if direction == model.DirectionTX && object.Access == "wo" {
				diagnostics = append(diagnostics, diag.Error(diag.CodePDODirectionMismatch, fmt.Sprintf("TX PDO object 0x%04x:%02x must be readable", object.Index, object.Subindex)))
				continue
			}
			if !supportedBitWidth(object.BitLength) {
				diagnostics = append(diagnostics, diag.Error(diag.CodePDOUnsupportedWidth, fmt.Sprintf("object 0x%04x:%02x uses unsupported bit width %d", object.Index, object.Subindex, object.BitLength)))
				continue
			}

			layout.Entries = append(layout.Entries, model.LayoutEntry{
				Index:      object.Index,
				Subindex:   object.Subindex,
				Name:       object.Name,
				Type:       object.Type,
				BitOffset:  layout.BitLength,
				ByteOffset: layout.BitLength / 8,
				BitLength:  object.BitLength,
				ByteLength: uint32(object.BitLength+7) / 8,
				PDOIndex:   pdo.Index,
			})
			layout.BitLength += uint32(object.BitLength)
		}
	}

	layout.ByteLength = (layout.BitLength + 7) / 8
	padding := layout.ByteLength*8 - layout.BitLength
	layout.PaddingBits = uint8(padding)
	return layout, diagnostics
}

func supportedBitWidth(bitLength uint16) bool {
	return bitLength == 1 || bitLength == 2 || bitLength == 4 || bitLength%8 == 0
}

func BuildAddressMap(device model.Device, layout model.Layout) (model.AddressMap, []diag.Diagnostic) {
	addressMap := model.AddressMap{
		RX: region(model.DirectionRX, device.ProcessData.RX, layout.RX.ByteLength),
		TX: region(model.DirectionTX, device.ProcessData.TX, layout.TX.ByteLength),
	}
	return addressMap, validateAddressMap(addressMap)
}

func region(direction model.Direction, configured model.ProcessDataRegion, requiredBytes uint32) model.AddressRegion {
	address := configured.Address
	fromDefault := false
	if !configured.Configured {
		fromDefault = true
		if direction == model.DirectionRX {
			address = 0x1000
		} else {
			address = 0x1100
		}
	}
	bufferBytes := configured.Size
	if bufferBytes == 0 {
		bufferBytes = 0x0100
	}
	return model.AddressRegion{Direction: direction, Address: address, RequiredBytes: requiredBytes, BufferBytes: bufferBytes, End: address + bufferBytes, FromDefault: fromDefault}
}

func validateAddressMap(addressMap model.AddressMap) []diag.Diagnostic {
	regions := []model.AddressRegion{addressMap.RX, addressMap.TX}
	var diagnostics []diag.Diagnostic
	for _, region := range regions {
		if region.Address%8 != 0 {
			diagnostics = append(diagnostics, diag.Error(diag.CodePDIAddressInvalid, fmt.Sprintf("%s process data address 0x%04x must be 8-byte aligned", region.Direction, region.Address)))
		}
		if region.Address < 0x1000 || region.End > 0x2000 || region.End < region.Address {
			diagnostics = append(diagnostics, diag.Error(diag.CodePDILayoutOutOfRange, fmt.Sprintf("%s process data region 0x%04x-0x%04x is outside ESC process memory", region.Direction, region.Address, region.End)))
		}
		if region.RequiredBytes > region.BufferBytes {
			diagnostics = append(diagnostics, diag.Error(diag.CodePDIBufferTooSmall, fmt.Sprintf("%s process data requires %d bytes but buffer has %d", region.Direction, region.RequiredBytes, region.BufferBytes)))
		}
		if region.Address+region.RequiredBytes > region.End {
			diagnostics = append(diagnostics, diag.Error(diag.CodePDILayoutOutOfRange, fmt.Sprintf("%s process data layout ends at 0x%04x outside buffer end 0x%04x", region.Direction, region.Address+region.RequiredBytes, region.End)))
		}
	}
	if overlaps(addressMap.RX, addressMap.TX) {
		diagnostics = append(diagnostics, diag.Error(diag.CodePDIRegionOverlap, fmt.Sprintf("RX process data region 0x%04x-0x%04x overlaps TX region 0x%04x-0x%04x", addressMap.RX.Address, addressMap.RX.End, addressMap.TX.Address, addressMap.TX.End)))
	}
	return diagnostics
}

func overlaps(left model.AddressRegion, right model.AddressRegion) bool {
	return left.Address < right.End && right.Address < left.End
}
