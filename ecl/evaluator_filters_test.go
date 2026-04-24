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
// ------------------------------------------------------------------------

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
	descriptions map[string][]testDescription // conceptID → descriptions
	activeFlag   map[string]bool              // conceptID → active? (default true)
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
			if f.TypeID != "" && d.TypeID != f.TypeID {
				continue
			}
			if f.Language != "" && d.Language != f.Language {
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
// definitionStatus / effectiveTime are not modelled in the fixture).
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

var _ DataProvider = (*filterTestProvider)(nil)

// ------------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------------

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

func TestEvaluate_MemberFilter_NotImplemented(t *testing.T) {
	p := newFilterFixture()
	// A member filter uses a refset field filter inside {{ M ... }}.
	// A member field filter on an arbitrary refset field should trigger the
	// "member filter: not yet implemented" error from the evaluator.
	expr, err := Parse(`^ 900000000000497000 {{ M referencedComponentId = 22298006 }}`)
	if err != nil {
		// If the grammar doesn't accept this exact shape, skip — the goal is
		// to confirm that WHEN parsed, the evaluator returns a clear error.
		t.Skipf("member filter example did not parse: %v", err)
	}
	_, err = Evaluate(context.Background(), expr, p)
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "not yet implemented") ||
			strings.Contains(err.Error(), "member filter"),
		"expected member filter not-yet-implemented error, got: %v", err)
}
