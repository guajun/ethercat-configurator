package pdo

import (
	"testing"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/diag"
)

func TestBasicIOFixtureLayout(t *testing.T) {
	loadResult := config.LoadFile("../../examples/lan9252-basic/device.yaml")
	layout, diagnostics := BuildLayout(loadResult.Device)
	diagnostics = append(loadResult.Diagnostics, diagnostics...)
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
	if layout.RX.ByteLength != 3 || layout.TX.ByteLength != 3 {
		t.Fatalf("expected 3 byte RX/TX layouts, got RX=%d TX=%d", layout.RX.ByteLength, layout.TX.ByteLength)
	}
	if layout.RX.Entries[0].BitOffset != 0 || layout.RX.Entries[1].BitOffset != 16 {
		t.Fatalf("unexpected RX offsets: %#v", layout.RX.Entries)
	}
}

func TestPackedBitFixtureLayout(t *testing.T) {
	loadResult := config.LoadFile("../../fixtures/pdo/packed-bits.yaml")
	layout, diagnostics := BuildLayout(loadResult.Device)
	diagnostics = append(loadResult.Diagnostics, diagnostics...)
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
	if layout.RX.BitLength != 6 || layout.RX.ByteLength != 1 || layout.RX.PaddingBits != 2 {
		t.Fatalf("unexpected packed RX layout: %#v", layout.RX)
	}
	if layout.RX.Entries[2].BitOffset != 2 {
		t.Fatalf("expected third entry bit offset 2, got %d", layout.RX.Entries[2].BitOffset)
	}
}

func TestLayoutDiagnostics(t *testing.T) {
	loadResult := config.LoadFile("../../fixtures/pdo/invalid-layout.yaml")
	_, diagnostics := BuildLayout(loadResult.Device)
	assertHasDiagnostic(t, diagnostics, diag.CodePDODuplicateMapping)
	assertHasDiagnostic(t, diagnostics, diag.CodePDOEntryNotFound)
	assertHasDiagnostic(t, diagnostics, diag.CodePDODirectionMismatch)
	assertHasDiagnostic(t, diagnostics, diag.CodePDOUnsupportedWidth)
}

func TestAddressMapDiagnosticsAndOverride(t *testing.T) {
	loadResult := config.LoadFile("../../fixtures/pdi/default-too-small.yaml")
	layout, diagnostics := BuildLayout(loadResult.Device)
	if len(diagnostics) != 0 {
		t.Fatalf("expected no layout diagnostics, got %#v", diagnostics)
	}
	addressMap, diagnostics := BuildAddressMap(loadResult.Device, layout)
	assertHasDiagnostic(t, diagnostics, diag.CodePDIBufferTooSmall)
	if !addressMap.RX.FromDefault || !addressMap.TX.FromDefault {
		t.Fatalf("expected default-derived address map, got %#v", addressMap)
	}

	fixedResult := config.LoadFile("../../fixtures/pdi/explicit-fit.yaml")
	fixedLayout, diagnostics := BuildLayout(fixedResult.Device)
	if len(diagnostics) != 0 {
		t.Fatalf("expected no fixed layout diagnostics, got %#v", diagnostics)
	}
	fixedMap, diagnostics := BuildAddressMap(fixedResult.Device, fixedLayout)
	if len(diagnostics) != 0 {
		t.Fatalf("expected explicit addresses to pass, got %#v", diagnostics)
	}
	if fixedMap.RX.FromDefault || fixedMap.TX.FromDefault {
		t.Fatalf("expected configured addresses, got %#v", fixedMap)
	}
}

func TestAddressMapOverlapAlignmentAndRange(t *testing.T) {
	loadResult := config.LoadFile("../../fixtures/pdi/invalid-addresses.yaml")
	layout, diagnostics := BuildLayout(loadResult.Device)
	if len(diagnostics) != 0 {
		t.Fatalf("expected no layout diagnostics, got %#v", diagnostics)
	}
	_, diagnostics = BuildAddressMap(loadResult.Device, layout)
	assertHasDiagnostic(t, diagnostics, diag.CodePDIAddressInvalid)
	assertHasDiagnostic(t, diagnostics, diag.CodePDIRegionOverlap)
	assertHasDiagnostic(t, diagnostics, diag.CodePDILayoutOutOfRange)
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
