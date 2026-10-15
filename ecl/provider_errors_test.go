package ecl_test

import (
	"context"
	"errors"
	"fmt"
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
// Nine of these surfaced unwrapped until every provider error went through
// providerError; they are ordinary regression cases now.
func TestEvaluate_EveryProviderFailureIsErrProvider(t *testing.T) {
	base := standardProvider(t) // read-only, so one copy serves every case
	for _, tc := range []struct {
		method, expr string
	}{
		{method: "Descendants", expr: `< 404684003`},
		{method: "Ancestors", expr: `> 22298006`},
		{method: "Children", expr: `<! 404684003`},
		{method: "Parents", expr: `>! 22298006`},
		{method: "ConceptExists", expr: `22298006`},
		{method: "AllConcepts", expr: `*`},
		{method: "RelationshipTargets", expr: `22298006 . 363698007`},
		{method: "ConcreteValues", expr: `<< 404684003 : 1142139005 = #5`},
		{method: "PropertiesByGroup", expr: `<< 404684003 : 363698007 = 74281007`},
		{method: "MatchDescription", expr: `<< 404684003 {{ D term = "heart" }}`},
		{method: "FilterConcepts", expr: `<< 404684003 {{ C active = true }}`},
		{method: "RefsetMembers", expr: `^ 900000000000497000`},
		{method: "RefsetsContainingMembers", expr: `^R 22298006`},
		{method: "HistoricalAssociations", expr: `22298006 {{ +HISTORY }}`},
		{method: "ResolveIdentifier", expr: `LOINC#1234-5`},
		{method: "MatchDialect", expr: `<< 404684003 {{ D dialectId = 900000000000508004 }}`},
		{method: "RefsetMembersFiltered", expr: `^ 447562003 {{ M mapGroup = "2" }}`},
	} {
		t.Run(tc.method, func(t *testing.T) {
			p := &methodFailingProvider{base: base, fail: tc.method, called: map[string]bool{}}
			_, err := evaluate(p, tc.expr)
			require.Truef(t, p.called[tc.method], "%q never called %s, so the case tests nothing; pick an expression that reaches it", tc.expr, tc.method)
			require.ErrorIsf(t, err, errBackendDown, "%q must keep the provider's own cause", tc.expr)

			require.Truef(t, errors.Is(err, ecl.ErrProvider), "a failing %s surfaces without ErrProvider: %v", tc.method, err)
		})
	}
}

// TestEvaluate_ProviderRefusalsAreNotBackendFailures covers the errors a
// provider may return that are not a failing backend. A capability refusing a
// form it cannot answer, and the caller's own cancellation or deadline, keep
// their identity and do NOT read as ErrProvider: one is a 501, the other is no
// answer owed at all. A backend's OWN timeout, while the caller's context is
// still live, is a slow backend, and does read as ErrProvider.
func TestEvaluate_ProviderRefusalsAreNotBackendFailures(t *testing.T) {
	base := standardProvider(t)
	run := func(t *testing.T, p *methodFailingProvider, ctx context.Context) error {
		t.Helper()
		_, err := ecl.Evaluate(ctx, mustParse(t, `22298006`), p)
		require.True(t, p.called["ConceptExists"], "the provider was never asked")
		return err
	}

	t.Run("unsupported", func(t *testing.T) {
		refusal := fmt.Errorf("%w: not here", ecl.ErrUnsupportedFeature)
		err := run(t, &methodFailingProvider{base: base, fail: "ConceptExists", called: map[string]bool{}, cause: refusal}, context.Background())
		require.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
		require.NotErrorIs(t, err, ecl.ErrProvider)
	})
	t.Run("caller canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := &methodFailingProvider{base: base, fail: "ConceptExists", called: map[string]bool{}, onFail: cancel, causeFromCtx: ctx}
		err := run(t, p, ctx)
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, ecl.ErrProvider)
	})
	t.Run("caller canceled, driver's own error", func(t *testing.T) {
		// A driver often answers a canceled request with an error of its own
		// that does not wrap context.Canceled.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := &methodFailingProvider{base: base, fail: "ConceptExists", called: map[string]bool{}, onFail: cancel, cause: errors.New("conn closed")}
		err := run(t, p, ctx)
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, ecl.ErrProvider)
	})
	t.Run("caller canceled, cause already ErrProvider", func(t *testing.T) {
		// A composed provider may hand back an error that already carries
		// ErrProvider; once the caller has given up it must not read as one.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := &methodFailingProvider{base: base, fail: "ConceptExists", called: map[string]bool{}, onFail: cancel, cause: fmt.Errorf("%w: inner", ecl.ErrProvider)}
		err := run(t, p, ctx)
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, ecl.ErrProvider)
	})
	t.Run("backend timeout, caller live", func(t *testing.T) {
		err := run(t, &methodFailingProvider{base: base, fail: "ConceptExists", called: map[string]bool{}, cause: context.DeadlineExceeded}, context.Background())
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, ecl.ErrProvider, "a backend's own timeout is a slow backend")
	})
}

// methodFailingProvider delegates to base and fails the one method named by
// fail, with errBackendDown from review_regressions_test.go.
type methodFailingProvider struct {
	base   ecl.DataProvider
	fail   string
	called map[string]bool
	cause  error // errBackendDown when nil

	// onFail runs before failing, and causeFromCtx, when set, makes the
	// failure that context's error: together they model a caller canceling
	// while the provider works.
	onFail       func()
	causeFromCtx context.Context
}

func (f *methodFailingProvider) hit(method string) error {
	f.called[method] = true
	if method != f.fail {
		return nil
	}
	if f.onFail != nil {
		f.onFail()
	}
	switch {
	case f.causeFromCtx != nil:
		return f.causeFromCtx.Err()
	case f.cause != nil:
		return f.cause
	}
	return errBackendDown
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
