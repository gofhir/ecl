package mrcm

import (
	"context"
	"testing"
	"time"

	"github.com/gofhir/ecl/scg"
)

// FuzzLoadFromBytes asserts that loading an MRCM document never panics, and —
// the part worth the trouble — that a model which LOADS is a model the validator
// can USE.
//
// The two halves matter for different reasons. LoadFromBytes takes a document
// from wherever an operator got it: a release package, a hand-edited file, an
// endpoint. And a model that parses but then makes Validate panic is the worse
// bug, because the failure surfaces far from its cause — the loader is what is
// supposed to reject a rule the rest of the package cannot handle, and every
// field it lets through becomes an assumption downstream.
//
// So each accepted model is run against a fixed expression. The verdict is not
// checked: arbitrary rules produce arbitrary verdicts and asserting one would
// just encode the fuzzer's output. What is checked is that a verdict comes back
// at all, and that Result is coherent when it does.
func FuzzLoadFromBytes(f *testing.F) {
	for _, seed := range mrcmSeeds {
		f.Add([]byte(seed))
	}

	// Parsed here rather than through mustParseSCG, which takes a *testing.T.
	expr, err := scg.Parse("22298006:{363698007=74281007,363698007=425391005}")
	if err != nil {
		f.Fatalf("the fixed expression must parse: %v", err)
	}
	provider := newTestProvider()

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("LoadFromBytes or Validate panicked on %d bytes: %v\n%q", len(data), r, data)
			}
		}()

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

		start := time.Now()
		res, err := Validate(context.Background(), expr, model, provider)
		elapsed := time.Since(start)

		if err != nil {
			// A provider or context failure would be an error; with this stub
			// provider there is none to have, so an error here means a rule the
			// loader accepted reached code that could not handle it. That is the
			// boundary this target exists to police.
			t.Errorf("a model the loader ACCEPTED made Validate fail: %v\n%q", err, data)
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
			t.Errorf("Validate took %s on a %d-byte model:\n%q",
				elapsed.Round(time.Millisecond), len(data), data)
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
