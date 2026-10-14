package providertest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/providertest"
)

// TestMatchDescription_EffectiveTime covers the description's own
// effectiveTime. DescriptionFilterOpts has carried the field all along, but the
// in-memory provider never read it, so a description effectiveTime filter
// matched every concept with a description — which would have made a correct
// evaluator look broken the day it started sending the field.
func TestMatchDescription_EffectiveTime(t *testing.T) {
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		// The concept's own time differs from both descriptions', so a provider
		// that read the concept instead would answer differently.
		Concepts: []providertest.FixtureConcept{{ID: "22298006", EffectiveTime: "20020131"}, {ID: "73211009"}},
		// 64572001's description declares no effectiveTime, so it must match
		// no comparison at all, "!=" included.
		Descriptions: []providertest.FixtureDescription{
			{ID: "200001011", Concept: "22298006", Language: "en", Value: "heart", EffectiveTime: "20240131"},
			{ID: "200004016", Concept: "73211009", Language: "en", Value: "heart", EffectiveTime: "20100131"},
			{Concept: "64572001", Language: "en", Value: "heart"},
		},
	})
	for _, tc := range []struct {
		time, op string
		want     []string
	}{
		{time: "20240131", op: "=", want: []string{"22298006"}},
		{time: "20020131", op: "=", want: nil},
		{time: "20240131", op: "!=", want: []string{"73211009"}},
		{time: "20150101", op: ">", want: []string{"22298006"}},
		{time: "20150101", op: "<=", want: []string{"73211009"}},
	} {
		t.Run(tc.op+tc.time, func(t *testing.T) {
			got, err := p.MatchDescription(context.Background(), ecl.DescriptionFilterOpts{EffectiveTime: tc.time, EffectiveTimeOp: tc.op})
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, got.Slice())
		})
	}

	// The negated path matches one description against every dimension through
	// its own function, so it needs its own check: here the term is negated and
	// the effectiveTime keeps the positive reading.
	negated, ok := p.(ecl.NegatingDescriptionProvider)
	require.True(t, ok, "the in-memory provider implements the negated matcher")
	got, err := negated.MatchDescriptionNegated(context.Background(), ecl.NegatedDescriptionFilterOpts{
		Opts:        ecl.DescriptionFilterOpts{Term: "nonexistent", EffectiveTime: "20240131", EffectiveTimeOp: "="},
		TermNegated: true,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"22298006"}, got.Slice())

	// So does the description-id path. Both ids are listed, so only the time
	// can tell them apart.
	byID, ok := p.(ecl.DescriptionIDProvider)
	require.True(t, ok, "the in-memory provider implements the description-id matcher")
	got, err = byID.MatchDescriptionByID(context.Background(), ecl.DescriptionIDFilterOpts{
		Opts:           ecl.DescriptionFilterOpts{EffectiveTime: "20240131", EffectiveTimeOp: "="},
		DescriptionIDs: []string{"200001011", "200004016"},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"22298006"}, got.Slice())
}
