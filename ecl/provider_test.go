package ecl

import (
	"context"
	"testing"
)

// mockProvider is a zero-value stub used to verify DataProvider is satisfiable.
// Real evaluator tests (in later phases) will provide a functional mock.
type mockProvider struct{}

func (m *mockProvider) Descendants(_ context.Context, _ []string, _ bool) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) Ancestors(_ context.Context, _ []string, _ bool) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) Children(_ context.Context, _ []string, _ bool) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) Parents(_ context.Context, _ []string, _ bool) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) ConceptExists(_ context.Context, _ []string) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) AllConcepts(_ context.Context) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) RelationshipTargets(_ context.Context, _, _ Set) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) RelationshipSources(_ context.Context, _, _ Set) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) ConcreteValues(_ context.Context, _, _ string) ([]ConcreteValue, error) {
	return nil, nil
}

func (m *mockProvider) PropertiesByGroup(_ context.Context, _ string) (map[int][]Relationship, error) {
	return nil, nil
}

func (m *mockProvider) MatchDescription(_ context.Context, _ DescriptionFilterOpts) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) FilterConcepts(_ context.Context, _ Set, _ ConceptFilterOpts) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) RefsetMembers(_ context.Context, _ []string) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) HistoricalAssociations(_ context.Context, _ Set, _ string) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) ResolveIdentifier(_ context.Context, _ string, _ string) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) MatchDialect(_ context.Context, _ Set, _ DialectFilterOpts) (Set, error) {
	return NewSet(), nil
}

func (m *mockProvider) RefsetMembersFiltered(_ context.Context, _ []string, _ MemberFilterOpts) (Set, error) {
	return NewSet(), nil
}

// Compile-time check: mockProvider implements DataProvider.
var _ DataProvider = (*mockProvider)(nil)

// TestDataProviderCompiles verifies the DataProvider interface is satisfiable.
// Real behavioral tests come with the evaluator in later phases.
func TestDataProviderCompiles(_ *testing.T) {
	var p DataProvider = &mockProvider{}
	_ = p
}
