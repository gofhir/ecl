package mrcm

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/scg"
)

// Issue Kind constants.
const (
	IssueKindDomainViolation      = "domain_violation"
	IssueKindRangeViolation       = "range_violation"
	IssueKindCardinalityViolation = "cardinality_violation"
	IssueKindGroupedViolation     = "grouped_violation"
	IssueKindUngroupedViolation   = "ungrouped_violation"
	IssueKindUnknownAttribute     = "unknown_attribute"

	// IssueKindInvalidRule reports a rule in the model that could not be
	// applied — typically an unparseable domain or range ECL. It is an issue
	// rather than a hard error so one broken rule cannot hide the violations
	// already found, nor the rules that are fine.
	IssueKindInvalidRule = "invalid_rule"
)

// Result reports the outcome of validating an SCG expression against an MRCM
// model.
type Result struct {
	// Valid is true if no issues were reported.
	Valid bool

	// Issues lists MRCM rule violations found in the expression.
	Issues []Issue
}

// Issue describes a single MRCM validation failure.
type Issue struct {
	// Kind is one of the IssueKind* constants.
	Kind string

	// AttributeID is the SCTID of the relevant attribute.
	AttributeID string

	// FocusID is the focus concept (for domain violations).
	FocusID string

	// ValueID is the attribute value (for range violations).
	ValueID string

	// Message is human-readable.
	Message string

	// Path describes the location in the expression, e.g.
	//   refinement[0].attr[1]
	//   refinement[1].attr[0].value (nested path)
	Path string
}

// Validate validates an SCG expression against an MRCM model.
//
// For each focus concept of the expression, the validator walks the
// refinement attributes and:
//
//   - Looks up domain rules for the attribute in the model.
//   - Evaluates each rule's domain ECL via the ECL engine and checks that the
//     focus concept is a member of the result. Otherwise reports a
//     domain_violation.
//   - Checks the rule's grouped flag against the SCG group structure.
//     Mismatches yield grouped_violation or ungrouped_violation.
//   - Looks up range rules for the attribute. For ConceptRef values it
//     evaluates the range ECL and checks membership; misses are
//     range_violation. Concrete values and nested expressions are not subject
//     to ECL range checks.
//   - Recurses into nested SCG expressions (using the nested expression's own
//     focus concepts as focus).
//
// Cardinality is checked against the model's rules rather than against the
// attributes present in the expression, so a mandatory attribute that is missing
// entirely is reported. Multiple domain rows for one attribute are alternatives:
// the focus concept only has to be in one of them. In-group cardinality
// (AttributeDomain.InGroupCardinality) is loaded but not yet enforced.
//
// A nil expression, model, or provider returns an error.
func Validate(ctx context.Context, expr *scg.Expression, model *Model, provider ecl.DataProvider) (*Result, error) {
	if expr == nil {
		return nil, fmt.Errorf("mrcm: nil expression")
	}
	if model == nil {
		return nil, fmt.Errorf("mrcm: nil model")
	}
	if provider == nil {
		return nil, fmt.Errorf("mrcm: nil data provider")
	}

	v := &validator{
		ctx:      ctx,
		model:    model,
		provider: provider,
		// memo caches ECL expression -> evaluated set, scoped to this run.
		memo: make(map[string]ecl.Set),
	}

	if err := v.validateExpression(expr, ""); err != nil {
		return nil, err
	}

	return &Result{
		Valid:  len(v.issues) == 0,
		Issues: v.issues,
	}, nil
}

// validator carries the per-Validate-call state.
type validator struct {
	ctx      context.Context
	model    *Model
	provider ecl.DataProvider
	issues   []Issue
	memo     map[string]ecl.Set
}

// validateExpression walks every focus concept and refinement attribute of
// expr and applies MRCM rules. The pathPrefix is a dotted path that describes
// the location of expr inside a parent expression (empty for the top level).
func (v *validator) validateExpression(expr *scg.Expression, pathPrefix string) error {
	// Per-attribute counts (across all groups) for the cardinality check.
	counts := make(map[string]int)
	for _, group := range expr.Refinements {
		for _, attr := range group.Attributes {
			counts[attr.Name.SCTID]++
		}
	}

	for _, focus := range expr.FocusConcepts {
		for gi, group := range expr.Refinements {
			for ai, attr := range group.Attributes {
				attrPath := joinPath(pathPrefix, fmt.Sprintf("refinement[%d].attr[%d]", gi, ai))
				if err := v.validateAttribute(focus.SCTID, attr, group.Grouped, attrPath); err != nil {
					return err
				}
			}
		}
		if err := v.validateCardinality(focus.SCTID, counts, pathPrefix); err != nil {
			return err
		}
	}

	// Nested expressions are validated once, OUTSIDE the focus loop: they have
	// their own focus concepts, so recursing per focus emitted the same issue
	// once per focus of the parent (2^depth copies for nested expressions).
	for gi, group := range expr.Refinements {
		for ai, attr := range group.Attributes {
			if attr.Value.Nested == nil {
				continue
			}
			attrPath := joinPath(pathPrefix, fmt.Sprintf("refinement[%d].attr[%d]", gi, ai))
			if err := v.validateExpression(attr.Value.Nested, attrPath+".value"); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateCardinality checks the per-attribute counts against the cardinality of
// the domain rules applicable to the focus concept.
//
// It walks the MODEL's rules rather than the counts map. Iterating the counts
// could only ever see attributes that are PRESENT, so `count < Min` was
// unreachable for a mandatory attribute that was missing — the very case the
// minimum exists to catch. A rule with Min:1 and the attribute absent reported
// Valid=true with no issues.
func (v *validator) validateCardinality(focusID string, counts map[string]int, pathPrefix string) error {
	for attrID, domains := range v.model.AllDomains() {
		applicable := v.applicableDomains(focusID, attrID, domains)
		if len(applicable) == 0 {
			// The attribute does not apply to this focus concept, so its
			// cardinality says nothing about this expression.
			continue
		}
		count := counts[attrID]
		for _, r := range applicable {
			if count < r.Cardinality.Min {
				v.issues = append(v.issues, Issue{
					Kind:        IssueKindCardinalityViolation,
					AttributeID: attrID,
					FocusID:     focusID,
					Message: fmt.Sprintf("attribute %s occurs %d time(s) on focus %s, MRCM requires at least %d",
						attrID, count, focusID, r.Cardinality.Min),
					Path: pathPrefix,
				})
			}
			if r.Cardinality.Max >= 0 && count > r.Cardinality.Max {
				v.issues = append(v.issues, Issue{
					Kind:        IssueKindCardinalityViolation,
					AttributeID: attrID,
					FocusID:     focusID,
					Message: fmt.Sprintf("attribute %s occurs %d time(s) on focus %s, MRCM allows at most %d",
						attrID, count, focusID, r.Cardinality.Max),
					Path: pathPrefix,
				})
			}
		}
	}
	return nil
}

// applicableDomains returns the mandatory domain rows of an attribute whose
// domain ECL contains the focus concept. The rows of the MRCM Attribute Domain
// refset are alternatives, so being in any one of them makes the attribute
// applicable.
//
// An invalid domain ECL is reported as an issue rather than aborting the whole
// validation: a broken rule must not hide the violations already collected, nor
// the rules that are fine.
func (v *validator) applicableDomains(focusID, attrID string, domains []AttributeDomain) []AttributeDomain {
	var applicable []AttributeDomain
	for _, rule := range domains {
		if rule.RuleStrengthID != "" && rule.RuleStrengthID != RuleStrengthMandatory {
			continue
		}
		ok, err := v.eclContains(rule.DomainECL, focusID)
		if err != nil {
			v.issues = append(v.issues, Issue{
				Kind:        IssueKindInvalidRule,
				AttributeID: attrID,
				FocusID:     focusID,
				Message: fmt.Sprintf("domain ECL %q of attribute %s could not be evaluated: %v",
					rule.DomainECL, attrID, err),
			})
			continue
		}
		if ok {
			applicable = append(applicable, rule)
		}
	}
	return applicable
}

// domainSummary renders the domain ECLs of a rule set for an error message.
func domainSummary(domains []AttributeDomain) string {
	ecls := make([]string, 0, len(domains))
	for _, d := range domains {
		ecls = append(ecls, d.DomainECL)
	}
	return strings.Join(ecls, " OR ")
}

// validateAttribute applies MRCM rules to a single attribute of a focus
// concept.
func (v *validator) validateAttribute(focusID string, attr scg.Attribute, grouped bool, path string) error {
	attrID := attr.Name.SCTID

	domains := v.model.FindDomains(attrID)
	ranges := v.model.FindRanges(attrID)

	if len(domains) == 0 && len(ranges) == 0 {
		v.issues = append(v.issues, Issue{
			Kind:        IssueKindUnknownAttribute,
			AttributeID: attrID,
			FocusID:     focusID,
			Message:     fmt.Sprintf("attribute %s has no MRCM rules", attrID),
			Path:        path,
		})
		return nil
	}

	// Domain & grouped checks.
	//
	// The MRCM Attribute Domain refset holds ONE ROW PER DOMAIN (and per
	// contentTypeId) for an attribute, so the rows are alternatives: the focus
	// concept has to be in at least one of them. Applicability and conformance
	// used to be conflated, requiring the focus to be in EVERY row and emitting a
	// domain_violation for each one it was not in. With the two rows the refset
	// distributes for Finding site, a valid expression came back invalid with a
	// spurious violation.
	//
	// So: collect the applicable rows first, report a domain violation only if
	// none apply, and check grouped/cardinality against the applicable rows alone.
	applicable := v.applicableDomains(focusID, attrID, domains)

	if len(domains) > 0 && len(applicable) == 0 {
		v.issues = append(v.issues, Issue{
			Kind:        IssueKindDomainViolation,
			AttributeID: attrID,
			FocusID:     focusID,
			Message: fmt.Sprintf("focus concept %s is not in any domain of attribute %s (%s)",
				focusID, attrID, domainSummary(domains)),
			Path: path + ".name",
		})
	}

	for _, rule := range applicable {
		// Grouped / ungrouped check.
		if rule.Grouped && !grouped {
			v.issues = append(v.issues, Issue{
				Kind:        IssueKindUngroupedViolation,
				AttributeID: attrID,
				FocusID:     focusID,
				Message: fmt.Sprintf("attribute %s must appear inside a relationship group",
					attrID),
				Path: path,
			})
		} else if !rule.Grouped && grouped {
			v.issues = append(v.issues, Issue{
				Kind:        IssueKindGroupedViolation,
				AttributeID: attrID,
				FocusID:     focusID,
				Message: fmt.Sprintf("attribute %s must not appear inside a relationship group",
					attrID),
				Path: path,
			})
		}
	}

	// Range checks (only for concept-typed values; concrete values are not
	// subject to ECL ranges in this implementation, and nested expressions
	// are validated by recursion).
	if attr.Value.Concept != nil {
		valueID := attr.Value.Concept.SCTID
		for _, rule := range ranges {
			if rule.RuleStrengthID != "" && rule.RuleStrengthID != RuleStrengthMandatory {
				continue
			}
			ok, err := v.eclContains(rule.RangeECL, valueID)
			if err != nil {
				return fmt.Errorf("evaluating range ECL for attribute %s: %w", attrID, err)
			}
			if !ok {
				v.issues = append(v.issues, Issue{
					Kind:        IssueKindRangeViolation,
					AttributeID: attrID,
					FocusID:     focusID,
					ValueID:     valueID,
					Message: fmt.Sprintf("value %s is not in range (%s) of attribute %s",
						valueID, rule.RangeECL, attrID),
					Path: path + ".value",
				})
			}
		}
	}

	return nil
}

// eclContains parses + evaluates the given ECL expression (with memoisation)
// and reports whether conceptID is a member of the result.
func (v *validator) eclContains(eclExpr, conceptID string) (bool, error) {
	set, err := v.evalECL(eclExpr)
	if err != nil {
		return false, err
	}
	if set == nil {
		return false, nil
	}
	return set.Contains(conceptID), nil
}

// evalECL parses and evaluates an ECL expression, memoising the result for
// the duration of the Validate call.
func (v *validator) evalECL(eclExpr string) (ecl.Set, error) {
	if set, ok := v.memo[eclExpr]; ok {
		return set, nil
	}
	parsed, err := ecl.Parse(eclExpr)
	if err != nil {
		return nil, fmt.Errorf("parse ECL %q: %w", eclExpr, err)
	}
	set, err := ecl.Evaluate(v.ctx, parsed, v.provider)
	if err != nil {
		return nil, fmt.Errorf("evaluate ECL %q: %w", eclExpr, err)
	}
	v.memo[eclExpr] = set
	return set, nil
}

// joinPath joins two path segments with "." if both are non-empty.
func joinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}
