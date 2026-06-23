package firmware

import (
	"os"
	"strings"
	"testing"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/pdo"
)

func TestGenerateHeaderFixture(t *testing.T) {
	loadResult := config.LoadFile("../../examples/lan9252-basic/device.yaml")
	layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
	addressMap, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
	allDiagnostics := append(loadResult.Diagnostics, layoutDiagnostics...)
	allDiagnostics = append(allDiagnostics, addressDiagnostics...)
	if len(allDiagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", allDiagnostics)
	}

	data, err := GenerateHeader(loadResult.Device, layout, addressMap)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	text := string(data)
	golden, err := os.ReadFile("../../fixtures/firmware/ethercat_device.h")
	if err != nil {
		t.Fatalf("expected golden header fixture, got %v", err)
	}
	if string(golden) != text {
		t.Fatalf("expected generated header to match golden fixture")
	}
	checks := []string{
		"#define LAN9252_BASIC_RX_PDI_ADDRESS 0x1000U",
		"#define LAN9252_BASIC_TX_PDI_ADDRESS 0x1100U",
		"#define LAN9252_BASIC_RX_CONTROL_WORD_BYTE_OFFSET 0U",
		"#define LAN9252_BASIC_RX_ENABLE_OUTPUT_BIT_OFFSET 16U",
		"typedef struct LAN9252_BASIC_PACKED",
		"_Static_assert(3U == LAN9252_BASIC_RX_REQUIRED_BYTES",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("expected header to contain %q, got %q", check, text)
		}
	}
}
