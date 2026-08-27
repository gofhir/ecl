package ecl_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofhir/ecl/ecl"
)

// FuzzParse asserts that Parse never panics and never returns a nil tree with a
// nil error, for any input at all.
//
// This target should have existed from the first commit. Its absence is how the
// parser came to spend 7.6 seconds and 92 million allocations on a 2.2 KB
// expression without anyone noticing — a five-minute poke at pathological shapes
// found it, which is exactly the work a fuzzer does continuously and for free.
//
// Parse is reachable from untrusted input in the use this library is built for: a
// FHIR server taking an expression from a URL. So the properties worth asserting
// are the ones a hostile caller would attack.
//
// Run a short pass with:
//
//	go test ./ecl/ -run '^$' -fuzz FuzzParse -fuzztime 60s
//
// The seed corpus is every official SNOMED International example plus a set of
// shapes chosen to be awkward. Seeds are not required to parse: an input that is
// rejected cleanly is a perfectly good starting point for mutation, and several
// of the seeds below are deliberately invalid.
func FuzzParse(f *testing.F) {
	for _, seed := range fuzzSeeds(f) {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// A panic here fails the test on its own; the point of the explicit
		// recover is the message, which names the input that caused it.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Parse panicked on %d bytes of input: %v\n%q", len(input), r, input)
			}
		}()

		start := time.Now()
		expr, err := ecl.Parse(input)
		elapsed := time.Since(start)

		switch {
		case err != nil && expr != nil:
			t.Errorf("Parse returned both a tree and an error, so a caller checking\n"+
				"only one of them gets a wrong answer either way: %v\n%q", err, input)
		case err == nil && expr == nil:
			// The dangerous direction. A nil tree with a nil error is what the
			// old EOF handling produced for "11687002 GARBAGE": the caller
			// evaluates a truncated or absent constraint and gets a plausible,
			// wrong concept set.
			t.Errorf("Parse returned a nil tree with a nil error for %q", input)
		}

		// A time limit inside a fuzz target is a blunt instrument — a loaded CI
		// machine can be slow for reasons that have nothing to do with the input
		// — so it is set far above any legitimate parse. Both input limits keep
		// the real work bounded; this only catches a NEW pathological shape,
		// which is the class of bug that motivated the whole file.
		if elapsed > 5*time.Second {
			t.Errorf("Parse took %s on %d bytes, which no valid expression should:\n%q",
				elapsed.Round(time.Millisecond), len(input), input)
		}
	})
}

// fuzzSeeds returns the seed corpus: the vendored official examples, plus shapes
// picked to stress the axes that turned out to matter.
func fuzzSeeds(f *testing.F) []string {
	f.Helper()

	seeds := []string{
		"",
		" ",
		"404684003",
		"<< 404684003",
		"* : 363698007 = 74281007",
		"* : { 363698007 = 74281007, 116676008 = 55641003 }",
		"* : [0..0] 363698007 = *",
		"* : R 363698007 = 22298006",
		`<< 404684003 {{ D term = "heart" }}`,
		`<< 404684003 {{ D term = ("a" "b"), language = es }}`,
		`<< 404684003 {{ C effectiveTime >= "20240131" }}`,
		"^ 900000000000497000",

		// Unbalanced and malformed: the error paths deserve seeds too.
		"(((",
		")))",
		"{{{",
		"* : ",
		"404684003 GARBAGE",
		"* : 363698007 =",
		"[1..0] 363698007 = *",

		// Data that contains structural characters. These are why the nesting
		// prescan skips quoted strings and |terms|; counting brackets inside
		// them would reject valid expressions.
		`404684003 |Finding (site)|`,
		`* {{ D term = "a (b" }}`,
		`* {{ D term = "a\"b" }}`,
		`404684003 |a|b|c|`,

		// Non-ASCII, which broke the trailing-input message once by being sliced
		// on a byte offset instead of a rune index.
		"404684003 |ááá| GARBAGE",
		"404684003 |Crohn’s disease|",
		"\x00\x01\xff\xfe",

		// Just past and just under the nesting limit.
		strings.Repeat("(", ecl.MaxNestingDepth+1) + "404684003",
		strings.Repeat("(", 4) + "404684003" + strings.Repeat(")", 4),
	}

	// Every official example, so the fuzzer starts from input the specification
	// itself declares valid rather than only from what this file could imagine.
	root := officialExamplesDir
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".txt") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		seeds = append(seeds, strings.TrimSpace(string(raw)))
		return nil
	})
	if err != nil {
		f.Fatalf("reading the official example corpus for seeds: %v", err)
	}

	return seeds
}
