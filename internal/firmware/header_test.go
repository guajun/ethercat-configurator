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
	if normalizeNewlines(string(golden)) != text {
		t.Fatalf("expected generated header to match golden fixture")
	}
	checks := []string{
		"#define CUST_BYTE_NUM_OUT\t3",
		"#define CUST_BYTE_NUM_IN\t3",
		"#define TOT_BYTE_NUM_ROUND_OUT\t3",
		"typedef union",
		"uint8_t Byte[TOT_BYTE_NUM_ROUND_OUT];",
		"} PROCBUFFER_OUT;",
		"} PROCBUFFER_IN;",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("expected header to contain %q, got %q", check, text)
		}
	}
	for _, forbidden := range []string{"PDI_ADDRESS", "BYTE_OFFSET", "BIT_MASK", "_Static_assert", "RxProcessData"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected EasyCAT-style header to omit %q, got %q", forbidden, text)
		}
	}
}

func normalizeNewlines(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}
