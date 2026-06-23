package sii

import (
	"bytes"
	"os"
	"testing"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/pdo"
)

func TestGenerateParseAndChecksum(t *testing.T) {
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
	golden, err := os.ReadFile("../../fixtures/sii/lan9252-basic.bin")
	if err != nil {
		t.Fatalf("expected golden SII fixture, got %v", err)
	}
	if !bytes.Equal(golden, data) {
		t.Fatalf("expected generated SII binary to match golden fixture")
	}
	summary, parseDiagnostics := Parse(data)
	if len(parseDiagnostics) != 0 {
		t.Fatalf("expected no parse diagnostics, got %#v", parseDiagnostics)
	}
	if summary.Identity.VendorID != "0x00000a88" || summary.Identity.ProductCode != "0x00000001" {
		t.Fatalf("unexpected identity summary: %#v", summary.Identity)
	}
	if len(summary.PDOs) != 2 || summary.PDOs[0].Entries[1].BitOffset != 16 {
		t.Fatalf("expected PDO layout preservation, got %#v", summary.PDOs)
	}
	if summary.ProcessData[0].Address != "0x1000" || summary.ProcessData[1].Address != "0x1100" {
		t.Fatalf("expected process data addresses, got %#v", summary.ProcessData)
	}

	data[12] ^= 0xff
	_, parseDiagnostics = Parse(data)
	assertHasDiagnostic(t, parseDiagnostics, diag.CodeSIIChecksumInvalid)
}

func assertHasDiagnostic(t *testing.T, diagnostics []diag.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %s in %#v", code, diagnostics)
}
