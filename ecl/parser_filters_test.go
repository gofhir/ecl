package ecl

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl/ast"
)

// filterOf returns the first filter of the given type from a parsed expression.
func filterOf[T ast.Filter](t *testing.T, expr string) T {
	t.Helper()
	tree, err := Parse(expr)
	require.NoErrorf(t, err, "parsing %q", expr)
	filtered, ok := tree.(*ast.Filtered)
	require.Truef(t, ok, "expected *ast.Filtered, got %T", tree)
	for _, f := range filtered.Filters {
		if typed, ok := f.(T); ok {
			return typed
		}
	}
	var zero T
	require.Failf(t, "filter not found", "no %T in %q", zero, expr)
	return zero
}

// TestParse_TermFilterDecodesEscapes covers the escape sequences the grammar
// defines. The raw token text keeps the backslashes, so a term that reached the
// provider undecoded searched for the backslash too and never matched.
func TestParse_TermFilterDecodesEscapes(t *testing.T) {
	tests := map[string]string{
		`{{ term = "a\"b" }}`:  `a"b`,
		`{{ term = "a\\b" }}`:  `a\b`,
		`{{ term = "plain" }}`: "plain",
	}
	for filter, want := range tests {
		t.Run(filter, func(t *testing.T) {
			tf := filterOf[*ast.TermFilter](t, "< 404684003 "+filter)
			require.Len(t, tf.Terms, 1)
			require.Equal(t, want, tf.Terms[0].Text)
			//nolint:staticcheck // asserting the deprecated field stays populated IS the point
			require.Equal(t, want, tf.Term, "the deprecated scalar must stay populated")
		})
	}
}

// TestParse_WildTermKeepsEscapedAsterisk covers the one escape that must NOT be
// decoded: in a wild pattern, decoding `\*` to a bare asterisk would turn a
// literal asterisk into a wildcard.
func TestParse_WildTermKeepsEscapedAsterisk(t *testing.T) {
	tf := filterOf[*ast.TermFilter](t, `< 404684003 {{ term = wild:"a\*b" }}`)
	require.Len(t, tf.Terms, 1)
	require.Equal(t, `a\*b`, tf.Terms[0].Text)
	require.Equal(t, "wild", tf.Terms[0].MatchType)

	// An unescaped asterisk stays a wildcard.
	glob := filterOf[*ast.TermFilter](t, `< 404684003 {{ term = wild:"a*b" }}`)
	require.Equal(t, "a*b", glob.Terms[0].Text)
}

// TestParse_TermFilterSet covers a set of terms. GetText() of the whole set used
// to be stored as one term, parentheses and inner quotes included, which no
// provider could match.
func TestParse_TermFilterSet(t *testing.T) {
	tf := filterOf[*ast.TermFilter](t, `< 404684003 {{ term = ("heart" "attack") }}`)
	require.Len(t, tf.Terms, 2)
	require.Equal(t, "heart", tf.Terms[0].Text)
	require.Equal(t, "attack", tf.Terms[1].Text)

	// Each member may declare its own search style.
	mixed := filterOf[*ast.TermFilter](t, `< 404684003 {{ term = ("heart" wild:"attac*") }}`)
	require.Len(t, mixed.Terms, 2)
	require.Equal(t, "match", mixed.Terms[0].MatchType)
	require.Equal(t, "wild", mixed.Terms[1].MatchType)
}

// TestParse_ModuleAndDefinitionStatusSets covers concept-reference sets, which
// kept only their first element.
func TestParse_ModuleAndDefinitionStatusSets(t *testing.T) {
	mf := filterOf[*ast.ModuleFilter](t, `<< 404684003 {{ C moduleId = (900000000000207008 900000000000012004) }}`)
	require.Len(t, mf.Modules, 2)
	//nolint:staticcheck // asserting the deprecated field stays populated IS the point
	require.NotNil(t, mf.Module, "the deprecated scalar must stay populated")

	df := filterOf[*ast.DefinitionStatusFilter](t, `<< 404684003 {{ C definitionStatus = (primitive defined) }}`)
	require.Len(t, df.Values, 2)
	require.NotNil(t, df.Value) //nolint:staticcheck // deprecated field must stay populated
}

// TestParse_EffectiveTimeSet covers a time value set, which was stored as one
// value with the parentheses included.
func TestParse_EffectiveTimeSet(t *testing.T) {
	ef := filterOf[*ast.EffectiveTimeFilter](t, `<< 404684003 {{ C effectiveTime = ("20240131" "20230731") }}`)
	require.Len(t, ef.Values, 2)
	require.Equal(t, "20240131", ef.Values[0])
	require.Equal(t, "20230731", ef.Values[1])
	//nolint:staticcheck // asserting the deprecated field stays populated IS the point
	require.Equal(t, "20240131", ef.Value, "the deprecated scalar must stay populated")
}

// TestParse_DescriptionIDFilterIsModeled is the important one: this branch was
// skipped without emitting anything, so when the filter was a constraint's only
// clause the ast.Filtered node was never built and the query silently returned
// every descendant.
func TestParse_DescriptionIDFilterIsModeled(t *testing.T) {
	withFilter, err := Parse("< 404684003 {{ D id = 123456789012 }}")
	require.NoError(t, err)
	bare, err := Parse("< 404684003")
	require.NoError(t, err)
	require.False(t, reflect.DeepEqual(withFilter, bare),
		"the description id filter evaporated: the query returns a superset")

	f := filterOf[*ast.DescriptionIDFilter](t, "< 404684003 {{ D id = 123456789012 }}")
	require.Equal(t, "=", f.Op)
	require.Equal(t, []string{"123456789012"}, f.IDs)

	set := filterOf[*ast.DescriptionIDFilter](t, "< 404684003 {{ D id = (123456789012 234567890123) }}")
	require.Len(t, set.IDs, 2)
}
