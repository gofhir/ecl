// Package scg implements the SNOMED CT Compositional Grammar (SCG) for
// post-coordinated expressions.
//
// SCG is a much smaller grammar than ECL (~15 rules) and is parsed via a
// hand-written recursive descent parser with no external dependencies.
//
// Specification:
// https://docs.snomed.org/snomed-ct-specifications/snomed-ct-compositional-grammar-specification
package scg

// Definition status prefix values.
const (
	// DefStatusEquivalent is the "===" prefix meaning the expression is
	// equivalent to the named definition. This is the default when no prefix
	// is given, per the SCG specification.
	DefStatusEquivalent = "==="

	// DefStatusSubtype is the "<<<" prefix meaning the expression is a
	// subtype of the named definition.
	DefStatusSubtype = "<<<"
)

// Expression is the top-level SCG AST node.
type Expression struct {
	// DefinitionStatus is "===" (equivalent, the default) or "<<<" (subtype).
	DefinitionStatus string

	// FocusConcepts is one or more concept references joined by "+".
	FocusConcepts []ConceptRef

	// Refinements are the attribute groups after ":".
	// An empty slice means no refinement.
	// Ungrouped attributes are modeled as a single AttributeGroup with Grouped=false.
	Refinements []AttributeGroup
}

// ConceptRef is a SNOMED CT concept reference with optional term.
type ConceptRef struct {
	SCTID string
	Term  string
}

// AttributeGroup is either a brace-enclosed group or an ungrouped attribute set.
type AttributeGroup struct {
	Grouped    bool
	Attributes []Attribute
}

// Attribute is a single name=value pair.
type Attribute struct {
	Name  ConceptRef
	Value AttributeValue
}

// AttributeValue is one of: ConceptRef, nested Expression, or ConcreteValue.
// Exactly one of the three pointer fields will be non-nil for a well-formed
// attribute value.
type AttributeValue struct {
	Concept  *ConceptRef
	Nested   *Expression
	Concrete *ConcreteValue
}

// ConcreteValue is a non-concept attribute value.
type ConcreteValue struct {
	// Kind is one of: "integer", "decimal", "string", "boolean".
	Kind   string
	Int    int64
	Float  float64
	String string
	Bool   bool
}
