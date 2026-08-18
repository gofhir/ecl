package ecl

import "context"

// DataProvider abstracts the SNOMED CT data source for the ECL evaluator.
// Consumers implement this against their storage (PostgreSQL closure tables,
// in-memory maps, Elasticsearch, etc.).
//
// Most methods take slices/sets so they can be answered with a batch query.
// Two are per-concept by signature (ConcreteValues and PropertiesByGroup) and
// are called once per focus concept; batching them is a planned breaking change.
//
// # Contract
//
// These rules apply to every method and are what the evaluator relies on. They
// are not optional: an implementation that breaks them produces wrong results
// rather than errors.
//
//   - Never return a nil Set. Use NewSet() for the empty result. The evaluator
//     normalizes nil defensively, but do not rely on that.
//   - An empty (non-nil) input Set or slice yields the empty Set — never the
//     whole terminology.
//   - Results are unordered. The evaluator sorts when it needs determinism.
//   - Hierarchy methods are transitive, except Children and Parents, which are
//     depth 1 by definition.
//   - The active axis belongs to FilterConcepts alone. No other method may
//     filter by the active flag: doing so makes expressions such as
//     `* {{ C active = false }}` unable to return anything, because the
//     wildcard is resolved before the filter is applied.
//
// The bundled conformance suite exercises these rules; run it against your
// implementation to check them.
type DataProvider interface {
	// ── Hierarchy ──────────────────────────────────────────────────────────
	// Descendants returns all transitive descendants of the given concepts.
	// When includeSelf is true, the input concepts are included in the result.
	Descendants(ctx context.Context, conceptIDs []string, includeSelf bool) (Set, error)

	// Ancestors returns all transitive ancestors of the given concepts.
	Ancestors(ctx context.Context, conceptIDs []string, includeSelf bool) (Set, error)

	// Children returns the direct children (depth 1) of the given concepts.
	Children(ctx context.Context, conceptIDs []string, includeSelf bool) (Set, error)

	// Parents returns the direct parents (depth 1) of the given concepts.
	Parents(ctx context.Context, conceptIDs []string, includeSelf bool) (Set, error)

	// ── Concepts ───────────────────────────────────────────────────────────
	// ConceptExists returns the subset of the input that exists in the
	// terminology, whether active or not. Like AllConcepts, it must not filter
	// by the active flag.
	ConceptExists(ctx context.Context, conceptIDs []string) (Set, error)

	// AllConcepts returns every concept that exists in the terminology, active
	// or not (used for the wildcard *).
	//
	// It must NOT filter by the active flag. The wildcard is resolved before
	// filters are applied, so an implementation that returns only active
	// concepts makes `* {{ C active = false }}` return the empty set no matter
	// what FilterConcepts does. Restricting the active axis is FilterConcepts'
	// job.
	AllConcepts(ctx context.Context) (Set, error)

	// ── Attributes (for refinements) ───────────────────────────────────────
	// RelationshipTargets returns the union of target concept IDs of relationships
	// whose source is in sourceIDs and type is in typeIDs.
	//
	// When sourceIDs is nil, every source is considered (wildcard). The
	// evaluator passes nil for the reverse-wildcard form `R attr = *`, so an
	// implementation that dereferences sourceIDs without checking will panic on
	// that expression. An empty non-nil Set still means "no sources", i.e. the
	// empty result.
	RelationshipTargets(ctx context.Context, sourceIDs Set, typeIDs Set) (Set, error)

	// RelationshipSources returns the union of source concept IDs of relationships
	// whose target is in targetIDs and type is in typeIDs (for reverse flag "R").
	// When targetIDs is nil, all targets are considered (wildcard) — though the
	// evaluator currently always passes a concrete set here.
	RelationshipSources(ctx context.Context, targetIDs Set, typeIDs Set) (Set, error)

	// ConcreteValues returns concrete values for the given source concept and attribute type.
	ConcreteValues(ctx context.Context, sourceID string, typeID string) ([]ConcreteValue, error)

	// ── Grouped attributes ─────────────────────────────────────────────────
	// PropertiesByGroup returns all attribute relationships of a concept, grouped by
	// relationship group number. Group 0 means "ungrouped". Used for two-phase
	// grouped refinement evaluation.
	PropertiesByGroup(ctx context.Context, conceptID string) (map[int][]Relationship, error)

	// ── Description filters ────────────────────────────────────────────────
	// MatchDescription returns concept IDs whose descriptions match the filter.
	MatchDescription(ctx context.Context, filter DescriptionFilterOpts) (Set, error)

	// ── Concept filters ────────────────────────────────────────────────────
	// FilterConcepts restricts a concept set using concept metadata filters
	// (active, definitionStatus, module, effectiveTime).
	FilterConcepts(ctx context.Context, concepts Set, filter ConceptFilterOpts) (Set, error)

	// ── Refsets ────────────────────────────────────────────────────────────
	// RefsetMembers returns the concept IDs that are members of any of the given refsets.
	RefsetMembers(ctx context.Context, refsetIDs []string) (Set, error)

	// RefsetsContainingMembers returns the set of refset IDs that contain
	// any of the given concept IDs as a member. Used to evaluate the ^R
	// (refset containing any) operator. This is the inverse direction of
	// RefsetMembers: given concepts, find the refsets they belong to.
	RefsetsContainingMembers(ctx context.Context, conceptIDs []string) (Set, error)

	// ── History supplements (v2.0) ─────────────────────────────────────────
	// HistoricalAssociations returns the INACTIVE concepts that were replaced by
	// any of the given concepts, according to the profile (MIN, MOD, MAX, ALL).
	//
	// Direction matters and is the opposite of what one might assume. The spec
	// defines the supplement as
	//
	//	(X) OR (^ 900000000000527005 {{ M targetComponentId = (X) }})
	//
	// so the input is the set of (typically active) concepts, and the result is
	// the historical concepts whose targetComponentId points AT them. An
	// implementation that expands active concepts to their replacements instead
	// makes `{{ +HISTORY }}` a silent no-op for every realistic input.
	HistoricalAssociations(ctx context.Context, conceptIDs Set, profile string) (Set, error)

	// ── Alternate identifiers (v2.2) ──────────────────────────────────────
	// ResolveIdentifier resolves an alternate identifier (scheme#code) to
	// SNOMED CT concept IDs.
	ResolveIdentifier(ctx context.Context, scheme string, code string) (Set, error)

	// ── Dialect filter ────────────────────────────────────────────────────
	// MatchDialect returns concept IDs whose descriptions match the dialect
	// filter constraints.
	MatchDialect(ctx context.Context, concepts Set, filter DialectFilterOpts) (Set, error)

	// ── Member filter ─────────────────────────────────────────────────────
	// RefsetMembersFiltered returns concept IDs from refset members that match
	// the member field filter.
	RefsetMembersFiltered(ctx context.Context, refsetIDs []string, filter MemberFilterOpts) (Set, error)
}

// Relationship is a single attribute relationship of a concept.
type Relationship struct {
	TypeID        string
	TargetID      string // "" when ConcreteValue is set
	GroupNum      int
	ConcreteValue *ConcreteValue // nil for concept-valued relationships
}

// ConcreteValue is a concrete (non-concept) attribute value.
type ConcreteValue struct {
	// Kind is "integer", "decimal", "string", or "boolean".
	Kind string
	// Value holds the raw value as a string for the caller to parse.
	Value string
}

// DescriptionFilterOpts describes which descriptions to match.
// Empty / nil slice fields are ignored (no filter on that dimension).
// Slice fields use any-of semantics: a description matches if it has
// any one of the listed values.
type DescriptionFilterOpts struct {
	// Term is a substring or phrase to match (case-insensitive).
	Term string

	// MatchType is "match", "wild" (glob), or "regex". Default "match".
	MatchType string

	// TypeIDs filters by description type SCTIDs (any-of). Empty = no filter.
	// Examples: 900000000000003001 (FSN), 900000000000013009 (synonym).
	TypeIDs []string

	// Languages filters by language codes (any-of). Empty = no filter.
	Languages []string

	// Active filters by description active flag. Nil = don't filter.
	Active *bool

	// ModuleIDs filters by module SCTIDs (any-of).
	ModuleIDs []string

	// EffectiveTime filters by effectiveTime (YYYYMMDD) with comparison operator.
	EffectiveTime   string
	EffectiveTimeOp string // "=", "!=", "<", "<=", ">", ">="
}

// ConceptFilterOpts describes concept-level metadata filters.
// Slice fields use any-of semantics.
type ConceptFilterOpts struct {
	// Active filters by concept active flag. Nil = don't filter.
	Active *bool

	// DefinitionStatusIDs filters by definitionStatus SCTIDs (any-of).
	DefinitionStatusIDs []string

	// ModuleIDs filters by module SCTIDs (any-of).
	ModuleIDs []string

	// EffectiveTime + Op as in DescriptionFilterOpts.
	EffectiveTime   string
	EffectiveTimeOp string
}

// DialectFilterOpts describes dialect filter constraints for descriptions.
type DialectFilterOpts struct {
	Dialects []DialectEntryOpts
	Negate   bool
}

// DialectEntryOpts pairs a set of dialect refsets with an optional set of
// acceptabilities. Match if (any DialectID) AND (any AcceptabilityID); empty
// AcceptabilityIDs means any acceptability.
type DialectEntryOpts struct {
	DialectIDs       []string // SCTIDs of dialect language refsets (any-of)
	AcceptabilityIDs []string // optional; nil/empty = any acceptability
}

// MemberFilterOpts describes member-level field filter constraints.
type MemberFilterOpts struct {
	FieldName string
	Op        string // "=" or "!="
	ValueSet  Set    // pre-resolved concept IDs
}
