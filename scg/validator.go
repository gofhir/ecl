package scg

import (
	"context"
	"fmt"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/sctid"
)

// ValidationResult reports the outcome of validating an SCG expression
// against the terminology.
type ValidationResult struct {
	// Valid is true if all concept references are well-formed (Verhoeff
	// check) and exist in the terminology.
	Valid bool

	// Issues lists validation errors found in the expression.
	Issues []Issue
}

// Issue represents a single validation error.
type Issue struct {
	// SCTID is the concept ID that failed validation.
	SCTID string

	// Kind is one of: "unknown" (not in terminology), "inactive" (exists
	// but not active — currently not checked here, reserved for future
	// use), or "invalid_sctid" (fails Verhoeff check digit).
	Kind string

	// Path describes where the error occurred, e.g.
	//   focus[0]
	//   refinement[2].attr[0].name
	//   refinement[1].attr[1].value
	Path string

	// Message is a human-readable error message.
	Message string
}

// Issue Kind constants.
const (
	IssueKindUnknown      = "unknown"
	IssueKindInactive     = "inactive"
	IssueKindInvalidSCTID = "invalid_sctid"
)

// Validate validates a parsed SCG expression against the terminology.
//
// It performs two checks:
//
//  1. Every SCTID must pass the Verhoeff check digit algorithm. Failures are
//     reported with Kind="invalid_sctid".
//  2. Every well-formed SCTID must exist in the terminology (queried in a
//     single batch via DataProvider.ConceptExists). Missing IDs are reported
//     with Kind="unknown".
//
// The expression is walked depth-first, including nested expressions.
func Validate(ctx context.Context, expr *Expression, provider ecl.DataProvider) (*ValidationResult, error) {
	if expr == nil {
		return nil, fmt.Errorf("scg: nil expression")
	}
	if provider == nil {
		return nil, fmt.Errorf("scg: nil data provider")
	}

	// Collect every (id, path) reference in the expression and pre-flag
	// any that fail the Verhoeff check.
	type ref struct {
		id   string
		path string
	}
	var refs []ref
	var issues []Issue

	collect := func(id, path string) {
		if !sctid.IsValid(id) {
			issues = append(issues, Issue{
				SCTID:   id,
				Kind:    IssueKindInvalidSCTID,
				Path:    path,
				Message: fmt.Sprintf("SCTID %q fails Verhoeff check", id),
			})
			return
		}
		refs = append(refs, ref{id: id, path: path})
	}
	walkExpression(expr, "", &collect)

	// Batch query the provider for existence.
	if len(refs) > 0 {
		ids := make([]string, 0, len(refs))
		seen := make(map[string]bool, len(refs))
		for _, r := range refs {
			if !seen[r.id] {
				seen[r.id] = true
				ids = append(ids, r.id)
			}
		}
		existing, err := provider.ConceptExists(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("scg: ConceptExists failed: %w", err)
		}
		for _, r := range refs {
			if !existing.Contains(r.id) {
				issues = append(issues, Issue{
					SCTID:   r.id,
					Kind:    IssueKindUnknown,
					Path:    r.path,
					Message: fmt.Sprintf("SCTID %s not found in terminology", r.id),
				})
			}
		}
	}

	return &ValidationResult{
		Valid:  len(issues) == 0,
		Issues: issues,
	}, nil
}

// walkExpression visits every concept reference in expr, prefixing paths
// with the supplied parent path.
func walkExpression(expr *Expression, prefix string, collect *func(id, path string)) {
	for i, fc := range expr.FocusConcepts {
		path := joinPath(prefix, fmt.Sprintf("focus[%d]", i))
		(*collect)(fc.SCTID, path)
	}
	for gi, grp := range expr.Refinements {
		groupPath := joinPath(prefix, fmt.Sprintf("refinement[%d]", gi))
		for ai, attr := range grp.Attributes {
			attrPath := fmt.Sprintf("%s.attr[%d]", groupPath, ai)
			(*collect)(attr.Name.SCTID, attrPath+".name")
			walkAttributeValue(attr.Value, attrPath+".value", collect)
		}
	}
}

// walkAttributeValue visits a single attribute value, recursing into nested
// expressions as needed. Concrete values contribute no SCTID references.
func walkAttributeValue(v AttributeValue, path string, collect *func(id, path string)) {
	switch {
	case v.Concept != nil:
		(*collect)(v.Concept.SCTID, path)
	case v.Nested != nil:
		walkExpression(v.Nested, path, collect)
	case v.Concrete != nil:
		// No SCTIDs in concrete values.
	}
}

// joinPath joins two path segments with a "." if both are non-empty.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}
