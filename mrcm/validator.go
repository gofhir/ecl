package mrcm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/scg"
)

// errInvalidRule marks a failure caused by the MODEL — a rule whose ECL does not
// parse, or that names a construct the evaluator cannot handle. Those are
// reported as an invalid_rule Issue.
//
// It exists to keep that case apart from a provider or context failure, where the
// expression may be perfectly valid and we simply could not check it. Reporting
// the latter as an MRCM violation would tell the caller their data is wrong when
// the truth is that the backend is down.
var errInvalidRule = errors.New("invalid MRCM rule")

// Issue Kind constants.
const (
	IssueKindDomainViolation      = "domain_violation"
	IssueKindRangeViolation       = "range_violation"
	IssueKindCardinalityViolation = "cardinality_violation"

	// IssueKindInGroupCardinalityViolation reports too many distinct values of an
	// attribute inside ONE relationship group. It is separate from
	// cardinality_violation because the two constrain different things and an
	// expression can satisfy one while breaking the other: three finding sites in
	// three groups is fine under a concept cardinality of 0..*, and a violation
	// under an in-group cardinality of 0..1 only if two of them share a group.
	IssueKindInGroupCardinalityViolation = "in_group_cardinality_violation"
	IssueKindGroupedViolation            = "grouped_violation"
	IssueKindUngroupedViolation          = "ungrouped_violation"
	IssueKindUnknownAttribute            = "unknown_attribute"

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
// the focus concept only has to be in one of them.
//
// In-group cardinality (AttributeDomain.InGroupCardinality) is enforced per
// relationship group; see validateInGroupCardinality, which also records what the
// specification leaves open about the minimum.
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
		// domains is the model's rules grouped by attribute, built once: the
		// cardinality check needs it per focus concept and per nested
		// expression, and rebuilding the map each time is pure allocation.
		domains: model.AllDomains(),
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
	domains  map[string][]AttributeDomain
}

// validateExpression walks every focus concept and refinement attribute of
// expr and applies MRCM rules. The pathPrefix is a dotted path that describes
// the location of expr inside a parent expression (empty for the top level).
func (v *validator) validateExpression(expr *scg.Expression, pathPrefix string) error {
	// Per-attribute counts of DISTINCT VALUES across all groups, for the
	// concept-level cardinality check. See distinctValueCounts: the specification
	// counts values, not occurrences.
	counts := distinctValueCounts(allAttributes(expr))

	for _, focus := range expr.FocusConcepts {
		for gi, group := range expr.Refinements {
			for ai, attr := range group.Attributes {
				attrPath := joinPath(pathPrefix, fmt.Sprintf("refinement[%d].attr[%d]", gi, ai))
				if err := v.validateAttribute(focus.SCTID, attr, group.Grouped, attrPath); err != nil {
					return err
				}
			}
		}
		// Cardinality constrains a POSTCOORDINATED definition. A bare concept
		// reference asserts nothing about its own attributes — under the SCG
		// default it means "equivalent to this concept", whose definition is
		// whatever the terminology says — so there is nothing missing and nothing
		// to count. Checking it anyway reported a mandatory attribute as absent
		// for every precoordinated code.
		if len(expr.Refinements) > 0 {
			if err := v.validateCardinality(focus.SCTID, counts, pathPrefix); err != nil {
				return err
			}
			if err := v.validateInGroupCardinality(focus.SCTID, expr, pathPrefix); err != nil {
				return err
			}
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
	// Sorted, so Result.Issues is stable across runs: iterating the map directly
	// made the order vary, which no caller can diff or snapshot.
	attrIDs := make([]string, 0, len(v.domains))
	for attrID := range v.domains {
		attrIDs = append(attrIDs, attrID)
	}
	sort.Strings(attrIDs)

	for _, attrID := range attrIDs {
		domains := v.domains[attrID]
		// Only rules that could constrain THIS expression. Without the guard an
		// unevaluatable rule for an attribute the expression never mentions
		// emitted invalid_rule and flipped Valid=false — an accusation about a
		// part of the model the caller did not touch.
		if counts[attrID] == 0 && !v.hasMandatoryMinimum(domains) {
			continue
		}
		applicable, _, err := v.applicableDomains(focusID, attrID, domains)
		if err != nil {
			return err
		}
		if len(applicable) == 0 {
			// The attribute does not apply to this focus concept, so its
			// cardinality says nothing about this expression.
			continue
		}
		// The applicable rows are alternatives here too: the count only has to fit
		// ONE of them. Checking every row produced a spurious violation whenever
		// two applicable rows disagreed on Min or Max.
		count := counts[attrID]
		satisfied := false
		for _, r := range applicable {
			if count >= r.Cardinality.Min && (r.Cardinality.Max < 0 || count <= r.Cardinality.Max) {
				satisfied = true
				break
			}
		}
		if !satisfied {
			v.issues = append(v.issues, Issue{
				Kind:        IssueKindCardinalityViolation,
				AttributeID: attrID,
				FocusID:     focusID,
				Message: fmt.Sprintf("attribute %s occurs %d time(s) on focus %s, which fits none of its MRCM cardinalities (%s)",
					attrID, count, focusID, cardinalitySummary(applicable)),
				Path: pathPrefix,
			})
		}
	}
	return nil
}

// hasMandatoryMinimum reports whether any mandatory row of an attribute demands
// at least one occurrence. Only those matter for an attribute the expression does
// not mention: every other rule is vacuously satisfied by a count of zero.
func (v *validator) hasMandatoryMinimum(domains []AttributeDomain) bool {
	for _, r := range domains {
		if r.RuleStrengthID != "" && r.RuleStrengthID != RuleStrengthMandatory {
			continue
		}
		if r.Cardinality.Min > 0 {
			return true
		}
	}
	return false
}

// applicableDomains returns the mandatory domain rows of an attribute whose
// domain ECL contains the focus concept. The rows of the MRCM Attribute Domain
// refset are alternatives, so being in any one of them makes the attribute
// applicable.
//
// An invalid domain ECL is reported as an issue rather than aborting the whole
// validation: a broken rule must not hide the violations already collected, nor
// the rules that are fine.
func (v *validator) applicableDomains(focusID, attrID string, domains []AttributeDomain) (applicable []AttributeDomain, mandatory int, err error) {
	for _, rule := range domains {
		if rule.RuleStrengthID != "" && rule.RuleStrengthID != RuleStrengthMandatory {
			// Advisory rows are not enforced, and they must not be counted as
			// mandatory either: doing so made an attribute whose only rows are
			// optional look like a domain violation.
			continue
		}
		mandatory++
		ok, err := v.eclContains(rule.DomainECL, focusID)
		if err != nil {
			// A malformed rule is a defect in the MODEL, reported as an issue so
			// one bad rule cannot hide the rest of the report. A provider or
			// context failure is not: the expression may be perfectly valid and we
			// simply could not check it, so reporting it as invalid would be a
			// false accusation. Propagate those.
			if !errors.Is(err, errInvalidRule) {
				return nil, 0, fmt.Errorf("evaluating domain ECL of attribute %s: %w", attrID, err)
			}
			v.addIssueOnce(Issue{
				Kind:        IssueKindInvalidRule,
				AttributeID: attrID,
				Message: fmt.Sprintf("domain ECL %q of attribute %s could not be evaluated: %v",
					rule.DomainECL, attrID, err),
			})
			// The rule could not be checked, so it says nothing about the focus
			// concept. Do not let it stand in for "the focus is out of domain".
			mandatory--
			continue
		}
		if ok {
			applicable = append(applicable, rule)
		}
	}
	return applicable, mandatory, nil
}

// addIssueOnce appends an issue unless an identical one is already recorded.
//
// A model-level defect does not depend on the focus concept or on where in the
// expression it was noticed, so walking every attribute occurrence and every
// focus concept would otherwise report it several times.
func (v *validator) addIssueOnce(issue Issue) {
	for _, existing := range v.issues {
		if existing == issue {
			return
		}
	}
	v.issues = append(v.issues, issue)
}

// cardinalitySummary renders the cardinalities of a rule set for an error message.
func cardinalitySummary(rules []AttributeDomain) string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		maxVal := "*"
		if r.Cardinality.Max >= 0 {
			maxVal = fmt.Sprintf("%d", r.Cardinality.Max)
		}
		out = append(out, fmt.Sprintf("[%d..%s]", r.Cardinality.Min, maxVal))
	}
	return strings.Join(out, " or ")
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
	applicable, mandatory, err := v.applicableDomains(focusID, attrID, domains)
	if err != nil {
		return err
	}

	// Report a violation only when a mandatory rule was actually checked and none
	// of them contained the focus. Using len(domains) here made an attribute
	// whose rows are all advisory, or whose ECL failed to parse, come back as a
	// domain violation -- a claim about the caller's data that was never tested.
	if mandatory > 0 && len(applicable) == 0 {
		v.issues = append(v.issues, Issue{
			Kind:        IssueKindDomainViolation,
			AttributeID: attrID,
			FocusID:     focusID,
			Message: fmt.Sprintf("focus concept %s is not in any domain of attribute %s (%s)",
				focusID, attrID, domainSummary(domains)),
			Path: path + ".name",
		})
	}

	// Grouped / ungrouped. The applicable rows are ALTERNATIVES, so satisfying
	// one is enough. Requiring every row made an attribute unwritable whenever
	// two applicable rows disagreed on Grouped: both violations were reported at
	// once, and no expression could avoid them.
	if len(applicable) > 0 {
		satisfied := false
		for _, rule := range applicable {
			if rule.Grouped == grouped {
				satisfied = true
				break
			}
		}
		if !satisfied {
			kind, msg := IssueKindGroupedViolation, "attribute %s must not appear inside a relationship group"
			if !grouped {
				kind, msg = IssueKindUngroupedViolation, "attribute %s must appear inside a relationship group"
			}
			v.issues = append(v.issues, Issue{
				Kind:        kind,
				AttributeID: attrID,
				FocusID:     focusID,
				Message:     fmt.Sprintf(msg, attrID),
				Path:        path,
			})
		}
	}

	// Range checks (only for concept-typed values; concrete values are not
	// subject to ECL ranges in this implementation, and nested expressions
	// are validated by recursion).
	if attr.Value.Concept != nil {
		valueID := attr.Value.Concept.SCTID

		// Range rows are ALTERNATIVES, like domain rows: the MRCM Attribute Range
		// refset holds one row per contentTypeId, so the value has to satisfy at
		// least one. Requiring every row reported a spurious range_violation for
		// each row it did not match.
		var (
			checked   int
			satisfied bool
			ecls      []string
		)
		for _, rule := range ranges {
			if rule.RuleStrengthID != "" && rule.RuleStrengthID != RuleStrengthMandatory {
				continue
			}
			ok, err := v.eclContains(rule.RangeECL, valueID)
			if err != nil {
				if !errors.Is(err, errInvalidRule) {
					return fmt.Errorf("evaluating range ECL of attribute %s: %w", attrID, err)
				}
				v.addIssueOnce(Issue{
					Kind:        IssueKindInvalidRule,
					AttributeID: attrID,
					Message: fmt.Sprintf("range ECL %q of attribute %s could not be evaluated: %v",
						rule.RangeECL, attrID, err),
				})
				continue
			}
			checked++
			ecls = append(ecls, rule.RangeECL)
			if ok {
				satisfied = true
				break
			}
		}
		if checked > 0 && !satisfied {
			v.issues = append(v.issues, Issue{
				Kind:        IssueKindRangeViolation,
				AttributeID: attrID,
				FocusID:     focusID,
				ValueID:     valueID,
				Message: fmt.Sprintf("value %s is not in range (%s) of attribute %s",
					valueID, strings.Join(ecls, " OR "), attrID),
				Path: path + ".value",
			})
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
		return nil, fmt.Errorf("%w: parse ECL %q: %w", errInvalidRule, eclExpr, err)
	}
	set, err := ecl.Evaluate(v.ctx, parsed, v.provider)
	if err != nil {
		// A construct the evaluator does not support is a property of the rule;
		// anything else (provider failure, cancellation) is not.
		if errors.Is(err, ecl.ErrUnsupportedFeature) {
			return nil, fmt.Errorf("%w: evaluate ECL %q: %w", errInvalidRule, eclExpr, err)
		}
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

// validateInGroupCardinality checks each relationship group against the in-group
// cardinality of the domain rules applicable to the focus concept.
//
// The specification defines the field as
//
//	"The number of times the given attribute can be assigned a distinct
//	 (non-redundant) value within a single relationship group"
//
// which is a different constraint from attributeCardinality, not a refinement of
// it: three finding sites spread over three groups satisfy a concept cardinality
// of 0..* and an in-group cardinality of 0..1 at the same time, while the same
// three in ONE group satisfy the first and break the second. So the two checks are
// independent and report different Kinds.
//
// # Only grouped rules, only braced groups
//
// A rule with Grouped false describes an attribute that does not live in
// relationship groups, so its in-group cardinality constrains nothing. And SCG's
// ungrouped attribute set is not a relationship group — the same reading the
// evaluator applies to group 0 of PropertiesByGroup — so it is skipped. Counting
// it as a group would let `a = x, a = y` break an in-group maximum of 1 that says
// nothing about ungrouped attributes.
//
// # The minimum, and why it applies only where the attribute is
//
// The specification text settles the MAXIMUM — how many times an attribute "can
// be" assigned a value in a group — and does not say whether a MINIMUM applies to
// every group of the concept or only to groups where the attribute appears. The
// readings differ sharply: under "every group", an in-group cardinality of 1..1 on
// one attribute makes any group without that attribute a violation.
//
// The reference implementation settles it. SNOMED International's Snowstorm
// generates an ECL attribute rule from each MRCM row, rendering the concept
// cardinality as a GROUP cardinality and the in-group cardinality as the attribute
// cardinality INSIDE that group. From its own tests
// (MRCMDomainTemplatesAndRuleGeneratorTest):
//
//	<< 404684003 |Clinical finding|: [0..*] { [0..1] 255234002 |After| = ... }
//
// Under ECL — the language this repository implements — `[0..*]` on the group
// means zero or more MATCHING groups, so a concept whose groups do not contain the
// attribute satisfies the rule trivially. The inner `[0..1]` constrains only the
// groups that participate. An in-group minimum therefore never demands the
// attribute in a group that does not use it, which is what this enforces.
//
// The consequence, stated so nobody meets it by surprise: an in-group cardinality
// of 1..1 behaves like 0..1, and a minimum above 1 means "a group that uses this
// attribute needs at least that many distinct values".
//
// It is also the safe direction independently of the evidence. Under the other
// reading, an expression whose groups are legitimately heterogeneous — a finding
// with a site group and a separate "due to" group — is accused of violating a rule
// about an attribute it never put there, and this package has already had to
// remove one accusation of exactly that shape.
func (v *validator) validateInGroupCardinality(focusID string, expr *scg.Expression, pathPrefix string) error {
	for gi, group := range expr.Refinements {
		if !group.Grouped {
			continue
		}
		counts := distinctValueCounts(group.Attributes)

		// Sorted, so Result.Issues is stable across runs.
		attrIDs := make([]string, 0, len(counts))
		for attrID := range counts {
			attrIDs = append(attrIDs, attrID)
		}
		sort.Strings(attrIDs)

		for _, attrID := range attrIDs {
			applicable, _, err := v.applicableDomains(focusID, attrID, v.domains[attrID])
			if err != nil {
				return err
			}

			// Only rows that put the attribute in a group AND state an in-group
			// cardinality have anything to say.
			grouped := make([]AttributeDomain, 0, len(applicable))
			for _, r := range applicable {
				if r.Grouped && inGroupSpecified(r.InGroupCardinality) {
					grouped = append(grouped, r)
				}
			}
			if len(grouped) == 0 {
				continue
			}

			// Alternatives, as everywhere else: the count only has to fit ONE
			// applicable row. Requiring every row produces a spurious violation
			// whenever two rows disagree, which is what the domain and cardinality
			// checks were both fixed for.
			count := counts[attrID]
			satisfied := false
			for _, r := range grouped {
				if count >= r.InGroupCardinality.Min &&
					(r.InGroupCardinality.Max < 0 || count <= r.InGroupCardinality.Max) {
					satisfied = true
					break
				}
			}
			if satisfied {
				continue
			}

			v.addIssueOnce(Issue{
				Kind:        IssueKindInGroupCardinalityViolation,
				AttributeID: attrID,
				FocusID:     focusID,
				Message: fmt.Sprintf("attribute %s has %d distinct value(s) in one relationship group on focus %s, which fits none of its MRCM in-group cardinalities (%s)",
					attrID, count, focusID, inGroupCardinalitySummary(grouped)),
				Path: joinPath(pathPrefix, fmt.Sprintf("refinement[%d]", gi)),
			})
		}
	}
	return nil
}

// allAttributes flattens every attribute of an expression, grouped or not.
func allAttributes(expr *scg.Expression) []scg.Attribute {
	var out []scg.Attribute
	for _, group := range expr.Refinements {
		out = append(out, group.Attributes...)
	}
	return out
}

// distinctValueCounts counts, per attribute, how many DISTINCT values it is
// assigned.
//
// Both cardinality fields are defined over "a distinct (non-redundant) value", so
// `363698007 = 74281007, 363698007 = 74281007` is one value asserted twice, not
// two values. Counting occurrences instead reported a cardinality violation for an
// expression that merely repeated itself — which a normalizer would collapse and
// which asserts nothing extra.
//
// "Non-redundant" goes further than distinct: two values where one subsumes the
// other are redundant too, and collapsing those needs subsumption testing against
// the terminology. That is deliberately NOT done here. It would turn a pure
// counting step into one that fails when the provider is unavailable, and it can
// only ever lower a count — so the effect of leaving it out is a violation
// reported that a fully normalizing implementation would not report, never the
// reverse. Callers who need it should normalize before validating.
func distinctValueCounts(attrs []scg.Attribute) map[string]int {
	seen := make(map[string]map[string]struct{}, len(attrs))
	for _, attr := range attrs {
		key := valueKey(attr.Value)
		vals, ok := seen[attr.Name.SCTID]
		if !ok {
			vals = make(map[string]struct{}, 1)
			seen[attr.Name.SCTID] = vals
		}
		vals[key] = struct{}{}
	}

	out := make(map[string]int, len(seen))
	for attrID, vals := range seen {
		out[attrID] = len(vals)
	}
	return out
}

// valueKey renders an attribute value as a comparison key.
//
// The prefixes keep the three kinds apart, so a concept 74281007 and the string
// "74281007" are not accidentally the same value. A nested expression is keyed by
// its rendered form: two structurally identical nested definitions are one value,
// and anything else is a distinct one.
func valueKey(val scg.AttributeValue) string {
	switch {
	case val.Concept != nil:
		return "c:" + val.Concept.SCTID
	case val.Concrete != nil:
		return "k:" + val.Concrete.Kind + ":" + concreteText(val.Concrete)
	case val.Nested != nil:
		return "n:" + nestedKey(val.Nested)
	default:
		return "?"
	}
}

func concreteText(c *scg.ConcreteValue) string {
	switch c.Kind {
	case "integer":
		return strconv.FormatInt(c.Int, 10)
	case "decimal":
		return strconv.FormatFloat(c.Float, 'f', -1, 64)
	case "boolean":
		return strconv.FormatBool(c.Bool)
	default:
		return c.String
	}
}

// nestedKey renders a nested expression canonically: focus concepts and each
// group's attributes are sorted, so two expressions that differ only in the order
// they were written produce the same key. Order carries no meaning in SCG.
func nestedKey(expr *scg.Expression) string {
	var b strings.Builder
	b.WriteString(expr.DefinitionStatus)

	focus := make([]string, 0, len(expr.FocusConcepts))
	for _, f := range expr.FocusConcepts {
		focus = append(focus, f.SCTID)
	}
	sort.Strings(focus)
	b.WriteString("|" + strings.Join(focus, "+"))

	groups := make([]string, 0, len(expr.Refinements))
	for _, g := range expr.Refinements {
		pairs := make([]string, 0, len(g.Attributes))
		for _, a := range g.Attributes {
			pairs = append(pairs, a.Name.SCTID+"="+valueKey(a.Value))
		}
		sort.Strings(pairs)
		prefix := "u:"
		if g.Grouped {
			prefix = "g:"
		}
		groups = append(groups, prefix+strings.Join(pairs, ","))
	}
	sort.Strings(groups)
	b.WriteString("|" + strings.Join(groups, ";"))

	return b.String()
}

// inGroupCardinalitySummary renders the in-group cardinalities of some rules, for
// an issue message.
func inGroupCardinalitySummary(rules []AttributeDomain) string {
	parts := make([]string, 0, len(rules))
	for _, r := range rules {
		maxText := "*"
		if r.InGroupCardinality.Max >= 0 {
			maxText = strconv.Itoa(r.InGroupCardinality.Max)
		}
		parts = append(parts, fmt.Sprintf("%d..%s", r.InGroupCardinality.Min, maxText))
	}
	return strings.Join(parts, " or ")
}

// inGroupSpecified reports whether a rule actually states an in-group cardinality.
//
// The zero value of Cardinality is {Min: 0, Max: 0}, which read literally means
// "this attribute may never appear inside a relationship group" — so enforcing it
// as written would make every AttributeDomain built as a Go literal without the
// field forbid its own attribute. LoadFromJSON does not have that problem: an
// absent or empty inGroupCardinality becomes 0..* there. A Model assembled in code
// has no way to tell "unset" from "0..0".
//
// Treated as unset, therefore. The direction of the error matters: this can only
// fail to enforce a constraint, never invent a violation, and the constraint it
// gives up is one nothing needs — "never inside a group" is what Grouped false
// already says, and it is already reported as a grouped_violation. A JSON model
// that spells out "0..0" is a no-op here and should set "grouped": false instead.
func inGroupSpecified(c Cardinality) bool {
	return c.Min != 0 || c.Max != 0
}
