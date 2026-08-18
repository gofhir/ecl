package ecl_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Expected IDs are computed against testdata/conformance/fixtures/standard.yaml,
// where 22298006 has a 363698007 relationship in group 1 AND in group 2, while
// 73211009 has one in group 1 only. That asymmetry is what makes these
// assertions discriminate: before the fix all four cardinalities returned the
// same set, because filterByAttributeGroup stopped at the first matching group
// and never read AttributeGroup.Cardinality at all.

func TestEvaluate_GroupCardinality(t *testing.T) {
	none := evalFixture(t, "* : { 363698007 = * }")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, none.Slice(),
		"a nil cardinality must behave as [1..*]")

	one := evalFixture(t, "* : [1..1] { 363698007 = * }")
	require.ElementsMatch(t, []string{"73211009"}, one.Slice(),
		"[1..1] must exclude the concept with two matching groups")

	two := evalFixture(t, "* : [2..*] { 363698007 = * }")
	require.ElementsMatch(t, []string{"22298006"}, two.Slice(),
		"[2..*] must select only the concept with two matching groups")

	require.NotEqual(t, one.Slice(), two.Slice(), "group cardinality is being ignored")
}

// TestEvaluate_GroupCardinalityZero covers [0..0], the form the ECL guide uses
// to express absence. It used to return exactly the inverse: the concepts that
// DID have the group.
func TestEvaluate_GroupCardinalityZero(t *testing.T) {
	with := evalFixture(t, "* : { 363698007 = 74281007 }")
	require.ElementsMatch(t, []string{"22298006"}, with.Slice())

	without := evalFixture(t, "* : [0..0] { 363698007 = 74281007 }")
	require.NotEmpty(t, without.Slice())
	require.NotContains(t, without.Slice(), "22298006",
		"[0..0] returned the concepts that DO have the group")
	// The two sets must partition the terminology, not coincide.
	require.Contains(t, without.Slice(), "73211009")
	for _, id := range with.Slice() {
		require.NotContains(t, without.Slice(), id)
	}
}

// TestEvaluate_ConcreteCardinality covers the concrete-value path, which decided
// with a bool and an early break.
func TestEvaluate_ConcreteCardinality(t *testing.T) {
	one := evalFixture(t, "* : [1..1] 1142139005 = #5")
	require.ElementsMatch(t, []string{"73211009"}, one.Slice())

	two := evalFixture(t, "* : [2..*] 1142139005 = #5")
	require.Empty(t, two.Slice(), "no concept has that value twice")

	zero := evalFixture(t, "* : [0..0] 1142139005 = #5")
	require.NotContains(t, zero.Slice(), "73211009")
	require.Contains(t, zero.Slice(), "22298006")
}

// TestEvaluate_ConcreteNotEqualsIsNotInverted is the anti-regression test for
// the concrete path: the comparison operator is applied inside
// compareFloat/compareString/compareBool, so counting must NOT subtract from the
// total the way the concept-valued path does. Doing so would invert results that
// are already correct.
func TestEvaluate_ConcreteNotEqualsIsNotInverted(t *testing.T) {
	// 73211009 is the only concept with 1142139005, and its value is 5.
	eq := evalFixture(t, "* : 1142139005 = #5")
	require.ElementsMatch(t, []string{"73211009"}, eq.Slice())

	neq5 := evalFixture(t, "* : 1142139005 != #5")
	require.Empty(t, neq5.Slice(), "no concept has a count that is not 5")

	neq2 := evalFixture(t, "* : 1142139005 != #2")
	require.ElementsMatch(t, []string{"73211009"}, neq2.Slice(), "5 != 2, so it qualifies")
}

// TestEvaluate_GroupZeroIsNotACountedGroup pins the decision that relationship
// group 0 ("ungrouped", per the PropertiesByGroup contract) is not counted as a
// matching group. Ungrouped attributes are matched by the ungrouped path.
func TestEvaluate_GroupZeroIsNotACountedGroup(t *testing.T) {
	// Every relationship in the fixture sits in group 1 or 2, so the ungrouped
	// and grouped forms agree here; the test documents the intent and will catch
	// a fixture change that introduces group 0 without revisiting the decision.
	ungrouped := evalFixture(t, "* : 363698007 = 74281007")
	grouped := evalFixture(t, "* : { 363698007 = 74281007 }")
	require.ElementsMatch(t, ungrouped.Slice(), grouped.Slice())
}
