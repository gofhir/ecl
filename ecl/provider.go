package ecl

import "context"

// DataProvider abstracts the SNOMED CT data source for the ECL evaluator.
// Consumers implement this against their storage (PostgreSQL closure tables,
// in-memory maps, Elasticsearch, etc.).
//
// All methods take slices/sets to enable batch queries and avoid N+1 patterns.
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
	// ConceptExists returns the subset of the input that exists in the terminology.
	ConceptExists(ctx context.Context, conceptIDs []string) (Set, error)

	// AllConcepts returns all active concepts (used for the wildcard *).
	AllConcepts(ctx context.Context) (Set, error)

	// ── Attributes (for refinements) ───────────────────────────────────────
	// RelationshipTargets returns the union of target concept IDs of relationships
	// whose source is in sourceIDs and type is in typeIDs.
	RelationshipTargets(ctx context.Context, sourceIDs Set, typeIDs Set) (Set, error)

	// RelationshipSources returns the union of source concept IDs of relationships
	// whose target is in targetIDs and type is in typeIDs (for reverse flag "R").
	// When targetIDs is nil, all targets are considered (wildcard).
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
	// HistoricalAssociations expands a set of inactive concepts to their
	// historical replacements according to the given profile (MIN, MOD, MAX, ALL).
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
