package esi

import (
	"os"
	"strings"
	"testing"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/pdo"
)

func TestGenerateAndParseFixture(t *testing.T) {
	loadResult := config.LoadFile("../../examples/lan9252-basic/device.yaml")
	layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
	addressMap, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
	allDiagnostics := append(loadResult.Diagnostics, layoutDiagnostics...)
	allDiagnostics = append(allDiagnostics, addressDiagnostics...)
	if len(allDiagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", allDiagnostics)
	}

	data, err := Generate(loadResult.Device, layout, addressMap)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `<ProductCode>0x00000001</ProductCode>`) || !strings.Contains(text, `<SyncManager Index="2" Direction="rx" StartAddress="0x1000" Length="256" PdoAssign="0x1c12"></SyncManager>`) {
		t.Fatalf("expected deterministic ESI XML, got %q", text)
	}
	golden, err := os.ReadFile("../../fixtures/esi/lan9252-basic.xml")
	if err != nil {
		t.Fatalf("expected golden ESI fixture, got %v", err)
	}
	if string(golden) != text {
		t.Fatalf("expected generated ESI XML to match golden fixture")
	}

	summary, parseDiagnostics := Parse(data)
	if len(parseDiagnostics) != 0 {
		t.Fatalf("expected no parse diagnostics, got %#v", parseDiagnostics)
	}
	if summary.Identity.VendorID != "0x00000a88" || summary.Identity.Name != "LAN9252 Basic" {
		t.Fatalf("unexpected identity summary: %#v", summary.Identity)
	}
	if len(summary.PDOs) != 2 || summary.PDOs[0].Entries[1].BitOffset != 16 {
		t.Fatalf("expected PDO layout preservation, got %#v", summary.PDOs)
	}
	if summary.ProcessData[0].Address != "0x1000" || summary.ProcessData[1].Address != "0x1100" {
		t.Fatalf("expected process data addresses, got %#v", summary.ProcessData)
	}
}
