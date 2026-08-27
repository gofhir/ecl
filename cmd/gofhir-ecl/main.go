// Command gofhir-ecl is a CLI for parsing, explaining, and evaluating SNOMED CT
// Expression Constraint Language (ECL) expressions, plus running a curated
// conformance suite against the gofhir/ecl evaluator.
//
// Subcommands:
//
//	validate    parse an ECL expression and report syntax errors
//	explain     parse and pretty-print the AST
//	eval        evaluate an ECL expression against an in-memory YAML fixture
//	conformance run the bundled v2.2 conformance suite
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gofhir/ecl/ecl"
)

const usage = `gofhir-ecl — SNOMED CT ECL parser, evaluator, and conformance runner

Usage:
  gofhir-ecl <command> [flags] [args]

Commands:
  validate     parse an ECL expression; report syntax errors
  explain      parse and pretty-print the AST
  eval         evaluate against an in-memory YAML fixture
  conformance  run the bundled v2.2 conformance suite
  version      print the build version
  help         print this help

Run "gofhir-ecl <command> -h" for command-specific flags.

Exit codes:
  0  success (including -h)
  1  runtime error
  2  usage error
  3  invalid ECL syntax
  4  ECL feature not supported by this build
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
	case "validate":
		err = runValidate(args)
	case "explain":
		err = runExplain(args)
	case "eval":
		err = runEval(args)
	case "conformance":
		err = runConformance(args)
	case "version":
		fmt.Println(version)
	case "help", "-h", "--help":
		fmt.Print(usage)
		os.Exit(exitOK)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}

	os.Exit(exitCode(err))
}

// Exit codes. Distinguishing them lets a caller tell "your expression is wrong"
// from "this build cannot do that yet".
const (
	exitOK          = 0
	exitError       = 1
	exitUsage       = 2
	exitInvalidECL  = 3
	exitUnsupported = 4
)

// errUsage marks a wrong invocation: a missing or extra argument. Kept distinct
// from a runtime failure so scripts can tell "you called me wrong" from
// "something broke".
var errUsage = errors.New("usage")

// exitCode maps an error to a process exit code. Note that flag.ErrHelp is not a
// failure: -h used to exit 1 with an "error: flag: help requested" prefix.
func exitCode(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	}

	fmt.Fprintf(os.Stderr, "error: %v\n", err)

	var pe *ecl.ParseError
	switch {
	case errors.Is(err, errUsage):
		return exitUsage
	case errors.As(err, &pe):
		return exitInvalidECL
	case errors.Is(err, ecl.ErrUnsupportedFeature):
		return exitUnsupported
	default:
		return exitError
	}
}

// version is overridden at build time via -ldflags="-X main.version=...".
var version = "dev"
