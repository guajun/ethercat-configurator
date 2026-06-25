package config

import (
	"testing"

	"github.com/guajun/ethercat-configurator/internal/diag"
)

func TestLoadValidDevice(t *testing.T) {
	result := LoadFile("../../examples/lan9252-basic/device.yaml")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	if result.Device.Identity.Name != "LAN9252 Basic" {
		t.Fatalf("expected device name, got %q", result.Device.Identity.Name)
	}
	if len(result.Device.ObjectDictionary) != 4 {
		t.Fatalf("expected 4 OD entries, got %d", len(result.Device.ObjectDictionary))
	}
	if result.Device.ProcessData.RX.BufferMode != "single" || result.Device.ProcessData.TX.BufferMode != "single" {
		t.Fatalf("expected default single buffer modes, got rx=%q tx=%q", result.Device.ProcessData.RX.BufferMode, result.Device.ProcessData.TX.BufferMode)
	}
}

func TestLoadProcessDataBufferMode(t *testing.T) {
	result := LoadFile("../../examples/easycat-safe-64/device.yaml")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	if result.Device.ProcessData.RX.BufferMode != "triple" || result.Device.ProcessData.TX.BufferMode != "triple" {
		t.Fatalf("expected explicit triple buffer modes, got rx=%q tx=%q", result.Device.ProcessData.RX.BufferMode, result.Device.ProcessData.TX.BufferMode)
	}
}

func TestInvalidFixtureDiagnostics(t *testing.T) {
	result := LoadFile("../../fixtures/invalid/bad-device.yaml")
	assertHasDiagnostic(t, result.Diagnostics, diag.CodeODInvalidIndex)
	assertHasDiagnostic(t, result.Diagnostics, diag.CodeODDuplicateIndex)
	assertHasDiagnostic(t, result.Diagnostics, diag.CodeODInvalidType)
	assertHasDiagnostic(t, result.Diagnostics, diag.CodeODInvalidAccess)
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
