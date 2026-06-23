package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const Version = "ethercat-configurator dev"

const (
	ExitSuccess         = 0
	ExitValidationError = 2
	ExitInputError      = 64
	ExitInternalError   = 70
)

type diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type runOptions struct {
	JSON    bool
	Quiet   bool
	Version bool
	Help    bool
}

type cliError struct {
	exitCode   int
	diagnostic diagnostic
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

	for _, arg := range args {
		switch arg {
		case "--json":
			options.JSON = true
		case "--quiet":
			options.Quiet = true
		case "--version":
			options.Version = true
		case "--help", "-h":
			options.Help = true
		default:
			if strings.HasPrefix(arg, "--") {
				return options, remaining, inputError("ECONFIG_INPUT_UNKNOWN_FLAG", fmt.Sprintf("unknown flag %q", arg))
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
	if !options.Quiet {
		writeResult(stdout, options, "validate", args[0])
	}
	return nil
}

func runGen(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "gen requires an artifact type")
	}

	switch args[0] {
	case "esi", "sii", "header":
		if len(args) == 1 {
			return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", fmt.Sprintf("gen %s requires a device file", args[0]))
		}
		if !options.Quiet {
			writeResult(stdout, options, "gen "+args[0], args[1])
		}
		return nil
	default:
		return inputError("ECONFIG_INPUT_UNKNOWN_COMMAND", fmt.Sprintf("unknown gen artifact %q", args[0]))
	}
}

func runInspect(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "inspect requires an artifact type")
	}

	switch args[0] {
	case "esi", "sii":
		if len(args) == 1 {
			return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", fmt.Sprintf("inspect %s requires an artifact file", args[0]))
		}
		if !options.Quiet {
			writeResult(stdout, options, "inspect "+args[0], args[1])
		}
		return nil
	default:
		return inputError("ECONFIG_INPUT_UNKNOWN_COMMAND", fmt.Sprintf("unknown inspect artifact %q", args[0]))
	}
}

func runReport(args []string, options runOptions, stdout io.Writer) *cliError {
	if len(args) == 0 {
		return inputError("ECONFIG_INPUT_MISSING_ARGUMENT", "report requires a device file")
	}
	if !options.Quiet {
		writeResult(stdout, options, "report", args[0])
	}
	return nil
}

func writeResult(stdout io.Writer, options runOptions, command string, target string) {
	if options.JSON {
		result := struct {
			Command string `json:"command"`
			Target  string `json:"target"`
			Status  string `json:"status"`
		}{Command: command, Target: target, Status: "ok"}
		_ = json.NewEncoder(stdout).Encode(result)
		return
	}

	fmt.Fprintf(stdout, "%s %s: ok\n", command, target)
}

func writeError(stderr io.Writer, options runOptions, err *cliError) int {
	if options.JSON {
		payload := struct {
			Diagnostics []diagnostic `json:"diagnostics"`
		}{Diagnostics: []diagnostic{err.diagnostic}}
		_ = json.NewEncoder(stderr).Encode(payload)
	} else if !options.Quiet {
		fmt.Fprintf(stderr, "%s: %s\n", err.diagnostic.Code, err.diagnostic.Message)
	}

	return err.exitCode
}

func inputError(code string, message string) *cliError {
	return &cliError{
		exitCode: ExitInputError,
		diagnostic: diagnostic{
			Code:     code,
			Severity: "error",
			Message:  message,
		},
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
