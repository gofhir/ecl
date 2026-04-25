// Package mrcm implements SNOMED CT's Machine Readable Concept Model (MRCM).
//
// The MRCM defines, for each SNOMED CT attribute (relationship type), the set
// of focus concepts that may use the attribute (its domain) and the set of
// values it may take (its range). Both are expressed as ECL constraints.
//
// MRCM constraints are typically distributed by SNOMED International as RF2
// reference sets (e.g. the MRCM Attribute Domain refset and the MRCM Attribute
// Range refset). For convenience this package accepts MRCM data via a JSON
// loader; callers may also construct a Model programmatically.
//
// Specification:
// https://confluence.ihtsdotools.org/display/DOCMRCM
package mrcm

// Standard MRCM rule strength SCTIDs.
const (
	// RuleStrengthMandatory means the rule must hold for an expression to be valid.
	RuleStrengthMandatory = "723597001"

	// RuleStrengthOptional means the rule is advisory.
	RuleStrengthOptional = "723598006"
)

// Model is the complete MRCM ruleset.
//
// The zero value is a valid empty model.
type Model struct {
	// Domains lists attribute domain constraints.
	Domains []AttributeDomain

	// Ranges lists attribute range constraints.
	Ranges []AttributeRange
}

// AttributeDomain restricts which concepts can have a given attribute.
type AttributeDomain struct {
	// AttributeID is the SCTID of the attribute (e.g. 363698007 for Finding site).
	AttributeID string

	// DomainECL is an ECL expression matching valid focus concepts (e.g. "<< 404684003").
	DomainECL string

	// Grouped indicates whether the attribute must appear in a relationship group.
	Grouped bool

	// Cardinality is the allowed cardinality for the attribute on a focus concept.
	Cardinality Cardinality

	// InGroupCardinality is the cardinality within a single relationship group.
	InGroupCardinality Cardinality

	// RuleStrengthID indicates whether the rule is mandatory (723597001) or
	// optional (723598006). Default is mandatory.
	RuleStrengthID string
}

// AttributeRange restricts the values an attribute may take.
type AttributeRange struct {
	// AttributeID is the SCTID of the attribute.
	AttributeID string

	// RangeECL is an ECL expression matching valid values (e.g. "<< 442083009").
	RangeECL string

	// RangeRuleECL is an alternative form (kept verbatim for tooling).
	RangeRuleECL string

	// RuleStrengthID is mandatory or optional, like AttributeDomain.
	RuleStrengthID string
}

// Cardinality describes [min..max] occurrence constraints.
//
// A Max value of -1 means "unbounded" (rendered as "*" in MRCM cardinality
// strings such as "0..*" or "1..*").
type Cardinality struct {
	Min int
	Max int // -1 = unbounded ("*")
}

// FindDomains returns all domain rules for the given attribute. The returned
// slice shares storage with the model and must not be modified by the caller.
func (m *Model) FindDomains(attributeID string) []AttributeDomain {
	if m == nil {
		return nil
	}
	var out []AttributeDomain
	for _, d := range m.Domains {
		if d.AttributeID == attributeID {
			out = append(out, d)
		}
	}
	return out
}

// FindRanges returns all range rules for the given attribute. The returned
// slice shares storage with the model and must not be modified by the caller.
func (m *Model) FindRanges(attributeID string) []AttributeRange {
	if m == nil {
		return nil
	}
	var out []AttributeRange
	for _, r := range m.Ranges {
		if r.AttributeID == attributeID {
			out = append(out, r)
		}
	}
	return out
}
