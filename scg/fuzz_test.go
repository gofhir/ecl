package scg_test

import (
	"testing"
	"time"

	"github.com/gofhir/ecl/scg"
)

// FuzzParse asserts that Parse never panics, never returns a nil expression with
// a nil error, and never runs away, for any input at all.
//
// Parse takes untrusted input in the same places ecl.Parse does — a
// post-coordinated expression arriving in a FHIR Coding, a normal form read back
// from a terminology server — and it is hand-written recursive descent, so the
// failure modes worth guarding are a panic on a shape the grammar did not
// anticipate and unbounded recursion on deep nesting.
//
// Measured before this target existed, it was clean: 190 KB with 10,000
// attributes in 1 ms, and 500,000 levels of nesting on a 10 MB input without
// exhausting the stack, since Go grows a goroutine stack on demand. That is a
// measurement of one afternoon, not a property. This is the property.
//
// Run a longer pass with:
//
//	make fuzz FUZZTIME=10m
func FuzzParse(f *testing.F) {
	for _, seed := range scgSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Parse panicked on %d bytes of input: %v\n%q", len(input), r, input)
			}
		}()

		start := time.Now()
		expr, err := scg.Parse(input)
		elapsed := time.Since(start)

		switch {
		case err != nil && expr != nil:
			t.Errorf("Parse returned both an expression and an error, so a caller checking\n"+
				"only one of them gets a wrong answer either way: %v\n%q", err, input)
		case err == nil && expr == nil:
			t.Errorf("Parse returned a nil expression with a nil error for %q", input)
		case err == nil && len(expr.FocusConcepts) == 0:
			// An expression with no focus concept is not one: every SCG
			// expression names at least one, and a caller reading
			// FocusConcepts[0] would panic on it.
			t.Errorf("Parse accepted %q as valid but produced no focus concept", input)
		}

		// Far above any legitimate parse, so a loaded CI machine does not trip
		// it. It is here to catch a NEW pathological shape, which is the whole
		// reason for the file.
		if elapsed > 5*time.Second {
			t.Errorf("Parse took %s on %d bytes, which no valid expression should:\n%q",
				elapsed.Round(time.Millisecond), len(input), input)
		}
	})
}

// scgSeeds is the seed corpus.
//
// The last three are real: they are long normal forms returned by
// r4.ontoserver.csiro.au for actual SNOMED CT concepts, which exercise nesting,
// multiple focus concepts joined by "+", several relationship groups and terms
// containing punctuation — combinations this file would not have thought to
// invent. Seeds are not required to parse; several below are deliberately
// malformed, and a cleanly rejected input is a perfectly good starting point for
// mutation.
var scgSeeds = []string{
	"",
	" ",
	"22298006",
	"22298006 |Myocardial infarction|",
	"=== 22298006",
	"<<< 22298006",
	"22298006 : 363698007 = 74281007",
	"22298006:{363698007=74281007,116676008=55641003}",
	"22298006:{363698007=74281007}{116676008=55641003}",
	"373873005:246093002=(386053000:363698007=39057004)",
	"27658006:411116001=#20.5",
	"27658006:411116001=true",
	`27658006:411116001="text"`,
	"421720008+322236009",

	// Malformed: the error paths deserve seeds too.
	"22298006:",
	"22298006:363698007=",
	"22298006:{363698007=74281007",
	"22298006:}",
	":363698007=74281007",
	"+",
	"(((",
	"22298006 garbage",

	// Structural characters inside data.
	"22298006 |Myocardial (infarction)|",
	"22298006 |a:b,c{d}|",
	`22298006:411116001="a|b:c"`,
	"22298006 |Crohn’s disease|",
	"\x00\x01\xff\xfe",

	// Real long normal forms from a terminology server.
	"=== 414545008|Ischemic heart disease|+251061000|Myocardial necrosis|:{116676008|Associated morphology|=55641003|Infarct|,363698007|Finding site|=74281007|Myocardium structure|}",
	"<<< 73211009|Diabetes mellitus|:{363698007|Finding site|=113331007|Structure of endocrine system|}",
	"=== 40172005|Cardiac complication|+57054005|Acute myocardial infarction|:{42752001|Due to|=(63739005|Coronary occlusion|:{363698007|Finding site|=50018008|Left coronary artery structure|,116676008|Associated morphology|=50173008|Complete obstruction|})},{263502005|Clinical course|=424124008|Sudden onset AND/OR short duration|},{363698007|Finding site|=74281007|Myocardium structure|,116676008|Associated morphology|=55470003|Acute infarct|}",
}
