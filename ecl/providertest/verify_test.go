package providertest_test

import (
	"context"
	"testing"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/providertest"
)

// TestVerifyFixture runs the bundled suite against the bundled fixture. This
// verifies the EVALUATOR: the cases assert concrete concept IDs that only exist
// in that fixture.
func TestVerifyFixture(t *testing.T) {
	providertest.VerifyFixture(t)
}

// TestVerifyContract exercises VerifyContract exactly as a third party would, and
// proves the reference fixture satisfies the contract.
//
// The two functions used to be one. Running the fixture-specific cases against an
// arbitrary provider failed 89 of 116 for a provider that was entirely correct and
// simply carried different data, and the 27 that passed were mostly the syntax
// error cases, which never reach the provider at all — so the tool the README
// offered to implementors reported failure 77% of the time and gave false
// confidence the rest.
func TestVerifyContract(t *testing.T) {
	providertest.VerifyContract(t, func() ecl.DataProvider {
		p, err := providertest.BundledFixture("standard.yaml")
		if err != nil {
			t.Fatalf("loading bundled fixture: %v", err)
		}
		return p
	})
}

// TestVerifyContract_SkipsRatherThanFailsOnThinData covers a provider that is
// correct but carries almost no data: it must be skipped, not failed, which is
// the whole difference from the fixture suite.
func TestVerifyContract_SkipsRatherThanFailsOnThinData(t *testing.T) {
	providertest.VerifyContract(t, func() ecl.DataProvider { return thinProvider{} })
}

// thinProvider obeys every contract rule and knows two concepts with no hierarchy,
// no refsets and no history — the shape a partial implementation has early on.
type thinProvider struct{ ecl.UnimplementedDataProvider }

func (thinProvider) AllConcepts(context.Context) (ecl.Set, error) {
	return ecl.NewSetFromSlice([]string{"111000", "222000"}), nil
}

func (thinProvider) ConceptExists(_ context.Context, ids []string) (ecl.Set, error) {
	return ecl.NewSetFromSlice(ids), nil
}

func (thinProvider) Descendants(_ context.Context, ids []string, includeSelf bool) (ecl.Set, error) {
	if includeSelf {
		return ecl.NewSetFromSlice(ids), nil
	}
	return ecl.NewSet(), nil
}

func (thinProvider) Ancestors(_ context.Context, ids []string, includeSelf bool) (ecl.Set, error) {
	if includeSelf {
		return ecl.NewSetFromSlice(ids), nil
	}
	return ecl.NewSet(), nil
}

func (thinProvider) Children(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (thinProvider) Parents(context.Context, []string, bool) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (thinProvider) RelationshipTargets(context.Context, ecl.Set, ecl.Set) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

// FilterConcepts honors the active axis. Returning the input unchanged would be a
// real contract violation, and VerifyContract reports it -- which is how this test
// found out the first version of thinProvider was wrong.
func (thinProvider) FilterConcepts(_ context.Context, concepts ecl.Set, opts ecl.ConceptFilterOpts) (ecl.Set, error) {
	if opts.Active != nil && !*opts.Active {
		return ecl.NewSet(), nil // every concept this provider knows is active
	}
	return concepts, nil
}

func (thinProvider) MatchDescription(context.Context, ecl.DescriptionFilterOpts) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (thinProvider) RefsetMembers(context.Context, []string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (thinProvider) RefsetsContainingMembers(context.Context, []string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (thinProvider) HistoricalAssociations(context.Context, ecl.Set, string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}

func (thinProvider) ResolveIdentifier(context.Context, string, string) (ecl.Set, error) {
	return ecl.NewSet(), nil
}
