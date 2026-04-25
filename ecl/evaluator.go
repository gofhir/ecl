// Package ecl parses and evaluates SNOMED CT Expression Constraint Language (ECL) expressions
// against a pluggable DataProvider that supplies the underlying terminology.
package ecl

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofhir/ecl/ecl/ast"
)

// Evaluate evaluates an ECL AST against the given DataProvider and returns
// the set of matching SNOMED CT concept IDs.
//
// Phase 3.2 coverage:
//   - Hierarchy operators (8): <, <<, <!, <<!, >, >>, >!, >>!
//   - Set operators: AND, OR, MINUS
//   - Primitives: ConceptRef, Any (wildcard), Nested
//   - MemberOf (^): resolves refset members via DataProvider
//
// Deferred to later phases (returns "not yet implemented" error):
//   - Refinements (Refined)           — Phase 3.3
//   - DotExpression                   — Phase 3.5
//   - Filtered constraints            — Phase 4
//   - HistorySupplement               — Phase 5.1
//   - Top, Bottom, RefsetContainingAny, AltIdentifier (v2.2) — Phase 6
func Evaluate(ctx context.Context, expr ast.Expression, provider DataProvider) (Set, error) {
	if expr == nil {
		return NewSet(), nil
	}

	switch e := expr.(type) {
	// ── Primitives ───────────────────────────────────────────────────────

	case *ast.ConceptRef:
		return provider.ConceptExists(ctx, []string{e.ID})

	case *ast.Any:
		return provider.AllConcepts(ctx)

	case *ast.Nested:
		return Evaluate(ctx, e.Inner, provider)

	// ── Hierarchy operators ──────────────────────────────────────────────

	case *ast.DescendantOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Descendants(ctx, toIDSlice(inner), false)

	case *ast.DescendantOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Descendants(ctx, toIDSlice(inner), true)

	case *ast.ChildOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Children(ctx, toIDSlice(inner), false)

	case *ast.ChildOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Children(ctx, toIDSlice(inner), true)

	case *ast.AncestorOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Ancestors(ctx, toIDSlice(inner), false)

	case *ast.AncestorOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Ancestors(ctx, toIDSlice(inner), true)

	case *ast.ParentOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Parents(ctx, toIDSlice(inner), false)

	case *ast.ParentOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.Parents(ctx, toIDSlice(inner), true)

	// ── Set operators ────────────────────────────────────────────────────

	case *ast.And:
		left, err := Evaluate(ctx, e.Left, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		right, err := Evaluate(ctx, e.Right, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return left.Intersect(right), nil

	case *ast.Or:
		left, err := Evaluate(ctx, e.Left, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		right, err := Evaluate(ctx, e.Right, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return left.Union(right), nil

	case *ast.Minus:
		left, err := Evaluate(ctx, e.Left, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		right, err := Evaluate(ctx, e.Right, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return left.Minus(right), nil

	// ── MemberOf ─────────────────────────────────────────────────────────

	case *ast.MemberOf:
		// The operand evaluates to a set of refset IDs; we then fetch the
		// union of members across those refsets. Field projections
		// (^ [field1,field2]) are ignored in this phase — callers just get
		// the member concept IDs.
		refsetIDs, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.RefsetMembers(ctx, toIDSlice(refsetIDs))

	// ── Refinements (Phase 3.3, 3.4) ─────────────────────────────────────

	case *ast.Refined:
		focus, err := Evaluate(ctx, e.Focus, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T focus: %w", expr, err)
		}
		return applyRefinement(ctx, focus, e.Refinement, provider)

	// ── Dot notation (Phase 3.5) ─────────────────────────────────────────

	case *ast.DotExpression:
		source, err := Evaluate(ctx, e.Source, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T source: %w", expr, err)
		}
		typeIDs, err := Evaluate(ctx, e.Attribute, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T attribute: %w", expr, err)
		}
		return provider.RelationshipTargets(ctx, source, typeIDs)

	// ── Filters (Phase 4) ────────────────────────────────────────────────

	case *ast.Filtered:
		return evaluateFiltered(ctx, e, provider)

	// ── History supplements (Phase 5.1) ──────────────────────────────────

	case *ast.HistorySupplement:
		base, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T operand: %w", expr, err)
		}
		// Empty or unspecified profile defaults to HISTORY-MAX per spec.
		profile := e.Profile
		if profile == "" {
			profile = "HISTORY-MAX"
		}
		historical, err := provider.HistoricalAssociations(ctx, base, profile)
		if err != nil {
			return nil, fmt.Errorf("HistoricalAssociations: %w", err)
		}
		if historical == nil {
			return base, nil
		}
		return base.Union(historical), nil

	// ── v2.2 (Phase 6) ───────────────────────────────────────────────────

	case *ast.Top:
		base, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T operand: %w", expr, err)
		}
		return topOfSet(ctx, base, provider)

	case *ast.Bottom:
		base, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T operand: %w", expr, err)
		}
		return bottomOfSet(ctx, base, provider)

	case *ast.RefsetContainingAny:
		// Pragmatic: treat ^R like MemberOf. The operand resolves to refset
		// IDs; we return the union of members across those refsets.
		refsetIDs, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.RefsetMembers(ctx, toIDSlice(refsetIDs))

	case *ast.AltIdentifier:
		return nil, fmt.Errorf("AltIdentifier not yet implemented (requires alternate identifier resolution): %s#%s", e.Scheme, e.Code)

	default:
		return nil, fmt.Errorf("unsupported AST node type: %T", expr)
	}
}

// toIDSlice converts a Set to a []string of IDs for provider calls.
func toIDSlice(s Set) []string {
	if s == nil {
		return nil
	}
	return s.Slice()
}

// ---------------------------------------------------------------------------
// Refinement evaluation (Phase 3.3 + 3.4)
// ---------------------------------------------------------------------------.

// applyRefinement filters focus by evaluating a refinement.
//
// A refinement is a conjunction of:
//   - Ungrouped attribute clauses (all must match on some relationship)
//   - Grouped attribute clauses (all sub-attrs must match within a single group)
//   - Conjunction sub-refinements (all must match — AND)
//   - Disjunction sub-refinements (at least one must match — OR)
//
// The result is the subset of focus that satisfies every applicable clause.
func applyRefinement(ctx context.Context, focus Set, ref *ast.Refinement, provider DataProvider) (Set, error) {
	if ref == nil || focus == nil || focus.Len() == 0 {
		return focus, nil
	}

	result := focus

	// Ungrouped attributes — all must match (conjunction).
	for _, attr := range ref.Ungrouped {
		filtered, err := filterByAttribute(ctx, result, attr, provider)
		if err != nil {
			return nil, err
		}
		result = filtered
	}

	// Grouped attributes — each group filters the set.
	for _, grp := range ref.Groups {
		filtered, err := filterByAttributeGroup(ctx, result, grp, provider)
		if err != nil {
			return nil, err
		}
		result = filtered
	}

	// Conjunction sub-refinements — all must match.
	for _, sub := range ref.Conjunction {
		filtered, err := applyRefinement(ctx, result, sub, provider)
		if err != nil {
			return nil, err
		}
		result = filtered
	}

	// Disjunction sub-refinements — union of each sub's result on current focus.
	if len(ref.Disjunction) > 0 {
		acc := NewSet()
		for _, sub := range ref.Disjunction {
			subResult, err := applyRefinement(ctx, result, sub, provider)
			if err != nil {
				return nil, err
			}
			acc = acc.Union(subResult)
		}
		result = acc
	}

	return result, nil
}

// filterByAttribute returns the subset of focus whose concepts satisfy a single
// ungrouped attribute clause. Iterates per-concept using PropertiesByGroup.
func filterByAttribute(ctx context.Context, focus Set, attr *ast.Attribute, provider DataProvider) (Set, error) {
	if focus == nil || focus.Len() == 0 {
		return focus, nil
	}
	if attr == nil {
		return focus, nil
	}

	// Resolve attribute type IDs (the attribute name may be an ECL expression).
	typeIDs, err := Evaluate(ctx, attr.Name, provider)
	if err != nil {
		return nil, fmt.Errorf("evaluating attribute name: %w", err)
	}

	// Concrete value comparison: Value is IntegerValue / DecimalValue /
	// StringValue / BooleanValue, not a concept-valued expression. Route to
	// the concrete-value comparator. Numeric operators <, <=, >, >= always
	// target concrete values. For =/!= it depends on the Value type.
	if isConcreteValue(attr.Value) {
		return filterByConcreteValue(ctx, focus, attr, typeIDs, provider)
	}

	switch attr.Op {
	case "=", "!=":
		// concept-valued expression comparison → proceed below
	case "<", "<=", ">", ">=":
		return nil, fmt.Errorf("concrete-value operator %q requires a concrete value (got %T)", attr.Op, attr.Value)
	default:
		return nil, fmt.Errorf("unsupported attribute operator %q", attr.Op)
	}

	// Resolve the value set. A bare wildcard (*) means "any value".
	_, valueIsAny := attr.Value.(*ast.Any)
	var valueSet Set
	if !valueIsAny {
		valueSet, err = Evaluate(ctx, attr.Value, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating attribute value: %w", err)
		}
	}

	// Reverse attribute (R flag): "keep concepts in focus that appear as the
	// TARGET of a relationship where source ∈ valueSet and type ∈ typeIDs".
	// Implementation: compute the set of concepts that ARE such targets (using
	// RelationshipTargets with the value set as sources), then intersect with
	// focus. The "!=" semantics (no such inbound relationship) are the
	// complement within the focus.
	if attr.Reverse {
		var inbound Set
		var err error
		if valueIsAny {
			inbound, err = provider.RelationshipTargets(ctx, nil, typeIDs)
		} else {
			inbound, err = provider.RelationshipTargets(ctx, valueSet, typeIDs)
		}
		if err != nil {
			return nil, fmt.Errorf("reverse attribute lookup: %w", err)
		}
		if attr.Op == "=" {
			return focus.Intersect(inbound), nil
		}
		return focus.Minus(inbound), nil
	}

	// Forward attribute: iterate per-concept via PropertiesByGroup.
	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		count, err := conceptMatchesAttribute(ctx, id, typeIDs, valueSet, valueIsAny, provider)
		if err != nil {
			iterErr = err
			return false
		}
		keep := cardinalitySatisfied(attr.Cardinality, count)
		if attr.Op == "!=" {
			keep = !keep
		}
		if keep {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// conceptMatchesAttribute counts how many relationships of the concept have
// type ∈ typeIDs AND (valueIsAny OR target ∈ valueSet).
func conceptMatchesAttribute(ctx context.Context, conceptID string, typeIDs, valueSet Set, valueIsAny bool, provider DataProvider) (int, error) {
	groups, err := provider.PropertiesByGroup(ctx, conceptID)
	if err != nil {
		return 0, fmt.Errorf("PropertiesByGroup(%s): %w", conceptID, err)
	}
	count := 0
	for _, rels := range groups {
		for _, r := range rels {
			if !typeIDs.Contains(r.TypeID) {
				continue
			}
			if valueIsAny || valueSet.Contains(r.TargetID) {
				count++
			}
		}
	}
	return count, nil
}

// cardinalitySatisfied reports whether count satisfies the given cardinality.
// A nil cardinality is treated as the default [1..*] (at least one, unbounded).
func cardinalitySatisfied(c *ast.Cardinality, count int) bool {
	min, max := 1, -1
	if c != nil {
		min, max = c.Min, c.Max
	}
	if count < min {
		return false
	}
	if max != -1 && count > max {
		return false
	}
	return true
}

// filterByAttributeGroup returns the subset of focus that has at least one
// relationship group in which ALL the group's sub-attributes are satisfied.
func filterByAttributeGroup(ctx context.Context, focus Set, grp *ast.AttributeGroup, provider DataProvider) (Set, error) {
	if focus == nil || focus.Len() == 0 {
		return focus, nil
	}
	if grp == nil || len(grp.Attrs) == 0 {
		return focus, nil
	}
	// Pre-resolve each sub-attribute's type IDs and value set once.
	clauses := make([]attrClause, 0, len(grp.Attrs))
	for _, a := range grp.Attrs {
		typeIDs, err := Evaluate(ctx, a.Name, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating attribute name: %w", err)
		}
		if isConcreteValue(a.Value) {
			c := attrClause{op: a.Op, typeIDs: typeIDs, cardinality: a.Cardinality, isConcrete: true, reverse: a.Reverse}
			switch v := a.Value.(type) {
			case *ast.IntegerValue:
				c.concreteKind = "numeric"
				c.numericVal = float64(v.Value)
			case *ast.DecimalValue:
				c.concreteKind = "numeric"
				c.numericVal = v.Value
			case *ast.StringValue:
				c.concreteKind = "string"
				c.stringVal = v.Value
			case *ast.BooleanValue:
				c.concreteKind = "boolean"
				c.boolVal = v.Value
			}
			clauses = append(clauses, c)
			continue
		}
		switch a.Op {
		case "=", "!=":
		case "<", "<=", ">", ">=":
			return nil, fmt.Errorf("concrete-value operator %q requires a concrete value", a.Op)
		default:
			return nil, fmt.Errorf("unsupported attribute operator %q", a.Op)
		}
		c := attrClause{op: a.Op, typeIDs: typeIDs, cardinality: a.Cardinality, reverse: a.Reverse}
		if _, isAny := a.Value.(*ast.Any); isAny {
			c.valueIsAny = true
		} else {
			c.valueSet, err = Evaluate(ctx, a.Value, provider)
			if err != nil {
				return nil, fmt.Errorf("evaluating attribute value: %w", err)
			}
		}
		clauses = append(clauses, c)
	}

	hasReverse := false
	for _, c := range clauses {
		if c.reverse {
			hasReverse = true
			break
		}
	}

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		anyGroupSatisfies := false
		if !hasReverse {
			// Fast path: all forward clauses — check groups directly.
			groups, err := provider.PropertiesByGroup(ctx, id)
			if err != nil {
				iterErr = fmt.Errorf("PropertiesByGroup(%s): %w", id, err)
				return false
			}
			for _, rels := range groups {
				if groupSatisfiesClauses(rels, clauses) {
					anyGroupSatisfies = true
					break
				}
			}
		} else {
			// Slow path: reverse clauses require checking source concepts.
			matched, err := conceptMatchesGroupWithReverse(ctx, id, clauses, provider)
			if err != nil {
				iterErr = err
				return false
			}
			anyGroupSatisfies = matched
		}
		if anyGroupSatisfies {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// attrClause is a pre-resolved attribute sub-clause for group evaluation.
type attrClause struct {
	op          string
	typeIDs     Set
	valueSet    Set
	valueIsAny  bool
	cardinality *ast.Cardinality
	reverse     bool
	// Concrete value fields (mutually exclusive with valueSet).
	isConcrete   bool
	numericVal   float64
	stringVal    string
	boolVal      bool
	concreteKind string // "numeric", "string", "boolean"
}

// groupSatisfiesClauses reports whether a single relationship group satisfies
// every clause. Each clause counts matching relationships and validates
// against its cardinality constraint.
func groupSatisfiesClauses(rels []Relationship, clauses []attrClause) bool {
	for _, c := range clauses {
		count := 0
		for _, r := range rels {
			if !c.typeIDs.Contains(r.TypeID) {
				continue
			}
			if c.isConcrete {
				if r.ConcreteValue == nil {
					continue
				}
				if matchConcreteValue(r.ConcreteValue, c) {
					count++
				}
			} else {
				hit := c.valueIsAny || c.valueSet.Contains(r.TargetID)
				if hit {
					count++
				}
			}
		}
		if c.op == "!=" && !c.isConcrete {
			totalOfType := 0
			for _, r := range rels {
				if c.typeIDs.Contains(r.TypeID) {
					totalOfType++
				}
			}
			count = totalOfType - count
		}
		if !cardinalitySatisfied(c.cardinality, count) {
			return false
		}
	}
	return true
}

// conceptMatchesGroupWithReverse checks if a concept satisfies a group with
// reverse clauses. For each reverse clause, it finds sources that point to
// this concept with the given type, then checks if any source has a group
// where all clauses (forward and reverse) are satisfied.
func conceptMatchesGroupWithReverse(ctx context.Context, conceptID string, clauses []attrClause, provider DataProvider) (bool, error) {
	for _, c := range clauses {
		if !c.reverse {
			continue
		}
		focusSet := NewSetFromSlice([]string{conceptID})
		sources, err := provider.RelationshipSources(ctx, focusSet, c.typeIDs)
		if err != nil {
			return false, fmt.Errorf("reverse group lookup: %w", err)
		}
		// Filter sources to those in the value set (unless wildcard).
		if !c.valueIsAny && c.valueSet != nil {
			sources = sources.Intersect(c.valueSet)
		}
		if sources.Len() == 0 {
			return false, nil
		}
		found := false
		var iterErr error
		sources.Iter(func(srcID string) bool {
			srcGroups, err := provider.PropertiesByGroup(ctx, srcID)
			if err != nil {
				iterErr = err
				return false
			}
			for _, rels := range srcGroups {
				if groupSatisfiesClausesWithReverse(rels, clauses, conceptID) {
					found = true
					return false
				}
			}
			return true
		})
		if iterErr != nil {
			return false, iterErr
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// groupSatisfiesClausesWithReverse is like groupSatisfiesClauses but handles
// reverse clauses by checking that the relationship targets the given conceptID.
func groupSatisfiesClausesWithReverse(rels []Relationship, clauses []attrClause, reverseTargetID string) bool {
	for _, c := range clauses {
		count := 0
		for _, r := range rels {
			if !c.typeIDs.Contains(r.TypeID) {
				continue
			}
			if c.reverse {
				// Reverse: the relationship must target our focus concept.
				if r.TargetID == reverseTargetID {
					count++
				}
			} else if c.isConcrete {
				if r.ConcreteValue != nil && matchConcreteValue(r.ConcreteValue, c) {
					count++
				}
			} else {
				hit := c.valueIsAny || c.valueSet.Contains(r.TargetID)
				if hit {
					count++
				}
			}
		}
		if c.op == "!=" && !c.isConcrete && !c.reverse {
			totalOfType := 0
			for _, r := range rels {
				if c.typeIDs.Contains(r.TypeID) {
					totalOfType++
				}
			}
			count = totalOfType - count
		}
		if !cardinalitySatisfied(c.cardinality, count) {
			return false
		}
	}
	return true
}

// matchConcreteValue checks if a stored concrete value satisfies the clause comparison.
func matchConcreteValue(cv *ConcreteValue, c attrClause) bool {
	switch c.concreteKind {
	case "numeric":
		if cv.Kind != "integer" && cv.Kind != "decimal" {
			return false
		}
		f, err := strconv.ParseFloat(cv.Value, 64)
		if err != nil {
			return false
		}
		return compareFloat(f, c.op, c.numericVal)
	case "string":
		if cv.Kind != "string" {
			return false
		}
		return compareString(cv.Value, c.op, c.stringVal)
	case "boolean":
		if cv.Kind != "boolean" {
			return false
		}
		stored := cv.Value == "true"
		return compareBool(stored, c.op, c.boolVal)
	}
	return false
}

// ---------------------------------------------------------------------------
// Filtered constraints (Phase 4)
// ---------------------------------------------------------------------------.

// evaluateFiltered evaluates a *ast.Filtered by:
//  1. evaluating the operand to a base set,
//  2. grouping filter clauses into description/concept/member families,
//  3. delegating to provider.MatchDescription / provider.FilterConcepts,
//  4. intersecting the results with the base set.
//
// Member filters are not yet supported (they require refset field projection,
// which is deferred to a later phase).
func evaluateFiltered(ctx context.Context, e *ast.Filtered, provider DataProvider) (Set, error) {
	base, err := Evaluate(ctx, e.Operand, provider)
	if err != nil {
		return nil, fmt.Errorf("evaluating Filtered operand: %w", err)
	}
	descFilters, conceptFilters, memberFilters := categorizeFilters(e.Filters)

	result := base

	if len(memberFilters) > 0 {
		for _, mf := range memberFilters {
			field, ok := mf.(*ast.MemberFieldFilter)
			if !ok {
				continue
			}
			var valueSet Set
			if field.Value != nil {
				valueSet, err = Evaluate(ctx, field.Value, provider)
				if err != nil {
					return nil, fmt.Errorf("evaluating member filter value: %w", err)
				}
			}
			opts := MemberFilterOpts{
				FieldName: field.FieldName,
				Op:        field.Op,
				ValueSet:  valueSet,
			}
			// Get the refset IDs from the base operand if it's a MemberOf.
			var refsetIDs []string
			if mo, ok := e.Operand.(*ast.MemberOf); ok {
				refsetIDSet, innerErr := Evaluate(ctx, mo.Operand, provider)
				if innerErr != nil {
					return nil, fmt.Errorf("evaluating member filter refset: %w", innerErr)
				}
				refsetIDs = toIDSlice(refsetIDSet)
			} else {
				refsetIDs = toIDSlice(base)
			}
			filtered, innerErr := provider.RefsetMembersFiltered(ctx, refsetIDs, opts)
			if innerErr != nil {
				return nil, fmt.Errorf("RefsetMembersFiltered: %w", innerErr)
			}
			result = result.Intersect(filtered)
		}
	}

	// Description filters — build a single DescriptionFilterOpts from all
	// clauses, then call MatchDescription once. The result is a set of concept
	// IDs whose descriptions match; intersect with the current base (or subtract
	// when negated).
	if len(descFilters) > 0 {
		opts, negate, err := buildDescriptionFilterOpts(ctx, descFilters, provider)
		if err != nil {
			return nil, err
		}
		matches, err := provider.MatchDescription(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("MatchDescription: %w", err)
		}
		if negate {
			result = result.Minus(matches)
		} else if result == nil {
			result = matches
		} else {
			result = result.Intersect(matches)
		}
	}

	// Dialect filters — separate from description filters because they use
	// a dedicated provider method.
	for _, f := range descFilters {
		df, ok := f.(*ast.DialectFilter)
		if !ok {
			continue
		}
		dOpts, err := buildDialectFilterOpts(ctx, df, provider)
		if err != nil {
			return nil, err
		}
		matches, err := provider.MatchDialect(ctx, result, dOpts)
		if err != nil {
			return nil, fmt.Errorf("MatchDialect: %w", err)
		}
		if df.Op == "!=" {
			result = result.Minus(matches)
		} else {
			result = result.Intersect(matches)
		}
	}

	// Concept filters — build ConceptFilterOpts and delegate (or subtract when
	// negated).
	if len(conceptFilters) > 0 {
		opts, negate, err := buildConceptFilterOpts(ctx, conceptFilters, provider)
		if err != nil {
			return nil, err
		}
		filtered, err := provider.FilterConcepts(ctx, result, opts)
		if err != nil {
			return nil, fmt.Errorf("FilterConcepts: %w", err)
		}
		if negate {
			result = result.Minus(filtered)
		} else {
			result = filtered
		}
	}

	return result, nil
}

// categorizeFilters splits a flat list of filters into description, concept,
// and member families. Ambiguous filters (active/module/effectiveTime) default
// to the concept family — this is the most common usage in practice.
func categorizeFilters(filters []ast.Filter) (desc, concept, member []ast.Filter) {
	for _, f := range filters {
		switch f.(type) {
		case *ast.TermFilter, *ast.TypeFilter, *ast.LanguageFilter, *ast.DialectFilter:
			desc = append(desc, f)
		case *ast.DefinitionStatusFilter:
			concept = append(concept, f)
		case *ast.MemberFieldFilter:
			member = append(member, f)
		case *ast.ActiveFilter, *ast.ModuleFilter, *ast.EffectiveTimeFilter:
			// Ambiguous — default to concept family. Description-level
			// active/module/effectiveTime is rare in practice.
			concept = append(concept, f)
		}
	}
	return desc, concept, member
}

// buildDescriptionFilterOpts accumulates description filter clauses into a
// single DescriptionFilterOpts. Multiple clauses of the same kind are
// simplified by taking the last one (rarely seen in practice).
func buildDescriptionFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (DescriptionFilterOpts, bool, error) {
	var opts DescriptionFilterOpts
	var negate bool
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.TermFilter:
			if x.Op == "!=" {
				negate = true
			}
			opts.Term = x.Term
		case *ast.TypeFilter:
			if x.Op == "!=" {
				negate = true
			}
			// Resolve the first type SCTID. Multi-type sets simplify to the
			// first concept.
			if len(x.Types) > 0 {
				ids, err := Evaluate(ctx, x.Types[0], provider)
				if err != nil {
					return opts, false, fmt.Errorf("evaluating type filter: %w", err)
				}
				if ids != nil && ids.Len() > 0 {
					opts.TypeID = ids.Slice()[0]
				} else if ref, ok := x.Types[0].(*ast.ConceptRef); ok {
					// Fallback to the literal ID even if provider didn't find
					// it (tokens like FSN/SYN map to well-known SCTIDs that
					// test providers may not enumerate in ConceptExists).
					opts.TypeID = ref.ID
				}
			}
		case *ast.LanguageFilter:
			if x.Op == "!=" {
				negate = true
			}
			if len(x.Languages) > 0 {
				opts.Language = x.Languages[0]
			}
		case *ast.DialectFilter:
			// Handled separately in evaluateFiltered.
			continue
		}
	}
	return opts, negate, nil
}

// buildConceptFilterOpts accumulates concept filter clauses into ConceptFilterOpts.
func buildConceptFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (ConceptFilterOpts, bool, error) {
	var opts ConceptFilterOpts
	var negate bool
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.ActiveFilter:
			b := x.Value
			opts.Active = &b
		case *ast.DefinitionStatusFilter:
			if x.Op == "!=" {
				negate = true
			}
			if x.Value != nil {
				ids, err := Evaluate(ctx, x.Value, provider)
				if err != nil {
					return opts, false, fmt.Errorf("evaluating definitionStatus filter: %w", err)
				}
				if ids != nil && ids.Len() > 0 {
					opts.DefinitionStatusID = ids.Slice()[0]
				} else if ref, ok := x.Value.(*ast.ConceptRef); ok {
					opts.DefinitionStatusID = ref.ID
				}
			}
		case *ast.ModuleFilter:
			if x.Op == "!=" {
				negate = true
			}
			if x.Module != nil {
				ids, err := Evaluate(ctx, x.Module, provider)
				if err != nil {
					return opts, false, fmt.Errorf("evaluating module filter: %w", err)
				}
				if ids != nil && ids.Len() > 0 {
					opts.ModuleID = ids.Slice()[0]
				} else if ref, ok := x.Module.(*ast.ConceptRef); ok {
					opts.ModuleID = ref.ID
				}
			}
		case *ast.EffectiveTimeFilter:
			opts.EffectiveTime = x.Value
			opts.EffectiveTimeOp = x.Op
		}
	}
	return opts, negate, nil
}

// buildDialectFilterOpts converts an ast.DialectFilter into DialectFilterOpts
// by evaluating each dialect entry's dialect and acceptability expressions.
func buildDialectFilterOpts(ctx context.Context, df *ast.DialectFilter, provider DataProvider) (DialectFilterOpts, error) {
	opts := DialectFilterOpts{Negate: df.Op == "!="}
	for _, entry := range df.Dialects {
		var dialectID string
		if entry.Dialect != nil {
			ids, err := Evaluate(ctx, entry.Dialect, provider)
			if err != nil {
				return opts, fmt.Errorf("evaluating dialect: %w", err)
			}
			if ids != nil && ids.Len() > 0 {
				dialectID = ids.Slice()[0]
			} else if ref, ok := entry.Dialect.(*ast.ConceptRef); ok {
				dialectID = ref.ID
			}
		}
		var acceptID string
		if entry.Acceptability != nil {
			ids, err := Evaluate(ctx, entry.Acceptability, provider)
			if err != nil {
				return opts, fmt.Errorf("evaluating acceptability: %w", err)
			}
			if ids != nil && ids.Len() > 0 {
				acceptID = ids.Slice()[0]
			} else if ref, ok := entry.Acceptability.(*ast.ConceptRef); ok {
				acceptID = ref.ID
			}
		}
		opts.Dialects = append(opts.Dialects, DialectEntryOpts{
			DialectID:       dialectID,
			AcceptabilityID: acceptID,
		})
	}
	return opts, nil
}

// ---------------------------------------------------------------------------
// Concrete value comparisons (Phase 5.2)
// ---------------------------------------------------------------------------.

// isConcreteValue reports whether an Attribute.Value is a concrete literal
// (integer, decimal, string, or boolean) rather than a concept-valued
// expression.
func isConcreteValue(e ast.Expression) bool {
	switch e.(type) {
	case *ast.IntegerValue, *ast.DecimalValue, *ast.StringValue, *ast.BooleanValue:
		return true
	}
	return false
}

// filterByConcreteValue filters focus by a concrete-value attribute clause.
// It iterates per-concept, calling provider.ConcreteValues for each attribute
// type ID. If any stored concrete value satisfies the comparison, the concept
// is kept (or excluded, for the "!=" operator).
//
// Only numeric comparisons (=, !=, <, <=, >, >=) are implemented for
// IntegerValue and DecimalValue. String and boolean comparisons currently
// return "not yet implemented" so callers get a clear signal.
func filterByConcreteValue(ctx context.Context, focus Set, attr *ast.Attribute, typeIDs Set, provider DataProvider) (Set, error) {
	if focus == nil || focus.Len() == 0 {
		return focus, nil
	}
	// Extract the literal value. Numeric operators require a numeric literal.
	var (
		haveNumeric bool
		numeric     float64
		haveString  bool
		strVal      string
		haveBool    bool
		boolVal     bool
	)
	switch v := attr.Value.(type) {
	case *ast.IntegerValue:
		haveNumeric = true
		numeric = float64(v.Value)
	case *ast.DecimalValue:
		haveNumeric = true
		numeric = v.Value
	case *ast.StringValue:
		haveString = true
		strVal = v.Value
	case *ast.BooleanValue:
		haveBool = true
		boolVal = v.Value
	default:
		return nil, fmt.Errorf("filterByConcreteValue: unexpected value type %T", attr.Value)
	}

	switch attr.Op {
	case "=", "!=":
		// supported for numeric; string/boolean numeric-op paths share =/!=.
	case "<", "<=", ">", ">=":
		if !haveNumeric {
			return nil, fmt.Errorf("operator %q requires a numeric concrete value (got %T)", attr.Op, attr.Value)
		}
	default:
		return nil, fmt.Errorf("unsupported concrete-value operator %q", attr.Op)
	}

	typeIDList := typeIDs.Slice()

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		matched := false
		for _, typeID := range typeIDList {
			values, err := provider.ConcreteValues(ctx, id, typeID)
			if err != nil {
				iterErr = fmt.Errorf("ConcreteValues(%s, %s): %w", id, typeID, err)
				return false
			}
			for _, cv := range values {
				switch {
				case haveNumeric && (cv.Kind == "integer" || cv.Kind == "decimal"):
					f, parseErr := strconv.ParseFloat(cv.Value, 64)
					if parseErr != nil {
						continue
					}
					if compareFloat(f, attr.Op, numeric) {
						matched = true
					}
				case haveString && cv.Kind == "string":
					if compareString(cv.Value, attr.Op, strVal) {
						matched = true
					}
				case haveBool && cv.Kind == "boolean":
					stored := cv.Value == "true"
					if compareBool(stored, attr.Op, boolVal) {
						matched = true
					}
				}
				if matched {
					break
				}
			}
			if matched {
				break
			}
		}
		// compareFloat already applied the operator directly, so "matched"
		// means at least one stored value satisfied the comparison. No extra
		// inversion for "!=".
		if matched {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// compareFloat applies the given operator to a (stored) and b (expected).
func compareFloat(a float64, op string, b float64) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	case ">=":
		return a >= b
	}
	return false
}

// compareString applies the given operator to string values a (stored) and b (expected).
func compareString(a, op, b string) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	}
	return false
}

// compareBool applies the given operator to boolean values a (stored) and b (expected).
func compareBool(a bool, op string, b bool) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	}
	return false
}

// ---------------------------------------------------------------------------
// Top / Bottom of set (Phase 6.1, v2.2)
// ---------------------------------------------------------------------------.

// topOfSet returns the concepts in baseSet that have no parent in baseSet
// (i.e. the roots of the sub-graph induced by baseSet).
func topOfSet(ctx context.Context, baseSet Set, provider DataProvider) (Set, error) {
	if baseSet == nil || baseSet.Len() == 0 {
		return baseSet, nil
	}
	out := newMapSet()
	var iterErr error
	baseSet.Iter(func(id string) bool {
		parents, err := provider.Parents(ctx, []string{id}, false)
		if err != nil {
			iterErr = fmt.Errorf("Parents(%s): %w", id, err)
			return false
		}
		if parents == nil || parents.Intersect(baseSet).Len() == 0 {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// bottomOfSet returns the concepts in baseSet that have no child in baseSet
// (i.e. the leaves of the sub-graph induced by baseSet).
func bottomOfSet(ctx context.Context, baseSet Set, provider DataProvider) (Set, error) {
	if baseSet == nil || baseSet.Len() == 0 {
		return baseSet, nil
	}
	out := newMapSet()
	var iterErr error
	baseSet.Iter(func(id string) bool {
		children, err := provider.Children(ctx, []string{id}, false)
		if err != nil {
			iterErr = fmt.Errorf("Children(%s): %w", id, err)
			return false
		}
		if children == nil || children.Intersect(baseSet).Len() == 0 {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}
