package oracle_test

import (
	"context"
	"errors"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/internal/oracle"
)

// envURL names the terminology server to compare against. The test skips when it
// is unset, so `go test ./...` stays hermetic and offline.
//
//	ECL_ORACLE_URL=https://r4.ontoserver.csiro.au/fhir go test ./internal/oracle/ -v
const envURL = "ECL_ORACLE_URL"

// TestDifferential evaluates every corpus expression twice — here and on the
// server — and compares the concept sets.
//
// Read a failure carefully before changing anything. The server is another
// implementation, not the specification, so a divergence has four candidate
// explanations and only one of them is "this library is wrong":
//
//  1. This library composes the primitives incorrectly. Most likely, and the
//     reason the harness exists.
//  2. The server is wrong. It happens; it is software.
//  3. The specification admits both readings. Then the fix is a documented
//     decision, not a code change.
//  4. The harness fed this library different facts than the server used. Both
//     sides reach the same data through the same two operations, so this is the
//     least likely explanation — but it is the one that produced the harness's
//     own first bug, so rule it out rather than assume it.
//
// A case this library reports as unsupported is NOT a divergence: it is the
// fail-loudly policy working, and is reported as skipped with the reason.
func TestDifferential(t *testing.T) {
	baseURL := os.Getenv(envURL)
	if baseURL == "" {
		t.Skipf("set %s to a FHIR R4 terminology endpoint to run the differential test, e.g. %s=https://r4.ontoserver.csiro.au/fhir",
			envURL, envURL)
	}

	client := oracle.NewClient(baseURL)
	provider := oracle.NewProvider(client)

	var agreed, vacuous, skipped int
	start := time.Now()

	for _, c := range oracle.Corpus {
		t.Run(c.Expr, func(t *testing.T) {
			// A generous per-case budget: a refinement fetches the relationships of
			// every focus concept in batches, and a public server is not a local
			// database.
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()

			expr, err := ecl.Parse(c.Expr)
			if err != nil {
				t.Fatalf("the corpus expression does not parse, which is a corpus defect: %v", err)
			}

			ours, ourErr := ecl.Evaluate(ctx, expr, provider)

			switch {
			case errors.Is(ourErr, ecl.ErrUnsupportedFeature):
				skipped++
				t.Skipf("this library reports the construct unsupported, which is a documented gap rather than a divergence:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrNotAnswerable):
				skipped++
				t.Skipf("outside what a terminology API can supply as facts:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrUnreachable):
				skipped++
				t.Skipf("server unreachable, so nothing was compared:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrTooLarge):
				t.Fatalf("corpus defect — bound the expression instead of raising the cap:\n  %v", ourErr)
			case ourErr != nil:
				t.Fatalf("evaluating here failed: %v", ourErr)
			}

			theirCodes, err := client.ExpandECL(ctx, c.Expr)
			if err != nil {
				if errors.Is(err, oracle.ErrServerRejected) {
					skipped++
					t.Skipf("the server would not evaluate the expression, so there is nothing to compare against:\n  %v", err)
				}
				t.Fatalf("asking the server for the whole expression failed: %v", err)
			}
			theirs := ecl.NewSetFromSlice(theirCodes)

			onlyOurs := diff(ours, theirs)
			onlyTheirs := diff(theirs, ours)
			if len(onlyOurs) == 0 && len(onlyTheirs) == 0 {
				agreed++
				// Agreeing on the empty set decides nothing: two implementations
				// that both return nothing agree for any reason at all, including
				// both being broken. It is reported separately so it is never
				// counted as evidence, and a case that lands here should be
				// replaced with one whose result is non-empty.
				if ours.Len() == 0 {
					vacuous++
					t.Logf("VACUOUS: both sides returned the empty set, which decides nothing — replace this case (exercises: %s)", c.Exercises)
					return
				}
				t.Logf("agreed on %d concepts (exercises: %s)", ours.Len(), c.Exercises)
				return
			}

			t.Errorf(`DIVERGENCE on %s

  exercises:    %s
  here:         %d concepts
  server:       %d concepts
  only here:    %d %v
  only server:  %d %v

Do not "fix" this library to match without deciding which is right; see the
doc comment on TestDifferential for the four candidate explanations.`,
				c.Expr, c.Exercises,
				ours.Len(), theirs.Len(),
				len(onlyOurs), sample(onlyOurs),
				len(onlyTheirs), sample(onlyTheirs))
		})
	}

	t.Logf("%d agreed (%d of them vacuously, on the empty set), %d skipped, of %d cases; %d HTTP requests in %s",
		agreed, vacuous, skipped, len(oracle.Corpus), client.Requests, time.Since(start).Round(time.Second))
}

// diff returns the members of a that are absent from b, sorted.
func diff(a, b ecl.Set) []string {
	var out []string
	a.Iter(func(id string) bool {
		if !b.Contains(id) {
			out = append(out, id)
		}
		return true
	})
	sort.Strings(out)
	return out
}

// sample caps how much of a difference is printed. A divergence of hundreds of
// concepts is diagnosed from its shape, not by reading every id.
func sample(ids []string) []string {
	const maxShown = 10
	if len(ids) <= maxShown {
		return ids
	}
	return append(append([]string(nil), ids[:maxShown]...), "…")
}
