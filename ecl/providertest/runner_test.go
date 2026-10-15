package providertest

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// wantTotalCases is the number of bundled conformance cases. Raise it when you
// add cases — never lower it without explaining why in the commit message. The
// explicit count makes adding or deleting a case file a visible change in the
// diff instead of a silent loss of coverage.
const wantTotalCases = 154

// TestRunAllSuites runs EVERY bundled conformance suite against the bundled
// fixtures, one subtest per case.
//
// This test replaced an earlier one that loaded only 01-hierarchy.yaml, which
// left 36 of the 44 cases unexecuted by CI: they only ran if someone invoked
// the CLI by hand. A regression in refsets, filters, history, concrete values
// or v2.2 features could be merged with a green build.
func TestRunAllSuites(t *testing.T) {
	suites, err := LoadBundledSuites()
	require.NoError(t, err)
	require.NotEmpty(t, suites)
	for _, s := range suites {
		require.NotEmptyf(t, s.Cases, "suite %s has no cases", s.Name)
	}

	rep, err := RunBundled(context.Background(), nil)
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

// TestRunAllSuites_WithoutRowMatcher runs the bundled suites against the
// reference provider with ecl.DescriptionRowMatcher hidden, so the
// per-dimension path every provider without the capability takes keeps its
// coverage. A case that requires the capability passes there by being refused.
func TestRunAllSuites_WithoutRowMatcher(t *testing.T) {
	suites, err := LoadBundledSuites()
	require.NoError(t, err)
	rep, err := RunSuites(context.Background(), suites, RunOptions{LoadFixture: func(name string) (ecl.DataProvider, error) {
		p, err := BundledFixture(name)
		if err != nil {
			return nil, err
		}
		return WithoutRowMatcher(p), nil
	}})
	require.NoError(t, err)

	requiring := 0
	for _, s := range suites {
		for _, c := range s.Cases {
			if slices.Contains(c.Requires, "DescriptionRowMatcher") {
				requiring++
			}
		}
	}
	require.Positive(t, requiring, "some bundled case requires DescriptionRowMatcher, or the yaml field is not read")

	refused := 0
	for _, r := range rep.Results {
		assert.Truef(t, r.Passed || r.Skipped, "%s/%s without DescriptionRowMatcher: %s", r.Suite, r.Case.Name, r.Reason)
		if strings.HasPrefix(r.Reason, "refused as unsupported") {
			refused++
		}
	}
	require.Equal(t, requiring, refused, "the cases that require DescriptionRowMatcher, refused without it")
}

// TestWithoutRowMatcher_ForwardsEveryOtherCapability reads the capability
// interfaces from ecl/capabilities.go and checks that WithoutRowMatcher keeps
// every method of each but DescriptionRowMatcher. A capability added to the
// reference provider and forgotten there would otherwise make the
// per-dimension tests run without it, and stay green.
func TestWithoutRowMatcher_ForwardsEveryOtherCapability(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "../capabilities.go", nil, 0)
	require.NoError(t, err)
	wrapper := reflect.TypeOf(WithoutRowMatcher(NewInMemoryProvider(&FixtureSpec{})))
	reference := reflect.TypeOf(NewInMemoryProvider(&FixtureSpec{}))

	checked := 0
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok || ts.Name.Name == "DescriptionRowMatcher" {
				continue
			}
			for _, m := range iface.Methods.List {
				for _, name := range m.Names {
					if _, ok := reference.MethodByName(name.Name); !ok {
						continue // a capability the reference provider does not implement
					}
					checked++
					_, ok := wrapper.MethodByName(name.Name)
					assert.Truef(t, ok, "WithoutRowMatcher drops %s.%s; add the capability to its struct", ts.Name.Name, name.Name)
				}
			}
		}
	}
	require.NotZero(t, checked, "no capability method was found to check")
	_, rows := wrapper.MethodByName("MatchDescriptionRows")
	require.False(t, rows, "WithoutRowMatcher must hide DescriptionRowMatcher")
}
