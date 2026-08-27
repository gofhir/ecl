package ecl_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// nilProvider returns (nil, nil) from every method. That is the idiomatic Go
// shape for "nothing found" and the DataProvider godoc did not forbid it, so a
// third-party implementation could reasonably look like this. It used to panic
// inside Evaluate on `left.Intersect(right)`, taking down the calling service.
type nilProvider struct{}

func (nilProvider) Descendants(context.Context, []string, bool) (ecl.Set, error) { return nil, nil }
func (nilProvider) Ancestors(context.Context, []string, bool) (ecl.Set, error)   { return nil, nil }
func (nilProvider) Children(context.Context, []string, bool) (ecl.Set, error)    { return nil, nil }
func (nilProvider) Parents(context.Context, []string, bool) (ecl.Set, error)     { return nil, nil }
func (nilProvider) ConceptExists(context.Context, []string) (ecl.Set, error)     { return nil, nil }
func (nilProvider) AllConcepts(context.Context) (ecl.Set, error)                 { return nil, nil }

func (nilProvider) RelationshipTargets(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) RelationshipSources(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) ConcreteValues(context.Context, string, string) ([]ecl.ConcreteValue, error) {
	return nil, nil
}

func (nilProvider) PropertiesByGroup(context.Context, string) (map[int][]ecl.Relationship, error) {
	return nil, nil
}

func (nilProvider) MatchDescription(context.Context, ecl.DescriptionFilterOpts) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) FilterConcepts(context.Context, ecl.Set, ecl.ConceptFilterOpts) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) RefsetMembers(context.Context, []string) (ecl.Set, error) { return nil, nil }

func (nilProvider) RefsetsContainingMembers(context.Context, []string) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) HistoricalAssociations(context.Context, ecl.Set, string) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) ResolveIdentifier(context.Context, string, string) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) MatchDialect(context.Context, ecl.Set, ecl.DialectFilterOpts) (ecl.Set, error) {
	return nil, nil
}

func (nilProvider) RefsetMembersFiltered(context.Context, []string, ecl.MemberFilterOpts) (ecl.Set, error) {
	return nil, nil
}

// TestEvaluate_NilSetsFromProviderDoNotPanic pins the defensive normalization.
func TestEvaluate_NilSetsFromProviderDoNotPanic(t *testing.T) {
	for _, expr := range []string{
		"^ 900000000000509007 AND << 404684003",
		"<< 404684003",
		"* : 363698007 = *",
		"< 404684003 : R 363698007 = *",
		"404684003 MINUS 11687002",
		"!!> (<< 404684003)",
		"<< 404684003 {{ C active = true }}",
		"22298006 {{ +HISTORY }}",
	} {
		t.Run(expr, func(t *testing.T) {
			tree := mustParse(t, expr)
			set, err := ecl.Evaluate(context.Background(), tree, nilProvider{})
			require.NoError(t, err)
			require.NotNil(t, set, "Evaluate handed a nil Set to the caller")
			require.Zero(t, set.Len())
			require.Empty(t, set.Slice()) // must not panic
		})
	}
}

// TestEvaluate_RespectsCancelledContext covers cancellation: the evaluator never
// consulted ctx, so a canceled request kept issuing provider calls and returned
// a nil error.
func TestEvaluate_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ecl.Evaluate(ctx, mustParse(t, "<< 138875005 : 363698007 = *"), standardProvider(t))
	require.ErrorIs(t, err, context.Canceled)
}

// TestEvaluate_WildcardIncludesInactiveConcepts pins the active-axis rule: the
// wildcard is resolved before filters run, so AllConcepts must not filter by the
// active flag or `* {{ C active = false }}` can never return anything.
func TestEvaluate_WildcardIncludesInactiveConcepts(t *testing.T) {
	inactive := evalFixture(t, "* {{ C active = false }}")
	require.ElementsMatch(t, []string{"11111111", "22222222", "33333333"}, inactive.Slice())

	active := evalFixture(t, "* {{ C active = true }}")
	require.NotContains(t, active.Slice(), "11111111")
	require.Contains(t, active.Slice(), "22298006")
}

// TestUnimplementedDataProvider covers the embeddable base: an unimplemented
// method must report "not supported" rather than answer the empty set, which a
// caller would read as valid data.
func TestUnimplementedDataProvider(t *testing.T) {
	var p ecl.UnimplementedDataProvider

	_, err := p.Descendants(context.Background(), []string{"404684003"}, true)
	require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
	require.Contains(t, err.Error(), "Descendants")

	// Evaluating through it surfaces the same classifiable error.
	_, err = ecl.Evaluate(context.Background(), mustParse(t, "<< 404684003"), p)
	require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
}

// TestUnimplementedDataProvider_Embedding is the pattern the doc recommends: a
// partial provider implements what it can and inherits the rest.
func TestUnimplementedDataProvider_Embedding(t *testing.T) {
	p := partialProvider{}

	set, err := ecl.Evaluate(context.Background(), mustParse(t, "404684003"), p)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"404684003"}, set.Slice())

	// Anything it did not implement is reported, not silently empty.
	_, err = ecl.Evaluate(context.Background(), mustParse(t, "^ 900000000000497000"), p)
	require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
}

// partialProvider implements one method and embeds the rest.
type partialProvider struct {
	ecl.UnimplementedDataProvider
}

func (partialProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	return ecl.NewSetFromSlice(ids), nil
}
