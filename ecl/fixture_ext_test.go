// Package ecl_test holds the shared test harness for the remediation work
// described in docs/plans/2026-08-18-ecl-remediation-plan.md.
//
// Why an external test package: the helpers below load
// testdata/conformance/fixtures/standard.yaml through internal/conformance,
// and internal/conformance imports github.com/gofhir/ecl/ecl. An in-package
// test file (package ecl) that imported it would be an import cycle
// ("import cycle not allowed in test"), so these helpers live in package
// ecl_test and use only the exported API.
//
// Consequence to keep in mind when writing tests: the in-package helpers
// (newFixture, evalECL, filterTestProvider) are NOT reachable from here, and
// the two fixtures are not interchangeable. Every test must say which fixture
// its expected IDs were computed against:
//
//   - standard.yaml (via evalFixture below) — used by the CLI, the conformance
//     runner and this harness.
//   - newFixture() in evaluator_test.go — a different concept set; notably it
//     has 404684004, which standard.yaml does not, and relationships in groups
//     1 and 2.
package ecl_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
	"github.com/gofhir/ecl/internal/conformance"
)

// repoRoot resolves the repository root from this file's path so the helpers
// reach testdata/ regardless of the caller's working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	// .../go-ecl/ecl/fixture_ext_test.go → .../go-ecl
	return filepath.Join(filepath.Dir(file), "..")
}

// standardFixturePath returns the path of the bundled standard fixture.
func standardFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "conformance", "fixtures", "standard.yaml")
}

// standardProvider loads the bundled standard fixture as a DataProvider.
func standardProvider(t *testing.T) ecl.DataProvider {
	t.Helper()
	p, err := conformance.LoadFixtureFile(standardFixturePath(t))
	require.NoError(t, err, "loading standard.yaml")
	return p
}

// evalFixture parses and evaluates expr against standard.yaml, failing the
// test on any error. Expected IDs in callers must be computed against
// standard.yaml, not against newFixture().
func evalFixture(t *testing.T, expr string) ecl.Set {
	t.Helper()
	set, err := evalFixtureErr(t, expr)
	require.NoErrorf(t, err, "evaluating %q", expr)
	return set
}

// evalFixtureErr is evalFixture but hands the error back instead of failing,
// for tests that assert on the error itself.
func evalFixtureErr(t *testing.T, expr string) (ecl.Set, error) {
	t.Helper()
	tree, err := ecl.Parse(expr)
	if err != nil {
		return nil, err
	}
	return ecl.Evaluate(context.Background(), tree, standardProvider(t))
}

// mustParse fails the test if expr does not parse.
func mustParse(t *testing.T, expr string) ast.Expression {
	t.Helper()
	tree, err := ecl.Parse(expr)
	require.NoErrorf(t, err, "parsing %q", expr)
	return tree
}

// standardSpec unmarshals standard.yaml into its spec so tests can assert on
// the fixture's own declarations rather than hardcoding them.
func standardSpec(t *testing.T) *conformance.FixtureSpec {
	t.Helper()
	data, err := os.ReadFile(standardFixturePath(t))
	require.NoError(t, err)
	var spec conformance.FixtureSpec
	require.NoError(t, yaml.Unmarshal(data, &spec))
	return &spec
}

// isActiveInFixture reports the active flag standard.yaml declares for id.
// A concept with no explicit flag is active, matching the fixture loader.
func isActiveInFixture(t *testing.T, id string) bool {
	t.Helper()
	for _, c := range standardSpec(t).Concepts {
		if c.ID == id {
			return c.Active == nil || *c.Active
		}
	}
	t.Fatalf("concept %s is not declared in standard.yaml", id)
	return false
}

// TestHarness exercises the helpers themselves, so a broken harness fails here
// instead of producing confusing failures in the tests that depend on it.
func TestHarness(t *testing.T) {
	set := evalFixture(t, "<< 73211009")
	require.ElementsMatch(t, []string{"11687002", "73211009"}, set.Slice())

	_, err := evalFixtureErr(t, "<< invalid!!!")
	require.Error(t, err)

	require.NotNil(t, mustParse(t, "<< 404684003"))

	require.True(t, isActiveInFixture(t, "22298006"))
	require.False(t, isActiveInFixture(t, "11111111"), "11111111 is declared active: false")
}

// exampleProvider loads the bundled fixture for the Example functions, which
// cannot take a *testing.T.
func exampleProvider() ecl.DataProvider {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "testdata", "conformance", "fixtures", "standard.yaml")
	p, err := conformance.LoadFixtureFile(path)
	if err != nil {
		panic(err)
	}
	return p
}
