package diag

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	CodeConfigParseError      = "CONFIG_PARSE_ERROR"
	CodeODDuplicateIndex      = "OD_DUPLICATE_INDEX"
	CodeODInvalidIndex        = "OD_INVALID_INDEX"
	CodeODInvalidType         = "OD_INVALID_TYPE"
	CodeODInvalidAccess       = "OD_INVALID_ACCESS"
	CodePDOEntryNotFound      = "PDO_ENTRY_NOT_FOUND"
	CodePDODuplicateMapping   = "PDO_DUPLICATE_MAPPING"
	CodePDODirectionMismatch  = "PDO_DIRECTION_MISMATCH"
	CodePDOUnsupportedWidth   = "PDO_UNSUPPORTED_WIDTH"
	CodePDOSizeOverflow       = "PDO_SIZE_OVERFLOW"
	CodePDIAddressInvalid     = "PDI_ADDRESS_INVALID"
	CodePDIRegionOverlap      = "PDI_REGION_OVERLAP"
	CodePDISyncManagerOverlap = "PDI_SYNC_MANAGER_OVERLAP"
	CodePDIBufferTooSmall     = "PDI_BUFFER_TOO_SMALL"
	CodePDILayoutOutOfRange   = "PDI_LAYOUT_OUT_OF_RANGE"
	CodeESIParseError         = "ESI_PARSE_ERROR"
	CodeSIIParseError         = "SII_PARSE_ERROR"
	CodeSIIChecksumInvalid    = "SII_CHECKSUM_INVALID"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Diagnostic struct {
	Code     string            `json:"code"`
	Severity Severity          `json:"severity"`
	Message  string            `json:"message"`
	File     string            `json:"file,omitempty"`
	Line     int               `json:"line,omitempty"`
	Column   int               `json:"column,omitempty"`
	Details  map[string]string `json:"details,omitempty"`
}

type Result struct {
	Status      string       `json:"status"`
	Target      string       `json:"target,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

func Error(code string, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityError, Message: message}
}

func RenderJSON(writer io.Writer, result Result) error {
	result.Diagnostics = Sort(result.Diagnostics)
	if result.Diagnostics == nil {
		result.Diagnostics = []Diagnostic{}
	}
	encoder := json.NewEncoder(writer)
	return encoder.Encode(result)
}

func RenderText(writer io.Writer, diagnostics []Diagnostic) error {
	diagnostics = Sort(diagnostics)
	for _, diagnostic := range diagnostics {
		if _, err := fmt.Fprintln(writer, FormatText(diagnostic)); err != nil {
			return err
		}
	}
	return nil
}

func FormatText(diagnostic Diagnostic) string {
	location := formatLocation(diagnostic)
	if location == "" {
		return fmt.Sprintf("%s: %s", diagnostic.Code, diagnostic.Message)
	}
	return fmt.Sprintf("%s: %s: %s", diagnostic.Code, location, diagnostic.Message)
}

func Sort(diagnostics []Diagnostic) []Diagnostic {
	sorted := append([]Diagnostic(nil), diagnostics...)
	sort.SliceStable(sorted, func(leftIndex int, rightIndex int) bool {
		left := sorted[leftIndex]
		right := sorted[rightIndex]
		leftKey := []string{left.File, fmt.Sprintf("%012d", left.Line), fmt.Sprintf("%012d", left.Column), left.Code, string(left.Severity), left.Message, detailsKey(left.Details)}
		rightKey := []string{right.File, fmt.Sprintf("%012d", right.Line), fmt.Sprintf("%012d", right.Column), right.Code, string(right.Severity), right.Message, detailsKey(right.Details)}
		return strings.Join(leftKey, "\x00") < strings.Join(rightKey, "\x00")
	})
	return sorted
}

func formatLocation(diagnostic Diagnostic) string {
	if diagnostic.File == "" {
		return ""
	}
	if diagnostic.Line == 0 {
		return diagnostic.File
	}
	if diagnostic.Column == 0 {
		return fmt.Sprintf("%s:%d", diagnostic.File, diagnostic.Line)
	}
	return fmt.Sprintf("%s:%d:%d", diagnostic.File, diagnostic.Line, diagnostic.Column)
}

func detailsKey(details map[string]string) string {
	if len(details) == 0 {
		return ""
	}
	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+details[key])
	}
	return strings.Join(parts, "\x00")
}
