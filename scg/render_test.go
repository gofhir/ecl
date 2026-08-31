package scg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestString_RoundTrips is the property FuzzParseRenderParse checks, stated on
// the shapes worth naming: rendering an expression and parsing the result gives
// back an equal expression.
func TestString_RoundTrips(t *testing.T) {
	for _, expr := range []string{
		"22298006",
		"22298006 |Myocardial infarction|",
		"<<< 22298006",
		"22298006 : 363698007 = 74281007",
		"22298006:{363698007=74281007,116676008=55641003}",
		"22298006:363698007=74281007,{116676008=55641003}",
		"22298006:{363698007=74281007}{116676008=55641003}",
		"421720008+322236009:363698007=74281007",
		"373873005:246093002=(386053000:363698007=39057004)",
		"27658006:411116001=#20",
		"27658006:411116001=#20.5",
		"27658006:411116001=true",
		`27658006:411116001="text"`,
		"=== 414545008|Ischemic heart disease|+251061000|Myocardial necrosis|:{116676008|Associated morphology|=55641003|Infarct|,363698007|Finding site|=74281007|Myocardium structure|}",
	} {
		t.Run(expr, func(t *testing.T) {
			first, err := Parse(expr)
			require.NoError(t, err)

			second, err := Parse(first.String())
			require.NoErrorf(t, err, "the rendering does not parse: %q", first.String())
			require.Truef(t, reflect.DeepEqual(first, second),
				"rendering changed the expression\n  rendered: %q\n  before:   %#v\n  after:    %#v",
				first.String(), first, second)
		})
	}
}

// TestString_WholeDecimalKeepsItsPoint covers the first thing
// FuzzParseRenderParse found, in under a second.
//
// Compositional Grammar tells an integer from a decimal by the decimal POINT
// alone, and strconv renders 0.0 as "0" — so a whole-number decimal came back as an
// INTEGER — a different Kind for the same number, and silently, since nothing
// else in the expression changes.
func TestString_WholeDecimalKeepsItsPoint(t *testing.T) {
	first, err := Parse("000000:000000=#.0")
	require.NoError(t, err)
	assert.Equal(t, "=== 000000 : 000000 = #0.0", first.String())

	second, err := Parse(first.String())
	require.NoError(t, err)
	assert.Equal(t, "decimal", second.Refinements[0].Attributes[0].Value.Concrete.Kind,
		"a decimal must not become an integer by being rendered")
}

// TestString_NestedExpressionHasNoDefinitionStatus covers the other thing writing
// the round-trip property found immediately.
//
// The grammar allows a definition status on the top-level expression only, so
// `( === X )` does not parse — but the parser stamps every nested expression with
// the "===" default, so rendering it looked harmless until the output was fed
// back in.
func TestString_NestedExpressionHasNoDefinitionStatus(t *testing.T) {
	expr, err := Parse("373873005:246093002=(386053000:363698007=39057004)")
	require.NoError(t, err)

	rendered := expr.String()
	assert.Equal(t, "=== 373873005 : 246093002 = ( 386053000 : 363698007 = 39057004 )", rendered)
	assert.NotContains(t, rendered, "( ===", "a nested expression carries no definition status")

	_, err = Parse(rendered)
	require.NoError(t, err)
}

// TestString_EscapesStringLiterals covers the escaping half, which is the half
// worth having the property for: the equivalent asymmetry in this repository's ECL
// parser was a 26-byte input that grew the heap without bound.
func TestString_EscapesStringLiterals(t *testing.T) {
	for _, tc := range []struct{ value, rendered string }{
		{`a"b`, `"a\"b"`},
		{`C:\temp`, `"C:\\temp"`},
		{`ends with a backslash \`, `"ends with a backslash \\"`},
	} {
		t.Run(tc.value, func(t *testing.T) {
			expr := &Expression{
				DefinitionStatus: DefStatusEquivalent,
				FocusConcepts:    []ConceptRef{{SCTID: "27658006"}},
				Refinements: []AttributeGroup{{Attributes: []Attribute{{
					Name:  ConceptRef{SCTID: "411116001"},
					Value: AttributeValue{Concrete: &ConcreteValue{Kind: "string", String: tc.value}},
				}}}},
			}
			assert.Contains(t, expr.String(), tc.rendered)

			back, err := Parse(expr.String())
			require.NoError(t, err)
			assert.Equal(t, tc.value, back.Refinements[0].Attributes[0].Value.Concrete.String,
				"the value must survive rendering unchanged")
		})
	}
}

// TestString_TermWithPipeIsOmitted covers the one thing the renderer cannot
// express. Compositional Grammar has no escape for "|" inside a term, so emitting
// it would produce text that does not parse. The parser cannot build such a term;
// only code can.
func TestString_TermWithPipeIsOmitted(t *testing.T) {
	expr := &Expression{
		FocusConcepts: []ConceptRef{{SCTID: "22298006", Term: "a|b"}},
	}
	rendered := expr.String()
	assert.Equal(t, "=== 22298006", rendered)

	_, err := Parse(rendered)
	require.NoError(t, err, "dropping the term must leave something that parses")
}

// TestString_NilIsEmpty keeps the method safe on a nil receiver, which a caller
// gets from Parse on an error and may print in the same breath.
func TestString_NilIsEmpty(t *testing.T) {
	var expr *Expression
	assert.Equal(t, "", expr.String())
}
