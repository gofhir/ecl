package ecl_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/providertest"
)

// Semantic regressions with the result the SPECIFICATION requires, not the one
// the evaluator happens to return. They come from the adversarial review in
// docs/analisis-adversarial-2026-10-12.md. The defect markers name its findings,
// except H2b, which the document does not list and is defined here:
//
//	H1   D-prefixed active/moduleId/effectiveTime filters read the concept.
//	     The fix belongs in the evaluator (categorizeFilters files them under
//	     the concept family); the in-memory provider already answers them per
//	     description once they arrive in DescriptionFilterOpts. That struct
//	     cannot carry all of them yet: EffectiveTime holds one value, and
//	     ModuleIDs has no negated form.
//	H2   repeated clauses inside ONE block are not a conjunction.
//	H2b  separate blocks are not independent either: they are flattened into
//	     one predicate. The last term overrides the earlier ones (so block order
//	     changes the result), languages are pooled into an any-of, and a term and
//	     a language from DIFFERENT blocks must hold on the same description —
//	     which breaks the shape of official example 8.2.2. Found after the
//	     review, while checking its pending note on blocks.
//	H3   term and dialect in one block may be satisfied by different
//	     descriptions.
//	H4   numeric literals lose precision (float64) or saturate on overflow.
//
// A case with a defect marker is a STRICT expected failure: it passes while the
// evaluator still returns something other than the specified result, and fails
// once the two agree, so whoever fixes a defect has to remove its marker and the
// case becomes an ordinary regression test. A case without a marker already
// returns the specified result; several of those exist only to stop a fix for
// one reading (same description within a block) from breaking the other
// (possibly different descriptions across blocks).
//
// Normative basis, both in ecl/testdata/official-examples/8_description_filters:
//
//   - 8.2.2 `{{ term = "hjärt", language = sv }} {{ term = "heart", language = en }}`
//     matches "diseases with a Swedish description containing hjärt and an
//     English description containing heart": the clauses of one block hold on
//     the same description, separate blocks may hold on different ones.
//   - 8.1.8 `{{ term = "eye" }} {{ term = wild:"*itis" }}`: "the descriptions
//     that match each of the filters can either be the same description, or
//     different descriptions on the same concept".
//
// Section 6.8 also states that the description active, module and effective time
// filters match "concepts with a description" having that property.

const (
	semHeart     = "22298006"           // en "heart" (en-US); es "corazon" (en-GB, other module, newer); es "old" (inactive, en-GB)
	semOther     = "73211009"           // en "heart" (en-GB)
	semOldTime   = "20020131"           // semHeart's own effectiveTime
	semNewTime   = "20240131"           // the effectiveTime of semHeart's "corazon" only
	semModule    = "900000000000012004" // a module SCTID no concept of the fixture declares
	semGB        = "900000000000508004"
	semUS        = "900000000000509007"
	semPreferred = "900000000000548007"
	semStrength  = "1142139005"
)

// semanticsProvider is the smallest terminology that tells the readings apart.
// Each fact is there for a case below; none is incidental.
func semanticsProvider() ecl.DataProvider {
	inactive := false
	return providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		// The attribute is declared because an attribute name resolves like any
		// concept reference: undeclared, it names nothing and every concrete case
		// would return the empty set, agreeing with the specification vacuously.
		// No description is given to it, so it never enters a description filter.
		Concepts: []providertest.FixtureConcept{{ID: semHeart, EffectiveTime: semOldTime}, {ID: semOther}, {ID: semStrength}},
		Descriptions: []providertest.FixtureDescription{
			{ID: "200001011", Concept: semHeart, Language: "en", Value: "heart"},
			{ID: "200002017", Concept: semHeart, Language: "es", Value: "corazon", ModuleID: semModule, EffectiveTime: semNewTime},
			{ID: "200003010", Concept: semHeart, Language: "es", Value: "old", Active: &inactive},
			{ID: "200004016", Concept: semOther, Language: "en", Value: "heart"},
		},
		// The fixture keys dialect membership by concept and language, so the
		// en-GB membership of semHeart covers both Spanish descriptions, the
		// inactive one included. That is coarser than RF2, where membership
		// names a description: the H3 cases below tell term and dialect apart
		// across LANGUAGES, but a fix that correlated them per language instead
		// of per description would pass them. Closing that needs the fixture to
		// accept a description ID on dialect members.
		Dialects: []providertest.FixtureDialectMember{
			{ConceptID: semHeart, Language: "en", DialectID: semUS, AcceptabilityID: semPreferred},
			{ConceptID: semHeart, Language: "es", DialectID: semGB, AcceptabilityID: semPreferred},
			{ConceptID: semOther, Language: "en", DialectID: semGB, AcceptabilityID: semPreferred},
		},
		// 2^53 is the first integer above which float64 cannot represent every
		// integer, so 2^53+1 rounds onto it. MaxInt64 is where an overflowing
		// literal saturates, which makes that saturation show up as a match.
		Relationships: []providertest.FixtureRelationship{
			{Source: semHeart, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "integer", Value: "9007199254740992"}},
			{Source: semOther, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "integer", Value: "9223372036854775807"}},
		},
	})
}

type semCase struct {
	expr string
	want []string // what the specification requires
	// defect names the finding the evaluator still exhibits; empty when it
	// already conforms.
	defect string
	// now is what the evaluator returns while the defect stands. Pinning it
	// turns "anything but want" into one exact wrong answer, so a change that
	// trades one wrong result for another fails instead of passing unnoticed.
	// A defect case must set it; an empty pinned result is written `none`, so
	// that a forgotten pin cannot pass for a deliberate empty one.
	now []string
}

func TestSemantics_DescriptionFilters(t *testing.T) {
	both := []string{semHeart, semOther}
	heart := []string{semHeart}
	runSemCases(t, []semCase{
		// H1 — the D prefix scopes these to descriptions.
		{expr: `* {{ D active = false }}`, want: heart, defect: "H1", now: none},
		{expr: `* {{ D moduleId = ` + semModule + ` }}`, want: heart, defect: "H1", now: none},
		{expr: `* {{ D effectiveTime = "` + semNewTime + `" }}`, want: heart, defect: "H1", now: none},
		// No description carries the concept's own time; today the concept's is read.
		{expr: `* {{ D effectiveTime = "` + semOldTime + `" }}`, want: nil, defect: "H1", now: heart},
		// The concept-scoped forms, which must not move when H1 is fixed.
		{expr: `* {{ C active = false }}`, want: nil},
		{expr: `* {{ C moduleId = ` + semModule + ` }}`, want: nil},
		{expr: `* {{ C effectiveTime = "` + semOldTime + `" }}`, want: heart},

		// H2 — one block, one description: two values of one column cannot both hold.
		{expr: `* {{ D language = en, language = es }}`, want: nil, defect: "H2", now: both},
		{expr: `* {{ D term = "nonexistent", term = "heart" }}`, want: nil, defect: "H2", now: both},
		{expr: `* {{ D term = "heart", term = "corazon" }}`, want: nil, defect: "H2", now: heart},
		// A value list inside ONE clause is any-of, unlike repeated clauses. The
		// list is space-separated (languageCodeSet); `(en OR es)` would parse
		// "OR" as a third two-letter code. No description is French, so reading
		// only the first value fails the second case, only the last fails the
		// first, and dropping the clause returns all of `*`.
		{expr: `* {{ D language = (es fr) }}`, want: heart},
		{expr: `* {{ D language = (fr es) }}`, want: heart},

		// H2b — separate blocks, each satisfied on its own.
		{expr: `* {{ D term = "corazon" }} {{ D term = "heart" }}`, want: heart, defect: "H2b", now: both},
		{expr: `* {{ D term = "nonexistent" }} {{ D term = "corazon" }}`, want: nil, defect: "H2b", now: heart},
		{expr: `* {{ D term = "heart" }} {{ D term = "corazon" }}`, want: heart},
		{expr: `* {{ D term = "corazon" }} {{ D term = "nonexistent" }}`, want: nil},
		// Term and language from different blocks may hold on different
		// descriptions; the second case is the shape of official example 8.2.2.
		{expr: `* {{ D term = "heart" }} {{ D language = es }}`, want: heart, defect: "H2b", now: none},
		{expr: `* {{ D term = "corazon", language = es }} {{ D term = "heart", language = en }}`, want: heart, defect: "H2b", now: both},
		// semOther has no Spanish description; the blocks' languages are pooled into an any-of.
		{expr: `* {{ D language = es }} {{ D language = en }}`, want: heart, defect: "H2b", now: both},

		// H3 — term and dialect on the SAME description: semHeart's "heart" is en-US only.
		{expr: `* {{ D term = "heart", dialectId = ` + semGB + ` }}`, want: []string{semOther}, defect: "H3", now: both},
		// Across blocks semHeart qualifies through "corazon"; a fix for H3 that
		// correlated blocks too would wrongly drop it.
		{expr: `* {{ D term = "heart" }} {{ D dialectId = ` + semGB + ` }}`, want: both},
		{expr: `* {{ D dialectId = ` + semGB + ` }} {{ D term = "heart" }}`, want: both},
		{expr: `* {{ D term = "corazon" }} {{ D dialectId = ` + semUS + ` }}`, want: heart},
	})
}

// TestSemantics_ConcreteNumbers uses integer literals only. Whether a decimal
// literal compares numerically with an integer value is not settled by any text
// this suite could cite, so such a case would encode a guess as the specified
// result; it stays out until the policy is decided.
func TestSemantics_ConcreteNumbers(t *testing.T) {
	require64BitInt(t)
	attr := `* : ` + semStrength + ` `
	heart, other := []string{semHeart}, []string{semOther}
	runSemCases(t, []semCase{
		{expr: attr + `= #9007199254740992`, want: heart},
		{expr: attr + `= #9007199254740991`, want: nil},
		{expr: attr + `= #9007199254740994`, want: nil},
		{expr: attr + `= #9007199254740993`, want: nil, defect: "H4", now: heart},
		{expr: attr + `< #9007199254740993`, want: heart, defect: "H4", now: none},
		{expr: attr + `= #9223372036854775807`, want: other},
		{expr: attr + `= #9223372036854775806`, want: nil, defect: "H4", now: other},
	})
}

// TestSemantics_NumericOverflowIsRejected covers literals no Go integer holds.
// Saturating them to MaxInt64 turns the query into a different one without a
// word, so either the parser or the evaluator has to refuse it — and say which
// literal it refused. That is how this test tells the rejection apart from an
// unrelated error, with one exception: ANTLR's own syntax errors also quote the
// offending token, so a message in ANTLR's wording never counts as a rejection.
//
// The cardinality case overflows the MINIMUM: an overflowing maximum with a
// minimum of 0 is satisfied by every concept whatever the maximum becomes, so
// it would not show a saturation at all.
func TestSemantics_NumericOverflowIsRejected(t *testing.T) {
	require64BitInt(t)
	p := semanticsProvider()
	for _, tc := range []struct {
		expr, literal, defect string
		now                   []string // the result while the literal is accepted; none when empty
	}{
		{expr: `* : ` + semStrength + ` = #99999999999999999999999999999`, literal: "99999999999999999999999999999", defect: "H4", now: []string{semOther}},
		{expr: `* : [99999999999999999999..*] ` + semStrength + ` = *`, literal: "99999999999999999999", defect: "H4", now: none},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			set, err := evaluate(p, tc.expr)
			var got []string
			if err == nil {
				got = set.Slice()
			}
			rejected := err != nil && strings.Contains(err.Error(), tc.literal) && !isGrammarError(err)
			if tc.defect != "" && tc.now == nil {
				t.Fatalf("%q is marked %s but pins no current result: set now (none for the empty set)", tc.expr, tc.defect)
			}
			switch {
			case tc.defect == "":
				require.Truef(t, rejected, "%q must be rejected with an error naming %s; got %v", tc.expr, tc.literal, err)
			case rejected:
				t.Fatalf("%q is now rejected (%v), so defect %s looks fixed: drop its marker and its now", tc.expr, err, tc.defect)
			case err != nil:
				t.Fatalf("%q fails for a reason that does not name the literal %s: %v", tc.expr, tc.literal, err)
			case !equalIDs(got, tc.now):
				t.Fatalf("%q is still accepted but now returns %v instead of the pinned %v: the overflow is handled differently", tc.expr, got, tc.now)
			default:
				t.Logf("KNOWN DEFECT %s: %q is accepted instead of rejected", tc.defect, tc.expr)
			}
		})
	}
}

// isGrammarError reports whether err is the generated parser's own complaint
// rather than a deliberate rejection; these are ANTLR's message forms.
func isGrammarError(err error) bool {
	for _, form := range []string{"no viable alternative", "extraneous input", "mismatched input", "missing '", "missing {", "token recognition error"} {
		if strings.Contains(err.Error(), form) {
			return true
		}
	}
	return false
}

// none pins an empty current result explicitly; see semCase.now.
var none = []string{}

// require64BitInt skips a numeric test where int is narrower than 64 bits. The
// AST stores integer literals in int, so on a 32-bit platform every literal
// here saturates at MaxInt32 instead — H4 at a different boundary — and the
// expected and pinned results are the 64-bit ones.
func require64BitInt(t *testing.T) {
	t.Helper()
	if strconv.IntSize < 64 {
		t.Skipf("int is %d bits here; these cases are written for 64-bit int", strconv.IntSize)
	}
}

func runSemCases(t *testing.T, cases []semCase) {
	t.Helper()
	p := semanticsProvider()
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			got := evalIDs(t, p, tc.expr)
			want, now := tc.want, tc.now
			switch {
			case tc.defect == "":
				require.Nilf(t, tc.now, "%q has no defect marker, so its pinned wrong result is stale: drop now", tc.expr)
			case tc.now == nil:
				t.Fatalf("%q is marked %s but pins no current result: set now (none for the empty set)", tc.expr, tc.defect)
			}
			switch {
			case tc.defect == "":
				require.Truef(t, equalIDs(want, got), "%q returns %v; the specification requires %v", tc.expr, got, want)
			case equalIDs(want, got):
				t.Fatalf("%q now returns the specified %v, so defect %s looks fixed: drop its marker and its now", tc.expr, want, tc.defect)
			case !equalIDs(now, got):
				t.Fatalf("%q returns %v, neither the specified %v nor the pinned wrong result %v of defect %s: something else changed",
					tc.expr, got, want, now, tc.defect)
			default:
				t.Logf("KNOWN DEFECT %s: %q returns %v; the specification requires %v", tc.defect, tc.expr, got, want)
			}
		})
	}
}

// equalIDs compares two ID lists ignoring order, with nil and empty alike. It
// does not deduplicate; the lists come from a Set or are written by hand.
func equalIDs(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}
