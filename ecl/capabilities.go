package ecl

import "context"

// This file declares OPTIONAL capabilities a DataProvider may implement.
//
// The evaluator type-asserts for each one and falls back to the DataProvider
// method when it is absent, so implementing them is never required and never a
// breaking change — the same shape the standard library uses for io.ReaderFrom or
// http.Flusher.
//
// They exist because three things cannot be expressed through DataProvider as it
// stands, and widening that interface would break every implementation:
//
//   - PropertiesByGroup and ConcreteValues are per-concept BY SIGNATURE, so a
//     broad refinement issues one query per focus concept. Against SNOMED
//     International, `< 404684003 : 363698007 = *` is ~110,000 round trips.
//   - RelationshipTargets returns a Set, losing the inbound multiplicity that a
//     cardinality on a reverse attribute needs and the per-type total that "!="
//     needs. Those forms return ErrUnsupportedFeature without this.
//   - DescriptionFilterOpts has no way to say "match rows that do NOT satisfy
//     this", so a negated description filter returns ErrUnsupportedFeature.
//
// Implement the ones your storage can answer efficiently and leave the rest.
//
// The bundled providertest.VerifyContract has a check per capability: when it
// finds one, it asserts the capability AGREES with the required method it
// accelerates. That is the failure mode worth guarding — the evaluator prefers the
// capability, so a batch that disagrees with its per-concept equivalent silently
// decides every answer, and nothing else would notice.

// BatchPropertiesProvider answers PropertiesByGroup for many concepts at once.
//
// Implement it to collapse the evaluator's per-concept loop over refinements into
// one query. A refinement over N focus concepts costs one call instead of N.
//
// The result must contain an entry for every requested concept that has any
// relationship; a concept with none may be absent or map to an empty map. Group 0
// means "ungrouped", as in PropertiesByGroup.
type BatchPropertiesProvider interface {
	PropertiesByGroupBatch(ctx context.Context, conceptIDs []string) (map[string]map[int][]Relationship, error)
}

// BatchConcreteValuesProvider answers ConcreteValues for many (concept, type)
// pairs at once.
//
// Implement it to collapse the per-concept, per-type loop the concrete-value
// comparison would otherwise perform: without it, a wildcard attribute type over
// N concepts and T types costs N×T calls.
//
// The outer key is the concept ID and the inner key the attribute type ID. A pair
// with no values may be absent.
type BatchConcreteValuesProvider interface {
	ConcreteValuesBatch(ctx context.Context, conceptIDs, typeIDs []string) (map[string]map[string][]ConcreteValue, error)
}

// InboundRelationshipsProvider returns the relationships pointing AT each of the
// given concepts, preserving multiplicity.
//
// Implement it to enable the reverse-attribute forms the evaluator otherwise
// reports as unsupported:
//
//	[m..n] R attr = value      needs the inbound count
//	R attr != value            needs the per-type inbound total
//	[m..n] { R attr = value }  needs both
//
// The key is the TARGET concept — the one being pointed at — and the value every
// relationship whose target it is and whose type is in typeIDs. A concept with no
// inbound relationship of those types may be absent.
//
// This is the piece RelationshipTargets cannot supply: it returns a Set, so it
// answers "which concepts are pointed at" but not "how many times, and by what".
type InboundRelationshipsProvider interface {
	InboundRelationships(ctx context.Context, targetIDs Set, typeIDs Set) (map[string][]InboundRelationship, error)
}

// InboundRelationship is one relationship pointing at a concept.
type InboundRelationship struct {
	// SourceID is the concept the relationship comes from.
	SourceID string

	// TypeID is the attribute type.
	TypeID string

	// GroupNum is the relationship group on the SOURCE concept. 0 means
	// ungrouped.
	GroupNum int
}

// NegatingDescriptionProvider evaluates a description filter whose clauses may be
// negated at the ROW level.
//
// Implement it to enable `{{ D term != … }}`, `{{ D language != … }}` and
// `{{ D type != … }}`, which the evaluator otherwise reports as unsupported.
//
// Row-level is the whole point and cannot be emulated with set arithmetic: a
// negated description filter selects concepts that HAVE a description failing the
// comparison, so a concept with both an FSN and a Spanish synonym satisfies
// `language != es` through its FSN. Subtracting the concepts that have a Spanish
// description removes it, which is why the evaluator refuses to do that.
type NegatingDescriptionProvider interface {
	MatchDescriptionNegated(ctx context.Context, filter NegatedDescriptionFilterOpts) (Set, error)
}

// NegatedDescriptionFilterOpts carries a description filter in which each
// dimension states its own polarity.
//
// A concept matches when it has at least ONE description satisfying every
// dimension that is set, reading each through its Negate flag.
type NegatedDescriptionFilterOpts struct {
	// Opts holds the values to compare against, with the same any-of semantics
	// as DescriptionFilterOpts.
	Opts DescriptionFilterOpts

	// TermNegated inverts the Term comparison: match a description whose term
	// does NOT satisfy it.
	TermNegated bool

	// TypeIDsNegated inverts the TypeIDs comparison: match a description whose
	// type is NOT among them.
	TypeIDsNegated bool

	// LanguagesNegated inverts the Languages comparison.
	LanguagesNegated bool
}
