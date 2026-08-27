package ecl

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParse_RejectsTrailingInput covers the EOF anchor. The expressionconstraint
// rule in ECL.g4 does not end in EOF, so before this was fixed ANTLR stopped at
// the first complete parse and discarded the rest with err == nil: `validate`
// answered OK on garbage, and an expression with a malformed tail was evaluated
// truncated.
func TestParse_RejectsTrailingInput(t *testing.T) {
	cases := []struct {
		in  string
		why string
	}{
		{"11687002 TOTALGARBAGE", "trailing token after a complete expression"},
		{"<< 404684003 ESTO SOBRA", "trailing words after a hierarchy operator"},
		// MINUS takes exactly two operands (exclusionexpressionconstraint in
		// ECL.g4), so a chain without parentheses is invalid rather than
		// left-associative. It used to truncate silently to "A MINUS B".
		{"404684003 MINUS 11687002 MINUS 73211009", "chained MINUS without parentheses"},
		// Compound sets are homogeneous: mixing AND and OR needs parentheses.
		{"< 404684003 AND < 11687002 OR < 73211009", "AND and OR mixed without parentheses"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			expr, err := Parse(c.in)
			require.Errorf(t, err, "accepted an invalid expression (%s) and truncated the AST", c.why)
			require.Nil(t, expr)
		})
	}
}

// TestParse_AcceptsValidInput guards against the EOF check rejecting valid
// expressions: the expressionconstraint rule consumes trailing whitespace and
// comments itself, so those must still reach EOF.
func TestParse_AcceptsValidInput(t *testing.T) {
	for _, in := range []string{
		"<< 404684003",
		"  << 404684003  ",
		"<< 404684003 |Clinical finding|",
		`<< 404684003 {{ term = "x" }}`,
		"< 404684003 : 363698007 = 74281007",
		"404684003 MINUS 11687002",
		"(< 404684003) AND (< 11687002)",
	} {
		t.Run(in, func(t *testing.T) {
			expr, err := Parse(in)
			require.NoError(t, err)
			require.NotNil(t, expr)
		})
	}
}

// TestParse_ReportsLexerErrors covers registering the error listener on the
// lexer. Without it, ANTLR's default ConsoleErrorListener wrote the complaint to
// os.Stderr from inside the library, dropped the character from the stream, and
// Parse returned a corrupted AST with a nil error: `404684003 |Crohn’s disease|`
// yielded the term "Crohns disease".
//
// The assertion is that the problem is REPORTED, not that the expression is
// invalid ECL: the official ABNF does allow UTF8-2/3/4 in a term, and the real
// fix is widening UTF8_LETTER in ECL.g4 (which needs the parser regenerated).
func TestParse_ReportsLexerErrors(t *testing.T) {
	for _, in := range []string{
		"404684003 |Crohn’s disease|",  // U+2019 right single quotation mark
		"404684003 |Temperatura 37°C|", // U+00B0 degree sign
		"404684003 |Body structure|",   // U+00A0 no-break space (browser copy-paste)
	} {
		t.Run(in, func(t *testing.T) {
			_, err := Parse(in)
			require.Error(t, err, "the character was silently dropped from the token stream")
			require.Contains(t, err.Error(), "token recognition error")
		})
	}

	// The cut is not ASCII vs non-ASCII: UTF8_LETTER starts at 'À', so accented
	// Latin letters are fine and must keep parsing.
	_, err := Parse("404684003 |hipertensión|")
	require.NoError(t, err)
}

// TestParse_ErrorIsTyped covers the typed error, so consumers can classify a
// failure with errors.As instead of matching on message text.
func TestParse_ErrorIsTyped(t *testing.T) {
	_, err := Parse("<< invalid!!!")
	require.Error(t, err)

	var pe *ParseError
	require.ErrorAs(t, err, &pe)
	require.NotEmpty(t, pe.Errors)
	require.Positive(t, pe.Errors[0].Line)
	require.NotEmpty(t, pe.Errors[0].Msg)

	// The single-error text stays byte-compatible with the pre-ParseError format.
	require.Contains(t, err.Error(), "syntax error at ")
}

// TestParse_AccumulatesErrors covers the listener appending instead of
// overwriting: it used to assign to one field, so only the last error survived.
func TestParse_AccumulatesErrors(t *testing.T) {
	_, err := Parse("404684003 |Crohn’s disease°|")
	require.Error(t, err)

	var pe *ParseError
	require.ErrorAs(t, err, &pe)
	require.GreaterOrEqual(t, len(pe.Errors), 2, "only one of several lexer errors was kept")
}

// TestParseError_IsUnwrappable documents that errors.Is/As reach the concrete
// type through the CLI's fmt.Errorf("invalid ECL: %w", err) wrapper.
func TestParseError_IsUnwrappable(t *testing.T) {
	_, err := Parse("11687002 GARBAGE")
	wrapped := errors.Join(errors.New("outer"), err)

	var pe *ParseError
	require.ErrorAs(t, wrapped, &pe)
}
