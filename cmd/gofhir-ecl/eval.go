package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/internal/conformance"
)

func runEval(args []string) error {
	return runEvalWithOutput(args, os.Stdout)
}

func runEvalWithOutput(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr) // diagnostics go to stderr; out carries results only
	fixturePath := fs.String("fixture", "", "path to a YAML fixture defining the in-memory DataProvider (required)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gofhir-ecl eval --fixture <path> <expression>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Evaluates the expression against an in-memory provider built from the YAML")
		fmt.Fprintln(os.Stderr, "fixture. Prints the resulting concept IDs, one per line.")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *fixturePath == "" {
		fs.Usage()
		return fmt.Errorf("--fixture is required")
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("%w: expected exactly one expression argument", errUsage)
	}

	expr, err := readExpression(fs.Arg(0))
	if err != nil {
		return err
	}
	tree, err := ecl.Parse(expr)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	provider, err := conformance.LoadFixtureFile(*fixturePath)
	if err != nil {
		return fmt.Errorf("load fixture: %w", err)
	}

	result, err := ecl.Evaluate(context.Background(), tree, provider)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}
	ids := result.Slice()
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintln(out, id)
	}
	return nil
}
