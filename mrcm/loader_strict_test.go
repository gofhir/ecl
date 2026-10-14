package mrcm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
// Each case states the outcome the loader SHOULD have (reject) separately from
// the defect it still has, so dropping a marker turns the case into the
// regression test rather than inverting it. A case marked H6 is a strict
// expected failure, as in ecl/semantics_test.go: it passes while the document
// still loads, and fails once it is rejected for the right reason.
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
		defect                    string
	}{
		{name: "trailing JSON value", doc: `{} {"unexpected": true}`, control: `{}`, reject: true, defect: "H6"},
		{name: "trailing model", doc: `{"domains": []} {"domains": []}`, control: `{"domains": []}`, reject: true, defect: "H6"},
		{name: "trailing garbage", doc: `{} x`, control: `{}`, reject: true, defect: "H6"},
		{
			name:    "unknown domain rule strength",
			doc:     withStrength(domain, "domains", unknownStrength),
			control: withStrength(domain, "domains", RuleStrengthMandatory),
			names:   unknownStrength, reject: true, defect: "H6",
		},
		{
			name:    "unknown range rule strength",
			doc:     withStrength(rng, "ranges", unknownStrength),
			control: withStrength(rng, "ranges", RuleStrengthMandatory),
			names:   unknownStrength, reject: true, defect: "H6",
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
			case tc.defect == "":
				require.Truef(t, rejected, "%s must be rejected", tc.doc)
			case rejected:
				t.Fatalf("%s is now rejected (%v), so defect %s looks fixed: drop its marker", tc.doc, err, tc.defect)
			default:
				t.Logf("KNOWN DEFECT %s: %s loads without error", tc.defect, tc.doc)
			}
		})
	}
}
