package oracle_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/internal/oracle"
)

// envURL names the terminology server to compare against. The test skips when it
// is unset, so `go test ./...` stays hermetic and offline.
//
//	ECL_ORACLE_URL=https://r4.ontoserver.csiro.au/fhir go test ./internal/oracle/ -v
const envURL = "ECL_ORACLE_URL"

// envMinAgreed overrides how many required cases must agree on a non-empty
// set, for a server that legitimately answers fewer than the default; see
// oracle.RequiredAgreements.
const envMinAgreed = "ECL_ORACLE_MIN_AGREED"

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
// fail-loudly policy working, and the corpus declares it (Case.Unsupported).
//
// A skipped case is not evidence, though, and a run that compared nothing used
// to pass. Every outcome is recorded, and oracle.Verdict fails the run when it
// is not the evidence a green run claims: see it for the rules. A run filtered
// with -run checks every case it ran, but not the floor, which is about the
// whole corpus.
func TestDifferential(t *testing.T) {
	baseURL := os.Getenv(envURL)
	if baseURL == "" {
		t.Skipf("set %s to a FHIR R4 terminology endpoint to run the differential test, e.g. %s=https://r4.ontoserver.csiro.au/fhir",
			envURL, envURL)
	}

	minAgreed := oracle.RequiredAgreements(oracle.Corpus)
	if v := os.Getenv(envMinAgreed); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > minAgreed {
			t.Fatalf("%s=%q: want a number of cases from 1 to %d, the required ones", envMinAgreed, v, minAgreed)
		}
		minAgreed = n
	}

	client := oracle.NewClient(baseURL)
	provider := oracle.NewProvider(client)

	results := make([]oracle.Result, 0, len(oracle.Corpus))
	start := time.Now()

	for _, c := range oracle.Corpus {
		// Recorded however the subtest ends: a case that stops before an
		// outcome is known has failed, and says why itself. One that did not
		// run, filtered out by -run, is not recorded.
		outcome, ran, answered := oracle.Failed, false, false
		t.Run(c.Expr, func(t *testing.T) {
			ran = true
			// A generous per-case budget: a refinement fetches the relationships of
			// every focus concept in batches, and a public server is not a local
			// database.
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()

			expr, err := ecl.Parse(c.Expr)
			if err != nil {
				t.Fatalf("the corpus expression does not parse, which is a corpus defect: %v", err)
			}

			// A case guarded by Needs is skipped when this server's edition does
			// not carry the data it names, rather than reported as a divergence.
			if c.Needs != "" {
				probe, err := client.ExpandECL(ctx, c.Needs)
				if err != nil {
					if outcome = serverOutcome(ctx, err); outcome == oracle.Failed {
						t.Fatalf("probing for the data this case needs (%s) failed: %v", c.Needs, err)
					}
					t.Skipf("could not probe for the data this case needs (%s): %v", c.Needs, err)
				}
				if len(probe) == 0 {
					outcome = oracle.NoData
					t.Skipf("this server has no content for %s, so the case says nothing about the evaluator", c.Needs)
				}
			}

			ours, ourErr := ecl.Evaluate(ctx, expr, provider)

			switch {
			case ourErr != nil && ctx.Err() != nil:
				// The budget, not the server: the client reports a deadline
				// inside a request as unreachable.
				t.Fatalf("the case ran out of its time budget: %v", ourErr)
			case errors.Is(ourErr, ecl.ErrUnsupportedFeature):
				outcome = oracle.Refused
				t.Skipf("this library reports the construct unsupported, which is a documented gap rather than a divergence:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrNoData):
				outcome = oracle.NoData
				t.Skipf("this server's edition lacks data a primitive needs:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrNotAnswerable):
				outcome = oracle.NotAnswerable
				t.Skipf("outside what a terminology API can supply as facts:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrUnreachable) || errors.Is(ourErr, oracle.ErrServerRejected):
				// The server could not answer one of the PRIMITIVES, so this side
				// has no result to compare. It is not a divergence and it is not
				// this library's defect — but it is worth reading, because it names
				// what the server would not answer.
				if outcome = serverOutcome(ctx, ourErr); outcome == oracle.Failed {
					t.Fatalf("asking the server for a primitive timed out: %v", ourErr)
				}
				t.Skipf("the server could not answer a primitive this case needs:\n  %v", ourErr)
			case errors.Is(ourErr, oracle.ErrTooLarge):
				t.Fatalf("corpus defect — bound the expression instead of raising the cap:\n  %v", ourErr)
			case ourErr != nil:
				t.Fatalf("evaluating here failed: %v", ourErr)
			}

			answered = true
			theirCodes, err := client.ExpandECL(ctx, c.Expr)
			if err != nil {
				if outcome = serverOutcome(ctx, err); outcome != oracle.Failed {
					t.Skipf("the server would not evaluate the expression, so there is nothing to compare against:\n  %v", err)
				}
				t.Fatalf("asking the server for the whole expression failed: %v", err)
			}
			theirs := ecl.NewSetFromSlice(theirCodes)

			onlyOurs := diff(ours, theirs)
			onlyTheirs := diff(theirs, ours)
			if len(onlyOurs) == 0 && len(onlyTheirs) == 0 {
				// Agreeing on the empty set decides nothing: two implementations
				// that both return nothing agree for any reason at all, including
				// both being broken. It is never counted as evidence, and the
				// run fails until the case is replaced with a non-empty one.
				if ours.Len() == 0 {
					outcome = oracle.Vacuous
					t.Logf("VACUOUS: both sides returned the empty set, which decides nothing — replace this case (exercises: %s)", c.Exercises)
					return
				}
				outcome = oracle.Agreed
				t.Logf("agreed on %d concepts (exercises: %s)", ours.Len(), c.Exercises)
				return
			}

			outcome = oracle.Diverged
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
		if ran {
			results = append(results, oracle.Result{Case: c, Outcome: outcome, AnsweredHere: answered})
		}
	}

	t.Logf("%s, of %d cases (%d required to agree); %d HTTP requests in %s",
		oracle.Tally(results), len(oracle.Corpus), minAgreed, client.Requests, time.Since(start).Round(time.Second))
	if len(results) == 0 {
		t.Fatalf("no case ran, so nothing was compared; check the -run filter")
	}
	if len(results) < len(oracle.Corpus) {
		t.Logf("%d of %d cases ran, so the floor is not checked (nor %s, if set)", len(results), len(oracle.Corpus), envMinAgreed)
		minAgreed = 0
	}
	for _, n := range oracle.Notes(results) {
		t.Logf("note: %s", n)
	}
	for _, p := range oracle.Verdict(results, minAgreed) {
		t.Errorf("not evidence: %s", p)
	}
}

// serverOutcome classifies an error from the server met before anything was
// compared: a skip when the server is unreachable or would not answer, and a
// failure for anything else. An expansion over the cap is a corpus defect,
// and a timeout — the case's budget, or one request's — is not the server
// being down, although the client reports both as unreachable.
func serverOutcome(ctx context.Context, err error) oracle.Outcome {
	var timeout interface{ Timeout() bool }
	switch {
	case ctx.Err() != nil:
		return oracle.Failed
	case errors.As(err, &timeout) && timeout.Timeout():
		return oracle.Failed
	case errors.Is(err, oracle.ErrUnreachable):
		return oracle.Unreachable
	case errors.Is(err, oracle.ErrServerRejected):
		return oracle.ServerRejected
	}
	return oracle.Failed
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

// TestServerOutcome covers how a server error met before a comparison is
// classified, offline: a timeout is not the server being down, although the
// client reports it as unreachable.
func TestServerOutcome(t *testing.T) {
	live := context.Background()
	done, cancel := context.WithCancel(live)
	cancel()
	timeout := fmt.Errorf("%w: %w", oracle.ErrUnreachable, &url.Error{Op: "Post", URL: "x", Err: timeoutError{}})
	for _, tc := range []struct {
		name string
		ctx  context.Context
		err  error
		want oracle.Outcome
	}{
		{"down", live, fmt.Errorf("%w: connection refused", oracle.ErrUnreachable), oracle.Unreachable},
		{"rejected", live, fmt.Errorf("%w: HTTP 500", oracle.ErrServerRejected), oracle.ServerRejected},
		{"one request timed out", live, timeout, oracle.Failed},
		{"the case's budget", done, fmt.Errorf("%w: context canceled", oracle.ErrUnreachable), oracle.Failed},
		{"over the cap", live, fmt.Errorf("%w: 20000 codes", oracle.ErrTooLarge), oracle.Failed},
	} {
		require.Equalf(t, tc.want, serverOutcome(tc.ctx, tc.err), tc.name)
	}
}

type timeoutError struct{}

func (timeoutError) Error() string { return "timeout" }
func (timeoutError) Timeout() bool { return true }
