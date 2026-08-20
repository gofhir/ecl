// Package ast defines the AST node types produced by parsing SNOMED CT ECL expressions.
package ast

// Expression is the marker interface for all ECL AST nodes.
type Expression interface {
	eclNode()
}

// --- Hierarchy operators ---.

// DescendantOf represents the < constraint operator.
type DescendantOf struct{ Operand Expression }

// DescendantOrSelfOf represents the << constraint operator.
type DescendantOrSelfOf struct{ Operand Expression }

// ChildOf represents the <! Constraint operator.
type ChildOf struct{ Operand Expression }

// ChildOrSelfOf represents the <<! Constraint operator.
type ChildOrSelfOf struct{ Operand Expression }

// AncestorOf represents the > constraint operator.
type AncestorOf struct{ Operand Expression }

// AncestorOrSelfOf represents the >> constraint operator.
type AncestorOrSelfOf struct{ Operand Expression }

// ParentOf represents the >! Constraint operator.
type ParentOf struct{ Operand Expression }

// ParentOrSelfOf represents the >>! Constraint operator.
type ParentOrSelfOf struct{ Operand Expression }

// --- Set operators ---.

// And represents a conjunction (AND / ,) of two expression constraints.
type And struct{ Left, Right Expression }

// Or represents a disjunction (OR) of two expression constraints.
type Or struct{ Left, Right Expression }

// Minus represents an exclusion (MINUS) of two expression constraints.
type Minus struct{ Left, Right Expression }

// --- Primitives ---.

// ConceptRef represents a SNOMED CT concept reference with an optional display term.
type ConceptRef struct {
	ID   string
	Term string // optional display term (from |...|)
}

// Any represents the wildcard (*) matching all concepts.
type Any struct{}

// Nested represents a parenthesised sub-expression.
type Nested struct{ Inner Expression }

// --- MemberOf ---.

// MemberOf represents the ^ (member of) refset operator.
type MemberOf struct {
	Operand Expression
	Fields  []string // optional field names from ^[field1,field2]
}

// --- RefsetContainingAny ---.

// RefsetContainingAny represents the ^R operator.
type RefsetContainingAny struct{ Operand Expression }

// --- Dot notation ---.

// DotExpression represents attribute navigation via the dot (.) Operator.
type DotExpression struct {
	Source    Expression
	Attribute Expression
}

// --- Refinements ---.

// Refined represents an expression constraint with a colon-separated refinement.
type Refined struct {
	Focus      Expression
	Refinement *Refinement
}

// Refinement holds attribute constraints, optionally grouped, with conjunction/disjunction.
type Refinement struct {
	// AttrSet is the boolean tree of the refinement's ungrouped attribute
	// clauses. Prefer it over Ungrouped: a flat slice cannot express OR, so
	// `a = x OR b = y` and `a = x, b = y` were indistinguishable.
	AttrSet *AttributeSet

	Groups []*AttributeGroup

	// Nested is a parenthesised sub-refinement, which the grammar allows as the
	// first operand of a refinement. It is kept as its own node so the
	// parentheses survive: `({A} OR {B}) , C` must stay distinguishable from
	// `{A} , ({B} OR C)`.
	Nested *Refinement

	// Deprecated: use AttrSet. Kept populated for compatibility with readers
	// written against v1.1; it flattens AND and OR into one list and will be
	// removed in v2.
	Ungrouped []*Attribute

	// Conjunction and Disjunction hold the sub-refinements of a
	// conjunctionRefinementSet or disjunctionRefinementSet. The grammar admits
	// one or the other at a level, never both.
	Conjunction []*Refinement
	Disjunction []*Refinement
}

// AttrSetOp is the boolean operator joining the items of an AttributeSet.
type AttrSetOp string

// Operators for AttributeSet.
const (
	AttrSetAnd AttrSetOp = "AND"
	AttrSetOr  AttrSetOp = "OR"
)

// AttributeSet is a boolean tree of attribute clauses. It preserves the operator
// and the parenthesised nesting the ECL grammar allows, which a flat
// []*Attribute cannot express.
//
// Exactly one of Attr or Items is set:
//   - Attr:  a single attribute clause (a leaf)
//   - Items: nested sets combined with Op
//
// Note there is no group leaf. The grammar keeps the two levels apart:
// `subrefinement` admits an eclattributegroup, but `subattributeset` does not,
// so a group can never appear inside an attribute set — it arrives through
// Refinement.Groups or through a sibling sub-refinement instead.
type AttributeSet struct {
	Op    AttrSetOp
	Attr  *Attribute
	Items []*AttributeSet
}

// AttributeGroup represents a curly-brace-grouped set of attributes with optional cardinality.
type AttributeGroup struct {
	// AttrSet is the boolean tree of the group's clauses. Prefer it over Attrs,
	// which cannot express OR.
	AttrSet *AttributeSet

	// Deprecated: use AttrSet. Kept populated for compatibility; removed in v2.
	Attrs       []*Attribute
	Cardinality *Cardinality
}

// Attribute represents a single attribute constraint (name op value).
type Attribute struct {
	Name        Expression
	Op          string     // "=", "!=", "<", "<=", ">", ">="
	Value       Expression // can be ConceptRef, nested Expression, ConcreteValue, or Any
	Reverse     bool       // R prefix
	Cardinality *Cardinality
}

// Cardinality represents a [min..max] cardinality constraint.
type Cardinality struct {
	Min int
	Max int // -1 = unbounded (*)
}

// --- Concrete values ---.

// StringValue represents a quoted string literal in an attribute comparison.
type StringValue struct{ Value string }

// IntegerValue represents an integer literal in an attribute comparison.
type IntegerValue struct{ Value int }

// DecimalValue represents a decimal literal in an attribute comparison.
type DecimalValue struct{ Value float64 }

// BooleanValue represents a boolean literal in an attribute comparison.
type BooleanValue struct{ Value bool }

// --- Filters ---.

// Filtered represents an expression constraint with one or more filter constraints.
type Filtered struct {
	Operand Expression
	Filters []Filter
}

// Filter is a marker interface for filter constraint nodes.
type Filter interface {
	Expression
	filterKind() string
}

// SearchTerm is one search term of a term filter, with the search style the
// grammar declared for it.
type SearchTerm struct {
	// Text is the term with its escape sequences decoded, except that a wild
	// term keeps `\*` as-is so a literal asterisk stays distinguishable from a
	// wildcard.
	Text string

	// MatchType is "match" (word match, the default) or "wild" (glob).
	MatchType string
}

// TermFilter represents a description term filter ({{ term = "..." }}).
//
// The grammar allows a SET of terms — `{{ term = ("heart" "attack") }}` — with
// any-of semantics, and each member may declare its own search style. Terms
// carries them all.
type TermFilter struct {
	Op string // "=" or "!="

	// Terms holds every search term, with any-of semantics.
	Terms []SearchTerm

	// Deprecated: use Terms. Holds the first term, and cannot represent a set.
	Term string
	// Deprecated: use Terms. Holds the first term's match type.
	MatchType string
}

// TypeFilter represents a description type filter.
type TypeFilter struct {
	Op    string       // "=" or "!="
	Types []Expression // concept references for description type IDs
}

// LanguageFilter represents a description language filter.
type LanguageFilter struct {
	Op        string
	Languages []string
}

// DialectFilter represents a description dialect filter.
type DialectFilter struct {
	Op       string
	Dialects []DialectEntry
}

// DialectEntry pairs a dialect (concept ref or alias) with its optional
// acceptabilities.
type DialectEntry struct {
	Dialect Expression // concept ref or alias

	// Acceptabilities holds every acceptability the entry names, with any-of
	// semantics. The grammar allows a set: `(preferred acceptable)`.
	Acceptabilities []Expression

	// Deprecated: use Acceptabilities. Holds the first one only.
	Acceptability Expression
}

// ActiveFilter represents an active status filter.
type ActiveFilter struct {
	Value bool
}

// ModuleFilter represents a module filter.
type ModuleFilter struct {
	Op string

	// Modules holds every module the filter names, with any-of semantics. The
	// grammar allows a set: `{{ C moduleId = (900000000000207008 900000000000012004) }}`.
	Modules []Expression

	// Deprecated: use Modules. Holds the first module only.
	Module Expression
}

// EffectiveTimeFilter represents an effective time filter.
type EffectiveTimeFilter struct {
	Op string

	// Values holds every time value the filter names, with any-of semantics.
	Values []string

	// Deprecated: use Values. Holds the first value only.
	Value string
}

// ConceptActiveFilter is an alias for ActiveFilter used in concept filter constraints.
type ConceptActiveFilter = ActiveFilter

// ConceptModuleFilter is an alias for ModuleFilter used in concept filter constraints.
type ConceptModuleFilter = ModuleFilter

// ConceptEffectiveTimeFilter is an alias for EffectiveTimeFilter in concept filter constraints.
type ConceptEffectiveTimeFilter = EffectiveTimeFilter

// DefinitionStatusFilter represents a concept definition status filter.
type DefinitionStatusFilter struct {
	Op string

	// Values holds every definition status the filter names, with any-of
	// semantics.
	Values []Expression

	// Deprecated: use Values. Holds the first value only.
	Value Expression
}

// DescriptionIDFilter represents a description identifier filter
// ({{ D id = 123456789012 }}).
type DescriptionIDFilter struct {
	Op  string // "=" or "!="
	IDs []string
}

// MemberFieldFilter represents a member field filter in member filter constraints.
type MemberFieldFilter struct {
	FieldName string
	Op        string
	Value     Expression // concept ref set or expression constraint
}

// --- History supplement ---.

// HistorySupplement represents the {{ +HISTORY }} supplement.
type HistorySupplement struct {
	Operand Expression
	Profile string // "HISTORY-MIN", "HISTORY-MOD", "HISTORY-MAX", or "" for default
}

// --- v2.2 ---.

// AltIdentifier represents an alternative identifier (scheme#code).
type AltIdentifier struct {
	Scheme string
	Code   string
	Term   string
}

// Top represents the !!> (top) constraint operator.
type Top struct {
	Operand Expression
}

// Bottom represents the !!< (bottom) constraint operator.
type Bottom struct {
	Operand Expression
}

// --- eclNode() implementations ---.

func (*DescendantOf) eclNode()           {}
func (*DescendantOrSelfOf) eclNode()     {}
func (*ChildOf) eclNode()                {}
func (*ChildOrSelfOf) eclNode()          {}
func (*AncestorOf) eclNode()             {}
func (*AncestorOrSelfOf) eclNode()       {}
func (*ParentOf) eclNode()               {}
func (*ParentOrSelfOf) eclNode()         {}
func (*And) eclNode()                    {}
func (*Or) eclNode()                     {}
func (*Minus) eclNode()                  {}
func (*ConceptRef) eclNode()             {}
func (*Any) eclNode()                    {}
func (*Nested) eclNode()                 {}
func (*MemberOf) eclNode()               {}
func (*RefsetContainingAny) eclNode()    {}
func (*DotExpression) eclNode()          {}
func (*Refined) eclNode()                {}
func (*Refinement) eclNode()             {}
func (*AttributeSet) eclNode()           {}
func (*AttributeGroup) eclNode()         {}
func (*Attribute) eclNode()              {}
func (*Cardinality) eclNode()            {}
func (*StringValue) eclNode()            {}
func (*IntegerValue) eclNode()           {}
func (*DecimalValue) eclNode()           {}
func (*BooleanValue) eclNode()           {}
func (*Filtered) eclNode()               {}
func (*SearchTerm) eclNode()             {}
func (*TermFilter) eclNode()             {}
func (*DescriptionIDFilter) eclNode()    {}
func (*TypeFilter) eclNode()             {}
func (*LanguageFilter) eclNode()         {}
func (*DialectFilter) eclNode()          {}
func (*DialectEntry) eclNode()           {}
func (*ActiveFilter) eclNode()           {}
func (*ModuleFilter) eclNode()           {}
func (*EffectiveTimeFilter) eclNode()    {}
func (*DefinitionStatusFilter) eclNode() {}
func (*MemberFieldFilter) eclNode()      {}
func (*HistorySupplement) eclNode()      {}
func (*AltIdentifier) eclNode()          {}
func (*Top) eclNode()                    {}
func (*Bottom) eclNode()                 {}

// --- filterKind() implementations ---.

func (*TermFilter) filterKind() string             { return "term" }
func (*DescriptionIDFilter) filterKind() string    { return "descriptionId" }
func (*TypeFilter) filterKind() string             { return "type" }
func (*LanguageFilter) filterKind() string         { return "language" }
func (*DialectFilter) filterKind() string          { return "dialect" }
func (*ActiveFilter) filterKind() string           { return "active" }
func (*ModuleFilter) filterKind() string           { return "module" }
func (*EffectiveTimeFilter) filterKind() string    { return "effectiveTime" }
func (*DefinitionStatusFilter) filterKind() string { return "definitionStatus" }
func (*MemberFieldFilter) filterKind() string      { return "memberField" }
