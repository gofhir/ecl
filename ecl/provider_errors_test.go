package ecl_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// TestEvaluate_EveryProviderFailureIsErrProvider covers H5 of
// docs/analisis-adversarial-2026-10-12.md: ErrProvider is documented as the way
// to tell a backend failure (503) from a bad expression (400), so it has to hold
// for every method of the contract. TestErrProvider checks one representative
// method; this checks every required method the evaluator calls. The optional
// capabilities (batch lookups, negated and id description filters, dialect
// aliases, projection, inbound relationships) are outside it: they mostly wrap
// already, and each needs its own provider to be reached.
//
// Each case fails ONE method and evaluates an expression that reaches it. The
// provider records which methods ran, so a case whose expression stops reaching
// its method fails loudly instead of passing on another method's error. The
// wrapper implements only the required interface: with no optional capability
// to type-assert, the evaluator falls back to the base methods under test.
//
// RelationshipSources is absent on purpose: the evaluator never calls it. The
// reverse flag goes through RelationshipTargets with the value set as the
// sources (evaluator.go, "reverse attribute lookup"), so there is no failure of
// it to classify — though every DataProvider still has to implement it.
//
// A case covers the call site its expression reaches, not every call site of
// the method: Descendants, for one, is also called by the top/bottom operators,
// which already wrap. Dropping a marker therefore says that path is fixed; the
// fix should wrap at one place every call goes through, not per call site.
//
// A case marked H5 is a strict expected failure, like those in
// semantics_test.go: it passes while the cause comes back unwrapped, and fails
// once ErrProvider is added, so the fix has to drop the marker.
func TestEvaluate_EveryProviderFailureIsErrProvider(t *testing.T) {
	base := standardProvider(t) // read-only, so one copy serves every case
	for _, tc := range []struct {
		method, expr, defect string
	}{
		{method: "Descendants", expr: `< 404684003`, defect: "H5"},
		{method: "Ancestors", expr: `> 22298006`, defect: "H5"},
		{method: "Children", expr: `<! 404684003`, defect: "H5"},
		{method: "Parents", expr: `>! 22298006`, defect: "H5"},
		{method: "ConceptExists", expr: `22298006`, defect: "H5"},
		{method: "AllConcepts", expr: `*`, defect: "H5"},
		{method: "RelationshipTargets", expr: `22298006 . 363698007`, defect: "H5"},
		{method: "ConcreteValues", expr: `<< 404684003 : 1142139005 = #5`},
		{method: "PropertiesByGroup", expr: `<< 404684003 : 363698007 = 74281007`},
		{method: "MatchDescription", expr: `<< 404684003 {{ D term = "heart" }}`},
		{method: "FilterConcepts", expr: `<< 404684003 {{ C active = true }}`},
		{method: "RefsetMembers", expr: `^ 900000000000497000`},
		{method: "RefsetsContainingMembers", expr: `^R 22298006`, defect: "H5"},
		{method: "HistoricalAssociations", expr: `22298006 {{ +HISTORY }}`},
		{method: "ResolveIdentifier", expr: `LOINC#1234-5`, defect: "H5"},
		{method: "MatchDialect", expr: `<< 404684003 {{ D dialectId = 900000000000508004 }}`},
		{method: "RefsetMembersFiltered", expr: `^ 447562003 {{ M mapGroup = "2" }}`},
	} {
		t.Run(tc.method, func(t *testing.T) {
			p := &methodFailingProvider{base: base, fail: tc.method, called: map[string]bool{}}
			_, err := evaluate(p, tc.expr)
			require.Truef(t, p.called[tc.method], "%q never called %s, so the case tests nothing; pick an expression that reaches it", tc.expr, tc.method)
			require.ErrorIsf(t, err, errBackendDown, "%q must keep the provider's own cause", tc.expr)

			wrapped := errors.Is(err, ecl.ErrProvider)
			switch {
			case tc.defect == "":
				require.Truef(t, wrapped, "a failing %s surfaces without ErrProvider: %v", tc.method, err)
			case wrapped:
				t.Fatalf("a failing %s now surfaces as ErrProvider, so defect %s looks fixed: drop its marker", tc.method, tc.defect)
			default:
				t.Logf("KNOWN DEFECT %s: a failing %s surfaces without ErrProvider: %v", tc.defect, tc.method, err)
			}
		})
	}
}

// methodFailingProvider delegates to base and fails the one method named by
// fail, with errBackendDown from review_regressions_test.go.
type methodFailingProvider struct {
	base   ecl.DataProvider
	fail   string
	called map[string]bool
}

func (f *methodFailingProvider) hit(method string) error {
	f.called[method] = true
	if method == f.fail {
		return errBackendDown
	}
	return nil
}

func (f *methodFailingProvider) Descendants(ctx context.Context, ids []string, self bool) (ecl.Set, error) {
	if err := f.hit("Descendants"); err != nil {
		return nil, err
	}
	return f.base.Descendants(ctx, ids, self)
}

func (f *methodFailingProvider) Ancestors(ctx context.Context, ids []string, self bool) (ecl.Set, error) {
	if err := f.hit("Ancestors"); err != nil {
		return nil, err
	}
	return f.base.Ancestors(ctx, ids, self)
}

func (f *methodFailingProvider) Children(ctx context.Context, ids []string, self bool) (ecl.Set, error) {
	if err := f.hit("Children"); err != nil {
		return nil, err
	}
	return f.base.Children(ctx, ids, self)
}

func (f *methodFailingProvider) Parents(ctx context.Context, ids []string, self bool) (ecl.Set, error) {
	if err := f.hit("Parents"); err != nil {
		return nil, err
	}
	return f.base.Parents(ctx, ids, self)
}

func (f *methodFailingProvider) ConceptExists(ctx context.Context, ids []string) (ecl.Set, error) {
	if err := f.hit("ConceptExists"); err != nil {
		return nil, err
	}
	return f.base.ConceptExists(ctx, ids)
}

func (f *methodFailingProvider) AllConcepts(ctx context.Context) (ecl.Set, error) {
	if err := f.hit("AllConcepts"); err != nil {
		return nil, err
	}
	return f.base.AllConcepts(ctx)
}

func (f *methodFailingProvider) RelationshipTargets(ctx context.Context, sources, types ecl.Set) (ecl.Set, error) {
	if err := f.hit("RelationshipTargets"); err != nil {
		return nil, err
	}
	return f.base.RelationshipTargets(ctx, sources, types)
}

func (f *methodFailingProvider) RelationshipSources(ctx context.Context, targets, types ecl.Set) (ecl.Set, error) {
	if err := f.hit("RelationshipSources"); err != nil {
		return nil, err
	}
	return f.base.RelationshipSources(ctx, targets, types)
}

func (f *methodFailingProvider) ConcreteValues(ctx context.Context, source, typeID string) ([]ecl.ConcreteValue, error) {
	if err := f.hit("ConcreteValues"); err != nil {
		return nil, err
	}
	return f.base.ConcreteValues(ctx, source, typeID)
}

func (f *methodFailingProvider) PropertiesByGroup(ctx context.Context, id string) (map[int][]ecl.Relationship, error) {
	if err := f.hit("PropertiesByGroup"); err != nil {
		return nil, err
	}
	return f.base.PropertiesByGroup(ctx, id)
}

func (f *methodFailingProvider) MatchDescription(ctx context.Context, opts ecl.DescriptionFilterOpts) (ecl.Set, error) {
	if err := f.hit("MatchDescription"); err != nil {
		return nil, err
	}
	return f.base.MatchDescription(ctx, opts)
}

func (f *methodFailingProvider) FilterConcepts(ctx context.Context, concepts ecl.Set, opts ecl.ConceptFilterOpts) (ecl.Set, error) {
	if err := f.hit("FilterConcepts"); err != nil {
		return nil, err
	}
	return f.base.FilterConcepts(ctx, concepts, opts)
}

func (f *methodFailingProvider) RefsetMembers(ctx context.Context, ids []string) (ecl.Set, error) {
	if err := f.hit("RefsetMembers"); err != nil {
		return nil, err
	}
	return f.base.RefsetMembers(ctx, ids)
}

func (f *methodFailingProvider) RefsetsContainingMembers(ctx context.Context, ids []string) (ecl.Set, error) {
	if err := f.hit("RefsetsContainingMembers"); err != nil {
		return nil, err
	}
	return f.base.RefsetsContainingMembers(ctx, ids)
}

func (f *methodFailingProvider) HistoricalAssociations(ctx context.Context, ids ecl.Set, profile string) (ecl.Set, error) {
	if err := f.hit("HistoricalAssociations"); err != nil {
		return nil, err
	}
	return f.base.HistoricalAssociations(ctx, ids, profile)
}

func (f *methodFailingProvider) ResolveIdentifier(ctx context.Context, scheme, code string) (ecl.Set, error) {
	if err := f.hit("ResolveIdentifier"); err != nil {
		return nil, err
	}
	return f.base.ResolveIdentifier(ctx, scheme, code)
}

func (f *methodFailingProvider) MatchDialect(ctx context.Context, concepts ecl.Set, opts ecl.DialectFilterOpts) (ecl.Set, error) {
	if err := f.hit("MatchDialect"); err != nil {
		return nil, err
	}
	return f.base.MatchDialect(ctx, concepts, opts)
}

func (f *methodFailingProvider) RefsetMembersFiltered(ctx context.Context, ids []string, opts ecl.MemberFilterOpts) (ecl.Set, error) {
	if err := f.hit("RefsetMembersFiltered"); err != nil {
		return nil, err
	}
	return f.base.RefsetMembersFiltered(ctx, ids, opts)
}
