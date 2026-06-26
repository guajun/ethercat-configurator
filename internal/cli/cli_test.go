package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guajun/ethercat-configurator/internal/diag"
)

func TestVersionOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--version"}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if got := strings.TrimSpace(stdout.String()); got != Version {
		t.Fatalf("expected version %q, got %q", Version, got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestHelpOutputForRootAndNestedCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "root", args: []string{"--help"}, want: "Usage:\n  ethercat-configurator [--json]"},
		{name: "validate", args: []string{"validate", "--help"}, want: "Usage:\n  ethercat-configurator validate"},
		{name: "gen esi", args: []string{"gen", "esi", "--help"}, want: "Usage:\n  ethercat-configurator gen esi"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Run(test.args, &stdout, &stderr)

			if exitCode != ExitSuccess {
				t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("expected help to contain %q, got %q", test.want, stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got %q", stderr.String())
			}
		})
	}
}

func TestUnknownCommandJSONDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--json", "unknown"}, &stdout, &stderr)

	if exitCode != ExitInputError {
		t.Fatalf("expected exit code %d, got %d", ExitInputError, exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	want := "{\"status\":\"error\",\"diagnostics\":[{\"code\":\"ECONFIG_INPUT_UNKNOWN_COMMAND\",\"severity\":\"error\",\"message\":\"unknown command \\\"unknown\\\"\"}]}\n"
	if stderr.String() != want {
		t.Fatalf("expected JSON diagnostic %q, got %q", want, stderr.String())
	}
}

func TestValidateJSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--json", "validate", "specs"}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	want := "{\"status\":\"ok\",\"target\":\"specs\",\"diagnostics\":[]}\n"
	if stdout.String() != want {
		t.Fatalf("expected stable validate JSON %q, got %q", want, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestValidateDeviceFixture(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "../../examples/lan9252-basic/device.yaml"}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if got := strings.TrimSpace(stdout.String()); got != "validate ../../examples/lan9252-basic/device.yaml: ok" {
		t.Fatalf("expected device validation success, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestValidateDeviceJSONDiagnostics(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--json", "validate", "../../fixtures/pdi/default-too-small.yaml"}, &stdout, &stderr)

	if exitCode != ExitValidationError {
		t.Fatalf("expected exit code %d, got %d", ExitValidationError, exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), diag.CodePDIBufferTooSmall) || !strings.Contains(stderr.String(), diag.CodePDILayoutOutOfRange) {
		t.Fatalf("expected PDI diagnostics, got %q", stderr.String())
	}
}

func TestReportMarkdownOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"report", "../../examples/lan9252-basic/device.yaml"}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if !strings.Contains(stdout.String(), "# EtherCAT Configuration Report: LAN9252 Basic") || !strings.Contains(stdout.String(), "## Process Data Address Regions") {
		t.Fatalf("expected markdown report, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestReportJSONOutputAcceptsSingleDashJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"-json", "report", "../../examples/lan9252-basic/device.yaml"}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if !strings.HasPrefix(stdout.String(), "{\"target\":\"../../examples/lan9252-basic/device.yaml\"") || !strings.Contains(stdout.String(), "\"diagnostics\":[]") {
		t.Fatalf("expected JSON report, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestReportWritesOutputFile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "report.md")

	exitCode := Run([]string{"report", "../../examples/lan9252-basic/device.yaml", "-o", outputPath}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if got := strings.TrimSpace(stdout.String()); got != "report ../../examples/lan9252-basic/device.yaml: ok" {
		t.Fatalf("expected success output, got %q", got)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected report file, got %v", err)
	}
	if !strings.Contains(string(data), "## PDO Table") {
		t.Fatalf("expected markdown report file, got %q", string(data))
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestCommandLevelLongOutputFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "device.xml")

	exitCode := Run([]string{"gen", "esi", "../../examples/lan9252-basic/device.yaml", "--output", outputPath}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d stderr=%q", ExitSuccess, exitCode, stderr.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected output file, got %v", err)
	}
	if !strings.Contains(string(data), "<EtherCATInfo>") {
		t.Fatalf("expected generated XML, got %q", string(data))
	}
}

func TestReportMissingFileReturnsDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--json", "report", "../../fixtures/missing.yaml"}, &stdout, &stderr)

	if exitCode != ExitValidationError {
		t.Fatalf("expected exit code %d, got %d", ExitValidationError, exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), diag.CodeConfigParseError) {
		t.Fatalf("expected parse diagnostic, got %q", stderr.String())
	}
}

func TestGenCommandsWriteArtifacts(t *testing.T) {
	devicePath := "../../examples/lan9252-basic/device.yaml"
	tests := []struct {
		name     string
		artifact string
		fileName string
		contains []byte
	}{
		{name: "esi", artifact: "esi", fileName: "device.xml", contains: []byte("<EtherCATInfo>")},
		{name: "sii", artifact: "sii", fileName: "eeprom.bin", contains: []byte("ECSII001")},
		{name: "header", artifact: "header", fileName: "ethercat_device.h", contains: []byte("PROCBUFFER_OUT")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			outputPath := filepath.Join(t.TempDir(), test.fileName)

			exitCode := Run([]string{"gen", test.artifact, devicePath, "-o", outputPath}, &stdout, &stderr)

			if exitCode != ExitSuccess {
				t.Fatalf("expected exit code %d, got %d stderr=%q", ExitSuccess, exitCode, stderr.String())
			}
			data, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("expected generated artifact %s, got %v", outputPath, err)
			}
			if !bytes.Contains(data, test.contains) {
				t.Fatalf("expected generated %s to contain %q, got %q", test.artifact, test.contains, data)
			}
			if !strings.Contains(stdout.String(), "gen "+test.artifact+" "+devicePath+": ok") {
				t.Fatalf("expected success output, got %q", stdout.String())
			}
			if test.artifact == "header" && !strings.Contains(stdout.String(), "warning: OUT process data has 1 trailing reserved byte(s)") {
				t.Fatalf("expected header generation warning for reserved tail, got %q", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got %q", stderr.String())
			}
		})
	}
}

func TestGenESIContinuesWithSyncManagerOverlapWarning(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "device.xml")

	exitCode := Run([]string{"gen", "esi", "../../fixtures/easycat/known_bad_overlap/device.yaml", "-o", outputPath}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected generation to continue with warning, got exit code %d stderr=%q", exitCode, stderr.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected generated ESI file, got %v", err)
	}
	if !bytes.Contains(data, []byte("<EtherCATInfo>")) {
		t.Fatalf("expected generated ESI XML, got %q", data)
	}
	if !strings.Contains(stdout.String(), "warning: "+diag.CodePDISyncManagerOverlap) || !strings.Contains(stdout.String(), "0x1000-0x1240 overlaps TX start 0x1200") {
		t.Fatalf("expected explicit SyncManager overlap warning, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestInspectCommandsRenderSummaries(t *testing.T) {
	tests := []struct {
		name     string
		artifact string
		path     string
		want     string
	}{
		{name: "esi", artifact: "esi", path: "../../fixtures/esi/lan9252-basic.xml", want: "ESI LAN9252 Basic vendor=0x00000a88"},
		{name: "sii", artifact: "sii", path: "../../fixtures/sii/lan9252-basic.bin", want: "SII LAN9252 Basic vendor=0x00000a88"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := Run([]string{"inspect", test.artifact, test.path}, &stdout, &stderr)

			if exitCode != ExitSuccess {
				t.Fatalf("expected exit code %d, got %d stderr=%q", ExitSuccess, exitCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("expected inspect summary to contain %q, got %q", test.want, stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got %q", stderr.String())
			}
		})
	}
}

func TestInspectSIIJSONAndChecksumDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tempDir := t.TempDir()
	devicePath := "../../examples/lan9252-basic/device.yaml"
	siiPath := filepath.Join(tempDir, "eeprom.bin")

	if exitCode := Run([]string{"gen", "sii", devicePath, "-o", siiPath}, &stdout, &stderr); exitCode != ExitSuccess {
		t.Fatalf("expected gen sii success, got %d stderr=%q", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := Run([]string{"--json", "inspect", "sii", siiPath}, &stdout, &stderr); exitCode != ExitSuccess {
		t.Fatalf("expected inspect sii success, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"vendor_id":"0x00000a88"`) || !strings.Contains(stdout.String(), `"process_data"`) {
		t.Fatalf("expected SII JSON summary, got %q", stdout.String())
	}

	data, err := os.ReadFile(siiPath)
	if err != nil {
		t.Fatalf("expected SII file, got %v", err)
	}
	data[12] ^= 0xff
	if err := os.WriteFile(siiPath, data, 0o644); err != nil {
		t.Fatalf("expected to corrupt SII file, got %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := Run([]string{"--json", "inspect", "sii", siiPath}, &stdout, &stderr); exitCode != ExitValidationError {
		t.Fatalf("expected checksum validation error, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), diag.CodeSIIChecksumInvalid) {
		t.Fatalf("expected checksum diagnostic, got %q", stderr.String())
	}
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		name string
		err  *cliError
		want int
	}{
		{name: "input", err: inputError("ECONFIG_INPUT_TEST", "input failed"), want: ExitInputError},
		{name: "validation", err: &cliError{exitCode: ExitValidationError, diagnostic: diag.Error("ECONFIG_VALIDATION_TEST", "validation failed")}, want: ExitValidationError},
		{name: "internal", err: &cliError{exitCode: ExitInternalError, diagnostic: diag.Error("ECONFIG_INTERNAL_TEST", "internal failed")}, want: ExitInternalError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer

			if got := writeError(&stderr, runOptions{Quiet: true}, test.err); got != test.want {
				t.Fatalf("expected exit code %d, got %d", test.want, got)
			}
		})
	}
}

func TestMinimalFixtureCommandPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "specs"}, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Fatalf("expected exit code %d, got %d", ExitSuccess, exitCode)
	}
	if got := strings.TrimSpace(stdout.String()); got != "validate specs: ok" {
		t.Fatalf("expected fixture path success output, got %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}
