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

// TestRunSuite_Hierarchy loads the bundled hierarchy suite and confirms it
// passes against the bundled standard fixture. This guarantees the
// conformance runner stays correct as we extend the suite.
func TestRunSuite_Hierarchy(t *testing.T) {
	root := repoRoot(t)
	suitePath := filepath.Join(root, "testdata", "conformance", "cases", "01-hierarchy.yaml")
	fixtureDir := filepath.Join(root, "testdata", "conformance", "fixtures")

	suite, err := LoadSuite(suitePath)
	require.NoError(t, err)
	require.NotEmpty(t, suite.Cases)

	rep, err := RunSuite(context.Background(), suite, RunOptions{FixtureDir: fixtureDir})
	require.NoError(t, err)

	passed, failed, skipped, total := rep.Summary()
	if failed > 0 {
		for _, r := range rep.Results {
			if !r.Passed && !r.Skipped {
				t.Errorf("FAIL %s: %s", r.Case.Name, r.Reason)
			}
		}
	}
	assert.Equal(t, len(suite.Cases), total)
	assert.Equal(t, 0, failed)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, total, passed)
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
