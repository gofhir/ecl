package scg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// mockProvider is a tiny ecl.DataProvider used for validator tests.
// Only ConceptExists is exercised by the SCG validator; the other methods
// panic if called.
type mockProvider struct {
	known map[string]bool
}

func newMockProvider(ids ...string) *mockProvider {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return &mockProvider{known: m}
}

func (m *mockProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if m.known[id] {
			out = append(out, id)
		}
	}
	return ecl.NewSetFromSlice(out), nil
}

// The remaining DataProvider methods are unused by Validate. They panic to
// catch accidental coupling.
func (m *mockProvider) Descendants(context.Context, []string, bool) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) Ancestors(context.Context, []string, bool) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) Children(context.Context, []string, bool) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) Parents(context.Context, []string, bool) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) AllConcepts(context.Context) (ecl.Set, error) { panic("not used") }
func (m *mockProvider) RelationshipTargets(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) RelationshipSources(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) ConcreteValues(context.Context, string, string) ([]ecl.ConcreteValue, error) {
	panic("not used")
}
func (m *mockProvider) PropertiesByGroup(context.Context, string) (map[int][]ecl.Relationship, error) {
	panic("not used")
}
func (m *mockProvider) MatchDescription(context.Context, ecl.DescriptionFilterOpts) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) FilterConcepts(context.Context, ecl.Set, ecl.ConceptFilterOpts) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) RefsetMembers(context.Context, []string) (ecl.Set, error) {
	panic("not used")
}
func (m *mockProvider) HistoricalAssociations(context.Context, ecl.Set, string) (ecl.Set, error) {
	panic("not used")
}

func mustParse(t *testing.T, in string) *Expression {
	t.Helper()
	expr, err := Parse(in)
	require.NoError(t, err)
	return expr
}

func TestValidate_SingleConcept_Valid(t *testing.T) {
	expr := mustParse(t, "404684003")
	provider := newMockProvider("404684003")
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Empty(t, res.Issues)
}

func TestValidate_UnknownConcept(t *testing.T) {
	expr := mustParse(t, "404684003")
	provider := newMockProvider() // empty
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	require.Len(t, res.Issues, 1)
	assert.Equal(t, "404684003", res.Issues[0].SCTID)
	assert.Equal(t, IssueKindUnknown, res.Issues[0].Kind)
	assert.Equal(t, "focus[0]", res.Issues[0].Path)
}

func TestValidate_Refinement_AllValid(t *testing.T) {
	expr := mustParse(t, "22298006:363698007=74281007")
	provider := newMockProvider("22298006", "363698007", "74281007")
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Empty(t, res.Issues)
}

func TestValidate_Refinement_UnknownAttr(t *testing.T) {
	expr := mustParse(t, "22298006:363698007=74281007")
	// 363698007 is missing from the provider.
	provider := newMockProvider("22298006", "74281007")
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	require.Len(t, res.Issues, 1)
	assert.Equal(t, "363698007", res.Issues[0].SCTID)
	assert.Equal(t, IssueKindUnknown, res.Issues[0].Kind)
	assert.Equal(t, "refinement[0].attr[0].name", res.Issues[0].Path)
}

func TestValidate_Nested(t *testing.T) {
	expr := mustParse(t, "373873005:246093002=(386053000:363698007=39057004)")
	provider := newMockProvider(
		"373873005", "246093002", "386053000", "363698007", "39057004",
	)
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Empty(t, res.Issues)
}

func TestValidate_InvalidSCTID_VerhoefFails(t *testing.T) {
	// 404684004 has a wrong check digit (404684003 is the valid form).
	// We can't get this past the parser indirectly via Parse() since the
	// parser only checks length, so we build the AST manually OR pass it
	// through Parse directly (Parse does NOT enforce Verhoeff).
	expr, err := Parse("404684004:363698007=74281007")
	require.NoError(t, err)

	provider := newMockProvider("404684004", "363698007", "74281007")
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.False(t, res.Valid)

	// 404684004 should produce an invalid_sctid issue.
	var found bool
	for _, iss := range res.Issues {
		if iss.SCTID == "404684004" && iss.Kind == IssueKindInvalidSCTID {
			found = true
			assert.Equal(t, "focus[0]", iss.Path)
		}
	}
	assert.True(t, found, "expected invalid_sctid issue for 404684004; got %+v", res.Issues)
}

func TestValidate_NestedUnknownIsReported(t *testing.T) {
	expr := mustParse(t, "373873005:246093002=(386053000:363698007=39057004)")
	// Drop 39057004 from the provider — it's a nested attribute value.
	provider := newMockProvider("373873005", "246093002", "386053000", "363698007")
	res, err := Validate(context.Background(), expr, provider)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	require.Len(t, res.Issues, 1)
	assert.Equal(t, "39057004", res.Issues[0].SCTID)
	assert.Contains(t, res.Issues[0].Path, "refinement[0].attr[0].value")
}

func TestValidate_NilExpression(t *testing.T) {
	_, err := Validate(context.Background(), nil, newMockProvider())
	require.Error(t, err)
}

func TestValidate_NilProvider(t *testing.T) {
	expr := mustParse(t, "404684003")
	_, err := Validate(context.Background(), expr, nil)
	require.Error(t, err)
}
