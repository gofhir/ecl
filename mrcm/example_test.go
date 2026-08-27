package mrcm_test

import (
	"context"
	"fmt"

	"github.com/gofhir/ecl/mrcm"
	"github.com/gofhir/ecl/scg"
)

// ExampleValidate is the source of truth for the SCG + MRCM snippet in
// README.md.
//
// The README used to show `mrcm.LoadModel("mrcm.json")` and
// `mrcm.NewValidator(model, provider).Validate(ctx, expr)`. Neither function has
// ever existed: the real API is LoadFromJSON / LoadFromBytes plus the free
// function Validate, which returns a *Result as well as an error. Nothing caught
// the drift because no example compiled it.
func ExampleValidate() {
	model, err := mrcm.LoadFromBytes([]byte(`{
	  "domains": [
	    {
	      "attributeId": "363698007",
	      "domainEcl": "<< 404684003",
	      "grouped": true,
	      "cardinality": "0..*"
	    }
	  ],
	  "ranges": [
	    {
	      "attributeId": "363698007",
	      "rangeEcl": "<< 442083009"
	    }
	  ]
	}`))
	if err != nil {
		fmt.Println("load:", err)
		return
	}

	expr, err := scg.Parse("22298006 |Myocardial infarction| : { 363698007 |Finding site| = 74281007 |Body structure| }")
	if err != nil {
		fmt.Println("parse:", err)
		return
	}

	// provider is your ecl.DataProvider; the validator evaluates the rules' ECL
	// through it.
	res, err := mrcm.Validate(context.Background(), expr, model, exampleProvider())
	if err != nil {
		fmt.Println("validate:", err)
		return
	}

	fmt.Println("valid:", res.Valid)
	for _, issue := range res.Issues {
		fmt.Printf("%s: %s\n", issue.Kind, issue.Message)
	}
	// Output: valid: true
}

// ExampleValidate_violation shows what a reported issue looks like: the value
// falls outside the attribute's range.
func ExampleValidate_violation() {
	model, err := mrcm.LoadFromBytes([]byte(`{
	  "domains": [
	    {"attributeId": "363698007", "domainEcl": "<< 404684003", "grouped": true, "cardinality": "0..*"}
	  ],
	  "ranges": [
	    {"attributeId": "363698007", "rangeEcl": "<< 442083009"}
	  ]
	}`))
	if err != nil {
		fmt.Println(err)
		return
	}

	// 386053000 is not under 442083009, so it is outside the range.
	expr, err := scg.Parse("22298006 : { 363698007 = 386053000 }")
	if err != nil {
		fmt.Println(err)
		return
	}

	res, err := mrcm.Validate(context.Background(), expr, model, exampleProvider())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("valid:", res.Valid)
	fmt.Println("kind:", res.Issues[0].Kind)
	// Output:
	// valid: false
	// kind: range_violation
}
