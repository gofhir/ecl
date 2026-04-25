package ecl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl/ast"
)

func TestParse_SimpleDescendantOrSelf(t *testing.T) {
	expr, err := Parse("<< 404684003")
	require.NoError(t, err)
	dos, ok := expr.(*ast.DescendantOrSelfOf)
	require.True(t, ok)
	ref, ok := dos.Operand.(*ast.ConceptRef)
	require.True(t, ok)
	assert.Equal(t, "404684003", ref.ID)
}

func TestParse_ConceptWithTerm(t *testing.T) {
	expr, err := Parse("<< 404684003 |Clinical finding|")
	require.NoError(t, err)
	dos := expr.(*ast.DescendantOrSelfOf)
	ref := dos.Operand.(*ast.ConceptRef)
	assert.Equal(t, "404684003", ref.ID)
	assert.Equal(t, "Clinical finding", ref.Term)
}

func TestParse_Conjunction(t *testing.T) {
	expr, err := Parse("<< 404684003 AND << 73211009")
	require.NoError(t, err)
	_, ok := expr.(*ast.And)
	assert.True(t, ok)
}

func TestParse_Disjunction(t *testing.T) {
	expr, err := Parse("<< 404684003 OR << 73211009")
	require.NoError(t, err)
	_, ok := expr.(*ast.Or)
	assert.True(t, ok)
}

func TestParse_Exclusion(t *testing.T) {
	expr, err := Parse("<< 404684003 MINUS << 73211009")
	require.NoError(t, err)
	_, ok := expr.(*ast.Minus)
	assert.True(t, ok)
}

func TestParse_MemberOf(t *testing.T) {
	expr, err := Parse("^ 900000000000497000")
	require.NoError(t, err)
	_, ok := expr.(*ast.MemberOf)
	assert.True(t, ok)
}

func TestParse_Wildcard(t *testing.T) {
	expr, err := Parse("*")
	require.NoError(t, err)
	_, ok := expr.(*ast.Any)
	assert.True(t, ok)
}

func TestParse_Refined(t *testing.T) {
	expr, err := Parse("<< 404684003 : 363698007 = << 39057004")
	require.NoError(t, err)
	_, ok := expr.(*ast.Refined)
	assert.True(t, ok)
}

func TestParse_GroupedRefinement(t *testing.T) {
	expr, err := Parse("<< 404684003 : { 363698007 = << 39057004 }")
	require.NoError(t, err)
	_, ok := expr.(*ast.Refined)
	assert.True(t, ok)
}

func TestParse_DottedExpression(t *testing.T) {
	expr, err := Parse("<< 404684003 . 363698007")
	require.NoError(t, err)
	_, ok := expr.(*ast.DotExpression)
	assert.True(t, ok)
}

func TestParse_NestedParentheses(t *testing.T) {
	expr, err := Parse("(<< 404684003 OR << 73211009)")
	require.NoError(t, err)
	nested, ok := expr.(*ast.Nested)
	require.True(t, ok)
	_, ok = nested.Inner.(*ast.Or)
	assert.True(t, ok)
}

func TestParse_AllHierarchyOperators(t *testing.T) {
	ops := []struct {
		ecl      string
		typeName string
	}{
		{"< 123456789", "DescendantOf"},
		{"<< 123456789", "DescendantOrSelfOf"},
		{"<! 123456789", "ChildOf"},
		{"<<! 123456789", "ChildOrSelfOf"},
		{"> 123456789", "AncestorOf"},
		{">> 123456789", "AncestorOrSelfOf"},
		{">! 123456789", "ParentOf"},
		{">>! 123456789", "ParentOrSelfOf"},
	}
	for _, op := range ops {
		t.Run(op.typeName, func(t *testing.T) {
			_, err := Parse(op.ecl)
			assert.NoError(t, err, "failed to parse: %s", op.ecl)
		})
	}
}

func TestParse_SingleConcept(t *testing.T) {
	expr, err := Parse("404684003")
	require.NoError(t, err)
	ref, ok := expr.(*ast.ConceptRef)
	require.True(t, ok)
	assert.Equal(t, "404684003", ref.ID)
}

func TestParse_RefsetContainingAny(t *testing.T) {
	expr, err := Parse("^R 900000000000497000")
	require.NoError(t, err)
	rc, ok := expr.(*ast.RefsetContainingAny)
	require.True(t, ok)
	ref, ok := rc.Operand.(*ast.ConceptRef)
	require.True(t, ok)
	assert.Equal(t, "900000000000497000", ref.ID)
}

func TestParse_DescriptionFilter(t *testing.T) {
	expr, err := Parse(`<< 404684003 {{ D term = "diabetes" }}`)
	require.NoError(t, err)
	filt, ok := expr.(*ast.Filtered)
	require.True(t, ok, "expected *ast.Filtered, got %T", expr)
	assert.NotEmpty(t, filt.Filters, "expected at least one filter clause")
	// Operand should be the hierarchy expression that was filtered.
	_, ok = filt.Operand.(*ast.DescendantOrSelfOf)
	assert.True(t, ok, "expected filtered operand to be DescendantOrSelfOf")
}

func TestParse_Error(t *testing.T) {
	_, err := Parse("<<< invalid")
	assert.Error(t, err)
}

func TestParse_ErrorEmpty(t *testing.T) {
	_, err := Parse("")
	assert.Error(t, err, "empty input should produce a syntax error")
}

func TestParse_ErrorInvalidString(t *testing.T) {
	_, err := Parse("invalid ecl string !!!")
	require.Error(t, err)
	// Error message should carry line/column info.
	assert.Contains(t, err.Error(), "1:")
}

func TestParse_ErrorMissingOperand(t *testing.T) {
	_, err := Parse("<< ")
	assert.Error(t, err, "hierarchy operator without operand should fail")
}
