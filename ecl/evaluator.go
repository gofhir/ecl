// Package ecl parses and evaluates SNOMED CT Expression Constraint Language (ECL) expressions
// against a pluggable DataProvider that supplies the underlying terminology.
package ecl

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/ecl/ast"
)

// ErrUnsupportedFeature marks an ECL construct the evaluator recognizes and will
// not answer. Callers can classify it and reply 422 or 501 instead of serving a
// wrong result set:
//
//	if errors.Is(err, ecl.ErrUnsupportedFeature) { /* not implemented */ }
//
// It is returned rather than a silently incorrect set, which is the whole point.
// Three different situations produce it, and they are not equally permanent:
//
//   - The provider lacks an optional CAPABILITY the construct needs. Implement it
//     and the construct works; see capabilities.go. Most of them are this.
//   - The construct has no meaning to answer. A reverse attribute inside an
//     attribute group and a projection naming several member fields are here:
//     the reasons are written where the error is raised, and no amount of
//     provider work changes them.
//   - Two per-row negations would have to travel on one call, and neither
//     capability expresses that. Combining a description id filter with a negated
//     term clause is the example.
//
// The error text says which, so a caller can tell "implement this interface" from
// "this expression will never work".
var ErrUnsupportedFeature = errors.New("unsupported ECL feature")

// ErrProvider wraps a failure that came from the DataProvider rather than from
// the expression, so a caller can answer 503 instead of 400:
//
//	if errors.Is(err, ecl.ErrProvider) { /* the backend is unhealthy */ }
var ErrProvider = errors.New("data provider error")

// providerError classifies an error a DataProvider returned, at the one place
// every call goes through. It wraps the cause with ErrProvider and the method,
// keeping the cause reachable through errors.Is, except for two errors that are
// not a failing backend: ErrUnsupportedFeature, which a capability returns to
// refuse a form it cannot answer (501 rather than 503), and the caller's own
// cancellation or deadline. Those pass through with the method named but no
// ErrProvider.
//
// The second is decided by the caller's ctx, not by the error's identity: a
// backend's own query timeout also matches context.DeadlineExceeded, and while
// the caller's ctx is live that is a slow backend, which is exactly what
// ErrProvider reports; once the caller's ctx is done, any error is the request
// being abandoned.
//
// Before it existed, nine required methods returned their errors unwrapped
// (H5 in docs/analisis-adversarial-2026-10-12.md), so `errors.Is(err,
// ErrProvider)` held for some backend failures and not others.
func providerError(ctx context.Context, method string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrUnsupportedFeature) {
		return fmt.Errorf("%s: %w", method, err)
	}
	if cerr := ctx.Err(); cerr != nil {
		// The caller gave up, whatever the backend reported for it: a driver
		// often answers a canceled request with its own error ("conn closed")
		// rather than one wrapping context.Canceled. Both stay reachable.
		// The backend's own error is kept as text only: wrapping it as well
		// could carry an ErrProvider it holds, and the request was abandoned.
		if errors.Is(err, cerr) {
			return fmt.Errorf("%s: %w", method, cerr)
		}
		return fmt.Errorf("%s: %w (backend reported: %v)", method, cerr, err) //nolint:errorlint // %v on purpose: see above
	}
	return fmt.Errorf("%w: %s: %w", ErrProvider, method, err)
}

// providerSet applies providerError to a provider call that returns a Set, so
// a call site reads `return providerSet(ctx, "X")(provider.X(…))`.
func providerSet(ctx context.Context, method string) func(Set, error) (Set, error) {
	return func(s Set, err error) (Set, error) {
		if err != nil {
			return nil, providerError(ctx, method, err)
		}
		return s, nil
	}
}

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
//   - Filter constraints: term, type, language, dialect (both dialectId and the
//     alias form), description id, active, module, definitionStatus,
//     effectiveTime. Negation is per clause. A negated CONCEPT filter
//     ({{ C ... != ... }}) always works; a negated DESCRIPTION filter needs a
//     provider implementing NegatingDescriptionProvider, because its semantics is
//     a per-description-row negation that set arithmetic cannot express — see
//     capabilities.go.
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

// unwrapNested strips the parentheses around an expression: `((*))` is `*`.
func unwrapNested(e ast.Expression) ast.Expression {
	for {
		n, ok := e.(*ast.Nested)
		if !ok {
			return e
		}
		e = n.Inner
	}
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
		return providerSet(ctx, "ConceptExists")(provider.ConceptExists(ctx, []string{e.ID}))

	case *ast.Any:
		return providerSet(ctx, "AllConcepts")(provider.AllConcepts(ctx))

	case *ast.Nested:
		return Evaluate(ctx, e.Inner, provider)

	// ── Hierarchy operators ──────────────────────────────────────────────

	case *ast.DescendantOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Descendants")(provider.Descendants(ctx, toIDSlice(inner), false))

	case *ast.DescendantOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Descendants")(provider.Descendants(ctx, toIDSlice(inner), true))

	case *ast.ChildOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Children")(provider.Children(ctx, toIDSlice(inner), false))

	case *ast.ChildOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Children")(provider.Children(ctx, toIDSlice(inner), true))

	case *ast.AncestorOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Ancestors")(provider.Ancestors(ctx, toIDSlice(inner), false))

	case *ast.AncestorOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Ancestors")(provider.Ancestors(ctx, toIDSlice(inner), true))

	case *ast.ParentOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Parents")(provider.Parents(ctx, toIDSlice(inner), false))

	case *ast.ParentOrSelfOf:
		inner, err := Evaluate(ctx, e.Operand, provider)
		if err != nil {
			return nil, fmt.Errorf("evaluating %T: %w", expr, err)
		}
		return providerSet(ctx, "Parents")(provider.Parents(ctx, toIDSlice(inner), true))

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
		// union of members across those refsets, or project one of their fields.
		return evaluateMemberOf(ctx, e, nil, provider)

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
		return providerSet(ctx, "RelationshipTargets")(provider.RelationshipTargets(ctx, source, typeIDs))

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
			return nil, providerError(ctx, "HistoricalAssociations", err)
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
		return providerSet(ctx, "RefsetsContainingMembers")(provider.RefsetsContainingMembers(ctx, toIDSlice(operandIDs)))

	case *ast.AltIdentifier:
		return providerSet(ctx, "ResolveIdentifier")(provider.ResolveIdentifier(ctx, e.Scheme, e.Code))

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
			return nil, providerError(ctx, "reverse attribute lookup", err)
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
		return nil, providerError(ctx, "InboundRelationships", err)
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
			return nil, providerError(ctx, "ConcreteValuesBatch", err)
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
				return nil, providerError(ctx, fmt.Sprintf("ConcreteValues(%s, %s)", id, typeID), err)
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
			return nil, providerError(ctx, "PropertiesByGroupBatch", err)
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
			return nil, providerError(ctx, fmt.Sprintf("PropertiesByGroup(%s)", id), err)
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
	numericVal   number
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
		case *ast.IntegerValue, *ast.DecimalValue:
			c.concreteKind = concreteKindNumeric
			if c.numericVal, err = numericLiteral(v); err != nil {
				return attrClause{}, err
			}
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
		stored, ok := parseNumber(cv.Value)
		if !ok {
			return false
		}
		return compareNumbers(stored, c.op, c.numericVal)
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
// A member filter over a PROJECTION is the exception to step 4: it restricts
// which member rows are projected rather than which results survive, so it is
// handed to evaluateMemberOf and the operand is evaluated once, there. See
// projectedMemberOf. Several `{{ M … }}` blocks over one operand go to
// evaluateMemberBlocks.
func evaluateFiltered(ctx context.Context, e *ast.Filtered, provider DataProvider) (Set, error) {
	if blocks, mo := memberBlocks(e); len(blocks) > 1 {
		return evaluateMemberBlocks(ctx, blocks, mo, provider)
	}
	descFilters, conceptFilters, memberFilters, err := categorizeFilters(e.Filters, e.Family)
	if err != nil {
		return nil, err
	}

	// A projecting memberOf must see the member filters, so its operand is NOT
	// evaluated by the generic path above: `^ [targetComponentId] X {{ M
	// referencedComponentId = Y }}` filters one column and returns another, and
	// intersecting the returned column with a filter on a different column
	// answers nothing anybody asked.
	if len(memberFilters) > 0 && memberOfOperand(e.Operand) == nil {
		// The grammar admits `{{ M … }}` after a focus with no `^`, where there
		// are no member rows to filter, and a hand-built tree may nest member
		// clauses of Family "" over each other. Reading the focus concepts as
		// reference sets answers a question nobody wrote.
		return nil, errMemberFilterNeedsMemberOf
	}
	clauses, noRows, err := memberFilterOptsOf(ctx, memberFilters, provider)
	if err != nil {
		return nil, err
	}
	// The rows decide when the filter names the active column; otherwise `^ R`
	// means the active rows, and that default is applied per ROW where the
	// provider can carry it, so that an inactive row's value cannot qualify a
	// member through its active sibling. Elsewhere it stays the intersection
	// with the active members below, as before the column existed.
	rowsDecide := false
	if len(clauses) > 0 && memberOfOperand(e.Operand) != nil {
		clauses, rowsDecide = rowActiveDefault(clauses, provider)
	}

	var base Set
	if rowsDecide && projectedMemberOf(e.Operand) == nil {
		if noRows {
			// Nothing can match, so the clauses are not sent; the reference
			// set operand is still evaluated so that its errors surface.
			if _, err := Evaluate(ctx, memberOfOperand(e.Operand).Operand, provider); err != nil {
				return nil, fmt.Errorf("evaluating member filter refset: %w", err)
			}
			return NewSet(), nil
		}
		// The rows decide, so `^ R`'s own member set would be fetched only to be
		// replaced; membersMatching resolves the reference sets, and their
		// errors, itself.
		matched, err := membersMatching(ctx, e.Operand, clauses, provider)
		if err != nil {
			return nil, err
		}
		base, clauses = matched, nil
	} else if mo := projectedMemberOf(e.Operand); mo != nil {
		if base, err = evaluateMemberOf(ctx, mo, clauses, provider); err != nil {
			return nil, err
		}
		// Consumed by the projection. Everything below then treats the projected
		// values as an ordinary set of concept ids, which is what they are:
		// description and concept filters still apply to them.
		clauses = nil
	} else if base, err = Evaluate(ctx, e.Operand, provider); err != nil {
		return nil, fmt.Errorf("evaluating Filtered operand: %w", err)
	}
	// After the operand, so its errors and refusals still surface: a clause
	// that can match no row empties the result, it does not excuse the operand.
	if noRows {
		return NewSet(), nil
	}

	result := base

	// Every clause may have constrained nothing — excluding a module set that
	// names none — and then the members stand as they are.
	if len(clauses) > 0 {
		matched, err := membersMatching(ctx, e.Operand, clauses, provider)
		if err != nil {
			return nil, err
		}
		if rowsDecide {
			// The clauses say which rows count. `^ R` alone means the active
			// members, so intersecting with it would leave `{{ M active =
			// false }}` only the members that ALSO have an active row.
			result = matched
		} else {
			result = result.Intersect(matched)
		}
	}

	// Description filters, whole or per dimension; see filterByDescriptionBlock.
	// A wildcard operand is every concept: telling the row matcher "every
	// concept" (nil) spares it an IN-list of the whole terminology.
	_, wildcard := unwrapNested(e.Operand).(*ast.Any)
	if result, err = filterByDescriptionBlock(ctx, descFilters, result, wildcard, provider); err != nil {
		return nil, err
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
					return nil, providerError(ctx, "FilterConcepts", err)
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

// filterByDescriptionBlock applies one description block's clauses to result:
// whole, on single rows, through a DescriptionRowMatcher when the provider has
// one, and otherwise per dimension — MatchDescription for the clauses it
// carries, then MatchDialect for each dialect clause.
//
// On the per-dimension path, dialect clauses are answered by MatchDialect, not
// MatchDescription, so a dialect-only block must not consult MatchDescription
// at all: zero-value Opts ask "every description", which the contract answers
// with the empty Set.
func filterByDescriptionBlock(ctx context.Context, descFilters []ast.Filter, result Set, wildcard bool, provider DataProvider) (Set, error) {
	if len(descFilters) == 0 {
		return result, nil
	}
	// A nil result is empty, as Evaluate reads every nil Set.
	result = nonNil(result)
	if rows, ok := provider.(DescriptionRowMatcher); ok {
		// The whole block on single rows, so clauses on one dimension and
		// dialect clauses all hold on one description (H2, H3).
		candidates := result
		if wildcard {
			candidates = nil
		}
		matches, err := matchDescriptionRows(ctx, descFilters, candidates, result.Len() == 0, rows, provider)
		if err != nil {
			return nil, err
		}
		return result.Intersect(matches), nil
	}

	hasDescriptionClause := hasNonDialectClause(descFilters)

	if hasDescriptionClause {
		matches, err := matchDescriptionFilters(ctx, descFilters, provider)
		if err != nil {
			return nil, err
		}
		result = result.Intersect(matches)
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
		dOpts, err := buildDialectFilterOpts(ctx, df, provider)
		if err != nil {
			return nil, err
		}
		matches, err := provider.MatchDialect(ctx, result, dOpts)
		if err != nil {
			return nil, providerError(ctx, "MatchDialect", err)
		}
		if matches == nil {
			matches = NewSet()
		}
		result = result.Intersect(matches)
	}
	return result, nil
}

// matchDescriptionRows answers one description block through a
// DescriptionRowMatcher: every clause becomes a condition the provider tests
// on each description row, so nothing is fanned out, intersected at concept
// level or refused for being repeated. A clause that constrains nothing (a set
// naming no value, `active = *`) adds no condition; the block still asks for
// a description.
//
// With noCandidates the provider is not called once every clause is resolved:
// there is nothing the block could keep, but its operands' errors still surface.
func matchDescriptionRows(ctx context.Context, filters []ast.Filter, candidates Set, noCandidates bool, rows DescriptionRowMatcher, provider DataProvider) (Set, error) {
	f := DescriptionRowFilter{Concepts: candidates}
	matchesNothing := false
	for _, clause := range filters {
		// Every clause is resolved, even after one that can match nothing, so
		// a later clause's error or refusal still surfaces.
		nothing, err := addRowCondition(ctx, &f, clause, provider)
		if err != nil {
			return nil, err
		}
		matchesNothing = matchesNothing || nothing
	}
	if matchesNothing || noCandidates {
		return NewSet(), nil
	}
	matches, err := rows.MatchDescriptionRows(ctx, f)
	if err != nil {
		return nil, providerError(ctx, "MatchDescriptionRows", err)
	}
	return nonNil(matches), nil
}

// addRowCondition adds one clause to a DescriptionRowFilter. It reports a
// clause that no row can satisfy, so the block answers the empty set.
func addRowCondition(ctx context.Context, f *DescriptionRowFilter, clause ast.Filter, provider DataProvider) (nothing bool, err error) {
	switch x := clause.(type) {
	case *ast.TermFilter:
		if len(x.Terms) == 0 {
			// Only a hand-built tree names no term; it constrains nothing, as
			// on the per-dimension path.
			return false, nil
		}
		c := TermCondition{Negate: x.Op == "!="}
		for _, t := range x.Terms {
			c.Terms = append(c.Terms, SearchTermOpts{Term: t.Text, MatchType: t.MatchType})
		}
		f.Terms = append(f.Terms, c)
	case *ast.ActiveFilter:
		if x.Any {
			return false, nil
		}
		if f.Active != nil && *f.Active != x.Value {
			return true, nil
		}
		v := x.Value
		f.Active = &v
	case *ast.TypeFilter, *ast.LanguageFilter, *ast.ModuleFilter, *ast.EffectiveTimeFilter, *ast.DescriptionIDFilter:
		cv, err := resolveClauseValues(ctx, clause, provider)
		if err != nil || cv.dropped || cv.nothing {
			return cv.nothing, err
		}
		switch x := clause.(type) {
		case *ast.TypeFilter:
			f.Types = append(f.Types, ValueCondition{Values: cv.values, Negate: cv.negate})
		case *ast.LanguageFilter:
			f.Languages = append(f.Languages, ValueCondition{Values: cv.values, Negate: cv.negate})
		case *ast.ModuleFilter:
			f.Modules = append(f.Modules, ValueCondition{Values: cv.values, Negate: cv.negate})
		case *ast.DescriptionIDFilter:
			f.IDs = append(f.IDs, ValueCondition{Values: cv.values, Negate: cv.negate})
		case *ast.EffectiveTimeFilter:
			f.EffectiveTimes = append(f.EffectiveTimes, TimeCondition{Values: cv.values, Op: x.Op})
		}
	case *ast.DialectFilter:
		opts, err := buildDialectFilterOpts(ctx, x, provider)
		if err != nil {
			return false, err
		}
		f.Dialects = append(f.Dialects, DialectCondition{Entries: opts.Dialects, Negate: opts.Negate})
	default:
		// A clause this path does not know must not be dropped: the block
		// would then answer more than it asks.
		return false, fmt.Errorf("%w: description filter clause %T", ErrUnsupportedFeature, clause)
	}
	return false, nil
}

// clauseValues is one value clause of a description block, resolved the same
// way for both paths that answer a block, so they cannot disagree about what
// a clause means.
type clauseValues struct {
	values  []string
	negate  bool
	dropped bool // the clause constrains nothing and is left out
	nothing bool // the clause can hold on no row
}

// resolveClauseValues resolves a type, language, module, effectiveTime or
// description id clause. A clause that names no operand — only a hand-built
// tree can — constrains nothing. One whose operands all resolve to no value
// matches nothing when positive and constrains nothing when negated: no row
// is among no values.
func resolveClauseValues(ctx context.Context, clause ast.Filter, provider DataProvider) (clauseValues, error) {
	var (
		cv       clauseValues
		operands int
	)
	switch x := clause.(type) {
	case *ast.TypeFilter:
		cv.negate, operands = x.Op == "!=", len(x.Types)
		for _, e := range x.Types {
			ids, resolved, err := resolveFilterIDs(ctx, e, provider)
			if err != nil {
				return cv, fmt.Errorf("evaluating type filter: %w", err)
			}
			if resolved {
				cv.values = append(cv.values, ids...)
			}
		}
	case *ast.LanguageFilter:
		cv.negate, operands, cv.values = x.Op == "!=", len(x.Languages), x.Languages
	case *ast.ModuleFilter:
		cv.negate, operands = x.Op == "!=", len(moduleOperands(x))
		if operands > 0 {
			ids, _, err := resolveModuleIDs(ctx, x, provider)
			if err != nil {
				return cv, err
			}
			cv.values = ids
		}
	case *ast.EffectiveTimeFilter:
		cv.values = effectiveTimeValues(x)
		cv.negate, operands = x.Op == "!=", len(cv.values)
	case *ast.DescriptionIDFilter:
		cv.negate, operands, cv.values = x.Op == "!=", len(x.IDs), x.IDs
	default:
		return cv, fmt.Errorf("resolveClauseValues: %T is not a value clause", clause)
	}
	switch {
	case operands == 0:
		cv.dropped = true
	case len(cv.values) == 0 && cv.negate:
		cv.dropped = true
	case len(cv.values) == 0:
		cv.nothing = true
	}
	return cv, nil
}

// anyDescription returns the concepts that have at least one description. It
// asks through the Active field, so it needs a provider that honors it.
func anyDescription(ctx context.Context, provider DataProvider) (Set, error) {
	if !answersDescriptionMetadata(provider) {
		return nil, fmt.Errorf("%w: every clause of this description filter constrains nothing, so it asks only for a description, which is asked through DescriptionFilterOpts.Active and needs a provider implementing ecl.DescriptionMetadataProvider", ErrUnsupportedFeature)
	}
	out := NewSet()
	for _, active := range []bool{true, false} {
		flag := active
		got, err := provider.MatchDescription(ctx, DescriptionFilterOpts{Active: &flag})
		if err != nil {
			return nil, providerError(ctx, "MatchDescription", err)
		}
		out = out.Union(nonNil(got))
	}
	return out, nil
}

// errNoDescriptionMetadata refuses a description filter on active, moduleId or
// effectiveTime for a provider that has not declared it honors those fields.
var errNoDescriptionMetadata = fmt.Errorf("%w: active, moduleId and effectiveTime in a description filter need a provider implementing ecl.DescriptionMetadataProvider, because one written before they were sent may ignore them and answer every description", ErrUnsupportedFeature)

func answersDescriptionMetadata(provider DataProvider) bool {
	_, ok := provider.(DescriptionMetadataProvider)
	return ok
}

// namesDescriptionMetadata reports whether a description block has an active,
// moduleId or effectiveTime clause.
func namesDescriptionMetadata(filters []ast.Filter) bool {
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.ActiveFilter:
			// `active = *` too: it is answered through anyDescription, which
			// asks with the Active field.
			return true
		case *ast.ModuleFilter:
			if moduleConstrains(x) {
				return true
			}
		case *ast.EffectiveTimeFilter:
			if len(effectiveTimeValues(x)) > 0 {
				return true
			}
		}
	}
	return false
}

// rowActiveDefault adds `active = 1` to a member filter that does not name the
// active column, where the provider can carry it on the same row: `^ R` means
// the active rows. It reports whether the clauses now decide which rows count.
func rowActiveDefault(clauses []MemberFilterOpts, provider DataProvider) ([]MemberFilterOpts, bool) {
	if namesActive(clauses) {
		return clauses, true
	}
	if !carriesRowActive(provider) {
		return clauses, false
	}
	return append(append([]MemberFilterOpts(nil), clauses...),
		MemberFilterOpts{FieldName: MemberColumnActive, Op: "=", ValueSet: NewSetFromSlice([]string{"1"})}), true
}

var errMemberFilterNeedsMemberOf = fmt.Errorf("%w: a member filter needs a memberOf (^) operand, whose member rows it filters", ErrUnsupportedFeature)

// memberBlocks returns the `{{ M … }}` blocks nested over one operand,
// innermost first, and that operand. It returns no block for a node that is
// not a member block, and none for a chain with a node that also holds
// description or concept clauses, which only a hand-built tree has: such a
// node is left to evaluateFiltered, which reads its clauses by family.
func memberBlocks(e *ast.Filtered) (blocks [][]ast.Filter, operand ast.Expression) {
	var outerFirst [][]ast.Filter
	for {
		if !onlyMemberClauses(e) {
			return nil, nil
		}
		outerFirst = append(outerFirst, e.Filters)
		inner, ok := e.Operand.(*ast.Filtered)
		if !ok || inner.Family != ast.FamilyMember {
			break
		}
		e = inner
	}
	slices.Reverse(outerFirst)
	return outerFirst, e.Operand
}

// onlyMemberClauses reports whether e is a member block holding member
// clauses only.
func onlyMemberClauses(e *ast.Filtered) bool {
	if e.Family != ast.FamilyMember {
		return false
	}
	desc, concept, _, err := categorizeFilters(e.Filters, e.Family)
	return err == nil && len(desc) == 0 && len(concept) == 0
}

// evaluateMemberBlocks answers several `{{ M … }}` blocks over one operand
// (H2m). Over a plain `^ R` each block is satisfied on its own member row, so
// a member passes when each block holds on some row of it: the blocks'
// members intersect. Over a field projection the result is a column of ONE
// row, so every block holds on that row, as one block would.
//
// Every block is validated before any is sent, so a refusal does not depend
// on block order, and the reference sets and their members are fetched once
// for all blocks. Otherwise each block is what evaluateFiltered makes of it.
func evaluateMemberBlocks(ctx context.Context, blocks [][]ast.Filter, operand ast.Expression, provider DataProvider) (Set, error) {
	if projectedMemberOf(operand) != nil {
		return evaluateFiltered(ctx, &ast.Filtered{Operand: operand, Filters: slices.Concat(blocks...), Family: ast.FamilyMember}, provider)
	}
	mo := memberOfOperand(operand)
	if mo == nil {
		return nil, errMemberFilterNeedsMemberOf
	}
	type block struct {
		clauses    []MemberFilterOpts
		rowsDecide bool
	}
	perBlock := make([]block, len(blocks))
	noRows := false
	for i, b := range blocks {
		clauses, none, err := memberFilterOptsOf(ctx, b, provider)
		if err != nil {
			return nil, err
		}
		perBlock[i].clauses, noRows = clauses, noRows || none
	}
	if !noRows {
		// The rows' own default too, so a block that needs several clauses on
		// one row is refused here, before any block is sent; with no row to
		// match, nothing is sent, as for one block.
		for i := range perBlock {
			if len(perBlock[i].clauses) > 0 {
				perBlock[i].clauses, perBlock[i].rowsDecide = rowActiveDefault(perBlock[i].clauses, provider)
			}
			if err := checkMemberRowSupport(perBlock[i].clauses, provider); err != nil {
				return nil, err
			}
		}
	}
	refsetIDSet, err := Evaluate(ctx, mo.Operand, provider)
	if err != nil {
		return nil, fmt.Errorf("evaluating member filter refset: %w", err)
	}
	if noRows {
		// After the operand, so its errors still surface, as for one block.
		return NewSet(), nil
	}
	refsetIDs := toIDSlice(refsetIDSet)

	// `^ R`'s own members, the active rows: what a block that constrains
	// nothing leaves, and what a block whose rows do not decide intersects.
	var members Set
	activeMembers := func() (Set, error) {
		if members == nil {
			m, err := provider.RefsetMembers(ctx, refsetIDs)
			if err != nil {
				return nil, providerError(ctx, "RefsetMembers", err)
			}
			members = nonNil(m)
		}
		return members, nil
	}

	var result Set
	for i, blk := range perBlock {
		var (
			matched Set
			err     error
		)
		if len(blk.clauses) == 0 {
			matched, err = activeMembers()
		} else {
			matched, err = matchMemberFilters(ctx, refsetIDs, blk.clauses, provider)
			if err == nil && !blk.rowsDecide && matched.Len() > 0 {
				var m Set
				if m, err = activeMembers(); err == nil {
					matched = matched.Intersect(m)
				}
			}
		}
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result = matched
		} else {
			result = result.Intersect(matched)
		}
		if result.Len() == 0 {
			// Every block was validated above; the rest cannot add a member.
			break
		}
	}
	return result, nil
}

// memberOfOperand returns the memberOf a member filter's operand is, looking
// through parentheses: `(^ R) {{ M … }}` filters R's member rows like `^ R`.
// It returns nil for an operand with no member rows.
func memberOfOperand(e ast.Expression) *ast.MemberOf {
	for {
		switch x := e.(type) {
		case *ast.MemberOf:
			return x
		case *ast.Nested:
			e = x.Inner
		default:
			return nil
		}
	}
}

// namesActive reports whether member filter clauses already constrain the
// active column.
func namesActive(filters []MemberFilterOpts) bool {
	for _, f := range filters {
		if f.FieldName == MemberColumnActive {
			return true
		}
	}
	return false
}

// carriesRowActive reports whether the provider can take an active clause on a
// member row alongside the others: it answers the column, and it takes several
// clauses on one row.
func carriesRowActive(provider DataProvider) bool {
	_, columns := provider.(MemberRowColumnsProvider)
	_, rows := provider.(RefsetFieldProjector)
	return columns && rows
}

// hasNonDialectClause reports whether a description block still has a clause
// that MatchDescription answers; dialect clauses go to MatchDialect.
func hasNonDialectClause(filters []ast.Filter) bool {
	for _, f := range filters {
		if _, isDialect := f.(*ast.DialectFilter); !isDialect {
			return true
		}
	}
	return false
}

// membersMatching returns the members of the filtered reference sets that
// satisfy the member filter clauses.
//
// The reference sets are taken from the memberOf's own operand, never from
// the members: a member is not a reference set. A member filter whose operand
// has no memberOf is refused by evaluateFiltered before it gets here.
func membersMatching(ctx context.Context, operand ast.Expression, clauses []MemberFilterOpts, provider DataProvider) (Set, error) {
	mo := memberOfOperand(operand)
	if mo == nil {
		return nil, errMemberFilterNeedsMemberOf
	}
	refsetIDSet, err := Evaluate(ctx, mo.Operand, provider)
	if err != nil {
		return nil, fmt.Errorf("evaluating member filter refset: %w", err)
	}
	return matchMemberFilters(ctx, toIDSlice(refsetIDSet), clauses, provider)
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
		if a, ok := f.(*ast.ActiveFilter); ok && a.Any {
			// Either value: every concept is active or inactive.
			continue
		}
		if m, ok := f.(*ast.ModuleFilter); ok && !moduleConstrains(m) {
			// Only a hand-built tree names no module; it constrains nothing,
			// as in the other families.
			continue
		}
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
		ids, namesNothing, err := resolveModuleIDs(ctx, x, provider)
		if err != nil {
			return conceptClausePlan{}, err
		}
		if namesNothing {
			return conceptClausePlan{matchesNothing: true}, nil
		}
		opts.ModuleIDs = ids
	case *ast.EffectiveTimeFilter:
		values := effectiveTimeValues(x)
		opts.EffectiveTimeOp = x.Op
		if len(values) <= 1 {
			// None is no constraint: EffectiveTime stays empty.
			if len(values) == 1 {
				opts.EffectiveTime = values[0]
			}
			break
		}

		// One call per value. Under "!=" the set reads "none of these", so the
		// per-value results are intersected; every other operator keeps the
		// any-of union.
		plan := conceptClausePlan{intersect: x.Op == "!="}
		for _, v := range values {
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
	// Needs no provider, so it goes before anything that calls one.
	if err := unsupportedDescriptionFilter(descFilters); err != nil {
		return nil, err
	}
	if namesDescriptionMetadata(descFilters) && !answersDescriptionMetadata(provider) {
		return nil, errNoDescriptionMetadata
	}

	// Built next, because building drops the clauses that constrain nothing —
	// excluding a set that names no type, for one — and the polarity decisions
	// below must see only the clauses that remain. Deciding on the full list let
	// a dropped negated clause flip the polarity of a kept positive one.
	opts, matchesNothing, descFilters, err := buildDescriptionFilterOpts(ctx, descFilters, provider)
	if err != nil {
		return nil, err
	}
	if matchesNothing {
		// A clause named no concept, so nothing can match.
		return NewSet(), nil
	}
	if !hasNonDialectClause(descFilters) {
		// Every clause constrained nothing, but a description filter still asks
		// for a description: `{{ D typeId != (none) }}` is "has a description of
		// a type outside the empty set". Zero-value Opts cannot ask that — the
		// contract answers them with the empty Set — so it is asked as the two
		// halves of the active flag, which together are every description.
		return anyDescription(ctx, provider)
	}

	kind, negated := negatedDescriptionFilter(descFilters)
	negatingProvider, canNegate := provider.(NegatingDescriptionProvider)
	if negated && !canNegate {
		return nil, fmt.Errorf("%w: negated description filter (%s) requires a provider implementing ecl.NegatingDescriptionProvider; its negation is per description row and cannot be composed from sets",
			ErrUnsupportedFeature, kind)
	}
	idFilter := descriptionIDFilterOf(descFilters)
	idProvider, canMatchIDs := provider.(DescriptionIDProvider)
	switch {
	case idFilter != nil && !canMatchIDs:
		return nil, fmt.Errorf("%w: the description id filter requires a provider implementing ecl.DescriptionIDProvider; a description id cannot be carried by DescriptionFilterOpts, and passing it as a field every provider ignores would WIDEN the query to every concept matching the other clauses",
			ErrUnsupportedFeature)
	case idFilter != nil && negated:
		// Both negations are per description ROW, and one call cannot carry both:
		// DescriptionIDFilterOpts.Negate inverts the id comparison only. Guessing
		// which one applies would answer a question nobody asked.
		return nil, fmt.Errorf("%w: a description id filter combined with a negated %s clause needs both negations on one call, which neither capability expresses; use one or the other",
			ErrUnsupportedFeature, kind)
	}

	calls := descriptionCalls(descFilters, opts)
	switch {
	case idFilter != nil:
		// Still one call per term: the id constraint composes with a term set the
		// same way every sibling clause does.
		return unionOfCalls(ctx, calls, "MatchDescriptionByID", func(callOpts DescriptionFilterOpts) (Set, error) {
			return idProvider.MatchDescriptionByID(ctx, DescriptionIDFilterOpts{
				Opts:           callOpts,
				DescriptionIDs: idFilter.IDs,
				Negate:         idFilter.Op == "!=",
			})
		})
	case negated:
		// Fanned out like the positive path: a positive set in a sibling clause
		// is any-of, so the union of one negated call per value is exact. A
		// single call carried only the first value of the set.
		return unionOfCalls(ctx, calls, "MatchDescriptionNegated", func(callOpts DescriptionFilterOpts) (Set, error) {
			return negatingProvider.MatchDescriptionNegated(ctx, negatedOptsFor(descFilters, callOpts))
		})
	}
	// One call per term, unioned. A single term — the overwhelmingly common case —
	// is one call with the same Opts as before.
	return unionOfCalls(ctx, calls, "MatchDescription", func(callOpts DescriptionFilterOpts) (Set, error) {
		return provider.MatchDescription(ctx, callOpts)
	})
}

// unionOfCalls makes one provider call per DescriptionFilterOpts and unions the
// results; each set the calls fan out is any-of.
func unionOfCalls(ctx context.Context, calls []DescriptionFilterOpts, method string, call func(DescriptionFilterOpts) (Set, error)) (Set, error) {
	out := NewSet()
	for i, callOpts := range calls {
		matches, err := call(callOpts)
		if err != nil {
			return nil, providerError(ctx, method, err)
		}
		if i == 0 {
			out = nonNil(matches)
			continue
		}
		out = out.Union(nonNil(matches))
	}
	return out, nil
}

// descriptionCalls expands every positive value set the single-valued fields of
// DescriptionFilterOpts cannot carry — a term set and an effectiveTime set —
// into one call per combination. Each set is any-of, so the union of the calls
// is exact, and each call keeps the sibling clauses.
func descriptionCalls(filters []ast.Filter, base DescriptionFilterOpts) []DescriptionFilterOpts {
	calls := termCalls(filters, base)
	if _, et := fannedOutSets(filters); et != nil {
		values := effectiveTimeValues(et)
		out := make([]DescriptionFilterOpts, 0, len(calls)*len(values))
		for _, c := range calls {
			for _, v := range values {
				c.EffectiveTime = v
				out = append(out, c)
			}
		}
		calls = out
	}
	return calls
}

// checkDescriptionFanOut refuses a block whose term set and effectiveTime set
// multiply into more than maxDescriptionCalls provider calls. It reads only the
// AST, so it runs before any operand is resolved through the provider. The
// product comes from an expression a caller supplies; separate blocks would not
// be the same query — each would be satisfied on its own — so the advice is a
// disjunction, which keeps the any-of.
func checkDescriptionFanOut(filters []ast.Filter) error {
	tf, et := fannedOutSets(filters)
	if tf == nil || et == nil {
		// One set alone is one call per value, as it always was.
		return nil
	}
	terms, times := len(tf.Terms), len(effectiveTimeValues(et))
	if terms*times > maxDescriptionCalls {
		return fmt.Errorf("%w: this description filter combines a term set and an effectiveTime set into %d provider calls, above the limit of %d; write it as a disjunction of smaller filtered expressions, (X {{ D … }}) OR (X {{ D … }})",
			ErrUnsupportedFeature, terms*times, maxDescriptionCalls)
	}
	return nil
}

// maxDescriptionCalls bounds the product of a term set and an effectiveTime set
// in one block. A term set alone is one call per term, as it always was.
const maxDescriptionCalls = 256

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
	if tf, _ := fannedOutSets(filters); tf != nil {
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

// fannedOutSets returns the value sets descriptionCalls expands into one call
// per value: the first term set and the first effectiveTime set of the block,
// nil where there is none. The fan-out limit asks here too, as termCalls and
// descriptionCalls do, so it counts the calls actually made.
func fannedOutSets(filters []ast.Filter) (terms *ast.TermFilter, times *ast.EffectiveTimeFilter) {
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.TermFilter:
			if terms == nil && len(x.Terms) > 1 {
				terms = x
			}
		case *ast.EffectiveTimeFilter:
			if times == nil && len(effectiveTimeValues(x)) > 1 {
				times = x
			}
		}
	}
	return terms, times
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

// categorizeFilters splits one Filtered's clauses into description, concept and
// member families. The active, moduleId and effectiveTime filters exist in all
// three, so the block's own family decides; guessing "concept" for them, as this used to,
// sent `{{ D active = false }}` and every unprefixed `{{ active = … }}` to the
// concept. An empty family is a hand-built tree from before the field existed
// and keeps that reading.
func categorizeFilters(filters []ast.Filter, family ast.FilterFamily) (desc, concept, member []ast.Filter, err error) {
	switch family {
	case ast.FamilyDescription, ast.FamilyConcept, ast.FamilyMember, "":
	default:
		return nil, nil, nil, fmt.Errorf("%w: unknown filter family %q", ErrUnsupportedFeature, family)
	}
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
			// Shared by all three families, so only the block they came from
			// says which component they constrain.
			switch family {
			case ast.FamilyDescription:
				desc = append(desc, f)
			case ast.FamilyMember:
				// Columns of the member row, sent as member field comparisons by
				// memberFilterOptsOf. Filtering the referenced CONCEPTS instead,
				// as this used to, answered a question about different rows.
				member = append(member, f)
			default:
				// FamilyConcept, or "" for a hand-built tree that predates Family.
				concept = append(concept, f)
			}
		}
	}
	return desc, concept, member, nil
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
// DescriptionFilterOpts cannot express and no capability covers.
//
// The parser models every form, so the alternative would be to build Opts that
// silently drop part of the constraint — which is how a description-id filter
// used to widen a query to every descendant, before it was rejected and then
// given ecl.DescriptionIDProvider.
func unsupportedDescriptionFilter(filters []ast.Filter) error {
	if err := checkDescriptionFanOut(filters); err != nil {
		return err
	}
	for _, f := range filters {
		x, ok := f.(*ast.TermFilter)
		if ok {
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

// buildDescriptionFilterOpts accumulates the clauses of ONE description filter
// block into a single DescriptionFilterOpts. They all have to hold on the same
// description, so a dimension named twice is a conjunction on one row:
//
//   - type, language and moduleId keep the values both clauses allow, which is
//     exact: `language = en, language = es` can hold on no description;
//   - two active clauses agree or match nothing;
//   - a second term or effectiveTime clause, which the single Term and
//     EffectiveTime fields cannot carry, is reported unsupported. Keeping only
//     the last one, as this used to, answered a wider question than was asked.
//
// The negated forms are composed by the caller, which needs each dimension's
// polarity from one clause: a dimension that is both repeated and negated is
// reported unsupported here rather than guessed.
func buildDescriptionFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (DescriptionFilterOpts, bool, []ast.Filter, error) {
	b := descriptionOptsBuilder{seen: map[string]int{}, negated: map[string]bool{}}
	kept := make([]ast.Filter, 0, len(filters))
	nothing := false
	for _, f := range filters {
		var err error
		b.dropped = false
		switch x := f.(type) {
		case *ast.TermFilter:
			err = b.addTerm(x)
		case *ast.TypeFilter, *ast.LanguageFilter, *ast.ModuleFilter, *ast.EffectiveTimeFilter:
			err = b.addValues(ctx, x, provider)
		case *ast.ActiveFilter:
			b.addActive(x)
		case *ast.DialectFilter:
			// Handled separately in evaluateFiltered.
		}
		if err != nil {
			return b.opts, false, nil, err
		}
		// Resolved to the end, even after a clause that matches nothing, so a
		// later clause's error surfaces as it does on the row path.
		nothing = nothing || b.nothing
		b.nothing = false
		if !b.dropped {
			kept = append(kept, f)
		}
	}
	if nothing {
		return b.opts, true, nil, nil
	}
	return b.opts, false, kept, nil
}

// descriptionOptsBuilder folds the clauses of one block into one
// DescriptionFilterOpts; see buildDescriptionFilterOpts for the rules.
type descriptionOptsBuilder struct {
	opts    DescriptionFilterOpts
	seen    map[string]int
	negated map[string]bool
	nothing bool // a clause, or two together, can hold on no description
	dropped bool // the clause just added constrains nothing and is left out
}

// note counts a clause of kind and rejects a dimension that is both repeated
// and negated, whose polarities one call cannot carry.
func (b *descriptionOptsBuilder) note(kind string, neg bool) error {
	b.seen[kind]++
	b.negated[kind] = b.negated[kind] || neg
	if b.seen[kind] > 1 && b.negated[kind] {
		return fmt.Errorf("%w: a negated %s clause repeated in one description filter needs both polarities on one call", ErrUnsupportedFeature, kind)
	}
	return nil
}

func (b *descriptionOptsBuilder) addTerm(x *ast.TermFilter) error {
	if err := b.note("term", x.Op == "!="); err != nil {
		return err
	}
	if b.seen["term"] > 1 {
		return fmt.Errorf("%w: two term clauses in one description filter must hold on the same description, and DescriptionFilterOpts.Term carries one term", ErrUnsupportedFeature)
	}
	// matchDescriptionFilters has rejected a negated set before building
	// (unsupportedDescriptionFilter), and descriptionCalls fans out a positive
	// one, so the first term is the one this call carries.
	if len(x.Terms) > 0 {
		b.opts.Term = x.Terms[0].Text
		if mt := x.Terms[0].MatchType; mt != "" {
			b.opts.MatchType = mt
		}
	}
	return nil
}

// addValues folds a type, language, module or effectiveTime clause into the
// Opts, from the same resolution the row path uses (resolveClauseValues).
func (b *descriptionOptsBuilder) addValues(ctx context.Context, clause ast.Filter, provider DataProvider) error {
	cv, err := resolveClauseValues(ctx, clause, provider)
	switch {
	case err != nil:
		return err
	case cv.dropped:
		b.dropped = true
		return nil
	case cv.nothing:
		b.nothing = true
		return nil
	}
	switch x := clause.(type) {
	case *ast.TypeFilter:
		if err := b.note("type", cv.negate); err != nil {
			return err
		}
		b.opts.TypeIDs = conjoin(b.opts.TypeIDs, cv.values, b.seen["type"] > 1)
		b.nothing = len(b.opts.TypeIDs) == 0
	case *ast.LanguageFilter:
		if err := b.note("language", cv.negate); err != nil {
			return err
		}
		b.opts.Languages = conjoin(b.opts.Languages, cv.values, b.seen["language"] > 1)
		b.nothing = len(b.opts.Languages) == 0
	case *ast.ModuleFilter:
		if cv.negate {
			return fmt.Errorf("%w: a negated moduleId in a description filter is per description row, and DescriptionFilterOpts has no negated module", ErrUnsupportedFeature)
		}
		if err := b.note("moduleId", false); err != nil {
			return err
		}
		b.opts.ModuleIDs = conjoin(b.opts.ModuleIDs, cv.values, b.seen["moduleId"] > 1)
		b.nothing = len(b.opts.ModuleIDs) == 0
	case *ast.EffectiveTimeFilter:
		if err := b.note("effectiveTime", cv.negate); err != nil {
			return err
		}
		switch {
		case b.seen["effectiveTime"] > 1:
			// Two comparisons on one row — a range — need two fields.
			return fmt.Errorf("%w: two effectiveTime clauses in one description filter must hold on the same description, and DescriptionFilterOpts.EffectiveTime carries one comparison", ErrUnsupportedFeature)
		case len(cv.values) > 1 && cv.negate:
			// Every other operator reads a set as any-of, which
			// descriptionCalls fans out exactly; "!=" needs one row to fail
			// every value, which one call per value cannot decide.
			return fmt.Errorf("%w: an effectiveTime set with \"!=\" needs every value on one call, and DescriptionFilterOpts.EffectiveTime carries one", ErrUnsupportedFeature)
		}
		b.opts.EffectiveTime, b.opts.EffectiveTimeOp = cv.values[0], x.Op
	}
	return nil
}

func (b *descriptionOptsBuilder) addActive(x *ast.ActiveFilter) {
	if x.Any {
		// Either value constrains nothing; the block still asks for a
		// description (see anyDescription).
		b.dropped = true
		return
	}
	v := x.Value
	if b.opts.Active != nil && *b.opts.Active != v {
		b.nothing = true
		return
	}
	b.opts.Active = &v
}

// conjoin folds one clause's values into those of the clauses before it. The
// first clause sets the values; a repeated one keeps only the values both
// allow, because every clause of a block holds on the same description.
func conjoin(have, clause []string, repeated bool) []string {
	if !repeated {
		return clause
	}
	keep := make(map[string]bool, len(clause))
	for _, v := range clause {
		keep[v] = true
	}
	out := have[:0:0]
	for _, v := range have {
		if keep[v] {
			out = append(out, v)
		}
	}
	return out
}

// buildDialectFilterOpts converts an ast.DialectFilter into DialectFilterOpts
// by evaluating each dialect entry's dialect and acceptability expressions.
func buildDialectFilterOpts(ctx context.Context, df *ast.DialectFilter, provider DataProvider) (DialectFilterOpts, error) {
	opts := DialectFilterOpts{Negate: df.Op == "!="}
	if len(df.Dialects) == 0 && len(df.Aliases) == 0 {
		// Neither form carried an entry, so there is nothing to filter by.
		// Matching an empty entry list instead would return the empty set for
		// every such expression, which is what happened before this node was
		// populated at all. Both paths that answer a block come through here.
		return opts, fmt.Errorf("%w: dialect filter names no dialect", ErrUnsupportedFeature)
	}
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
		return providerError(ctx, "ResolveDialectAliases", err)
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
func literalText(e ast.Expression) (text string, ok bool, err error) {
	switch v := e.(type) {
	case *ast.StringValue:
		return v.Value, true, nil
	case *ast.IntegerValue, *ast.DecimalValue:
		// Canonical, so a member field compared by text sees "1.5" for #1.50;
		// validated and read like an attribute comparison's literal.
		n, err := numericLiteral(v)
		if err != nil {
			return "", false, err
		}
		return canonicalNumber(n), true, nil
	case *ast.BooleanValue:
		return strconv.FormatBool(v.Value), true, nil
	}
	return "", false, nil
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
		numeric     number
		haveString  bool
		strVal      string
		haveBool    bool
		boolVal     bool
	)
	switch v := attr.Value.(type) {
	case *ast.IntegerValue, *ast.DecimalValue:
		haveNumeric = true
		var err error
		if numeric, err = numericLiteral(v); err != nil {
			return nil, err
		}
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
					stored, ok := parseNumber(cv.Value)
					if !ok {
						continue
					}
					if compareNumbers(stored, attr.Op, numeric) {
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

// number is an exact numeric value, compared as cheaply as its form allows:
//
//   - an integer that fits int64, compared as one;
//   - a value of at most 15 significant digits, compared as its float64. A
//     correctly rounded conversion preserves order, and float64 holds 15.95
//     decimal digits, so two such values convert to floats that compare
//     exactly as the decimals do;
//   - two values of different sign or decimal magnitude, compared by those,
//     which the scan reads off the text without building either number;
//   - anything else, as an exact rational, built only when a comparison needs
//     it.
//
// Nearly every concrete value is one of the first two, which allocate nothing.
type number struct {
	i64     int64
	isInt64 bool
	f64     float64
	isFloat bool // f64 compares exactly (see above)

	// sign is -1, 0 or 1, and mag is floor(log10 |value|) for a non-zero
	// value, both read off the text.
	sign int
	mag  int64

	text string   // for the rational, built on demand
	r    *big.Rat // the rational, when already built
}

// rat returns the exact value, or nil for a text big.Rat cannot hold — an
// exponent beyond int64, which only a stored value can have.
func (n number) rat() *big.Rat {
	switch {
	case n.r != nil:
		return n.r
	case n.isInt64:
		return new(big.Rat).SetInt64(n.i64)
	}
	if r, ok := new(big.Rat).SetString(n.text); ok {
		return r
	}
	return nil
}

// minNormalFloat64 is the smallest positive normal float64.
const minNormalFloat64 = 2.2250738585072014e-308

// maxNumberDigits bounds the digits and the exponent of a LITERAL, which comes
// from an expression a caller supplies: an exact rational for 1e1000000 or a
// 200000-digit fraction costs milliseconds to build and to compare against,
// once per stored value. No concrete value comes near it.
const maxNumberDigits = 4000

// maxStoredDigits is the bound for a STORED value, terminology data rather
// than caller input, beyond which it is not built as a rational up front. It
// is still compared exactly: by sign and magnitude when those differ, which
// they do from every literal the literal bound admits whenever the stored
// exponent is beyond it, and as a rational otherwise.
const maxStoredDigits = 100000

// exponent is what scanExponent found.
type exponent int

const (
	noExponent       exponent = iota
	exponentOK                // present and within the bound
	exponentTooLarge          // well formed, beyond the bound
	exponentBad               // malformed
)

// scanExponent reads an optional exponent of parseNumber's form starting at i
// and returns where it ends, what it found and its value, saturated for one
// too long to hold.
func scanExponent(s string, i, limit int) (end int, state exponent, value int64) {
	if i >= len(s) || (s[i] != 'e' && s[i] != 'E') {
		return i, noExponent, 0
	}
	i++
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return i, exponentBad, 0
	}
	v, err := strconv.ParseInt(s[start:i], 10, 64)
	if err != nil || v > math.MaxInt64/4 {
		v = math.MaxInt64 / 4 // beyond any digit count; sums below stay in range
	}
	if neg {
		v = -v
	}
	state = exponentOK
	if v > int64(limit) || v < -int64(limit) {
		state = exponentTooLarge
	}
	return i, state, v
}

// mantissa is what scanMantissa found: the digits before the exponent.
type mantissa struct {
	end         int  // where the mantissa ends
	digits      int  // integer and fraction digits
	significant int  // digits from the first non-zero one
	fraction    bool // a "." was present
	negative    bool
	// lead is the decimal position of the first non-zero digit: 0 for the
	// units, 1 for the tens, -1 for the tenths; valid when significant > 0.
	lead int64
}

// scanMantissa reads the sign, integer part and optional fraction of s.
func scanMantissa(s string) mantissa {
	var m mantissa
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		m.negative = s[i] == '-'
		i++
	}
	intDigits, fracSeen := 0, 0
	for ; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.' && !m.fraction:
			m.fraction = true
			continue
		case c < '0' || c > '9':
			m.end = i
			return m.withLead(intDigits)
		}
		m.digits++
		if m.fraction {
			fracSeen++
		} else {
			intDigits++
		}
		if c != '0' && m.significant == 0 {
			if m.fraction {
				m.lead = -int64(fracSeen)
			} else {
				m.lead = int64(intDigits) // fixed up once the integer part is known
			}
		}
		if c != '0' || m.significant > 0 {
			m.significant++
		}
	}
	m.end = i
	return m.withLead(intDigits)
}

// withLead turns a lead counted from the left of the integer part into a
// position counted from the units.
func (m mantissa) withLead(intDigits int) mantissa {
	if m.significant > 0 && m.lead > 0 {
		m.lead = int64(intDigits) - m.lead
	}
	return m
}

// parseNumber reads a STORED decimal number — sign, digits, an optional
// fraction and an optional exponent, the forms RF2 writes concrete values in —
// or reports that s is not one. Converting through float64 alone, as this used
// to, made 2^53+1 equal to 2^53 (H4 in docs/analisis-adversarial-2026-10-12.md).
// It is a hand-written scan rather than big.Rat.SetString, which also takes
// fractions ("1/2") and hexadecimal ("0x10"), and rather than a regexp,
// because it runs once per stored value. A value beyond maxStoredDigits is
// kept by sign, magnitude and text, and compared exactly all the same.
func parseNumber(s string) (number, bool) {
	n, ok, beyond := scanNumber(s, maxStoredDigits)
	return n, ok || beyond
}

// parseNumberExact reads a LITERAL: like parseNumber, but a number beyond the
// bound given is refused.
func parseNumberExact(s string, limit int) (number, bool) {
	n, ok, _ := scanNumber(s, limit)
	return n, ok
}

// scanNumber reads a decimal number. The beyond result reports one that is well
// formed but over the bound, which comes back with its sign, magnitude and text
// only.
func scanNumber(s string, limit int) (n number, ok, beyond bool) {
	m := scanMantissa(s)
	if m.digits == 0 {
		return number{}, false, false
	}
	end, exp, expValue := scanExponent(s, m.end, limit)
	if end != len(s) || exp == exponentBad {
		return number{}, false, false
	}
	n = number{text: s}
	if m.significant > 0 {
		n.sign, n.mag = 1, m.lead+expValue
		if m.negative {
			n.sign = -1
		}
	}
	if exp == exponentTooLarge || m.digits > limit {
		return n, false, true
	}
	if !m.fraction && exp == noExponent {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			n.i64, n.isInt64 = v, true
			if v >= -1<<53 && v <= 1<<53 {
				n.f64, n.isFloat = float64(v), true // exact, no second parse
				return n, true, false
			}
		}
	}
	if m.significant <= 15 {
		// Below the smallest normal float64 the 15-digit guarantee no longer
		// holds, and a non-zero value may underflow to zero.
		if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsInf(f, 0) && (f == 0 && n.sign == 0 || math.Abs(f) >= minNormalFloat64) {
			n.f64, n.isFloat = f, true
		}
	}
	return n, true, false
}

// numberText returns a numeric literal's text. Text, as written, wins: it is
// the exact literal, and a hand-built node may set only Text. The one
// exception is a node whose Value was edited after parsing — a non-zero Value
// the Text no longer says — where the new Value is meant.
func numberText(e ast.Expression) (string, error) {
	switch v := e.(type) {
	case *ast.IntegerValue:
		if v.Text == "" {
			return strconv.Itoa(v.Value), nil
		}
		n, err := strconv.Atoi(v.Text)
		if agrees := err == nil && n == v.Value; v.Value != 0 && !agrees {
			return strconv.Itoa(v.Value), nil
		}
		return v.Text, nil
	case *ast.DecimalValue:
		if v.Text != "" {
			if f, err := strconv.ParseFloat(v.Text, 64); v.Value == 0 || err == nil && f == v.Value {
				return v.Text, nil
			}
		}
		if math.IsNaN(v.Value) || math.IsInf(v.Value, 0) {
			return "", fmt.Errorf("%w: numeric literal %v is not a finite number", ErrUnsupportedFeature, v.Value)
		}
		return strconv.FormatFloat(v.Value, 'f', -1, 64), nil
	}
	return "", fmt.Errorf("numberText: %T is not a numeric literal", e)
}

// numericLiteral returns the exact value of an integer or decimal literal.
func numericLiteral(e ast.Expression) (number, error) {
	text, err := numberText(e)
	if err != nil {
		return number{}, err
	}
	// Exact or refused: a literal comes from the expression, so one beyond
	// the bound is the caller's to shorten, not an approximation to accept.
	n, ok := parseNumberExact(text, maxNumberDigits)
	if !ok {
		return number{}, fmt.Errorf("%w: numeric literal %q is not a decimal number of at most %d digits", ErrUnsupportedFeature, text, maxNumberDigits)
	}
	n.r = n.rat() // built once for the clause, not once per stored value
	return n, nil
}

// canonicalNumber writes an exact value in canonical decimal form, for a
// comparison by TEXT: no "+", no trailing zeros, and zero without a sign.
// `#+1`, `#1.0` and `#-0` reach a member field as "1", "1" and "0", the form a
// stored value is written in. Every value here is a finite decimal, so its
// denominator is 2^a·5^b and max(a, b) fraction digits write it exactly.
func canonicalNumber(n number) string {
	r := n.rat()
	if r == nil {
		return n.text
	}
	if r.IsInt() {
		return r.Num().String()
	}
	d := new(big.Int).Set(r.Denom())
	two, five, zero := big.NewInt(2), big.NewInt(5), new(big.Int)
	twos, fives := 0, 0
	for mod := new(big.Int); mod.Mod(d, two).Cmp(zero) == 0; twos++ {
		d.Quo(d, two)
	}
	for mod := new(big.Int); mod.Mod(d, five).Cmp(zero) == 0; fives++ {
		d.Quo(d, five)
	}
	return strings.TrimRight(r.FloatString(max(twos, fives)), "0")
}

func cmpOrdered[T int64 | float64 | int](a, b T) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// compareNumbers applies the given operator to a (stored) and b (expected),
// exactly. Integers and decimals compare by value, as they always have.
func compareNumbers(a number, op string, b number) bool {
	var c int
	switch {
	case a.isInt64 && b.isInt64:
		c = cmpOrdered(a.i64, b.i64)
	case a.isFloat && b.isFloat:
		c = cmpOrdered(a.f64, b.f64)
	case a.sign != b.sign:
		c = cmpOrdered(a.sign, b.sign)
	case a.sign == 0:
		// Both zero, whatever their exponents say: 0e99999999999999999999 is 0.
	case a.mag != b.mag:
		// Same sign, different decimal magnitude: the larger magnitude is the
		// larger value for positives and the smaller for negatives.
		c = a.sign * cmpOrdered(a.mag, b.mag)
	default:
		ra, rb := a.rat(), b.rat()
		if ra == nil || rb == nil {
			return false // not a number big.Rat can hold; it matches nothing
		}
		c = ra.Cmp(rb)
	}
	switch op {
	case "=":
		return c == 0
	case "!=":
		return c != 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	case ">":
		return c > 0
	case ">=":
		return c >= 0
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
			iterErr = providerError(ctx, fmt.Sprintf("Ancestors(%s)", id), err)
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
			iterErr = providerError(ctx, fmt.Sprintf("Descendants(%s)", id), err)
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

// descriptionIDFilterOf returns the description id clause of a filter list, or
// nil. The grammar allows at most one per constraint.
func descriptionIDFilterOf(filters []ast.Filter) *ast.DescriptionIDFilter {
	for _, f := range filters {
		if x, ok := f.(*ast.DescriptionIDFilter); ok && len(x.IDs) > 0 {
			return x
		}
	}
	return nil
}

// evaluateMemberOf answers the memberOf operator, with or without a field
// projection.
//
// The filters argument carries the `{{ M ... }}` clauses of the surrounding constraint, when
// there is one. They belong HERE rather than being applied to the result: they
// restrict which member ROWS are projected, and for a projection the result is a
// different column of those rows. `^ [targetComponentId] X {{ M
// referencedComponentId = Y }}` filters on one field and returns another, so
// filtering the returned values would compare the wrong column.
func evaluateMemberOf(ctx context.Context, e *ast.MemberOf, filters []MemberFilterOpts, provider DataProvider) (Set, error) {
	refsetIDSet, err := Evaluate(ctx, e.Operand, provider)
	if err != nil {
		return nil, fmt.Errorf("evaluating *ast.MemberOf: %w", err)
	}
	refsetIDs := toIDSlice(refsetIDSet)

	if err := checkProjectionSupport(e); err != nil {
		return nil, err
	}

	if len(e.Fields) == 1 {
		projector, ok := provider.(RefsetFieldProjector)
		if !ok {
			return nil, fmt.Errorf("%w: the memberOf field projection ^[%s] requires a provider implementing ecl.RefsetFieldProjector; the members and the projected field are different columns of the same row, so it cannot be composed from the member set",
				ErrUnsupportedFeature, e.Fields[0])
		}
		// `^ R` means the active rows, projected or not. A filter that already
		// names the active column decides for itself; evaluateFiltered adds the
		// default to the others, and a bare projection gets it here, so adding a
		// filter that matches every row cannot shrink the result.
		filters, _ = rowActiveDefault(filters, provider)
		projected, err := projector.ProjectRefsetField(ctx, RefsetProjectionOpts{
			RefsetIDs: refsetIDs,
			Field:     e.Fields[0],
			Filters:   filters,
		})
		if err != nil {
			// A provider refusing a FIELD wraps ErrUnsupportedFeature, which
			// providerError passes through without ErrProvider.
			return nil, providerError(ctx, "ProjectRefsetField", err)
		}
		return nonNil(projected), nil
	}

	members, err := provider.RefsetMembers(ctx, refsetIDs)
	if err != nil {
		return nil, providerError(ctx, "RefsetMembers", err)
	}
	return nonNil(members), nil
}

// checkProjectionSupport reports the projection forms that cannot be a Set.
//
// Both are tabular rather than merely unimplemented: `^[a,b]` asks for two
// columns and `^[*]` for every column, and a Set is one column of concept ids.
// Picking one of the fields, or unioning them, would answer a different question
// — which is what `^[*]` used to do, silently, by leaving no trace in the AST at
// all and being evaluated as a plain `^`.
func checkProjectionSupport(e *ast.MemberOf) error {
	switch {
	case e.AllFields:
		return fmt.Errorf("%w: the memberOf wildcard projection ^[*] asks for every member field, which a Set of concept ids cannot carry; name a single field, or use a {{ M ... }} member filter",
			ErrUnsupportedFeature)
	case len(e.Fields) > 1:
		return fmt.Errorf("%w: the memberOf projection ^[%s] names %d fields, which is a table rather than a set of concept ids; name a single field",
			ErrUnsupportedFeature, strings.Join(e.Fields, ","), len(e.Fields))
	}
	return nil
}

// projectedMemberOf returns the operand when it is a memberOf carrying a field
// projection, and nil otherwise. Only a projection needs the special routing; a
// plain `^` composes with a member filter the ordinary way.
func projectedMemberOf(operand ast.Expression) *ast.MemberOf {
	mo := memberOfOperand(operand)
	if mo == nil {
		return nil
	}
	if len(mo.Fields) == 0 && !mo.AllFields {
		return nil
	}
	return mo
}

// memberFilterOptsOf converts every member filter clause of a constraint.
// The second result reports a clause that can match no row, so the caller can
// answer the empty set without asking the provider: sending it would rely on
// every provider reading an empty ValueSet the same way, which the contract
// does not say.
func memberFilterOptsOf(ctx context.Context, filters []ast.Filter, provider DataProvider) ([]MemberFilterOpts, bool, error) {
	out := make([]MemberFilterOpts, 0, len(filters))
	_, answersColumns := provider.(MemberRowColumnsProvider)
	actives := map[bool]bool{}
	contradicts := false
	for _, f := range filters {
		var (
			opts         MemberFilterOpts
			skip, noRows bool
			err          error
		)
		switch x := f.(type) {
		case *ast.MemberFieldFilter:
			opts, err = oneMemberFilterOpts(ctx, x, provider)
		case *ast.ActiveFilter, *ast.ModuleFilter, *ast.EffectiveTimeFilter:
			if !answersColumns {
				return nil, false, fmt.Errorf("%w: active, moduleId and effectiveTime in a member filter are columns of the member row, which need a provider implementing ecl.MemberRowColumnsProvider", ErrUnsupportedFeature)
			}
			if a, ok := x.(*ast.ActiveFilter); ok && !a.Any {
				// Opposite values in one block: no row holds both. Separate
				// blocks are separate nodes, which may hold on different rows.
				// Noted, not returned: the clauses after it must still be
				// validated, or a refusal would depend on clause order.
				contradicts = contradicts || actives[!a.Value]
				actives[a.Value] = true
			}
			opts, skip, noRows, err = memberColumnOpts(ctx, f, provider)
		default:
			continue
		}
		switch {
		case err != nil:
			return nil, false, err
		case noRows:
			// Noted, not returned, for the same reason as a contradiction.
			contradicts = true
		case !skip:
			out = append(out, opts)
		}
	}
	return out, contradicts, nil
}

// memberColumnOpts sends a member filter on one of the columns every member row
// has as a field comparison, under the RF2 column name; see MemberFilterOpts.
// The skip result reports a clause that constrains nothing (excluding a module
// set that names no module), and noRows one that matches no row (requiring one).
func memberColumnOpts(ctx context.Context, f ast.Filter, provider DataProvider) (opts MemberFilterOpts, skip, noRows bool, err error) {
	switch x := f.(type) {
	case *ast.ActiveFilter:
		values := []string{"0"}
		switch {
		case x.Any:
			// Either flag, so the rows decide rather than `^`'s active-only
			// default (see namesActive in evaluateFiltered).
			values = []string{"1", "0"}
		case x.Value:
			values = []string{"1"}
		}
		return MemberFilterOpts{FieldName: MemberColumnActive, Op: "=", ValueSet: NewSetFromSlice(values)}, false, false, nil
	case *ast.ModuleFilter:
		if !moduleConstrains(x) {
			return opts, true, false, nil // names no module: no constraint
		}
		ids, namesNothing, err := resolveModuleIDs(ctx, x, provider)
		if err != nil {
			return opts, false, false, err
		}
		op := x.Op
		if op == "" {
			op = "=" // a hand-built filter may leave it empty
		}
		if namesNothing {
			return opts, op == "!=", op != "!=", nil
		}
		return MemberFilterOpts{FieldName: MemberColumnModuleID, Op: op, ValueSet: NewSetFromSlice(ids)}, false, false, nil
	case *ast.EffectiveTimeFilter:
		et := x
		if et.Op != "=" && et.Op != "!=" && et.Op != "" {
			return opts, false, false, fmt.Errorf("%w: effectiveTime %s in a member filter needs an ordered comparison, and MemberFilterOpts compares by equality", ErrUnsupportedFeature, et.Op)
		}
		values := effectiveTimeValues(et)
		if len(values) == 0 {
			return opts, true, false, nil
		}
		op := et.Op
		if op == "" {
			op = "="
		}
		return MemberFilterOpts{FieldName: MemberColumnEffectiveTime, Op: op, ValueSet: NewSetFromSlice(values)}, false, false, nil
	default:
		return opts, false, false, fmt.Errorf("memberColumnOpts: %T is not a member row column", f)
	}
}

// resolveModuleIDs resolves every module a module filter names. The
// namesNothing result reports a set whose operands all resolved to no concept, which each family
// answers in its own terms; an empty slice alone would read as "no filter".
func resolveModuleIDs(ctx context.Context, x *ast.ModuleFilter, provider DataProvider) (ids []string, namesNothing bool, err error) {
	mods := moduleOperands(x)
	for _, mod := range mods {
		got, resolved, err := resolveFilterIDs(ctx, mod, provider)
		if err != nil {
			return nil, false, fmt.Errorf("evaluating module filter: %w", err)
		}
		if resolved {
			ids = append(ids, got...)
		}
	}
	return ids, len(mods) > 0 && len(ids) == 0, nil
}

// moduleConstrains reports whether a module filter names any module. One that
// names none — only a hand-built tree can — constrains nothing in any family.
func moduleConstrains(x *ast.ModuleFilter) bool {
	return len(moduleOperands(x)) > 0
}

// moduleOperands returns the modules a module filter names, falling back to
// the deprecated single-module field a hand-built tree may set alone.
func moduleOperands(x *ast.ModuleFilter) []ast.Expression {
	if len(x.Modules) == 0 && x.Module != nil { //nolint:staticcheck // hand-built trees may set only the deprecated field
		return []ast.Expression{x.Module} //nolint:staticcheck // as above
	}
	return x.Modules
}

// effectiveTimeValues returns the times an effectiveTime filter names, falling
// back to the deprecated single-value field a hand-built tree may set alone. A
// filter naming none — only a hand-built tree can — constrains nothing in every
// family.
func effectiveTimeValues(x *ast.EffectiveTimeFilter) []string {
	values := x.Values
	if len(values) == 0 && x.Value != "" { //nolint:staticcheck // hand-built trees may set only the deprecated field
		values = []string{x.Value} //nolint:staticcheck // as above
	}
	return values
}

// oneMemberFilterOpts converts a single member filter clause.
//
// A member field may hold a literal (a map target, an order, a flag) rather than
// a concept reference, so a literal must NOT go through Evaluate: there is
// nothing to resolve, and the type switch there rejects it with "unsupported AST
// node type: *ast.StringValue". That made every `{{ M mapTarget = "..." }}` fail
// at runtime even though the README listed memberField as supported.
func oneMemberFilterOpts(ctx context.Context, field *ast.MemberFieldFilter, provider DataProvider) (MemberFilterOpts, error) {
	var valueSet Set
	if field.Value != nil {
		lit, ok, err := literalText(field.Value)
		if err != nil {
			// A literal that is not a finite number, refused as such rather
			// than falling through to Evaluate and a generic error.
			return MemberFilterOpts{}, err
		}
		if ok {
			valueSet = NewSetFromSlice([]string{lit})
		} else {
			resolved, err := Evaluate(ctx, field.Value, provider)
			if err != nil {
				return MemberFilterOpts{}, fmt.Errorf("evaluating member filter value: %w", err)
			}
			valueSet = resolved
		}
	}
	return MemberFilterOpts{
		FieldName: field.FieldName,
		Op:        field.Op,
		ValueSet:  valueSet,
	}, nil
}

// checkMemberRowSupport refuses a member block that needs several clauses on
// one row from a provider that cannot take them; see matchMemberFilters.
func checkMemberRowSupport(clauses []MemberFilterOpts, provider DataProvider) error {
	if _, ok := provider.(RefsetFieldProjector); !ok && len(clauses) > 1 {
		return errOneRowNeedsProjector(len(clauses))
	}
	return nil
}

func errOneRowNeedsProjector(clauses int) error {
	return fmt.Errorf("%w: a member filter block with %d clauses requires a provider implementing ecl.RefsetFieldProjector; the clauses of a block must hold on ONE member row, and RefsetMembersFiltered takes one clause at a time, so composing them here would accept a concept whose different rows satisfy different clauses",
		ErrUnsupportedFeature, clauses)
}

// matchMemberFilters resolves the `{{ M ... }}` clauses of one constraint to the
// members that satisfy them.
//
// # Why more than one clause needs a capability
//
// RefsetMembersFiltered takes ONE MemberFilterOpts, so several clauses can only
// be composed by calling it once each and intersecting the concept sets. That is
// not what the expression asks. A reference set can hold several rows for the
// same member — a complex map has one per map group — and intersecting per clause
// accepts a concept whose FIRST row satisfies one clause and whose SECOND
// satisfies another, when the expression asks for a single row satisfying all of
// them.
//
// The specification's own examples are exactly this shape:
//
//	^ 447562003 |ICD-10 complex map| {{ M mapGroup = #2, mapPriority = #1, mapTarget = "J45.9" }}
//
// The three clauses are there to pick ONE row. Answered per clause, the query
// returns every concept that has a row in group 2, and a row at priority 1, and a
// row mapping to J45.9 — which is a wider set, and silently so.
//
// RefsetFieldProjector takes every clause on one call, so a provider implementing
// it can answer exactly; projecting referencedComponentId returns the members
// themselves. Without it, the multi-clause form is reported rather than
// approximated. A single clause has no such problem and still uses
// RefsetMembersFiltered, so nothing changes for it.
func matchMemberFilters(ctx context.Context, refsetIDs []string, clauses []MemberFilterOpts, provider DataProvider) (Set, error) {
	switch len(clauses) {
	case 0:
		return nil, fmt.Errorf("%w: member filter names no field", ErrUnsupportedFeature)
	case 1:
		filtered, err := provider.RefsetMembersFiltered(ctx, refsetIDs, clauses[0])
		if err != nil {
			return nil, providerError(ctx, "RefsetMembersFiltered", err)
		}
		return nonNil(filtered), nil
	}

	projector, ok := provider.(RefsetFieldProjector)
	if !ok {
		return nil, errOneRowNeedsProjector(len(clauses))
	}
	matched, err := projector.ProjectRefsetField(ctx, RefsetProjectionOpts{
		RefsetIDs: refsetIDs,
		Field:     "referencedComponentId",
		Filters:   clauses,
	})
	if err != nil {
		return nil, providerError(ctx, "ProjectRefsetField", err)
	}
	return nonNil(matched), nil
}
