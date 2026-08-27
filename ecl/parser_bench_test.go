package ecl_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gofhir/ecl/ecl"
)

// These benchmarks exist because a performance claim without a measurement is a
// guess, and because the shapes below were slow enough to matter: before the
// two-stage parsing strategy, a 2.2 KB expression with 100 refinement clauses
// took over five seconds, and 400 clauses took eighty-five. The cost was ANTLR's
// default ALL(*) adaptive prediction — visible in a CPU profile as
// ParserATNSimulator.closureWork plus heavy GC pressure — not the AST builder.
//
// Run them with:
//
//	go test ./ecl/ -run '^$' -bench Parse -benchmem
//
// What to look for is the SHAPE, not the absolute numbers. Both families should
// grow roughly linearly in the size parameter. A superlinear curve means the
// parser has fallen back to full LL prediction on ordinary input, which is the
// regression these guard.

// BenchmarkParse_RefinementClauses measures a flat conjunction of attribute
// clauses: `* : a = x, a = x, ...`.
//
// This is the case worth guarding, because it is not adversarial. A long
// conjunction is something a person writes by hand or a query builder emits, so
// any nonlinearity here is a production problem rather than a hardening one.
func BenchmarkParse_RefinementClauses(b *testing.B) {
	for _, n := range []int{10, 50, 100, 200} {
		expr := refinementExpr(n)
		b.Run(fmt.Sprintf("clauses=%d/bytes=%d", n, len(expr)), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := ecl.Parse(expr); err != nil {
					b.Fatalf("this expression must parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkParse_NestedParens measures nesting depth: `((((404684003))))`.
//
// Adversarial rather than realistic — nobody writes 100 nested parentheses — so
// it stands in for hostile input reaching Parse from a query string.
//
// This axis is the one that stayed superlinear: SLL prediction cut the constant by
// three orders of magnitude but not the curve. It is bounded instead, by
// ecl.MaxNestingDepth, which is why the deepest case here is that limit and not
// something larger — anything deeper is rejected before parsing, so the numbers
// below are the worst this package will do.
func BenchmarkParse_NestedParens(b *testing.B) {
	for _, n := range []int{10, 50, ecl.MaxNestingDepth} {
		expr := nestedParensExpr(n)
		b.Run(fmt.Sprintf("depth=%d/bytes=%d", n, len(expr)), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := ecl.Parse(expr); err != nil {
					b.Fatalf("this expression must parse: %v", err)
				}
			}
		})
	}
}

// BenchmarkParse_Typical measures an expression of the size real queries have, so
// the numbers above have something to be read against.
func BenchmarkParse_Typical(b *testing.B) {
	const expr = `<< 404684003 |Clinical finding| : ` +
		`363698007 |Finding site| = << 74281007 |Myocardium structure|, ` +
		`{ 116676008 |Associated morphology| = << 55641003 |Infarct| }`

	b.ReportAllocs()
	for b.Loop() {
		if _, err := ecl.Parse(expr); err != nil {
			b.Fatalf("this expression must parse: %v", err)
		}
	}
}

func refinementExpr(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "363698007 = 74281007"
	}
	return "* : " + strings.Join(parts, ", ")
}

func nestedParensExpr(n int) string {
	return strings.Repeat("(", n) + "404684003" + strings.Repeat(")", n)
}
