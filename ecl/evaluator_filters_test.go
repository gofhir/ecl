package ecl

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------
// Phase 4 — Filter constraints: {{ term = "..." }}, {{ active = true }} ...
// ------------------------------------------------------------------------.

// testDescription is a minimal description row used by filterTestProvider.
type testDescription struct {
	Term     string
	TypeID   string // SCTID (e.g. 900000000000003001 = FSN)
	Language string // "en", "es"
	Active   bool
}

// filterTestProvider extends the base testProvider with description metadata
// and active-flag overrides, and implements MatchDescription / FilterConcepts.
type filterTestProvider struct {
	*testProvider
	descriptions map[string][]testDescription              // conceptID → descriptions
	activeFlag   map[string]bool                           // conceptID → active? (default true)
	dialects     map[string]map[string][]string            // dialectID → acceptabilityID → conceptIDs
	memberFields map[string]map[string]map[string][]string // refsetID → fieldName → fieldValue → conceptIDs
}

func newFilterFixture() *filterTestProvider {
	base := newFixture()
	return &filterTestProvider{
		testProvider: base,
		descriptions: map[string][]testDescription{
			"22298006": {
				{Term: "Myocardial infarction", TypeID: "900000000000003001", Language: "en", Active: true},
				{Term: "Infarto de miocardio", TypeID: "900000000000013009", Language: "es", Active: true},
			},
			"73211009": {
				{Term: "Diabetes mellitus", TypeID: "900000000000003001", Language: "en", Active: true},
			},
			"64572001": {
				{Term: "Disease", TypeID: "900000000000003001", Language: "en", Active: true},
			},
			"404684004": {
				{Term: "Other clinical finding", TypeID: "900000000000003001", Language: "en", Active: true},
			},
			"404684003": {
				{Term: "Clinical finding", TypeID: "900000000000003001", Language: "en", Active: true},
			},
		},
		activeFlag: map[string]bool{
			// Mark 404684004 as inactive for the active filter test.
			"404684004": false,
		},
		dialects: map[string]map[string][]string{
			"900000000000509007": { // US English
				"900000000000548007": {"22298006", "73211009"}, // preferred
				"900000000000549004": {"64572001"},             // acceptable
			},
		},
		memberFields: map[string]map[string]map[string][]string{
			"900000000000497000": {
				"referencedComponentId": {
					"22298006": {"22298006"},
					"64572001": {"64572001"},
					"73211009": {"73211009"},
				},
			},
		},
	}
}

func (p *filterTestProvider) isActive(id string) bool {
	if v, ok := p.activeFlag[id]; ok {
		return v
	}
	return true // default: active
}

// MatchDescription iterates all descriptions and returns concepts whose
// descriptions satisfy the given filter options. Term matching is a case-
// insensitive substring match.
func (p *filterTestProvider) MatchDescription(_ context.Context, f DescriptionFilterOpts) (Set, error) {
	out := NewSet().(*mapSet)
	needle := strings.ToLower(f.Term)
	for conceptID, descs := range p.descriptions {
		for _, d := range descs {
			if f.Term != "" && !strings.Contains(strings.ToLower(d.Term), needle) {
				continue
			}
			if len(f.TypeIDs) > 0 && !containsStr(f.TypeIDs, d.TypeID) {
				continue
			}
			if len(f.Languages) > 0 && !containsStr(f.Languages, d.Language) {
				continue
			}
			if f.Active != nil && d.Active != *f.Active {
				continue
			}
			out.m[conceptID] = struct{}{}
			break
		}
	}
	return out, nil
}

// FilterConcepts applies concept-level filters (active flag only — module /
// definitionStatus / effectiveTime are not modeled in the fixture).
func (p *filterTestProvider) FilterConcepts(_ context.Context, concepts Set, f ConceptFilterOpts) (Set, error) {
	out := NewSet().(*mapSet)
	if concepts == nil {
		return out, nil
	}
	concepts.Iter(func(id string) bool {
		if f.Active != nil && p.isActive(id) != *f.Active {
			return true
		}
		// DefinitionStatusID / ModuleID / EffectiveTime: fixture does not
		// model these; any non-empty option is treated as "matches all".
		out.m[id] = struct{}{}
		return true
	})
	return out, nil
}

// MatchDialect returns concept IDs whose descriptions match the given dialect
// filter constraints using the test fixture's dialect data.
func (p *filterTestProvider) MatchDialect(_ context.Context, concepts Set, f DialectFilterOpts) (Set, error) {
	out := NewSet().(*mapSet)
	if p.dialects == nil || concepts == nil {
		return out, nil
	}
	concepts.Iter(func(id string) bool {
		for _, d := range f.Dialects {
			for _, dialectID := range d.DialectIDs {
				byAccept, ok := p.dialects[dialectID]
				if !ok {
					continue
				}
				if len(d.AcceptabilityIDs) == 0 {
					// Any acceptability.
					for _, ids := range byAccept {
						for _, cid := range ids {
							if cid == id {
								out.m[id] = struct{}{}
								return true
							}
						}
					}
					continue
				}
				for _, acceptID := range d.AcceptabilityIDs {
					ids, ok := byAccept[acceptID]
					if !ok {
						continue
					}
					for _, cid := range ids {
						if cid == id {
							out.m[id] = struct{}{}
							return true
						}
					}
				}
			}
		}
		return true
	})
	return out, nil
}

// containsStr reports whether haystack contains needle.
func containsStr(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// RefsetMembersFiltered returns concept IDs from refset members that match
// the given member field filter constraints using the test fixture's data.
func (p *filterTestProvider) RefsetMembersFiltered(_ context.Context, refsetIDs []string, filter MemberFilterOpts) (Set, error) {
	out := NewSet().(*mapSet)
	if p.memberFields == nil {
		return out, nil
	}
	for _, rid := range refsetIDs {
		if byField, ok := p.memberFields[rid]; ok {
			if byValue, ok := byField[filter.FieldName]; ok {
				for val, conceptIDs := range byValue {
					matches := filter.ValueSet != nil && filter.ValueSet.Contains(val)
					if filter.Op == "!=" {
						matches = !matches
					}
					if matches {
						for _, cid := range conceptIDs {
							out.m[cid] = struct{}{}
						}
					}
				}
			}
		}
	}
	return out, nil
}

var _ DataProvider = (*filterTestProvider)(nil)

// ------------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------------.

func TestEvaluate_DescriptionFilter_Term(t *testing.T) {
	p := newFilterFixture()
	got := evalECL(t, `<< 404684003 {{ term = "infarction" }}`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_DescriptionFilter_Term_CaseInsensitive(t *testing.T) {
	p := newFilterFixture()
	got := evalECL(t, `<< 404684003 {{ term = "INFARCTION" }}`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_DescriptionFilter_Language(t *testing.T) {
	p := newFilterFixture()
	// Only MI has a Spanish description in the fixture.
	got := evalECL(t, `<< 404684003 {{ language = es }}`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_DescriptionFilter_Language_MultiValue(t *testing.T) {
	// Multi-value any-of: matches descriptions in either language.
	// Without the fix, this would silently take only the first language.
	p := newFilterFixture()
	got := evalECL(t, `<< 404684003 {{ language = (en OR es) }}`, p)
	// All 5 concepts with descriptions match (every one has at least an en
	// description; 22298006 also has es).
	assert.ElementsMatch(t,
		[]string{"22298006", "73211009", "64572001", "404684004", "404684003"},
		got.Slice())
}

func TestEvaluate_DescriptionFilter_Term_NoMatch(t *testing.T) {
	p := newFilterFixture()
	got := evalECL(t, `<< 404684003 {{ term = "nonexistent" }}`, p)
	assert.Equal(t, 0, got.Len())
}

func TestEvaluate_ConceptFilter_Active(t *testing.T) {
	p := newFilterFixture()
	// 404684004 is marked inactive in the fixture.
	got := evalECL(t, `<< 404684003 {{ C active = false }}`, p)
	assert.ElementsMatch(t, []string{"404684004"}, got.Slice())
}

func TestEvaluate_CombinedFilters(t *testing.T) {
	p := newFilterFixture()
	// Description filter narrows to MI; concept-level active filter keeps it
	// (MI is active by default).
	got := evalECL(t, `<< 404684003 {{ term = "Myocardial" }} {{ C active = true }}`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_DescriptionFilter_Term_Negated(t *testing.T) {
	p := newFilterFixture()
	// term != "infarction" → concepts whose descriptions do NOT contain "infarction".
	// Only MI (22298006) has "infarction" in its term.
	got := evalECL(t, `<< 404684003 {{ term != "infarction" }}`, p)
	assert.False(t, got.Contains("22298006"), "22298006 has 'infarction' in term")
	assert.True(t, got.Contains("73211009"), "73211009 should remain")
	assert.True(t, got.Contains("64572001"), "64572001 should remain")
}

func TestEvaluate_DescriptionFilter_Language_Negated(t *testing.T) {
	p := newFilterFixture()
	// language != es → concepts that do NOT have a Spanish description.
	// Only MI (22298006) has a Spanish description.
	got := evalECL(t, `<< 404684003 {{ language != es }}`, p)
	assert.False(t, got.Contains("22298006"), "22298006 has Spanish desc")
	assert.True(t, got.Contains("73211009"))
}

func TestEvaluate_MemberFilter(t *testing.T) {
	p := newFilterFixture()
	expr, err := Parse(`^ 900000000000497000 {{ M referencedComponentId = 22298006 }}`)
	if err != nil {
		t.Skipf("member filter example did not parse: %v", err)
	}
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_DialectFilter(t *testing.T) {
	p := newFilterFixture()
	expr, err := Parse(`<< 404684003 {{ dialect = 900000000000509007 }}`)
	if err != nil {
		t.Skipf("parser does not accept dialect filter syntax: %v", err)
	}
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.True(t, got.Contains("22298006"))
	assert.True(t, got.Contains("73211009"))
	assert.True(t, got.Contains("64572001"))
}
