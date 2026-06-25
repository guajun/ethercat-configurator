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
	output.WriteString("#if defined(__GNUC__)\n")
	fmt.Fprintf(&output, "#define %s_PACKED __attribute__((packed))\n", prefix)
	output.WriteString("#else\n")
	fmt.Fprintf(&output, "#define %s_PACKED\n", prefix)
	output.WriteString("#endif\n\n")
	output.WriteString("#if defined(_MSC_VER)\n")
	output.WriteString("#pragma pack(push, 1)\n")
	output.WriteString("#endif\n\n")

	writeIdentity(&output, prefix, device.Identity)
	writeRegion(&output, prefix, "RX", addressMap.RX)
	writeRegion(&output, prefix, "TX", addressMap.TX)
	writeEntries(&output, prefix, "RX", layout.RX)
	writeEntries(&output, prefix, "TX", layout.TX)
	writeStruct(&output, prefix, "Rx", "RX", layout.RX)
	writeStruct(&output, prefix, "Tx", "TX", layout.TX)
	writeAssertions(&output, prefix, layout)

	output.WriteString("\n#if defined(_MSC_VER)\n")
	output.WriteString("#pragma pack(pop)\n")
	output.WriteString("#endif\n")
	fmt.Fprintf(&output, "\n#undef %s_PACKED\n\n", prefix)
	fmt.Fprintf(&output, "#endif /* %s_ETHERCAT_DEVICE_H */\n", prefix)
	return output.Bytes(), nil
}

func writeIdentity(output *bytes.Buffer, prefix string, identity model.Identity) {
	fmt.Fprintf(output, "#define %s_VENDOR_ID 0x%08XU\n", prefix, identity.VendorID)
	fmt.Fprintf(output, "#define %s_PRODUCT_CODE 0x%08XU\n", prefix, identity.ProductCode)
	fmt.Fprintf(output, "#define %s_REVISION 0x%08XU\n\n", prefix, identity.Revision)
}

func writeRegion(output *bytes.Buffer, prefix string, label string, region model.AddressRegion) {
	fmt.Fprintf(output, "#define %s_%s_PDI_ADDRESS 0x%04XU\n", prefix, label, region.Address)
	fmt.Fprintf(output, "#define %s_%s_REQUIRED_BYTES %dU\n", prefix, label, region.RequiredBytes)
	fmt.Fprintf(output, "#define %s_%s_BUFFER_BYTES %dU\n", prefix, label, region.BufferBytes)
	fmt.Fprintf(output, "#define %s_%s_BUFFER_COUNT %dU\n\n", prefix, label, model.BufferCount(region.BufferMode))
}

func writeEntries(output *bytes.Buffer, prefix string, direction string, layout model.LayoutDirection) {
	for _, entry := range layout.Entries {
		name := macroPrefix(entry.Name)
		fmt.Fprintf(output, "#define %s_%s_%s_INDEX 0x%04XU\n", prefix, direction, name, entry.Index)
		fmt.Fprintf(output, "#define %s_%s_%s_SUBINDEX 0x%02XU\n", prefix, direction, name, entry.Subindex)
		fmt.Fprintf(output, "#define %s_%s_%s_BYTE_OFFSET %dU\n", prefix, direction, name, entry.ByteOffset)
		fmt.Fprintf(output, "#define %s_%s_%s_BIT_OFFSET %dU\n", prefix, direction, name, entry.BitOffset)
		fmt.Fprintf(output, "#define %s_%s_%s_BIT_LENGTH %dU\n", prefix, direction, name, entry.BitLength)
		fmt.Fprintf(output, "#define %s_%s_%s_BYTE_LENGTH %dU\n", prefix, direction, name, entry.ByteLength)
		if entry.BitLength < 32 {
			fmt.Fprintf(output, "#define %s_%s_%s_BIT_MASK 0x%XU\n\n", prefix, direction, name, (uint64(1)<<entry.BitLength)-1)
		} else {
			fmt.Fprintf(output, "#define %s_%s_%s_BIT_MASK 0xFFFFFFFFU\n\n", prefix, direction, name)
		}
	}
}

func writeStruct(output *bytes.Buffer, prefix string, typePrefix string, direction string, layout model.LayoutDirection) {
	fmt.Fprintf(output, "typedef struct %s_PACKED\n{\n", prefix)
	byteCursor := uint32(0)
	for entryIndex, entry := range layout.Entries {
		if entry.BitOffset%8 != 0 || entry.BitLength%8 != 0 {
			continue
		}
		if entry.ByteOffset > byteCursor {
			fmt.Fprintf(output, "    uint8_t reserved_%02d[%d];\n", entryIndex, entry.ByteOffset-byteCursor)
			byteCursor = entry.ByteOffset
		}
		fmt.Fprintf(output, "    %s %s;\n", cType(entry.Type), fieldName(entry.Name))
		byteCursor += entry.ByteLength
	}
	if layout.ByteLength > byteCursor {
		fmt.Fprintf(output, "    uint8_t reserved_tail[%d];\n", layout.ByteLength-byteCursor)
	}
	if layout.ByteLength == 0 {
		output.WriteString("    uint8_t unused;\n")
	}
	fmt.Fprintf(output, "} %sProcessData;\n\n", typePrefix)
	fmt.Fprintf(output, "typedef union\n{\n    uint8_t bytes[%s_%s_REQUIRED_BYTES];\n    %sProcessData fields;\n} %sProcessBuffer;\n\n", prefix, direction, typePrefix, typePrefix)
}

func writeAssertions(output *bytes.Buffer, prefix string, layout model.Layout) {
	output.WriteString("#if defined(__STDC_VERSION__) && (__STDC_VERSION__ >= 201112L)\n")
	fmt.Fprintf(output, "_Static_assert(sizeof(RxProcessData) == %s_RX_REQUIRED_BYTES, \"RX process data size mismatch\");\n", prefix)
	fmt.Fprintf(output, "_Static_assert(sizeof(TxProcessData) == %s_TX_REQUIRED_BYTES, \"TX process data size mismatch\");\n", prefix)
	fmt.Fprintf(output, "_Static_assert(%dU == %s_RX_REQUIRED_BYTES, \"RX generated layout changed\");\n", layout.RX.ByteLength, prefix)
	fmt.Fprintf(output, "_Static_assert(%dU == %s_TX_REQUIRED_BYTES, \"TX generated layout changed\");\n", layout.TX.ByteLength, prefix)
	output.WriteString("#endif\n")
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
