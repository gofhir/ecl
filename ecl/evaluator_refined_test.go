package ecl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------
// Phase 3.3 — Ungrouped refinement ( : attr = value )
// ------------------------------------------------------------------------.

func TestEvaluate_SimpleRefinement_Eq(t *testing.T) {
	p := newFixture()
	// All descendants-or-self of Clinical finding with Finding site = Myocardium.
	// Both MI (22298006) and 404684004 have finding site = myocardium.
	got := evalECL(t, "<< 404684003 : 363698007 = 74281007", p)
	assert.ElementsMatch(t, []string{"22298006", "404684004"}, got.Slice())
}

func TestEvaluate_SimpleRefinement_Eq_HierarchicalValue(t *testing.T) {
	p := newFixture()
	// Finding site = << 74281007 (myocardium or any descendant). In this
	// fixture myocardium has no descendants, so same result as flat equality.
	got := evalECL(t, "<< 404684003 : 363698007 = << 74281007", p)
	assert.ElementsMatch(t, []string{"22298006", "404684004"}, got.Slice())
}

func TestEvaluate_SimpleRefinement_NotEq(t *testing.T) {
	p := newFixture()
	// Concepts in << 404684003 whose Finding site is NOT myocardium.
	// Per our AND-across-focus semantics, "!=" matches concepts that do not
	// have an outbound 363698007 relationship to 74281007.
	//   MI (22298006)       — has 363698007=74281007  → excluded
	//   diabetes (73211009) — has 363698007=113331007 → included (no rel to 74281007)
	//   404684004           — has 363698007=74281007  → excluded
	//   64572001, 404684003 — no such relationship    → included
	got := evalECL(t, "<< 404684003 : 363698007 != 74281007", p)
	assert.ElementsMatch(t, []string{"404684003", "64572001", "73211009"}, got.Slice())
}

func TestEvaluate_Refinement_MultipleAttrs_AND(t *testing.T) {
	p := newFixture()
	// Multiple ungrouped attrs separated by comma → conjoined (AND).
	// Concepts must have Finding site = myocardium AND Associated morphology = infarct.
	// Both MI (22298006) and 404684004 have both attrs individually (even if
	// in different groups for 404684004 — ungrouped semantics don't care).
	got := evalECL(t, "<< 404684003 : 363698007 = 74281007, 116676008 = 55641003", p)
	assert.ElementsMatch(t, []string{"22298006", "404684004"}, got.Slice())
}

// ------------------------------------------------------------------------
// Phase 3.4 — Grouped refinement ( : { a = v1, b = v2 } )
// ------------------------------------------------------------------------.

func TestEvaluate_GroupedRefinement(t *testing.T) {
	p := newFixture()
	// Grouped: both attributes must be in the SAME relationship group.
	// MI has both in group 1 → kept. 404684004 has them in DIFFERENT groups
	// (1 and 2) → excluded.
	got := evalECL(t, "<< 404684003 : { 363698007 = 74281007, 116676008 = 55641003 }", p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_GroupedRefinement_DifferentGroups_Fail(t *testing.T) {
	p := newFixture()
	// Same query as above but focused on 404684004 alone: the two attributes
	// exist but in different groups, so the grouped clause must filter it out.
	got := evalECL(t, "404684004 : { 363698007 = 74281007, 116676008 = 55641003 }", p)
	assert.Equal(t, 0, got.Len(),
		"404684004 has the two attrs in different groups; grouped clause must exclude it")
}

func TestEvaluate_GroupedRefinement_SingleAttr(t *testing.T) {
	p := newFixture()
	// A grouped clause with a single attribute is effectively the same as the
	// ungrouped equivalent.
	got := evalECL(t, "<< 404684003 : { 363698007 = 74281007 }", p)
	assert.ElementsMatch(t, []string{"22298006", "404684004"}, got.Slice())
}

// ------------------------------------------------------------------------
// Phase 3.5 — Dot notation ( X . Attr )
// ------------------------------------------------------------------------.

func TestEvaluate_DotExpression_Simple(t *testing.T) {
	p := newFixture()
	// 22298006 . 363698007 = the Finding site target of MI = {74281007}
	got := evalECL(t, "22298006 . 363698007", p)
	assert.ElementsMatch(t, []string{"74281007"}, got.Slice())
}

func TestEvaluate_DotExpression_HierarchicalSource(t *testing.T) {
	p := newFixture()
	// << 404684003 . 363698007 = union of finding sites across all
	// descendants-or-self: {74281007 (MI, 404684004), 113331007 (diabetes)}.
	got := evalECL(t, "<< 404684003 . 363698007", p)
	assert.ElementsMatch(t, []string{"74281007", "113331007"}, got.Slice())
}

func TestEvaluate_DotExpression_Chained(t *testing.T) {
	p := newFixture()
	// 22298006 . 363698007 . 363698007
	//   first dot  → {74281007}
	//   second dot → finding sites of 74281007 = {} (no relationships)
	got := evalECL(t, "22298006 . 363698007 . 363698007", p)
	assert.Equal(t, 0, got.Len())
}

func TestEvaluate_DotExpression_NoMatchingAttribute(t *testing.T) {
	p := newFixture()
	// Concept with no relationships (64572001 / disease).
	got := evalECL(t, "64572001 . 363698007", p)
	assert.Equal(t, 0, got.Len())
}

// ------------------------------------------------------------------------
// Reverse attribute
// ------------------------------------------------------------------------.

func TestEvaluate_Refinement_Reverse(t *testing.T) {
	p := newFixture()
	// Reverse: "concepts that are the target of a (Finding site) relationship
	// whose source is in << 404684003 — with myocardium as the starting point".
	// Query: "<< 74281007 : R 363698007 = << 404684003" — among concepts in
	// << 74281007 (just 74281007 here), keep those that are the target of a
	// 363698007 relationship from some concept in << 404684003.
	// 74281007 is the target of 363698007 from {22298006, 404684004} → kept.
	got := evalECL(t, "<< 74281007 : R 363698007 = << 404684003", p)
	assert.ElementsMatch(t, []string{"74281007"}, got.Slice())
}

// ------------------------------------------------------------------------
// Error handling / deferred features
// ------------------------------------------------------------------------.

// Concrete-value comparisons are implemented in Phase 5.2. When no concept in
// the focus set has a matching concrete value, the result is an empty set (no
// error). Fixture concept 363698007 is a concept-valued attribute (no concrete
// numeric values), so >= #5 legitimately matches nothing.
func TestEvaluate_Refinement_ConcreteOperator_NoMatch(t *testing.T) {
	p := newFixture()
	expr, err := Parse("<< 404684003 : 363698007 >= #5")
	require.NoError(t, err, "parse should succeed for numeric comparison")
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.Equal(t, 0, got.Len(), "no fixture concept has a numeric value for 363698007")
}
