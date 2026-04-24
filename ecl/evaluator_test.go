package ecl

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProvider is a map-backed DataProvider used by evaluator tests.
// It implements the full DataProvider interface; methods that are not
// relevant to Phase 3.2 return empty results.
type testProvider struct {
	descendants map[string][]string // key: conceptID, value: descendant IDs (NOT including self)
	ancestors   map[string][]string
	children    map[string][]string // direct
	parents     map[string][]string // direct
	exists      map[string]bool
	all         []string
	refsets     map[string][]string // key: refsetID, value: member concept IDs
}

// Hierarchy --------------------------------------------------------------

func (p *testProvider) Descendants(_ context.Context, conceptIDs []string, includeSelf bool) (Set, error) {
	out := NewSet().(*mapSet)
	for _, id := range conceptIDs {
		if includeSelf {
			out.m[id] = struct{}{}
		}
		for _, d := range p.descendants[id] {
			out.m[d] = struct{}{}
		}
	}
	return out, nil
}

func (p *testProvider) Ancestors(_ context.Context, conceptIDs []string, includeSelf bool) (Set, error) {
	out := NewSet().(*mapSet)
	for _, id := range conceptIDs {
		if includeSelf {
			out.m[id] = struct{}{}
		}
		for _, a := range p.ancestors[id] {
			out.m[a] = struct{}{}
		}
	}
	return out, nil
}

func (p *testProvider) Children(_ context.Context, conceptIDs []string, includeSelf bool) (Set, error) {
	out := NewSet().(*mapSet)
	for _, id := range conceptIDs {
		if includeSelf {
			out.m[id] = struct{}{}
		}
		for _, c := range p.children[id] {
			out.m[c] = struct{}{}
		}
	}
	return out, nil
}

func (p *testProvider) Parents(_ context.Context, conceptIDs []string, includeSelf bool) (Set, error) {
	out := NewSet().(*mapSet)
	for _, id := range conceptIDs {
		if includeSelf {
			out.m[id] = struct{}{}
		}
		for _, par := range p.parents[id] {
			out.m[par] = struct{}{}
		}
	}
	return out, nil
}

// Concepts ---------------------------------------------------------------

func (p *testProvider) ConceptExists(_ context.Context, conceptIDs []string) (Set, error) {
	out := NewSet().(*mapSet)
	for _, id := range conceptIDs {
		if p.exists[id] {
			out.m[id] = struct{}{}
		}
	}
	return out, nil
}

func (p *testProvider) AllConcepts(_ context.Context) (Set, error) {
	return NewSetFromSlice(p.all), nil
}

// Refsets ----------------------------------------------------------------

func (p *testProvider) RefsetMembers(_ context.Context, refsetIDs []string) (Set, error) {
	out := NewSet().(*mapSet)
	for _, rid := range refsetIDs {
		for _, m := range p.refsets[rid] {
			out.m[m] = struct{}{}
		}
	}
	return out, nil
}

// Stubs (not used in Phase 3.2) ------------------------------------------

func (p *testProvider) RelationshipTargets(_ context.Context, _ Set, _ Set) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) RelationshipSources(_ context.Context, _ Set, _ Set) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) ConcreteValues(_ context.Context, _ string, _ string) ([]ConcreteValue, error) {
	return nil, nil
}
func (p *testProvider) PropertiesByGroup(_ context.Context, _ string) (map[int][]Relationship, error) {
	return nil, nil
}
func (p *testProvider) MatchDescription(_ context.Context, _ DescriptionFilterOpts) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) FilterConcepts(_ context.Context, _ Set, _ ConceptFilterOpts) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) HistoricalAssociations(_ context.Context, _ Set, _ string) (Set, error) {
	return NewSet(), nil
}

var _ DataProvider = (*testProvider)(nil)

// ------------------------------------------------------------------------
// Fixture
//
//          138875005 (root)
//              │
//         404684003 (clinical finding)
//         /         \
//    22298006      64572001 (disease)
//   (MI)         /        \
//             73211009    404684004 (other)
//             (diabetes)
// ------------------------------------------------------------------------

func newFixture() *testProvider {
	return &testProvider{
		descendants: map[string][]string{
			"138875005": {"404684003", "22298006", "64572001", "73211009", "404684004"},
			"404684003": {"22298006", "64572001", "73211009", "404684004"},
			"64572001":  {"73211009", "404684004"},
		},
		ancestors: map[string][]string{
			"22298006":  {"404684003", "138875005"},
			"64572001":  {"404684003", "138875005"},
			"73211009":  {"64572001", "404684003", "138875005"},
			"404684004": {"64572001", "404684003", "138875005"},
			"404684003": {"138875005"},
		},
		children: map[string][]string{
			"138875005": {"404684003"},
			"404684003": {"22298006", "64572001"},
			"64572001":  {"73211009", "404684004"},
		},
		parents: map[string][]string{
			"404684003": {"138875005"},
			"22298006":  {"404684003"},
			"64572001":  {"404684003"},
			"73211009":  {"64572001"},
			"404684004": {"64572001"},
		},
		exists: map[string]bool{
			"138875005":          true,
			"404684003":          true,
			"22298006":           true,
			"64572001":           true,
			"73211009":           true,
			"404684004":          true,
			"900000000000497000": true, // refset concept itself exists
		},
		all: []string{"138875005", "404684003", "22298006", "64572001", "73211009", "404684004"},
		refsets: map[string][]string{
			"900000000000497000": {"22298006", "64572001", "73211009"},
		},
	}
}

// Helper: evaluate an ECL string and assert no error.
func evalECL(t *testing.T, ecl string, p DataProvider) Set {
	t.Helper()
	expr, err := Parse(ecl)
	require.NoError(t, err, "parse failed for %q", ecl)
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err, "evaluate failed for %q", ecl)
	return got
}

// ------------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------------

func TestEvaluate_ConceptRef(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "404684003", p)
	assert.Equal(t, []string{"404684003"}, got.Slice())
}

func TestEvaluate_ConceptRef_NotFound(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "999999999", p)
	assert.Equal(t, 0, got.Len())
}

func TestEvaluate_DescendantOf(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "< 404684003", p)
	assert.ElementsMatch(t,
		[]string{"22298006", "64572001", "73211009", "404684004"},
		got.Slice())
}

func TestEvaluate_DescendantOrSelfOf(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "<< 404684003", p)
	assert.ElementsMatch(t,
		[]string{"404684003", "22298006", "64572001", "73211009", "404684004"},
		got.Slice())
}

func TestEvaluate_AncestorOf(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "> 22298006", p)
	assert.ElementsMatch(t,
		[]string{"404684003", "138875005"},
		got.Slice())
}

func TestEvaluate_AncestorOrSelfOf(t *testing.T) {
	p := newFixture()
	got := evalECL(t, ">> 22298006", p)
	assert.ElementsMatch(t,
		[]string{"22298006", "404684003", "138875005"},
		got.Slice())
}

func TestEvaluate_ChildOf(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "<! 404684003", p)
	assert.ElementsMatch(t,
		[]string{"22298006", "64572001"},
		got.Slice())
}

func TestEvaluate_ParentOf(t *testing.T) {
	p := newFixture()
	got := evalECL(t, ">! 73211009", p)
	assert.ElementsMatch(t,
		[]string{"64572001"},
		got.Slice())
}

func TestEvaluate_Wildcard(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "*", p)
	assert.Equal(t, 6, got.Len())
}

func TestEvaluate_And(t *testing.T) {
	p := newFixture()
	// << 404684003 = {404684003, 22298006, 64572001, 73211009, 404684004}
	// << 64572001  = {64572001, 73211009, 404684004}
	// intersection = {64572001, 73211009, 404684004}
	got := evalECL(t, "<< 404684003 AND << 64572001", p)
	assert.ElementsMatch(t,
		[]string{"64572001", "73211009", "404684004"},
		got.Slice())
}

func TestEvaluate_Or(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "22298006 OR 73211009", p)
	assert.ElementsMatch(t,
		[]string{"22298006", "73211009"},
		got.Slice())
}

func TestEvaluate_Minus(t *testing.T) {
	p := newFixture()
	// << 404684003 = {404684003, 22298006, 64572001, 73211009, 404684004}
	// << 64572001  = {64572001, 73211009, 404684004}
	// difference   = {404684003, 22298006}
	got := evalECL(t, "<< 404684003 MINUS << 64572001", p)
	assert.ElementsMatch(t,
		[]string{"404684003", "22298006"},
		got.Slice())
}

func TestEvaluate_MemberOf_Simple(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "^ 900000000000497000", p)
	assert.ElementsMatch(t,
		[]string{"22298006", "64572001", "73211009"},
		got.Slice())
}

func TestEvaluate_Nested(t *testing.T) {
	p := newFixture()
	got := evalECL(t, "(<< 64572001)", p)
	assert.ElementsMatch(t,
		[]string{"64572001", "73211009", "404684004"},
		got.Slice())
}

func TestEvaluate_NotImplemented_Top(t *testing.T) {
	p := newFixture()
	expr, err := Parse("!!> 404684003")
	require.NoError(t, err, "parse of top operator should succeed")
	_, err = Evaluate(context.Background(), expr, p)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "not yet implemented"),
		"error should say 'not yet implemented', got: %v", err)
}
