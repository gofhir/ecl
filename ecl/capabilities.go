package ecl

import "context"

// This file declares OPTIONAL capabilities a DataProvider may implement.
//
// The evaluator type-asserts for each one and falls back to the DataProvider
// method when it is absent, so implementing them is never required and never a
// breaking change — the same shape the standard library uses for io.ReaderFrom or
// http.Flusher.
//
// They exist because some things cannot be expressed through DataProvider as it
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
//   - A dialect ALIAS is not a concept reference, and resolving it to a language
//     reference set is terminology data no parser can compute.
//   - DescriptionFilterOpts has no field for a description's own SCTID, and
//     adding one would make the constraint vanish for every existing provider.
//
// Note what is NOT here. A filter value SET — `{{ D term = ("a" "b") }}`,
// `{{ C effectiveTime = ("20240131" "20230731") }}` — has any-of semantics, so it
// is the union of the single-value filters and the evaluator decomposes it into
// one call per value. That needs no capability and works with every provider. A
// Terms or EffectiveTimes field would have been worse than useless: every existing
// implementation would ignore it in silence, and the filter would quietly match on
// one value out of the set.
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
//
// It does not unlock a reverse attribute inside an attribute group: that form is
// rejected because grouping has no meaning there, not for want of data.
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

// DialectAliasResolver maps a dialect alias to the language reference sets it
// names.
//
// Implement it to enable the alias form of the dialect filter:
//
//	{{ D dialect = en-gb }}
//	{{ D dialect = (en-gb en-us) }}
//
// Without it that form returns ErrUnsupportedFeature, while the explicit
// `dialectId = 900000000000508004` form always works.
//
// The mapping is terminology data, not something a parser can compute. Only the
// international English aliases are universal — every national dialect uses a
// language reference set in its own namespace, and the same alias can mean
// different refsets in different editions — so a table built into this library
// would resolve some expressions and silently mis-resolve others.
//
// Aliases are passed as written, so an implementation must decide its own case
// handling; "en-GB" and "en-gb" name the same reference set in practice. The
// result maps each alias to the SCTIDs of its language reference sets; an alias
// may legitimately map to several. An alias the implementation does not know must
// be ABSENT from the map rather than mapped to an empty slice: absent means "I
// cannot resolve this", which the evaluator reports, whereas silently dropping it
// would widen the query to every dialect.
type DialectAliasResolver interface {
	ResolveDialectAliases(ctx context.Context, aliases []string) (map[string][]string, error)
}

// DescriptionIDProvider evaluates a description filter that constrains the
// description's own SCTID.
//
// Implement it to enable `{{ D id = 12345 }}` and `{{ D id != 12345 }}`, which
// the evaluator otherwise reports as unsupported.
//
// It is a capability rather than a field on DescriptionFilterOpts for the reason
// every other one here exists: a new field is silently ignored by every provider
// written against the current contract, so the id constraint would simply vanish
// and the filter would return every concept satisfying the OTHER clauses. That is
// not a smaller answer, it is a WIDER one — the failure mode this package refuses
// everywhere — and it is exactly what the description id filter used to do before
// it was rejected outright.
//
// The constraint is per description ROW, like negation: the same description must
// carry one of the listed ids AND satisfy Opts. A concept whose FSN has the id and
// whose synonym has the term does not match `{{ D id = X, term = "y" }}`.
type DescriptionIDProvider interface {
	MatchDescriptionByID(ctx context.Context, filter DescriptionIDFilterOpts) (Set, error)
}

// DescriptionIDFilterOpts carries a description id constraint alongside the rest
// of the description filter.
type DescriptionIDFilterOpts struct {
	// Opts holds the sibling clauses — term, type, language and the rest — which
	// the SAME description row must also satisfy.
	Opts DescriptionFilterOpts

	// DescriptionIDs are the description SCTIDs to match, with any-of semantics.
	// Never empty when the evaluator calls this.
	DescriptionIDs []string

	// Negate inverts the id comparison: match a description whose id is NOT among
	// DescriptionIDs. It does not invert Opts, whose clauses stay positive —
	// negating those needs NegatingDescriptionProvider, and the evaluator reports
	// the combination rather than guessing which of the two should win.
	Negate bool
}
