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

func TestGenerateArtifactsAndInspect(t *testing.T) {
	tempDir := t.TempDir()
	devicePath := "../../examples/lan9252-basic/device.yaml"
	esiPath := filepath.Join(tempDir, "device.xml")
	siiPath := filepath.Join(tempDir, "eeprom.bin")
	headerPath := filepath.Join(tempDir, "ethercat_device.h")

	commands := [][]string{
		{"gen", "esi", devicePath, "-o", esiPath},
		{"gen", "sii", devicePath, "-o", siiPath},
		{"gen", "header", devicePath, "-o", headerPath},
	}
	for _, command := range commands {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := Run(command, &stdout, &stderr)
		if exitCode != ExitSuccess {
			t.Fatalf("expected %v to succeed with exit code %d, got %d stderr=%q", command, ExitSuccess, exitCode, stderr.String())
		}
	}

	esiData, err := os.ReadFile(esiPath)
	if err != nil {
		t.Fatalf("expected ESI file, got %v", err)
	}
	if !strings.Contains(string(esiData), "<EtherCATInfo>") || !strings.Contains(string(esiData), `StartAddress="0x1000"`) {
		t.Fatalf("expected generated ESI XML, got %q", string(esiData))
	}
	headerData, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatalf("expected header file, got %v", err)
	}
	if !strings.Contains(string(headerData), "LAN9252_BASIC_RX_CONTROL_WORD_BYTE_OFFSET") {
		t.Fatalf("expected generated header constants, got %q", string(headerData))
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exitCode := Run([]string{"inspect", "esi", esiPath}, &stdout, &stderr); exitCode != ExitSuccess {
		t.Fatalf("expected inspect esi success, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ESI LAN9252 Basic") {
		t.Fatalf("expected ESI summary, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := Run([]string{"--json", "inspect", "sii", siiPath}, &stdout, &stderr); exitCode != ExitSuccess {
		t.Fatalf("expected inspect sii success, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"vendor_id":"0x00000a88"`) || !strings.Contains(stdout.String(), `"process_data"`) {
		t.Fatalf("expected SII JSON summary, got %q", stdout.String())
	}
}

func TestInspectSIIChecksumDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	tempDir := t.TempDir()
	devicePath := "../../examples/lan9252-basic/device.yaml"
	siiPath := filepath.Join(tempDir, "eeprom.bin")

	if exitCode := Run([]string{"gen", "sii", devicePath, "-o", siiPath}, &stdout, &stderr); exitCode != ExitSuccess {
		t.Fatalf("expected gen sii success, got %d stderr=%q", exitCode, stderr.String())
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
	if !strings.Contains(stderr.String(), "SII_CHECKSUM_INVALID") {
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
