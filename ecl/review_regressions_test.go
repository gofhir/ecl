package ecl_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
)

// Regression tests for defects found reviewing this branch. Expected IDs are
// computed against ecl/providertest/testdata/fixtures/standard.yaml.

// TestEvaluate_ParenthesisedRefinementThenDisjunction covers a valid shape that
// the scope guard rejected.
//
// Keeping a parenthesised sub-refinement in Refinement.Conjunction conflated
// "the first sub-refinement was parenthesised" with "there is a conjunction set",
// so `(<refinement>) OR <clause>` looked like a node holding both a conjunction
// and a disjunction and errored out. It has its own field now.
func TestEvaluate_ParenthesisedRefinementThenDisjunction(t *testing.T) {
	set := evalFixture(t, "* : ({ 363698007 = 74281007 }) OR 1142139005 = #5")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, set.Slice())

	// The scope still has to survive: here the trailing conjunct must apply to
	// the whole disjunction.
	scoped := evalFixture(t, "* : ({ 363698007 = 74281007 } OR { 363698007 = 113331007 }) , 1142139005 = #5")
	require.ElementsMatch(t, []string{"73211009"}, scoped.Slice())
}

// TestEvaluate_FilterOperandMatchingNothing covers the difference between "no
// operand" and "an operand that names nothing".
//
// An empty field in the Opts means "do not filter on this dimension", so
// returning no IDs for an operand that legitimately matched nothing turned the
// clause into a no-op and the filter matched EVERYTHING.
func TestEvaluate_FilterOperandMatchingNothing(t *testing.T) {
	// 900000000000073002 has no descendants, so the operand names no concept.
	none := evalFixture(t, "<< 404684003 {{ C definitionStatusId = (< 900000000000073002) }}")
	require.Empty(t, none.Slice(), "a clause naming no concept matched everything")

	// Negated, the same clause removes nothing.
	all := evalFixture(t, "<< 404684003 {{ C definitionStatusId != (< 900000000000073002) }}")
	base := evalFixture(t, "<< 404684003")
	require.ElementsMatch(t, base.Slice(), all.Slice())

	// The positive form still works when the operand does name something.
	some := evalFixture(t, "<< 404684003 {{ C definitionStatusId = 900000000000073002 }}")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, some.Slice())
}

// TestEvaluate_NilGroupInHandBuiltAST covers a nil group in the AST. Since
// ecl/ast is public a consumer can build one, and the guard against it had been
// dropped.
func TestEvaluate_NilGroupInHandBuiltAST(t *testing.T) {
	expr := &ast.Refined{
		Focus:      &ast.Any{},
		Refinement: &ast.Refinement{Groups: []*ast.AttributeGroup{nil}},
	}
	set, err := ecl.Evaluate(context.Background(), expr, standardProvider(t))
	require.NoError(t, err)
	require.NotNil(t, set)
}

// TestEvaluate_NilSetFromRelationshipSources covers the reverse-clause-in-group
// path, which the nil hardening missed: that Set never travels through Evaluate,
// so it was dereferenced directly.
func TestEvaluate_NilSetFromRelationshipSources(t *testing.T) {
	set, err := ecl.Evaluate(context.Background(),
		mustParse(t, "< 404684003 : { R 363698007 = * }"), nilProvider{})
	require.NoError(t, err)
	require.NotNil(t, set)
	require.Zero(t, set.Len())
}

// TestErrProvider covers the sentinel being reachable. It was exported and
// documented for errors.Is classification but never actually wrapped, so the
// documented check could not fire.
func TestErrProvider(t *testing.T) {
	_, err := ecl.Evaluate(context.Background(),
		mustParse(t, "< 404684003 : 363698007 = *"), failingProvider{})
	require.Error(t, err)
	require.ErrorIs(t, err, ecl.ErrProvider)

	// A malformed expression is NOT a provider error.
	_, err = ecl.Parse("11687002 GARBAGE")
	require.Error(t, err)
	require.NotErrorIs(t, err, ecl.ErrProvider)
}

// TestEvaluate_CancellationStopsPerConceptLoops covers cancellation inside the
// per-concept loops, which is where the thousands of provider calls happen. The
// check used to sit only at node entry, so a canceled request ran the whole loop.
func TestEvaluate_CancellationStopsPerConceptLoops(t *testing.T) {
	counting := &countingProvider{DataProvider: standardProvider(t)}
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel as soon as the loop asks for its first concept.
	counting.onPropertiesByGroup = func() { cancel() }

	_, err := ecl.Evaluate(ctx, mustParse(t, "< 138875005 : 363698007 = *"), counting)
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, counting.propertiesByGroup, 2,
		"the loop kept calling the provider after cancellation")
}

// failingProvider fails every hierarchy and attribute lookup, standing in for an
// unhealthy backend.
type failingProvider struct{ ecl.UnimplementedDataProvider }

var errBackendDown = errors.New("backend down")

func (failingProvider) Descendants(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSetFromSlice([]string{"22298006"}), nil
}

func (failingProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	return ecl.NewSetFromSlice(ids), nil
}

func (failingProvider) AllConcepts(context.Context) (ecl.Set, error) {
	return ecl.NewSetFromSlice([]string{"22298006"}), nil
}

func (failingProvider) PropertiesByGroup(context.Context, string) (map[int][]ecl.Relationship, error) {
	return nil, errBackendDown
}

// countingProvider counts PropertiesByGroup calls and can run a hook on each one.
type countingProvider struct {
	ecl.DataProvider
	propertiesByGroup   int
	onPropertiesByGroup func()
}

func (c *countingProvider) PropertiesByGroup(ctx context.Context, id string) (map[int][]ecl.Relationship, error) {
	c.propertiesByGroup++
	if c.onPropertiesByGroup != nil {
		c.onPropertiesByGroup()
	}
	return c.DataProvider.PropertiesByGroup(ctx, id)
}

// TestEvaluate_DialectOnlyFilterSkipsMatchDescription covers a constraint whose
// only description clause is a dialect.
//
// Dialect clauses are answered by MatchDialect, so calling MatchDescription with
// zero-value Opts asks "every description" — which, under the contract stating
// that empty input yields the empty Set, returns nothing, and which even with a
// lenient provider drops every concept that has no descriptions at all.
func TestEvaluate_DialectOnlyFilterSkipsMatchDescription(t *testing.T) {
	// 22298006 is preferred in en-gb, 73211009 only exists in en-us.
	gb := evalFixture(t, "<< 138875005 {{ D dialectId = 900000000000508004 }}")
	require.ElementsMatch(t, []string{"22298006"}, gb.Slice())

	us := evalFixture(t, "<< 138875005 {{ D dialectId = 900000000000509007 }}")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, us.Slice())

	// Combining a dialect with a real description clause must still work.
	both := evalFixture(t, `<< 138875005 {{ D term = "Myocardial", dialectId = 900000000000508004 }}`)
	require.ElementsMatch(t, []string{"22298006"}, both.Slice())
}

// TestEvaluate_GroupCardinalityWithReverseIsRejected covers group cardinality
// combined with a reverse clause.
//
// The reverse path walks the groups of the SOURCE concepts, so it can only report
// 1 or 0 — not a count of the focus concept's groups. Applying a cardinality to
// that pseudo-count answered `[2..*]` with the empty set and `[0..0]` with
// everything: silently wrong, where the rest of the package reports what it
// cannot do.
func TestEvaluate_GroupCardinalityWithReverseIsRejected(t *testing.T) {
	for _, expr := range []string{
		"* : [2..*] { R 363698007 = 22298006 }",
		"* : [0..0] { R 363698007 = 22298006 }",
		"* : [1..1] { R 363698007 = 22298006 }",
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := evalFixtureErr(t, expr)
			require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
		})
	}

	// Without a cardinality the reverse group still works.
	set := evalFixture(t, "* : { R 363698007 = 22298006 }")
	require.ElementsMatch(t, []string{"74281007", "113331007"}, set.Slice())
}

// TestParse_DialectAcceptabilityIsPairedInOrder covers pairing each dialect with
// the acceptability that follows it.
//
// The grammar makes acceptability optional per entry, so ANTLR's flat
// AllEclconceptreference() and AllAcceptabilityset() lists cannot be zipped by
// index: doing that gave the first acceptability to the first dialect wherever it
// appeared, and made `(A B (X))` and `(A (X) B)` produce identical ASTs.
func TestParse_DialectAcceptabilityIsPairedInOrder(t *testing.T) {
	first := dialectFilterOf(t, "<< 404684003 {{ D dialectId = (900000000000508004 (900000000000548007) 900000000000509007) }}")
	require.Len(t, first.Dialects, 2)
	require.Len(t, first.Dialects[0].Acceptabilities, 1, "the acceptability belongs to the dialect it follows")
	require.Empty(t, first.Dialects[1].Acceptabilities)

	second := dialectFilterOf(t, "<< 404684003 {{ D dialectId = (900000000000508004 900000000000509007 (900000000000548007)) }}")
	require.Len(t, second.Dialects, 2)
	require.Empty(t, second.Dialects[0].Acceptabilities)
	require.Len(t, second.Dialects[1].Acceptabilities, 1)

	require.False(t, reflect.DeepEqual(first, second),
		"the two orderings must not produce the same AST")
}

// TestParse_AcceptabilitySetKeepsEveryValue covers an acceptability set, which
// used to be truncated to its first value and silently narrowed the query.
func TestParse_AcceptabilitySetKeepsEveryValue(t *testing.T) {
	f := dialectFilterOf(t, "<< 404684003 {{ D dialectId = 900000000000508004 (900000000000548007 900000000000549004) }}")
	require.Len(t, f.Dialects, 1)
	require.Len(t, f.Dialects[0].Acceptabilities, 2)
}

// dialectFilterOf returns the DialectFilter of a parsed expression.
func dialectFilterOf(t *testing.T, expr string) *ast.DialectFilter {
	t.Helper()
	tree := mustParse(t, expr)
	filtered, ok := tree.(*ast.Filtered)
	require.Truef(t, ok, "expected *ast.Filtered, got %T", tree)
	for _, f := range filtered.Filters {
		if df, ok := f.(*ast.DialectFilter); ok {
			return df
		}
	}
	require.FailNow(t, "no DialectFilter found")
	return nil
}

// TestEvaluate_ReversePathIsConsistent covers the reverse (R) attribute.
//
// RelationshipTargets returns a Set, so it loses how many inbound relationships
// each concept has and of which types. A cardinality needs the count and "!="
// needs the per-type total, and both used to be answered from the Set anyway:
// `[0..0] R a = x` returned exactly the concepts that DO have the relationship,
// and `R a != x` kept the "does not have it at all" reading the forward path
// abandoned.
//
// A provider implementing ecl.InboundRelationshipsProvider — as the reference
// fixture does — supplies what is missing, and the counting then mirrors the
// forward path exactly. Without that capability the same forms report
// ErrUnsupportedFeature; see TestEvaluate_ReverseCountingNeedsTheCapability.
func TestEvaluate_ReversePathIsConsistent(t *testing.T) {
	// Inbound 363698007: 74281007 <- 22298006, 113331007 <- 22298006 and <- 73211009.
	require.ElementsMatch(t, []string{"74281007", "113331007"},
		evalFixture(t, "* : R 363698007 = 22298006").Slice())

	// Each of those has exactly one inbound relationship from 22298006.
	require.ElementsMatch(t, []string{"74281007", "113331007"},
		evalFixture(t, "* : [1..1] R 363698007 = 22298006").Slice())
	require.Empty(t, evalFixture(t, "* : [2..*] R 363698007 = 22298006").Slice())

	// [0..0] is the complement, and must not coincide with the positive form.
	zero := evalFixture(t, "* : [0..0] R 363698007 = 22298006")
	require.NotContains(t, zero.Slice(), "74281007")
	require.NotContains(t, zero.Slice(), "113331007")
	require.Contains(t, zero.Slice(), "22298006")

	// "!=" selects concepts pointed at from somewhere OTHER than the value set,
	// not concepts nothing points at: 113331007 is also a finding site of
	// 73211009, while 74281007's only inbound is from 22298006.
	require.ElementsMatch(t, []string{"113331007"},
		evalFixture(t, "* : R 363698007 != 22298006").Slice())

	// The group forms stay unsupported: conceptMatchesGroupWithReverse walks the
	// groups of the SOURCE concepts, so the counting problem is not the only one.
	for _, expr := range []string{
		"* : { R 363698007 != 22298006 }",
		"* : { R 363698007 = 22298006 OR R 116676008 = 22298006 }",
		"* : [2..*] { R 363698007 = 22298006 }",
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := evalFixtureErr(t, expr)
			require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
		})
	}
}

// TestEvaluate_ReverseCountingNeedsTheCapability covers the fallback: a provider
// without InboundRelationshipsProvider must be told, not given a wrong count.
func TestEvaluate_ReverseCountingNeedsTheCapability(t *testing.T) {
	plain := withoutInboundCapability{standardProvider(t)}

	// The expressible form still works without the capability.
	set, err := ecl.Evaluate(context.Background(), mustParse(t, "* : R 363698007 = 22298006"), plain)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"74281007", "113331007"}, set.Slice())

	for _, expr := range []string{
		"* : [0..0] R 363698007 = 22298006",
		"* : R 363698007 != 22298006",
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := ecl.Evaluate(context.Background(), mustParse(t, expr), plain)
			require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
			require.Contains(t, err.Error(), "InboundRelationshipsProvider")
		})
	}
}

// withoutInboundCapability hides the optional capabilities of the provider it
// wraps, so the fallback path can be exercised. Embedding the interface rather
// than the concrete type is what drops the extra methods.
type withoutInboundCapability struct{ ecl.DataProvider }

// TestEvaluate_HandBuiltDisjunctionOnlyRefinement covers a refinement carrying
// only a Disjunction, which the parser cannot produce but a consumer can build:
// ecl/ast is public.
//
// Seeding the accumulator with the incoming focus (rather than the empty set)
// made it a no-op that returned everything.
func TestEvaluate_HandBuiltDisjunctionOnlyRefinement(t *testing.T) {
	clause := func(typeID, target string) *ast.Refinement {
		return &ast.Refinement{AttrSet: &ast.AttributeSet{Attr: &ast.Attribute{
			Name:  &ast.ConceptRef{ID: typeID},
			Op:    "=",
			Value: &ast.ConceptRef{ID: target},
		}}}
	}
	expr := &ast.Refined{
		Focus: &ast.Any{},
		Refinement: &ast.Refinement{Disjunction: []*ast.Refinement{
			clause("363698007", "74281007"),
			clause("1142139005", "5"),
		}},
	}

	set, err := ecl.Evaluate(context.Background(), expr, standardProvider(t))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"22298006"}, set.Slice(),
		"the disjunction must be the union of its operands, not the whole focus")
}

// TestParse_AcceptabilityTokenForm covers `(prefer)` / `(accept)`.
//
// Only the SCTID set used to be read, so the token form was dropped entirely --
// and an empty AcceptabilityIDs means "any acceptability" to the provider, so the
// query silently WIDENED instead of narrowing.
func TestParse_AcceptabilityTokenForm(t *testing.T) {
	tokens := dialectFilterOf(t, "<< 404684003 {{ D dialectId = 900000000000509007 (prefer) }}")
	require.Len(t, tokens.Dialects, 1)
	require.Len(t, tokens.Dialects[0].Acceptabilities, 1)

	// It must agree with the equivalent SCTID form.
	byToken := evalFixture(t, "<< 138875005 {{ D dialectId = 900000000000509007 (prefer) }}")
	bySCTID := evalFixture(t, "<< 138875005 {{ D dialectId = 900000000000509007 (900000000000548007) }}")
	require.ElementsMatch(t, bySCTID.Slice(), byToken.Slice())
	require.ElementsMatch(t, []string{"73211009"}, byToken.Slice())

	// And it must discriminate: 22298006 is only acceptable in en-us.
	accept := evalFixture(t, "<< 138875005 {{ D dialectId = 900000000000509007 (accept) }}")
	require.ElementsMatch(t, []string{"22298006"}, accept.Slice())
}

// TestParse_TrailingInputMessageIsValidUTF8 covers the trailing-input message.
// ANTLR's GetStart is a RUNE index, so slicing the input string by it cut
// multi-byte characters in half and produced mojibake.
func TestParse_TrailingInputMessageIsValidUTF8(t *testing.T) {
	_, err := ecl.Parse("404684003 |ááá| GARBAGE")
	require.Error(t, err)
	require.True(t, utf8.ValidString(err.Error()), "error message is not valid UTF-8: %q", err.Error())
	require.Contains(t, err.Error(), "GARBAGE")
}

// TestEvaluate_DescriptionFilter_Negated_WithCapability covers the row-level
// negation the optional ecl.NegatingDescriptionProvider enables. The bundled
// fixture implements it.
//
// Row-level is the whole point: 22298006 has an FSN in `en` and a synonym in `es`,
// so it satisfies `language != es` THROUGH THE FSN. The set-level Minus this
// replaced subtracted the concepts having a Spanish description and so excluded
// it — the defect this pins down.
func TestEvaluate_DescriptionFilter_Negated_WithCapability(t *testing.T) {
	spanish := evalFixture(t, "<< 404684003 {{ D language = es }}")
	require.ElementsMatch(t, []string{"22298006"}, spanish.Slice())

	notSpanish := evalFixture(t, "<< 404684003 {{ D language != es }}")
	require.Contains(t, notSpanish.Slice(), "22298006",
		"22298006 has an English FSN, so it HAS a description whose language is not es")

	// The two are not complements, which is exactly what set subtraction assumed.
	require.NotEqual(t, len(spanish.Slice())+len(notSpanish.Slice()),
		len(evalFixture(t, "<< 404684003").Slice()),
		"a row-level negation does not partition the base set")

	byType := evalFixture(t, "<< 404684003 {{ D type != fsn }}")
	require.ElementsMatch(t, []string{"11687002", "22298006"}, byType.Slice())
}

// TestEvaluate_NegatedDialectIsOwnedByTheProvider covers `{{ D dialectId != … }}`.
//
// The negation travels to the provider through DialectFilterOpts.Negate, so it
// needs no NegatingDescriptionProvider — and it must be applied exactly once.
// Counting the dialect clause among the row-level-negation family demanded a
// capability for an expression the provider can already answer, and the dialect
// loop then applied the clause a second time.
func TestEvaluate_NegatedDialectIsOwnedByTheProvider(t *testing.T) {
	// 22298006 is preferred in en-gb AND acceptable in en-us; 73211009 exists only
	// in en-us. So the negated form keeps 22298006 through its en-us row — it is
	// not the complement of the positive form.
	positive := evalFixture(t, "* {{ D dialectId = 900000000000508004 }}")
	require.ElementsMatch(t, []string{"22298006"}, positive.Slice())

	negated := evalFixture(t, "* {{ D dialectId != 900000000000508004 }}")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, negated.Slice())

	// Combined with a real description clause, both must apply.
	both := evalFixture(t, `<< 404684003 {{ D term = "Diabetes", dialectId != 900000000000508004 }}`)
	require.ElementsMatch(t, []string{"73211009"}, both.Slice())
}

// TestEvaluate_MemberOfProjectionIsClassifiable covers the field-projection
// rejection carrying the sentinel, so the CLI maps it to the documented exit code
// rather than to a generic failure.
func TestEvaluate_MemberOfProjectionIsClassifiable(t *testing.T) {
	_, err := evalFixtureErr(t, "^ [mapTarget] 900000000000497000")
	require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
	require.Contains(t, err.Error(), "^[mapTarget]", "the field list must not be rendered as a Go slice")
}
