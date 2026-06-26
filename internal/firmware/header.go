package firmware

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/guajun/ethercat-configurator/internal/model"
)

var identifierPattern = regexp.MustCompile(`[^A-Z0-9]+`)

func GenerateHeader(device model.Device, layout model.Layout, addressMap model.AddressMap) ([]byte, error) {
	prefix := macroPrefix(device.Identity.Name)
	var output bytes.Buffer
	fmt.Fprintf(&output, "#ifndef %s_ETHERCAT_DEVICE_H\n", prefix)
	fmt.Fprintf(&output, "#define %s_ETHERCAT_DEVICE_H\n\n", prefix)
	output.WriteString("#include <stdint.h>\n\n")
	writeProcessDataSizes(&output, layout)
	writeEasyCATBuffer(&output, "OUT", layout.RX)
	writeEasyCATBuffer(&output, "IN", layout.TX)
	fmt.Fprintf(&output, "#endif /* %s_ETHERCAT_DEVICE_H */\n", prefix)
	return output.Bytes(), nil
}

func writeProcessDataSizes(output *bytes.Buffer, layout model.Layout) {
	fmt.Fprintf(output, "#define CUST_BYTE_NUM_OUT\t%d\n", layout.RX.ByteLength)
	fmt.Fprintf(output, "#define CUST_BYTE_NUM_IN\t%d\n", layout.TX.ByteLength)
	fmt.Fprintf(output, "#define TOT_BYTE_NUM_ROUND_OUT\t%d\n", layout.RX.ByteLength)
	fmt.Fprintf(output, "#define TOT_BYTE_NUM_ROUND_IN\t%d\n\n\n", layout.TX.ByteLength)
}

func writeEasyCATBuffer(output *bytes.Buffer, direction string, layout model.LayoutDirection) {
	fmt.Fprintf(output, "typedef union\n{\n")
	fmt.Fprintf(output, "\tuint8_t Byte[TOT_BYTE_NUM_ROUND_%s];\n", direction)
	fmt.Fprintf(output, "\tstruct\n\t{\n")
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
