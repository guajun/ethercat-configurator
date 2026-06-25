package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/model"
	"gopkg.in/yaml.v3"
)

var typeBitLengths = map[string]uint16{
	"bool":   1,
	"bit":    1,
	"uint8":  8,
	"int8":   8,
	"uint16": 16,
	"int16":  16,
	"uint32": 32,
	"int32":  32,
	"uint64": 64,
	"int64":  64,
}

type LoadResult struct {
	Device      model.Device
	Diagnostics []diag.Diagnostic
}

type rawDevice struct {
	Identity         rawIdentity    `yaml:"identity"`
	ProcessData      rawProcessData `yaml:"process_data"`
	ObjectDictionary []rawObject    `yaml:"object_dictionary"`
	PDOs             rawPDOSet      `yaml:"pdos"`
}

type rawIdentity struct {
	VendorID    any    `yaml:"vendor_id"`
	ProductCode any    `yaml:"product_code"`
	Revision    any    `yaml:"revision"`
	Name        string `yaml:"name"`
}

type rawProcessData struct {
	RX rawProcessDataRegion `yaml:"rx"`
	TX rawProcessDataRegion `yaml:"tx"`
}

type rawProcessDataRegion struct {
	Address    any    `yaml:"address"`
	Size       uint32 `yaml:"size"`
	BufferMode string `yaml:"buffer_mode"`
}

type rawObject struct {
	Index        any    `yaml:"index"`
	Subindex     any    `yaml:"subindex"`
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	BitLength    uint16 `yaml:"bit_length"`
	Access       string `yaml:"access"`
	DefaultValue string `yaml:"default"`
	PDO          string `yaml:"pdo"`
}

type rawPDOSet struct {
	RX []rawPDO `yaml:"rx"`
	TX []rawPDO `yaml:"tx"`
}

type rawPDO struct {
	Index   any           `yaml:"index"`
	Name    string        `yaml:"name"`
	Entries []rawPDOEntry `yaml:"entries"`
}

type rawPDOEntry struct {
	Index    any `yaml:"index"`
	Subindex any `yaml:"subindex"`
}

func LoadFile(path string) LoadResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{Diagnostics: []diag.Diagnostic{diag.Error(diag.CodeConfigParseError, err.Error())}}
	}
	return Load(data, path)
}

func Load(data []byte, path string) LoadResult {
	var raw rawDevice
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return LoadResult{Diagnostics: []diag.Diagnostic{{Code: diag.CodeConfigParseError, Severity: diag.SeverityError, Message: err.Error(), File: path}}}
	}

	builder := normalizer{path: path}
	device := model.Device{
		Identity: model.Identity{
			VendorID:    builder.uint32(raw.Identity.VendorID, "vendor_id"),
			ProductCode: builder.uint32(raw.Identity.ProductCode, "product_code"),
			Revision:    builder.uint32(raw.Identity.Revision, "revision"),
			Name:        raw.Identity.Name,
		},
		ProcessData: model.ProcessData{
			RX: builder.region(raw.ProcessData.RX),
			TX: builder.region(raw.ProcessData.TX),
		},
	}

	seenObjects := map[uint32]struct{}{}
	for _, rawObject := range raw.ObjectDictionary {
		entry := model.ObjectEntry{
			Index:        builder.uint16(rawObject.Index, "object index"),
			Subindex:     builder.uint8(rawObject.Subindex, "object subindex"),
			Name:         rawObject.Name,
			Type:         strings.ToLower(rawObject.Type),
			BitLength:    rawObject.BitLength,
			Access:       strings.ToLower(rawObject.Access),
			DefaultValue: rawObject.DefaultValue,
			PDO:          model.Direction(strings.ToLower(rawObject.PDO)),
		}
		if entry.BitLength == 0 {
			entry.BitLength = typeBitLengths[entry.Type]
		}
		builder.validateObject(entry, seenObjects)
		seenObjects[model.ObjectKey(entry.Index, entry.Subindex)] = struct{}{}
		device.ObjectDictionary = append(device.ObjectDictionary, entry)
	}

	device.PDOs.RX = builder.pdos(raw.PDOs.RX)
	device.PDOs.TX = builder.pdos(raw.PDOs.TX)

	return LoadResult{Device: device, Diagnostics: builder.diagnostics}
}

type normalizer struct {
	path        string
	diagnostics []diag.Diagnostic
}

func (builder *normalizer) region(raw rawProcessDataRegion) model.ProcessDataRegion {
	region := model.ProcessDataRegion{Size: raw.Size, BufferMode: builder.bufferMode(raw.BufferMode)}
	if raw.Address != nil {
		region.Address = builder.uint32(raw.Address, "process data address")
		region.Configured = true
	}
	return region
}

func (builder *normalizer) bufferMode(value string) model.BufferMode {
	switch strings.ToLower(value) {
	case "", "single":
		return model.BufferModeSingle
	case "double":
		return model.BufferModeDouble
	case "triple":
		return model.BufferModeTriple
	default:
		builder.add(diag.CodeConfigParseError, fmt.Sprintf("unsupported process data buffer_mode %q", value))
		return model.BufferModeSingle
	}
}

func (builder *normalizer) pdos(rawPDOs []rawPDO) []model.PDO {
	pdos := make([]model.PDO, 0, len(rawPDOs))
	for _, rawPDO := range rawPDOs {
		pdo := model.PDO{Index: builder.uint16(rawPDO.Index, "PDO index"), Name: rawPDO.Name}
		for _, rawEntry := range rawPDO.Entries {
			pdo.Entries = append(pdo.Entries, model.PDOEntry{Index: builder.uint16(rawEntry.Index, "PDO entry index"), Subindex: builder.uint8(rawEntry.Subindex, "PDO entry subindex")})
		}
		pdos = append(pdos, pdo)
	}
	return pdos
}

func (builder *normalizer) validateObject(entry model.ObjectEntry, seen map[uint32]struct{}) {
	if entry.Index < 0x1000 {
		builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("object index 0x%04x is outside the object dictionary range", entry.Index))
	}
	if _, ok := seen[model.ObjectKey(entry.Index, entry.Subindex)]; ok {
		builder.add(diag.CodeODDuplicateIndex, fmt.Sprintf("duplicate object 0x%04x:%02x", entry.Index, entry.Subindex))
	}
	if _, ok := typeBitLengths[entry.Type]; !ok {
		builder.add(diag.CodeODInvalidType, fmt.Sprintf("unsupported object type %q", entry.Type))
	}
	if entry.Access != "ro" && entry.Access != "rw" && entry.Access != "wo" {
		builder.add(diag.CodeODInvalidAccess, fmt.Sprintf("unsupported access flag %q", entry.Access))
	}
	if entry.PDO != "" && entry.PDO != model.DirectionRX && entry.PDO != model.DirectionTX {
		builder.add(diag.CodePDODirectionMismatch, fmt.Sprintf("unsupported PDO direction %q", entry.PDO))
	}
}

func (builder *normalizer) uint8(value any, label string) uint8 {
	number := builder.uint64(value, label)
	if number > 0xff {
		builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("%s %d exceeds 8 bits", label, number))
		return 0
	}
	return uint8(number)
}

func (builder *normalizer) uint16(value any, label string) uint16 {
	number := builder.uint64(value, label)
	if number > 0xffff {
		builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("%s %d exceeds 16 bits", label, number))
		return 0
	}
	return uint16(number)
}

func (builder *normalizer) uint32(value any, label string) uint32 {
	number := builder.uint64(value, label)
	if number > 0xffffffff {
		builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("%s %d exceeds 32 bits", label, number))
		return 0
	}
	return uint32(number)
}

func (builder *normalizer) uint64(value any, label string) uint64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case int:
		if typed < 0 {
			builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("%s must be unsigned", label))
			return 0
		}
		return uint64(typed)
	case uint64:
		return typed
	case string:
		number, err := strconv.ParseUint(typed, 0, 64)
		if err != nil {
			builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("%s %q is not a valid unsigned integer", label, typed))
			return 0
		}
		return number
	default:
		builder.add(diag.CodeODInvalidIndex, fmt.Sprintf("%s has unsupported numeric value %v", label, value))
		return 0
	}
}

func (builder *normalizer) add(code string, message string) {
	builder.diagnostics = append(builder.diagnostics, diag.Diagnostic{Code: code, Severity: diag.SeverityError, Message: message, File: builder.path})
}
