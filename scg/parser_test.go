package scg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_SingleConcept(t *testing.T) {
	expr, err := Parse("404684003")
	require.NoError(t, err)
	require.NotNil(t, expr)
	assert.Equal(t, DefStatusSubtype, expr.DefinitionStatus)
	require.Len(t, expr.FocusConcepts, 1)
	assert.Equal(t, "404684003", expr.FocusConcepts[0].SCTID)
	assert.Empty(t, expr.FocusConcepts[0].Term)
	assert.Empty(t, expr.Refinements)
}

func TestParse_ConceptWithTerm(t *testing.T) {
	expr, err := Parse("404684003 |Clinical finding|")
	require.NoError(t, err)
	require.Len(t, expr.FocusConcepts, 1)
	assert.Equal(t, "404684003", expr.FocusConcepts[0].SCTID)
	assert.Equal(t, "Clinical finding", expr.FocusConcepts[0].Term)
}

func TestParse_SimpleRefinement(t *testing.T) {
	expr, err := Parse("22298006:363698007=74281007")
	require.NoError(t, err)
	require.Len(t, expr.FocusConcepts, 1)
	assert.Equal(t, "22298006", expr.FocusConcepts[0].SCTID)
	require.Len(t, expr.Refinements, 1)
	grp := expr.Refinements[0]
	assert.False(t, grp.Grouped)
	require.Len(t, grp.Attributes, 1)
	assert.Equal(t, "363698007", grp.Attributes[0].Name.SCTID)
	require.NotNil(t, grp.Attributes[0].Value.Concept)
	assert.Equal(t, "74281007", grp.Attributes[0].Value.Concept.SCTID)
}

func TestParse_WithSpaces(t *testing.T) {
	expr, err := Parse("22298006 : 363698007 = 74281007")
	require.NoError(t, err)
	require.Len(t, expr.FocusConcepts, 1)
	require.Len(t, expr.Refinements, 1)
	require.Len(t, expr.Refinements[0].Attributes, 1)
	attr := expr.Refinements[0].Attributes[0]
	assert.Equal(t, "363698007", attr.Name.SCTID)
	require.NotNil(t, attr.Value.Concept)
	assert.Equal(t, "74281007", attr.Value.Concept.SCTID)
}

func TestParse_MultipleFocus(t *testing.T) {
	expr, err := Parse("421720008+7946007")
	require.NoError(t, err)
	require.Len(t, expr.FocusConcepts, 2)
	assert.Equal(t, "421720008", expr.FocusConcepts[0].SCTID)
	assert.Equal(t, "7946007", expr.FocusConcepts[1].SCTID)
	assert.Empty(t, expr.Refinements)
}

func TestParse_MultipleAttrs_Ungrouped(t *testing.T) {
	expr, err := Parse("22298006:363698007=74281007,116676008=55641003")
	require.NoError(t, err)
	require.Len(t, expr.Refinements, 1)
	grp := expr.Refinements[0]
	assert.False(t, grp.Grouped)
	require.Len(t, grp.Attributes, 2)
	assert.Equal(t, "363698007", grp.Attributes[0].Name.SCTID)
	assert.Equal(t, "74281007", grp.Attributes[0].Value.Concept.SCTID)
	assert.Equal(t, "116676008", grp.Attributes[1].Name.SCTID)
	assert.Equal(t, "55641003", grp.Attributes[1].Value.Concept.SCTID)
}

func TestParse_GroupedRefinement(t *testing.T) {
	expr, err := Parse("22298006:{363698007=74281007,116676008=55641003}")
	require.NoError(t, err)
	require.Len(t, expr.Refinements, 1)
	grp := expr.Refinements[0]
	assert.True(t, grp.Grouped)
	require.Len(t, grp.Attributes, 2)
}

func TestParse_MultipleGroups(t *testing.T) {
	expr, err := Parse("71388002:{363698007=39057004},{363698007=53120007}")
	require.NoError(t, err)
	require.Len(t, expr.Refinements, 2)
	assert.True(t, expr.Refinements[0].Grouped)
	assert.True(t, expr.Refinements[1].Grouped)
	require.Len(t, expr.Refinements[0].Attributes, 1)
	require.Len(t, expr.Refinements[1].Attributes, 1)
	assert.Equal(t, "39057004", expr.Refinements[0].Attributes[0].Value.Concept.SCTID)
	assert.Equal(t, "53120007", expr.Refinements[1].Attributes[0].Value.Concept.SCTID)
}

func TestParse_NestedExpression(t *testing.T) {
	expr, err := Parse("373873005:246093002=(386053000:363698007=39057004)")
	require.NoError(t, err)
	require.Len(t, expr.FocusConcepts, 1)
	assert.Equal(t, "373873005", expr.FocusConcepts[0].SCTID)

	require.Len(t, expr.Refinements, 1)
	require.Len(t, expr.Refinements[0].Attributes, 1)
	attr := expr.Refinements[0].Attributes[0]
	assert.Equal(t, "246093002", attr.Name.SCTID)

	require.NotNil(t, attr.Value.Nested)
	nested := attr.Value.Nested
	require.Len(t, nested.FocusConcepts, 1)
	assert.Equal(t, "386053000", nested.FocusConcepts[0].SCTID)
	require.Len(t, nested.Refinements, 1)
	require.Len(t, nested.Refinements[0].Attributes, 1)
	assert.Equal(t, "363698007", nested.Refinements[0].Attributes[0].Name.SCTID)
	assert.Equal(t, "39057004", nested.Refinements[0].Attributes[0].Value.Concept.SCTID)
}

func TestParse_ConcreteInteger(t *testing.T) {
	expr, err := Parse("27658006:411116001=#20")
	require.NoError(t, err)
	require.Len(t, expr.Refinements, 1)
	require.Len(t, expr.Refinements[0].Attributes, 1)
	v := expr.Refinements[0].Attributes[0].Value
	require.NotNil(t, v.Concrete)
	assert.Equal(t, "integer", v.Concrete.Kind)
	assert.Equal(t, int64(20), v.Concrete.Int)
}

func TestParse_ConcreteDecimal(t *testing.T) {
	expr, err := Parse("27658006:411116001=#20.5")
	require.NoError(t, err)
	v := expr.Refinements[0].Attributes[0].Value
	require.NotNil(t, v.Concrete)
	assert.Equal(t, "decimal", v.Concrete.Kind)
	assert.InDelta(t, 20.5, v.Concrete.Float, 0.0001)
}

func TestParse_ConcreteString(t *testing.T) {
	expr, err := Parse(`27658006:411116001="hello"`)
	require.NoError(t, err)
	v := expr.Refinements[0].Attributes[0].Value
	require.NotNil(t, v.Concrete)
	assert.Equal(t, "string", v.Concrete.Kind)
	assert.Equal(t, "hello", v.Concrete.String)
}

func TestParse_ConcreteBoolean(t *testing.T) {
	expr, err := Parse("27658006:411116001=true")
	require.NoError(t, err)
	v := expr.Refinements[0].Attributes[0].Value
	require.NotNil(t, v.Concrete)
	assert.Equal(t, "boolean", v.Concrete.Kind)
	assert.True(t, v.Concrete.Bool)

	expr, err = Parse("27658006:411116001=false")
	require.NoError(t, err)
	v = expr.Refinements[0].Attributes[0].Value
	require.NotNil(t, v.Concrete)
	assert.Equal(t, "boolean", v.Concrete.Kind)
	assert.False(t, v.Concrete.Bool)
}

func TestParse_EquivalentTo(t *testing.T) {
	expr, err := Parse("===22298006")
	require.NoError(t, err)
	assert.Equal(t, DefStatusEquivalent, expr.DefinitionStatus)
	require.Len(t, expr.FocusConcepts, 1)
	assert.Equal(t, "22298006", expr.FocusConcepts[0].SCTID)
}

func TestParse_SubtypeOf(t *testing.T) {
	expr, err := Parse("<<<22298006")
	require.NoError(t, err)
	assert.Equal(t, DefStatusSubtype, expr.DefinitionStatus)
}

func TestParse_DefaultDefStatus(t *testing.T) {
	expr, err := Parse("22298006")
	require.NoError(t, err)
	assert.Equal(t, DefStatusSubtype, expr.DefinitionStatus)
}

func TestParse_Error_InvalidSCTID(t *testing.T) {
	_, err := Parse("12345")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid SCTID length")
}

func TestParse_Error_MissingValue(t *testing.T) {
	_, err := Parse("22298006:363698007=")
	require.Error(t, err)
}

func TestParse_Error_UnterminatedTerm(t *testing.T) {
	_, err := Parse("404684003 |Clinical")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated term")
}

func TestParse_Error_UnclosedGroup(t *testing.T) {
	_, err := Parse("22298006:{363698007=74281007")
	require.Error(t, err)
}

func TestParse_Error_UnclosedString(t *testing.T) {
	_, err := Parse(`27658006:411116001="hello`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated string")
}

func TestParse_Error_MissingFocus(t *testing.T) {
	_, err := Parse("")
	require.Error(t, err)
}

func TestParse_Error_TrailingGarbage(t *testing.T) {
	_, err := Parse("404684003 garbage")
	require.Error(t, err)
}

func TestParse_StringWithEscapes(t *testing.T) {
	expr, err := Parse(`27658006:411116001="he said \"hi\""`)
	require.NoError(t, err)
	v := expr.Refinements[0].Attributes[0].Value
	require.NotNil(t, v.Concrete)
	assert.Equal(t, `he said "hi"`, v.Concrete.String)
}

func TestParse_FocusWithTermAndRefinement(t *testing.T) {
	expr, err := Parse("22298006 |Myocardial infarction|: 363698007 |Finding site| = 74281007 |Myocardium|")
	require.NoError(t, err)
	require.Len(t, expr.FocusConcepts, 1)
	assert.Equal(t, "Myocardial infarction", expr.FocusConcepts[0].Term)
	require.Len(t, expr.Refinements, 1)
	require.Len(t, expr.Refinements[0].Attributes, 1)
	attr := expr.Refinements[0].Attributes[0]
	assert.Equal(t, "Finding site", attr.Name.Term)
	require.NotNil(t, attr.Value.Concept)
	assert.Equal(t, "Myocardium", attr.Value.Concept.Term)
}
