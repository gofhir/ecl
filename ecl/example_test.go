package ecl_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofhir/ecl/ecl"
)

// These examples are the source of truth for the snippets in README.md. CI
// compiles and runs them, so an API change breaks the build instead of leaving
// the README quietly wrong -- which is how the MRCM snippet came to reference two
// functions that never existed.

// ExampleParse shows parsing an expression and inspecting the error.
func ExampleParse() {
	expr, err := ecl.Parse("<< 404684003 |Clinical finding|")
	if err != nil {
		fmt.Println("parse failed:", err)
		return
	}
	fmt.Printf("%T\n", expr)
	// Output: *ast.DescendantOrSelfOf
}

// ExampleParse_errors shows that a syntax error is a typed *ecl.ParseError,
// carrying every position the parser reported.
func ExampleParse_errors() {
	_, err := ecl.Parse("11687002 TOTALGARBAGE")

	var pe *ecl.ParseError
	if errors.As(err, &pe) {
		for _, se := range pe.Errors {
			fmt.Printf("line %d, column %d\n", se.Line, se.Column)
		}
	}
	// Output: line 1, column 9
}

// ExampleEvaluate evaluates an expression against a DataProvider. Here the
// provider is the bundled in-memory fixture; in production it is your own
// implementation over PostgreSQL, Elasticsearch or a terminology server.
func ExampleEvaluate() {
	provider := exampleProvider()

	expr, err := ecl.Parse("<< 73211009")
	if err != nil {
		fmt.Println(err)
		return
	}
	set, err := ecl.Evaluate(context.Background(), expr, provider)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, id := range set.Slice() {
		fmt.Println(id)
	}
	// Output:
	// 11687002
	// 73211009
}

// ExampleEvaluate_refinement shows a refinement with a disjunction, which is
// evaluated as a union.
func ExampleEvaluate_refinement() {
	provider := exampleProvider()

	expr, err := ecl.Parse("* : 363698007 = 74281007 OR 363698007 = 113331007")
	if err != nil {
		fmt.Println(err)
		return
	}
	set, err := ecl.Evaluate(context.Background(), expr, provider)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(set.Slice())
	// Output: [22298006 73211009]
}

// ExampleErrUnsupportedFeature shows classifying a construct this build cannot
// evaluate. The evaluator returns an error rather than a silently wrong set, so a
// server can answer 501 instead of serving bad data.
//
// A reverse attribute inside an attribute group is used here because its rejection
// is permanent by design: braces assert that the clauses share a relationship
// group of the FOCUS concept, and a reverse relationship belongs to the source, so
// there is nothing to group. Most other constructs report the same sentinel only
// until the provider implements the matching optional capability — see
// ecl/capabilities.go.
func ExampleErrUnsupportedFeature() {
	provider := exampleProvider()

	expr, err := ecl.Parse("* : { R 363698007 = 22298006 }")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = ecl.Evaluate(context.Background(), expr, provider)
	fmt.Println(errors.Is(err, ecl.ErrUnsupportedFeature))
	// Output: true
}
