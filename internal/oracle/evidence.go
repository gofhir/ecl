package oracle

import (
	"fmt"
	"strings"
)

// Outcome is what one corpus case showed when compared with the server.
type Outcome int

const (
	// Agreed means both sides returned the same non-empty set. The only outcome
	// that is evidence.
	Agreed Outcome = iota
	// Vacuous means both sides returned the empty set, which two broken
	// implementations agree on as readily as two correct ones.
	Vacuous
	// Diverged means the sets differ.
	Diverged
	// Refused means this library reported the expression ErrUnsupportedFeature.
	Refused
	// NotAnswerable means the expression needs a primitive the harness cannot ask a
	// FHIR terminology API for.
	NotAnswerable
	// ServerRejected means the server would not answer a primitive, or the
	// expression itself.
	ServerRejected
	// NoData means the server's edition has no content for the case's Needs probe.
	NoData
	// Unreachable means the server could not be reached, so nothing was compared.
	Unreachable
	// Failed means the harness could not finish the case: a corpus defect, a
	// time budget exceeded, an error from either side that is none of the
	// above. The case fails on its own.
	Failed
)

var outcomeNames = map[Outcome]string{
	Agreed: "agreed", Vacuous: "vacuous", Diverged: "diverged", Refused: "refused here",
	NotAnswerable: "not answerable", ServerRejected: "rejected by the server", NoData: "no data on the server",
	Unreachable: "server unreachable", Failed: "failed",
}

func (o Outcome) String() string { return outcomeNames[o] }

// Result is one case's outcome.
type Result struct {
	Case    Case
	Outcome Outcome

	// AnsweredHere is set when this library evaluated the case, whatever
	// the server then did: for a case declared Unsupported, that alone shows
	// the refusal went away.
	AnsweredHere bool
}

// RequiredAgreements is how many cases a run must agree on, non-empty, by
// default: the required ones (see Required). A server that legitimately
// answers fewer can be given another number through the test's
// ECL_ORACLE_MIN_AGREED.
func RequiredAgreements(corpus []Case) int {
	n := 0
	for _, c := range corpus {
		if Required(c) {
			n++
		}
	}
	return n
}

// Required reports whether a case counts toward the floor: it names data every
// SNOMED CT edition has, this library is expected to answer it, and no known
// server defect keeps it from being compared. Only these cases' agreements
// count, so an agreement elsewhere cannot stand in for a required case that
// was not compared.
func Required(c Case) bool {
	return c.Needs == "" && !c.Unsupported && c.ServerMayReject == ""
}

// Verdict returns why a run is not the evidence a green run claims to be, or
// nothing when it is. A divergence is reported by its own case and is not
// repeated here.
//
// A skip used to be neutral, so a run in which nothing at all was compared —
// the server down, every primitive rejected, every case agreeing on the empty
// set — passed. Now:
//
//   - an unreachable server fails the run: the comparison was asked for;
//   - a vacuous agreement is a corpus defect, since the corpus is chosen for
//     non-empty answers;
//   - a refusal must be declared (Case.Unsupported), and a declared one must
//     still happen, so the corpus says what it covers;
//   - a case the harness cannot answer is a corpus defect: that does not
//     depend on the server (data the server's edition lacks is NoData);
//   - fewer than minAgreed non-empty agreements on required cases fails the
//     run, whatever the reasons for the rest; the required cases that did not
//     agree are named.
//
// A case that failed or diverged is reported by its own subtest.
func Verdict(results []Result, minAgreed int) []string {
	var (
		problems, missing []string
		agreed            int
		by                = map[Outcome][]string{}
	)
	for _, r := range results {
		by[r.Outcome] = append(by[r.Outcome], r.Case.Expr)
		switch {
		case !Required(r.Case):
		case r.Outcome == Agreed:
			agreed++
		case r.Outcome == Diverged || r.Outcome == Failed:
			// Reported by its own subtest; counted, not repeated.
		default:
			missing = append(missing, fmt.Sprintf("%s (%s)", r.Case.Expr, r.Outcome))
		}
		switch {
		case r.Case.Unsupported && (r.AnsweredHere || r.Outcome == Agreed || r.Outcome == Diverged):
			// Only an answer shows the refusal went away. An unreachable
			// server or a rejected primitive says nothing about it, and nor
			// does an empty answer: an empty focus is answered before the
			// construct that is refused is reached.
			problems = append(problems, fmt.Sprintf("%q is declared unsupported but was answered here (%s); clear Unsupported", r.Case.Expr, r.Outcome))
		case !r.Case.Unsupported && r.Outcome == Refused:
			problems = append(problems, fmt.Sprintf("%q was refused here; declare it Unsupported or replace it", r.Case.Expr))
		}
	}
	for _, o := range []Outcome{Unreachable, Vacuous, NotAnswerable} {
		if cases := by[o]; len(cases) > 0 {
			problems = append(problems, fmt.Sprintf("%d case(s) %s: %s", len(cases), o, listed(cases)))
		}
	}
	if agreed < minAgreed {
		problems = append(problems, fmt.Sprintf("only %d required case(s) agreed on a non-empty set, %d required; besides those diverged or failed, did not agree: %s",
			agreed, minAgreed, listed(missing)))
	}
	return problems
}

// Notes returns what a run suggests about the corpus without failing it: a
// declaration that may be stale on this server, or a rejection that no
// declaration accounts for but that does not count toward the floor either.
func Notes(results []Result) []string {
	var notes []string
	for _, r := range results {
		switch {
		case r.Case.ServerMayReject != "" && r.Outcome == Agreed:
			notes = append(notes, fmt.Sprintf("%q is declared ServerMayReject but this server answered it; if the defect is gone, clear the declaration so the case counts", r.Case.Expr))
		case r.Case.ServerMayReject == "" && r.Case.Needs != "" && r.Outcome == ServerRejected:
			notes = append(notes, fmt.Sprintf("%q was rejected by the server, which Needs does not cover: it is for an edition without the data", r.Case.Expr))
		}
	}
	return notes
}

// listed names a few cases; a run where every case failed alike is diagnosed
// from the count, not by reading all of them.
func listed(cases []string) string {
	const maxListed = 3
	if len(cases) <= maxListed {
		return strings.Join(cases, "; ")
	}
	return fmt.Sprintf("%s; and %d more", strings.Join(cases[:maxListed], "; "), len(cases)-maxListed)
}

// Tally counts the outcomes of a run, in a fixed order, e.g. "35 agreed, 1
// refused here, 3 rejected by the server".
func Tally(results []Result) string {
	n := map[Outcome]int{}
	for _, r := range results {
		n[r.Outcome]++
	}
	var parts []string
	for o := Agreed; o <= Failed; o++ {
		if n[o] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n[o], o))
		}
	}
	if len(parts) == 0 {
		return "no case"
	}
	return strings.Join(parts, ", ")
}
