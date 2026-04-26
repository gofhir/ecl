package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/gofhir/ecl/internal/conformance"
)

func runConformance(args []string) error {
	return runConformanceWithOutput(args, os.Stdout)
}

func runConformanceWithOutput(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	fs.SetOutput(out)
	suiteDir := fs.String("suites", "testdata/conformance/cases", "directory containing suite YAML files")
	fixtureDir := fs.String("fixtures", "testdata/conformance/fixtures", "directory for fixture YAML files referenced by suites")
	filter := fs.String("filter", "", "regex; only cases whose name matches are run")
	verbose := fs.Bool("v", false, "print PASS lines (failures and skips are always printed)")
	fs.Usage = func() {
		fmt.Fprintln(out, "Usage: gofhir-ecl conformance [-suites <dir>] [-fixtures <dir>] [-filter <regex>] [-v]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Runs the bundled ECL v2.2 conformance suite. Exits with a non-zero status")
		fmt.Fprintln(out, "if any case fails.")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	var pattern *regexp.Regexp
	if *filter != "" {
		p, err := regexp.Compile(*filter)
		if err != nil {
			return fmt.Errorf("invalid -filter regex: %w", err)
		}
		pattern = p
	}

	suites, err := loadSuites(*suiteDir)
	if err != nil {
		return err
	}
	if len(suites) == 0 {
		return fmt.Errorf("no suite YAML files found in %q", *suiteDir)
	}

	rep, err := conformance.RunSuites(context.Background(), suites, conformance.RunOptions{
		Filter:     pattern,
		FixtureDir: *fixtureDir,
	})
	if err != nil {
		return err
	}

	for _, r := range rep.Results {
		if r.Passed && !*verbose {
			continue
		}
		fmt.Fprintln(out, conformance.FormatResult(r))
	}

	passed, failed, skipped, total := rep.Summary()
	fmt.Fprintf(out, "\n%d passed, %d failed, %d skipped, %d total\n", passed, failed, skipped, total)
	if failed > 0 {
		return fmt.Errorf("%d case(s) failed", failed)
	}
	return nil
}

func loadSuites(dir string) ([]*conformance.Suite, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read suites dir %q: %w", dir, err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)

	suites := make([]*conformance.Suite, 0, len(paths))
	for _, p := range paths {
		s, err := conformance.LoadSuite(p)
		if err != nil {
			return nil, fmt.Errorf("loading suite %q: %w", p, err)
		}
		suites = append(suites, s)
	}
	return suites, nil
}
