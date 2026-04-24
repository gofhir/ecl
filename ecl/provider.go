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

	// ── History supplements (v2.0) ─────────────────────────────────────────
	// HistoricalAssociations expands a set of inactive concepts to their
	// historical replacements according to the given profile (MIN, MOD, MAX, ALL).
	HistoricalAssociations(ctx context.Context, conceptIDs Set, profile string) (Set, error)
}

// Relationship is a single attribute relationship of a concept.
type Relationship struct {
	TypeID   string
	TargetID string
	GroupNum int
}

// ConcreteValue is a concrete (non-concept) attribute value.
type ConcreteValue struct {
	// Kind is "integer", "decimal", "string", or "boolean".
	Kind string
	// Value holds the raw value as a string for the caller to parse.
	Value string
}

// DescriptionFilterOpts describes which descriptions to match.
// Empty fields are ignored (no filter on that dimension).
type DescriptionFilterOpts struct {
	// Term is a substring or phrase to match (case-insensitive).
	Term string

	// MatchType is "match", "wild" (glob), or "regex". Default "match".
	MatchType string

	// TypeID filters by description type (SCTID), e.g. synonym, FSN.
	TypeID string

	// Language filters by language code (e.g., "en", "es").
	Language string

	// Active filters by description active flag. Nil = don't filter.
	Active *bool

	// ModuleID filters by module SCTID.
	ModuleID string

	// EffectiveTime filters by effectiveTime (YYYYMMDD) with comparison operator.
	EffectiveTime   string
	EffectiveTimeOp string // "=", "!=", "<", "<=", ">", ">="
}

// ConceptFilterOpts describes concept-level metadata filters.
type ConceptFilterOpts struct {
	// Active filters by concept active flag. Nil = don't filter.
	Active *bool

	// DefinitionStatusID filters by definitionStatus SCTID.
	DefinitionStatusID string

	// ModuleID filters by module SCTID.
	ModuleID string

	// EffectiveTime + Op as in DescriptionFilterOpts.
	EffectiveTime   string
	EffectiveTimeOp string
}
