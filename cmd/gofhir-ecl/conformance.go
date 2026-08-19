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

	"github.com/gofhir/ecl/ecl/providertest"
)

func runConformance(args []string) error {
	return runConformanceWithOutput(args, os.Stdout)
}

func runConformanceWithOutput(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	fs.SetOutput(os.Stderr) // diagnostics go to stderr; out carries results only
	// Empty by default: the suites and fixtures are embedded in the
	// providertest package, so the command works from any directory. It used to
	// default to a path relative to the working directory, which meant a
	// `go install`ed binary failed with "no such file or directory" -- the exact
	// use the README recommends for CI.
	suiteDir := fs.String("suites", "", "directory of suite YAML files (default: the embedded suite)")
	fixtureDir := fs.String("fixtures", "", "directory of fixture YAML files referenced by suites (default: embedded)")
	filter := fs.String("filter", "", "regex; only cases whose name matches are run")
	verbose := fs.Bool("v", false, "print PASS lines (failures and skips are always printed)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gofhir-ecl conformance [-suites <dir>] [-fixtures <dir>] [-filter <regex>] [-v]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Runs the bundled ECL v2.2 conformance suite. Exits with a non-zero status")
		fmt.Fprintln(os.Stderr, "if any case fails.")
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

	rep, err := runSuites(context.Background(), *suiteDir, *fixtureDir, pattern)
	if err != nil {
		return err
	}

	for _, r := range rep.Results {
		if r.Passed && !*verbose {
			continue
		}
		fmt.Fprintln(out, providertest.FormatResult(r))
	}

	passed, failed, skipped, total := rep.Summary()
	fmt.Fprintf(out, "\n%d passed, %d failed, %d skipped, %d total\n", passed, failed, skipped, total)
	if failed > 0 {
		return fmt.Errorf("%d case(s) failed", failed)
	}
	return nil
}

// runSuites executes the embedded suite, or the one in suiteDir when the caller
// points at a directory of their own.
func runSuites(ctx context.Context, suiteDir, fixtureDir string, pattern *regexp.Regexp) (*providertest.Report, error) {
	if suiteDir == "" {
		return providertest.RunBundled(ctx, pattern)
	}
	suites, err := loadSuites(suiteDir)
	if err != nil {
		return nil, err
	}
	if len(suites) == 0 {
		return nil, fmt.Errorf("no suite YAML files found in %q", suiteDir)
	}
	if fixtureDir == "" {
		// Suites declare fixture paths relative to their own directory.
		fixtureDir = filepath.Join(suiteDir, "..", "fixtures")
	}
	return providertest.RunSuites(ctx, suites, providertest.RunOptions{
		Filter:     pattern,
		FixtureDir: fixtureDir,
	})
}

func loadSuites(dir string) ([]*providertest.Suite, error) {
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

	suites := make([]*providertest.Suite, 0, len(paths))
	for _, p := range paths {
		s, err := providertest.LoadSuite(p)
		if err != nil {
			return nil, fmt.Errorf("loading suite %q: %w", p, err)
		}
		suites = append(suites, s)
	}
	return suites, nil
}
