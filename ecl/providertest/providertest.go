// Package providertest runs the bundled ECL v2.2 conformance suite against a
// DataProvider implementation.
//
// It exists because the contract a DataProvider has to satisfy is not fully
// expressible in the interface's godoc — the direction of HistoricalAssociations,
// which method owns the active flag, whether nil means "empty" or "wildcard" — and
// the only honest way to check an implementation is to run the suite against it:
//
//	func TestMyProvider(t *testing.T) {
//	    providertest.Verify(t, func() ecl.DataProvider { return newMyProvider(t) })
//	}
//
// This code used to live under internal/, which Go forbids third parties from
// importing, while the README offered it as the reference implementors should
// read. The suites and fixtures are embedded, so they travel with the package and
// work from any working directory — including from a `go install`ed binary, where
// the previous relative-path lookup failed outright.
package providertest

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"testing"

	"github.com/gofhir/ecl/ecl"
)

// bundled holds the conformance suites and fixtures.
//
// The directory lives inside this package because a //go:embed pattern cannot
// escape the package directory: `//go:embed ../../testdata` does not compile.
//
//go:embed testdata
var bundled embed.FS

const (
	casesDir    = "testdata/cases"
	fixturesDir = "testdata/fixtures"
)

// LoadBundledSuites parses every conformance suite that ships with this package.
func LoadBundledSuites() ([]*Suite, error) {
	entries, err := fs.ReadDir(bundled, casesDir)
	if err != nil {
		return nil, fmt.Errorf("read bundled cases: %w", err)
	}
	suites := make([]*Suite, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".yaml" {
			continue
		}
		data, err := bundled.ReadFile(path.Join(casesDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		suite, err := ParseSuite(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		suites = append(suites, suite)
	}
	if len(suites) == 0 {
		return nil, fmt.Errorf("no bundled suites found in %s", casesDir)
	}
	return suites, nil
}

// BundledFixture loads one of the fixtures that ship with this package as an
// in-memory DataProvider. Pass "standard.yaml" for the default one.
func BundledFixture(name string) (ecl.DataProvider, error) {
	data, err := bundled.ReadFile(path.Join(fixturesDir, path.Base(name)))
	if err != nil {
		return nil, fmt.Errorf("read bundled fixture %s: %w", name, err)
	}
	return LoadFixture(data)
}

// BundledFixtureBytes returns the raw YAML of a bundled fixture, for callers that
// want to inspect the declarations rather than evaluate against them.
func BundledFixtureBytes(name string) ([]byte, error) {
	data, err := bundled.ReadFile(path.Join(fixturesDir, path.Base(name)))
	if err != nil {
		return nil, fmt.Errorf("read bundled fixture %s: %w", name, err)
	}
	return data, nil
}

// RunBundled runs every bundled suite against the bundled fixtures and returns
// the report. Used by the CLI and by the package's own tests.
//
// The filter, when non-nil, restricts execution to cases whose name matches it.
func RunBundled(ctx context.Context, filter *regexp.Regexp) (*Report, error) {
	suites, err := LoadBundledSuites()
	if err != nil {
		return nil, err
	}
	return RunSuites(ctx, suites, RunOptions{Filter: filter, LoadFixture: BundledFixture})
}

// Verify runs the bundled conformance suite against the provider that
// newProvider returns, reporting one subtest per case.
//
// A fresh provider is requested per case so state cannot leak between them.
// Cases whose expectations depend on data the provider does not have will fail —
// that is the point: the suite is the executable statement of the contract.
func Verify(t *testing.T, newProvider func() ecl.DataProvider) {
	t.Helper()

	suites, err := LoadBundledSuites()
	if err != nil {
		t.Fatalf("loading bundled suites: %v", err)
	}

	for _, suite := range suites {
		for _, c := range suite.Cases {
			t.Run(suite.Name+"/"+c.Name, func(t *testing.T) {
				res := runCase(context.Background(), suite.Name, c, newProvider())
				switch {
				case res.Skipped:
					t.Skip(res.Reason)
				case !res.Passed:
					t.Errorf("expression: %s\nreason: %s", c.Expression, res.Reason)
				}
			})
		}
	}
}
