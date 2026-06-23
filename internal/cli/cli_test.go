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
