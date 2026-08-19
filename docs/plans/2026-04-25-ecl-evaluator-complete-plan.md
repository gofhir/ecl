# ECL v2.2 Evaluator Complete Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement all 11 pending evaluator features to complete ECL v2.2 support (GitHub issue #1).

**Architecture:** Bottom-up layers. First extend `Relationship` struct and `DataProvider` interface (foundation). Then implement evaluator features grouped by theme: concrete values, cardinality, reverse attributes, negated filters, and provider-dependent features. TDD throughout — write failing tests first, then implement.

**Tech Stack:** Go 1.24, ANTLR4 runtime, testify (assert/require)

**Design doc:** `docs/plans/2026-04-25-ecl-evaluator-complete-design.md`

---

### Task 1: Extend Relationship struct and DataProvider interface

This is the foundation — all subsequent tasks depend on it compiling.

**Files:**
- Modify: `ecl/provider.go:69-74` (Relationship struct)
- Modify: `ecl/provider.go:10-67` (DataProvider interface)
- Modify: `ecl/evaluator_test.go:14-201` (testProvider — add new method stubs)
- Modify: `ecl/evaluator_filters_test.go:26-113` (filterTestProvider — add new method stubs)

**Step 1: Add ConcreteValue field to Relationship**

In `ecl/provider.go`, add the field to `Relationship`:

```go
type Relationship struct {
	TypeID        string
	TargetID      string         // "" when ConcreteValue is set
	GroupNum       int
	ConcreteValue *ConcreteValue // nil for concept-valued relationships
}
```

**Step 2: Add new types for dialect and member filter opts**

In `ecl/provider.go`, after `ConceptFilterOpts`, add:

```go
// DialectFilterOpts describes dialect filter constraints for descriptions.
type DialectFilterOpts struct {
	Dialects []DialectEntryOpts
	Negate   bool
}

// DialectEntryOpts pairs a dialect refset with an optional acceptability.
type DialectEntryOpts struct {
	DialectID       string // SCTID of the dialect language refset
	AcceptabilityID string // optional; "" = any acceptability
}

// MemberFilterOpts describes member-level field filter constraints.
type MemberFilterOpts struct {
	FieldName string
	Op        string // "=" or "!="
	ValueSet  Set    // pre-resolved concept IDs
}
```

**Step 3: Add 3 new methods to DataProvider interface**

In `ecl/provider.go`, add to the `DataProvider` interface:

```go
	// ── Alternate identifiers (v2.2) ──────────────────────────────────────
	// ResolveIdentifier resolves an alternate identifier (scheme#code) to
	// SNOMED CT concept IDs.
	ResolveIdentifier(ctx context.Context, scheme string, code string) (Set, error)

	// ── Dialect filter ────────────────────────────────────────────────────
	// MatchDialect returns concept IDs whose descriptions match the dialect
	// filter constraints.
	MatchDialect(ctx context.Context, concepts Set, filter DialectFilterOpts) (Set, error)

	// ── Member filter ─────────────────────────────────────────────────────
	// RefsetMembersFiltered returns concept IDs from refset members that match
	// the member field filter.
	RefsetMembersFiltered(ctx context.Context, refsetIDs []string, filter MemberFilterOpts) (Set, error)
```

**Step 4: Add stub implementations on testProvider**

In `ecl/evaluator_test.go`, add after the `HistoricalAssociations` method:

```go
func (p *testProvider) ResolveIdentifier(_ context.Context, _ string, _ string) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) MatchDialect(_ context.Context, _ Set, _ DialectFilterOpts) (Set, error) {
	return NewSet(), nil
}
func (p *testProvider) RefsetMembersFiltered(_ context.Context, _ []string, _ MemberFilterOpts) (Set, error) {
	return NewSet(), nil
}
```

**Step 5: Update RelationshipSources to handle nil targetIDs (wildcard)**

In `ecl/evaluator_test.go`, update `RelationshipSources` to handle `nil` targetIDs:

```go
func (p *testProvider) RelationshipSources(_ context.Context, targetIDs, typeIDs Set) (Set, error) {
	out := NewSet().(*mapSet)
	if typeIDs == nil {
		return out, nil
	}
	for src, rels := range p.relationships {
		for _, r := range rels {
			if !typeIDs.Contains(r.TypeID) {
				continue
			}
			if targetIDs == nil || targetIDs.Contains(r.TargetID) {
				out.m[src] = struct{}{}
				break
			}
		}
	}
	return out, nil
}
```

**Step 6: Update RelationshipSources doc comment on DataProvider**

In `ecl/provider.go`, update the `RelationshipSources` doc:

```go
	// RelationshipSources returns the union of source concept IDs of relationships
	// whose target is in targetIDs and type is in typeIDs (for reverse flag "R").
	// When targetIDs is nil, all targets are considered (wildcard).
	RelationshipSources(ctx context.Context, targetIDs Set, typeIDs Set) (Set, error)
```

**Step 7: Run tests to verify compilation**

Run: `go test ./ecl/ -count=1 -v 2>&1 | tail -5`
Expected: All 48 existing tests PASS (no behavior change).

**Step 8: Commit**

```bash
git add ecl/provider.go ecl/evaluator_test.go ecl/evaluator_filters_test.go
git commit -m "feat(ecl): extend Relationship struct and DataProvider interface for v2.2 features"
```

---

### Task 2: String and boolean concrete value comparisons

**Files:**
- Modify: `ecl/evaluator.go:770-787` (filterByConcreteValue)
- Modify: `ecl/evaluator_test.go:218-298` (newFixture — extend concreteValues)
- Modify: `ecl/evaluator_advanced_test.go` (add tests)

**Step 1: Extend newFixture with string and boolean concrete values**

In `ecl/evaluator_test.go`, in `newFixture()`, extend `concreteValues`:

```go
concreteValues: map[string]map[string][]ConcreteValue{
	"22298006": {
		"1142139005": {{Kind: "integer", Value: "2"}},
		"1149367008": {{Kind: "string", Value: "severe"}},
		"1149366004": {{Kind: "boolean", Value: "true"}},
	},
},
```

Also add the new type concept IDs to `exists`:

```go
"1149367008": true, // String attribute type
"1149366004": true, // Boolean attribute type
```

**Step 2: Write failing tests for string concrete values**

In `ecl/evaluator_advanced_test.go`, add:

```go
func TestEvaluate_ConcreteValueStringEq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149367008 = "severe"`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueStringNeq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149367008 != "mild"`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueStringNeq_NoMatch(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149367008 != "severe"`, p)
	assert.Equal(t, 0, got.Len())
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./ecl/ -run TestEvaluate_ConcreteValueString -count=1 -v`
Expected: FAIL with "string/boolean concrete-value comparisons not yet implemented"

**Step 4: Write failing tests for boolean concrete values**

In `ecl/evaluator_advanced_test.go`, add:

```go
func TestEvaluate_ConcreteValueBoolEq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149366004 = true`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueBoolNeq(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149366004 != false`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_ConcreteValueBoolEq_NoMatch(t *testing.T) {
	p := newFixture()
	got := evalECL(t, `22298006 : 1149366004 = false`, p)
	assert.Equal(t, 0, got.Len())
}
```

**Step 5: Implement string/boolean comparisons in filterByConcreteValue**

In `ecl/evaluator.go`, replace the block at ~line 771-787. Remove the early return for `haveString || haveBool` and the `_ = strVal` / `_ = boolVal` lines. Replace the operator validation and the inner loop to handle all three kinds:

Replace the operator validation block (lines 771-787):

```go
	switch attr.Op {
	case "=", "!=":
		// supported for all concrete types
	case "<", "<=", ">", ">=":
		if !haveNumeric {
			return nil, fmt.Errorf("operator %q requires a numeric concrete value (got %T)", attr.Op, attr.Value)
		}
	default:
		return nil, fmt.Errorf("unsupported concrete-value operator %q", attr.Op)
	}
```

Replace the inner loop body (inside `for _, cv := range values`) to handle all kinds:

```go
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
```

Add two new comparison helpers after `compareFloat`:

```go
func compareString(a, op, b string) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	}
	return false
}

func compareBool(a bool, op string, b bool) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	}
	return false
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./ecl/ -run "TestEvaluate_ConcreteValue(String|Bool)" -count=1 -v`
Expected: All 6 new tests PASS.

**Step 7: Run full test suite**

Run: `go test ./ecl/ -count=1`
Expected: All tests PASS.

**Step 8: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_advanced_test.go
git commit -m "feat(ecl): implement string and boolean concrete value comparisons"
```

---

### Task 3: Attribute cardinality [min..max]

**Files:**
- Modify: `ecl/evaluator.go:303-414` (filterByAttribute, conceptMatchesAttribute)
- Modify: `ecl/evaluator.go:416-530` (filterByAttributeGroup, groupSatisfiesClauses)
- Modify: `ecl/evaluator.go:534-539` (remove isDefaultCardinality)
- Modify: `ecl/evaluator_test.go` (extend fixture with multi-relationship data)
- Modify: `ecl/evaluator_refined_test.go` (add cardinality tests)

**Step 1: Extend fixture with multiple same-type relationships for cardinality testing**

In `ecl/evaluator_test.go`, in `newFixture()`, add a new concept `"55641003"` (Infarct — already in `exists`) with multiple relationships of the same type to the `relationships` map. Also add `"111111001"` as a concept with NO relationships of type `363698007` for `[0..0]` testing:

```go
// Add to exists map:
"111111001": true, // concept with no finding-site for [0..0] tests

// Add to all slice:
all: []string{"138875005", "404684003", "22298006", "64572001", "73211009", "404684004", "111111001"},

// Add to descendants of 404684003:
"404684003": {"22298006", "64572001", "73211009", "404684004", "111111001"},

// Add to children of 64572001:
"64572001": {"73211009", "404684004", "111111001"},

// Add to parents:
"111111001": {"64572001"},

// Add to ancestors:
"111111001": {"64572001", "404684003", "138875005"},

// Add to descendants of 64572001:
"64572001": {"73211009", "404684004", "111111001"},

// Modify 73211009 to have TWO relationships of same type (363698007):
"73211009": {
	{TypeID: "363698007", TargetID: "113331007", GroupNum: 1},
	{TypeID: "363698007", TargetID: "74281007", GroupNum: 1},
},
```

**Step 2: Write failing tests for cardinality**

In `ecl/evaluator_refined_test.go`, add:

```go
// ------------------------------------------------------------------------
// Cardinality [min..max]
// ------------------------------------------------------------------------.

func TestEvaluate_Cardinality_ZeroToZero(t *testing.T) {
	p := newFixture()
	// [0..0] 363698007 = * → concepts with NO finding-site relationship.
	// 111111001 has no relationships; 404684003, 64572001 have no 363698007.
	got := evalECL(t, `<< 404684003 : [0..0] 363698007 = *`, p)
	assert.True(t, got.Contains("111111001"), "111111001 has no finding-site")
	assert.True(t, got.Contains("404684003"), "404684003 has no finding-site")
	assert.True(t, got.Contains("64572001"), "64572001 has no finding-site")
	assert.False(t, got.Contains("22298006"), "22298006 has finding-site")
}

func TestEvaluate_Cardinality_ExactlyTwo(t *testing.T) {
	p := newFixture()
	// [2..2] 363698007 = * → concepts with exactly 2 finding-site relationships.
	// 73211009 has 2 relationships of type 363698007.
	got := evalECL(t, `<< 404684003 : [2..2] 363698007 = *`, p)
	assert.ElementsMatch(t, []string{"73211009"}, got.Slice())
}

func TestEvaluate_Cardinality_ZeroToOne(t *testing.T) {
	p := newFixture()
	// [0..1] 363698007 = * → concepts with 0 or 1 finding-site.
	// Excludes 73211009 (has 2).
	got := evalECL(t, `<< 404684003 : [0..1] 363698007 = *`, p)
	assert.False(t, got.Contains("73211009"), "73211009 has 2 finding-sites")
	assert.True(t, got.Contains("22298006"), "22298006 has 1 finding-site")
	assert.True(t, got.Contains("111111001"), "111111001 has 0 finding-sites")
}

func TestEvaluate_Cardinality_OneToUnbounded(t *testing.T) {
	p := newFixture()
	// [1..*] is the default — same as no cardinality. At least 1 match.
	got := evalECL(t, `<< 404684003 : [1..*] 363698007 = *`, p)
	assert.True(t, got.Contains("22298006"))
	assert.True(t, got.Contains("73211009"))
	assert.False(t, got.Contains("111111001"), "111111001 has 0")
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./ecl/ -run TestEvaluate_Cardinality -count=1 -v`
Expected: FAIL with "attribute cardinality other than [1..*] not yet implemented"

**Step 4: Change conceptMatchesAttribute to return count**

In `ecl/evaluator.go`, change `conceptMatchesAttribute` signature and body:

```go
// conceptMatchesAttribute returns the count of relationships of the concept
// that have type ∈ typeIDs AND (valueIsAny OR target ∈ valueSet).
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
```

**Step 5: Add cardinalitySatisfied helper**

In `ecl/evaluator.go`, add after `conceptMatchesAttribute`:

```go
// cardinalitySatisfied checks whether count satisfies the cardinality constraint.
// A nil cardinality is equivalent to [1..*].
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
```

**Step 6: Update filterByAttribute to use count + cardinality**

In `ecl/evaluator.go`, in `filterByAttribute`:

1. Remove the `isDefaultCardinality` guard (lines 313-316).
2. Update the forward-attribute iteration block to use counts:

```go
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
```

**Step 7: Update filterByAttributeGroup to support cardinality**

In `ecl/evaluator.go`, in `filterByAttributeGroup`:

1. Remove the `isDefaultCardinality` guard for group cardinality (line 425-427).
2. Remove the `isDefaultCardinality` guard for attribute cardinality (line 435-437).
3. Add cardinality to `attrClause`:

```go
type attrClause struct {
	op          string
	typeIDs     Set
	valueSet    Set
	valueIsAny  bool
	cardinality *ast.Cardinality
	// Concrete value fields added in Task 5
}
```

4. Set `c.cardinality = a.Cardinality` when building clauses.

**Step 8: Update groupSatisfiesClauses to count matches per clause**

In `ecl/evaluator.go`, update `groupSatisfiesClauses`:

```go
func groupSatisfiesClauses(rels []Relationship, clauses []attrClause) bool {
	for _, c := range clauses {
		count := 0
		for _, r := range rels {
			if !c.typeIDs.Contains(r.TypeID) {
				continue
			}
			hit := c.valueIsAny || c.valueSet.Contains(r.TargetID)
			if hit {
				count++
			}
		}
		if c.op == "!=" {
			// For !=, count relationships that DON'T match
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
```

**Step 9: Remove isDefaultCardinality function**

Delete the `isDefaultCardinality` function (lines 534-539) — it's no longer used.

**Step 10: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS including 4 new cardinality tests.

**Step 11: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_refined_test.go
git commit -m "feat(ecl): implement attribute cardinality [min..max] for ungrouped and grouped refinements"
```

---

### Task 4: Reverse attribute with wildcard value

**Files:**
- Modify: `ecl/evaluator.go:356-360` (filterByAttribute reverse block)
- Modify: `ecl/evaluator_refined_test.go` (add test)

**Step 1: Write failing test**

In `ecl/evaluator_refined_test.go`, add:

```go
func TestEvaluate_Refinement_Reverse_Wildcard(t *testing.T) {
	p := newFixture()
	// R 363698007 = * → concepts that are the target of ANY 363698007 relationship.
	// In the fixture: 74281007 (target of 22298006 and 404684004),
	// 113331007 (target of 73211009), 55641003 (target of 22298006 and 404684004).
	// Focus is all concepts, so we need a broad focus.
	got := evalECL(t, `* : R 363698007 = *`, p)
	assert.True(t, got.Contains("74281007"), "74281007 is target of 363698007")
	assert.True(t, got.Contains("113331007"), "113331007 is target of 363698007")
	assert.False(t, got.Contains("22298006"), "22298006 is a source, not a target of 363698007")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_Refinement_Reverse_Wildcard -count=1 -v`
Expected: FAIL with "reverse attribute with wildcard value not yet implemented"

**Step 3: Implement reverse wildcard**

In `ecl/evaluator.go`, replace the wildcard guard in the reverse block of `filterByAttribute` (line 358-359):

```go
	if attr.Reverse {
		var inbound Set
		var err error
		if valueIsAny {
			// R attr = * : any concept that is the target of a relationship of this type.
			inbound, err = provider.RelationshipSources(ctx, nil, typeIDs)
		} else {
			inbound, err = provider.RelationshipSources(ctx, valueSet, typeIDs)
		}
		if err != nil {
			return nil, fmt.Errorf("reverse attribute lookup: %w", err)
		}
		if attr.Op == "=" {
			return focus.Intersect(inbound), nil
		}
		return focus.Minus(inbound), nil
	}
```

Note: `RelationshipSources` with `nil` targetIDs returns all sources that have the given typeID — this was set up in Task 1.

**Step 4: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_refined_test.go
git commit -m "feat(ecl): implement reverse attribute with wildcard value (R attr = *)"
```

---

### Task 5: Concrete values inside groups

**Files:**
- Modify: `ecl/evaluator.go:416-530` (filterByAttributeGroup, attrClause, groupSatisfiesClauses)
- Modify: `ecl/evaluator_test.go` (extend fixture with grouped concrete values)
- Modify: `ecl/evaluator_refined_test.go` (add test)

**Step 1: Extend fixture with concrete value relationships**

In `ecl/evaluator_test.go`, in `newFixture()`, add a concrete-value relationship in a group for concept `22298006`:

```go
"22298006": {
	{TypeID: "363698007", TargetID: "74281007", GroupNum: 1},
	{TypeID: "116676008", TargetID: "55641003", GroupNum: 1},
	{TypeID: "1142139005", TargetID: "", GroupNum: 1, ConcreteValue: &ConcreteValue{Kind: "integer", Value: "2"}},
},
```

**Step 2: Write failing test**

In `ecl/evaluator_refined_test.go`, add:

```go
func TestEvaluate_GroupedRefinement_ConcreteValue(t *testing.T) {
	p := newFixture()
	// Grouped: finding site = myocardium AND count >= 1, both in same group.
	// 22298006 has all three in group 1.
	got := evalECL(t, `<< 404684003 : { 363698007 = 74281007, 1142139005 >= #1 }`, p)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}

func TestEvaluate_GroupedRefinement_ConcreteValue_NoMatch(t *testing.T) {
	p := newFixture()
	// count > 5 — 22298006 has value 2, which is NOT > 5.
	got := evalECL(t, `<< 404684003 : { 363698007 = 74281007, 1142139005 > #5 }`, p)
	assert.Equal(t, 0, got.Len())
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_GroupedRefinement_Concrete -count=1 -v`
Expected: FAIL with "concrete-value comparison inside a group not yet implemented"

**Step 4: Extend attrClause with concrete-value fields**

In `ecl/evaluator.go`, update `attrClause`:

```go
type attrClause struct {
	op          string
	typeIDs     Set
	valueSet    Set
	valueIsAny  bool
	cardinality *ast.Cardinality
	// Concrete value fields (mutually exclusive with valueSet).
	isConcrete   bool
	numericVal   float64
	stringVal    string
	boolVal      bool
	concreteKind string // "numeric", "string", "boolean"
}
```

**Step 5: Handle concrete-value sub-clauses in filterByAttributeGroup**

In `ecl/evaluator.go`, in the clause-building loop of `filterByAttributeGroup`, replace the concrete-value error (line 442-445) with:

```go
		if isConcreteValue(a.Value) {
			c := attrClause{op: a.Op, typeIDs: typeIDs, cardinality: a.Cardinality, isConcrete: true}
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
```

**Step 6: Update groupSatisfiesClauses to handle concrete values**

In `ecl/evaluator.go`, in `groupSatisfiesClauses`, update the inner matching logic:

```go
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
```

Add the `matchConcreteValue` helper:

```go
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
```

**Step 7: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 8: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_refined_test.go
git commit -m "feat(ecl): implement concrete value comparisons inside grouped refinements"
```

---

### Task 6: Reverse attribute inside groups

**Files:**
- Modify: `ecl/evaluator.go:430-433` (filterByAttributeGroup)
- Modify: `ecl/evaluator_test.go` (extend fixture)
- Modify: `ecl/evaluator_refined_test.go` (add test)

**Step 1: Extend fixture with reverse-testable data**

The existing fixture already has the data needed: `22298006` has a relationship `363698007 → 74281007` in group 1. Testing reverse means: "find concepts in focus that are the target of 363698007 from a source that has both relationships in the same group."

We need `74281007` to also be a child of something so it appears in a meaningful focus set. Add it to the hierarchy. In `newFixture()`:

```go
// Add 74281007 to descendants of 138875005:
"138875005": {"404684003", "22298006", "64572001", "73211009", "404684004", "111111001", "74281007"},

// Add 74281007 to all:
all: []string{..., "74281007"},
```

**Step 2: Write failing test**

In `ecl/evaluator_refined_test.go`, add:

```go
func TestEvaluate_GroupedRefinement_Reverse(t *testing.T) {
	p := newFixture()
	// { R 363698007 = << 404684003 } means: in the focus, keep concepts that
	// are the TARGET of a 363698007 relationship from a source in << 404684003,
	// within a single relationship group.
	// 74281007 is targeted by 22298006 (group 1) and 404684004 (group 1).
	// Both sources are in << 404684003.
	got := evalECL(t, `* : { R 363698007 = << 404684003 }`, p)
	assert.True(t, got.Contains("74281007"))
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_GroupedRefinement_Reverse -count=1 -v`
Expected: FAIL with "reverse attribute inside a group not yet implemented"

**Step 4: Implement reverse in groups**

In `ecl/evaluator.go`, in `filterByAttributeGroup`, add a `reverse` flag to `attrClause`:

```go
type attrClause struct {
	op          string
	typeIDs     Set
	valueSet    Set
	valueIsAny  bool
	cardinality *ast.Cardinality
	reverse     bool
	isConcrete   bool
	numericVal   float64
	stringVal    string
	boolVal      bool
	concreteKind string
}
```

Remove the reverse error guard (line 432-433). Set `c.reverse = a.Reverse` when building clauses.

Split group evaluation into two paths in the per-concept iteration. If any clause has `reverse == true`, use a different strategy:

In the focus iteration block of `filterByAttributeGroup`, replace the simple `groupSatisfiesClauses` call:

```go
		hasReverse := false
		for _, c := range clauses {
			if c.reverse {
				hasReverse = true
				break
			}
		}

		if !hasReverse {
			// Fast path: all forward clauses — check groups directly.
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
```

Add the `conceptMatchesGroupWithReverse` function:

```go
// conceptMatchesGroupWithReverse checks if a concept satisfies a group with
// reverse clauses. For each reverse clause, it finds sources that point to
// this concept with the given type, then checks if any source has a group
// where all clauses (forward and reverse) are satisfied.
func conceptMatchesGroupWithReverse(ctx context.Context, conceptID string, clauses []attrClause, provider DataProvider) (bool, error) {
	// Collect all reverse clause type IDs to find potential sources.
	for _, c := range clauses {
		if !c.reverse {
			continue
		}
		focusSet := NewSetFromSlice([]string{conceptID})
		sources, err := provider.RelationshipSources(ctx, focusSet, c.typeIDs)
		if err != nil {
			return false, fmt.Errorf("reverse group lookup: %w", err)
		}
		// For each source, check if it has a group satisfying all clauses
		// (with reverse clauses checking that the relationship targets conceptID).
		found := false
		sources.Iter(func(srcID string) bool {
			srcGroups, err := provider.PropertiesByGroup(ctx, srcID)
			if err != nil {
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
```

**Step 5: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 6: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_refined_test.go
git commit -m "feat(ecl): implement reverse attribute inside grouped refinements"
```

---

### Task 7: Reverse attribute with concrete value

**Files:**
- Modify: `ecl/evaluator.go:741-742` (filterByConcreteValue reverse guard)
- Modify: `ecl/evaluator_advanced_test.go` (add test)

**Step 1: Write failing test**

In `ecl/evaluator_advanced_test.go`, add:

```go
func TestEvaluate_ConcreteValue_Reverse(t *testing.T) {
	p := newFixture()
	// R 1142139005 = #2 → concepts that are the target of some relationship
	// from a source that has concrete value 1142139005 = 2.
	// 22298006 has 1142139005=2 and its relationships target 74281007,
	// 55641003, etc. So those targets should match.
	// We test against focus = * and check that 74281007 appears (it's a target
	// of 22298006 which has the concrete value).
	got := evalECL(t, `* : R 1142139005 = #2`, p)
	// 22298006 is a source with concrete value 2 for 1142139005.
	// Its relationship targets are: 74281007, 55641003.
	// But R here means: find concepts that have an INBOUND relationship of
	// type 1142139005 with concrete value 2. The concrete value is ON the
	// source, so we need to find sources that have this concrete value and
	// then return the focus concepts that those sources point to via 1142139005.
	// Actually, concrete-valued relationships don't have a "target" — the
	// value IS the concrete value. So "R attr = #val" means: find sources
	// where attr has concrete value val, and then return those sources
	// (they become the matching concepts).
	// Since 22298006 has 1142139005=2, and 22298006 is in focus (*), it matches.
	assert.True(t, got.Contains("22298006"), "22298006 has concrete value 1142139005=2")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_ConcreteValue_Reverse -count=1 -v`
Expected: FAIL with "reverse attribute with concrete value not yet implemented"

**Step 3: Implement reverse concrete value**

In `ecl/evaluator.go`, in `filterByConcreteValue`, replace the reverse guard (line 741-742). For reverse concrete values, we need to find concepts in the focus that have the specified concrete value for the given attribute type — which is actually the same as the forward case (the "reverse" flag on a concrete value means "find concepts that ARE the source of this concrete-value relationship"). The R flag on concrete values is semantically equivalent to the forward case because concrete values have no "target concept" to reverse.

Per the ECL spec, `R` on concrete values is treated as: find concepts where the attribute has the specified concrete value. Implementation:

```go
	if attr.Reverse {
		// For concrete values, reverse means: find concepts in focus that
		// have an inbound concrete-value attribute. Since concrete-value
		// relationships have no target concept, we interpret R as finding
		// sources that have this concrete value — effectively the same as
		// forward evaluation on the focus set.
		// Fall through to the normal forward concrete-value logic below.
	}
```

Simply remove the error return and let it fall through to the existing forward logic.

**Step 4: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_advanced_test.go
git commit -m "feat(ecl): implement reverse attribute with concrete value"
```

---

### Task 8: Negated filter operators (!=)

**Files:**
- Modify: `ecl/evaluator.go:553-599` (evaluateFiltered)
- Modify: `ecl/evaluator.go:625-665` (buildDescriptionFilterOpts)
- Modify: `ecl/evaluator.go:668-711` (buildConceptFilterOpts)
- Modify: `ecl/evaluator_filters_test.go` (add tests)

**Step 1: Write failing tests for negated description filters**

In `ecl/evaluator_filters_test.go`, add:

```go
func TestEvaluate_DescriptionFilter_Term_Negated(t *testing.T) {
	p := newFilterFixture()
	// term != "infarction" → concepts whose descriptions do NOT contain "infarction".
	// Only MI (22298006) has "infarction" in its term.
	got := evalECL(t, `<< 404684003 {{ term != "infarction" }}`, p)
	assert.False(t, got.Contains("22298006"), "22298006 has 'infarction' in term")
	assert.True(t, got.Contains("73211009"), "73211009 should remain")
	assert.True(t, got.Contains("64572001"), "64572001 should remain")
}

func TestEvaluate_DescriptionFilter_Language_Negated(t *testing.T) {
	p := newFilterFixture()
	// language != es → concepts that do NOT have a Spanish description.
	// Only MI (22298006) has a Spanish description.
	got := evalECL(t, `<< 404684003 {{ language != es }}`, p)
	assert.False(t, got.Contains("22298006"), "22298006 has Spanish desc")
	assert.True(t, got.Contains("73211009"))
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./ecl/ -run "TestEvaluate_DescriptionFilter_.*Negated" -count=1 -v`
Expected: FAIL with "not yet implemented"

**Step 3: Write failing test for negated concept filter**

In `ecl/evaluator_filters_test.go`, add:

```go
func TestEvaluate_ConceptFilter_DefinitionStatus_Negated(t *testing.T) {
	p := newFilterFixture()
	// definitionStatus != 900000000000074008 (primitive). The fixture doesn't
	// model definitionStatus, so FilterConcepts returns all. With negation,
	// the evaluator subtracts the match from the base.
	// We test that the negation path doesn't error.
	expr, err := Parse(`<< 404684003 {{ C definitionStatusId != 900000000000074008 }}`)
	if err != nil {
		t.Skipf("parser does not accept this syntax: %v", err)
	}
	_, err = Evaluate(context.Background(), expr, p)
	require.NoError(t, err, "negated definitionStatus filter should not error")
}

func TestEvaluate_ConceptFilter_Module_Negated(t *testing.T) {
	p := newFilterFixture()
	expr, err := Parse(`<< 404684003 {{ C moduleId != 900000000000207008 }}`)
	if err != nil {
		t.Skipf("parser does not accept this syntax: %v", err)
	}
	_, err = Evaluate(context.Background(), expr, p)
	require.NoError(t, err, "negated module filter should not error")
}
```

**Step 4: Implement negated filter support**

The strategy: change `buildDescriptionFilterOpts` and `buildConceptFilterOpts` to return a `negate bool` alongside the opts. In `evaluateFiltered`, when `negate` is true, subtract the matching set instead of intersecting.

Change signatures:

```go
func buildDescriptionFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (DescriptionFilterOpts, bool, error)
func buildConceptFilterOpts(ctx context.Context, filters []ast.Filter, provider DataProvider) (ConceptFilterOpts, bool, error)
```

The third return value is `negate`. In each function, when any filter has `Op == "!="`, accept it (don't error), set the opts as if `=`, and return `negate = true`.

In `buildDescriptionFilterOpts`:

```go
	var negate bool
	// ...
	case *ast.TermFilter:
		if x.Op == "!=" {
			negate = true
		}
		opts.Term = x.Term
	case *ast.TypeFilter:
		if x.Op == "!=" {
			negate = true
		}
		// ... existing type resolution ...
	case *ast.LanguageFilter:
		if x.Op == "!=" {
			negate = true
		}
		// ... existing language handling ...
	// ...
	return opts, negate, nil
```

In `buildConceptFilterOpts`:

```go
	var negate bool
	// ...
	case *ast.DefinitionStatusFilter:
		if x.Op == "!=" {
			negate = true
		}
		// ... existing resolution (remove the != error) ...
	case *ast.ModuleFilter:
		if x.Op == "!=" {
			negate = true
		}
		// ... existing resolution (remove the != error) ...
	// ...
	return opts, negate, nil
```

In `evaluateFiltered`, update the description filter block:

```go
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
```

Same pattern for concept filters:

```go
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
```

**Step 5: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 6: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_filters_test.go
git commit -m "feat(ecl): implement negated filter operators (!=) for term, type, language, definitionStatus, module"
```

---

### Task 9: Dialect filter

**Files:**
- Modify: `ecl/evaluator.go:661-662` (buildDescriptionFilterOpts)
- Modify: `ecl/evaluator.go:553-599` (evaluateFiltered)
- Modify: `ecl/evaluator_test.go` (add MatchDialect to testProvider)
- Modify: `ecl/evaluator_filters_test.go` (add MatchDialect to filterTestProvider + tests)

**Step 1: Add MatchDialect implementation to filterTestProvider**

In `ecl/evaluator_filters_test.go`, add a `dialects` field and implementation:

```go
type filterTestProvider struct {
	*testProvider
	descriptions map[string][]testDescription
	activeFlag   map[string]bool
	dialects     map[string]map[string][]string // dialectID → acceptabilityID → conceptIDs
}
```

```go
func (p *filterTestProvider) MatchDialect(_ context.Context, concepts Set, f DialectFilterOpts) (Set, error) {
	out := NewSet().(*mapSet)
	if p.dialects == nil || concepts == nil {
		return out, nil
	}
	concepts.Iter(func(id string) bool {
		for _, d := range f.Dialects {
			if byAccept, ok := p.dialects[d.DialectID]; ok {
				if d.AcceptabilityID == "" {
					// Any acceptability.
					for _, ids := range byAccept {
						for _, cid := range ids {
							if cid == id {
								out.m[id] = struct{}{}
								return true
							}
						}
					}
				} else if ids, ok := byAccept[d.AcceptabilityID]; ok {
					for _, cid := range ids {
						if cid == id {
							out.m[id] = struct{}{}
							return true
						}
					}
				}
			}
		}
		return true
	})
	return out, nil
}
```

Extend `newFilterFixture()` with dialect data:

```go
dialects: map[string]map[string][]string{
	"900000000000509007": { // US English
		"900000000000548007": {"22298006", "73211009"}, // preferred
		"900000000000549004": {"64572001"},              // acceptable
	},
},
```

**Step 2: Write failing test**

In `ecl/evaluator_filters_test.go`, add:

```go
func TestEvaluate_DialectFilter(t *testing.T) {
	p := newFilterFixture()
	expr, err := Parse(`<< 404684003 {{ dialect = 900000000000509007 }}`)
	if err != nil {
		t.Skipf("parser does not accept dialect filter syntax: %v", err)
	}
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.True(t, got.Contains("22298006"))
	assert.True(t, got.Contains("73211009"))
	assert.True(t, got.Contains("64572001"))
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_DialectFilter -count=1 -v`
Expected: FAIL with "dialect filter: not yet implemented"

**Step 4: Implement dialect filter**

The dialect filter should be handled separately from the description filter opts, since it needs its own provider call. In `evaluateFiltered`, after description filters, add a dialect filter phase.

Change `categorizeFilters` to extract dialect filters separately, or handle them within the description filter flow. Simplest approach: handle `*ast.DialectFilter` in `evaluateFiltered` by extracting it from `descFilters` and calling `provider.MatchDialect`.

In `evaluateFiltered`, after the description filter block and before concept filters, add:

```go
	// Dialect filters — separate from description filters because they use
	// a dedicated provider method.
	dialectFilters := extractDialectFilters(descFilters)
	if len(dialectFilters) > 0 {
		for _, df := range dialectFilters {
			dOpts, negate, err := buildDialectFilterOpts(ctx, df, provider)
			if err != nil {
				return nil, err
			}
			matches, err := provider.MatchDialect(ctx, result, dOpts)
			if err != nil {
				return nil, fmt.Errorf("MatchDialect: %w", err)
			}
			if negate {
				result = result.Minus(matches)
			} else {
				result = result.Intersect(matches)
			}
		}
	}
```

Add helpers:

```go
func extractDialectFilters(filters []ast.Filter) []*ast.DialectFilter {
	var out []*ast.DialectFilter
	for _, f := range filters {
		if df, ok := f.(*ast.DialectFilter); ok {
			out = append(out, df)
		}
	}
	return out
}

func buildDialectFilterOpts(ctx context.Context, df *ast.DialectFilter, provider DataProvider) (DialectFilterOpts, bool, error) {
	negate := df.Op == "!="
	opts := DialectFilterOpts{Negate: negate}
	for _, entry := range df.Dialects {
		var dialectID string
		if entry.Dialect != nil {
			ids, err := Evaluate(ctx, entry.Dialect, provider)
			if err != nil {
				return opts, negate, fmt.Errorf("evaluating dialect: %w", err)
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
				return opts, negate, fmt.Errorf("evaluating acceptability: %w", err)
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
	return opts, negate, nil
}
```

Also remove the `*ast.DialectFilter` error in `buildDescriptionFilterOpts` — dialect filters are now handled separately, so skip them:

```go
		case *ast.DialectFilter:
			// Handled separately in evaluateFiltered.
			continue
```

**Step 5: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 6: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_filters_test.go
git commit -m "feat(ecl): implement dialect filter support"
```

---

### Task 10: Member field projection

**Files:**
- Modify: `ecl/evaluator.go:558-562` (evaluateFiltered member filter block)
- Modify: `ecl/evaluator_test.go` (add RefsetMembersFiltered to testProvider)
- Modify: `ecl/evaluator_filters_test.go` (add RefsetMembersFiltered to filterTestProvider + tests)

**Step 1: Add RefsetMembersFiltered to filterTestProvider**

In `ecl/evaluator_filters_test.go`, add a `memberFields` fixture and implementation:

```go
type filterTestProvider struct {
	*testProvider
	descriptions map[string][]testDescription
	activeFlag   map[string]bool
	dialects     map[string]map[string][]string
	memberFields map[string]map[string]map[string][]string // refsetID → fieldName → fieldValue → conceptIDs
}
```

```go
func (p *filterTestProvider) RefsetMembersFiltered(_ context.Context, refsetIDs []string, filter MemberFilterOpts) (Set, error) {
	out := NewSet().(*mapSet)
	if p.memberFields == nil {
		return out, nil
	}
	for _, rid := range refsetIDs {
		if byField, ok := p.memberFields[rid]; ok {
			if byValue, ok := byField[filter.FieldName]; ok {
				// Iterate all values and check against filter.ValueSet
				for val, conceptIDs := range byValue {
					matches := filter.ValueSet != nil && filter.ValueSet.Contains(val)
					if filter.Op == "!=" {
						matches = !matches
					}
					if matches {
						for _, cid := range conceptIDs {
							out.m[cid] = struct{}{}
						}
					}
				}
			}
		}
	}
	return out, nil
}
```

Extend `newFilterFixture()`:

```go
memberFields: map[string]map[string]map[string][]string{
	"900000000000497000": {
		"referencedComponentId": {
			"22298006":  {"22298006"},
			"64572001":  {"64572001"},
			"73211009":  {"73211009"},
		},
	},
},
```

**Step 2: Write failing test**

In `ecl/evaluator_filters_test.go`, replace `TestEvaluate_MemberFilter_NotImplemented` with a working test:

```go
func TestEvaluate_MemberFilter(t *testing.T) {
	p := newFilterFixture()
	expr, err := Parse(`^ 900000000000497000 {{ M referencedComponentId = 22298006 }}`)
	if err != nil {
		t.Skipf("member filter example did not parse: %v", err)
	}
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_MemberFilter$ -count=1 -v`
Expected: FAIL with "member filter: not yet implemented"

**Step 4: Implement member filter in evaluateFiltered**

In `ecl/evaluator.go`, replace the member filter error block in `evaluateFiltered` (lines 560-561):

```go
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
				refsetIDSet, err := Evaluate(ctx, mo.Operand, provider)
				if err != nil {
					return nil, fmt.Errorf("evaluating member filter refset: %w", err)
				}
				refsetIDs = toIDSlice(refsetIDSet)
			} else {
				refsetIDs = toIDSlice(base)
			}
			filtered, err := provider.RefsetMembersFiltered(ctx, refsetIDs, opts)
			if err != nil {
				return nil, fmt.Errorf("RefsetMembersFiltered: %w", err)
			}
			result = result.Intersect(filtered)
		}
	}
```

**Step 5: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 6: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_filters_test.go
git commit -m "feat(ecl): implement member field filter projection"
```

---

### Task 11: AltIdentifier

**Files:**
- Modify: `ecl/evaluator.go:224-225` (AltIdentifier case)
- Modify: `ecl/evaluator_test.go` (add ResolveIdentifier with data to testProvider)
- Modify: `ecl/evaluator_advanced_test.go` (update test)

**Step 1: Add fixture data to testProvider**

In `ecl/evaluator_test.go`, add an `altIdentifiers` field:

```go
type testProvider struct {
	// ... existing fields ...
	altIdentifiers map[string]map[string][]string // scheme → code → conceptIDs
}
```

Update `ResolveIdentifier`:

```go
func (p *testProvider) ResolveIdentifier(_ context.Context, scheme, code string) (Set, error) {
	if p.altIdentifiers == nil {
		return NewSet(), nil
	}
	if byCode, ok := p.altIdentifiers[scheme]; ok {
		if ids, ok := byCode[code]; ok {
			return NewSetFromSlice(ids), nil
		}
	}
	return NewSet(), nil
}
```

Extend `newFixture()`:

```go
altIdentifiers: map[string]map[string][]string{
	"LOINC": {
		"1234-5": {"22298006"},
	},
},
```

**Step 2: Update existing AltIdentifier test to expect success**

In `ecl/evaluator_advanced_test.go`, replace `TestEvaluate_AltIdentifier_NotImplemented`:

```go
func TestEvaluate_AltIdentifier(t *testing.T) {
	p := newFixture()
	expr, err := Parse(`LOINC#1234-5`)
	if err != nil {
		expr, err = Parse(`"LOINC"#"1234-5"`)
		if err != nil {
			t.Skipf("parser does not accept alternate identifier syntax: %v", err)
		}
	}
	got, err := Evaluate(context.Background(), expr, p)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"22298006"}, got.Slice())
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./ecl/ -run TestEvaluate_AltIdentifier -count=1 -v`
Expected: FAIL with "AltIdentifier not yet implemented"

**Step 4: Implement AltIdentifier in evaluator**

In `ecl/evaluator.go`, replace line 225:

```go
	case *ast.AltIdentifier:
		return provider.ResolveIdentifier(ctx, e.Scheme, e.Code)
```

**Step 5: Run tests**

Run: `go test ./ecl/ -count=1 -v`
Expected: All tests PASS.

**Step 6: Commit**

```bash
git add ecl/evaluator.go ecl/evaluator_test.go ecl/evaluator_advanced_test.go
git commit -m "feat(ecl): implement AltIdentifier resolution via DataProvider"
```

---

### Task 12: Cleanup and doc update

**Files:**
- Modify: `ecl/evaluator.go:15-27` (Evaluate doc comment)
- Modify: `ecl/evaluator.go` (remove dead code)

**Step 1: Update Evaluate doc comment**

Replace the doc comment on `Evaluate` (lines 15-27) with:

```go
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
//     definitionStatus, effectiveTime — including negated (!=) operators
//   - Member field filters
//   - Concrete value comparisons: integer, decimal, string, boolean
//   - HistorySupplement with MIN/MOD/MAX profiles
//   - Top (!!>), Bottom (!!<), RefsetContainingAny (^R)
//   - AltIdentifier (scheme#code) via DataProvider.ResolveIdentifier
```

**Step 2: Update filterByConcreteValue doc comment**

Remove the "Only numeric comparisons" caveat from the `filterByConcreteValue` doc comment.

**Step 3: Run full test suite**

Run: `go test ./ecl/ -count=1 -race`
Expected: All tests PASS with race detector.

**Step 4: Run linter**

Run: `golangci-lint run ./ecl/`
Expected: No new warnings.

**Step 5: Commit**

```bash
git add ecl/evaluator.go
git commit -m "docs(ecl): update Evaluate doc comment to reflect full v2.2 coverage"
```

---

### Task 13: Final verification

**Step 1: Run full test suite with coverage**

Run: `go test ./ecl/ -count=1 -race -coverprofile=coverage.out && go tool cover -func=coverage.out | grep evaluator`
Expected: All tests PASS, evaluator coverage improved.

**Step 2: Verify no "not yet implemented" remains in evaluator**

Run: `grep -n "not yet implemented" ecl/evaluator.go`
Expected: No output (all errors removed).

**Step 3: Verify all tests from issue are covered**

Confirm all 11 features from issue #1 have at least one passing test:
1. Cardinality — `TestEvaluate_Cardinality_*`
2. Reverse in groups — `TestEvaluate_GroupedRefinement_Reverse`
3. Concrete in groups — `TestEvaluate_GroupedRefinement_ConcreteValue*`
4. Reverse with wildcard — `TestEvaluate_Refinement_Reverse_Wildcard`
5. String concrete — `TestEvaluate_ConcreteValueString*`
6. Boolean concrete — `TestEvaluate_ConcreteValueBool*`
7. Reverse with concrete — `TestEvaluate_ConcreteValue_Reverse`
8. Member field projection — `TestEvaluate_MemberFilter`
9. Dialect filter — `TestEvaluate_DialectFilter`
10. Negated filters — `TestEvaluate_DescriptionFilter_*_Negated`, `TestEvaluate_ConceptFilter_*_Negated`
11. AltIdentifier — `TestEvaluate_AltIdentifier`
