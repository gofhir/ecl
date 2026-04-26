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
	"fmt"
	"os"
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// version is overridden at build time via -ldflags="-X main.version=...".
var version = "dev"
