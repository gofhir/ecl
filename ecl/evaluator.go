// Package ecl parses and evaluates SNOMED CT Expression Constraint Language (ECL) expressions
// against a pluggable DataProvider that supplies the underlying terminology.
package ecl

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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
			return nil, fmt.Errorf("MemberOf field projection ^[%v] is not supported by Evaluate; use a member filter constraint instead", e.Fields)
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

	// A node must never hold both, or the parenthesised scope was lost while
	// parsing and the disjunction below cannot be composed correctly.
	if len(ref.Conjunction) > 0 && len(ref.Disjunction) > 0 {
		return nil, fmt.Errorf("refinement has both a conjunction and a disjunction in one node: the parenthesised scope was lost while parsing")
	}

	result := focus

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
		acc := result
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
	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		rels, err := conceptRelationships(ctx, id, provider)
		if err != nil {
			iterErr = err
			return false
		}
		if clauseSatisfied(rels, clause) {
			out.m[id] = struct{}{}
		}
		return true
	})
	if iterErr != nil {
		return nil, iterErr
	}
	return out, nil
}

// conceptRelationships returns every relationship of a concept, with the group
// structure flattened. Ungrouped attribute clauses match a relationship in any
// group, so they are evaluated against this flat view.
func conceptRelationships(ctx context.Context, conceptID string, provider DataProvider) ([]Relationship, error) {
	groups, err := provider.PropertiesByGroup(ctx, conceptID)
	if err != nil {
		return nil, fmt.Errorf("PropertiesByGroup(%s): %w", conceptID, err)
	}
	var out []Relationship
	for _, rels := range groups {
		out = append(out, rels...)
	}
	return out, nil
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
	// The reverse path below works on the flattened leaves, so it still treats a
	// group's clauses as a conjunction: `{ R a = x OR b = y }` is evaluated as
	// AND. Known limitation, tracked with the rest of the reverse-path work.
	clauses := tree.leaves()

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
		// matchingGroups is how many relationship groups satisfy the clause
		// tree. Counting is what makes group cardinality work at all: the code
		// used to stop at the first match, so [0..0], [1..1] and [2..*] were
		// indistinguishable from "at least one" and [0..0] returned exactly the
		// inverse of the requested set.
		matchingGroups := 0
		if !hasReverse {
			// Fast path: all forward clauses — check groups directly.
			groups, err := provider.PropertiesByGroup(ctx, id)
			if err != nil {
				iterErr = fmt.Errorf("PropertiesByGroup(%s): %w", id, err)
				return false
			}
			for gnum, rels := range groups {
				// Group 0 is "ungrouped" per the PropertiesByGroup contract, so
				// it is not a relationship group and must not be counted as one.
				// Ungrouped attributes are matched by filterByAttribute instead.
				if gnum == 0 {
					continue
				}
				if groupSatisfiesSet(rels, tree) {
					matchingGroups++
				}
			}
		} else {
			// Slow path: reverse clauses require checking source concepts.
			//
			// This returns 1 or 0, not a real count: it walks the groups of the
			// SOURCE concepts, so any count derived from it would be counting
			// someone else's groups. Group cardinality combined with a reverse
			// clause is therefore not supported; see the known limitations.
			matched, err := conceptMatchesGroupWithReverse(ctx, id, clauses, provider)
			if err != nil {
				iterErr = err
				return false
			}
			if matched {
				matchingGroups = 1
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
			switch {
			case c.reverse:
				// Reverse: the relationship must target our focus concept.
				if r.TargetID == reverseTargetID {
					count++
				}
			case c.isConcrete:
				if r.ConcreteValue != nil && matchConcreteValue(r.ConcreteValue, c) {
					count++
				}
			default:
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
		// A negated description filter cannot be expressed as a set operation.
		// `{{ D type != fsn }}` means "has a description whose type is not FSN",
		// which is a per-DESCRIPTION-ROW negation. Subtracting the concepts that
		// have an FSN also removes concepts that have both an FSN and a synonym,
		// so the set-level Minus used here before produced wrong results
		// silently. Expressing it properly needs negation fields on
		// DescriptionFilterOpts, i.e. a provider contract change.
		//
		// Until then, fail loudly: a classifiable error beats a wrong set. The
		// same choice is already made for ^[field] projections above.
		if kind, negated := negatedDescriptionFilter(descFilters); negated {
			return nil, fmt.Errorf("%w: negated description filter (%s) requires row-level negation in the DataProvider", ErrUnsupportedFeature, kind)
		}
		opts, err := buildDescriptionFilterOpts(ctx, descFilters, provider)
		if err != nil {
			return nil, err
		}
		matches, err := provider.MatchDescription(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("MatchDescription: %w", err)
		}
		if matches == nil {
			matches = NewSet()
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
	// Negated dialect filters are rejected above with the rest of the
	// description family, so only the positive form reaches here. That also
	// resolves a double-negation hazard: DialectFilterOpts.Negate already asks
	// the provider to negate, and this loop used to apply Minus on top of it.
	// The provider is the single owner of the negation.
	for _, f := range descFilters {
		df, ok := f.(*ast.DialectFilter)
		if !ok {
			continue
		}
		if len(df.Dialects) == 0 {
			// Reached by the alias form (`dialect = en-gb`): mapping an alias to
			// the SCTID of a language reference set is terminology data, and only
			// the international English aliases are universal — national dialects
			// use namespace-specific refset IDs. Use the dialectId form
			// (`dialectId = 900000000000508004`) meanwhile.
			//
			// Intersecting with an empty match instead would return the empty set
			// for every dialect expression without a word, which is what happened
			// before this node was populated at all.
			return nil, fmt.Errorf("%w: dialect alias filter needs an alias-to-refset mapping; use the dialectId form", ErrUnsupportedFeature)
		}
		dOpts, err := buildDialectFilterOpts(ctx, df, provider)
		if err != nil {
			return nil, err
		}
		matches, err := provider.MatchDialect(ctx, result, dOpts)
		if err != nil {
			return nil, fmt.Errorf("MatchDialect: %w", err)
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
		opts, err := conceptClauseOpts(ctx, cl.filter, provider)
		if err != nil {
			return nil, err
		}
		matched, err := provider.FilterConcepts(ctx, result, opts)
		if err != nil {
			return nil, fmt.Errorf("FilterConcepts: %w", err)
		}
		if matched == nil {
			matched = NewSet()
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

// conceptClauseOpts builds the ConceptFilterOpts for exactly one clause.
func conceptClauseOpts(ctx context.Context, f ast.Filter, provider DataProvider) (ConceptFilterOpts, error) {
	var opts ConceptFilterOpts
	switch x := f.(type) {
	case *ast.ActiveFilter:
		b := x.Value
		opts.Active = &b
	case *ast.DefinitionStatusFilter:
		ids, err := resolveFilterIDs(ctx, x.Value, provider)
		if err != nil {
			return opts, fmt.Errorf("evaluating definitionStatus filter: %w", err)
		}
		opts.DefinitionStatusIDs = ids
	case *ast.ModuleFilter:
		ids, err := resolveFilterIDs(ctx, x.Module, provider)
		if err != nil {
			return opts, fmt.Errorf("evaluating module filter: %w", err)
		}
		opts.ModuleIDs = ids
	case *ast.EffectiveTimeFilter:
		opts.EffectiveTime = x.Value
		opts.EffectiveTimeOp = x.Op
	}
	return opts, nil
}

// resolveFilterIDs evaluates a filter operand to concept IDs, falling back to
// the literal SCTID when the provider does not enumerate it (well-known
// metadata concepts are often absent from a test provider's ConceptExists).
func resolveFilterIDs(ctx context.Context, e ast.Expression, provider DataProvider) ([]string, error) {
	if e == nil {
		return nil, nil
	}
	ids, err := Evaluate(ctx, e, provider)
	if err != nil {
		return nil, err
	}
	if ids != nil && ids.Len() > 0 {
		return ids.Slice(), nil
	}
	if ref, ok := e.(*ast.ConceptRef); ok {
		return []string{ref.ID}, nil
	}
	return nil, nil
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
		case *ast.DialectFilter:
			if x.Op == "!=" {
				return "dialect", true
			}
		}
	}
	return "", false
}

// buildDescriptionFilterOpts accumulates description filter clauses into a
// single DescriptionFilterOpts. Multiple clauses of the same kind are
// simplified by taking the last one (rarely seen in practice).
//
// Every clause here is positive: evaluateFiltered rejects the negated forms
// before calling this, because they cannot be composed at set level.
func buildDescriptionFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (DescriptionFilterOpts, error) {
	var opts DescriptionFilterOpts
	for _, f := range filters {
		switch x := f.(type) {
		case *ast.TermFilter:
			opts.Term = x.Term
			if x.MatchType != "" {
				opts.MatchType = x.MatchType
			}
		case *ast.TypeFilter:
			// Collect all type SCTIDs from the filter (any-of semantics).
			for _, typeExpr := range x.Types {
				ids, err := resolveFilterIDs(ctx, typeExpr, provider)
				if err != nil {
					return opts, fmt.Errorf("evaluating type filter: %w", err)
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
	return opts, nil
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
		var acceptIDs []string
		if entry.Acceptability != nil {
			ids, err := Evaluate(ctx, entry.Acceptability, provider)
			if err != nil {
				return opts, fmt.Errorf("evaluating acceptability: %w", err)
			}
			if ids != nil && ids.Len() > 0 {
				acceptIDs = ids.Slice()
			} else if ref, ok := entry.Acceptability.(*ast.ConceptRef); ok {
				acceptIDs = []string{ref.ID}
			}
		}
		opts.Dialects = append(opts.Dialects, DialectEntryOpts{
			DialectIDs:       dialectIDs,
			AcceptabilityIDs: acceptIDs,
		})
	}
	return opts, nil
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

	out := newMapSet()
	var iterErr error
	focus.Iter(func(id string) bool {
		// Count the values that satisfy the comparison, so the clause's
		// cardinality can be applied. This used to be a bool with an early
		// break, which made [0..0], [1..1] and [2..*] all behave as "at least
		// one".
		matches := 0
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
		ancestors, err := provider.Ancestors(ctx, []string{id}, false)
		if err != nil {
			iterErr = fmt.Errorf("Ancestors(%s): %w", id, err)
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
		descendants, err := provider.Descendants(ctx, []string{id}, false)
		if err != nil {
			iterErr = fmt.Errorf("Descendants(%s): %w", id, err)
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
