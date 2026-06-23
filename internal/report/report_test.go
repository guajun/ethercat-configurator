package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/pdo"
)

func TestRenderMarkdownStableFixture(t *testing.T) {
	reportData := buildFixtureReport(t, "../../examples/lan9252-basic/device.yaml")
	var output bytes.Buffer

	if err := RenderMarkdown(&output, reportData); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := `# EtherCAT Configuration Report: LAN9252 Basic

Target: ` + "`../../examples/lan9252-basic/device.yaml`" + `

## Identity

| Field | Value |
| --- | --- |
| Vendor ID | ` + "`0x00000a88`" + ` |
| Product Code | ` + "`0x00000001`" + ` |
| Revision | ` + "`0x00000001`" + ` |
| Name | LAN9252 Basic |

## Object Dictionary Summary

Total entries: 4

| Index | Subindex | Name | Type | Bits | Access | PDO | Default |
| --- | --- | --- | --- | ---: | --- | --- | --- |
| ` + "`0x6000`" + ` | ` + "`0x01`" + ` | Status Word | ` + "`uint16`" + ` | 16 | ` + "`ro`" + ` | ` + "`tx`" + ` | 0 |
| ` + "`0x6000`" + ` | ` + "`0x02`" + ` | Input Ready | ` + "`bool`" + ` | 1 | ` + "`ro`" + ` | ` + "`tx`" + ` | false |
| ` + "`0x7000`" + ` | ` + "`0x01`" + ` | Control Word | ` + "`uint16`" + ` | 16 | ` + "`rw`" + ` | ` + "`rx`" + ` | 0 |
| ` + "`0x7000`" + ` | ` + "`0x02`" + ` | Enable Output | ` + "`bool`" + ` | 1 | ` + "`rw`" + ` | ` + "`rx`" + ` | false |

## PDO Table

| Direction | PDO | PDO Name | Entry | Name | Type | Byte Offset | Bit Offset | Bits | Bytes |
| --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: |
| ` + "`rx`" + ` | ` + "`0x1600`" + ` | Outputs | ` + "`0x7000:0x01`" + ` | Control Word | ` + "`uint16`" + ` | 0 | 0 | 16 | 2 |
| ` + "`rx`" + ` | ` + "`0x1600`" + ` | Outputs | ` + "`0x7000:0x02`" + ` | Enable Output | ` + "`bool`" + ` | 2 | 16 | 1 | 1 |
| ` + "`tx`" + ` | ` + "`0x1a00`" + ` | Inputs | ` + "`0x6000:0x01`" + ` | Status Word | ` + "`uint16`" + ` | 0 | 0 | 16 | 2 |
| ` + "`tx`" + ` | ` + "`0x1a00`" + ` | Inputs | ` + "`0x6000:0x02`" + ` | Input Ready | ` + "`bool`" + ` | 2 | 16 | 1 | 1 |

## Process Data Address Regions

| Direction | Address | Required Bytes | Buffer Bytes | End | Source |
| --- | --- | ---: | ---: | --- | --- |
| ` + "`rx`" + ` | ` + "`0x1000`" + ` | 3 | 256 | ` + "`0x1100`" + ` | ` + "`default`" + ` |
| ` + "`tx`" + ` | ` + "`0x1100`" + ` | 3 | 256 | ` + "`0x1200`" + ` | ` + "`default`" + ` |

## Sync Manager Sizes

| Sync Manager | Direction | Required Bytes | Buffer Bytes |
| ---: | --- | ---: | ---: |
| 2 | ` + "`rx`" + ` | 3 | 256 |
| 3 | ` + "`tx`" + ` | 3 | 256 |

## Diagnostics

No diagnostics.
`
	if output.String() != want {
		t.Fatalf("expected stable markdown report %q, got %q", want, output.String())
	}
}

func TestBuildReportIncludesDefaultRegionDiagnostics(t *testing.T) {
	reportData := buildFixtureReport(t, "../../fixtures/pdi/default-too-small.yaml")

	if len(reportData.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics in report")
	}
	if reportData.ProcessData[0].Source != "default" || reportData.ProcessData[0].RequiredBytes != 512 || reportData.ProcessData[0].BufferBytes != 256 {
		t.Fatalf("expected obvious oversized default RX region, got %#v", reportData.ProcessData[0])
	}

	var output bytes.Buffer
	if err := RenderMarkdown(&output, reportData); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.Contains(output.String(), diag.CodePDIBufferTooSmall) || !strings.Contains(output.String(), "| `rx` | `0x1000` | 512 | 256 |") {
		t.Fatalf("expected oversized default region and diagnostics, got %q", output.String())
	}
}

func TestRenderJSONStableFixture(t *testing.T) {
	reportData := buildFixtureReport(t, "../../examples/lan9252-basic/device.yaml")
	var output bytes.Buffer

	if err := RenderJSON(&output, reportData); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	wantPrefix := "{\"target\":\"../../examples/lan9252-basic/device.yaml\",\"identity\":{\"vendor_id\":\"0x00000a88\",\"product_code\":\"0x00000001\",\"revision\":\"0x00000001\",\"name\":\"LAN9252 Basic\"},"
	if !strings.HasPrefix(output.String(), wantPrefix) || !strings.Contains(output.String(), "\"diagnostics\":[]") {
		t.Fatalf("expected stable JSON report, got %q", output.String())
	}
}

func buildFixtureReport(t *testing.T, path string) Report {
	t.Helper()
	loadResult := config.LoadFile(path)
	layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
	addressMap, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
	diagnostics := append([]diag.Diagnostic(nil), loadResult.Diagnostics...)
	diagnostics = append(diagnostics, layoutDiagnostics...)
	diagnostics = append(diagnostics, addressDiagnostics...)
	return Build(path, loadResult.Device, layout, addressMap, diagnostics)
}
