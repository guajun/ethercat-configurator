package diag

import (
	"bytes"
	"testing"
)

func TestRenderJSONSortsDiagnostics(t *testing.T) {
	var output bytes.Buffer

	err := RenderJSON(&output, Result{
		Status: "error",
		Target: "device.yaml",
		Diagnostics: []Diagnostic{
			{Code: CodePDOEntryNotFound, Severity: SeverityError, Message: "PDO entry 0x7000:01 not found", File: "device.yaml", Line: 20, Column: 5},
			{Code: CodeConfigParseError, Severity: SeverityError, Message: "invalid YAML", File: "device.yaml", Line: 2, Column: 1, Details: map[string]string{"hint": "check indentation"}},
		},
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	want := "{\"status\":\"error\",\"target\":\"device.yaml\",\"diagnostics\":[{\"code\":\"CONFIG_PARSE_ERROR\",\"severity\":\"error\",\"message\":\"invalid YAML\",\"file\":\"device.yaml\",\"line\":2,\"column\":1,\"details\":{\"hint\":\"check indentation\"}},{\"code\":\"PDO_ENTRY_NOT_FOUND\",\"severity\":\"error\",\"message\":\"PDO entry 0x7000:01 not found\",\"file\":\"device.yaml\",\"line\":20,\"column\":5}]}\n"
	if output.String() != want {
		t.Fatalf("expected JSON snapshot %q, got %q", want, output.String())
	}
}

func TestRenderTextIncludesActionableLocation(t *testing.T) {
	var output bytes.Buffer

	err := RenderText(&output, []Diagnostic{
		{Code: CodeODInvalidType, Severity: SeverityError, Message: "unsupported object type BOOL32", File: "device.yaml", Line: 12, Column: 9},
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	want := "OD_INVALID_TYPE: device.yaml:12:9: unsupported object type BOOL32\n"
	if output.String() != want {
		t.Fatalf("expected text diagnostic %q, got %q", want, output.String())
	}
}
