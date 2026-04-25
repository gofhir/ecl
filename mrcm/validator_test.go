package mrcm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/scg"
)

// stubProvider is a minimal DataProvider for MRCM validator tests.
//
// It supports only the methods the ECL engine needs to evaluate "<<", "<",
// concept refs and similar simple expressions used by the test fixtures.
type stubProvider struct {
	descendants map[string][]string // id -> descendants (excluding self)
	exists      map[string]bool
}

var _ ecl.DataProvider = (*stubProvider)(nil)

func (s *stubProvider) Descendants(_ context.Context, ids []string, includeSelf bool) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range ids {
		if includeSelf {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
		}
		if d, ok := s.descendants[id]; ok {
			out = out.Union(ecl.NewSetFromSlice(d))
		}
	}
	return out, nil
}

func (s *stubProvider) Ancestors(_ context.Context, _ []string, _ bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) Children(_ context.Context, _ []string, _ bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) Parents(_ context.Context, _ []string, _ bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	out := ecl.NewSet()
	have := make([]string, 0, len(ids))
	for _, id := range ids {
		if s.exists[id] {
			have = append(have, id)
		}
	}
	return out.Union(ecl.NewSetFromSlice(have)), nil
}

func (s *stubProvider) AllConcepts(_ context.Context) (ecl.Set, error) {
	all := make([]string, 0, len(s.exists))
	for id, ok := range s.exists {
		if ok {
			all = append(all, id)
		}
	}
	return ecl.NewSetFromSlice(all), nil
}

func (s *stubProvider) RelationshipTargets(_ context.Context, _, _ ecl.Set) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) RelationshipSources(_ context.Context, _, _ ecl.Set) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) ConcreteValues(_ context.Context, _, _ string) ([]ecl.ConcreteValue, error) {
	return nil, nil
}

func (s *stubProvider) PropertiesByGroup(_ context.Context, _ string) (map[int][]ecl.Relationship, error) {
	return nil, nil
}

func (s *stubProvider) MatchDescription(_ context.Context, _ ecl.DescriptionFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) FilterConcepts(_ context.Context, _ ecl.Set, _ ecl.ConceptFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) RefsetMembers(_ context.Context, _ []string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) RefsetsContainingMembers(_ context.Context, _ []string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) HistoricalAssociations(_ context.Context, _ ecl.Set, _ string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) ResolveIdentifier(_ context.Context, _, _ string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) MatchDialect(_ context.Context, _ ecl.Set, _ ecl.DialectFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (s *stubProvider) RefsetMembersFiltered(_ context.Context, _ []string, _ ecl.MemberFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

// ------------------------------------------------------------------------
// Test fixture
//
// Hierarchy:
//
//   404684003 (clinical finding)
//     ├── 22298006 (myocardial infarction)
//     └── 73211009 (diabetes)
//
//   442083009 (anatomical or acquired body structure)
//     ├── 74281007 (heart structure)
//     └── 425391005 (thymus structure)
//
//   386053000 (evaluation procedure)  // unrelated, used as off-domain focus
//
// MRCM (default test model):
//
//   363698007 (Finding site)
//     domain: << 404684003   grouped=true   cardinality=0..*  inGroup=0..1
//     range:  << 442083009
// ------------------------------------------------------------------------.

func newTestProvider() *stubProvider {
	return &stubProvider{
		descendants: map[string][]string{
			"404684003": {"22298006", "73211009"},
			"442083009": {"74281007", "425391005"},
		},
		exists: map[string]bool{
			"404684003": true,
			"22298006":  true,
			"73211009":  true,
			"442083009": true,
			"74281007":  true,
			"425391005": true,
			"386053000": true,
			"363698007": true,
		},
	}
}

func newTestModel() *Model {
	return &Model{
		Domains: []AttributeDomain{
			{
				AttributeID:        "363698007",
				DomainECL:          "<< 404684003",
				Grouped:            true,
				Cardinality:        Cardinality{Min: 0, Max: -1},
				InGroupCardinality: Cardinality{Min: 0, Max: 1},
				RuleStrengthID:     RuleStrengthMandatory,
			},
		},
		Ranges: []AttributeRange{
			{
				AttributeID:    "363698007",
				RangeECL:       "<< 442083009",
				RuleStrengthID: RuleStrengthMandatory,
			},
		},
	}
}

func mustParseSCG(t *testing.T, s string) *scg.Expression {
	t.Helper()
	expr, err := scg.Parse(s)
	require.NoError(t, err)
	return expr
}

// ------------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------------.

func TestValidate_ValidExpression(t *testing.T) {
	// Focus 22298006 is in << 404684003 ✓, value 74281007 is in << 442083009 ✓.
	// Attribute is grouped (rule says grouped=true) by using "{ ... }".
	expr := mustParseSCG(t, "22298006:{363698007=74281007}")
	res, err := Validate(context.Background(), expr, newTestModel(), newTestProvider())
	require.NoError(t, err)

	assert.True(t, res.Valid, "issues: %+v", res.Issues)
	assert.Empty(t, res.Issues)
}

func TestValidate_DomainViolation(t *testing.T) {
	// 386053000 (evaluation procedure) is NOT a clinical finding -> domain_violation.
	expr := mustParseSCG(t, "386053000:{363698007=74281007}")
	res, err := Validate(context.Background(), expr, newTestModel(), newTestProvider())
	require.NoError(t, err)

	assert.False(t, res.Valid)
	require.NotEmpty(t, res.Issues)

	var found bool
	for _, iss := range res.Issues {
		if iss.Kind == IssueKindDomainViolation && iss.AttributeID == "363698007" && iss.FocusID == "386053000" {
			found = true
		}
	}
	assert.True(t, found, "expected domain_violation in issues: %+v", res.Issues)
}

func TestValidate_RangeViolation(t *testing.T) {
	// Value 73211009 (diabetes) is a clinical finding, NOT a body structure -> range_violation.
	expr := mustParseSCG(t, "22298006:{363698007=73211009}")
	res, err := Validate(context.Background(), expr, newTestModel(), newTestProvider())
	require.NoError(t, err)

	assert.False(t, res.Valid)
	require.NotEmpty(t, res.Issues)

	var found bool
	for _, iss := range res.Issues {
		if iss.Kind == IssueKindRangeViolation && iss.AttributeID == "363698007" && iss.ValueID == "73211009" {
			found = true
		}
	}
	assert.True(t, found, "expected range_violation in issues: %+v", res.Issues)
}

func TestValidate_UnknownAttribute(t *testing.T) {
	// Attribute 116676008 has no rules in the test model -> unknown_attribute.
	expr := mustParseSCG(t, "22298006:{116676008=74281007}")
	res, err := Validate(context.Background(), expr, newTestModel(), newTestProvider())
	require.NoError(t, err)

	require.NotEmpty(t, res.Issues)
	var found bool
	for _, iss := range res.Issues {
		if iss.Kind == IssueKindUnknownAttribute && iss.AttributeID == "116676008" {
			found = true
		}
	}
	assert.True(t, found, "expected unknown_attribute issue: %+v", res.Issues)
}

func TestValidate_GroupedViolation(t *testing.T) {
	// Rule says attribute must be grouped, but expression is ungrouped.
	expr := mustParseSCG(t, "22298006:363698007=74281007")
	res, err := Validate(context.Background(), expr, newTestModel(), newTestProvider())
	require.NoError(t, err)

	assert.False(t, res.Valid)
	var found bool
	for _, iss := range res.Issues {
		if iss.Kind == IssueKindUngroupedViolation && iss.AttributeID == "363698007" {
			found = true
		}
	}
	assert.True(t, found, "expected ungrouped_violation in issues: %+v", res.Issues)
}

func TestValidate_NestedExpression(t *testing.T) {
	// Top-level: 22298006 with grouped 363698007 = (74281007:{363698007=425391005})
	// Inner expression: focus 74281007 must itself be in << 404684003 for the
	// inner attribute to validate. It isn't (74281007 is body structure), so
	// the inner attribute use is a domain violation. We verify the validator
	// recurses (i.e. surfaces an issue from the inner expression).
	expr := mustParseSCG(t, "22298006:{363698007=(74281007:{363698007=425391005})}")
	res, err := Validate(context.Background(), expr, newTestModel(), newTestProvider())
	require.NoError(t, err)

	// Outer attribute is fine; inner has a domain violation on focus 74281007.
	var inner bool
	for _, iss := range res.Issues {
		if iss.Kind == IssueKindDomainViolation && iss.FocusID == "74281007" {
			inner = true
		}
	}
	assert.True(t, inner, "expected nested domain_violation on focus 74281007: %+v", res.Issues)
}

func TestValidate_MultipleAttrsAllValid(t *testing.T) {
	// Two grouped attributes, both with the same MRCM-defined attribute and
	// valid focus + values.
	model := newTestModel()
	// Add a second attribute (246075003 = causative agent) with same domain
	// and the body-structure range so the test stays self-contained.
	model.Domains = append(model.Domains, AttributeDomain{
		AttributeID:        "246075003",
		DomainECL:          "<< 404684003",
		Grouped:            false,
		Cardinality:        Cardinality{Min: 0, Max: -1},
		InGroupCardinality: Cardinality{Min: 0, Max: 1},
		RuleStrengthID:     RuleStrengthMandatory,
	})
	model.Ranges = append(model.Ranges, AttributeRange{
		AttributeID:    "246075003",
		RangeECL:       "<< 442083009",
		RuleStrengthID: RuleStrengthMandatory,
	})

	provider := newTestProvider()
	provider.exists["246075003"] = true

	expr := mustParseSCG(t, "22298006:246075003=425391005,{363698007=74281007}")
	res, err := Validate(context.Background(), expr, model, provider)
	require.NoError(t, err)

	assert.True(t, res.Valid, "issues: %+v", res.Issues)
}

func TestValidate_PartialFailure(t *testing.T) {
	// One valid attribute + one with range violation => Valid=false, exactly
	// one range_violation issue. We use the model from
	// TestValidate_MultipleAttrsAllValid so two attributes are known.
	model := newTestModel()
	model.Domains = append(model.Domains, AttributeDomain{
		AttributeID:        "246075003",
		DomainECL:          "<< 404684003",
		Grouped:            false,
		Cardinality:        Cardinality{Min: 0, Max: -1},
		InGroupCardinality: Cardinality{Min: 0, Max: 1},
		RuleStrengthID:     RuleStrengthMandatory,
	})
	model.Ranges = append(model.Ranges, AttributeRange{
		AttributeID:    "246075003",
		RangeECL:       "<< 442083009",
		RuleStrengthID: RuleStrengthMandatory,
	})

	provider := newTestProvider()
	provider.exists["246075003"] = true

	// 246075003 = 73211009 (clinical finding, NOT a body structure) -> range violation.
	// Inside grouped: 363698007 = 74281007 (valid body structure).
	expr := mustParseSCG(t, "22298006:246075003=73211009,{363698007=74281007}")
	res, err := Validate(context.Background(), expr, model, provider)
	require.NoError(t, err)

	assert.False(t, res.Valid)
	var rangeViols int
	for _, iss := range res.Issues {
		if iss.Kind == IssueKindRangeViolation {
			rangeViols++
		}
	}
	assert.Equal(t, 1, rangeViols)
}

func TestValidate_NilArgs(t *testing.T) {
	_, err := Validate(context.Background(), nil, newTestModel(), newTestProvider())
	assert.Error(t, err)

	expr := mustParseSCG(t, "22298006")
	_, err = Validate(context.Background(), expr, nil, newTestProvider())
	assert.Error(t, err)
	_, err = Validate(context.Background(), expr, newTestModel(), nil)
	assert.Error(t, err)
}
