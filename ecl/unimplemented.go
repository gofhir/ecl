package ecl

import (
	"context"
	"fmt"
)

// UnimplementedDataProvider returns ErrUnsupportedFeature from every method.
//
// Embed it in your own provider to satisfy the interface while implementing only
// the parts you need:
//
//	type myProvider struct {
//	    ecl.UnimplementedDataProvider
//	    db *sql.DB
//	}
//
//	func (p *myProvider) Descendants(ctx context.Context, ids []string, self bool) (ecl.Set, error) {
//	    // …your query
//	}
//
// Two reasons to prefer this over declaring the 18 methods by hand:
//
// First, adding a method to DataProvider stops being a breaking change for
// anything that embeds this type — and methods WILL be added: several fixes
// (batching PropertiesByGroup and ConcreteValues, row-level negation for
// description filters, cardinality on reverse clauses) need signatures this
// interface does not have yet. This exists so that day is a minor release for
// embedders rather than a compile error.
//
// Second, an unimplemented method returns a classifiable error instead of the
// empty set, so a partially implemented provider reports "not supported" rather
// than silently answering "no matches" — which reads as valid data.
//
// The zero value is ready to use.
type UnimplementedDataProvider struct{}

func (UnimplementedDataProvider) Descendants(context.Context, []string, bool) (Set, error) {
	return nil, unimplemented("Descendants")
}

func (UnimplementedDataProvider) Ancestors(context.Context, []string, bool) (Set, error) {
	return nil, unimplemented("Ancestors")
}

func (UnimplementedDataProvider) Children(context.Context, []string, bool) (Set, error) {
	return nil, unimplemented("Children")
}

func (UnimplementedDataProvider) Parents(context.Context, []string, bool) (Set, error) {
	return nil, unimplemented("Parents")
}

func (UnimplementedDataProvider) ConceptExists(context.Context, []string) (Set, error) {
	return nil, unimplemented("ConceptExists")
}

func (UnimplementedDataProvider) AllConcepts(context.Context) (Set, error) {
	return nil, unimplemented("AllConcepts")
}

func (UnimplementedDataProvider) RelationshipTargets(context.Context, Set, Set) (Set, error) {
	return nil, unimplemented("RelationshipTargets")
}

func (UnimplementedDataProvider) RelationshipSources(context.Context, Set, Set) (Set, error) {
	return nil, unimplemented("RelationshipSources")
}

func (UnimplementedDataProvider) ConcreteValues(context.Context, string, string) ([]ConcreteValue, error) {
	return nil, unimplemented("ConcreteValues")
}

func (UnimplementedDataProvider) PropertiesByGroup(context.Context, string) (map[int][]Relationship, error) {
	return nil, unimplemented("PropertiesByGroup")
}

func (UnimplementedDataProvider) MatchDescription(context.Context, DescriptionFilterOpts) (Set, error) {
	return nil, unimplemented("MatchDescription")
}

func (UnimplementedDataProvider) FilterConcepts(context.Context, Set, ConceptFilterOpts) (Set, error) {
	return nil, unimplemented("FilterConcepts")
}

func (UnimplementedDataProvider) RefsetMembers(context.Context, []string) (Set, error) {
	return nil, unimplemented("RefsetMembers")
}

func (UnimplementedDataProvider) RefsetsContainingMembers(context.Context, []string) (Set, error) {
	return nil, unimplemented("RefsetsContainingMembers")
}

func (UnimplementedDataProvider) HistoricalAssociations(context.Context, Set, string) (Set, error) {
	return nil, unimplemented("HistoricalAssociations")
}

func (UnimplementedDataProvider) ResolveIdentifier(context.Context, string, string) (Set, error) {
	return nil, unimplemented("ResolveIdentifier")
}

func (UnimplementedDataProvider) MatchDialect(context.Context, Set, DialectFilterOpts) (Set, error) {
	return nil, unimplemented("MatchDialect")
}

func (UnimplementedDataProvider) RefsetMembersFiltered(context.Context, []string, MemberFilterOpts) (Set, error) {
	return nil, unimplemented("RefsetMembersFiltered")
}

func unimplemented(method string) error {
	return fmt.Errorf("%w: DataProvider.%s is not implemented by this provider", ErrUnsupportedFeature, method)
}

// Compile-time check that the zero value satisfies the interface. If a method is
// added to DataProvider and not to this type, the build fails here -- which is
// the point.
var _ DataProvider = UnimplementedDataProvider{}
