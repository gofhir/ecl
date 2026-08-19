package ecl_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
)

// Regression tests for defects found reviewing this branch. Expected IDs are
// computed against ecl/providertest/testdata/fixtures/standard.yaml.

// TestEvaluate_ParenthesisedRefinementThenDisjunction covers a valid shape that
// the scope guard rejected.
//
// Keeping a parenthesised sub-refinement in Refinement.Conjunction conflated
// "the first sub-refinement was parenthesised" with "there is a conjunction set",
// so `(<refinement>) OR <clause>` looked like a node holding both a conjunction
// and a disjunction and errored out. It has its own field now.
func TestEvaluate_ParenthesisedRefinementThenDisjunction(t *testing.T) {
	set := evalFixture(t, "* : ({ 363698007 = 74281007 }) OR 1142139005 = #5")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, set.Slice())

	// The scope still has to survive: here the trailing conjunct must apply to
	// the whole disjunction.
	scoped := evalFixture(t, "* : ({ 363698007 = 74281007 } OR { 363698007 = 113331007 }) , 1142139005 = #5")
	require.ElementsMatch(t, []string{"73211009"}, scoped.Slice())
}

// TestEvaluate_FilterOperandMatchingNothing covers the difference between "no
// operand" and "an operand that names nothing".
//
// An empty field in the Opts means "do not filter on this dimension", so
// returning no IDs for an operand that legitimately matched nothing turned the
// clause into a no-op and the filter matched EVERYTHING.
func TestEvaluate_FilterOperandMatchingNothing(t *testing.T) {
	// 900000000000073002 has no descendants, so the operand names no concept.
	none := evalFixture(t, "<< 404684003 {{ C definitionStatusId = (< 900000000000073002) }}")
	require.Empty(t, none.Slice(), "a clause naming no concept matched everything")

	// Negated, the same clause removes nothing.
	all := evalFixture(t, "<< 404684003 {{ C definitionStatusId != (< 900000000000073002) }}")
	base := evalFixture(t, "<< 404684003")
	require.ElementsMatch(t, base.Slice(), all.Slice())

	// The positive form still works when the operand does name something.
	some := evalFixture(t, "<< 404684003 {{ C definitionStatusId = 900000000000073002 }}")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, some.Slice())
}

// TestEvaluate_NilGroupInHandBuiltAST covers a nil group in the AST. Since
// ecl/ast is public a consumer can build one, and the guard against it had been
// dropped.
func TestEvaluate_NilGroupInHandBuiltAST(t *testing.T) {
	expr := &ast.Refined{
		Focus:      &ast.Any{},
		Refinement: &ast.Refinement{Groups: []*ast.AttributeGroup{nil}},
	}
	set, err := ecl.Evaluate(context.Background(), expr, standardProvider(t))
	require.NoError(t, err)
	require.NotNil(t, set)
}

// TestEvaluate_NilSetFromRelationshipSources covers the reverse-clause-in-group
// path, which the nil hardening missed: that Set never travels through Evaluate,
// so it was dereferenced directly.
func TestEvaluate_NilSetFromRelationshipSources(t *testing.T) {
	set, err := ecl.Evaluate(context.Background(),
		mustParse(t, "< 404684003 : { R 363698007 = * }"), nilProvider{})
	require.NoError(t, err)
	require.NotNil(t, set)
	require.Zero(t, set.Len())
}

// TestErrProvider covers the sentinel being reachable. It was exported and
// documented for errors.Is classification but never actually wrapped, so the
// documented check could not fire.
func TestErrProvider(t *testing.T) {
	_, err := ecl.Evaluate(context.Background(),
		mustParse(t, "< 404684003 : 363698007 = *"), failingProvider{})
	require.Error(t, err)
	require.ErrorIs(t, err, ecl.ErrProvider)

	// A malformed expression is NOT a provider error.
	_, err = ecl.Parse("11687002 GARBAGE")
	require.Error(t, err)
	require.NotErrorIs(t, err, ecl.ErrProvider)
}

// TestEvaluate_CancellationStopsPerConceptLoops covers cancellation inside the
// per-concept loops, which is where the thousands of provider calls happen. The
// check used to sit only at node entry, so a canceled request ran the whole loop.
func TestEvaluate_CancellationStopsPerConceptLoops(t *testing.T) {
	counting := &countingProvider{DataProvider: standardProvider(t)}
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel as soon as the loop asks for its first concept.
	counting.onPropertiesByGroup = func() { cancel() }

	_, err := ecl.Evaluate(ctx, mustParse(t, "< 138875005 : 363698007 = *"), counting)
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, counting.propertiesByGroup, 2,
		"the loop kept calling the provider after cancellation")
}

// failingProvider fails every hierarchy and attribute lookup, standing in for an
// unhealthy backend.
type failingProvider struct{ ecl.UnimplementedDataProvider }

var errBackendDown = errors.New("backend down")

func (failingProvider) Descendants(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSetFromSlice([]string{"22298006"}), nil
}

func (failingProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	return ecl.NewSetFromSlice(ids), nil
}

func (failingProvider) AllConcepts(context.Context) (ecl.Set, error) {
	return ecl.NewSetFromSlice([]string{"22298006"}), nil
}

func (failingProvider) PropertiesByGroup(context.Context, string) (map[int][]ecl.Relationship, error) {
	return nil, errBackendDown
}

// countingProvider counts PropertiesByGroup calls and can run a hook on each one.
type countingProvider struct {
	ecl.DataProvider
	propertiesByGroup   int
	onPropertiesByGroup func()
}

func (c *countingProvider) PropertiesByGroup(ctx context.Context, id string) (map[int][]ecl.Relationship, error) {
	c.propertiesByGroup++
	if c.onPropertiesByGroup != nil {
		c.onPropertiesByGroup()
	}
	return c.DataProvider.PropertiesByGroup(ctx, id)
}
