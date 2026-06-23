package cli

import (
	"bytes"
	"strings"
	"testing"
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
	for _, want := range []string{"\"diagnostics\"", "\"ECONFIG_INPUT_UNKNOWN_COMMAND\"", "\"severity\":\"error\""} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("expected JSON diagnostic to contain %q, got %q", want, stderr.String())
		}
	}
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		name string
		err  *cliError
		want int
	}{
		{name: "input", err: inputError("ECONFIG_INPUT_TEST", "input failed"), want: ExitInputError},
		{name: "validation", err: &cliError{exitCode: ExitValidationError, diagnostic: diagnostic{Code: "ECONFIG_VALIDATION_TEST", Severity: "error", Message: "validation failed"}}, want: ExitValidationError},
		{name: "internal", err: &cliError{exitCode: ExitInternalError, diagnostic: diagnostic{Code: "ECONFIG_INTERNAL_TEST", Severity: "error", Message: "internal failed"}}, want: ExitInternalError},
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
