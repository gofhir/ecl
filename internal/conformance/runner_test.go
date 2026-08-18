package conformance

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot resolves the repository root from this file's path so tests can
// reach testdata/ without depending on the caller's working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	// .../go-ecl/internal/conformance/runner_test.go → .../go-ecl
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// wantTotalCases is the number of bundled conformance cases. Raise it when you
// add cases — never lower it without explaining why in the commit message. The
// explicit count makes adding or deleting a case file a visible change in the
// diff instead of a silent loss of coverage.
const wantTotalCases = 91

// TestRunAllSuites runs EVERY bundled conformance suite against the bundled
// fixtures, one subtest per case.
//
// This test replaced an earlier one that loaded only 01-hierarchy.yaml, which
// left 36 of the 44 cases unexecuted by CI: they only ran if someone invoked
// the CLI by hand. A regression in refsets, filters, history, concrete values
// or v2.2 features could be merged with a green build.
func TestRunAllSuites(t *testing.T) {
	root := repoRoot(t)
	casesDir := filepath.Join(root, "testdata", "conformance", "cases")
	fixtureDir := filepath.Join(root, "testdata", "conformance", "fixtures")

	paths, err := filepath.Glob(filepath.Join(casesDir, "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no suites found in %s", casesDir)

	suites := make([]*Suite, 0, len(paths))
	for _, p := range paths {
		s, err := LoadSuite(p)
		require.NoErrorf(t, err, "LoadSuite(%s)", p)
		require.NotEmptyf(t, s.Cases, "suite %s has no cases", p)
		suites = append(suites, s)
	}

	rep, err := RunSuites(context.Background(), suites, RunOptions{FixtureDir: fixtureDir})
	require.NoError(t, err)

	for _, r := range rep.Results {
		t.Run(r.Suite+"/"+r.Case.Name, func(t *testing.T) {
			if r.Skipped {
				t.Skip(r.Reason)
			}
			if !r.Passed {
				t.Errorf("expression: %s\nreason: %s", r.Case.Expression, r.Reason)
			}
		})
	}

	passed, failed, skipped, total := rep.Summary()
	t.Logf("%d passed, %d failed, %d skipped, %d total", passed, failed, skipped, total)
	assert.Equal(t, 0, failed)
	assert.Equal(t, wantTotalCases, total, "case count changed; update wantTotalCases")
}

// TestRunCase_ExpectError verifies that a case marked ExpectError passes
// when the evaluator returns an error.
func TestRunCase_ExpectError(t *testing.T) {
	// MemberOf with field projection is documented to error out.
	spec := &FixtureSpec{
		Concepts: []FixtureConcept{{ID: "12345"}},
		Refsets:  map[string][]string{"12345": {"99999"}},
	}
	provider := NewInMemoryProvider(spec)

	c := Case{
		Name:        "memberOf with field projection errors",
		Expression:  "^[id] 12345",
		ExpectError: true,
	}
	got := runCase(context.Background(), "test", c, provider)
	if !got.Passed {
		t.Errorf("expected pass, got fail: %s", got.Reason)
	}
}

// TestFormatResult covers the human-readable result formatter.
func TestFormatResult(t *testing.T) {
	pass := Result{Suite: "S", Case: Case{Name: "c"}, Passed: true}
	if got := FormatResult(pass); !strings.HasPrefix(got, "PASS") {
		t.Errorf("pass should start with PASS: %q", got)
	}
	skip := Result{Suite: "S", Case: Case{Name: "c"}, Skipped: true, Reason: "tbd"}
	if got := FormatResult(skip); !strings.HasPrefix(got, "SKIP") {
		t.Errorf("skip should start with SKIP: %q", got)
	}
	fail := Result{Suite: "S", Case: Case{Name: "c"}, Reason: "expected=[a] got=[b]"}
	if got := FormatResult(fail); !strings.HasPrefix(got, "FAIL") {
		t.Errorf("fail should start with FAIL: %q", got)
	}
}

// TestEqualSlices is a small sanity check for the comparison helper.
func TestEqualSlices(t *testing.T) {
	assert.True(t, equalSlices([]string{}, []string{}))
	assert.True(t, equalSlices([]string{"a", "b"}, []string{"a", "b"}))
	assert.False(t, equalSlices([]string{"a"}, []string{"a", "b"}))
	assert.False(t, equalSlices([]string{"a", "b"}, []string{"a", "c"}))
}
