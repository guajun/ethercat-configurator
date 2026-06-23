package esi

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sort"

	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/model"
)

type Summary struct {
	Identity     IdentitySummary     `json:"identity"`
	Objects      []ObjectSummary     `json:"objects"`
	PDOs         []PDOSummary        `json:"pdos"`
	ProcessData  []ProcessDataRegion `json:"process_data"`
	SyncManagers []SyncManager       `json:"sync_managers"`
}

type IdentitySummary struct {
	VendorID    string `json:"vendor_id"`
	ProductCode string `json:"product_code"`
	Revision    string `json:"revision"`
	Name        string `json:"name"`
}

type ObjectSummary struct {
	Index        string `json:"index"`
	Subindex     string `json:"subindex"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	BitLength    uint16 `json:"bit_length"`
	Access       string `json:"access"`
	PDO          string `json:"pdo,omitempty"`
	DefaultValue string `json:"default,omitempty"`
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
	Name       string `json:"name"`
	Type       string `json:"type"`
	BitLength  uint16 `json:"bit_length"`
	BitOffset  uint32 `json:"bit_offset"`
	ByteOffset uint32 `json:"byte_offset"`
}

type ProcessDataRegion struct {
	Direction     string `json:"direction"`
	Address       string `json:"address"`
	RequiredBytes uint32 `json:"required_bytes"`
	BufferBytes   uint32 `json:"buffer_bytes"`
}

type SyncManager struct {
	Index        uint8  `json:"index"`
	Direction    string `json:"direction"`
	StartAddress string `json:"start_address"`
	Length       uint32 `json:"length"`
	PDOAssign    string `json:"pdo_assign"`
}

type etherCATInfo struct {
	XMLName      xml.Name        `xml:"EtherCATInfo"`
	Vendor       esiVendor       `xml:"Vendor"`
	Descriptions esiDescriptions `xml:"Descriptions"`
}

type esiVendor struct {
	ID   string `xml:"Id"`
	Name string `xml:"Name"`
}

type esiDescriptions struct {
	Devices esiDevices `xml:"Devices"`
}

type esiDevices struct {
	Device esiDevice `xml:"Device"`
}

type esiDevice struct {
	Type         string                 `xml:"Type"`
	Name         string                 `xml:"Name"`
	ProductCode  string                 `xml:"ProductCode"`
	Revision     string                 `xml:"RevisionNo"`
	ProcessData  []esiProcessDataRegion `xml:"ProcessData>Region"`
	Objects      []esiObject            `xml:"Objects>Object"`
	RxPDOs       []esiPDO               `xml:"RxPdo"`
	TxPDOs       []esiPDO               `xml:"TxPdo"`
	SyncManagers []esiSyncManager       `xml:"SyncManagers>SyncManager"`
}

type esiProcessDataRegion struct {
	Direction     string `xml:"Direction,attr"`
	Address       string `xml:"Address,attr"`
	RequiredBytes uint32 `xml:"RequiredBytes,attr"`
	BufferBytes   uint32 `xml:"BufferBytes,attr"`
}

type esiObject struct {
	Index        string `xml:"Index,attr"`
	Subindex     string `xml:"SubIndex,attr"`
	Name         string `xml:"Name,attr"`
	Type         string `xml:"Type,attr"`
	BitLength    uint16 `xml:"BitLength,attr"`
	Access       string `xml:"Access,attr"`
	PDO          string `xml:"Pdo,attr,omitempty"`
	DefaultValue string `xml:"Default,attr,omitempty"`
}

type esiPDO struct {
	Index   string        `xml:"Index,attr"`
	Name    string        `xml:"Name,attr"`
	Entries []esiPDOEntry `xml:"Entry"`
}

type esiPDOEntry struct {
	Index      string `xml:"Index,attr"`
	Subindex   string `xml:"SubIndex,attr"`
	Name       string `xml:"Name,attr"`
	Type       string `xml:"Type,attr"`
	BitLength  uint16 `xml:"BitLength,attr"`
	BitOffset  uint32 `xml:"BitOffset,attr"`
	ByteOffset uint32 `xml:"ByteOffset,attr"`
}

type esiSyncManager struct {
	Index        uint8  `xml:"Index,attr"`
	Direction    string `xml:"Direction,attr"`
	StartAddress string `xml:"StartAddress,attr"`
	Length       uint32 `xml:"Length,attr"`
	PDOAssign    string `xml:"PdoAssign,attr"`
}

func Generate(device model.Device, layout model.Layout, addressMap model.AddressMap) ([]byte, error) {
	document := buildDocument(device, layout, addressMap)
	var output bytes.Buffer
	output.WriteString(xml.Header)
	encoder := xml.NewEncoder(&output)
	encoder.Indent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	output.WriteByte('\n')
	return output.Bytes(), nil
}

func Parse(data []byte) (Summary, []diag.Diagnostic) {
	var document etherCATInfo
	if err := xml.Unmarshal(data, &document); err != nil {
		return Summary{}, []diag.Diagnostic{diag.Error(diag.CodeESIParseError, err.Error())}
	}
	return summarize(document), nil
}

func RenderText(writer io.Writer, summary Summary) error {
	_, err := fmt.Fprintf(writer, "ESI %s vendor=%s product=%s revision=%s objects=%d pdos=%d\n", summary.Identity.Name, summary.Identity.VendorID, summary.Identity.ProductCode, summary.Identity.Revision, len(summary.Objects), len(summary.PDOs))
	return err
}

func RenderJSON(writer io.Writer, summary Summary) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(summary)
}

func buildDocument(device model.Device, layout model.Layout, addressMap model.AddressMap) etherCATInfo {
	return etherCATInfo{
		Vendor: esiVendor{ID: hex32(device.Identity.VendorID), Name: device.Identity.Name},
		Descriptions: esiDescriptions{Devices: esiDevices{Device: esiDevice{
			Type:         device.Identity.Name,
			Name:         device.Identity.Name,
			ProductCode:  hex32(device.Identity.ProductCode),
			Revision:     hex32(device.Identity.Revision),
			ProcessData:  []esiProcessDataRegion{processData(addressMap.RX), processData(addressMap.TX)},
			Objects:      objects(device.ObjectDictionary),
			RxPDOs:       pdos(device.PDOs.RX, layout.RX),
			TxPDOs:       pdos(device.PDOs.TX, layout.TX),
			SyncManagers: []esiSyncManager{syncManager(2, addressMap.RX, "0x1c12"), syncManager(3, addressMap.TX, "0x1c13")},
		}}},
	}
}

func summarize(document etherCATInfo) Summary {
	device := document.Descriptions.Devices.Device
	summary := Summary{
		Identity: IdentitySummary{VendorID: document.Vendor.ID, ProductCode: device.ProductCode, Revision: device.Revision, Name: device.Name},
	}
	for _, object := range device.Objects {
		summary.Objects = append(summary.Objects, ObjectSummary{Index: object.Index, Subindex: object.Subindex, Name: object.Name, Type: object.Type, BitLength: object.BitLength, Access: object.Access, PDO: object.PDO, DefaultValue: object.DefaultValue})
	}
	for _, pdo := range device.RxPDOs {
		summary.PDOs = append(summary.PDOs, summarizePDO("rx", pdo))
	}
	for _, pdo := range device.TxPDOs {
		summary.PDOs = append(summary.PDOs, summarizePDO("tx", pdo))
	}
	for _, region := range device.ProcessData {
		summary.ProcessData = append(summary.ProcessData, ProcessDataRegion{Direction: region.Direction, Address: region.Address, RequiredBytes: region.RequiredBytes, BufferBytes: region.BufferBytes})
	}
	for _, syncManager := range device.SyncManagers {
		summary.SyncManagers = append(summary.SyncManagers, SyncManager{Index: syncManager.Index, Direction: syncManager.Direction, StartAddress: syncManager.StartAddress, Length: syncManager.Length, PDOAssign: syncManager.PDOAssign})
	}
	return summary
}

func summarizePDO(direction string, pdo esiPDO) PDOSummary {
	summary := PDOSummary{Direction: direction, Index: pdo.Index, Name: pdo.Name}
	for _, entry := range pdo.Entries {
		summary.Entries = append(summary.Entries, PDOEntrySummary{Index: entry.Index, Subindex: entry.Subindex, Name: entry.Name, Type: entry.Type, BitLength: entry.BitLength, BitOffset: entry.BitOffset, ByteOffset: entry.ByteOffset})
	}
	return summary
}

func processData(region model.AddressRegion) esiProcessDataRegion {
	return esiProcessDataRegion{Direction: string(region.Direction), Address: hex16(uint16(region.Address)), RequiredBytes: region.RequiredBytes, BufferBytes: region.BufferBytes}
}

func objects(entries []model.ObjectEntry) []esiObject {
	sorted := append([]model.ObjectEntry(nil), entries...)
	sort.SliceStable(sorted, func(leftIndex int, rightIndex int) bool {
		left := sorted[leftIndex]
		right := sorted[rightIndex]
		if left.Index != right.Index {
			return left.Index < right.Index
		}
		return left.Subindex < right.Subindex
	})
	objects := make([]esiObject, 0, len(sorted))
	for _, entry := range sorted {
		objects = append(objects, esiObject{Index: hex16(entry.Index), Subindex: hex8(entry.Subindex), Name: entry.Name, Type: entry.Type, BitLength: entry.BitLength, Access: entry.Access, PDO: string(entry.PDO), DefaultValue: entry.DefaultValue})
	}
	return objects
}

func pdos(pdos []model.PDO, layout model.LayoutDirection) []esiPDO {
	byPDO := map[uint16][]model.LayoutEntry{}
	for _, entry := range layout.Entries {
		byPDO[entry.PDOIndex] = append(byPDO[entry.PDOIndex], entry)
	}
	result := make([]esiPDO, 0, len(pdos))
	for _, pdo := range pdos {
		resultPDO := esiPDO{Index: hex16(pdo.Index), Name: pdo.Name}
		for _, entry := range byPDO[pdo.Index] {
			resultPDO.Entries = append(resultPDO.Entries, esiPDOEntry{Index: hex16(entry.Index), Subindex: hex8(entry.Subindex), Name: entry.Name, Type: entry.Type, BitLength: entry.BitLength, BitOffset: entry.BitOffset, ByteOffset: entry.ByteOffset})
		}
		result = append(result, resultPDO)
	}
	return result
}

func syncManager(index uint8, region model.AddressRegion, assign string) esiSyncManager {
	return esiSyncManager{Index: index, Direction: string(region.Direction), StartAddress: hex16(uint16(region.Address)), Length: region.BufferBytes, PDOAssign: assign}
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
