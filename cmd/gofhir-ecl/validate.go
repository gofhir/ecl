package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gofhir/ecl/ecl"
)

func runValidate(args []string) error {
	return runValidateWithOutput(args, os.Stdout)
}

// runValidateWithOutput is the testable seam: callers can capture stdout.
func runValidateWithOutput(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr) // diagnostics go to stderr; out carries results only
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gofhir-ecl validate <expression>")
		fmt.Fprintln(os.Stderr, "       gofhir-ecl validate -")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Reads the expression from the given argument, or from stdin when '-' is given.")
		fmt.Fprintln(os.Stderr, "Exits 0 on valid syntax, non-zero with a parse error otherwise.")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("%w: expected exactly one expression argument", errUsage)
	}

	expr, err := readExpression(fs.Arg(0))
	if err != nil {
		return err
	}
	if _, err := ecl.Parse(expr); err != nil {
		return fmt.Errorf("invalid ECL: %w", err)
	}
	fmt.Fprintln(out, "OK")
	return nil
}

// readExpression returns the literal argument, or the contents of stdin when
// the argument is "-".
func readExpression(arg string) (string, error) {
	if arg != "-" {
		return arg, nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(b), nil
}
