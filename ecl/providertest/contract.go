package providertest

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/gofhir/ecl/ecl"
)

// VerifyContract checks a DataProvider against the rules the evaluator relies on,
// using the provider's OWN data.
//
// Every assertion here is data-independent: it probes the provider for concepts,
// refsets and relationships it actually has, then asserts an invariant that must
// hold whatever those are. So it works against a small in-memory map, a
// PostgreSQL closure table, or a full SNOMED International release.
//
//	func TestMyProvider(t *testing.T) {
//	    providertest.VerifyContract(t, func() ecl.DataProvider {
//	        return newMyProvider(t)
//	    })
//	}
//
// A check whose invariant the provider's data cannot exercise — no inactive
// concepts to test the active axis with, no refsets to test the refset inverse —
// is skipped with a reason rather than failed. A skip means "not proven here", so
// read the output: a provider that skips most of the suite has not been verified.
//
// This is deliberately NOT the bundled conformance suite. Those cases assert
// concrete concept IDs from the bundled fixture, so a correct provider carrying
// different data fails almost all of them: measured, 89 of 116. That measurement
// is why this function exists. Use VerifyFixture for the suite.
//
// A fresh provider is requested per check so state cannot leak between them.
func VerifyContract(t *testing.T, newProvider func() ecl.DataProvider) {
	t.Helper()

	checks := []struct {
		name string
		run  func(context.Context, *testing.T, ecl.DataProvider)
	}{
		{"NeverReturnsNilSet", checkNeverReturnsNilSet},
		{"EmptyInputYieldsEmptyOutput", checkEmptyInputYieldsEmptyOutput},
		{"NilSourceIDsIsWildcard", checkNilSourceIDsIsWildcard},
		{"HierarchyIsTransitive", checkHierarchyIsTransitive},
		{"IncludeSelfAddsTheInput", checkIncludeSelfAddsTheInput},
		{"DescendantsAndAncestorsAgree", checkDescendantsAndAncestorsAgree},
		{"ChildrenAreDescendants", checkChildrenAreDescendants},
		{"ActiveAxisBelongsToFilterConcepts", checkActiveAxisBelongsToFilterConcepts},
		{"RefsetMembershipIsInvertible", checkRefsetMembershipIsInvertible},
		{"HistoryProfilesAreNested", checkHistoryProfilesAreNested},
		{"ResultsAreDeterministic", checkResultsAreDeterministic},

		// Optional capabilities. Each check skips when the provider does not
		// implement the interface; when it does, the check asserts that the
		// capability AGREES with the required method it accelerates. A batch that
		// disagrees with its per-concept equivalent is the worst possible bug
		// here: the evaluator prefers the batch, so the disagreement decides the
		// answer and nothing else would notice.
		{"BatchPropertiesAgreesWithPerConcept", checkBatchPropertiesAgrees},
		{"BatchConcreteValuesAgreesWithPerPair", checkBatchConcreteValuesAgrees},
		{"InboundRelationshipsAgreesWithTargets", checkInboundAgreesWithTargets},
		{"NegatedDescriptionIsNotSetSubtraction", checkNegatedDescriptionIsRowLevel},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			c.run(context.Background(), t, newProvider())
		})
	}
}

// checkNeverReturnsNilSet covers the contract's first rule. Returning (nil, nil)
// for "nothing found" is idiomatic Go and used to panic inside the evaluator.
func checkNeverReturnsNilSet(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	all := mustAllConcepts(ctx, t, p)
	ids := sample(all, 1)

	assertNonNil := func(name string, set ecl.Set, err error) {
		if err != nil {
			return // an error is a legitimate answer; a nil Set with nil error is not
		}
		if set == nil {
			t.Errorf("%s returned a nil Set; the contract requires ecl.NewSet() for the empty result", name)
		}
	}

	assertNonNil("AllConcepts", all, nil)

	set, err := p.Descendants(ctx, ids, false)
	assertNonNil("Descendants", set, err)
	set, err = p.Ancestors(ctx, ids, false)
	assertNonNil("Ancestors", set, err)
	set, err = p.Children(ctx, ids, false)
	assertNonNil("Children", set, err)
	set, err = p.Parents(ctx, ids, false)
	assertNonNil("Parents", set, err)
	set, err = p.ConceptExists(ctx, ids)
	assertNonNil("ConceptExists", set, err)
	set, err = p.RefsetMembers(ctx, ids)
	assertNonNil("RefsetMembers", set, err)
	set, err = p.RefsetsContainingMembers(ctx, ids)
	assertNonNil("RefsetsContainingMembers", set, err)
	set, err = p.FilterConcepts(ctx, ecl.NewSetFromSlice(ids), ecl.ConceptFilterOpts{})
	assertNonNil("FilterConcepts", set, err)
	set, err = p.MatchDescription(ctx, ecl.DescriptionFilterOpts{Term: "zzz-no-such-term"})
	assertNonNil("MatchDescription", set, err)
	set, err = p.HistoricalAssociations(ctx, ecl.NewSetFromSlice(ids), "HISTORY-MAX")
	assertNonNil("HistoricalAssociations", set, err)
	set, err = p.ResolveIdentifier(ctx, "no-such-scheme", "no-such-code")
	assertNonNil("ResolveIdentifier", set, err)
}

// checkEmptyInputYieldsEmptyOutput covers the rule that an empty input must not be
// read as "no filter" and answered with the whole terminology.
func checkEmptyInputYieldsEmptyOutput(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	assertEmpty := func(name string, set ecl.Set, err error) {
		if err != nil {
			return
		}
		if set != nil && set.Len() > 0 {
			t.Errorf("%s answered %d concepts for an empty input; an empty input must yield the empty Set", name, set.Len())
		}
	}

	set, err := p.Descendants(ctx, nil, false)
	assertEmpty("Descendants(nil)", set, err)
	set, err = p.Descendants(ctx, []string{}, true)
	assertEmpty("Descendants(empty, includeSelf)", set, err)
	set, err = p.Ancestors(ctx, nil, false)
	assertEmpty("Ancestors(nil)", set, err)
	set, err = p.Children(ctx, nil, false)
	assertEmpty("Children(nil)", set, err)
	set, err = p.Parents(ctx, nil, false)
	assertEmpty("Parents(nil)", set, err)
	set, err = p.ConceptExists(ctx, nil)
	assertEmpty("ConceptExists(nil)", set, err)
	set, err = p.RefsetMembers(ctx, nil)
	assertEmpty("RefsetMembers(nil)", set, err)
	set, err = p.FilterConcepts(ctx, ecl.NewSet(), ecl.ConceptFilterOpts{})
	assertEmpty("FilterConcepts(empty)", set, err)
	set, err = p.RelationshipTargets(ctx, ecl.NewSet(), ecl.NewSet())
	assertEmpty("RelationshipTargets(empty, empty)", set, err)
}

// checkNilSourceIDsIsWildcard covers the convention the evaluator relies on for
// the reverse-wildcard form `R attr = *`. A provider that dereferences sourceIDs
// without checking panics on that expression.
func checkNilSourceIDsIsWildcard(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RelationshipTargets panicked on nil sourceIDs: %v\n"+
				"nil means wildcard here; the evaluator passes it for `R attr = *`", r)
		}
	}()

	all := mustAllConcepts(ctx, t, p)
	types := ecl.NewSetFromSlice(sample(all, 5))

	wildcard, err := p.RelationshipTargets(ctx, nil, types)
	if err != nil {
		t.Skipf("RelationshipTargets reported: %v", err)
	}
	if wildcard == nil {
		t.Error("RelationshipTargets returned a nil Set for the wildcard form")
	}

	// An empty non-nil Set is NOT the wildcard: it means "no sources".
	empty, err := p.RelationshipTargets(ctx, ecl.NewSet(), types)
	if err == nil && empty != nil && empty.Len() > 0 {
		t.Errorf("RelationshipTargets answered %d targets for an empty (non-nil) source set; "+
			"only nil means wildcard", empty.Len())
	}
}

// checkHierarchyIsTransitive covers Descendants and Ancestors being transitive:
// a descendant's descendants are descendants too.
func checkHierarchyIsTransitive(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	root, children := findConceptWithChildren(ctx, t, p)

	descendants, err := p.Descendants(ctx, []string{root}, false)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	descendants = nonNilSet(descendants)

	for _, child := range sample(ecl.NewSetFromSlice(children), 3) {
		if !descendants.Contains(child) {
			t.Errorf("Children(%s) includes %s but Descendants(%s) does not", root, child, root)
		}
		grandchildren, err := p.Descendants(ctx, []string{child}, false)
		if err != nil {
			continue
		}
		for _, g := range sample(nonNilSet(grandchildren), 3) {
			if !descendants.Contains(g) {
				t.Errorf("%s is a descendant of %s, which is a descendant of %s, "+
					"but Descendants(%s) omits it: the relation is not transitive",
					g, child, root, root)
			}
		}
	}
}

// checkIncludeSelfAddsTheInput covers the includeSelf flag: it adds the input
// concepts and changes nothing else.
func checkIncludeSelfAddsTheInput(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	root, _ := findConceptWithChildren(ctx, t, p)

	without, err := p.Descendants(ctx, []string{root}, false)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	with, err := p.Descendants(ctx, []string{root}, true)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	without, with = nonNilSet(without), nonNilSet(with)

	if !with.Contains(root) {
		t.Errorf("Descendants(%s, includeSelf=true) omits %s", root, root)
	}
	if without.Contains(root) {
		t.Errorf("Descendants(%s, includeSelf=false) includes %s", root, root)
	}
	if got, want := with.Len(), without.Len()+1; got != want {
		t.Errorf("Descendants(%s, includeSelf=true) has %d concepts, want %d: "+
			"includeSelf must add the input and nothing else", root, got, want)
	}
}

// checkDescendantsAndAncestorsAgree covers the two directions describing the same
// relation: if y is a descendant of x then x is an ancestor of y.
func checkDescendantsAndAncestorsAgree(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	root, _ := findConceptWithChildren(ctx, t, p)

	descendants, err := p.Descendants(ctx, []string{root}, false)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	for _, d := range sample(nonNilSet(descendants), 5) {
		ancestors, err := p.Ancestors(ctx, []string{d}, false)
		if err != nil {
			continue
		}
		if !nonNilSet(ancestors).Contains(root) {
			t.Errorf("%s is in Descendants(%s) but %s is not in Ancestors(%s): "+
				"the two directions disagree", d, root, root, d)
		}
	}
}

// checkChildrenAreDescendants covers Children being the depth-1 slice of
// Descendants, and Parents likewise for Ancestors.
func checkChildrenAreDescendants(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	root, children := findConceptWithChildren(ctx, t, p)

	descendants, err := p.Descendants(ctx, []string{root}, false)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	for _, c := range children {
		if !nonNilSet(descendants).Contains(c) {
			t.Errorf("Children(%s) includes %s but Descendants(%s) does not", root, c, root)
		}
	}

	child := children[0]
	parents, err := p.Parents(ctx, []string{child}, false)
	if err != nil {
		t.Skipf("Parents reported: %v", err)
	}
	ancestors, err := p.Ancestors(ctx, []string{child}, false)
	if err != nil {
		t.Skipf("Ancestors reported: %v", err)
	}
	for _, parent := range sample(nonNilSet(parents), 5) {
		if !nonNilSet(ancestors).Contains(parent) {
			t.Errorf("Parents(%s) includes %s but Ancestors(%s) does not", child, parent, child)
		}
	}
}

// checkActiveAxisBelongsToFilterConcepts covers the rule that only FilterConcepts
// may filter by the active flag.
//
// The wildcard is resolved BEFORE filters are applied, so an AllConcepts that
// returns only active concepts makes `* {{ C active = false }}` unable to return
// anything, whatever FilterConcepts does.
func checkActiveAxisBelongsToFilterConcepts(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	all := mustAllConcepts(ctx, t, p)

	inactiveSet, err := p.FilterConcepts(ctx, all, ecl.ConceptFilterOpts{Active: boolPtr(false)})
	if err != nil {
		t.Skipf("FilterConcepts reported: %v", err)
	}
	inactiveSet = nonNilSet(inactiveSet)

	if inactiveSet.Len() == 0 {
		t.Skip("the provider reports no inactive concepts among AllConcepts, so the active axis cannot be exercised; " +
			"if your terminology DOES have inactive concepts, AllConcepts is filtering them out, which breaks `* {{ C active = false }}`")
	}

	// Every inactive concept must also be visible through ConceptExists, which
	// must not filter by the flag either.
	ids := sample(inactiveSet, 5)
	exists, err := p.ConceptExists(ctx, ids)
	if err != nil {
		t.Skipf("ConceptExists reported: %v", err)
	}
	for _, id := range ids {
		if !nonNilSet(exists).Contains(id) {
			t.Errorf("ConceptExists omits the inactive concept %s; only FilterConcepts may filter by the active flag", id)
		}
	}

	// The two halves must partition the input.
	activeSet, err := p.FilterConcepts(ctx, all, ecl.ConceptFilterOpts{Active: boolPtr(true)})
	if err != nil {
		t.Skipf("FilterConcepts reported: %v", err)
	}
	if got, want := nonNilSet(activeSet).Len()+inactiveSet.Len(), all.Len(); got != want {
		t.Errorf("FilterConcepts(active=true) and (active=false) cover %d concepts, but AllConcepts has %d: "+
			"the two must partition the input", got, want)
	}
}

// checkRefsetMembershipIsInvertible covers RefsetMembers and
// RefsetsContainingMembers describing the same relation from the two ends, which
// is what the ^R operator relies on.
func checkRefsetMembershipIsInvertible(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	all := mustAllConcepts(ctx, t, p)

	// Any concept may be a refset; probe for one that has members.
	for _, candidate := range sample(all, 40) {
		members, err := p.RefsetMembers(ctx, []string{candidate})
		if err != nil || nonNilSet(members).Len() == 0 {
			continue
		}
		for _, m := range sample(nonNilSet(members), 3) {
			refsets, err := p.RefsetsContainingMembers(ctx, []string{m})
			if err != nil {
				t.Skipf("RefsetsContainingMembers reported: %v", err)
			}
			if !nonNilSet(refsets).Contains(candidate) {
				t.Errorf("%s is in RefsetMembers(%s) but %s is not in RefsetsContainingMembers(%s): "+
					"the two directions disagree, so ^R cannot work", m, candidate, candidate, m)
			}
		}
		return
	}
	t.Skip("no refset with members found among the sampled concepts, so the refset inverse cannot be exercised")
}

// checkHistoryProfilesAreNested covers the MIN / MOD / MAX profiles.
//
// MIN follows SAME_AS only, MOD adds a few more association types and MAX follows
// every one, so the results must nest. A provider that ignores the profile returns
// the same set for all three, which this catches whenever any association exists.
func checkHistoryProfilesAreNested(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	all := mustAllConcepts(ctx, t, p)
	probe := ecl.NewSetFromSlice(sample(all, 40))

	minSet, err := p.HistoricalAssociations(ctx, probe, "HISTORY-MIN")
	if err != nil {
		t.Skipf("HistoricalAssociations reported: %v", err)
	}
	modSet, err := p.HistoricalAssociations(ctx, probe, "HISTORY-MOD")
	if err != nil {
		t.Skipf("HistoricalAssociations reported: %v", err)
	}
	maxSet, err := p.HistoricalAssociations(ctx, probe, "HISTORY-MAX")
	if err != nil {
		t.Skipf("HistoricalAssociations reported: %v", err)
	}
	minSet, modSet, maxSet = nonNilSet(minSet), nonNilSet(modSet), nonNilSet(maxSet)

	if maxSet.Len() == 0 {
		t.Skip("no historical association found for the sampled concepts, so the profiles cannot be exercised; " +
			"note the input is the set of REPLACEMENT concepts, not the historical ones")
	}
	if minSet.Minus(modSet).Len() > 0 {
		t.Error("HISTORY-MIN returned concepts HISTORY-MOD did not: MIN must be a subset of MOD")
	}
	if modSet.Minus(maxSet).Len() > 0 {
		t.Error("HISTORY-MOD returned concepts HISTORY-MAX did not: MOD must be a subset of MAX")
	}
}

// checkResultsAreDeterministic covers a provider answering the same question the
// same way twice, which every set operation in the evaluator assumes.
func checkResultsAreDeterministic(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	all := mustAllConcepts(ctx, t, p)
	ids := sample(all, 5)

	first, err := p.Descendants(ctx, ids, true)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	second, err := p.Descendants(ctx, ids, true)
	if err != nil {
		t.Skipf("Descendants reported: %v", err)
	}
	a, b := nonNilSet(first).Slice(), nonNilSet(second).Slice()
	if len(a) != len(b) {
		t.Fatalf("Descendants returned %d concepts, then %d for the same input", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Descendants returned different results for the same input: %v vs %v", a, b)
		}
	}
}

// checkBatchPropertiesAgrees covers BatchPropertiesProvider returning the same
// relationships as PropertiesByGroup.
//
// The evaluator prefers the batch when it is present, so a disagreement decides
// every refinement's answer and nothing else in the suite would catch it.
func checkBatchPropertiesAgrees(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	batch, ok := p.(ecl.BatchPropertiesProvider)
	if !ok {
		t.Skip("the provider does not implement ecl.BatchPropertiesProvider; the evaluator will call PropertiesByGroup once per focus concept")
	}

	all := mustAllConcepts(ctx, t, p)
	ids := sample(all, 20)

	batched, err := batch.PropertiesByGroupBatch(ctx, ids)
	if err != nil {
		t.Fatalf("PropertiesByGroupBatch: %v", err)
	}

	for _, id := range ids {
		single, err := p.PropertiesByGroup(ctx, id)
		if err != nil {
			continue
		}
		for group, rels := range single {
			if len(rels) != len(batched[id][group]) {
				t.Errorf("concept %s group %d: PropertiesByGroup returned %d relationships, PropertiesByGroupBatch %d",
					id, group, len(rels), len(batched[id][group]))
			}
		}
		// The batch must not invent groups either.
		for group := range batched[id] {
			if _, ok := single[group]; !ok {
				t.Errorf("concept %s: PropertiesByGroupBatch reported group %d, which PropertiesByGroup does not", id, group)
			}
		}
	}
}

// checkBatchConcreteValuesAgrees covers BatchConcreteValuesProvider returning the
// same values as ConcreteValues, for the same reason.
func checkBatchConcreteValuesAgrees(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	batch, ok := p.(ecl.BatchConcreteValuesProvider)
	if !ok {
		t.Skip("the provider does not implement ecl.BatchConcreteValuesProvider; the evaluator will call ConcreteValues once per concept and type")
	}

	all := mustAllConcepts(ctx, t, p)
	ids := sample(all, 20)
	types := sample(all, 10) // any concept may be an attribute type

	batched, err := batch.ConcreteValuesBatch(ctx, ids, types)
	if err != nil {
		t.Fatalf("ConcreteValuesBatch: %v", err)
	}

	found := false
	for _, id := range ids {
		for _, typeID := range types {
			single, err := p.ConcreteValues(ctx, id, typeID)
			if err != nil {
				continue
			}
			if len(single) > 0 {
				found = true
			}
			if len(single) != len(batched[id][typeID]) {
				t.Errorf("concept %s type %s: ConcreteValues returned %d values, ConcreteValuesBatch %d",
					id, typeID, len(single), len(batched[id][typeID]))
			}
		}
	}
	if !found {
		t.Skip("no concrete value found among the sampled pairs, so agreement could only be checked on empty results")
	}
}

// checkInboundAgreesWithTargets covers InboundRelationshipsProvider agreeing with
// RelationshipTargets about WHICH concepts are pointed at.
//
// The capability adds the multiplicity a Set cannot carry, but the membership must
// still match: the evaluator uses RelationshipTargets for `R attr = value` and the
// capability for the counted forms, so a disagreement makes two spellings of one
// question give different answers.
func checkInboundAgreesWithTargets(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	inbound, ok := p.(ecl.InboundRelationshipsProvider)
	if !ok {
		t.Skip("the provider does not implement ecl.InboundRelationshipsProvider; a cardinality or \"!=\" on a reverse attribute will report ErrUnsupportedFeature")
	}

	all := mustAllConcepts(ctx, t, p)
	types := ecl.NewSetFromSlice(sample(all, 10))

	byTarget, err := inbound.InboundRelationships(ctx, all, types)
	if err != nil {
		t.Fatalf("InboundRelationships: %v", err)
	}
	if len(byTarget) == 0 {
		t.Skip("no inbound relationship found for the sampled types, so agreement cannot be checked")
	}

	// Every target the capability reports must also be a target according to
	// RelationshipTargets, asked with the sources the capability named.
	for target, rels := range byTarget {
		sources := make([]string, 0, len(rels))
		for _, r := range rels {
			sources = append(sources, r.SourceID)
		}
		targets, err := p.RelationshipTargets(ctx, ecl.NewSetFromSlice(sources), types)
		if err != nil {
			t.Skipf("RelationshipTargets reported: %v", err)
		}
		if !nonNilSet(targets).Contains(target) {
			t.Errorf("InboundRelationships says %v point at %s, but RelationshipTargets does not report %s as a target",
				sources, target, target)
		}
	}
}

// checkNegatedDescriptionIsRowLevel covers NegatingDescriptionProvider negating
// per description ROW rather than by subtracting sets.
//
// The distinction is the entire reason the capability exists: a concept holding
// both an FSN and a Spanish synonym satisfies `language != es` through the FSN, so
// the negated result is NOT the complement of the positive one. A provider that
// implements the interface by subtracting has changed nothing.
func checkNegatedDescriptionIsRowLevel(ctx context.Context, t *testing.T, p ecl.DataProvider) {
	negating, ok := p.(ecl.NegatingDescriptionProvider)
	if !ok {
		t.Skip("the provider does not implement ecl.NegatingDescriptionProvider; negated description filters will report ErrUnsupportedFeature")
	}

	// Find a concept with descriptions in TWO languages: it is the only shape that
	// tells row-level negation apart from set subtraction, because it must appear
	// in both the positive and the negated result.
	byLanguage := map[string]ecl.Set{}
	for _, code := range []string{"en", "es", "fr", "de", "nl", "sv", "da", "no", "fi", "et"} {
		set, err := p.MatchDescription(ctx, ecl.DescriptionFilterOpts{Languages: []string{code}})
		if err != nil {
			t.Skipf("MatchDescription reported: %v", err)
		}
		if s := nonNilSet(set); s.Len() > 0 {
			byLanguage[code] = s
		}
	}
	if len(byLanguage) < 2 {
		t.Skip("fewer than two description languages found among the common codes, so row-level negation cannot be told apart from set subtraction here")
	}

	// Sorted, so the check exercises the same language every run: iterating the
	// map picked one at random and returned after the first usable candidate, which
	// made coverage vary between runs.
	languages := make([]string, 0, len(byLanguage))
	for code := range byLanguage {
		languages = append(languages, code)
	}
	sort.Strings(languages)

	for _, language := range languages {
		positive := byLanguage[language]
		// Concepts in this language that are also in some other language.
		multilingual := ecl.NewSet()
		for other, set := range byLanguage {
			if other != language {
				multilingual = multilingual.Union(positive.Intersect(set))
			}
		}
		if multilingual.Len() == 0 {
			continue
		}

		negated, err := negating.MatchDescriptionNegated(ctx, ecl.NegatedDescriptionFilterOpts{
			Opts:             ecl.DescriptionFilterOpts{Languages: []string{language}},
			LanguagesNegated: true,
		})
		if err != nil {
			t.Fatalf("MatchDescriptionNegated: %v", err)
		}

		missing := multilingual.Minus(nonNilSet(negated))
		if missing.Len() > 0 {
			t.Errorf("%v have a description in %q AND in another language, so they satisfy `language != %s` "+
				"through that other description, but MatchDescriptionNegated omits them. "+
				"The negation has to be applied per description ROW; subtracting the concepts that have a %q "+
				"description is what this capability exists to replace.",
				missing.Slice(), language, language, language)
		}
		return
	}
	t.Skip("no concept found with descriptions in two languages, so row-level negation cannot be told apart from set subtraction here")
}

// ---------------------------------------------------------------------------
// Probing helpers
// ---------------------------------------------------------------------------.

// mustAllConcepts fetches AllConcepts, skipping the check when the provider
// cannot answer it: everything here needs somewhere to start.
func mustAllConcepts(ctx context.Context, t *testing.T, p ecl.DataProvider) ecl.Set {
	t.Helper()
	all, err := p.AllConcepts(ctx)
	if err != nil {
		if errors.Is(err, ecl.ErrUnsupportedFeature) {
			t.Skipf("AllConcepts is not implemented, so the contract cannot be probed: %v", err)
		}
		t.Skipf("AllConcepts reported: %v", err)
	}
	all = nonNilSet(all)
	if all.Len() == 0 {
		t.Skip("AllConcepts is empty, so there is no data to probe")
	}
	return all
}

// findConceptWithChildren returns a concept that has children, skipping the check
// when the provider exposes no hierarchy.
func findConceptWithChildren(ctx context.Context, t *testing.T, p ecl.DataProvider) (parent string, children []string) {
	t.Helper()
	all := mustAllConcepts(ctx, t, p)

	for _, id := range sample(all, 40) {
		children, err := p.Children(ctx, []string{id}, false)
		if err != nil {
			continue
		}
		if kids := nonNilSet(children).Slice(); len(kids) > 0 {
			return id, kids
		}
	}
	t.Skip("no concept with children found among the sampled concepts, so the hierarchy cannot be probed")
	return "", nil
}

// sample returns up to n elements of a set, in the set's sorted order so the
// choice is deterministic across runs.
func sample(set ecl.Set, n int) []string {
	if set == nil {
		return nil
	}
	ids := set.Slice()
	if len(ids) > n {
		ids = ids[:n]
	}
	return ids
}

// nonNilSet normalizes a Set so a provider that breaks the non-nil rule produces
// a reported failure rather than a panic inside this suite.
func nonNilSet(s ecl.Set) ecl.Set {
	if s == nil {
		return ecl.NewSet()
	}
	return s
}

func boolPtr(b bool) *bool { return &b }
