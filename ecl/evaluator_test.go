package ecl

import (
	"context"
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

	// Relationships per concept: key = source conceptID, value = list of
	// outbound attribute relationships. Used by RelationshipTargets,
	// RelationshipSources, and PropertiesByGroup.
	relationships map[string][]Relationship

	// Historical associations: key = inactive conceptID, value = replacement
	// concept IDs (e.g. SAME-AS / REPLACED-BY targets).
	historical map[string][]string

	// Concrete values: key = sourceID, inner key = typeID, value = list of
	// concrete attribute values.
	concreteValues map[string]map[string][]ConcreteValue
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

// Relationships (Phase 3.3 – 3.5) ---------------------------------------

func (p *testProvider) RelationshipTargets(_ context.Context, sourceIDs Set, typeIDs Set) (Set, error) {
	out := NewSet().(*mapSet)
	if sourceIDs == nil || typeIDs == nil {
		return out, nil
	}
	sourceIDs.Iter(func(src string) bool {
		for _, r := range p.relationships[src] {
			if typeIDs.Contains(r.TypeID) {
				out.m[r.TargetID] = struct{}{}
			}
		}
		return true
	})
	return out, nil
}

func (p *testProvider) RelationshipSources(_ context.Context, targetIDs Set, typeIDs Set) (Set, error) {
	out := NewSet().(*mapSet)
	if targetIDs == nil || typeIDs == nil {
		return out, nil
	}
	for src, rels := range p.relationships {
		for _, r := range rels {
			if !typeIDs.Contains(r.TypeID) {
				continue
			}
			if targetIDs.Contains(r.TargetID) {
				out.m[src] = struct{}{}
				break
			}
		}
	}
	return out, nil
}

func (p *testProvider) PropertiesByGroup(_ context.Context, conceptID string) (map[int][]Relationship, error) {
	rels, ok := p.relationships[conceptID]
	if !ok {
		return nil, nil
	}
	groups := make(map[int][]Relationship)
	for _, r := range rels {
		groups[r.GroupNum] = append(groups[r.GroupNum], r)
	}
	return groups, nil
}

// Concrete values / history (Phase 5) -----------------------------------

func (p *testProvider) ConcreteValues(_ context.Context, sourceID string, typeID string) ([]ConcreteValue, error) {
	if p.concreteValues == nil {
		return nil, nil
	}
	if byType, ok := p.concreteValues[sourceID]; ok {
		return byType[typeID], nil
	}
	return nil, nil
}

func (p *testProvider) HistoricalAssociations(_ context.Context, conceptIDs Set, _ string) (Set, error) {
	out := NewSet().(*mapSet)
	if conceptIDs == nil || p.historical == nil {
		return out, nil
	}
	conceptIDs.Iter(func(id string) bool {
		for _, h := range p.historical[id] {
			out.m[h] = struct{}{}
		}
		return true
	})
	return out, nil
}

// Stubs (not exercised by Phase 3 – 5 tests) -----------------------------

func (p *testProvider) MatchDescription(_ context.Context, _ DescriptionFilterOpts) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) FilterConcepts(_ context.Context, _ Set, _ ConceptFilterOpts) (Set, error) {
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
			// attribute type and target concepts used in relationship tests
			"363698007": true, // Finding site (attribute type)
			"116676008": true, // Associated morphology (attribute type)
			"74281007":   true, // Myocardium (target)
			"55641003":   true, // Infarct (target)
			"113331007":  true, // Endocrine system (target)
			"1142139005": true, // Count attribute (for concrete-value tests)
		},
		all: []string{"138875005", "404684003", "22298006", "64572001", "73211009", "404684004"},
		refsets: map[string][]string{
			"900000000000497000": {"22298006", "64572001", "73211009"},
		},
		// Attribute relationships for refinement / dot tests.
		//
		//   22298006  (MI)       group 1: 363698007 = 74281007
		//                        group 1: 116676008 = 55641003
		//   73211009  (diabetes) group 1: 363698007 = 113331007
		//   404684004 (other)    group 1: 363698007 = 74281007
		//                        group 2: 116676008 = 55641003
		//                        (two attrs but in DIFFERENT groups)
		relationships: map[string][]Relationship{
			"22298006": {
				{TypeID: "363698007", TargetID: "74281007", GroupNum: 1},
				{TypeID: "116676008", TargetID: "55641003", GroupNum: 1},
			},
			"73211009": {
				{TypeID: "363698007", TargetID: "113331007", GroupNum: 1},
			},
			"404684004": {
				{TypeID: "363698007", TargetID: "74281007", GroupNum: 1},
				{TypeID: "116676008", TargetID: "55641003", GroupNum: 2},
			},
		},
		// Historical associations: 404684004 is (pretend) inactive and has
		// been replaced by 22298006 (MI). See Phase 5.1 tests.
		historical: map[string][]string{
			"404684004": {"22298006"},
		},
		// Concrete values: 22298006 (MI) has a pretend "Count" attribute
		// (1142139005) with integer value 2. See Phase 5.2 tests.
		concreteValues: map[string]map[string][]ConcreteValue{
			"22298006": {
				"1142139005": {{Kind: "integer", Value: "2"}},
			},
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

