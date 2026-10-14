package mrcm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl/providertest"
	"github.com/gofhir/ecl/scg"
)

// TestLoad_RejectsWhatTheModelCannotMean covers H6 of
// docs/analisis-adversarial-2026-10-12.md: documents the loader accepts although
// they do not say what the caller meant. Both shapes turn a configuration error
// into a model that validates less than intended, with no error anywhere.
//
//   - A second JSON value after the model: Decode reads one value and stops, so
//     whatever follows — a concatenated file, a stray paste — is dropped, and
//     DisallowUnknownFields never sees it.
//   - A rule strength that is a well-formed SCTID but neither of the two
//     strengths the validator knows: the validator skips every rule that is not
//     mandatory, so a typo silently turns a mandatory rule off.
//
// `null` is left out on purpose. Whether it means "no rules" or "no document"
// is a policy decision the analysis leaves open, and a test here would encode a
// guess as the requirement.
//
// Each case states the outcome the loader must have (reject). These loaded
// without error until the loader required the end of the document and one of
// the two known strengths; they are ordinary regression cases now.
//
// "The right reason" is pinned by control: the same document without the
// offending part, which must load. The two differ only in that part, so an
// error on doc alone can only come from it, whatever its wording. The names field adds a
// token the error must mention where the wording is predictable (the SCTID of
// an unknown strength); the trailing-value cases rely on control alone.
func TestLoad_RejectsWhatTheModelCannotMean(t *testing.T) {
	const domain = `{"attributeId": "363698007", "domainEcl": "<< 404684003", "ruleStrengthId": "%s"}`
	const rng = `{"attributeId": "363698007", "rangeEcl": "<< 123037004", "ruleStrengthId": "%s"}`
	// unknownStrength is a well-formed SCTID (a module) that appears nowhere
	// else in either rule, so an error naming it is about the strength.
	const unknownStrength = "900000000000012004"
	withStrength := func(rule, field, strength string) string {
		return `{"` + field + `": [` + strings.Replace(rule, "%s", strength, 1) + `]}`
	}
	for _, tc := range []struct {
		name, doc, control, names string
		reject                    bool
	}{
		{name: "trailing JSON value", doc: `{} {"unexpected": true}`, control: `{}`, reject: true},
		{name: "trailing model", doc: `{"domains": []} {"domains": []}`, control: `{"domains": []}`, reject: true},
		{name: "trailing garbage", doc: `{} x`, control: `{}`, reject: true},
		// A stray closing bracket, which json.Decoder.More does not report.
		{name: "trailing brace", doc: `{} }`, control: `{}`, reject: true},
		{name: "trailing bracket", doc: `{"domains": []}]`, control: `{"domains": []}`, reject: true},
		// Whitespace after the model is not data.
		{name: "trailing whitespace", doc: "{} \n\t\r "},
		{
			name:    "unknown domain rule strength",
			doc:     withStrength(domain, "domains", unknownStrength),
			control: withStrength(domain, "domains", RuleStrengthMandatory),
			names:   unknownStrength, reject: true,
		},
		{
			name:    "unknown range rule strength",
			doc:     withStrength(rng, "ranges", unknownStrength),
			control: withStrength(rng, "ranges", RuleStrengthMandatory),
			names:   unknownStrength, reject: true,
		},
		// The two known strengths keep loading, so the fix cannot simply reject
		// every explicit strength.
		{name: "mandatory strength", doc: withStrength(domain, "domains", RuleStrengthMandatory)},
		{name: "optional strength", doc: withStrength(domain, "domains", RuleStrengthOptional)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.control != "" {
				_, err := LoadFromBytes([]byte(tc.control))
				require.NoErrorf(t, err, "the control %s must load, or the case cannot isolate its defect", tc.control)
			}
			_, err := LoadFromBytes([]byte(tc.doc))
			rejected := err != nil && strings.Contains(err.Error(), tc.names)
			switch {
			case !tc.reject:
				require.NoErrorf(t, err, "%s must load", tc.doc)
			case err != nil && !rejected:
				t.Fatalf("%s fails, but the error does not name %s: %v", tc.doc, tc.names, err)
			default:
				require.Truef(t, rejected, "%s must be rejected", tc.doc)
			}
		})
	}
}

// TestValidate_RefusesUnknownRuleStrength covers a Model built in Go rather than
// loaded: the loader refuses an unknown strength, but the validator skips every
// rule that is not mandatory, so it has to refuse one too.
func TestValidate_RefusesUnknownRuleStrength(t *testing.T) {
	model := &Model{Domains: []AttributeDomain{{
		AttributeID: "363698007", DomainECL: "*", Cardinality: Cardinality{Max: -1},
		RuleStrengthID: "723597002", // one digit off mandatory
	}}}
	expr, err := scg.Parse("22298006 : 363698007 = 74281007")
	require.NoError(t, err)
	_, err = Validate(context.Background(), expr, model, providertest.NewInMemoryProvider(&providertest.FixtureSpec{}))
	require.ErrorContains(t, err, "723597002")
}
