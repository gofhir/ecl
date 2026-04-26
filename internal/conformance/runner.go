package conformance

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gofhir/ecl/ecl"
)

// Result captures the outcome of a single case run.
type Result struct {
	Suite   string
	Case    Case
	Passed  bool
	Skipped bool
	Reason  string // failure reason or skip note; empty on pass
}

// Report aggregates the outcome of a full run.
type Report struct {
	Results []Result
}

// Summary returns counts: passed, failed, skipped, total.
func (r *Report) Summary() (passed, failed, skipped, total int) {
	for _, res := range r.Results {
		total++
		switch {
		case res.Skipped:
			skipped++
		case res.Passed:
			passed++
		default:
			failed++
		}
	}
	return
}

// RunOptions controls suite execution.
type RunOptions struct {
	// Filter, when non-nil, restricts execution to cases whose name
	// matches the regex.
	Filter *regexp.Regexp

	// FixtureDir is the directory used to resolve relative fixture paths
	// declared in suite files.
	FixtureDir string
}

// RunSuite executes a single suite against the configured fixture and
// returns a Report.
func RunSuite(ctx context.Context, suite *Suite, opts RunOptions) (*Report, error) {
	provider, err := loadFixtureRelative(suite.Fixture, opts.FixtureDir)
	if err != nil {
		return nil, fmt.Errorf("loading fixture %q: %w", suite.Fixture, err)
	}
	rep := &Report{}
	for _, c := range suite.Cases {
		if opts.Filter != nil && !opts.Filter.MatchString(c.Name) {
			continue
		}
		rep.Results = append(rep.Results, runCase(ctx, suite.Name, c, provider))
	}
	return rep, nil
}

// RunSuites executes a slice of suites concatenating their reports.
func RunSuites(ctx context.Context, suites []*Suite, opts RunOptions) (*Report, error) {
	combined := &Report{}
	for _, s := range suites {
		r, err := RunSuite(ctx, s, opts)
		if err != nil {
			return combined, err
		}
		combined.Results = append(combined.Results, r.Results...)
	}
	return combined, nil
}

func runCase(ctx context.Context, suiteName string, c Case, provider ecl.DataProvider) Result {
	if c.Skip != "" {
		return Result{Suite: suiteName, Case: c, Skipped: true, Reason: c.Skip}
	}
	tree, err := ecl.Parse(c.Expression)
	if err != nil {
		if c.ExpectError {
			return Result{Suite: suiteName, Case: c, Passed: true}
		}
		return Result{Suite: suiteName, Case: c, Passed: false,
			Reason: fmt.Sprintf("parse error: %v", err)}
	}
	got, err := ecl.Evaluate(ctx, tree, provider)
	if err != nil {
		if c.ExpectError {
			return Result{Suite: suiteName, Case: c, Passed: true}
		}
		return Result{Suite: suiteName, Case: c, Passed: false,
			Reason: fmt.Sprintf("eval error: %v", err)}
	}
	if c.ExpectError {
		return Result{Suite: suiteName, Case: c, Passed: false,
			Reason: "expected an error but evaluation succeeded"}
	}
	gotIDs := got.Slice()
	sort.Strings(gotIDs)
	wantIDs := append([]string(nil), c.ExpectedIDs...)
	sort.Strings(wantIDs)
	if !equalSlices(gotIDs, wantIDs) {
		return Result{Suite: suiteName, Case: c, Passed: false,
			Reason: fmt.Sprintf("expected=%v got=%v", wantIDs, gotIDs)}
	}
	return Result{Suite: suiteName, Case: c, Passed: true}
}

func loadFixtureRelative(name, dir string) (ecl.DataProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("suite has no fixture set")
	}
	path := name
	if !filepath.IsAbs(path) && dir != "" {
		path = filepath.Join(dir, name)
	}
	return LoadFixtureFile(path)
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// FormatResult returns a one-line summary of the case outcome.
func FormatResult(r Result) string {
	switch {
	case r.Skipped:
		return fmt.Sprintf("SKIP  %s :: %s — %s", r.Suite, r.Case.Name, r.Reason)
	case r.Passed:
		return fmt.Sprintf("PASS  %s :: %s", r.Suite, r.Case.Name)
	default:
		return fmt.Sprintf("FAIL  %s :: %s\n      %s", r.Suite, r.Case.Name,
			strings.ReplaceAll(r.Reason, "\n", "\n      "))
	}
}
