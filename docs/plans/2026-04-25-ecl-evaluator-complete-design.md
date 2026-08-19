# ECL v2.2 Evaluator: Complete Implementation Design

**Date:** 2026-04-25
**Issue:** #1 — Complete ECL v2.2 evaluator: 9 pending features
**Approach:** Bottom-up by layers, single branch

## Context

The ANTLR4 parser handles the full ECL v2.2 grammar. The evaluator (`ecl/evaluator.go`) returns "not yet implemented" errors for 11 features. All 11 will be implemented in this work.

## Layer 1 — Evaluator-only changes (no interface changes)

### 1.1 String/Boolean concrete values

**File:** `evaluator.go`, `filterByConcreteValue` function (~line 782)

Remove the early-return error for `haveString || haveBool`. Add two comparison branches inside the per-concept iteration loop:

- **String:** When `cv.Kind == "string"`, compare `cv.Value` against `strVal` using `=` (exact match) and `!=`. Operators `<`, `<=`, `>`, `>=` return an error for strings.
- **Boolean:** When `cv.Kind == "boolean"`, parse `cv.Value` as `"true"/"false"`, compare against `boolVal` using `=` and `!=`. Ordering operators return an error for booleans.

### 1.2 Negated filter operators (`!=`)

**Files:** `evaluator.go`, `buildDescriptionFilterOpts` and `buildConceptFilterOpts`

For each of the 5 filter types (term, type, language, definitionStatus, module), the `!=` case currently returns an error. Replace with post-process negation in the evaluator:

1. Evaluate the filter as if it were `=` (build opts without negation).
2. Call `MatchDescription` / `FilterConcepts` normally to get the matching set.
3. Subtract from the current result: `result = result.Minus(matchingSet)`.

This avoids adding `Negate` fields to the opts structs and keeps provider implementations simple. The evaluator already has access to the `base`/`result` set at each stage.

Implementation: track which filters are negated in `buildDescriptionFilterOpts` / `buildConceptFilterOpts` by returning a parallel slice of booleans or a wrapper struct. Apply the subtraction in `evaluateFiltered` after each provider call.

### 1.3 Reverse attribute with wildcard (`R attr = *`)

**File:** `evaluator.go`, `filterByAttribute` (~line 358)

When `attr.Reverse && valueIsAny`: use `RelationshipSources(ctx, nil, typeIDs)` where `nil` targetIDs means "any target" (wildcard). Document this contract on the `RelationshipSources` method.

The test provider's `RelationshipSources` already iterates all relationships by source — handling `nil` targetIDs means skipping the `targetIDs.Contains()` check.

### 1.4 Reverse attribute inside groups

**File:** `evaluator.go`, `filterByAttributeGroup` (~line 433)

When `a.Reverse == true` inside a group, the semantics are: the focus concept must be the **target** of a relationship from some source, where the source has a group containing both the reverse-attribute relationship pointing to the focus concept and all other attributes in the group clause.

Implementation:
1. Collect reverse clauses separately from forward clauses.
2. For each concept in focus, find sources via `RelationshipSources(ctx, {conceptID}, typeIDs)`.
3. For each source, call `PropertiesByGroup` and check if any group satisfies: (a) the reverse relationship targets the focus concept, and (b) all other forward clauses match in the same group.

### 1.5 Concrete values inside groups

**File:** `evaluator.go`, `filterByAttributeGroup` (~line 445), `provider.go`

Extend `Relationship` struct with an optional concrete value field:

```go
type Relationship struct {
    TypeID        string
    TargetID      string         // "" when ConcreteValue is set
    GroupNum       int
    ConcreteValue *ConcreteValue // nil for concept-valued relationships
}
```

Extend `attrClause` to carry concrete-value comparison data:

```go
type attrClause struct {
    op          string
    typeIDs     Set
    valueSet    Set
    valueIsAny  bool
    // Concrete value fields (mutually exclusive with valueSet)
    isConcrete  bool
    numericVal  float64
    stringVal   string
    boolVal     bool
    concreteKind string // "numeric", "string", "boolean"
}
```

In `groupSatisfiesClauses`, when `c.isConcrete`, match against `r.ConcreteValue` instead of `r.TargetID`.

### 1.6 Reverse attribute with concrete value

**File:** `evaluator.go`, `filterByConcreteValue` (~line 742)

Remove the early-return error. For reverse concrete values: find sources that point to the focus concept via `RelationshipSources`, then check if those sources have the concrete value for the given attribute type. This reuses `ConcreteValues` provider method.

### 1.7 Cardinality `[min..max]`

**Files:** `evaluator.go`, `filterByAttribute` (~line 314), `filterByAttributeGroup` (~line 426, 436)

Change `conceptMatchesAttribute` to return a **count** (`int`) instead of `bool`:

```go
func conceptMatchesAttribute(...) (int, error)
```

In `filterByAttribute`, after getting the count:
- `[0..0]`: keep concept if count == 0
- `[min..max]`: keep if min <= count <= max (max == -1 means unbounded)

Same logic in `groupSatisfiesClauses` — each clause counts matching relationships within the group and validates against its cardinality.

Remove `isDefaultCardinality` guard at lines 314, 426, 436.

## Layer 2 — DataProvider extensions

### 2.1 AltIdentifier — `ResolveIdentifier`

New method on `DataProvider`:

```go
ResolveIdentifier(ctx context.Context, scheme string, code string) (Set, error)
```

The evaluator's `case *ast.AltIdentifier` calls `provider.ResolveIdentifier(ctx, e.Scheme, e.Code)` directly.

### 2.2 Dialect filter — `MatchDialect`

New method and opts type:

```go
MatchDialect(ctx context.Context, concepts Set, filter DialectFilterOpts) (Set, error)
```

```go
type DialectFilterOpts struct {
    Dialects []DialectEntryOpts
    Negate   bool
}

type DialectEntryOpts struct {
    DialectID       string // SCTID of the dialect language refset
    AcceptabilityID string // optional; "" = any acceptability
}
```

In `buildDescriptionFilterOpts`, the `*ast.DialectFilter` case resolves dialect expressions to SCTIDs and builds `DialectFilterOpts`. The evaluator calls `MatchDialect` and intersects (or subtracts, for `!=`) the result.

### 2.3 Member field projection — `RefsetMembersFiltered`

New method and opts type:

```go
RefsetMembersFiltered(ctx context.Context, refsetIDs []string, filter MemberFilterOpts) (Set, error)
```

```go
type MemberFilterOpts struct {
    FieldName string
    Op        string // "=" or "!="
    ValueSet  Set    // pre-resolved concept IDs
}
```

In `evaluateFiltered`, when `memberFilters` is non-empty, resolve each `MemberFieldFilter`'s value expression to a Set, build `MemberFilterOpts`, and call `RefsetMembersFiltered`.

## Layer 3 — Relationship struct change

Add `ConcreteValue *ConcreteValue` to `Relationship`. This field is `nil` for concept-valued relationships (zero impact on existing consumers). When set, `TargetID` is `""`.

This enables `PropertiesByGroup` to return concrete values alongside concept-valued relationships, maintaining group correlation.

## Layer 4 — Testing strategy

Extend existing test files (no new test files):

| Feature | Test file | Fixture changes |
|---------|-----------|-----------------|
| String/Boolean concrete | `evaluator_advanced_test.go` | Add `Kind: "string"` and `Kind: "boolean"` entries to `concreteValues` |
| Negated filters (`!=`) | `evaluator_filters_test.go` | Reuse `filterTestProvider`, add `!=` test cases |
| Reverse with wildcard | `evaluator_refined_test.go` | Use existing relationships fixture |
| Reverse in groups | `evaluator_refined_test.go` | Add reverse-relationship data to fixture |
| Concrete in groups | `evaluator_refined_test.go` | Add `ConcreteValue` to `Relationship` entries |
| Reverse + concrete | `evaluator_advanced_test.go` | Combine reverse + concrete fixtures |
| Cardinality | `evaluator_refined_test.go` | Add concepts with multiple same-type relationships |
| AltIdentifier | `evaluator_advanced_test.go` | Implement `ResolveIdentifier` stub on `testProvider` |
| Dialect filter | `evaluator_filters_test.go` | Implement `MatchDialect` stub on `filterTestProvider` |
| Member field projection | `evaluator_filters_test.go` | Implement `RefsetMembersFiltered` stub |

## Layer 5 — Cleanup

- Update the doc comment on `Evaluate` (lines 15-27) to remove the "deferred" mentions.
- Remove `isDefaultCardinality` helper (no longer needed).
- Remove `_ = strVal` / `_ = boolVal` silencing lines.

## Non-goals

- No changes to the parser or AST nodes.
- No changes to `set.go`.
- No performance optimization of `RelationshipSources` with `nil` targetIDs — correctness first.
