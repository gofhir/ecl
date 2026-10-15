package mrcm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/scg"
)

// TestParseConcreteRange_Forms covers the Template Syntax v1.1 rule the MRCM
// Attribute Range refset uses for concrete attributes (concreteValueReplacement),
// with the one leniency the distributed refsets need: integer bounds in a
// decimal set.
func TestParseConcreteRange_Forms(t *testing.T) {
	for _, s := range []string{
		"int", "dec", "str", "bool", " int ( #1 ) ",
		"int(>#0..)", "int(#1..#10)", "int(..<#10)", "int(#-5..#+5)", "int(#1 #3 #5..#7)",
		"dec(>#0..)", "dec(#0.5..#10)", "dec(#0.25 #1.5)",
		`str("mg")`, `str("mg" "mL")`, `str("a \"quoted\" \\ word")`,
		"bool(true)", "bool(TRUE false)", "int /* a comment */ (#1)",
		// ABNF literals are case-insensitive; a range of one value is a value.
		"INT(#1)", "Dec", "int(#5..#5)", "int(>#5..<#7)", "dec(>#5..<#6)", "int(#-1..#0)",
	} {
		_, isConcrete, err := parseConcreteRange(s)
		require.Truef(t, isConcrete, "%q is a concrete range", s)
		require.NoErrorf(t, err, "%q", s)
	}
	for _, s := range []string{
		"int(", "int()", "int(#)", "int(1)", "int(#01)", "int(#1.5)", "int(#1#2)", "int(>#1)",
		"int(..)", "int(#10..#1)", "int(#1..#2", "int(#1) x", "int x",
		"dec(#1.)", `str(mg)`, `str("mg)`, `str("a\x")`, "bool(yes)",
		"int /*", "int(#1)/*", `str("")`,
		// Ranges that hold no value.
		"int(>#5..<#5)", "int(#5..<#5)", "int(>#1..#1)", "int(>#5..<#6)",
		// Only a non-zero integer takes a sign.
		"int(#-0)", "dec(#+0.0)",
	} {
		_, isConcrete, err := parseConcreteRange(s)
		require.Truef(t, isConcrete, "%q starts like a concrete range", s)
		require.Errorf(t, err, "%q is malformed", s)
	}
	_, _, err := parseConcreteRange("int(#1")
	require.ErrorContains(t, err, "expected )")

	for _, s := range []string{"<< 442083009", "*", "^ 723264001", "(<< 1 OR << 2)", "", "integer"} {
		_, isConcrete, err := parseConcreteRange(s)
		require.Falsef(t, isConcrete, "%q is ECL", s)
		require.NoError(t, err)
	}
}

// TestConcreteRange_Contains covers bounds, sets and the value kinds: an
// integer is a decimal number, a decimal is not an integer, and a string or a
// boolean is only its own kind.
func TestConcreteRange_Contains(t *testing.T) {
	integer := func(n int64) *scg.ConcreteValue { return &scg.ConcreteValue{Kind: "integer", Int: n} }
	decimal := func(f float64) *scg.ConcreteValue { return &scg.ConcreteValue{Kind: "decimal", Float: f} }
	for _, tc := range []struct {
		rng string
		in  []*scg.ConcreteValue
		out []*scg.ConcreteValue
	}{
		{"int(>#0..)", []*scg.ConcreteValue{integer(1), integer(1 << 62)}, []*scg.ConcreteValue{integer(0), integer(-1), decimal(1.5)}},
		{"int(#1..<#10)", []*scg.ConcreteValue{integer(1), integer(9)}, []*scg.ConcreteValue{integer(10), integer(0)}},
		{"int(#1 #5..#7)", []*scg.ConcreteValue{integer(1), integer(6)}, []*scg.ConcreteValue{integer(2), integer(8)}},
		{"int", []*scg.ConcreteValue{integer(-3)}, []*scg.ConcreteValue{decimal(2)}},
		{"dec(>#0..)", []*scg.ConcreteValue{decimal(0.001), integer(250)}, []*scg.ConcreteValue{decimal(0), decimal(-0.5), integer(0)}},
		// #0.1 is outside `dec(>#0.1..)` although its float64 is a little above 0.1.
		{"dec(>#0.1..#0.3)", []*scg.ConcreteValue{decimal(0.2), decimal(0.3)}, []*scg.ConcreteValue{decimal(0.1), decimal(0.30000000000000004)}},
		{`str("mg" "mL")`, []*scg.ConcreteValue{{Kind: "string", String: "mL"}}, []*scg.ConcreteValue{{Kind: "string", String: "ml"}, integer(1)}},
		{"bool(true)", []*scg.ConcreteValue{{Kind: "boolean", Bool: true}}, []*scg.ConcreteValue{{Kind: "boolean"}, {Kind: "string", String: "true"}}},
	} {
		r, _, err := parseConcreteRange(tc.rng)
		require.NoError(t, err, tc.rng)
		for _, v := range tc.in {
			require.Truef(t, r.contains(v), "%+v in %s", *v, tc.rng)
		}
		for _, v := range tc.out {
			require.Falsef(t, r.contains(v), "%+v not in %s", *v, tc.rng)
		}
	}
}

// rangeModel is the test model with one more attribute, 1142135004, whose
// range is concrete, and 363698007's range rows given as rngs.
func rangeModel(rngs ...string) *Model {
	m := newTestModel()
	m.Ranges = nil
	for _, r := range rngs {
		m.Ranges = append(m.Ranges, AttributeRange{AttributeID: "363698007", RangeECL: r, RuleStrengthID: RuleStrengthMandatory})
	}
	m.Domains = append(m.Domains, AttributeDomain{
		AttributeID: "1142135004", DomainECL: "<< 404684003", Grouped: true,
		Cardinality: Cardinality{Max: -1}, InGroupCardinality: Cardinality{Max: 1}, RuleStrengthID: RuleStrengthMandatory,
	})
	m.Ranges = append(m.Ranges, AttributeRange{AttributeID: "1142135004", RangeECL: "dec(>#0..)", RuleStrengthID: RuleStrengthMandatory})
	return m
}

// TestValidate_RangeOfEveryValueKind covers H7: a value used to be checked
// against its range only when it was a concept, so `#12` under a concept range
// came back valid. Each kind of value is now checked against each kind of
// range, and what cannot be decided is reported in Unchecked.
func TestValidate_RangeOfEveryValueKind(t *testing.T) {
	type want struct {
		valid, complete bool
		issue           string // the one issue kind, if any
	}
	for _, tc := range []struct {
		name, rng, expr string
		want            want
	}{
		{"concrete value, concept range", "<< 442083009", "22298006:{363698007=#12}", want{false, true, IssueKindRangeViolation}},
		{"concrete value in a concrete range", "<< 442083009", "22298006:{1142135004=#250}", want{true, true, ""}},
		{"concrete value out of a concrete range", "<< 442083009", "22298006:{1142135004=#0}", want{false, true, IssueKindRangeViolation}},
		{"concept, concrete range", "<< 442083009", "22298006:{1142135004=74281007}", want{false, true, IssueKindRangeViolation}},
		{"nested, focus in << range", "<< 442083009", "22298006:{363698007=(74281007:{363698007=425391005})}", want{false, true, IssueKindDomainViolation}},
		{"nested, focus out of << range", "<< 442083009", "22298006:{363698007=(73211009)}", want{false, true, IssueKindRangeViolation}},
		{"nested, one focus of two in range", "<< 442083009", "22298006:{363698007=(73211009+74281007)}", want{true, true, ""}},
		{"nested, range of one concept", "74281007", "22298006:{363698007=(74281007)}", want{true, false, ""}},
		{"nested, range with MINUS", "<< 442083009 MINUS 425391005", "22298006:{363698007=(74281007)}", want{true, false, ""}},
		{"nested, concrete range", "<< 442083009", "22298006:{1142135004=(74281007)}", want{false, true, IssueKindRangeViolation}},
		// A broken row is not decided: the value may be in it.
		{"malformed concrete range", "int(#1", "22298006:{363698007=#1}", want{false, false, IssueKindInvalidRule}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Validate(context.Background(), mustParseSCG(t, tc.expr), rangeModel(tc.rng), newTestProvider())
			require.NoError(t, err)
			require.Equal(t, tc.want.valid, res.Valid, "issues: %+v", res.Issues)
			require.Equal(t, tc.want.complete, res.Complete, "unchecked: %+v", res.Unchecked)
			if tc.want.issue == "" {
				require.Empty(t, res.Issues)
			} else {
				require.Len(t, res.Issues, 1, "%+v", res.Issues)
				require.Equal(t, tc.want.issue, res.Issues[0].Kind)
			}
			for _, u := range res.Unchecked {
				require.Equal(t, IssueKindUncheckedRange, u.Kind)
			}
		})
	}
}

// TestValidate_UncheckedIsReportedOnce covers an expression with two focus
// concepts: the range is checked per focus, and does not depend on it.
func TestValidate_UncheckedIsReportedOnce(t *testing.T) {
	res, err := Validate(context.Background(), mustParseSCG(t, "22298006+73211009:{363698007=(74281007)}"), rangeModel("74281007"), newTestProvider())
	require.NoError(t, err)
	require.Len(t, res.Unchecked, 1)
	require.Equal(t, "(74281007)", res.Unchecked[0].ValueID)
	require.Equal(t, "refinement[0].attr[0].value", res.Unchecked[0].Path)
}

// TestValidate_NestedRangeAsWritten covers nested expressions against `< X`
// and AND: a refined expression is strictly below its focus concept, and the
// sides of an AND may be satisfied by different focus concepts.
func TestValidate_NestedRangeAsWritten(t *testing.T) {
	rangeIssues := func(rng, expr string) []Issue {
		t.Helper()
		res, err := Validate(context.Background(), mustParseSCG(t, expr), rangeModel(rng), newTestProvider())
		require.NoError(t, err)
		require.True(t, res.Complete, "%+v", res.Unchecked)
		var out []Issue
		for _, iss := range res.Issues {
			// The outer value only: the nested expression's own attribute is
			// checked against the same range.
			if iss.Kind == IssueKindRangeViolation && iss.Path == "refinement[0].attr[0].value" {
				out = append(out, iss)
			}
		}
		return out
	}
	// Refined, 74281007 : … is below 74281007; alone, it is 74281007.
	require.Empty(t, rangeIssues("< 74281007", "22298006:{363698007=(74281007:{363698007=425391005})}"))
	require.Len(t, rangeIssues("< 74281007", "22298006:{363698007=(74281007)}"), 1)
	// Each side of the AND by its own focus concept.
	const both = "<< 442083009 AND << 404684003"
	require.Empty(t, rangeIssues(both, "22298006:{363698007=(74281007+22298006)}"))
	require.Len(t, rangeIssues(both, "22298006:{363698007=(74281007)}"), 1)
}

// TestValidate_RangeRowsPrecedence covers range rows as alternatives: one
// satisfied row decides, a row that cannot be decided — the validator cannot
// tell, or the rule is broken — keeps the value from being called out of
// range, and only rows all decided against it make a violation.
func TestValidate_RangeRowsPrecedence(t *testing.T) {
	const nested = "22298006:{363698007=(74281007)}"
	for _, tc := range []struct {
		name            string
		rows            []string
		valid, complete bool
		kinds           []string
	}{
		{"satisfied and unchecked", []string{"74281007", "<< 442083009"}, true, true, nil},
		{"unchecked and violated", []string{"74281007", "<< 404684003"}, true, false, nil},
		{"broken and violated", []string{"int(#1", "<< 404684003"}, false, false, []string{IssueKindInvalidRule}},
		{"violated twice", []string{"<< 404684003", "<< 73211009"}, false, true, []string{IssueKindRangeViolation}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Validate(context.Background(), mustParseSCG(t, nested), rangeModel(tc.rows...), newTestProvider())
			require.NoError(t, err)
			require.Equal(t, tc.valid, res.Valid, "%+v", res.Issues)
			require.Equal(t, tc.complete, res.Complete, "%+v", res.Unchecked)
			var kinds []string
			for _, iss := range res.Issues {
				kinds = append(kinds, iss.Kind)
			}
			require.Equal(t, tc.kinds, kinds)
		})
	}
}

// TestValidate_RangeViolationIsReportedOnce covers an expression with two
// focus concepts and a value out of range: one violation, not one per focus.
func TestValidate_RangeViolationIsReportedOnce(t *testing.T) {
	res, err := Validate(context.Background(), mustParseSCG(t, "22298006+73211009:{363698007=#5}"), rangeModel("<< 442083009"), newTestProvider())
	require.NoError(t, err)
	require.Len(t, res.Issues, 1, "%+v", res.Issues)
	require.Equal(t, IssueKindRangeViolation, res.Issues[0].Kind)
}

// TestValidate_RangeFormsEndToEnd covers the range forms the unit tests check
// one by one, through Validate: an OR and `*` against a nested value, a string
// and a boolean range, and an attribute with no value.
func TestValidate_RangeFormsEndToEnd(t *testing.T) {
	validate := func(m *Model, expr *scg.Expression) *Result {
		t.Helper()
		res, err := Validate(context.Background(), expr, m, newTestProvider())
		require.NoError(t, err)
		return res
	}
	outer := func(res *Result) []string {
		var kinds []string
		for _, iss := range res.Issues {
			if iss.Path == "refinement[0].attr[0].value" {
				kinds = append(kinds, iss.Kind)
			}
		}
		return kinds
	}
	for rng, wantIn := range map[string]bool{
		"<< 404684003 OR << 442083009": true,
		"<< 404684003 OR << 386053000": false,
		"*":                            true,
	} {
		res := validate(rangeModel(rng), mustParseSCG(t, "22298006:{363698007=(74281007)}"))
		require.Equal(t, wantIn, outer(res) == nil, "%s: %+v", rng, res.Issues)
		require.True(t, res.Complete, rng)
	}

	for rng, values := range map[string]map[string]bool{
		`str("mg" "mL")`: {`"mg"`: true, `"g"`: false, "#1": false},
		"bool(true)":     {"true": true, "false": false, `"true"`: false},
	} {
		for value, in := range values {
			res := validate(rangeModel(rng), mustParseSCG(t, "22298006:{363698007="+value+"}"))
			require.Equal(t, in, res.Valid, "%s in %s: %+v", value, rng, res.Issues)
		}
	}

	// Only a hand-built expression has an attribute with no value.
	expr := mustParseSCG(t, "22298006:{363698007=74281007}")
	expr.Refinements[0].Attributes[0].Value = scg.AttributeValue{}
	res := validate(rangeModel("<< 442083009"), expr)
	require.True(t, res.Valid, "%+v", res.Issues)
}

// TestValidate_NestedRangeErrorStopsTheCheck covers a provider failing on the
// left side of an OR while a nested value is checked: the failure propagates,
// although the right side alone would answer, and it is not read as the value
// being out of range nor as a broken rule.
func TestValidate_NestedRangeErrorStopsTheCheck(t *testing.T) {
	_, err := Validate(context.Background(), mustParseSCG(t, "22298006:{363698007=(74281007)}"),
		&Model{Ranges: []AttributeRange{{AttributeID: "363698007", RangeECL: "<< 404684003 OR << 442083009", RuleStrengthID: RuleStrengthMandatory}}},
		failingOn{stubProvider: newTestProvider(), id: "404684003"})
	require.ErrorIs(t, err, errBackendDownOn)
	require.NotErrorIs(t, err, errInvalidRule)
}

var errBackendDownOn = errors.New("backend down")

// failingOn fails the hierarchy calls that name one concept.
type failingOn struct {
	*stubProvider
	id string
}

func (f failingOn) Descendants(ctx context.Context, ids []string, includeSelf bool) (ecl.Set, error) {
	for _, id := range ids {
		if id == f.id {
			return nil, errBackendDownOn
		}
	}
	return f.stubProvider.Descendants(ctx, ids, includeSelf)
}
