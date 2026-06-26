package firmware

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/guajun/ethercat-configurator/internal/model"
)

var identifierPattern = regexp.MustCompile(`[^A-Z0-9]+`)

type HeaderWarning struct {
	Direction string
	Field     string
	Offset    uint32
	Bytes     uint32
	Message   string
}

func GenerateHeader(device model.Device, layout model.Layout, addressMap model.AddressMap) ([]byte, error) {
	data, _, err := GenerateHeaderWithWarnings(device, layout, addressMap)
	return data, err
}

func GenerateHeaderWithWarnings(device model.Device, layout model.Layout, addressMap model.AddressMap) ([]byte, []HeaderWarning, error) {
	prefix := macroPrefix(device.Identity.Name)
	var output bytes.Buffer
	fmt.Fprintf(&output, "#ifndef %s_ETHERCAT_DEVICE_H\n", prefix)
	fmt.Fprintf(&output, "#define %s_ETHERCAT_DEVICE_H\n\n", prefix)
	output.WriteString("#include <stdint.h>\n\n")
	writeProcessDataSizes(&output, layout)
	warnings := append(writeEasyCATBuffer(&output, "OUT", layout.RX), writeEasyCATBuffer(&output, "IN", layout.TX)...)
	fmt.Fprintf(&output, "#endif /* %s_ETHERCAT_DEVICE_H */\n", prefix)
	return output.Bytes(), warnings, nil
}

func writeProcessDataSizes(output *bytes.Buffer, layout model.Layout) {
	fmt.Fprintf(output, "#define CUST_BYTE_NUM_OUT\t%d\n", layout.RX.ByteLength)
	fmt.Fprintf(output, "#define CUST_BYTE_NUM_IN\t%d\n", layout.TX.ByteLength)
	fmt.Fprintf(output, "#define TOT_BYTE_NUM_ROUND_OUT\t%d\n", layout.RX.ByteLength)
	fmt.Fprintf(output, "#define TOT_BYTE_NUM_ROUND_IN\t%d\n\n\n", layout.TX.ByteLength)
}

func writeEasyCATBuffer(output *bytes.Buffer, direction string, layout model.LayoutDirection) []HeaderWarning {
	fmt.Fprintf(output, "typedef union\n{\n")
	fmt.Fprintf(output, "\tuint8_t Byte[TOT_BYTE_NUM_ROUND_%s];\n", direction)
	fmt.Fprintf(output, "\tstruct\n\t{\n")
	warnings := headerWarnings(direction, layout)
	byteCursor := uint32(0)
	for entryIndex, entry := range layout.Entries {
		if entry.BitOffset%8 != 0 || entry.BitLength%8 != 0 {
			continue
		}
		if entry.ByteOffset > byteCursor {
			fmt.Fprintf(output, "\t\tuint8_t reserved_%02d[%d];\n", entryIndex, entry.ByteOffset-byteCursor)
			byteCursor = entry.ByteOffset
		}
		fmt.Fprintf(output, "\t\t%s %s;\n", cType(entry.Type), fieldName(entry.Name))
		byteCursor += entry.ByteLength
	}
	if layout.ByteLength > byteCursor {
		fmt.Fprintf(output, "\t\tuint8_t reserved_tail[%d];\n", layout.ByteLength-byteCursor)
	}
	if layout.ByteLength == 0 {
		output.WriteString("\t\tuint8_t unused;\n")
	}
	fmt.Fprintf(output, "\t} Cust;\n")
	fmt.Fprintf(output, "} PROCBUFFER_%s;\n\n\n", direction)
	return warnings
}

func headerWarnings(direction string, layout model.LayoutDirection) []HeaderWarning {
	entries := append([]model.LayoutEntry(nil), layout.Entries...)
	sort.SliceStable(entries, func(leftIndex int, rightIndex int) bool {
		return entries[leftIndex].BitOffset < entries[rightIndex].BitOffset
	})
	var warnings []HeaderWarning
	byteCursor := uint32(0)
	for entryIndex, entry := range entries {
		if entry.BitOffset%8 != 0 || entry.BitLength%8 != 0 {
			warnings = append(warnings, HeaderWarning{Direction: direction, Field: fieldName(entry.Name), Offset: entry.BitOffset / 8, Bytes: entry.ByteLength, Message: fmt.Sprintf("%s field %s uses bit-level layout and is omitted from EasyCAT-style struct; access it through Byte[]", direction, fieldName(entry.Name))})
			continue
		}
		if entry.ByteOffset > byteCursor {
			warnings = append(warnings, HeaderWarning{Direction: direction, Field: fmt.Sprintf("reserved_%02d", entryIndex), Offset: byteCursor, Bytes: entry.ByteOffset - byteCursor, Message: fmt.Sprintf("%s process data has %d reserved byte(s) before %s; struct fields are not contiguous", direction, entry.ByteOffset-byteCursor, fieldName(entry.Name))})
		}
		byteCursor = entry.ByteOffset + entry.ByteLength
	}
	if layout.ByteLength > byteCursor {
		warnings = append(warnings, HeaderWarning{Direction: direction, Field: "reserved_tail", Offset: byteCursor, Bytes: layout.ByteLength - byteCursor, Message: fmt.Sprintf("%s process data has %d trailing reserved byte(s); generated struct does not cover the full buffer with named fields", direction, layout.ByteLength-byteCursor)})
	}
	return warnings
}

func cType(modelType string) string {
	switch modelType {
	case "bool", "bit", "uint8":
		return "uint8_t"
	case "int8":
		return "int8_t"
	case "uint16":
		return "uint16_t"
	case "int16":
		return "int16_t"
	case "uint32":
		return "uint32_t"
	case "int32":
		return "int32_t"
	case "uint64":
		return "uint64_t"
	case "int64":
		return "int64_t"
	default:
		return "uint8_t"
	}
}

func macroPrefix(value string) string {
	upper := strings.ToUpper(value)
	clean := identifierPattern.ReplaceAllString(upper, "_")
	clean = strings.Trim(clean, "_")
	if clean == "" {
		return "ETHERCAT_DEVICE"
	}
	return clean
}

func fieldName(value string) string {
	lower := strings.ToLower(value)
	var builder strings.Builder
	lastUnderscore := false
	for _, char := range lower {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "field"
	}
	return result
}
