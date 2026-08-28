package ecl

import (
	"strings"
	"testing"
	"time"

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

func TestParse_TermFilter_MatchTypes(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		matchType string
	}{
		{"default match", `<< 404684003 {{ D term = "diabetes" }}`, "match"},
		{"explicit match", `<< 404684003 {{ D term = match: "diabetes" }}`, "match"},
		{"wild", `<< 404684003 {{ D term = wild: "diabet*" }}`, "wild"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := Parse(tc.input)
			require.NoError(t, err)
			filt, ok := expr.(*ast.Filtered)
			require.True(t, ok)
			var tf *ast.TermFilter
			for _, f := range filt.Filters {
				if x, ok := f.(*ast.TermFilter); ok {
					tf = x
					break
				}
			}
			require.NotNil(t, tf, "expected a TermFilter clause")
			require.Len(t, tf.Terms, 1)
			assert.Equal(t, tc.matchType, tf.Terms[0].MatchType)
		})
	}
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

// TestParse_InvalidEscapeInTermTerminates covers an invalid escape sequence
// inside a quoted term.
//
// `* {{ D term = "C:\temp" }}` is 26 bytes and something a person types by
// accident — any backslash that is not \\ or \" qualifies. Under SLL prediction
// paired with ANTLR's BailErrorStrategy it did not terminate: the heap grew past
// 5 GB and kept going, because Bail makes Sync a no-op and without
// resynchronization the parser loops in a subrule without consuming input.
//
// FuzzParse found it twelve seconds after that target was written, which is the
// argument for the target. The seed corpus under testdata/fuzz/ keeps the exact
// input as a regression case; this test states the property in a form that names
// the cause.
func TestParse_InvalidEscapeInTermTerminates(t *testing.T) {
	for _, expr := range []string{
		`* {{ D term = "C:\temp" }}`,
		`* {{ D term = "a\b" }}`,
		`* {{ D term = "\b" }}`,
		`* {{ D term = "a\bc\bd" }}`,
	} {
		t.Run(expr, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				// The result is not the point — rejecting it is fine, accepting
				// it would be fine. Returning at all is the property.
				_, _ = Parse(expr)
			}()
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				t.Fatalf("Parse did not return within 20s for %d bytes of input, so it is not terminating", len(expr))
			}
		})
	}

	// The valid escapes must keep working, or the fix would be "reject anything
	// with a backslash".
	for _, expr := range []string{
		`* {{ D term = "a\\b" }}`,
		`* {{ D term = "a\"b" }}`,
	} {
		if _, err := Parse(expr); err != nil {
			t.Errorf("%s must parse: %v", expr, err)
		}
	}
}

// TestParse_InputLimits covers the size and nesting guards.
//
// They exist because ANTLR gives no way to interrupt a parse in progress, so a
// context deadline would let a caller stop waiting while the goroutine kept
// burning CPU. Bounding the input before starting is the only defense that
// actually bounds the work.
func TestParse_InputLimits(t *testing.T) {
	t.Run("nesting over the limit is rejected", func(t *testing.T) {
		expr := strings.Repeat("(", MaxNestingDepth+1) + "404684003" +
			strings.Repeat(")", MaxNestingDepth+1)
		_, err := Parse(expr)
		require.Error(t, err)

		var pe *ParseError
		require.ErrorAs(t, err, &pe, "the limits report through the same type as a syntax error, so a caller answering 400 needs no new case")
		require.Contains(t, pe.Error(), "nesting")
	})

	t.Run("nesting at the limit is accepted", func(t *testing.T) {
		expr := strings.Repeat("(", MaxNestingDepth) + "404684003" +
			strings.Repeat(")", MaxNestingDepth)
		_, err := Parse(expr)
		require.NoError(t, err, "the limit itself must be usable, or it is really a limit of MaxNestingDepth-1")
	})

	t.Run("size over the limit is rejected", func(t *testing.T) {
		_, err := Parse(strings.Repeat("4", MaxInputBytes+1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "over the")
	})

	// Brackets inside a quoted term or between pipes are DATA, not structure.
	// Counting them would reject valid expressions -- and `"a("` is one paren
	// that never closes, so a naive counter would drift upward across clauses.
	t.Run("brackets in data do not count toward nesting", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("* : 363698007 = 404684003 |Finding (site|")
		for range MaxNestingDepth + 10 {
			sb.WriteString(` {{ D term = "a(b{c" }}`)
		}
		depth, _, _ := maxNestingDepth(sb.String())
		require.LessOrEqual(t, depth, 2,
			"a bracket inside a quoted term or a |term| must not be counted as structure")
	})
}
