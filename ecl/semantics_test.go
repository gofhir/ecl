package ecl_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
	"github.com/gofhir/ecl/ecl/providertest"
)

// Semantic regressions with the result the SPECIFICATION requires, not the one
// the evaluator happens to return. They come from the adversarial review in
// docs/analisis-adversarial-2026-10-12.md, whose findings the defect markers
// name (H2b is recorded in its addendum, section 11.2):
//
//	H1   D-prefixed active/moduleId/effectiveTime filters read the concept.
//	     Fixed: every filter block carries its family (ast.Filtered.Family).
//	     DescriptionFilterOpts still cannot carry several effectiveTime values
//	     or a negated moduleId; those forms are reported unsupported.
//	H2   repeated clauses inside ONE block are not a conjunction. Fixed for
//	     type, language and moduleId, whose values are intersected; two term
//	     clauses are reported unsupported, since one call carries one term.
//	H2b  separate blocks were flattened into one predicate: the last term
//	     overrode the earlier ones, languages pooled into an any-of, and a term
//	     and a language from different blocks had to hold on one description.
//	     Fixed: one Filtered per block.
//	H2m  the same flattening across `{{ M … }}` blocks, which still share one
//	     Filtered over their MemberOf: clauses of different M blocks must hold
//	     on one member row.
//	H3   term and dialect in one block may be satisfied by different
//	     descriptions.
//	H4   numeric literals lost precision (float64) or saturated on overflow.
//	     Fixed: comparisons are exact (math/big on the literal as written), and
//	     an integer or cardinality wider than int is a parse error.
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
	// unsupported pins, instead of now, that the evaluator reports the case
	// ErrUnsupportedFeature: it fails loudly rather than answering, which is
	// the policy while the contract cannot express the query, but it is still
	// not the specified result.
	unsupported bool
}

func TestSemantics_DescriptionFilters(t *testing.T) {
	both := []string{semHeart, semOther}
	heart := []string{semHeart}
	runSemCases(t, []semCase{
		// H1 — the D prefix scopes these to descriptions.
		{expr: `* {{ D active = false }}`, want: heart},
		{expr: `* {{ D moduleId = ` + semModule + ` }}`, want: heart},
		{expr: `* {{ D effectiveTime = "` + semNewTime + `" }}`, want: heart},
		// No description carries the concept's own time; today the concept's is read.
		{expr: `* {{ D effectiveTime = "` + semOldTime + `" }}`, want: nil},
		// No prefix is a description filter: the grammar makes D optional and
		// C mandatory, so these read the description, like their D forms.
		{expr: `* {{ active = false }}`, want: heart},
		{expr: `* {{ effectiveTime = "` + semNewTime + `" }}`, want: heart},
		// A positive effectiveTime set is any-of, answered one call per value:
		// only "corazon" carries either time.
		{expr: `* {{ D effectiveTime = ("` + semOldTime + `" "` + semNewTime + `") }}`, want: heart},
		// `active = *` admits either value, but a description filter still asks
		// for a description, so semStrength, which has none, is out; as a
		// concept filter it constrains nothing. `active != *` admits neither.
		{expr: `* {{ active = * }}`, want: both},
		{expr: `* {{ C active = * }}`, want: []string{semHeart, semOther, semStrength}},
		{expr: `* {{ active != * }}`, want: nil},
		{expr: `* {{ C active != * }}`, want: nil},
		// An ordered effectiveTime set is any-of too: only "corazon" carries a
		// time after the first, and none is after the second.
		{expr: `* {{ D effectiveTime > ("19000101" "30000101") }}`, want: heart},
		// Excluding a type set that names no type excludes nothing: both
		// concepts with descriptions remain.
		{expr: `* {{ D typeId != (< ` + semGB + `) }}`, want: both},
		// A positive term set beside a negated clause: every term of the set
		// has to reach the negated call. Only "corazon" is non-English.
		{expr: `* {{ D term = ("heart" "corazon"), language != en }}`, want: heart},
		// The concept-scoped forms, which must not move when H1 is fixed.
		{expr: `* {{ C active = false }}`, want: nil},
		{expr: `* {{ C moduleId = ` + semModule + ` }}`, want: nil},
		{expr: `* {{ C effectiveTime = "` + semOldTime + `" }}`, want: heart},

		// H2 — one block, one description: two values of one column cannot both hold.
		{expr: `* {{ D language = en, language = es }}`, want: nil},
		{expr: `* {{ D term = "nonexistent", term = "heart" }}`, want: nil, defect: "H2", unsupported: true},
		{expr: `* {{ D term = "heart", term = "corazon" }}`, want: nil, defect: "H2", unsupported: true},
		// A value list inside ONE clause is any-of, unlike repeated clauses. The
		// list is space-separated (languageCodeSet); `(en OR es)` would parse
		// "OR" as a third two-letter code. No description is French, so reading
		// only the first value fails the second case, only the last fails the
		// first, and dropping the clause returns all of `*`.
		{expr: `* {{ D language = (es fr) }}`, want: heart},
		{expr: `* {{ D language = (fr es) }}`, want: heart},

		// H2b — separate blocks, each satisfied on its own.
		{expr: `* {{ D term = "corazon" }} {{ D term = "heart" }}`, want: heart},
		{expr: `* {{ D term = "nonexistent" }} {{ D term = "corazon" }}`, want: nil},
		{expr: `* {{ D term = "heart" }} {{ D term = "corazon" }}`, want: heart},
		{expr: `* {{ D term = "corazon" }} {{ D term = "nonexistent" }}`, want: nil},
		// Term and language from different blocks may hold on different
		// descriptions; the second case is the shape of official example 8.2.2.
		{expr: `* {{ D term = "heart" }} {{ D language = es }}`, want: heart},
		{expr: `* {{ D term = "corazon", language = es }} {{ D term = "heart", language = en }}`, want: heart},
		// semOther has no Spanish description; the blocks' languages are pooled into an any-of.
		{expr: `* {{ D language = es }} {{ D language = en }}`, want: heart},

		// H3 — term and dialect on the SAME description: semHeart's "heart" is en-US only.
		{expr: `* {{ D term = "heart", dialectId = ` + semGB + ` }}`, want: []string{semOther}, defect: "H3", now: both},
		// The metadata clauses share H3: active and dialect are answered by two
		// calls, so semHeart's inactive "old" (en-GB) and active "heart" (en-US)
		// together pass a block that no single description satisfies.
		{expr: `* {{ D active = false, dialectId = ` + semUS + ` }}`, want: nil, defect: "H3", now: heart},
		// Across blocks semHeart qualifies through "corazon"; a fix for H3 that
		// correlated blocks too would wrongly drop it.
		{expr: `* {{ D term = "heart" }} {{ D dialectId = ` + semGB + ` }}`, want: both},
		{expr: `* {{ D dialectId = ` + semGB + ` }} {{ D term = "heart" }}`, want: both},
		{expr: `* {{ D term = "corazon" }} {{ D dialectId = ` + semUS + ` }}`, want: heart},
	})
}

// TestSemantics_MemberFilterBlocks covers H2m on standard.yaml, whose complex
// map gives 95566004 one row in map group 2 and ANOTHER row mapping to I10.
func TestSemantics_MemberFilterBlocks(t *testing.T) {
	runSemCasesOn(t, standardProvider(t), []semCase{
		// One block: one row must satisfy both clauses, and none does.
		{expr: `^ 447562003 {{ M mapGroup = "2", mapTarget = "I10" }}`, want: nil},
		// Two blocks: each may be satisfied by its own row.
		{expr: `^ 447562003 {{ M mapGroup = "2" }} {{ M mapTarget = "I10" }}`, want: []string{"95566004"}, defect: "H2m", now: none},
		// The member filter belongs to `^ R`; the hierarchy operator applies to
		// the members it leaves. 95566004 has no descendants in the fixture.
		{expr: `<< ^ 447562003 {{ M mapGroup = "2" }}`, want: []string{"95566004"}},
	})
}

// TestSemantics_InactiveRowsDoNotQualifyAMember covers a member with an
// active row and an inactive one. `^ R` means the active rows, so a field
// filter must be satisfied by an ACTIVE row; intersecting with the active
// members let the inactive row's value qualify the member through its sibling.
// The fixture declares both capabilities the per-row default needs.
func TestSemantics_InactiveRowsDoNotQualifyAMember(t *testing.T) {
	const refset, member = "447562003", "95566004"
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts: []providertest.FixtureConcept{{ID: refset}, {ID: member}},
		// Declared under refsets too, so `^ R` alone finds the member and the
		// "X" case can only be told apart by which ROW qualifies it.
		Refsets: map[string][]string{refset: {member}},
		MemberFields: []providertest.FixtureMemberField{
			{Refset: refset, Member: member, Row: "1", FieldName: "mapTarget", Value: "Y"},
			{Refset: refset, Member: member, Row: "2", FieldName: "mapTarget", Value: "X"},
			{Refset: refset, Member: member, Row: "2", FieldName: "active", Value: "0"},
			// A declared empty value is a value, not a missing column.
			{Refset: refset, Member: member, Row: "1", FieldName: "mapAdvice", Value: ""},
		},
	})
	runSemCasesOn(t, p, []semCase{
		{expr: `^ ` + refset + ` {{ M mapTarget = "Y" }}`, want: []string{member}},
		{expr: `^ ` + refset + ` {{ M mapTarget = "X" }}`, want: nil},
		// Asking for the inactive rows explicitly reaches the X row.
		{expr: `^ ` + refset + ` {{ M mapTarget = "X", active = false }}`, want: []string{member}},
		{expr: `^ ` + refset + ` {{ M mapAdvice = "" }}`, want: []string{member}},
	})
}

// TestSemantics_ProjectionKeepsTheActiveRowDefault covers `^ [field] R`, with
// and without a member filter. A source with two associations keeps both
// targets, and the inactive one is left out either way: a filter matching
// every row must not change what the projection returns.
func TestSemantics_ProjectionKeepsTheActiveRowDefault(t *testing.T) {
	const sameAs, source, kept, retired = "900000000000527005", "11111111", "22298006", "73211009"
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts: []providertest.FixtureConcept{{ID: sameAs}, {ID: source}, {ID: kept}, {ID: retired}},
		HistoricalAssociations: []providertest.FixtureHistory{
			{Refset: sameAs, Source: source, Target: kept},
			{Refset: sameAs, Source: source, Target: retired},
		},
		// Row names the association it belongs to: its target.
		MemberFields: []providertest.FixtureMemberField{
			{Refset: sameAs, Member: source, Row: retired, FieldName: "active", Value: "0"},
		},
	})
	runSemCasesOn(t, p, []semCase{
		{expr: `^ [targetComponentId] ` + sameAs, want: []string{kept}},
		{expr: `^ [targetComponentId] ` + sameAs + ` {{ M referencedComponentId = ` + source + ` }}`, want: []string{kept}},
		{expr: `^ [targetComponentId] ` + sameAs + ` {{ M active = * }}`, want: []string{kept, retired}},
		{expr: `^ ` + sameAs + ` {{ M targetComponentId = ` + kept + ` }}`, want: []string{source}},
	})
}

// TestSemantics_InactiveAssociationIsNotHistory covers an association row
// marked inactive by a memberFields entry with no Row. It belongs to the
// association's row, so the association is neither a current member nor part
// of a history supplement.
func TestSemantics_InactiveAssociationIsNotHistory(t *testing.T) {
	const sameAs, retiredSource, liveSource, target = "900000000000527005", "11111111", "33333333", "22298006"
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts: []providertest.FixtureConcept{{ID: sameAs}, {ID: target}, {ID: retiredSource}, {ID: liveSource}},
		HistoricalAssociations: []providertest.FixtureHistory{
			{Refset: sameAs, Source: retiredSource, Target: target},
			{Refset: sameAs, Source: liveSource, Target: target},
		},
		MemberFields: []providertest.FixtureMemberField{
			{Refset: sameAs, Member: retiredSource, FieldName: "active", Value: "0"},
		},
	})
	runSemCasesOn(t, p, []semCase{
		{expr: `^ ` + sameAs, want: []string{liveSource}},
		{expr: target + ` {{ +HISTORY }}`, want: []string{target, liveSource}},
	})
}

// columnsWithoutProjector declares the member row columns but cannot take
// several clauses on one row.
type columnsWithoutProjector struct{ ecl.DataProvider }

func (columnsWithoutProjector) AnswersMemberRowColumns() {}

// TestSemantics_ContradictionIsAnsweredWithoutTheProvider covers `active != *`,
// which no row satisfies. Its two clauses are not sent, so a provider that
// cannot take two clauses on one row still answers the empty set rather than
// refusing an expression whose answer is known.
func TestSemantics_ContradictionIsAnsweredWithoutTheProvider(t *testing.T) {
	p := columnsWithoutProjector{standardProvider(t)}
	runSemCasesOn(t, p, []semCase{
		{expr: `^ 900000000000497000 {{ M active != * }}`, want: nil},
	})
}

// activeMembersOnly answers RefsetMembers the way an RF2-faithful provider
// does — only members with an active row — and declares the member row
// columns, which its embedded fixture answers.
type activeMembersOnly struct {
	ecl.DataProvider
	active []string
}

func (a activeMembersOnly) RefsetMembers(context.Context, []string) (ecl.Set, error) {
	return ecl.NewSetFromSlice(a.active), nil
}

func (activeMembersOnly) AnswersMemberRowColumns() {}

// TestSemantics_MemberActiveFilterDecidesTheRows covers `{{ M active = false }}`
// against such a provider. `^ R` alone is the active members, so intersecting
// the filter's rows with it left only members that also had an active row.
func TestSemantics_MemberActiveFilterDecidesTheRows(t *testing.T) {
	const refset, retired, current = "447562003", "22298006", "73211009"
	base := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts:     []providertest.FixtureConcept{{ID: refset}, {ID: retired}, {ID: current}},
		Refsets:      map[string][]string{refset: {retired, current}},
		MemberFields: []providertest.FixtureMemberField{{Refset: refset, Member: retired, FieldName: "active", Value: "0"}},
	})
	p := activeMembersOnly{DataProvider: base, active: []string{current}}
	runSemCasesOn(t, p, []semCase{
		{expr: `^ ` + refset, want: []string{current}},
		{expr: `^ ` + refset + ` {{ M active = false }}`, want: []string{retired}},
		{expr: `^ ` + refset + ` {{ M active = true }}`, want: []string{current}},
		// Either flag lets the inactive row count, which `^` alone does not.
		{expr: `^ ` + refset + ` {{ M active = * }}`, want: []string{retired, current}},
		// Parentheses do not hide the memberOf from its member filter.
		{expr: `(^ ` + refset + `) {{ M active = false }}`, want: []string{retired}},
	})
}

// TestSemantics_MemberRowColumns covers active, moduleId and effectiveTime in
// a member filter: columns of the member ROW, sent as member field comparisons.
// They used to filter the referenced concept instead, which answered for a
// different component. The rows here differ from their concepts on purpose.
func TestSemantics_MemberRowColumns(t *testing.T) {
	const refset, retired, current = "447562003", "22298006", "73211009"
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts: []providertest.FixtureConcept{{ID: refset}, {ID: retired}, {ID: current}},
		Refsets:  map[string][]string{refset: {retired, current}},
		MemberFields: []providertest.FixtureMemberField{
			// Both concepts are active; only the member rows differ.
			{Refset: refset, Member: retired, FieldName: "active", Value: "0"},
			{Refset: refset, Member: current, FieldName: "moduleId", Value: semModule},
			{Refset: refset, Member: current, FieldName: "effectiveTime", Value: semNewTime},
		},
	})
	runSemCasesOn(t, p, []semCase{
		// A row that declares no active column is active.
		{expr: `^ ` + refset + ` {{ M active = true }}`, want: []string{current}},
		{expr: `^ ` + refset + ` {{ M active = false }}`, want: []string{retired}},
		{expr: `^ ` + refset + ` {{ M moduleId = ` + semModule + ` }}`, want: []string{current}},
		{expr: `^ ` + refset + ` {{ M effectiveTime = "` + semNewTime + `" }}`, want: []string{current}},
		// A module set naming no module: requiring one matches no row, and
		// excluding one excludes nothing, which leaves `^ R` alone: the members
		// with an active row, so not retired.
		{expr: `^ ` + refset + ` {{ M moduleId = (< ` + semGB + `) }}`, want: nil},
		{expr: `^ ` + refset + ` {{ M moduleId != (< ` + semGB + `) }}`, want: []string{current}},
		// `active != *` matches no row.
		{expr: `^ ` + refset + ` {{ M active != * }}`, want: nil},
		// Either flag: the rows decide, so the inactive row counts too.
		{expr: `^ ` + refset + ` {{ M active = * }}`, want: []string{retired, current}},
		// The concept-scoped reading filters the concepts of `^ R`'s active
		// members. Both concepts are active, but retired is not such a member.
		{expr: `^ ` + refset + ` {{ C active = true }}`, want: []string{current}},
		{expr: `^ ` + refset, want: []string{current}},
	})
}

// TestSemantics_DroppedClauseKeepsSiblingPolarity covers a clause that
// constrains nothing — excluding a type set that names no type — being dropped
// before the polarity of the block is decided. Left in the list, it sent the
// block down the negated path and its "!=" was read onto the kept positive
// clause: `typeId = FSN, typeId != (nothing)` asked for descriptions that are
// NOT FSNs.
func TestSemantics_DroppedClauseKeepsSiblingPolarity(t *testing.T) {
	const fsnConcept, synConcept = "22298006", "73211009"
	const fsn, syn = "900000000000003001", "900000000000013009"
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts: []providertest.FixtureConcept{{ID: fsnConcept}, {ID: synConcept}},
		Descriptions: []providertest.FixtureDescription{
			{Concept: fsnConcept, Language: "en", TypeID: fsn, Value: "heart (x)"},
			{Concept: synConcept, Language: "en", TypeID: syn, Value: "heart"},
		},
	})
	runSemCasesOn(t, p, []semCase{
		{expr: `* {{ D typeId = ` + fsn + `, typeId != (< ` + semGB + `) }}`, want: []string{fsnConcept}},
		{expr: `* {{ D typeId != (< ` + semGB + `), typeId = ` + fsn + ` }}`, want: []string{fsnConcept}},
	})
}

// zeroOptsAnswerNothing answers MatchDescription with zero-value Opts the way
// the contract says — the empty Set — where the in-memory fixture is lenient.
type zeroOptsAnswerNothing struct{ ecl.DataProvider }

// AnswersDescriptionMetadata declares that it honors Active, which the
// embedded fixture does.
func (zeroOptsAnswerNothing) AnswersDescriptionMetadata() {}

func (z zeroOptsAnswerNothing) MatchDescription(ctx context.Context, opts ecl.DescriptionFilterOpts) (ecl.Set, error) {
	if reflect.ValueOf(opts).IsZero() {
		return ecl.NewSet(), nil
	}
	return z.DataProvider.MatchDescription(ctx, opts)
}

// TestSemantics_BlockOfDroppedClausesStillNeedsADescription covers a block
// whose every clause constrains nothing. It still asks for a description, and
// must ask in a form the contract answers: zero-value Opts come back empty.
func TestSemantics_BlockOfDroppedClausesStillNeedsADescription(t *testing.T) {
	p := zeroOptsAnswerNothing{semanticsProvider()}
	runSemCasesOn(t, p, []semCase{
		// semStrength has no description, so it is not "a concept with a
		// description of a type outside the empty set".
		{expr: `* {{ D typeId != (< ` + semGB + `) }}`, want: []string{semHeart, semOther}},
	})
}

// TestSemantics_MemberRowColumnsNeedTheCapability covers a provider that does
// not declare ecl.MemberRowColumnsProvider. Written before these filters were
// sent, it would look "active" up as an additional field and answer the empty
// set; the evaluator must refuse instead.
func TestSemantics_MemberRowColumnsNeedTheCapability(t *testing.T) {
	// Embedding only the required interface hides every capability.
	p := struct{ ecl.DataProvider }{standardProvider(t)}
	for _, expr := range []string{
		`^ 900000000000497000 {{ M active = true }}`,
		`^ 900000000000497000 {{ M moduleId = 900000000000207008 }}`,
		`^ 900000000000497000 {{ M effectiveTime = "20240131" }}`,
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(p, expr)
			require.ErrorIsf(t, err, ecl.ErrUnsupportedFeature, "%q must be refused without the capability", expr)
		})
	}
}

// TestSemantics_DescriptionMetadataNeedsTheCapability covers a provider that
// does not declare ecl.DescriptionMetadataProvider: written before these
// fields were sent, it may ignore them and answer every description.
func TestSemantics_DescriptionMetadataNeedsTheCapability(t *testing.T) {
	p := struct{ ecl.DataProvider }{semanticsProvider()}
	for _, expr := range []string{
		`* {{ D active = false }}`,
		`* {{ active = false }}`, // no prefix: a description filter
		`* {{ D moduleId = ` + semModule + ` }}`,
		`* {{ D effectiveTime = "` + semNewTime + `" }}`,
		`* {{ D active = * }}`,
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(p, expr)
			require.ErrorIsf(t, err, ecl.ErrUnsupportedFeature, "%q must be refused without the capability", expr)
		})
	}
	// A module clause that names nothing constrains nothing, so it does not
	// need the capability either: the block is a plain term match.
	tree := mustParse(t, `* {{ D term = "heart", moduleId = `+semModule+` }}`)
	mf := tree.(*ast.Filtered).Filters[1].(*ast.ModuleFilter)
	mf.Modules, mf.Module = nil, nil //nolint:staticcheck // the deprecated field is the point
	_, err := ecl.Evaluate(context.Background(), tree, p)
	require.NoError(t, err)

	// The other description clauses and the concept forms need nothing new.
	for _, expr := range []string{`* {{ D term = "heart" }}`, `* {{ C active = false }}`} {
		_, err := evaluate(p, expr)
		require.NoErrorf(t, err, "%q", expr)
	}
}

// TestSemantics_DeprecatedModuleFieldStillFilters covers a hand-built module
// filter that sets only the deprecated Module field, as a v1 reader's tree
// may. It must filter like the Modules form, not lose its constraint.
func TestSemantics_DeprecatedModuleFieldStillFilters(t *testing.T) {
	p := semanticsProvider()
	for _, expr := range []string{`* {{ D moduleId = ` + semModule + ` }}`, `* {{ C moduleId = ` + semModule + ` }}`} {
		t.Run(expr, func(t *testing.T) {
			want := evalIDs(t, p, expr)
			tree := mustParse(t, expr)
			mf := tree.(*ast.Filtered).Filters[0].(*ast.ModuleFilter)
			mf.Module, mf.Modules = mf.Modules[0], nil //nolint:staticcheck // the deprecated field is the point
			got, err := ecl.Evaluate(context.Background(), tree, p)
			require.NoError(t, err)
			require.Equal(t, want, got.Slice())
		})
	}
}

// TestSemantics_ModuleFilterNamingNothingConstrainsNothing covers a hand-built
// module filter with no module at all, in each family: it constrains nothing,
// so "!=" must not subtract every concept, as the concept family did.
func TestSemantics_ModuleFilterNamingNothingConstrainsNothing(t *testing.T) {
	p := semanticsProvider()
	for _, tc := range []struct{ expr, unfiltered string }{
		{`* {{ C moduleId != ` + semModule + ` }}`, `*`},
		{`* {{ D moduleId = ` + semModule + ` }}`, `* {{ D active = * }}`},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			tree := mustParse(t, tc.expr)
			// Both fields: the parser fills the deprecated one too, and it is
			// a module the filter still names.
			mf := tree.(*ast.Filtered).Filters[0].(*ast.ModuleFilter)
			mf.Modules, mf.Module = nil, nil //nolint:staticcheck // the deprecated field is the point
			got, err := ecl.Evaluate(context.Background(), tree, p)
			require.NoError(t, err)
			require.Equal(t, evalIDs(t, p, tc.unfiltered), got.Slice())
		})
	}
}

// TestSemantics_UnrepresentableFiltersFailLoudly covers the filter forms the
// provider contract cannot carry yet. Answering them from a partial reading is
// how H1 and H2 came about, so each must be reported ErrUnsupportedFeature
// rather than evaluated wider than written.
func TestSemantics_UnrepresentableFiltersFailLoudly(t *testing.T) {
	p := semanticsProvider()
	for _, expr := range []string{
		// DescriptionFilterOpts has no negated module and one effectiveTime.
		`* {{ D moduleId != ` + semModule + ` }}`,
		// A negated set needs one row to fail every value.
		`* {{ D effectiveTime != ("` + semOldTime + `" "` + semNewTime + `") }}`,
		`* {{ D effectiveTime >= "` + semOldTime + `", effectiveTime <= "` + semNewTime + `" }}`,
		// A member filter with no `^`: the grammar admits it, but there are no
		// member rows to filter.
		`< 447562003 {{ M mapGroup = "2" }}`,
		// A row filter that can match nothing still lets its operand report
		// what it cannot do — here a projection of every field.
		`^ [*] 447562003 {{ M active != * }}`,
		// A contradictory active pair does not excuse a later clause that
		// cannot be carried, whatever the clause order.
		`^ 900000000000497000 {{ M active != *, effectiveTime < "` + semOldTime + `" }}`,
		`^ 900000000000497000 {{ M effectiveTime < "` + semOldTime + `", active != * }}`,
		// A module clause matching no row does not excuse a later one either.
		`^ 900000000000497000 {{ M moduleId = (< ` + semGB + `), effectiveTime < "` + semOldTime + `" }}`,
		`^ 900000000000497000 {{ M effectiveTime < "` + semOldTime + `", moduleId = (< ` + semGB + `) }}`,
		// Two value sets in one block multiply into provider calls (a term set
		// alone is one call per term, however long): 17 terms
		// by 16 times is above the limit.
		`* {{ D term = ("a" "b" "c" "d" "e" "f" "g" "h" "i" "j" "k" "l" "m" "n" "o" "p" "q"), effectiveTime = ("20010101" "20020101" "20030101" "20040101" "20050101" "20060101" "20070101" "20080101" "20090101" "20100101" "20110101" "20120101" "20130101" "20140101" "20150101" "20160101") }}`,
		// MemberFilterOpts compares member columns by equality only.
		`^ 900000000000497000 {{ M effectiveTime >= "` + semOldTime + `" }}`,
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := evaluate(p, expr)
			require.ErrorIsf(t, err, ecl.ErrUnsupportedFeature, "%q must fail loudly", expr)
		})
	}
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
		{expr: attr + `= #9007199254740993`, want: nil},
		{expr: attr + `< #9007199254740993`, want: heart},
		{expr: attr + `= #9223372036854775807`, want: other},
		{expr: attr + `= #9223372036854775806`, want: nil},
	})
}

// TestSemantics_NumericLiteralsKeepTheirText covers two edges of the exact
// comparison. A literal compared as TEXT, as a member field is, must travel as
// written — "1.5", not the fraction "3/2". And a stored value is read as a
// decimal number only: a fraction or a hexadecimal string, which an exact
// rational parser would otherwise accept, is not one.
func TestSemantics_NumericLiteralsKeepTheirText(t *testing.T) {
	const refset, member, fraction, hex = "447562003", "95566004", "22298006", "73211009"
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts: []providertest.FixtureConcept{{ID: refset}, {ID: member}, {ID: fraction}, {ID: hex}, {ID: semStrength}},
		Refsets:  map[string][]string{refset: {member}},
		MemberFields: []providertest.FixtureMemberField{
			{Refset: refset, Member: member, FieldName: "mapPriority", Value: "1.5"},
			{Refset: refset, Member: member, FieldName: "mapGroup", Value: "1"},
			{Refset: refset, Member: member, FieldName: "mapRule", Value: "0"},
		},
		Relationships: []providertest.FixtureRelationship{
			{Source: fraction, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "decimal", Value: "1/2"}},
			{Source: hex, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "integer", Value: "0x10"}},
			// An exponent is a decimal form RF2 can write; one beyond the bound is
			// compared through its float64 (+Inf here) rather than dropped.
			{Source: member, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "decimal", Value: "5e2"}},
			{Source: refset, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "decimal", Value: "1e100000"}},
			// Tiny, but a number: it used to round to about 0 and must not drop out.
			{Source: fraction, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "decimal", Value: "1e-401"}},
			// More digits than float64 tells apart from 0.1.
			{Source: hex, TypeID: semStrength, Concrete: &providertest.FixtureConcrete{Kind: "decimal", Value: "0.1"}},
		},
	})
	runSemCasesOn(t, p, []semCase{
		// Compared as text in its canonical form, whatever was typed.
		{expr: `^ ` + refset + ` {{ M mapPriority = #1.5 }}`, want: []string{member}},
		{expr: `^ ` + refset + ` {{ M mapPriority = #1.50 }}`, want: []string{member}},
		{expr: `^ ` + refset + ` {{ M mapGroup = #+1 }}`, want: []string{member}},
		{expr: `^ ` + refset + ` {{ M mapGroup = #1.0 }}`, want: []string{member}},
		{expr: `^ ` + refset + ` {{ M mapRule = #-0 }}`, want: []string{member}},
		{expr: `* : ` + semStrength + ` = #0.5`, want: nil},
		{expr: `* : ` + semStrength + ` = #16`, want: nil},
		{expr: `* : ` + semStrength + ` = #500`, want: []string{member}},
		{expr: `* : ` + semStrength + ` > #1000`, want: []string{refset}},
		// 1e-401 is above zero and below one; 0.1 is below 0.10000000000000001
		// although both round to the same float64.
		{expr: `* : ` + semStrength + ` > #0`, want: []string{member, fraction, hex, refset}},
		{expr: `* : ` + semStrength + ` < #0.000001`, want: []string{fraction}},
		{expr: `* : ` + semStrength + ` = #0.10000000000000001`, want: nil},
		{expr: `* : ` + semStrength + ` < #0.10000000000000001`, want: []string{fraction, hex}},
	})

	// A hand-built member literal that is not a finite number is refused as
	// unsupported, not as a generic error.
	tree := mustParse(t, `^ `+refset+` {{ M mapPriority = #1.5 }}`)
	tree.(*ast.Filtered).Filters[0].(*ast.MemberFieldFilter).Value = &ast.DecimalValue{Value: math.NaN()}
	_, err := ecl.Evaluate(context.Background(), tree, p)
	require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)

	// Text counts only while it still says Value. A parsed node whose Value is
	// edited compares against the new Value: 501 is not the stored 5e2.
	tree = mustParse(t, `* : `+semStrength+` = #500`)
	require.Equal(t, []string{member}, evalIDs(t, p, `* : `+semStrength+` = #500`))
	tree.(*ast.Refined).Refinement.AttrSet.Attr.Value.(*ast.IntegerValue).Value = 501
	got, err := ecl.Evaluate(context.Background(), tree, p)
	require.NoError(t, err)
	require.Empty(t, got.Slice(), "the edited Value, not the stale Text, decides")

	// A hand-built node may set only Text; Value is then the zero value, and
	// Text, the exact literal, is what it means.
	tree = mustParse(t, `* : `+semStrength+` = #500`)
	tree.(*ast.Refined).Refinement.AttrSet.Attr.Value = &ast.IntegerValue{Text: "500"}
	got, err = ecl.Evaluate(context.Background(), tree, p)
	require.NoError(t, err)
	require.Equal(t, []string{member}, got.Slice())

	// The limit is in digits: 4000 of them with a "." still parse.
	_, err = ecl.Parse(`* : ` + semStrength + ` < #0.` + strings.Repeat("1", 3999))
	require.NoError(t, err)

	// A literal of more digits than any concrete value has is refused at parse
	// time rather than built as an exact rational compared once per stored value.
	_, err = ecl.Parse(`* : ` + semStrength + ` < #0.` + strings.Repeat("0", 5000) + `1`)
	var pe *ecl.ParseError
	require.ErrorAs(t, err, &pe)
	require.ErrorContains(t, err, "limit")
}

// TestSemantics_LongTermSetIsStillOneCallPerTerm covers a term set beyond the
// fan-out limit: the limit is on the PRODUCT of two sets, and a term set alone
// was always answered one call per term.
func TestSemantics_LongTermSetIsStillOneCallPerTerm(t *testing.T) {
	terms := make([]string, 300)
	for i := range terms {
		terms[i] = fmt.Sprintf(`"t%d"`, i)
	}
	terms[0] = `"heart"`
	runSemCases(t, []semCase{
		{expr: `* {{ D term = (` + strings.Join(terms, " ") + `) }}`, want: []string{semHeart, semOther}},
	})
}

// TestSemantics_LongEffectiveTimeSetIsStillOneCallPerTime covers an
// effectiveTime set beyond the fan-out limit with no term set beside it: the
// limit is on the product of the two, so one set alone is not refused.
func TestSemantics_LongEffectiveTimeSetIsStillOneCallPerTime(t *testing.T) {
	times := make([]string, 300)
	for i := range times {
		times[i] = fmt.Sprintf(`"%d0101"`, 1700+i)
	}
	times[0] = `"` + semNewTime + `"`
	runSemCases(t, []semCase{
		{expr: `* {{ D effectiveTime = (` + strings.Join(times, " ") + `) }}`, want: []string{semHeart}},
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
		{expr: `* : ` + semStrength + ` = #99999999999999999999999999999`, literal: "99999999999999999999999999999"},
		{expr: `* : [99999999999999999999..*] ` + semStrength + ` = *`, literal: "99999999999999999999"},
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
	runSemCasesOn(t, semanticsProvider(), cases)
}

func runSemCasesOn(t *testing.T, p ecl.DataProvider, cases []semCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			if tc.unsupported {
				require.NotEmptyf(t, tc.defect, "%q pins unsupported without a defect marker", tc.expr)
				set, err := evaluate(p, tc.expr)
				switch {
				case errors.Is(err, ecl.ErrUnsupportedFeature):
					t.Logf("KNOWN DEFECT %s: %q is reported unsupported; the specification requires %v", tc.defect, tc.expr, tc.want)
				case err != nil:
					t.Fatalf("%q fails with %v instead of the pinned ErrUnsupportedFeature", tc.expr, err)
				case equalIDs(tc.want, set.Slice()):
					t.Fatalf("%q now returns the specified %v, so defect %s looks fixed: drop its marker and unsupported", tc.expr, tc.want, tc.defect)
				default:
					t.Fatalf("%q returns %v, neither the specified %v nor the pinned ErrUnsupportedFeature: something else changed", tc.expr, set.Slice(), tc.want)
				}
				return
			}
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
