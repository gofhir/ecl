package mrcm

import (
	"context"
	"testing"
	"time"

	"github.com/gofhir/ecl/scg"
)

// FuzzLoadFromBytes asserts that loading an MRCM document never panics, and —
// the part worth the trouble — that a model which LOADS is a model the validator
// can USE, against an expression that is fuzzed too.
//
// The two halves matter for different reasons. LoadFromBytes takes a document
// from wherever an operator got it: a release package, a hand-edited file, an
// endpoint. And a model that parses but then breaks Validate is the worse bug,
// because the failure surfaces far from its cause — the loader is what is
// supposed to reject a rule the rest of the package cannot handle, so every field
// it lets through becomes an assumption downstream.
//
// The EXPRESSION is a second fuzzed argument rather than a fixed string. With one
// fixed expression the validator only ever saw the attributes that expression
// happened to mention, so most of it — nested expressions, concrete values,
// several focus concepts, groups that do not match any rule — was never reached
// no matter how the model was mutated. Fuzzing the pair is what puts the walking
// code under test rather than only the loading code.
//
// The verdict is not checked: arbitrary rules against an arbitrary expression
// produce an arbitrary verdict, and asserting one would encode the fuzzer's output
// as an expectation. What is checked is that a verdict comes back at all and that
// Result is coherent when it does.
func FuzzLoadFromBytes(f *testing.F) {
	// Every model against every expression: the fuzzer mutates one argument at a
	// time, so pairing them here is what gives it somewhere to start on both axes.
	for _, model := range mrcmSeeds {
		for _, expr := range scgSeedsForMRCM {
			f.Add([]byte(model), expr)
		}
	}

	provider := newTestProvider()

	f.Fuzz(func(t *testing.T, data []byte, exprText string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("LoadFromBytes or Validate panicked on a %d-byte model and %q: %v\n%q",
					len(data), exprText, r, data)
			}
		}()

		// The model is loaded FIRST and unconditionally. Parsing the expression
		// first and returning early on failure would gate the loader's own
		// invariants on the OTHER argument happening to be valid, so most inputs
		// would never reach LoadFromBytes at all — which is a weaker target than
		// the one this replaced.
		start := time.Now()
		model, err := LoadFromBytes(data)
		switch {
		case err != nil && model != nil:
			t.Errorf("LoadFromBytes returned both a model and an error: %v\n%q", err, data)
			return
		case err != nil:
			return // a rejected document is a perfectly good outcome
		case model == nil:
			t.Errorf("LoadFromBytes returned a nil model with a nil error for %q", data)
			return
		}

		expr, exprErr := scg.Parse(exprText)
		if exprErr != nil || expr == nil {
			return // scg.FuzzParse owns the parser; this target is about what follows
		}

		res, err := Validate(context.Background(), expr, model, provider)

		// Timed across BOTH steps. Timing only Validate would miss a document
		// that is slow to LOAD — every domain rule carries an ECL expression the
		// loader parses, so the cost is not all on one side.
		elapsed := time.Since(start)

		if err != nil {
			// A provider or context failure would be an error; with this stub
			// provider there is none to have, so an error here means a rule the
			// loader accepted reached code that could not handle it. That is the
			// boundary this target exists to police.
			t.Errorf("a model the loader ACCEPTED made Validate fail on %q: %v\n%q", exprText, err, data)
			return
		}
		if res == nil {
			t.Errorf("Validate returned a nil Result with a nil error for %q", data)
			return
		}
		if res.Valid != (len(res.Issues) == 0) {
			t.Errorf("Result.Valid is %v with %d issues, so the two disagree and a caller\n"+
				"checking either one gets a different answer\n%q", res.Valid, len(res.Issues), data)
		}

		if elapsed > 5*time.Second {
			t.Errorf("loading and validating took %s on a %d-byte model and %q:\n%q",
				elapsed.Round(time.Millisecond), len(data), exprText, data)
		}
	})
}

// mrcmSeeds covers the shapes the loader has to distinguish: valid documents,
// documents that are valid JSON but invalid MRCM, and bytes that are not JSON.
var mrcmSeeds = []string{
	`{}`,
	`{"domains":[],"ranges":[]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"<< 404684003","grouped":true,` +
		`"cardinality":"0..*","inGroupCardinality":"0..1","ruleStrengthId":"723597001"}],` +
		`"ranges":[{"attributeId":"363698007","rangeEcl":"<< 442083009","ruleStrengthId":"723597001"}]}`,

	// Cardinality strings, the field with its own small parser.
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","cardinality":"1..1"}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","cardinality":"0..0"}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","cardinality":"2..1"}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","cardinality":"-1..*"}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","cardinality":".."}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","cardinality":"0..99999999999999999999"}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","inGroupCardinality":"0..1"}]}`,

	// SCTIDs and rule strengths, both validated on load.
	`{"domains":[{"attributeId":"not-an-sctid","domainEcl":"*"}]}`,
	`{"domains":[{"attributeId":"","domainEcl":"*"}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":"*","ruleStrengthId":"999"}]}`,

	// ECL that does not parse, and ECL that parses but is expensive.
	`{"domains":[{"attributeId":"363698007","domainEcl":"<< ((("}]}`,
	`{"domains":[{"attributeId":"363698007","domainEcl":""}]}`,
	`{"ranges":[{"attributeId":"363698007","rangeEcl":"GARBAGE"}]}`,

	// Not MRCM, and not JSON.
	`[]`,
	`null`,
	`{"domains":{}}`,
	`{"domains":[null]}`,
	``,
	`{`,
	"\x00\x01\xff\xfe",
}

// scgSeedsForMRCM are the expressions paired with each model seed. They are
// chosen for the SHAPES the validator walks — grouped and ungrouped attributes,
// a nested expression as a value, a concrete value, several focus concepts,
// and a bare concept with no refinement at all, which is the case that once made
// the validator report a missing mandatory attribute for every precoordinated
// code.
var scgSeedsForMRCM = []string{
	"22298006",
	"22298006:363698007=74281007",
	"22298006:{363698007=74281007,363698007=425391005}",
	"22298006:{363698007=74281007}{116676008=55641003}",
	"22298006+73211009:{363698007=74281007}",
	"22298006:363698007=(74281007:363698007=425391005)",
	"22298006:1142139005=#500",
	`22298006:1142139005="a"`,
}
