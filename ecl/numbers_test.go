package ecl

import (
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseNumber_Forms covers what parseNumber accepts: the decimal forms RF2
// writes concrete values in, and nothing an exact rational parser would also
// take ("1/2", "0x10").
func TestParseNumber_Forms(t *testing.T) {
	for _, s := range []string{"0", "-0", "+1", "42", "1.", ".5", "1.50", "-3.25", "5e2", "5E-2", "1e+3", "1e-401"} {
		_, ok := parseNumber(s)
		require.Truef(t, ok, "%q is a decimal number", s)
	}
	for _, s := range []string{"", ".", "+", "+-1", "1e", "1e+", "1.2.3", "1/2", "0x10", "Inf", "NaN", "1 ", " 1", "1e5x"} {
		_, ok := parseNumber(s)
		require.Falsef(t, ok, "%q is not a decimal number", s)
	}

	// A stored value beyond any bound is still a number, kept by sign and
	// magnitude; a literal beyond its own bound is refused.
	n, ok := parseNumber("1e200000")
	require.True(t, ok)
	require.Equal(t, 1, n.sign)
	require.Equal(t, int64(200000), n.mag)
	_, ok = parseNumberExact("1e5000", maxNumberDigits)
	require.False(t, ok)
}

// TestCompareNumbers_AgreesWithExactArithmetic checks every pair of tricky
// values through compareNumbers against big.Rat, so a fast path (int64 or
// float64) that disagrees with exact arithmetic cannot hide.
func TestCompareNumbers_AgreesWithExactArithmetic(t *testing.T) {
	values := []string{
		"0", "-0", "1", "-1", "0.1", "0.10000000000000001", "0.3", "0.30000000000000004",
		"9007199254740992", "9007199254740993", "9007199254740992.5",
		"9223372036854775807", "9223372036854775808", "-9223372036854775808",
		"123456789012345", "1234567890123456", "1e-308", "2.2250738585072014e-308", "1e-320",
		"5e2", "500", "500.0", "1e-401", "99999999999999999999.5", "1e-5000", "-1e-5000", "1e5000",
		"1e-200000", "-1e200000", "0.000123", "-0.05", "12.5e-3",
	}
	ops := []string{"=", "!=", "<", "<=", ">", ">="}
	for _, a := range values {
		for _, b := range values {
			na, ok := parseNumber(a)
			require.True(t, ok, a)
			nb, err := numericLiteralFromText(b)
			if err != nil {
				// Beyond the literal bound, which is narrower than the stored
				// one: such a value can only be on the stored side.
				continue
			}

			ra, _ := new(big.Rat).SetString(a)
			rb, _ := new(big.Rat).SetString(b)
			c := ra.Cmp(rb)
			want := map[string]bool{"=": c == 0, "!=": c != 0, "<": c < 0, "<=": c <= 0, ">": c > 0, ">=": c >= 0}
			for _, op := range ops {
				require.Equalf(t, want[op], compareNumbers(na, op, nb), "%s %s %s", a, op, b)
			}
		}
	}
}

// numericLiteralFromText parses b the way a literal is parsed, for the test.
func numericLiteralFromText(b string) (number, error) {
	n, ok := parseNumberExact(b, maxNumberDigits)
	if !ok {
		return number{}, errNotANumber
	}
	n.r = n.rat()
	return n, nil
}

var errNotANumber = &ParseError{Errors: []SyntaxError{{Msg: "not a number"}}}

// TestParse_TooLongLiteralIsOneError covers a literal beyond the digit limit:
// one error for the token, not one from the limit and another from the int
// conversion it would also fail.
func TestParse_TooLongLiteralIsOneError(t *testing.T) {
	_, err := Parse(`* : 1142139005 = #` + strings.Repeat("9", 5000))
	var pe *ParseError
	require.ErrorAs(t, err, &pe)
	require.Len(t, pe.Errors, 1)
}

// TestCompareNumbers_ExponentBeyondInt64 covers a stored value whose exponent
// big.Rat cannot hold. Its sign and magnitude still order it, a zero is zero
// whatever its exponent, and nothing dereferences the rational that does not
// exist.
func TestCompareNumbers_ExponentBeyondInt64(t *testing.T) {
	five, err := numericLiteralFromText("5")
	require.NoError(t, err)
	zero, err := numericLiteralFromText("0")
	require.NoError(t, err)

	huge, ok := parseNumber("1e100000000000000000000")
	require.True(t, ok)
	require.True(t, compareNumbers(huge, ">", five))
	require.False(t, compareNumbers(huge, "=", five))

	tiny, ok := parseNumber("1e-100000000000000000000")
	require.True(t, ok)
	require.True(t, compareNumbers(tiny, ">", zero))
	require.True(t, compareNumbers(tiny, "<", five))

	nought, ok := parseNumber("0e100000000000000000000")
	require.True(t, ok)
	require.True(t, compareNumbers(nought, "=", zero))
	require.True(t, compareNumbers(nought, "<", five))
}
