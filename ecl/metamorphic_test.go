package ecl_test

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// Metamorphic tests: relations between the results of DIFFERENT expressions,
// so they need neither a reference implementation nor a hand-computed expected
// set. The filter and set-algebra relations hold whatever the data; the numeric
// ones assume each concept stores at most one value for the attribute, which
// the semantics fixture guarantees. They do not replace the specification —
// an evaluator wrong in a self-consistent way passes them — but they are cheap,
// and they catch exactly the defects a second engine tends to share: the
// block-order dependence of H2b and the float64 ordering of H4 (see
// semantics_test.go) both surface here without anyone writing down a result,
// as does H2. H1 and H3 do not: a wrong scope or a lost correlation can be
// perfectly self-consistent, so only the hand-written cases catch those.
//
// Every relation that does not hold today is listed in a knownViolations map
// with the defect behind it. The list is strict in both directions: a violation
// missing from it fails, and so does a listed one that no longer occurs, so
// fixing a defect forces its entries out.

// semBlocks are description-filter clauses chosen so that pairs of them
// disagree on the semantics fixture (semanticsProvider in semantics_test.go).
var semBlocks = []string{
	`term = "heart"`,
	`term = "corazon"`,
	`term = "nonexistent"`,
	`language = en`,
	`language = es`,
	`dialectId = ` + semGB,
}

func TestMetamorphic_FilterBlocks(t *testing.T) {
	p := semanticsProvider()
	known := map[string]string{
		// H2b flattens the blocks: the last term wins, so order matters...
		`commute | {{ D term = "heart" }} {{ D term = "corazon" }}`:       "H2b",
		`commute | {{ D term = "heart" }} {{ D term = "nonexistent" }}`:   "H2b",
		`commute | {{ D term = "corazon" }} {{ D term = "nonexistent" }}`: "H2b",
		// ...languages are pooled into an any-of...
		`blocks=AND | {{ D language = en }} {{ D language = es }}`: "H2b",
		// ...and a term and a language from different blocks are required of
		// one description.
		`blocks=AND | {{ D term = "heart" }} {{ D language = es }}`:   "H2b",
		`blocks=AND | {{ D term = "corazon" }} {{ D language = en }}`: "H2b",
		// H2 is the same flattening inside one block: the last term wins...
		`clauses commute | {{ D term = "heart", term = "corazon" }}`:       "H2",
		`clauses commute | {{ D term = "heart", term = "nonexistent" }}`:   "H2",
		`clauses commute | {{ D term = "corazon", term = "nonexistent" }}`: "H2",
		// ...and two languages widen the block instead of narrowing it.
		`block⊆each clause | {{ D language = en, language = es }}`: "H2",
	}
	ev := newEvalCache(p)
	var got []string
	for i, a := range semBlocks {
		for _, b := range semBlocks[i+1:] {
			pair := fmt.Sprintf(`{{ D %s }} {{ D %s }}`, a, b)
			block := fmt.Sprintf(`{{ D %s, %s }}`, a, b)
			ab := ev.ids(t, `* `+pair)
			one := ev.ids(t, `* `+block)

			if !equalIDs(ab, ev.ids(t, fmt.Sprintf(`* {{ D %s }} {{ D %s }}`, b, a))) {
				got = append(got, "commute | "+pair)
			}
			if !equalIDs(ab, ev.ids(t, fmt.Sprintf(`(* {{ D %s }}) AND (* {{ D %s }})`, a, b))) {
				got = append(got, "blocks=AND | "+pair)
			}
			// Under flattening the one-block and two-block forms compile to the
			// same predicate, so the next relation cannot fail today; the two
			// after it compare against single-clause blocks and can.
			if !subsetIDs(one, ab) {
				got = append(got, "block⊆blocks | "+block)
			}
			if !equalIDs(one, ev.ids(t, fmt.Sprintf(`* {{ D %s, %s }}`, b, a))) {
				got = append(got, "clauses commute | "+block)
			}
			if !subsetIDs(one, ev.ids(t, fmt.Sprintf(`* {{ D %s }}`, a))) || !subsetIDs(one, ev.ids(t, fmt.Sprintf(`* {{ D %s }}`, b))) {
				got = append(got, "block⊆each clause | "+block)
			}
		}
	}
	checkKnownViolations(t, got, known)
}

// algebraExprs mix hierarchy, refinement, filters and set operators over
// standard.yaml, so the set identities below are checked on results produced
// by every part of the evaluator rather than on literal sets.
var algebraExprs = []string{
	`<< 404684003`,
	`< 64572001`,
	`<< 404684003 : 363698007 = *`,
	`<< 404684003 : 363698007 = << 74281007`,
	`<< 404684003 {{ D term = "infarction" }}`,
	`<< 404684003 {{ C definitionStatusId = 900000000000073002 }}`,
	`(< 404684003) MINUS (<< 22298006)`,
}

func TestMetamorphic_SetAlgebra(t *testing.T) {
	ev := newEvalCache(standardProvider(t))
	var got []string
	for i, a := range algebraExprs {
		ra := ev.ids(t, a)
		if len(ra) == 0 {
			t.Fatalf("%q is empty on standard.yaml, which makes every identity below vacuous; pick another expression", a)
		}
		for _, id := range []struct{ name, format string }{
			{"A AND A = A", "(%[1]s) AND (%[1]s)"},
			{"A OR A = A", "(%[1]s) OR (%[1]s)"},
		} {
			if !equalIDs(ra, ev.ids(t, fmt.Sprintf(id.format, a))) {
				got = append(got, id.name+" | "+a)
			}
		}
		if r := ev.ids(t, fmt.Sprintf("(%[1]s) MINUS (%[1]s)", a)); len(r) != 0 {
			got = append(got, "A MINUS A = ∅ | "+a)
		}
		for j, b := range algebraExprs {
			if i == j {
				continue
			}
			// Commutativity is symmetric, so it is checked once per unordered pair.
			ab := fmt.Sprintf("(%s) AND (%s)", a, b)
			if i < j && !equalIDs(ev.ids(t, ab), ev.ids(t, fmt.Sprintf("(%s) AND (%s)", b, a))) {
				got = append(got, "AND commutes | "+ab)
			}
			if !subsetIDs(ev.ids(t, ab), ra) {
				got = append(got, "A AND B ⊆ A | "+ab)
			}
			or := fmt.Sprintf("(%s) OR (%s)", a, b)
			if i < j && !equalIDs(ev.ids(t, or), ev.ids(t, fmt.Sprintf("(%s) OR (%s)", b, a))) {
				got = append(got, "OR commutes | "+or)
			}
			if !subsetIDs(ra, ev.ids(t, or)) {
				got = append(got, "A ⊆ A OR B | "+or)
			}
			minus := fmt.Sprintf("(%s) MINUS (%s)", a, b)
			rm := ev.ids(t, minus)
			if !subsetIDs(rm, ra) || !disjointIDs(rm, ev.ids(t, b)) {
				got = append(got, "A MINUS B ⊆ A, disjoint from B | "+minus)
			}
		}
	}
	checkKnownViolations(t, got, map[string]string{})
}

// numericLiterals straddle the two stored values in the semantics fixture: 2^53,
// where float64 stops telling neighboring integers apart, and MaxInt64, where
// they are 1024 apart. They are integers only (see TestSemantics_ConcreteNumbers
// on decimals) and listed in ascending value, so the test orders them by
// position and never through the float64 conversion it is checking.
var numericLiterals = []string{
	"9007199254740991",
	"9007199254740992",
	"9007199254740993",
	"9007199254740994",
	"9223372036854775806",
	"9223372036854775807",
}

func TestMetamorphic_NumericComparisons(t *testing.T) {
	require64BitInt(t)
	ev := newEvalCache(semanticsProvider())
	q := func(op, lit string) []string {
		return ev.ids(t, fmt.Sprintf(`* : %s %s #%s`, semStrength, op, lit))
	}
	known := map[string]string{
		// 2^53+1 rounds onto the stored 2^53.
		`= ⊆ < next | = #9007199254740992 ⊆ < #9007199254740993`:         "H4",
		`= values are exclusive | #9007199254740992 / #9007199254740993`: "H4",
		// MaxInt64-1 rounds onto the stored MaxInt64.
		`= ⊆ < next | = #9223372036854775806 ⊆ < #9223372036854775807`:         "H4",
		`= values are exclusive | #9223372036854775806 / #9223372036854775807`: "H4",
	}
	var got []string
	for i, n := range numericLiterals {
		eq, lt, le, gt, ge, ne := q("=", n), q("<", n), q("<=", n), q(">", n), q(">=", n), q("!=", n)
		if !equalIDs(unionIDs(lt, eq), le) {
			got = append(got, "< ∪ = is <= | #"+n)
		}
		if !equalIDs(unionIDs(gt, eq), ge) {
			got = append(got, "> ∪ = is >= | #"+n)
		}
		if !disjointIDs(eq, ne) || !disjointIDs(lt, ge) || !disjointIDs(gt, le) {
			got = append(got, "complements are disjoint | #"+n)
		}
		for _, m := range numericLiterals[i+1:] {
			// A concept with one value v = n satisfies v < m for every larger m.
			if !subsetIDs(eq, q("<", m)) {
				got = append(got, fmt.Sprintf("= ⊆ < next | = #%s ⊆ < #%s", n, m))
			}
			// With a single stored value, no concept equals two different numbers.
			if !disjointIDs(eq, q("=", m)) {
				got = append(got, fmt.Sprintf("= values are exclusive | #%s / #%s", n, m))
			}
		}
	}
	checkKnownViolations(t, got, known)
}

// checkKnownViolations fails on any violation not listed and on any listed one
// that did not occur, and logs the listed ones that did.
func checkKnownViolations(t *testing.T, got []string, known map[string]string) {
	t.Helper()
	seen := map[string]bool{}
	for _, v := range got {
		seen[v] = true
		if d, ok := known[v]; ok {
			t.Logf("KNOWN DEFECT %s: %s", d, v)
			continue
		}
		t.Errorf("relation violated: %s", v)
	}
	var fixed []string
	for v, d := range known {
		if !seen[v] {
			fixed = append(fixed, fmt.Sprintf("%s (%s)", v, d))
		}
	}
	sort.Strings(fixed)
	if len(fixed) > 0 {
		t.Errorf("these listed violations no longer occur, so their defect looks fixed — remove them:\n  %s", strings.Join(fixed, "\n  "))
	}
}

// evalIDs evaluates expr and returns its IDs, sorted as Set.Slice promises. A
// construct the evaluator reports unsupported fails the test: every expression
// here is meant to be inside the supported subset, and silently dropping one
// would shrink the evidence without anyone noticing.
func evalIDs(t *testing.T, p ecl.DataProvider, expr string) []string {
	t.Helper()
	set, err := evaluate(p, expr)
	if errors.Is(err, ecl.ErrUnsupportedFeature) {
		t.Fatalf("%q is unsupported; remove it rather than skip it: %v", expr, err)
	}
	require.NoErrorf(t, err, "evaluating %q", expr)
	return set.Slice()
}

// evalCache evaluates each distinct expression once: the relations above reuse
// the same operands across many pairs. It holds no *testing.T, so a failure is
// always reported on the test that asked, even from inside a subtest.
type evalCache struct {
	p    ecl.DataProvider
	seen map[string][]string
}

func newEvalCache(p ecl.DataProvider) *evalCache {
	return &evalCache{p: p, seen: map[string][]string{}}
}

func (c *evalCache) ids(t *testing.T, expr string) []string {
	t.Helper()
	if ids, ok := c.seen[expr]; ok {
		return ids
	}
	ids := evalIDs(t, c.p, expr)
	c.seen[expr] = ids
	return ids
}

func toSet(ids []string) map[string]bool {
	in := make(map[string]bool, len(ids))
	for _, id := range ids {
		in[id] = true
	}
	return in
}

func subsetIDs(a, b []string) bool {
	in := toSet(b)
	return !slices.ContainsFunc(a, func(id string) bool { return !in[id] })
}

func disjointIDs(a, b []string) bool {
	in := toSet(b)
	return !slices.ContainsFunc(a, func(id string) bool { return in[id] })
}

func unionIDs(a, b []string) []string {
	return slices.Compact(slices.Sorted(slices.Values(append(slices.Clone(a), b...))))
}
