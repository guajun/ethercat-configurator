package sii

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"sort"

	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/model"
)

const magic = "ECSII001"

const (
	categoryIdentity uint16 = 1
	categoryStrings  uint16 = 2
	categoryObjects  uint16 = 3
	categoryPDOs     uint16 = 4
	categoryPDI      uint16 = 5
)

type Summary struct {
	Identity    IdentitySummary     `json:"identity"`
	Strings     []string            `json:"strings"`
	Objects     []ObjectSummary     `json:"objects"`
	PDOs        []PDOSummary        `json:"pdos"`
	ProcessData []ProcessDataRegion `json:"process_data"`
	Checksum    string              `json:"checksum"`
}

type IdentitySummary struct {
	VendorID    string `json:"vendor_id"`
	ProductCode string `json:"product_code"`
	Revision    string `json:"revision"`
	Name        string `json:"name"`
}

type ObjectSummary struct {
	Index     string `json:"index"`
	Subindex  string `json:"subindex"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	BitLength uint16 `json:"bit_length"`
	Access    string `json:"access"`
	PDO       string `json:"pdo,omitempty"`
}

type PDOSummary struct {
	Direction string            `json:"direction"`
	Index     string            `json:"index"`
	Name      string            `json:"name"`
	Entries   []PDOEntrySummary `json:"entries"`
}

type PDOEntrySummary struct {
	Index      string `json:"index"`
	Subindex   string `json:"subindex"`
	BitLength  uint16 `json:"bit_length"`
	BitOffset  uint32 `json:"bit_offset"`
	ByteOffset uint32 `json:"byte_offset"`
}

type ProcessDataRegion struct {
	Direction     string `json:"direction"`
	Address       string `json:"address"`
	RequiredBytes uint32 `json:"required_bytes"`
	BufferBytes   uint32 `json:"buffer_bytes"`
	BufferMode    string `json:"buffer_mode"`
}

type category struct {
	kind uint16
	data []byte
}

func Generate(device model.Device, layout model.Layout, addressMap model.AddressMap) ([]byte, error) {
	categories := []category{
		{kind: categoryIdentity, data: identityCategory(device.Identity)},
		{kind: categoryStrings, data: stringsCategory(device)},
		{kind: categoryObjects, data: objectsCategory(device.ObjectDictionary)},
		{kind: categoryPDOs, data: pdosCategory(device.PDOs, layout)},
		{kind: categoryPDI, data: pdiCategory(addressMap)},
	}

	var payload bytes.Buffer
	payload.WriteString(magic)
	_ = binary.Write(&payload, binary.LittleEndian, uint16(len(categories)))
	for _, category := range categories {
		_ = binary.Write(&payload, binary.LittleEndian, category.kind)
		_ = binary.Write(&payload, binary.LittleEndian, uint32(len(category.data)))
		payload.Write(category.data)
	}
	checksum := crc32.ChecksumIEEE(payload.Bytes())
	_ = binary.Write(&payload, binary.LittleEndian, checksum)
	return payload.Bytes(), nil
}

func Parse(data []byte) (Summary, []diag.Diagnostic) {
	if len(data) < len(magic)+2+4 || string(data[:len(magic)]) != magic {
		return Summary{}, []diag.Diagnostic{diag.Error(diag.CodeSIIParseError, "missing ECSII001 header")}
	}
	stored := binary.LittleEndian.Uint32(data[len(data)-4:])
	computed := crc32.ChecksumIEEE(data[:len(data)-4])
	var diagnostics []diag.Diagnostic
	if stored != computed {
		diagnostics = append(diagnostics, diag.Error(diag.CodeSIIChecksumInvalid, fmt.Sprintf("checksum 0x%08x does not match computed 0x%08x", stored, computed)))
	}

	reader := bytes.NewReader(data[len(magic) : len(data)-4])
	var count uint16
	if err := binary.Read(reader, binary.LittleEndian, &count); err != nil {
		return Summary{}, append(diagnostics, diag.Error(diag.CodeSIIParseError, err.Error()))
	}
	summary := Summary{Checksum: hex32(stored)}
	for index := 0; index < int(count); index++ {
		var kind uint16
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &kind); err != nil {
			return summary, append(diagnostics, diag.Error(diag.CodeSIIParseError, err.Error()))
		}
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			return summary, append(diagnostics, diag.Error(diag.CodeSIIParseError, err.Error()))
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return summary, append(diagnostics, diag.Error(diag.CodeSIIParseError, err.Error()))
		}
		applyCategory(&summary, kind, payload)
	}
	return summary, diagnostics
}

func RenderText(writer io.Writer, summary Summary) error {
	_, err := fmt.Fprintf(writer, "SII %s vendor=%s product=%s revision=%s objects=%d pdos=%d checksum=%s\n", summary.Identity.Name, summary.Identity.VendorID, summary.Identity.ProductCode, summary.Identity.Revision, len(summary.Objects), len(summary.PDOs), summary.Checksum)
	return err
}

func RenderJSON(writer io.Writer, summary Summary) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(summary)
}

func identityCategory(identity model.Identity) []byte {
	var output bytes.Buffer
	_ = binary.Write(&output, binary.LittleEndian, identity.VendorID)
	_ = binary.Write(&output, binary.LittleEndian, identity.ProductCode)
	_ = binary.Write(&output, binary.LittleEndian, identity.Revision)
	writeString(&output, identity.Name)
	return output.Bytes()
}

func stringsCategory(device model.Device) []byte {
	seen := map[string]struct{}{}
	values := []string{device.Identity.Name}
	for _, entry := range device.ObjectDictionary {
		values = append(values, entry.Name)
	}
	for _, pdo := range append(append([]model.PDO(nil), device.PDOs.RX...), device.PDOs.TX...) {
		values = append(values, pdo.Name)
	}
	var strings []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		strings = append(strings, value)
	}
	sort.Strings(strings)
	var output bytes.Buffer
	_ = binary.Write(&output, binary.LittleEndian, uint16(len(strings)))
	for _, value := range strings {
		writeString(&output, value)
	}
	return output.Bytes()
}

func objectsCategory(entries []model.ObjectEntry) []byte {
	sorted := append([]model.ObjectEntry(nil), entries...)
	sort.SliceStable(sorted, func(leftIndex int, rightIndex int) bool {
		left := sorted[leftIndex]
		right := sorted[rightIndex]
		if left.Index != right.Index {
			return left.Index < right.Index
		}
		return left.Subindex < right.Subindex
	})
	var output bytes.Buffer
	_ = binary.Write(&output, binary.LittleEndian, uint16(len(sorted)))
	for _, entry := range sorted {
		_ = binary.Write(&output, binary.LittleEndian, entry.Index)
		_ = binary.Write(&output, binary.LittleEndian, entry.Subindex)
		_ = binary.Write(&output, binary.LittleEndian, entry.BitLength)
		writeString(&output, entry.Name)
		writeString(&output, entry.Type)
		writeString(&output, entry.Access)
		writeString(&output, string(entry.PDO))
	}
	return output.Bytes()
}

func pdosCategory(pdos model.PDOSet, layout model.Layout) []byte {
	var output bytes.Buffer
	all := append(pdosForDirection(model.DirectionRX, pdos.RX, layout.RX), pdosForDirection(model.DirectionTX, pdos.TX, layout.TX)...)
	_ = binary.Write(&output, binary.LittleEndian, uint16(len(all)))
	for _, pdo := range all {
		writeString(&output, pdo.Direction)
		writeString(&output, pdo.Index)
		writeString(&output, pdo.Name)
		_ = binary.Write(&output, binary.LittleEndian, uint16(len(pdo.Entries)))
		for _, entry := range pdo.Entries {
			writeString(&output, entry.Index)
			writeString(&output, entry.Subindex)
			_ = binary.Write(&output, binary.LittleEndian, entry.BitLength)
			_ = binary.Write(&output, binary.LittleEndian, entry.BitOffset)
			_ = binary.Write(&output, binary.LittleEndian, entry.ByteOffset)
		}
	}
	return output.Bytes()
}

func pdiCategory(addressMap model.AddressMap) []byte {
	var output bytes.Buffer
	regions := []model.AddressRegion{addressMap.RX, addressMap.TX}
	_ = binary.Write(&output, binary.LittleEndian, uint16(len(regions)))
	for _, region := range regions {
		writeString(&output, string(region.Direction))
		_ = binary.Write(&output, binary.LittleEndian, region.Address)
		_ = binary.Write(&output, binary.LittleEndian, region.RequiredBytes)
		_ = binary.Write(&output, binary.LittleEndian, region.BufferBytes)
		writeString(&output, string(region.BufferMode))
	}
	return output.Bytes()
}

func applyCategory(summary *Summary, kind uint16, payload []byte) {
	reader := bytes.NewReader(payload)
	switch kind {
	case categoryIdentity:
		var vendorID uint32
		var productCode uint32
		var revision uint32
		_ = binary.Read(reader, binary.LittleEndian, &vendorID)
		_ = binary.Read(reader, binary.LittleEndian, &productCode)
		_ = binary.Read(reader, binary.LittleEndian, &revision)
		summary.Identity = IdentitySummary{VendorID: hex32(vendorID), ProductCode: hex32(productCode), Revision: hex32(revision), Name: readString(reader)}
	case categoryStrings:
		var count uint16
		_ = binary.Read(reader, binary.LittleEndian, &count)
		for index := 0; index < int(count); index++ {
			summary.Strings = append(summary.Strings, readString(reader))
		}
	case categoryObjects:
		var count uint16
		_ = binary.Read(reader, binary.LittleEndian, &count)
		for index := 0; index < int(count); index++ {
			var objectIndex uint16
			var subindex uint8
			var bitLength uint16
			_ = binary.Read(reader, binary.LittleEndian, &objectIndex)
			_ = binary.Read(reader, binary.LittleEndian, &subindex)
			_ = binary.Read(reader, binary.LittleEndian, &bitLength)
			summary.Objects = append(summary.Objects, ObjectSummary{Index: hex16(objectIndex), Subindex: hex8(subindex), BitLength: bitLength, Name: readString(reader), Type: readString(reader), Access: readString(reader), PDO: readString(reader)})
		}
	case categoryPDOs:
		var count uint16
		_ = binary.Read(reader, binary.LittleEndian, &count)
		for index := 0; index < int(count); index++ {
			pdo := PDOSummary{Direction: readString(reader), Index: readString(reader), Name: readString(reader)}
			var entryCount uint16
			_ = binary.Read(reader, binary.LittleEndian, &entryCount)
			for entryIndex := 0; entryIndex < int(entryCount); entryIndex++ {
				entry := PDOEntrySummary{Index: readString(reader), Subindex: readString(reader)}
				_ = binary.Read(reader, binary.LittleEndian, &entry.BitLength)
				_ = binary.Read(reader, binary.LittleEndian, &entry.BitOffset)
				_ = binary.Read(reader, binary.LittleEndian, &entry.ByteOffset)
				pdo.Entries = append(pdo.Entries, entry)
			}
			summary.PDOs = append(summary.PDOs, pdo)
		}
	case categoryPDI:
		var count uint16
		_ = binary.Read(reader, binary.LittleEndian, &count)
		for index := 0; index < int(count); index++ {
			var address uint32
			var requiredBytes uint32
			var bufferBytes uint32
			direction := readString(reader)
			_ = binary.Read(reader, binary.LittleEndian, &address)
			_ = binary.Read(reader, binary.LittleEndian, &requiredBytes)
			_ = binary.Read(reader, binary.LittleEndian, &bufferBytes)
			summary.ProcessData = append(summary.ProcessData, ProcessDataRegion{Direction: direction, Address: hex16(uint16(address)), RequiredBytes: requiredBytes, BufferBytes: bufferBytes, BufferMode: readString(reader)})
		}
	}
}

func pdosForDirection(direction model.Direction, pdos []model.PDO, layout model.LayoutDirection) []PDOSummary {
	byPDO := map[uint16][]model.LayoutEntry{}
	for _, entry := range layout.Entries {
		byPDO[entry.PDOIndex] = append(byPDO[entry.PDOIndex], entry)
	}
	summaries := make([]PDOSummary, 0, len(pdos))
	for _, pdo := range pdos {
		summary := PDOSummary{Direction: string(direction), Index: hex16(pdo.Index), Name: pdo.Name}
		for _, entry := range byPDO[pdo.Index] {
			summary.Entries = append(summary.Entries, PDOEntrySummary{Index: hex16(entry.Index), Subindex: hex8(entry.Subindex), BitLength: entry.BitLength, BitOffset: entry.BitOffset, ByteOffset: entry.ByteOffset})
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func writeString(writer io.Writer, value string) {
	_ = binary.Write(writer, binary.LittleEndian, uint16(len(value)))
	_, _ = io.WriteString(writer, value)
}

func readString(reader io.Reader) string {
	var length uint16
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return ""
	}
	data := make([]byte, length)
	_, _ = io.ReadFull(reader, data)
	return string(data)
}

func hex8(value uint8) string {
	return fmt.Sprintf("0x%02x", value)
}

func hex16(value uint16) string {
	return fmt.Sprintf("0x%04x", value)
}

func hex32(value uint32) string {
	return fmt.Sprintf("0x%08x", value)
}
