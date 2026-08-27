package ecl_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
)

// All expected IDs in this file are computed against
// testdata/conformance/fixtures/standard.yaml, where:
//
//	22298006 has 363698007=74281007 + 116676008=55641003 (group 1)
//	                363698007=113331007                  (group 2)
//	73211009 has 363698007=113331007 + 1142139005=#5     (group 1)

// TestParse_ConjunctionAndDisjunctionDiffer is the root cause: the parser used
// to flatten both branches into one slice, so the two ASTs were identical and no
// evaluator could tell them apart.
func TestParse_ConjunctionAndDisjunctionDiffer(t *testing.T) {
	and := mustParse(t, "< 404684003 : 363698007 = 74281007 , 116676008 = 55641003")
	or := mustParse(t, "< 404684003 : 363698007 = 74281007 OR 116676008 = 55641003")
	require.False(t, reflect.DeepEqual(and, or),
		"AND and OR produce the same AST: the operator is lost while parsing")
}

// TestEvaluate_UngroupedDisjunction covers the headline bug: a disjunction of
// ungrouped attributes was intersected, so it returned the empty set.
func TestEvaluate_UngroupedDisjunction(t *testing.T) {
	left := evalFixture(t, "* : 363698007 = 74281007")
	require.ElementsMatch(t, []string{"22298006"}, left.Slice())

	right := evalFixture(t, "* : 363698007 = 113331007")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, right.Slice())

	or := evalFixture(t, "* : 363698007 = 74281007 OR 363698007 = 113331007")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, or.Slice(),
		"OR was evaluated as AND")

	// The conjunction of the same two clauses must still intersect.
	and := evalFixture(t, "* : 363698007 = 74281007 , 363698007 = 113331007")
	require.ElementsMatch(t, []string{"22298006"}, and.Slice())
}

// TestEvaluate_DisjunctionOfGroups covers the same defect one level up, where
// the grammar routes the operands through Refinement.Disjunction.
func TestEvaluate_DisjunctionOfGroups(t *testing.T) {
	or := evalFixture(t, "* : { 363698007 = 74281007 } OR { 1142139005 = #5 }")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, or.Slice())

	and := evalFixture(t, "* : { 363698007 = 74281007 } , { 1142139005 = #5 }")
	require.Empty(t, and.Slice(), "no concept satisfies both groups")
}

// TestEvaluate_MixedAttributeAndGroup is the anti-regression case for C1. The
// grammar cannot put a group inside an attribute set, so the comma promotes the
// group to Refinement.Conjunction: an early return on AttrSet would drop it and
// silently widen the result.
func TestEvaluate_MixedAttributeAndGroup(t *testing.T) {
	set := evalFixture(t, "* : 363698007 = 74281007 , { 116676008 = 55641003 }")
	require.ElementsMatch(t, []string{"22298006"}, set.Slice())

	// Same shape, but the group's clause matches nobody: the result must be
	// empty rather than falling back to the attribute alone.
	none := evalFixture(t, "* : 363698007 = 74281007 , { 1142139005 = #5 }")
	require.Empty(t, none.Slice(), "the group constraint was dropped")
}

// TestEvaluate_ParenthesisedDisjunctionKeepsScope is the anti-regression case
// for C3: flattening the parentheses let a later conjunct escape the union.
func TestEvaluate_ParenthesisedDisjunctionKeepsScope(t *testing.T) {
	// (A OR B) , C  where C only holds for 73211009 → only 73211009 qualifies.
	set := evalFixture(t, "* : ({ 363698007 = 74281007 } OR { 363698007 = 113331007 }) , 1142139005 = #5")
	require.ElementsMatch(t, []string{"73211009"}, set.Slice())
}

// TestEvaluate_NestedAttributeDisjunction covers a parenthesised attribute set,
// which recurses through collectSubAttributeSet rather than through the
// refinement level.
func TestEvaluate_NestedAttributeDisjunction(t *testing.T) {
	set := evalFixture(t, "* : (363698007 = 74281007 OR 1142139005 = #5)")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, set.Slice())
}

// TestEvaluate_RefinementRejectsLostScope guards the invariant that lets the
// disjunction be composed by union: a node must never carry both a conjunction
// and a disjunction. The parser cannot produce that any more, so the check is
// exercised by hand-building the AST — which is also how a consumer could hit
// it, since ecl/ast is public.
func TestEvaluate_RefinementRejectsLostScope(t *testing.T) {
	attr := func(typeID, target string) *ast.Attribute {
		return &ast.Attribute{
			Name:  &ast.ConceptRef{ID: typeID},
			Op:    "=",
			Value: &ast.ConceptRef{ID: target},
		}
	}
	expr := &ast.Refined{
		Focus: &ast.Any{},
		Refinement: &ast.Refinement{
			Conjunction: []*ast.Refinement{{Ungrouped: []*ast.Attribute{attr("363698007", "74281007")}}},
			Disjunction: []*ast.Refinement{{Ungrouped: []*ast.Attribute{attr("1142139005", "5")}}},
		},
	}

	_, err := ecl.Evaluate(context.Background(), expr, standardProvider(t))
	require.Error(t, err, "a refinement with both branches must be rejected, not silently mis-composed")
	require.Contains(t, err.Error(), "parenthesised scope")
}
