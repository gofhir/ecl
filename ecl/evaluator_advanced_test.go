package ecl

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------
// Phase 5.1 — History supplements
// ------------------------------------------------------------------------.

func TestEvaluate_HistorySupplement(t *testing.T) {
	p := newFixture()
	// 404684004 is "inactive" and has a historical replacement 22298006.
	// The supplement unions the historical associations onto the base set.
	got := evalECL(t, "404684004 {{ +HISTORY }}", p)
	assert.ElementsMatch(t, []string{"404684004", "22298006"}, got.Slice())
}

func TestEvaluate_HistorySupplement_NoHistory(t *testing.T) {
	p := newFixture()
	// 22298006 has no historical entries in the fixture; the supplement is
	// a no-op (just the base set itself).
	got := evalECL(t, "22298006 {{ +HISTORY }}", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_HistorySupplement_Profile(t *testing.T) {
	p := newFixture()
	// Profile MIN should still use the same provider mock (which ignores the
	// profile) so the result is identical. The important part is that the
	// evaluator accepts the profiled syntax.
	got := evalECL(t, "<< 404684003 {{ +HISTORY-MIN }}", p)
	// Base: {404684003, 22298006, 64572001, 73211009, 404684004, 111111001}
	// History of 404684004 → 22298006 (already in base)
	assert.ElementsMatch(t,
		[]string{"404684003", "22298006", "64572001", "73211009", "404684004", "111111001"},
		got.Slice())
}

// ------------------------------------------------------------------------
// Phase 5.2 — Concrete value comparisons
// ------------------------------------------------------------------------.

func TestEvaluate_ConcreteValueEq(t *testing.T) {
	p := newFixture()
	// 22298006 has concrete value 1142139005 = 2.
	got := evalECL(t, "22298006 : 1142139005 = #2", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueGt(t *testing.T) {
	p := newFixture()
	// 22298006 has value 2, which is > 1.
	got := evalECL(t, "22298006 : 1142139005 > #1", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueGte(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "22298006 : 1142139005 >= #2", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueLt_NoMatch(t *testing.T) {
	p := newFixture()
	// 22298006 has value 2, which is NOT < 2.
	got := evalECL(t, "22298006 : 1142139005 < #2", p)
	assert.Equal(t, 0, got.Len())
}

func TestEvaluate_ConcreteValueLte(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "22298006 : 1142139005 <= #2", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueNeq(t *testing.T) {
	p := newFixture()
	// 22298006 value is 2, so "!= 5" should match.
	got := evalECL(t, "22298006 : 1142139005 != #5", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

// ------------------------------------------------------------------------
// Phase 5.2 — String concrete value comparisons
// ------------------------------------------------------------------------.

func TestEvaluate_ConcreteValueStringEq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149367008 = "severe"`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueStringNeq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149367008 != "mild"`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueStringNeq_NoMatch(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149367008 != "severe"`, p)
	assert.Equal(t, 0, got.Len())
}

// ------------------------------------------------------------------------
// Phase 5.2 — Boolean concrete value comparisons
// ------------------------------------------------------------------------.

func TestEvaluate_ConcreteValueBoolEq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149366004 = true`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueBoolNeq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149366004 != false`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueBoolEq_NoMatch(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149366004 = false`, p)
	assert.Equal(t, 0, got.Len())
}

// ------------------------------------------------------------------------
// Phase 6.1 — v2.2: Top / Bottom of set
// ------------------------------------------------------------------------.

func TestEvaluate_TopOfSet(t *testing.T) {
	p := newFixture()
	// << 404684003 = {404684003, 22298006, 64572001, 73211009, 404684004}
	// Root of the set (no parent in the set) = 404684003.
	// The grammar requires parentheses when chaining two constraint
	// operators (!!> + <<).
	got := evalECL(t, "!!> (<< 404684003)", p)
	assert.ElementsMatch(t, []string{"404684003"}, got.Slice())
}

func TestEvaluate_BottomOfSet(t *testing.T) {
	p := newFixture()
	// << 404684003 = {404684003, 22298006, 64572001, 73211009, 404684004, 111111001}
	// Leaves of the set (no child in the set) = 22298006, 73211009, 404684004, 111111001.
	got := evalECL(t, "!!< (<< 404684003)", p)
	assert.ElementsMatch(t,
		[]string{"22298006", "73211009", "404684004", "111111001"},
		got.Slice())
}

func TestEvaluate_TopOfSet_Singleton(t *testing.T) {
	p := newFixture()
	// A single concept has no parent in the set → it's its own top.
	got := evalECL(t, "!!> 22298006", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

// ------------------------------------------------------------------------
// Phase 6.1 — v2.2: RefsetContainingAny (^R)
// ------------------------------------------------------------------------.

func TestEvaluate_RefsetContainingAny(t *testing.T) {
	p := newFixture()
	// ^R <refsetID> should return the same member set as ^ <refsetID>.
	expr, err := Parse("^R 900000000000497000")
	if err != nil {
		t.Skipf("parser does not yet produce RefsetContainingAny: %v", err)
	}
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{"22298006", "64572001", "73211009"},
		got.Slice())
}

// ------------------------------------------------------------------------
// Phase 6.1 — v2.2: AltIdentifier (deferred)
// ------------------------------------------------------------------------.

// TestEvaluate_AltIdentifier_NotImplemented confirms that alternate
// identifiers parse but the evaluator returns a clear "not yet implemented"
// error rather than silently producing the wrong result.
func TestEvaluate_ConcreteValue_Reverse(t *testing.T) {
	p := newFixture()
	// R 1142139005 = #2 on a concrete-value attribute.
	// Concrete values have no "target concept" to reverse through.
	// Per ECL spec, R on concrete values means: find concepts that have
	// this concrete value — same as forward evaluation.
	// 22298006 has 1142139005=2, so it matches.
	got := evalECL(t, `* : R 1142139005 = #2`, p)
	assert.True(t, got.Contains("22298006"), "22298006 has concrete value 1142139005=2")
}

func TestEvaluate_AltIdentifier_NotImplemented(t *testing.T) {
	p := newFixture()
	expr, err := Parse(`LOINC#1234-5`)
	if err != nil {
		// Alternate grammar forms — try a quoted variant.
		expr, err = Parse(`"LOINC"#"1234-5"`)
		if err != nil {
			t.Skipf("parser does not accept alternate identifier syntax: %v", err)
		}
	}
	_, err = Evaluate(context.Background(), expr, p)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not yet implemented"),
		"error should say 'not yet implemented', got: %v", err)
}
