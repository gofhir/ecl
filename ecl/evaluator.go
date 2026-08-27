// Package ecl parses and evaluates SNOMED CT Expression Constraint Language (ECL) expressions
// against a pluggable DataProvider that supplies the underlying terminology.
package ecl

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/ecl/ast"
)

// ErrUnsupportedFeature marks an ECL construct the evaluator recognizes but
// cannot evaluate correctly yet. Callers can classify it and answer 422 or 501
// instead of serving a wrong result set:
//
//	if errors.Is(err, ecl.ErrUnsupportedFeature) { /* not implemented */ }
//
// It is returned rather than a silently incorrect set whenever the semantics
// cannot be expressed through the current DataProvider contract.
var ErrUnsupportedFeature = errors.New("unsupported ECL feature")

// ErrProvider wraps a failure that came from the DataProvider rather than from
// the expression, so a caller can answer 503 instead of 400:
//
//	if errors.Is(err, ecl.ErrProvider) { /* the backend is unhealthy */ }
var ErrProvider = errors.New("data provider error")

// Evaluate evaluates an ECL AST against the given DataProvider and returns
// the set of matching SNOMED CT concept IDs.
//
// Full ECL v2.2 coverage:
//   - Hierarchy operators (8): <, <<, <!, <<!, >, >>, >!, >>!
//   - Set operators: AND, OR, MINUS
//   - Primitives: ConceptRef, Any (wildcard), Nested
//   - MemberOf (^): resolves refset members via DataProvider
//   - Refinements (ungrouped/grouped) with cardinality [min..max]
//   - Reverse attribute (R flag) including wildcard and concrete values
//   - DotExpression (attribute navigation)
//   - Filter constraints: term, type, language, dialect, active, module,
//     definitionStatus, effectiveTime. Concept filters ({{ C ... }}) support
//     the negated (!=) operator per clause; negated DESCRIPTION filters
//     ({{ D ... }}) return ErrUnsupportedFeature, because their semantics is a
//     per-description-row negation that the DataProvider contract cannot yet
//     express — see ErrUnsupportedFeature.
//   - Member field filters
//   - Concrete value comparisons: integer, decimal, string, boolean
//   - HistorySupplement with MIN/MOD/MAX profiles
//   - Top (!!>), Bottom (!!<), RefsetContainingAny (^R)
//   - AltIdentifier (scheme#code) via DataProvider.ResolveIdentifier
func Evaluate(ctx context.Context, expr ast.Expression, provider DataProvider) (Set, error) {
	if expr == nil {
		return NewSet(), nil
	}
	// Evaluating a broad expression can issue thousands of provider calls, so
	// honor cancellation. Checked on entry, which covers every recursive step.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	set, err := evaluateNode(ctx, expr, provider)
	if err != nil {
		return nil, err
	}
	// Never hand a nil Set to the caller: the contract requires providers to
	// return non-nil, but a wrong provider must yield an empty result rather
	// than a nil-pointer panic in the caller's Slice().
	return nonNil(set), nil
}

// nonNil normalizes a Set returned by a DataProvider. The DataProvider contract
// requires implementations to return a non-nil Set; this keeps a provider that
// breaks the rule from panicking the evaluator, since returning (nil, nil) for
// "nothing found" is idiomatic Go and an easy mistake to make.
func nonNil(s Set) Set {
	if s == nil {
		return NewSet()
	}
	return s
}

// evaluateNode is the type switch behind Evaluate.
func evaluateNode(ctx context.Context, expr ast.Expression, provider DataProvider) (Set, error) {
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
		// union of members across those refsets.
		//
		// Field projections (^ [field1,field2]) cannot be expressed through
		// the Set return type because Set carries only concept IDs, not
		// per-field values. Reject explicitly rather than silently dropping
		// the projection. Use a MemberFieldFilter inside a {{ M ... }}
		// constraint when you need per-field filtering.
		if len(e.Fields) > 0 {
			return nil, fmt.Errorf("%w: MemberOf field projection ^[%s]; Set carries concept IDs only, so use a {{ M ... }} member filter instead",
				ErrUnsupportedFeature, strings.Join(e.Fields, ","))
		}
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
			return nil, fmt.Errorf("%w: HistoricalAssociations: %w", ErrProvider, err)
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
		// ^R <operand> returns the set of refsets that contain any concept
		// in the operand as a member. This is the inverse direction of ^
		// (which maps refset → members).
		operandIDs, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return provider.RefsetsContainingMembers(ctx, toIDSlice(operandIDs))

	case *ast.AltIdentifier:
		return provider.ResolveIdentifier(ctx, e.Scheme, e.Code)

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

	// The grammar admits a conjunction set OR a disjunction set at a level, never
	// both, and the disjunction below is composed on that basis. A node holding
	// both means the parenthesised scope was lost while parsing.
	if len(ref.Conjunction) > 0 && len(ref.Disjunction) > 0 {
		return nil, fmt.Errorf("refinement has both a conjunction and a disjunction in one node: the parenthesised scope was lost while parsing")
	}

	result := focus

	// A parenthesised sub-refinement is part of the first operand, so it filters
	// before the conjunction and disjunction stages below.
	if ref.Nested != nil {
		filtered, err := applyRefinement(ctx, result, ref.Nested, provider)
		if err != nil {
			return nil, err
		}
		result = filtered
	}

	// Ungrouped attribute clauses. AttrSet is the boolean tree; it replaces the
	// Ungrouped loop only, and the group/conjunction/disjunction stages below
	// still run. Returning early here would silently drop the sibling groups of
	// an expression like `: a = x , { b = y }`, where the grammar puts the group
	// in Conjunction because subattributeset admits no groups.
	if ref.AttrSet != nil {
		filtered, err := applyAttrSet(ctx, result, ref.AttrSet, provider)
		if err != nil {
			return nil, err
		}
		result = filtered
	} else {
		for _, attr := range ref.Ungrouped { //nolint:staticcheck // deprecated path, for hand-built ASTs
			filtered, err := filterByAttribute(ctx, result, attr, provider)
			if err != nil {
				return nil, err
			}
			result = filtered
		}
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

	// Disjunction sub-refinements. Each disjunct is an ALTERNATIVE to the first
	// sub-refinement, so every branch is evaluated against the incoming focus
	// and the results are unioned — including the first sub-refinement's own
	// result, which the stages above already computed into `result`.
	//
	// Evaluating the disjuncts against `result` instead of `focus` is what made
	// `A OR B` behave as `focus ∩ A ∩ B`, i.e. usually the empty set.
	if len(ref.Disjunction) > 0 {
		// Start from `result` only when a first operand actually filtered it.
		// Seeding with the unfiltered focus made a hand-built
		// Refinement{Disjunction: ...} a no-op that returned everything.
		acc := NewSet()
		if ref.AttrSet != nil || len(ref.Ungrouped) > 0 || len(ref.Groups) > 0 || ref.Nested != nil || len(ref.Conjunction) > 0 { //nolint:staticcheck // deprecated field read for hand-built ASTs
			acc = result
		}
		for _, sub := range ref.Disjunction {
			subResult, err := applyRefinement(ctx, focus, sub, provider)
			if err != nil {
				return nil, err
			}
			acc = acc.Union(subResult)
		}
		result = acc
	}

	return result, nil
}

// applyAttrSet filters focus by a boolean tree of attribute clauses.
//
// The invariant that makes OR work: every branch is evaluated against the SAME
// incoming focus. Chaining them (feeding each branch the previous branch's
// result) turns a union into an intersection.
func applyAttrSet(ctx context.Context, focus Set, set *ast.AttributeSet, provider DataProvider) (Set, error) {
	if set == nil || focus == nil || focus.Len() == 0 {
		return focus, nil
	}
	if set.Attr != nil {
		return filterByAttribute(ctx, focus, set.Attr, provider)
	}

	if set.Op == ast.AttrSetOr {
		acc := NewSet()
		for _, item := range set.Items {
			sub, err := applyAttrSet(ctx, focus, item, provider) // focus, never acc
			if err != nil {
				return nil, err
			}
			acc = acc.Union(sub)
		}
		return acc, nil
	}

	result := focus
	for _, item := range set.Items {
		sub, err := applyAttrSet(ctx, result, item, provider)
		if err != nil {
			return nil, err
		}
		result = sub
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
	// TARGET of a relationship whose source is in valueSet and type is in
	// typeIDs". Implemented by asking the provider for those targets and
	// intersecting with the focus.
	//
	// Two forms cannot be answered that way, because RelationshipTargets returns
	// a Set and so loses how MANY inbound relationships each concept has, and of
	// which types:
	//
	//   - a cardinality, which needs the count;
	//   - "!=", which per the spec selects concepts having an inbound
	//     relationship OF THAT TYPE from some other source, and so needs the
	//     per-type total.
	//
	// Both used to be answered anyway, and wrongly: `[0..0] R a = x` returned
	// exactly the concepts that DO have the relationship, and `R a != x` kept the
	// "does not have it at all" reading that the forward path abandoned. Report
	// them instead, as the group-level analog already does.
	if attr.Reverse {
		// A provider that supplies the inbound relationships can answer the forms
		// RelationshipTargets cannot, because it preserves the multiplicity and
		// the types that a cardinality and "!=" need.
		if attr.Cardinality != nil || attr.Op == "!=" {
			inboundProvider, ok := provider.(InboundRelationshipsProvider)
			if !ok {
				if attr.Cardinality != nil {
					return nil, fmt.Errorf("%w: cardinality [%d..%s] on a reverse attribute requires a provider implementing ecl.InboundRelationshipsProvider; RelationshipTargets returns a Set and so loses the inbound count",
						ErrUnsupportedFeature, attr.Cardinality.Min, cardinalityMaxText(attr.Cardinality))
				}
				return nil, fmt.Errorf("%w: %q on a reverse attribute requires a provider implementing ecl.InboundRelationshipsProvider; the per-type inbound total is needed and RelationshipTargets does not return it",
					ErrUnsupportedFeature, attr.Op)
			}
			return filterByReverseCounted(ctx, focus, attr, typeIDs, valueSet, valueIsAny, inboundProvider)
		}

		var inbound Set
		var err error
		if valueIsAny {
			inbound, err = provider.RelationshipTargets(ctx, nil, typeIDs)
		} else {
			inbound, err = provider.RelationshipTargets(ctx, valueSet, typeIDs)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: reverse attribute lookup: %w", ErrProvider, err)
		}
		return focus.Intersect(nonNil(inbound)), nil
	}

	// Forward attribute: iterate per-concept via PropertiesByGroup.
	//
	// The clause is evaluated over ALL of the concept's relationships at once
	// (groups flattened), which is what "ungrouped" means: the attribute may sit
	// in any group. clauseSatisfied then applies the same "!=" rule the grouped
	// path uses, so the two paths agree.
	clause := attrClause{
		op:          attr.Op,
		typeIDs:     typeIDs,
		valueSet:    valueSet,
		valueIsAny:  valueIsAny,
		cardinality: attr.Cardinality,
	}
	properties, err := propertiesFor(ctx, provider, focus)
	if err != nil {
		return nil, err
	}

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		if err := ctx.Err(); err != nil {
			iterErr = err
			return false
		}
		if clauseSatisfied(flattenGroups(properties[id]), clause) {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// filterByReverseCounted evaluates a reverse attribute clause that needs the
// inbound COUNT — a cardinality, or "!=" — using InboundRelationshipsProvider.
//
// The counting rules mirror the forward path exactly, which is the point: the two
// directions of the same clause must agree.
//
//	matching    inbound relationships of the type whose source is in the value set
//	totalOfType inbound relationships of the type, whatever the source
//
// and "!=" counts `totalOfType - matching`, so it selects concepts pointed at from
// somewhere OTHER than the value set — not concepts nothing points at.
func filterByReverseCounted(
	ctx context.Context,
	focus Set,
	attr *ast.Attribute,
	typeIDs, valueSet Set,
	valueIsAny bool,
	provider InboundRelationshipsProvider,
) (Set, error) {
	inbound, err := provider.InboundRelationships(ctx, focus, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: InboundRelationships: %w", ErrProvider, err)
	}

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		if err := ctx.Err(); err != nil {
			iterErr = err
			return false
		}

		// Count SOURCE CONCEPTS, not inbound relationships. The specification
		// says a cardinality on a reversed refinement "constrains the number of
		// source concepts (matching the given criteria) for which each
		// destination concept may be relevant attribute value", and illustrates
		// it with `[3..3] R 127489000` meaning "an active ingredient of exactly
		// three products". Counting relationships instead diverges whenever one
		// source points at the same target twice with the same attribute type —
		// two relationships, still one product.
		matching, totalOfType := map[string]struct{}{}, map[string]struct{}{}
		for _, rel := range inbound[id] {
			if !typeIDs.Contains(rel.TypeID) {
				continue
			}
			totalOfType[rel.SourceID] = struct{}{}
			if valueIsAny || valueSet.Contains(rel.SourceID) {
				matching[rel.SourceID] = struct{}{}
			}
		}

		count := len(matching)
		if attr.Op == "!=" {
			// Sources of this type that are NOT in the value set.
			count = len(totalOfType) - len(matching)
		}
		if cardinalitySatisfied(attr.Cardinality, count) {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// concreteValuesFor fetches the concrete values of every (concept, type) pair the
// clause needs.
//
// A provider implementing BatchConcreteValuesProvider answers in one call. Without
// it this falls back to ConcreteValues once per pair, which the signature forces:
// a wildcard attribute type over N concepts and T types costs N×T round trips.
func concreteValuesFor(ctx context.Context, provider DataProvider, focus Set, typeIDs []string) (map[string]map[string][]ConcreteValue, error) {
	ids := toIDSlice(focus)
	if len(ids) == 0 || len(typeIDs) == 0 {
		return map[string]map[string][]ConcreteValue{}, nil
	}

	if batch, ok := provider.(BatchConcreteValuesProvider); ok {
		out, err := batch.ConcreteValuesBatch(ctx, ids, typeIDs)
		if err != nil {
			return nil, fmt.Errorf("%w: ConcreteValuesBatch: %w", ErrProvider, err)
		}
		if out == nil {
			out = map[string]map[string][]ConcreteValue{}
		}
		return out, nil
	}

	out := make(map[string]map[string][]ConcreteValue, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, typeID := range typeIDs {
			values, err := provider.ConcreteValues(ctx, id, typeID)
			if err != nil {
				return nil, fmt.Errorf("%w: ConcreteValues(%s, %s): %w", ErrProvider, id, typeID, err)
			}
			if len(values) == 0 {
				continue
			}
			if out[id] == nil {
				out[id] = make(map[string][]ConcreteValue)
			}
			out[id][typeID] = values
		}
	}
	return out, nil
}

// propertiesFor fetches the relationships of every concept in the focus set.
//
// A provider that implements BatchPropertiesProvider answers in one call. Without
// it this falls back to PropertiesByGroup once per concept — the N+1 the interface
// forces, which against SNOMED International means ~110,000 round trips for
// `< 404684003 : 363698007 = *`. The optional capability is how that is fixed
// without widening DataProvider and breaking every implementation.
func propertiesFor(ctx context.Context, provider DataProvider, focus Set) (map[string]map[int][]Relationship, error) {
	ids := toIDSlice(focus)
	if len(ids) == 0 {
		return map[string]map[int][]Relationship{}, nil
	}

	if batch, ok := provider.(BatchPropertiesProvider); ok {
		out, err := batch.PropertiesByGroupBatch(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("%w: PropertiesByGroupBatch: %w", ErrProvider, err)
		}
		if out == nil {
			out = map[string]map[int][]Relationship{}
		}
		return out, nil
	}

	out := make(map[string]map[int][]Relationship, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		groups, err := provider.PropertiesByGroup(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("%w: PropertiesByGroup(%s): %w", ErrProvider, id, err)
		}
		out[id] = groups
	}
	return out, nil
}

// flattenGroups collapses a concept's relationship groups into one slice.
//
// Ungrouped attribute clauses match a relationship in ANY group, so they are
// evaluated against this flat view.
func flattenGroups(groups map[int][]Relationship) []Relationship {
	var out []Relationship
	for _, rels := range groups {
		out = append(out, rels...)
	}
	return out
}

// cardinalitySatisfied reports whether count satisfies the given cardinality.
// A nil cardinality is treated as the default [1..*] (at least one, unbounded).
func cardinalitySatisfied(c *ast.Cardinality, count int) bool {
	minVal, maxVal := 1, -1
	if c != nil {
		minVal, maxVal = c.Min, c.Max
	}
	if count < minVal {
		return false
	}
	if maxVal != -1 && count > maxVal {
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
	// A hand-built AST may carry a nil group; ecl/ast is public, so this is
	// reachable without the parser.
	if grp == nil {
		return focus, nil
	}
	set := grp.AttrSet
	if set == nil {
		// Hand-built AST using the deprecated field: rebuild the tree as a
		// conjunction so both paths share one implementation.
		set = attrSetFromSlice(grp.Attrs) //nolint:staticcheck // deprecated field support
	}
	if set == nil {
		return focus, nil
	}

	// Pre-resolve every leaf's type IDs and value set once, keeping the boolean
	// shape so `{ a = x OR b = y }` is a disjunction within the group instead of
	// a conjunction.
	tree, err := resolveClauseSet(ctx, set, provider)
	if err != nil {
		return nil, err
	}
	if err := rejectReverseInGroup(tree.leaves()); err != nil {
		return nil, err
	}

	// Fetched once for the whole focus set rather than once per concept; see
	// propertiesFor.
	properties, err := propertiesFor(ctx, provider, focus)
	if err != nil {
		return nil, err
	}

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		if err := ctx.Err(); err != nil {
			iterErr = err
			return false
		}
		// matchingGroups is how many relationship groups satisfy the clause
		// tree. Counting is what makes group cardinality work at all: the code
		// used to stop at the first match, so [0..0], [1..1] and [2..*] were
		// indistinguishable from "at least one" and [0..0] returned exactly the
		// inverse of the requested set.
		matchingGroups := 0
		for gnum, rels := range properties[id] {
			// Group 0 is "ungrouped" per the PropertiesByGroup contract, so it is
			// not a relationship group and must not be counted as one. Ungrouped
			// attributes are matched by filterByAttribute instead.
			if gnum == 0 {
				continue
			}
			if groupSatisfiesSet(rels, tree) {
				matchingGroups++
			}
		}
		// A nil cardinality is the default [1..*], same as everywhere else.
		if cardinalitySatisfied(grp.Cardinality, matchingGroups) {
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
	concreteKind string // concreteKindNumeric, concreteKindString, concreteKindBoolean
}

// Internal classification of concrete value clauses. These are coarser than
// ConcreteValue.Kind (which uses "integer" vs "decimal") because <, <=, >, >=
// treat both numeric kinds the same.
const (
	concreteKindNumeric = "numeric"
	concreteKindString  = "string"
	concreteKindBoolean = "boolean"
)

// rejectReverseInGroup refuses a reverse attribute inside an attribute group.
//
// Braces assert that the clauses co-occur in ONE relationship group OF THE FOCUS
// CONCEPT — that is the only thing grouping adds, and it is what distinguishes a
// heart infarct from a concept carrying heart-site in one group and
// infarct-morphology in another. A reverse clause's relationship does not belong
// to the focus at all: it belongs to the source. So there is nothing of the
// focus's to group, and only two readings remain, both bad:
//
//   - `{ R a = x }` alone: the braces add nothing. Measured before this check,
//     the result was identical to the ungrouped `R a = x`.
//   - `{ R a = x, b = y }`: the group has to belong to someone, and it can only be
//     the source — so `b = y` silently constrained the SOURCE rather than the
//     focus. Measured: `* : { R 363698007 = 22298006, 116676008 = 55641003 }`
//     returned 74281007, which has no relationships of its own and therefore
//     cannot satisfy the second clause. What was checked was 22298006's morphology.
//
// So the construct is either redundant or misleading, never useful. The
// specification neither permits nor forbids it — §6.2 describes reverse attributes
// and attribute groups separately, §6.3 defines reverse cardinality only for an
// ungrouped refinement, and the official example set
// (IHTSDO/snomed-expression-constraint-language) never places R inside braces —
// and Ontoserver rejects it outright with "Cannot reverse an attribute inside a
// group". This library now agrees.
func rejectReverseInGroup(clauses []attrClause) error {
	for _, c := range clauses {
		if !c.reverse {
			continue
		}
		return fmt.Errorf("%w: a reverse attribute cannot appear inside an attribute group — grouping asserts that the clauses share a relationship group of the FOCUS concept, and a reverse relationship belongs to the source; write it outside the braces",
			ErrUnsupportedFeature)
	}
	return nil
}

// cardinalityMaxText renders a cardinality's upper bound, with "*" for unbounded.
func cardinalityMaxText(c *ast.Cardinality) string {
	if c == nil || c.Max < 0 {
		return "*"
	}
	return strconv.Itoa(c.Max)
}

// clauseSet is the resolved counterpart of ast.AttributeSet: a boolean tree
// whose leaves carry pre-resolved type IDs and value sets.
type clauseSet struct {
	op    ast.AttrSetOp
	leaf  *attrClause
	items []*clauseSet
}

// leaves flattens the tree into the clause slice the reverse path still uses.
func (cs *clauseSet) leaves() []attrClause {
	if cs == nil {
		return nil
	}
	if cs.leaf != nil {
		return []attrClause{*cs.leaf}
	}
	var out []attrClause
	for _, item := range cs.items {
		out = append(out, item.leaves()...)
	}
	return out
}

// attrSetFromSlice rebuilds a conjunction tree from the deprecated flat field,
// so an AST built by hand against v1.1 still evaluates.
func attrSetFromSlice(attrs []*ast.Attribute) *ast.AttributeSet {
	switch len(attrs) {
	case 0:
		return nil
	case 1:
		return &ast.AttributeSet{Attr: attrs[0]}
	}
	items := make([]*ast.AttributeSet, 0, len(attrs))
	for _, a := range attrs {
		items = append(items, &ast.AttributeSet{Attr: a})
	}
	return &ast.AttributeSet{Op: ast.AttrSetAnd, Items: items}
}

// resolveClauseSet resolves every leaf of an attribute tree once, up front.
func resolveClauseSet(ctx context.Context, set *ast.AttributeSet, provider DataProvider) (*clauseSet, error) {
	if set == nil {
		return nil, nil
	}
	if set.Attr != nil {
		c, err := resolveAttrClause(ctx, set.Attr, provider)
		if err != nil {
			return nil, err
		}
		return &clauseSet{leaf: &c}, nil
	}
	out := &clauseSet{op: set.Op}
	for _, item := range set.Items {
		sub, err := resolveClauseSet(ctx, item, provider)
		if err != nil {
			return nil, err
		}
		if sub != nil {
			out.items = append(out.items, sub)
		}
	}
	return out, nil
}

// resolveAttrClause pre-resolves one attribute clause's type IDs and value.
func resolveAttrClause(ctx context.Context, a *ast.Attribute, provider DataProvider) (attrClause, error) {
	typeIDs, err := Evaluate(ctx, a.Name, provider)
	if err != nil {
		return attrClause{}, fmt.Errorf("evaluating attribute name: %w", err)
	}

	if isConcreteValue(a.Value) {
		c := attrClause{op: a.Op, typeIDs: typeIDs, cardinality: a.Cardinality, isConcrete: true, reverse: a.Reverse}
		switch v := a.Value.(type) {
		case *ast.IntegerValue:
			c.concreteKind = concreteKindNumeric
			c.numericVal = float64(v.Value)
		case *ast.DecimalValue:
			c.concreteKind = concreteKindNumeric
			c.numericVal = v.Value
		case *ast.StringValue:
			c.concreteKind = concreteKindString
			c.stringVal = v.Value
		case *ast.BooleanValue:
			c.concreteKind = concreteKindBoolean
			c.boolVal = v.Value
		}
		return c, nil
	}

	switch a.Op {
	case "=", "!=":
	case "<", "<=", ">", ">=":
		return attrClause{}, fmt.Errorf("concrete-value operator %q requires a concrete value", a.Op)
	default:
		return attrClause{}, fmt.Errorf("unsupported attribute operator %q", a.Op)
	}

	c := attrClause{op: a.Op, typeIDs: typeIDs, cardinality: a.Cardinality, reverse: a.Reverse}
	if _, isAny := a.Value.(*ast.Any); isAny {
		c.valueIsAny = true
		return c, nil
	}
	c.valueSet, err = Evaluate(ctx, a.Value, provider)
	if err != nil {
		return attrClause{}, fmt.Errorf("evaluating attribute value: %w", err)
	}
	return c, nil
}

// groupSatisfiesSet reports whether a single relationship group satisfies the
// clause tree, honoring AND/OR at each level.
func groupSatisfiesSet(rels []Relationship, cs *clauseSet) bool {
	if cs == nil {
		return true
	}
	if cs.leaf != nil {
		return clauseSatisfied(rels, *cs.leaf)
	}
	if cs.op == ast.AttrSetOr {
		for _, item := range cs.items {
			if groupSatisfiesSet(rels, item) {
				return true
			}
		}
		return false
	}
	for _, item := range cs.items {
		if !groupSatisfiesSet(rels, item) {
			return false
		}
	}
	return true
}

// clauseSatisfied reports whether one relationship group satisfies one clause:
// it counts the matching relationships and validates the count against the
// clause's cardinality.
func clauseSatisfied(rels []Relationship, c attrClause) bool {
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
			continue
		}
		if c.valueIsAny || c.valueSet.Contains(r.TargetID) {
			count++
		}
	}

	// "attr != X" selects relationships OF THAT TYPE whose value is not in X,
	// so it counts the complement within the type — not the complement of the
	// concept. For concrete values the operator is already applied inside
	// matchConcreteValue, so no inversion happens here.
	if c.op == "!=" && !c.isConcrete {
		totalOfType := 0
		for _, r := range rels {
			if c.typeIDs.Contains(r.TypeID) {
				totalOfType++
			}
		}
		count = totalOfType - count
	}

	return cardinalitySatisfied(c.cardinality, count)
}

// matchConcreteValue checks if a stored concrete value satisfies the clause comparison.
func matchConcreteValue(cv *ConcreteValue, c attrClause) bool {
	switch c.concreteKind {
	case concreteKindNumeric:
		if cv.Kind != "integer" && cv.Kind != "decimal" {
			return false
		}
		f, err := strconv.ParseFloat(cv.Value, 64)
		if err != nil {
			return false
		}
		return compareFloat(f, c.op, c.numericVal)
	case concreteKindString:
		if cv.Kind != "string" {
			return false
		}
		return compareString(cv.Value, c.op, c.stringVal)
	case concreteKindBoolean:
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
			// A member field may hold a literal (a map target, an order, a flag)
			// rather than a concept reference, so a literal must NOT go through
			// Evaluate: there is nothing to resolve, and the type switch there
			// rejects it with "unsupported AST node type: *ast.StringValue".
			// That made every `{{ M mapTarget = "..." }}` fail at runtime even
			// though the README lists memberField as supported.
			var valueSet Set
			if field.Value != nil {
				if lit, ok := literalText(field.Value); ok {
					valueSet = NewSetFromSlice([]string{lit})
				} else {
					valueSet, err = Evaluate(ctx, field.Value, provider)
					if err != nil {
						return nil, fmt.Errorf("evaluating member filter value: %w", err)
					}
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
				return nil, fmt.Errorf("%w: RefsetMembersFiltered: %w", ErrProvider, innerErr)
			}
			result = result.Intersect(filtered)
		}
	}

	// Description filters — build a single DescriptionFilterOpts from all
	// clauses, then call MatchDescription once. The result is a set of concept
	// IDs whose descriptions match; intersect with the current base (or subtract
	// when negated).
	// Dialect clauses are answered by MatchDialect, not MatchDescription, so a
	// dialect-only constraint must not consult MatchDescription at all: passing
	// it zero-value Opts asks "every description", and under the contract an
	// empty input yields the empty Set. Even a lenient provider drops every
	// concept that happens to have no descriptions.
	hasDescriptionClause := false
	for _, f := range descFilters {
		if _, isDialect := f.(*ast.DialectFilter); !isDialect {
			hasDescriptionClause = true
			break
		}
	}

	if hasDescriptionClause {
		matches, err := matchDescriptionFilters(ctx, descFilters, provider)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = matches
		} else {
			result = result.Intersect(matches)
		}
	}

	// Dialect filters — separate from description filters because they use a
	// dedicated provider method.
	//
	// The provider is the single owner of the negation: buildDialectFilterOpts
	// puts the operator in DialectFilterOpts.Negate and MatchDialect answers
	// accordingly. This loop must therefore intersect whatever comes back and
	// never subtract, or the negation would be applied twice.
	for _, f := range descFilters {
		df, ok := f.(*ast.DialectFilter)
		if !ok {
			continue
		}
		if len(df.Dialects) == 0 && len(df.Aliases) == 0 {
			// Neither form carried an entry, so there is nothing to filter by.
			// Intersecting with an empty match instead would return the empty set
			// for every such expression, which is what happened before this node
			// was populated at all.
			return nil, fmt.Errorf("%w: dialect filter names no dialect", ErrUnsupportedFeature)
		}
		dOpts, err := buildDialectFilterOpts(ctx, df, provider)
		if err != nil {
			return nil, err
		}
		matches, err := provider.MatchDialect(ctx, result, dOpts)
		if err != nil {
			return nil, fmt.Errorf("%w: MatchDialect: %w", ErrProvider, err)
		}
		if matches == nil {
			matches = NewSet()
		}
		result = result.Intersect(matches)
	}

	// Concept filters — one provider call per clause, composed with Intersect
	// for "=" and Minus for "!=".
	//
	// A concept filter clause identifies a set of CONCEPTS, so set-level
	// composition is exact here. It is not exact for description filters, which
	// is why those are handled separately above.
	//
	// The previous implementation collapsed every clause into one Opts plus a
	// single `negate` bool, set by any clause using "!=", and then subtracted the
	// set satisfying the whole conjunction. That silently discarded the sibling
	// positive clauses: `{{ C active = false, definitionStatusId != X }}`
	// returned active concepts, because the negated conjunction was empty and
	// Minus subtracted nothing.
	for _, cl := range buildConceptFilterClauses(conceptFilters) {
		plan, err := planConceptClause(ctx, cl.filter, provider)
		if err != nil {
			return nil, err
		}
		// The clause names no concept, so it matches nothing. Passing empty Opts
		// to the provider would instead mean "no filter on this dimension" and
		// the clause would match everything.
		matched := NewSet()
		if !plan.matchesNothing {
			for i, opts := range plan.calls {
				got, err := provider.FilterConcepts(ctx, result, opts)
				if err != nil {
					return nil, fmt.Errorf("%w: FilterConcepts: %w", ErrProvider, err)
				}
				switch {
				case i == 0:
					matched = nonNil(got)
				case plan.intersect:
					matched = matched.Intersect(nonNil(got))
				default:
					matched = matched.Union(nonNil(got))
				}
			}
		}
		if cl.negate {
			result = result.Minus(matched)
		} else {
			result = result.Intersect(matched)
		}
	}

	return result, nil
}

// conceptFilterClause is a single concept filter clause with its own polarity,
// so each one can be composed independently.
type conceptFilterClause struct {
	filter ast.Filter
	negate bool
}

// buildConceptFilterClauses splits the concept filter family into individually
// composable clauses.
func buildConceptFilterClauses(filters []ast.Filter) []conceptFilterClause {
	out := make([]conceptFilterClause, 0, len(filters))
	for _, f := range filters {
		negate := false
		switch x := f.(type) {
		case *ast.DefinitionStatusFilter:
			negate = x.Op == "!="
		case *ast.ModuleFilter:
			negate = x.Op == "!="
		case *ast.EffectiveTimeFilter:
			// The operator is part of the comparison the provider performs
			// (>=, <, ...), so it is passed through rather than negated here.
		case *ast.ActiveFilter:
			// ActiveFilter carries a bool, not an operator: `active = false` is
			// a positive clause selecting inactive concepts.
		}
		out = append(out, conceptFilterClause{filter: f, negate: negate})
	}
	return out
}

// conceptClausePlan is how one concept filter clause becomes provider calls.
type conceptClausePlan struct {
	// calls holds one ConceptFilterOpts per FilterConcepts call. There is more
	// than one only for an effectiveTime SET, because ConceptFilterOpts carries a
	// single time value and a single operator.
	calls []ConceptFilterOpts

	// intersect combines the calls with AND rather than OR. A value set means
	// any-of, which is a union — except under "!=", where "not any of these" is
	// the intersection of the individual negations.
	intersect bool

	// matchesNothing reports that the clause's operand names no concept, so the
	// clause matches nothing. It cannot be expressed through the Opts, where an
	// empty field means "do not filter on this dimension".
	matchesNothing bool
}

// planConceptClause builds the provider calls for exactly one concept filter
// clause.
//
// Most clauses are one call. A value set is expressed differently per dimension:
// definitionStatus and module carry any-of natively in their ID slices, so a set
// is still one call, while effectiveTime carries one value and one operator and
// therefore becomes one call per value. Decomposing it that way is exact — a
// union of single-value filters is what any-of means — and needs nothing of the
// provider, which is why it is preferred over widening ConceptFilterOpts: a new
// field would be silently ignored by every provider written against the current
// contract, and the filter would quietly match on one value out of the set.
func planConceptClause(ctx context.Context, f ast.Filter, provider DataProvider) (conceptClausePlan, error) {
	var opts ConceptFilterOpts
	switch x := f.(type) {
	case *ast.ActiveFilter:
		b := x.Value
		opts.Active = &b
	case *ast.DefinitionStatusFilter:
		// Every value, not just the first: DefinitionStatusIDs is an any-of
		// slice, so a set is expressible here.
		for _, val := range x.Values {
			ids, resolved, err := resolveFilterIDs(ctx, val, provider)
			if err != nil {
				return conceptClausePlan{}, fmt.Errorf("evaluating definitionStatus filter: %w", err)
			}
			if !resolved {
				continue
			}
			opts.DefinitionStatusIDs = append(opts.DefinitionStatusIDs, ids...)
		}
		if len(x.Values) > 0 && len(opts.DefinitionStatusIDs) == 0 {
			return conceptClausePlan{matchesNothing: true}, nil
		}
	case *ast.ModuleFilter:
		for _, mod := range x.Modules {
			ids, resolved, err := resolveFilterIDs(ctx, mod, provider)
			if err != nil {
				return conceptClausePlan{}, fmt.Errorf("evaluating module filter: %w", err)
			}
			if !resolved {
				continue
			}
			opts.ModuleIDs = append(opts.ModuleIDs, ids...)
		}
		if len(x.Modules) > 0 && len(opts.ModuleIDs) == 0 {
			return conceptClausePlan{matchesNothing: true}, nil
		}
	case *ast.EffectiveTimeFilter:
		opts.EffectiveTimeOp = x.Op
		if len(x.Values) <= 1 {
			if len(x.Values) == 1 {
				opts.EffectiveTime = x.Values[0]
			}
			break
		}

		// One call per value. Under "!=" the set reads "none of these", so the
		// per-value results are intersected; every other operator keeps the
		// any-of union.
		plan := conceptClausePlan{intersect: x.Op == "!="}
		for _, v := range x.Values {
			plan.calls = append(plan.calls, ConceptFilterOpts{EffectiveTime: v, EffectiveTimeOp: x.Op})
		}
		return plan, nil
	}
	return conceptClausePlan{calls: []ConceptFilterOpts{opts}}, nil
}

// resolveFilterIDs evaluates a filter operand to concept IDs, falling back to
// the literal SCTID when the provider does not enumerate it (well-known metadata
// concepts are often absent from a test provider's ConceptExists).
//
// The second result distinguishes "no operand" from "an operand that resolved to
// nothing", which the ID slice alone cannot express: an empty slice in the Opts
// means "do not filter on this dimension", so returning nil for an operand that
// legitimately matched no concept turned the clause into a no-op and the filter
// matched EVERYTHING. Measured before the fix:
//
//	{{ C definitionStatusId = (< 900000000000073002) }}  -> the whole base set
//
// when the correct answer is the empty set.
func resolveFilterIDs(ctx context.Context, e ast.Expression, provider DataProvider) (ids []string, resolved bool, err error) {
	if e == nil {
		return nil, false, nil
	}
	set, err := Evaluate(ctx, e, provider)
	if err != nil {
		return nil, false, err
	}
	if set != nil && set.Len() > 0 {
		return set.Slice(), true, nil
	}
	if ref, ok := e.(*ast.ConceptRef); ok {
		return []string{ref.ID}, true, nil
	}
	// The operand is present but names nothing, so the clause matches nothing.
	return nil, false, nil
}

// matchDescriptionFilters resolves the description family to the set of concepts
// having a description that satisfies it.
//
// A negated clause is routed to NegatingDescriptionProvider when the provider
// offers it, because the negation is per description ROW and cannot be composed
// from sets: `{{ D type != fsn }}` means "has a description whose type is not
// FSN", so subtracting the concepts that have an FSN also removes concepts holding
// both an FSN and a synonym. Without the capability the form is reported rather
// than answered wrongly.
func matchDescriptionFilters(ctx context.Context, descFilters []ast.Filter, provider DataProvider) (Set, error) {
	kind, negated := negatedDescriptionFilter(descFilters)
	negatingProvider, canNegate := provider.(NegatingDescriptionProvider)
	if negated && !canNegate {
		return nil, fmt.Errorf("%w: negated description filter (%s) requires a provider implementing ecl.NegatingDescriptionProvider; its negation is per description row and cannot be composed from sets",
			ErrUnsupportedFeature, kind)
	}
	if err := unsupportedDescriptionFilter(descFilters); err != nil {
		return nil, err
	}

	opts, matchesNothing, err := buildDescriptionFilterOpts(ctx, descFilters, provider)
	if err != nil {
		return nil, err
	}
	if matchesNothing {
		// A clause named no concept, so nothing can match.
		return NewSet(), nil
	}

	if negated {
		matches, err := negatingProvider.MatchDescriptionNegated(ctx, negatedOptsFor(descFilters, opts))
		if err != nil {
			return nil, fmt.Errorf("%w: MatchDescriptionNegated: %w", ErrProvider, err)
		}
		return nonNil(matches), nil
	}

	// One call per term, unioned. A single term — the overwhelmingly common case —
	// is one call with the same Opts as before.
	out := NewSet()
	for i, callOpts := range termCalls(descFilters, opts) {
		matches, err := provider.MatchDescription(ctx, callOpts)
		if err != nil {
			return nil, fmt.Errorf("%w: MatchDescription: %w", ErrProvider, err)
		}
		if i == 0 {
			out = nonNil(matches)
			continue
		}
		out = out.Union(nonNil(matches))
	}
	return out, nil
}

// termCalls expands a positive term SET into one DescriptionFilterOpts per term.
//
// `{{ D term = ("heart" "attack") }}` is any-of, so it is the union of the
// single-term filters — and each call still carries the sibling clauses, which is
// what makes `(term = a OR term = b) AND language = x` come out right.
//
// Decomposing is preferred over widening DescriptionFilterOpts with a Terms
// slice: a new field is silently ignored by every provider written against the
// current contract, so the filter would quietly match on one term out of the set
// and return a narrower answer than asked for. Nothing detects that.
//
// Each term carries its own match type ("match" or "wild"), so the type travels
// with its term rather than being taken from the first one.
func termCalls(filters []ast.Filter, base DescriptionFilterOpts) []DescriptionFilterOpts {
	for _, f := range filters {
		tf, ok := f.(*ast.TermFilter)
		if !ok || len(tf.Terms) <= 1 {
			continue
		}
		out := make([]DescriptionFilterOpts, 0, len(tf.Terms))
		for _, t := range tf.Terms {
			opts := base
			opts.Term = t.Text
			opts.MatchType = t.MatchType
			out = append(out, opts)
		}
		return out
	}
	return []DescriptionFilterOpts{base}
}

// negatedOptsFor carries each clause's polarity alongside the values, so the
// provider can negate the dimensions that were written with "!=" and only those.
func negatedOptsFor(descFilters []ast.Filter, opts DescriptionFilterOpts) NegatedDescriptionFilterOpts {
	out := NegatedDescriptionFilterOpts{Opts: opts}
	for _, f := range descFilters {
		switch x := f.(type) {
		case *ast.TermFilter:
			out.TermNegated = x.Op == "!="
		case *ast.TypeFilter:
			out.TypeIDsNegated = x.Op == "!="
		case *ast.LanguageFilter:
			out.LanguagesNegated = x.Op == "!="
		}
	}
	return out
}

// categorizeFilters splits a flat list of filters into description, concept,
// and member families. Ambiguous filters (active/module/effectiveTime) default
// to the concept family — this is the most common usage in practice.
func categorizeFilters(filters []ast.Filter) (desc, concept, member []ast.Filter) {
	for _, f := range filters {
		switch f.(type) {
		case *ast.TermFilter, *ast.TypeFilter, *ast.LanguageFilter, *ast.DialectFilter,
			*ast.DescriptionIDFilter:
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

// negatedDescriptionFilter reports whether any description filter clause uses
// "!=", and which kind it was. Dialect filters count: they belong to the same
// family and their negation is equally per-row.
func negatedDescriptionFilter(filters []ast.Filter) (string, bool) {
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.TermFilter:
			if x.Op == "!=" {
				return "term", true
			}
		case *ast.TypeFilter:
			if x.Op == "!=" {
				return "type", true
			}
		case *ast.LanguageFilter:
			if x.Op == "!=" {
				return "language", true
			}
			// A DialectFilter is deliberately absent here. Its negation travels
			// to the provider through DialectFilterOpts.Negate, so it needs no
			// NegatingDescriptionProvider — and counting it here demanded that
			// capability for an expression the provider can already answer,
			// while the dialect loop applied the clause a second time.
		}
	}
	return "", false
}

// unsupportedDescriptionFilter reports the description filter forms that
// DescriptionFilterOpts cannot express.
//
// The parser now models all of them, so the alternative would be to build Opts
// that silently drop part of the constraint — which is how a description-id
// filter used to widen a query to every descendant. Expressing them needs new
// Opts fields, i.e. a provider contract change.
func unsupportedDescriptionFilter(filters []ast.Filter) error {
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.DescriptionIDFilter:
			return fmt.Errorf("%w: description id filter needs a DescriptionFilterOpts field to pass the ids to the provider", ErrUnsupportedFeature)
		case *ast.TermFilter:
			// A POSITIVE term set is any-of, so it decomposes into one provider
			// call per term, unioned — see termCalls. A NEGATED one does not: it
			// selects concepts having a description whose term satisfies neither
			// value, and that must be decided per description ROW. Intersecting
			// per-term negations would instead accept a concept with one row
			// failing the first term and a DIFFERENT row failing the second.
			if len(x.Terms) > 1 && x.Op == "!=" {
				return fmt.Errorf("%w: a negated term filter with %d terms needs the whole set on one call, because a single description row must fail every value; DescriptionFilterOpts.Term carries one term",
					ErrUnsupportedFeature, len(x.Terms))
			}
		}
	}
	return nil
}

// buildDescriptionFilterOpts accumulates description filter clauses into a
// single DescriptionFilterOpts. Multiple clauses of the same kind are
// simplified by taking the last one (rarely seen in practice).
//
// Every clause here is positive: evaluateFiltered rejects the negated forms
// before calling this, because they cannot be composed at set level.
func buildDescriptionFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (DescriptionFilterOpts, bool, error) {
	var opts DescriptionFilterOpts
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.TermFilter:
			// unsupportedDescriptionFilter has already rejected a set, so there
			// is exactly one term here.
			if len(x.Terms) > 0 {
				opts.Term = x.Terms[0].Text
				if mt := x.Terms[0].MatchType; mt != "" {
					opts.MatchType = mt
				}
			}
		case *ast.TypeFilter:
			// Collect all type SCTIDs from the filter (any-of semantics).
			for _, typeExpr := range x.Types {
				ids, resolved, err := resolveFilterIDs(ctx, typeExpr, provider)
				if err != nil {
					return opts, false, fmt.Errorf("evaluating type filter: %w", err)
				}
				if !resolved {
					// The operand names no description type, so the filter
					// matches nothing.
					return opts, true, nil
				}
				opts.TypeIDs = append(opts.TypeIDs, ids...)
			}
		case *ast.LanguageFilter:
			opts.Languages = append(opts.Languages, x.Languages...)
		case *ast.DialectFilter:
			// Handled separately in evaluateFiltered.
			continue
		}
	}
	return opts, false, nil
}

// buildDialectFilterOpts converts an ast.DialectFilter into DialectFilterOpts
// by evaluating each dialect entry's dialect and acceptability expressions.
func buildDialectFilterOpts(ctx context.Context, df *ast.DialectFilter, provider DataProvider) (DialectFilterOpts, error) {
	opts := DialectFilterOpts{Negate: df.Op == "!="}
	for _, entry := range df.Dialects {
		var dialectIDs []string
		if entry.Dialect != nil {
			ids, err := Evaluate(ctx, entry.Dialect, provider)
			if err != nil {
				return opts, fmt.Errorf("evaluating dialect: %w", err)
			}
			if ids != nil && ids.Len() > 0 {
				dialectIDs = ids.Slice()
			} else if ref, ok := entry.Dialect.(*ast.ConceptRef); ok {
				dialectIDs = []string{ref.ID}
			}
		}
		// Every acceptability, not just the first: AcceptabilityIDs is an any-of
		// slice, so a set is expressible here.
		var acceptIDs []string
		for _, acc := range entry.Acceptabilities {
			ids, _, err := resolveFilterIDs(ctx, acc, provider)
			if err != nil {
				return opts, fmt.Errorf("evaluating acceptability: %w", err)
			}
			acceptIDs = append(acceptIDs, ids...)
		}
		opts.Dialects = append(opts.Dialects, DialectEntryOpts{
			DialectIDs:       dialectIDs,
			AcceptabilityIDs: acceptIDs,
		})
	}

	if err := appendAliasDialects(ctx, &opts, df.Aliases, provider); err != nil {
		return opts, err
	}
	return opts, nil
}

// appendAliasDialects resolves the alias form of the dialect filter through the
// optional ecl.DialectAliasResolver and appends the result as ordinary entries.
//
// Alias resolution is terminology data — see the capability's godoc — so without
// a provider that implements it the only honest answer is ErrUnsupportedFeature.
// An alias the provider does not return is reported for the same reason: dropping
// it would widen the query to every dialect, which is the failure mode this
// package refuses everywhere else.
func appendAliasDialects(ctx context.Context, opts *DialectFilterOpts, aliases []ast.DialectAliasEntry, provider DataProvider) error {
	if len(aliases) == 0 {
		return nil
	}

	resolver, ok := provider.(DialectAliasResolver)
	if !ok {
		names := make([]string, 0, len(aliases))
		for _, a := range aliases {
			names = append(names, a.Alias)
		}
		return fmt.Errorf("%w: the dialect alias form (%s) requires a provider implementing ecl.DialectAliasResolver, because mapping an alias to a language reference set is terminology data; the dialectId form always works",
			ErrUnsupportedFeature, strings.Join(names, ", "))
	}

	names := make([]string, 0, len(aliases))
	for _, a := range aliases {
		names = append(names, a.Alias)
	}
	resolved, err := resolver.ResolveDialectAliases(ctx, names)
	if err != nil {
		return fmt.Errorf("%w: ResolveDialectAliases: %w", ErrProvider, err)
	}

	for _, a := range aliases {
		dialectIDs, known := resolved[a.Alias]
		if !known || len(dialectIDs) == 0 {
			return fmt.Errorf("%w: the provider does not resolve the dialect alias %q; use its dialectId instead of matching every dialect",
				ErrUnsupportedFeature, a.Alias)
		}

		var acceptIDs []string
		for _, acc := range a.Acceptabilities {
			ids, _, err := resolveFilterIDs(ctx, acc, provider)
			if err != nil {
				return fmt.Errorf("evaluating acceptability: %w", err)
			}
			acceptIDs = append(acceptIDs, ids...)
		}
		opts.Dialects = append(opts.Dialects, DialectEntryOpts{
			DialectIDs:       dialectIDs,
			AcceptabilityIDs: acceptIDs,
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// Concrete value comparisons (Phase 5.2)
// ---------------------------------------------------------------------------.

// literalText renders a concrete literal node as the text a provider stores for
// it, reporting false for anything that is not a literal.
//
// Used by member field filters, whose fields hold arbitrary values (map targets,
// orders, flags) rather than concept IDs.
func literalText(e ast.Expression) (string, bool) {
	switch v := e.(type) {
	case *ast.StringValue:
		return v.Value, true
	case *ast.IntegerValue:
		return strconv.Itoa(v.Value), true
	case *ast.DecimalValue:
		return strconv.FormatFloat(v.Value, 'f', -1, 64), true
	case *ast.BooleanValue:
		return strconv.FormatBool(v.Value), true
	}
	return "", false
}

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
// Supports numeric (integer/decimal) comparisons with all operators (=, !=,
// <, <=, >, >=), and string/boolean comparisons with = and != only.
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

	valuesByConcept, err := concreteValuesFor(ctx, provider, focus, typeIDList)
	if err != nil {
		return nil, err
	}

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		if err := ctx.Err(); err != nil {
			iterErr = err
			return false
		}
		// Count the values that satisfy the comparison, so the clause's
		// cardinality can be applied. This used to be a bool with an early
		// break, which made [0..0], [1..1] and [2..*] all behave as "at least
		// one".
		matches := 0
		for _, typeID := range typeIDList {
			for _, cv := range valuesByConcept[id][typeID] {
				switch {
				case haveNumeric && (cv.Kind == "integer" || cv.Kind == "decimal"):
					f, parseErr := strconv.ParseFloat(cv.Value, 64)
					if parseErr != nil {
						continue
					}
					if compareFloat(f, attr.Op, numeric) {
						matches++
					}
				case haveString && cv.Kind == "string":
					if compareString(cv.Value, attr.Op, strVal) {
						matches++
					}
				case haveBool && cv.Kind == "boolean":
					stored := cv.Value == "true"
					if compareBool(stored, attr.Op, boolVal) {
						matches++
					}
				}
			}
		}
		// compareFloat/compareString/compareBool already applied the operator,
		// so `matches` is "how many stored values satisfy the comparison". No
		// extra inversion for "!=" -- subtracting from the total here, as the
		// concept-valued path does, would invert a correct result.
		if cardinalitySatisfied(attr.Cardinality, matches) {
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
// It uses Ancestors, not Parents: a member whose immediate parent is outside
// baseSet may still have a grandparent inside it, and such a member is not a
// top. With Parents (depth 1) it was wrongly reported as one, which happens for
// any set that is not transitively closed -- i.e. the result of a refset, a
// filter or a MINUS, which is the common case. `!!> << X` happened to be right
// by construction, which is why the tests passed.
func topOfSet(ctx context.Context, baseSet Set, provider DataProvider) (Set, error) {
	if baseSet == nil || baseSet.Len() == 0 {
		return baseSet, nil
	}
	out := newMapSet()
	var iterErr error
	baseSet.Iter(func(id string) bool {
		// One provider call per concept, so this is where a canceled request has
		// to stop. Checking only on entry to Evaluate let a canceled context run
		// the whole loop.
		if err := ctx.Err(); err != nil {
			iterErr = err
			return false
		}
		ancestors, err := provider.Ancestors(ctx, []string{id}, false)
		if err != nil {
			iterErr = fmt.Errorf("%w: Ancestors(%s): %w", ErrProvider, id, err)
			return false
		}
		if ancestors == nil || ancestors.Intersect(baseSet).Len() == 0 {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// bottomOfSet returns the concepts in baseSet that have no proper descendant in
// baseSet (i.e. the leaves of the sub-graph induced by baseSet).
//
// Symmetric to topOfSet: it uses Descendants rather than Children, for the same
// reason.
func bottomOfSet(ctx context.Context, baseSet Set, provider DataProvider) (Set, error) {
	if baseSet == nil || baseSet.Len() == 0 {
		return baseSet, nil
	}
	out := newMapSet()
	var iterErr error
	baseSet.Iter(func(id string) bool {
		// One provider call per concept, so this is where a canceled request has
		// to stop. Checking only on entry to Evaluate let a canceled context run
		// the whole loop.
		if err := ctx.Err(); err != nil {
			iterErr = err
			return false
		}
		descendants, err := provider.Descendants(ctx, []string{id}, false)
		if err != nil {
			iterErr = fmt.Errorf("%w: Descendants(%s): %w", ErrProvider, id, err)
			return false
		}
		if descendants == nil || descendants.Intersect(baseSet).Len() == 0 {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}
