package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/model"
)

type Report struct {
	Target           string                  `json:"target"`
	Identity         Identity                `json:"identity"`
	ObjectDictionary ObjectDictionarySummary `json:"object_dictionary"`
	PDOs             []PDORow                `json:"pdos"`
	ProcessData      []ProcessDataRegion     `json:"process_data"`
	SyncManagers     []SyncManager           `json:"sync_managers"`
	Diagnostics      []diag.Diagnostic       `json:"diagnostics"`
}

type Identity struct {
	VendorID    string `json:"vendor_id"`
	ProductCode string `json:"product_code"`
	Revision    string `json:"revision"`
	Name        string `json:"name"`
}

type ObjectDictionarySummary struct {
	TotalEntries int             `json:"total_entries"`
	Entries      []ObjectSummary `json:"entries"`
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

type PDORow struct {
	Direction  string `json:"direction"`
	PDOIndex   string `json:"pdo_index"`
	PDOName    string `json:"pdo_name,omitempty"`
	Entry      string `json:"entry"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	ByteOffset uint32 `json:"byte_offset"`
	BitOffset  uint32 `json:"bit_offset"`
	BitLength  uint16 `json:"bit_length"`
	ByteLength uint32 `json:"byte_length"`
}

type ProcessDataRegion struct {
	Direction     string `json:"direction"`
	Address       string `json:"address"`
	RequiredBytes uint32 `json:"required_bytes"`
	BufferBytes   uint32 `json:"buffer_bytes"`
	BufferMode    string `json:"buffer_mode"`
	End           string `json:"end"`
	Source        string `json:"source"`
}

type SyncManager struct {
	Index         uint8  `json:"index"`
	Direction     string `json:"direction"`
	RequiredBytes uint32 `json:"required_bytes"`
	BufferBytes   uint32 `json:"buffer_bytes"`
	BufferMode    string `json:"buffer_mode"`
}

func Build(target string, device model.Device, layout model.Layout, addressMap model.AddressMap, diagnostics []diag.Diagnostic) Report {
	return Report{
		Target: target,
		Identity: Identity{
			VendorID:    hex32(device.Identity.VendorID),
			ProductCode: hex32(device.Identity.ProductCode),
			Revision:    hex32(device.Identity.Revision),
			Name:        device.Identity.Name,
		},
		ObjectDictionary: buildObjectDictionary(device.ObjectDictionary),
		PDOs:             buildPDORows(device.PDOs, layout),
		ProcessData:      buildProcessData(addressMap),
		SyncManagers:     buildSyncManagers(addressMap),
		Diagnostics:      diag.Sort(diagnostics),
	}
}

func RenderJSON(writer io.Writer, report Report) error {
	if report.Diagnostics == nil {
		report.Diagnostics = []diag.Diagnostic{}
	}
	encoder := json.NewEncoder(writer)
	return encoder.Encode(report)
}

func RenderMarkdown(writer io.Writer, report Report) error {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "# EtherCAT Configuration Report: %s\n\n", markdownText(report.Identity.Name))
	fmt.Fprintf(builder, "Target: `%s`\n\n", markdownText(report.Target))

	builder.WriteString("## Identity\n\n")
	builder.WriteString("| Field | Value |\n")
	builder.WriteString("| --- | --- |\n")
	fmt.Fprintf(builder, "| Vendor ID | `%s` |\n", report.Identity.VendorID)
	fmt.Fprintf(builder, "| Product Code | `%s` |\n", report.Identity.ProductCode)
	fmt.Fprintf(builder, "| Revision | `%s` |\n", report.Identity.Revision)
	fmt.Fprintf(builder, "| Name | %s |\n\n", markdownText(report.Identity.Name))

	builder.WriteString("## Object Dictionary Summary\n\n")
	fmt.Fprintf(builder, "Total entries: %d\n\n", report.ObjectDictionary.TotalEntries)
	builder.WriteString("| Index | Subindex | Name | Type | Bits | Access | PDO | Default |\n")
	builder.WriteString("| --- | --- | --- | --- | ---: | --- | --- | --- |\n")
	for _, entry := range report.ObjectDictionary.Entries {
		fmt.Fprintf(builder, "| `%s` | `%s` | %s | `%s` | %d | `%s` | `%s` | %s |\n", entry.Index, entry.Subindex, markdownText(entry.Name), entry.Type, entry.BitLength, entry.Access, entry.PDO, markdownText(entry.DefaultValue))
	}
	builder.WriteString("\n")

	builder.WriteString("## PDO Table\n\n")
	builder.WriteString("| Direction | PDO | PDO Name | Entry | Name | Type | Byte Offset | Bit Offset | Bits | Bytes |\n")
	builder.WriteString("| --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, row := range report.PDOs {
		fmt.Fprintf(builder, "| `%s` | `%s` | %s | `%s` | %s | `%s` | %d | %d | %d | %d |\n", row.Direction, row.PDOIndex, markdownText(row.PDOName), row.Entry, markdownText(row.Name), row.Type, row.ByteOffset, row.BitOffset, row.BitLength, row.ByteLength)
	}
	builder.WriteString("\n")

	builder.WriteString("## Process Data Address Regions\n\n")
	builder.WriteString("| Direction | Address | Required Bytes | Buffer Bytes | Buffer Mode | End | Source |\n")
	builder.WriteString("| --- | --- | ---: | ---: | --- | --- | --- |\n")
	for _, region := range report.ProcessData {
		fmt.Fprintf(builder, "| `%s` | `%s` | %d | %d | `%s` | `%s` | `%s` |\n", region.Direction, region.Address, region.RequiredBytes, region.BufferBytes, region.BufferMode, region.End, region.Source)
	}
	builder.WriteString("\n")

	builder.WriteString("## Sync Manager Sizes\n\n")
	builder.WriteString("| Sync Manager | Direction | Required Bytes | Buffer Bytes | Buffer Mode |\n")
	builder.WriteString("| ---: | --- | ---: | ---: | --- |\n")
	for _, syncManager := range report.SyncManagers {
		fmt.Fprintf(builder, "| %d | `%s` | %d | %d | `%s` |\n", syncManager.Index, syncManager.Direction, syncManager.RequiredBytes, syncManager.BufferBytes, syncManager.BufferMode)
	}
	builder.WriteString("\n")

	builder.WriteString("## Diagnostics\n\n")
	if len(report.Diagnostics) == 0 {
		builder.WriteString("No diagnostics.\n")
	} else {
		builder.WriteString("| Severity | Code | Location | Message |\n")
		builder.WriteString("| --- | --- | --- | --- |\n")
		for _, diagnostic := range report.Diagnostics {
			fmt.Fprintf(builder, "| `%s` | `%s` | %s | %s |\n", diagnostic.Severity, diagnostic.Code, markdownText(location(diagnostic)), markdownText(diagnostic.Message))
		}
	}

	_, err := io.WriteString(writer, builder.String())
	return err
}

func buildObjectDictionary(entries []model.ObjectEntry) ObjectDictionarySummary {
	sorted := append([]model.ObjectEntry(nil), entries...)
	sort.SliceStable(sorted, func(leftIndex int, rightIndex int) bool {
		left := sorted[leftIndex]
		right := sorted[rightIndex]
		if left.Index != right.Index {
			return left.Index < right.Index
		}
		return left.Subindex < right.Subindex
	})

	summary := ObjectDictionarySummary{TotalEntries: len(sorted), Entries: make([]ObjectSummary, 0, len(sorted))}
	for _, entry := range sorted {
		summary.Entries = append(summary.Entries, ObjectSummary{
			Index:        hex16(entry.Index),
			Subindex:     hex8(entry.Subindex),
			Name:         entry.Name,
			Type:         entry.Type,
			BitLength:    entry.BitLength,
			Access:       entry.Access,
			PDO:          string(entry.PDO),
			DefaultValue: entry.DefaultValue,
		})
	}
	return summary
}

func buildPDORows(pdos model.PDOSet, layout model.Layout) []PDORow {
	names := map[uint16]string{}
	for _, pdo := range append(append([]model.PDO(nil), pdos.RX...), pdos.TX...) {
		names[pdo.Index] = pdo.Name
	}

	rows := make([]PDORow, 0, len(layout.RX.Entries)+len(layout.TX.Entries))
	rows = append(rows, rowsForLayout(model.DirectionRX, layout.RX, names)...)
	rows = append(rows, rowsForLayout(model.DirectionTX, layout.TX, names)...)
	return rows
}

func rowsForLayout(direction model.Direction, layout model.LayoutDirection, names map[uint16]string) []PDORow {
	rows := make([]PDORow, 0, len(layout.Entries))
	for _, entry := range layout.Entries {
		rows = append(rows, PDORow{
			Direction:  string(direction),
			PDOIndex:   hex16(entry.PDOIndex),
			PDOName:    names[entry.PDOIndex],
			Entry:      fmt.Sprintf("%s:%s", hex16(entry.Index), hex8(entry.Subindex)),
			Name:       entry.Name,
			Type:       entry.Type,
			ByteOffset: entry.ByteOffset,
			BitOffset:  entry.BitOffset,
			BitLength:  entry.BitLength,
			ByteLength: entry.ByteLength,
		})
	}
	return rows
}

func buildProcessData(addressMap model.AddressMap) []ProcessDataRegion {
	return []ProcessDataRegion{processDataRegion(addressMap.RX), processDataRegion(addressMap.TX)}
}

func processDataRegion(region model.AddressRegion) ProcessDataRegion {
	source := "configured"
	if region.FromDefault {
		source = "default"
	}
	return ProcessDataRegion{
		Direction:     string(region.Direction),
		Address:       hex16(uint16(region.Address)),
		RequiredBytes: region.RequiredBytes,
		BufferBytes:   region.BufferBytes,
		BufferMode:    string(region.BufferMode),
		End:           hex16(uint16(region.End)),
		Source:        source,
	}
}

func buildSyncManagers(addressMap model.AddressMap) []SyncManager {
	return []SyncManager{
		{Index: 2, Direction: string(addressMap.RX.Direction), RequiredBytes: addressMap.RX.RequiredBytes, BufferBytes: addressMap.RX.BufferBytes, BufferMode: string(addressMap.RX.BufferMode)},
		{Index: 3, Direction: string(addressMap.TX.Direction), RequiredBytes: addressMap.TX.RequiredBytes, BufferBytes: addressMap.TX.BufferBytes, BufferMode: string(addressMap.TX.BufferMode)},
	}
}

func markdownText(value string) string {
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, "|", "\\|")
}

func location(diagnostic diag.Diagnostic) string {
	if diagnostic.File == "" {
		return ""
	}
	if diagnostic.Line == 0 {
		return diagnostic.File
	}
	if diagnostic.Column == 0 {
		return fmt.Sprintf("%s:%d", diagnostic.File, diagnostic.Line)
	}
	return fmt.Sprintf("%s:%d:%d", diagnostic.File, diagnostic.Line, diagnostic.Column)
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
