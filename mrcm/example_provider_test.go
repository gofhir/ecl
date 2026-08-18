package mrcm_test

import (
	"context"

	"github.com/gofhir/ecl/ecl"
)

// exampleProvider is a minimal DataProvider for the examples: just enough
// hierarchy for the rules' ECL to evaluate.
//
// Note it returns non-nil Sets everywhere, as the DataProvider contract requires.
type exampleDataProvider struct{}

func exampleProvider() ecl.DataProvider { return exampleDataProvider{} }

// descendants of the two concepts the example rules refer to.
var exampleDescendants = map[string][]string{
	"404684003": {"22298006", "73211009"},
	"442083009": {"74281007", "425391005"},
}

func (exampleDataProvider) Descendants(_ context.Context, ids []string, includeSelf bool) (ecl.Set, error) {
	out := []string{}
	for _, id := range ids {
		if includeSelf {
			out = append(out, id)
		}
		out = append(out, exampleDescendants[id]...)
	}
	return ecl.NewSetFromSlice(out), nil
}

func (exampleDataProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	return ecl.NewSetFromSlice(ids), nil
}

func (exampleDataProvider) Ancestors(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}
func (exampleDataProvider) Children(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}
func (exampleDataProvider) Parents(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}
func (exampleDataProvider) AllConcepts(context.Context) (ecl.Set, error) { return ecl.NewSet(), nil }
func (exampleDataProvider) RelationshipTargets(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) RelationshipSources(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) ConcreteValues(context.Context, string, string) ([]ecl.ConcreteValue, error) {
	return nil, nil
}

func (exampleDataProvider) PropertiesByGroup(context.Context, string) (map[int][]ecl.Relationship, error) {
	return nil, nil
}

func (exampleDataProvider) MatchDescription(context.Context, ecl.DescriptionFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) FilterConcepts(_ context.Context, concepts ecl.Set, _ ecl.ConceptFilterOpts) (ecl.Set, error) {
	return concepts, nil
}

func (exampleDataProvider) RefsetMembers(context.Context, []string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) RefsetsContainingMembers(context.Context, []string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) HistoricalAssociations(context.Context, ecl.Set, string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) ResolveIdentifier(context.Context, string, string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) MatchDialect(context.Context, ecl.Set, ecl.DialectFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (exampleDataProvider) RefsetMembersFiltered(context.Context, []string, ecl.MemberFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}
