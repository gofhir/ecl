package oracle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVerdict covers when a run of the differential test is evidence. It runs
// offline: the policy is what H8 found missing, and a policy that is only
// exercised against a live server is not tested at all.
func TestVerdict(t *testing.T) {
	core := Case{Expr: "core"}
	core2 := Case{Expr: "core2"}
	edition := Case{Expr: "edition", Needs: "^ 1"}
	gap := Case{Expr: "gap", Unsupported: true}
	defect := Case{Expr: "defect", ServerMayReject: "a known server defect"}
	r := func(c Case, o Outcome) Result { return Result{Case: c, Outcome: o} }

	for _, tc := range []struct {
		name    string
		results []Result
		min     int
		want    []string // one substring per problem; none for evidence
	}{
		{"every case answered", []Result{r(core, Agreed), r(core2, Agreed), r(edition, Agreed), r(gap, Refused), r(defect, Agreed)}, 2, nil},
		{"the edition lacks a refset", []Result{r(core, Agreed), r(edition, NoData), r(gap, Refused)}, 1, nil},
		{"a known server defect", []Result{r(core, Agreed), r(defect, ServerRejected)}, 1, nil},
		// The floor is about required cases: an edition's agreement or a
		// defect case's cannot stand in for a required case not compared.
		{"another case standing in for a required one", []Result{r(core, Agreed), r(core2, ServerRejected), r(edition, Agreed), r(defect, Agreed)}, 2,
			[]string{"only 1 required case(s) agreed on a non-empty set, 2 required; besides those diverged or failed, did not agree: core2 (rejected by the server)"}},
		// A required case that diverged fails on its own; the floor counts it
		// without listing it again.
		{"a required divergence under the floor", []Result{r(core, Agreed), r(core2, Diverged)}, 2,
			[]string{"did not agree: "}},
		// The declared refusal is not reported as gone: nothing was compared.
		{"nothing reachable", []Result{r(core, Unreachable), r(core2, Unreachable), r(edition, Unreachable), r(gap, Unreachable)}, 2,
			[]string{"4 case(s) server unreachable: core; core2; edition; and 1 more", "only 0 required case(s) agreed"}},
		{"agreement on nothing", []Result{r(core, Vacuous), r(core2, Agreed)}, 1, []string{"1 case(s) vacuous: core"}},
		{"a refusal nobody declared", []Result{r(core, Refused), r(core2, Agreed)}, 1, []string{`"core" was refused here`}},
		{"a declared refusal that went away", []Result{r(core, Agreed), r(gap, Agreed)}, 1, []string{`"gap" is declared unsupported but was answered here (agreed)`}},
		{"a declared refusal that diverged", []Result{r(core, Agreed), r(gap, Diverged)}, 1, []string{`"gap" is declared unsupported but was answered here (diverged)`}},
		{"a declared refusal the server kept from happening", []Result{r(core, Agreed), r(gap, ServerRejected)}, 1, nil},
		{"a declared refusal that failed", []Result{r(core, Agreed), r(gap, Failed)}, 1, nil},
		// Answered here, then rejected by the server: the refusal is gone.
		{"a declared refusal answered here, rejected there", []Result{r(core, Agreed), {Case: gap, Outcome: ServerRejected, AnsweredHere: true}}, 1,
			[]string{`"gap" is declared unsupported but was answered here (rejected by the server)`}},
		// The edition's data is missing for a case outside the floor: no failure.
		{"a known server defect on an edition without the data", []Result{r(core, Agreed), r(defect, NoData)}, 1, nil},
		// An empty answer does not show the refusal went away: an empty focus
		// is answered before the refused construct is reached.
		{"a declared refusal answered empty", []Result{r(core, Agreed), r(gap, Vacuous)}, 1, []string{"1 case(s) vacuous: gap"}},
		{"a known server defect answered empty", []Result{r(core, Agreed), r(defect, Vacuous)}, 1, []string{"1 case(s) vacuous: defect"}},
		{"a case the harness cannot answer", []Result{r(core, NotAnswerable), r(core2, Agreed)}, 1, []string{"1 case(s) not answerable: core"}},
		// Divergences and failures are reported by their own subtest.
		{"a divergence or a failure", []Result{r(core, Diverged), r(core2, Failed), r(edition, Agreed)}, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Verdict(tc.results, tc.min)
			require.Len(t, got, len(tc.want), "%q", got)
			for i, w := range tc.want {
				require.Truef(t, strings.Contains(got[i], w), "problem %d %q lacks %q", i, got[i], w)
			}
		})
	}
}

// TestNotes covers the hints a run gives about its corpus without failing.
func TestNotes(t *testing.T) {
	edition := Case{Expr: "edition", Needs: "^ 1"}
	defect := Case{Expr: "defect", ServerMayReject: "a known server defect"}
	require.Empty(t, Notes([]Result{{Case: defect, Outcome: ServerRejected}, {Case: edition, Outcome: NoData}}))
	notes := Notes([]Result{{Case: defect, Outcome: Agreed}, {Case: edition, Outcome: ServerRejected}})
	require.Len(t, notes, 2)
	require.Contains(t, notes[0], "clear the declaration")
	require.Contains(t, notes[1], "which Needs does not cover")
}

// TestTally covers the run's one-line summary: every outcome, in a fixed order.
func TestTally(t *testing.T) {
	require.Equal(t, "no case", Tally(nil))
	require.Equal(t, "2 agreed, 1 rejected by the server, 1 failed",
		Tally([]Result{{Outcome: Failed}, {Outcome: Agreed}, {Outcome: ServerRejected}, {Outcome: Agreed}}))
}

// TestRequiredAgreements covers the default floor: the cases every edition can
// answer and this library is expected to, which is most of the bundled corpus.
func TestRequiredAgreements(t *testing.T) {
	require.Equal(t, 1, RequiredAgreements([]Case{{Expr: "a"}, {Expr: "b", Needs: "^ 1"}, {Expr: "c", Unsupported: true}, {Expr: "d", ServerMayReject: "x"}}))

	// The README's oracle section and §11.12 of the adversarial analysis state
	// these numbers; change them together.
	require.Equal(t, 31, RequiredAgreements(Corpus))
	require.Len(t, Corpus, 39)
}
