package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/diag"
	"github.com/guajun/ethercat-configurator/internal/esi"
	"github.com/guajun/ethercat-configurator/internal/firmware"
	"github.com/guajun/ethercat-configurator/internal/model"
	"github.com/guajun/ethercat-configurator/internal/pdo"
	"github.com/guajun/ethercat-configurator/internal/report"
	"github.com/guajun/ethercat-configurator/internal/sii"
)

const Version = "ethercat-configurator dev"

const (
	ExitSuccess         = 0
	ExitValidationError = 2
	ExitInputError      = 64
	ExitInternalError   = 70
)

type runOptions struct {
	JSON    bool
	Quiet   bool
	Version bool
	Help    bool
}

type cliError struct {
	exitCode    int
	diagnostic  diag.Diagnostic
	diagnostics []diag.Diagnostic
}

func (err cliError) Error() string {
	return err.diagnostic.Message
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	options, remaining, err := parseGlobal(args)
	if err != nil {
		return writeError(stderr, options, err)
	}

	if options.Version {
		if !options.Quiet {
			fmt.Fprintln(stdout, Version)
		}
		return ExitSuccess
	}

	if len(remaining) == 0 {
		writeHelp(stdout, "root")
		return ExitSuccess
	}

	if err := dispatch(remaining, options, stdout); err != nil {
		return writeError(stderr, options, err)
	}

	return ExitSuccess
}

func parseGlobal(args []string) (runOptions, []string, *cliError) {
	var options runOptions
	remaining := make([]string, 0, len(args))
	commandSeen := false

	for _, arg := range args {
		switch arg {
		case "--json", "-json":
			options.JSON = true
		case "--quiet":
			options.Quiet = true
		case "--version":
			options.Version = true
		case "--help", "-h":
			options.Help = true
		default:
			if strings.HasPrefix(arg, "--") && !commandSeen {
				return options, remaining, inputError("ECONFIG_INPUT_UNKNOWN_FLAG", fmt.Sprintf("unknown flag %q", arg))
			}
			if !strings.HasPrefix(arg, "-") {
				commandSeen = true
			}
			remaining = append(remaining, arg)
		}
	}

	return options, remaining, nil
}

func dispatch(args []string, options runOptions, stdout io.Writer) *cliError {
	command := args[0]

	if options.Help {
		return helpFor(args, stdout)
	}

	switch command {
	case "validate":
		return runValidate(args[1:], options, stdout)
	case "gen":
		return runGen(args[1:], options, stdout)
	case "inspect":
		return runInspect(args[1:], options, stdout)
	case "report":
		return runReport(args[1:], options, stdout)
	default:
		return inputError("ECONFIG_INPUT_UNKNOWN_COMMAND", fmt.Sprintf("unknown command %q", command))
	}
}

func helpFor(args []string, stdout io.Writer) *cliError {
	switch strings.Join(args, " ") {
	case "validate":
		writeHelp(stdout, "validate")
	case "gen":
		writeHelp(stdout, "gen")
	case "gen esi":
		writeHelp(stdout, "gen esi")
	case "gen sii":
		writeHelp(stdout, "gen sii")
	case "gen header":
		writeHelp(stdout, "gen header")
	case "inspect":
		writeHelp(stdout, "inspect")
	case "inspect esi":
		writeHelp(stdout, "inspect esi")
	case "inspect sii":
		writeHelp(stdout, "inspect sii")
	case "report":
		writeHelp(stdout, "report")
	default:
		return inputError("ECONFIG_INPUT_UNKNOWN_COMMAND", fmt.Sprintf("unknown command %q", strings.Join(args, " ")))
	}

	return nil
}

func runValidate(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "validate requires a device file or specs target")
	}
	if len(args) > 1 {
		return inputError("ECONFIG_INPUT_TOO_MANY_ARGUMENTS", "validate accepts exactly one target")
	}
	if args[0] != "specs" {
		return validateDevice(args[0], options, stdout)
	}
	if !options.Quiet {
		writeValidateResult(stdout, options, args[0])
	}
	return nil
}

func validateDevice(path string, options runOptions, stdout io.Writer) *cliError {
	loadResult := config.LoadFile(path)
	diagnostics := append([]diag.Diagnostic(nil), loadResult.Diagnostics...)
	layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
	diagnostics = append(diagnostics, layoutDiagnostics...)
	_, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
	diagnostics = append(diagnostics, addressDiagnostics...)
	if len(diagnostics) > 0 {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	if !options.Quiet {
		writeValidateResult(stdout, options, path)
	}
	return nil
}

func writeValidateResult(stdout io.Writer, options runOptions, target string) {
	if options.JSON {
		_ = diag.RenderJSON(stdout, diag.Result{Status: "ok", Target: target, Diagnostics: nil})
		return
	}

	fmt.Fprintf(stdout, "validate %s: ok\n", target)
}

func runGen(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "gen requires an artifact type")
	}

	switch args[0] {
	case "esi":
		return runGenESI(args[1:], options, stdout)
	case "sii":
		return runGenSII(args[1:], options, stdout)
	case "header":
		return runGenHeader(args[1:], options, stdout)
	default:
		return inputError("ECONFIG_INPUT_UNKNOWN_COMMAND", fmt.Sprintf("unknown gen artifact %q", args[0]))
	}
}

func runInspect(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "inspect requires an artifact type")
	}

	switch args[0] {
	case "esi":
		return runInspectESI(args[1:], options, stdout)
	case "sii":
		return runInspectSII(args[1:], options, stdout)
	default:
		return inputError("ECONFIG_INPUT_UNKNOWN_COMMAND", fmt.Sprintf("unknown inspect artifact %q", args[0]))
	}
}

func runGenESI(args []string, options runOptions, stdout io.Writer) *cliError {
	target, outputPath, err := parseOutputArgs(args, "gen esi", "device file")
	if err != nil {
		return err
	}
	device, layout, addressMap, diagnostics := loadValidatedDevice(target)
	if len(diagnostics) > 0 {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	data, generateErr := esi.Generate(device, layout, addressMap)
	if generateErr != nil {
		return internalError("ECONFIG_INTERNAL_ESI_GENERATE", generateErr.Error())
	}
	if writeErr := os.WriteFile(outputPath, data, 0o644); writeErr != nil {
		return internalError("ECONFIG_INTERNAL_ESI_WRITE", writeErr.Error())
	}
	if !options.Quiet {
		writeResult(stdout, options, "gen esi", target)
	}
	return nil
}

func runGenSII(args []string, options runOptions, stdout io.Writer) *cliError {
	target, outputPath, err := parseOutputArgs(args, "gen sii", "device file")
	if err != nil {
		return err
	}
	device, layout, addressMap, diagnostics := loadValidatedDevice(target)
	if len(diagnostics) > 0 {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	data, generateErr := sii.Generate(device, layout, addressMap)
	if generateErr != nil {
		return internalError("ECONFIG_INTERNAL_SII_GENERATE", generateErr.Error())
	}
	if writeErr := os.WriteFile(outputPath, data, 0o644); writeErr != nil {
		return internalError("ECONFIG_INTERNAL_SII_WRITE", writeErr.Error())
	}
	if !options.Quiet {
		writeResult(stdout, options, "gen sii", target)
	}
	return nil
}

func runGenHeader(args []string, options runOptions, stdout io.Writer) *cliError {
	target, outputPath, err := parseOutputArgs(args, "gen header", "device file")
	if err != nil {
		return err
	}
	device, layout, addressMap, diagnostics := loadValidatedDevice(target)
	if len(diagnostics) > 0 {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	data, warnings, generateErr := firmware.GenerateHeaderWithWarnings(device, layout, addressMap)
	if generateErr != nil {
		return internalError("ECONFIG_INTERNAL_HEADER_GENERATE", generateErr.Error())
	}
	if writeErr := os.WriteFile(outputPath, data, 0o644); writeErr != nil {
		return internalError("ECONFIG_INTERNAL_HEADER_WRITE", writeErr.Error())
	}
	if !options.Quiet {
		writeResult(stdout, options, "gen header", target)
		writeHeaderWarnings(stdout, warnings)
	}
	return nil
}

func writeHeaderWarnings(stdout io.Writer, warnings []firmware.HeaderWarning) {
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning.Message)
	}
}

func runInspectESI(args []string, options runOptions, stdout io.Writer) *cliError {
	target, err := parseInspectArgs(args, "inspect esi")
	if err != nil {
		return err
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		return &cliError{exitCode: ExitValidationError, diagnostic: diag.Error(diag.CodeESIParseError, readErr.Error())}
	}
	summary, diagnostics := esi.Parse(data)
	if len(diagnostics) > 0 {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	if options.JSON {
		if renderErr := esi.RenderJSON(stdout, summary); renderErr != nil {
			return internalError("ECONFIG_INTERNAL_ESI_RENDER", renderErr.Error())
		}
	} else if !options.Quiet {
		if renderErr := esi.RenderText(stdout, summary); renderErr != nil {
			return internalError("ECONFIG_INTERNAL_ESI_RENDER", renderErr.Error())
		}
	}
	return nil
}

func runInspectSII(args []string, options runOptions, stdout io.Writer) *cliError {
	target, err := parseInspectArgs(args, "inspect sii")
	if err != nil {
		return err
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		return &cliError{exitCode: ExitValidationError, diagnostic: diag.Error(diag.CodeSIIParseError, readErr.Error())}
	}
	summary, diagnostics := sii.Parse(data)
	if len(diagnostics) > 0 {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	if options.JSON {
		if renderErr := sii.RenderJSON(stdout, summary); renderErr != nil {
			return internalError("ECONFIG_INTERNAL_SII_RENDER", renderErr.Error())
		}
	} else if !options.Quiet {
		if renderErr := sii.RenderText(stdout, summary); renderErr != nil {
			return internalError("ECONFIG_INTERNAL_SII_RENDER", renderErr.Error())
		}
	}
	return nil
}

func runReport(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "report requires a device file")
	}
	target, outputPath, err := parseReportArgs(args)
	if err != nil {
		return err
	}

	loadResult := config.LoadFile(target)
	diagnostics := append([]diag.Diagnostic(nil), loadResult.Diagnostics...)
	if hasDiagnostic(diagnostics, diag.CodeConfigParseError) {
		return &cliError{exitCode: ExitValidationError, diagnostic: diagnostics[0], diagnostics: diagnostics}
	}
	layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
	diagnostics = append(diagnostics, layoutDiagnostics...)
	addressMap, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
	diagnostics = append(diagnostics, addressDiagnostics...)
	reportData := report.Build(target, loadResult.Device, layout, addressMap, diagnostics)

	var output strings.Builder
	if options.JSON {
		if renderErr := report.RenderJSON(&output, reportData); renderErr != nil {
			return internalError("ECONFIG_INTERNAL_REPORT_RENDER", renderErr.Error())
		}
	} else if renderErr := report.RenderMarkdown(&output, reportData); renderErr != nil {
		return internalError("ECONFIG_INTERNAL_REPORT_RENDER", renderErr.Error())
	}

	if outputPath != "" {
		if writeErr := os.WriteFile(outputPath, []byte(output.String()), 0o644); writeErr != nil {
			return internalError("ECONFIG_INTERNAL_REPORT_WRITE", writeErr.Error())
		}
		if !options.Quiet {
			fmt.Fprintf(stdout, "report %s: ok\n", target)
		}
		return nil
	}

	_, _ = io.WriteString(stdout, output.String())
	return nil
}

func parseReportArgs(args []string) (string, string, *cliError) {
	target := ""
	outputPath := ""
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "-o", "--output":
			if index+1 >= len(args) {
				return "", "", inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "report output flag requires a path")
			}
			index++
			outputPath = args[index]
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", inputError("ECONFIG_INPUT_UNKNOWN_FLAG", fmt.Sprintf("unknown report flag %q", arg))
			}
			if target != "" {
				return "", "", inputError("ECONFIG_INPUT_TOO_MANY_ARGUMENTS", "report accepts exactly one device file")
			}
			target = arg
		}
	}
	if target == "" {
		return "", "", inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "report requires a device file")
	}
	return target, outputPath, nil
}

func parseOutputArgs(args []string, command string, targetLabel string) (string, string, *cliError) {
	target := ""
	outputPath := ""
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "-o", "--output":
			if index+1 >= len(args) {
				return "", "", inputError("ECONFIG_INPUT_MISSING_ARGUMENT", command+" output flag requires a path")
			}
			index++
			outputPath = args[index]
		default:
			if strings.HasPrefix(arg, "-") {
				return "", "", inputError("ECONFIG_INPUT_UNKNOWN_FLAG", fmt.Sprintf("unknown %s flag %q", command, arg))
			}
			if target != "" {
				return "", "", inputError("ECONFIG_INPUT_TOO_MANY_ARGUMENTS", command+" accepts exactly one "+targetLabel)
			}
			target = arg
		}
	}
	if target == "" {
		return "", "", inputError("ECONFIG_INPUT_MISSING_ARGUMENT", command+" requires a "+targetLabel)
	}
	if outputPath == "" {
		return "", "", inputError("ECONFIG_INPUT_MISSING_ARGUMENT", command+" requires -o <path>")
	}
	return target, outputPath, nil
}

func parseInspectArgs(args []string, command string) (string, *cliError) {
	if len(args) == 0 {
		return "", inputError("ECONFIG_INPUT_MISSING_ARGUMENT", command+" requires an artifact file")
	}
	if len(args) > 1 {
		return "", inputError("ECONFIG_INPUT_TOO_MANY_ARGUMENTS", command+" accepts exactly one artifact file")
	}
	if strings.HasPrefix(args[0], "-") {
		return "", inputError("ECONFIG_INPUT_UNKNOWN_FLAG", fmt.Sprintf("unknown %s flag %q", command, args[0]))
	}
	return args[0], nil
}

func loadValidatedDevice(path string) (model.Device, model.Layout, model.AddressMap, []diag.Diagnostic) {
	loadResult := config.LoadFile(path)
	diagnostics := append([]diag.Diagnostic(nil), loadResult.Diagnostics...)
	if hasDiagnostic(diagnostics, diag.CodeConfigParseError) {
		return loadResult.Device, model.Layout{}, model.AddressMap{}, diagnostics
	}
	layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
	diagnostics = append(diagnostics, layoutDiagnostics...)
	addressMap, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
	diagnostics = append(diagnostics, addressDiagnostics...)
	return loadResult.Device, layout, addressMap, diagnostics
}

func hasDiagnostic(diagnostics []diag.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func writeResult(stdout io.Writer, options runOptions, command string, target string) {
	if options.JSON {
		result := struct {
			Command string `json:"command"`
			Target  string `json:"target"`
			Status  string `json:"status"`
		}{Command: command, Target: target, Status: "ok"}
		_ = diag.RenderJSON(stdout, diag.Result{Status: result.Status, Target: result.Target, Diagnostics: nil})
		return
	}

	fmt.Fprintf(stdout, "%s %s: ok\n", command, target)
}

func writeError(stderr io.Writer, options runOptions, err *cliError) int {
	diagnostics := err.diagnostics
	if len(diagnostics) == 0 {
		diagnostics = []diag.Diagnostic{err.diagnostic}
	}
	if options.JSON {
		_ = diag.RenderJSON(stderr, diag.Result{Status: "error", Diagnostics: diagnostics})
	} else if !options.Quiet {
		_ = diag.RenderText(stderr, diagnostics)
	}

	return err.exitCode
}

func inputError(code string, message string) *cliError {
	return &cliError{
		exitCode:   ExitInputError,
		diagnostic: diag.Error(code, message),
	}
}

func internalError(code string, message string) *cliError {
	return &cliError{
		exitCode:   ExitInternalError,
		diagnostic: diag.Error(code, message),
	}
}

func writeHelp(stdout io.Writer, topic string) {
	fmt.Fprint(stdout, helpText(topic))
}

func helpText(topic string) string {
	switch topic {
	case "root":
		return `EtherCAT Configurator

Usage:
  ethercat-configurator [--json] [--quiet] [--version] <command> [arguments]

Commands:
  validate   Validate a device file or specs metadata.
  report     Generate a reviewable report.
  gen        Generate EtherCAT artifacts.
  inspect    Inspect generated EtherCAT artifacts.

Global Flags:
  --json      Write machine-readable output where supported.
  --quiet     Suppress human-readable success output.
  --version   Print the CLI version.
  --help      Print deterministic usage text.
`
	case "validate":
		return `Usage:
  ethercat-configurator validate [--json] [--quiet] <device.yaml|specs>

Validates a declarative EtherCAT SubDevice configuration or specs metadata.
`
	case "gen":
		return `Usage:
  ethercat-configurator gen <artifact> [arguments]

Artifacts:
  esi      Generate minimal ESI XML.
  sii      Generate SII EEPROM binary.
  header   Generate firmware process-data header.
`
	case "gen esi":
		return `Usage:
  ethercat-configurator gen esi [--json] [--quiet] <device.yaml> -o <device.xml>

Generates minimal ESI XML from the canonical device model.
`
	case "gen sii":
		return `Usage:
  ethercat-configurator gen sii [--json] [--quiet] <device.yaml> -o <eeprom.bin>

Generates an SII EEPROM binary from the canonical device model.
`
	case "gen header":
		return `Usage:
  ethercat-configurator gen header [--json] [--quiet] <device.yaml> -o <ethercat_device.h>

Generates a firmware header from the process-data address map.
`
	case "inspect":
		return `Usage:
  ethercat-configurator inspect <artifact> [arguments]

Artifacts:
  esi      Inspect ESI XML.
  sii      Inspect SII EEPROM binary.
`
	case "inspect esi":
		return `Usage:
  ethercat-configurator inspect esi [--json] [--quiet] <device.xml>

Inspects ESI XML without requiring EtherCAT hardware.
`
	case "inspect sii":
		return `Usage:
  ethercat-configurator inspect sii [--json] [--quiet] <eeprom.bin>

Inspects an SII EEPROM binary without requiring EtherCAT hardware.
`
	case "report":
		return `Usage:
  ethercat-configurator report [--json] [--quiet] <device.yaml> [-o <report.md>]

Generates a deterministic review report from the canonical device model.
`
	default:
		return ""
	}
}
