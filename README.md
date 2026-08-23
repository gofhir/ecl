# go-ecl

[![CI](https://github.com/gofhir/ecl/actions/workflows/ci.yml/badge.svg)](https://github.com/gofhir/ecl/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/gofhir/ecl.svg)](https://pkg.go.dev/github.com/gofhir/ecl)
[![Release](https://img.shields.io/github/v/release/gofhir/ecl)](https://github.com/gofhir/ecl/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Embeddable parser and evaluator for the SNOMED CT **Expression Constraint Language (ECL) v2.2** in pure Go. Comes with parsers for **SNOMED Compositional Grammar (SCG)** and a **Machine Readable Concept Model (MRCM)** validator, plus a `gofhir-ecl` CLI and a 116-case conformance suite.

```go
import "github.com/gofhir/ecl/ecl"

ast, _   := ecl.Parse("<< 404684003 |Clinical finding| {{ D term = wild: \"Diabet*\" }}")
result, _ := ecl.Evaluate(ctx, ast, yourProvider)
// result.Slice() == []string{"73211009", ...}
```

## Why

ECL is the standard query language for SNOMED CT. Until now, evaluating it from Go meant standing up [Snowstorm](https://github.com/IHTSDO/snowstorm) (Java + Elasticsearch) over HTTP. `go-ecl` is the first Go-native implementation, designed to embed inside FHIR servers, ValueSet authoring tools, CDS pipelines, edge devices, and CI validators — without the JVM.

## Status

- ✅ **ECL v2.2 evaluator** — hierarchy operators, compound (`AND`/`OR`/`MINUS`), refinements with conjunction **and disjunction**, attribute groups with cardinality, reverse `R`, dot notation, concrete values (integer/decimal/string/boolean), history supplements with MIN/MOD/MAX, Top/Bottom, MemberOf, RefsetContainingAny (`^R`), AltIdentifier
- ✅ **Filters** — term (word-prefix and `wild`), description type, language, `dialectId`, active, module, effectiveTime, definitionStatus, memberField
- ✅ **SCG** parser + validator
- ✅ **MRCM** loader + validator (uses ECL evaluator internally)
- ✅ **SCTID** Verhoeff checksum + partition validation
- ✅ **116/116** bundled conformance cases pass, all executed by CI
- 📦 Latest release: see [GitHub Releases](https://github.com/gofhir/ecl/releases)

### Known limitations

Each of these returns `ecl.ErrUnsupportedFeature` rather than a silently wrong
result, so you can classify it with `errors.Is` and answer 501 instead of serving
bad data:

| Construct | Why |
|---|---|
| `{{ D term != … }}`, `{{ D language != … }}`, `{{ D type != … }}`, **unless** the provider implements `NegatingDescriptionProvider` | Negating a description filter is a per-ROW operation: a concept with both an FSN and a Spanish synonym satisfies `language != es` through its FSN, so set subtraction is wrong. Negated **concept** filters (`{{ C … != … }}`) always work. |
| `{{ D dialect = en-gb }}` (alias form) | Mapping a dialect alias to a language reference set's SCTID is terminology data; only the international English aliases are universal. Use `{{ D dialectId = 900000000000508004 }}`. |
| `^[field]` projection | `Set` carries concept IDs only. Use a `{{ M … }}` member filter. |
| `{{ D id = … }}` | The parser models it, but `DescriptionFilterOpts` has no field to carry the ids to the provider. |
| A term filter with a SET of terms — `{{ D term = ("a" "b") }}` | Any-of semantics, which `DescriptionFilterOpts.Term` cannot express. A single term, including a multi-word one, works. |
| An `effectiveTime` filter with a set of values | Same reason: `ConceptFilterOpts` carries one value and one operator. |
| A cardinality or `!=` on a reverse (`R`) attribute, **unless** the provider implements `InboundRelationshipsProvider` | `RelationshipTargets` returns a `Set`, so it loses the inbound count and the per-type total. With the capability these work. |
| A cardinality on a reverse attribute **inside a group** | The group belongs to the source concept, so what `[2..*]` counts is undefined. ECL 6.3 defines reverse cardinality only for an ungrouped refinement, the official example set never places `R` inside braces, and Ontoserver rejects the construct outright. See the note below. |
| `AttributeDomain.InGroupCardinality` (MRCM) | Loaded and exposed, not yet enforced. |

## Install

```bash
go get github.com/gofhir/ecl
```

CLI:

```bash
go install github.com/gofhir/ecl/cmd/gofhir-ecl@latest
```

## Library usage

### Parse + evaluate

```go
import (
    "context"
    "github.com/gofhir/ecl/ecl"
)

ctx := context.Background()
ast, err := ecl.Parse("<< 404684003 |Clinical finding|")
if err != nil { /* handle parse error */ }

set, err := ecl.Evaluate(ctx, ast, provider) // provider implements ecl.DataProvider
if err != nil { /* handle evaluation error */ }

for _, id := range set.Slice() {
    // id is a SNOMED CT concept ID matching the constraint
}
```

### `DataProvider` contract

You implement [`ecl.DataProvider`](ecl/provider.go) against your storage (PostgreSQL closure tables, in-memory maps, Elasticsearch, an HTTP terminology server, …). Read the contract in the interface's godoc before you start — it states the rules the evaluator relies on (never return a nil `Set`, empty input yields empty output, only `FilterConcepts` may filter by the `active` flag, and the direction of `HistoricalAssociations`, which is the opposite of the intuitive reading).

Most methods take sets or slices so they can be answered with one batch query. Two are per-concept by signature — `PropertiesByGroup` and `ConcreteValues` — so on their own a broad refinement issues one query per focus concept. **Implement the optional `ecl.BatchPropertiesProvider` to collapse that into a single call**; see [Optional capabilities](#optional-capabilities). The 18 required methods split into:

| Group | Methods |
|---|---|
| Hierarchy | `Descendants`, `Ancestors`, `Children`, `Parents` |
| Concepts | `ConceptExists`, `AllConcepts` |
| Attributes | `RelationshipTargets`, `RelationshipSources`, `PropertiesByGroup`, `ConcreteValues` |
| Descriptions | `MatchDescription`, `MatchDialect` |
| Concept filters | `FilterConcepts` |
| Refsets | `RefsetMembers`, `RefsetsContainingMembers`, `RefsetMembersFiltered` |
| History | `HistoricalAssociations` |
| v2.2 | `ResolveIdentifier` (alternate identifiers) |

Two things make implementing it tractable:

**`ecl.UnimplementedDataProvider`** — embed it and implement only what you need. Every method you skip returns `ErrUnsupportedFeature` instead of the empty set, so a partial provider reports "not supported" rather than silently answering "no matches". It also means future methods are not a breaking change for you.

```go
type myProvider struct {
    ecl.UnimplementedDataProvider
    db *sql.DB
}
```

### Optional capabilities

Three things cannot be expressed through `DataProvider` as it stands, and widening
that interface would break every implementation. They are offered as **optional
interfaces** the evaluator type-asserts for instead — the shape the standard
library uses for `io.ReaderFrom` or `http.Flusher`. Implement what your storage
answers well; the evaluator falls back or reports the forms it cannot handle.

| Interface | What it unlocks |
|---|---|
| `BatchPropertiesProvider` | One query per refinement instead of one per focus concept. Measured against the bundled fixture: 27 calls → 1. |
| `BatchConcreteValuesProvider` | The same for concrete-value comparisons, which otherwise cost N×T calls. |
| `InboundRelationshipsProvider` | `[m..n] R attr = value` and `R attr != value`, which need the inbound count and the per-type total that `RelationshipTargets` cannot return. |
| `NegatingDescriptionProvider` | `{{ D term != … }}`, `{{ D language != … }}`, `{{ D type != … }}`, whose negation is per description ROW and cannot be emulated with set arithmetic. |

The reference provider in [`ecl/providertest/fixture.go`](ecl/providertest/fixture.go)
implements the first three; read it for a worked example.

#### A note on reverse attributes inside a group

This library evaluates `{ R attr = value }` — reading it as "some source concept
has a relationship group holding (attr → focus)". Be aware that **Ontoserver
rejects the construct entirely** with *"Cannot reverse an attribute inside a
group"*, and that the specification neither permits nor forbids it: §6.2 describes
reverse attributes and attribute groups separately, §6.3 defines reverse
cardinality only for an ungrouped refinement, and the
[official example set](https://github.com/IHTSDO/snomed-expression-constraint-language)
never places `R` inside braces.

One consequence is worth stating plainly: in a MIXED group such as
`{ R 363698007 = X, 116676008 = Y }`, the second clause constrains the SOURCE
concept, not the focus — because that is whose group it is. If you expected it to
constrain the focus, write it outside the braces. Portable ECL keeps `R` out of
groups.

**`ecl/providertest`** — check your implementation against the rules the godoc cannot fully state:

```go
func TestMyProvider(t *testing.T) {
    providertest.VerifyContract(t, func() ecl.DataProvider { return newMyProvider(t) })
}
```

`VerifyContract` probes **your** data — it asks the provider for concepts, refsets
and relationships it actually has, then asserts the invariants that must hold
whatever those are: never return a nil `Set`, an empty input yields the empty
`Set`, `nil` sourceIDs means wildcard, the hierarchy is transitive, only
`FilterConcepts` may filter by `active`, refset membership is invertible, the
history profiles nest. A check your data cannot exercise is **skipped with a
reason**, not failed, so read the output: a provider that skips most of the suite
has not been verified.

`providertest.VerifyFixture(t)` is the other half. It runs the 116 bundled cases,
whose expectations are concrete concept IDs, so it verifies the **evaluator** and
only passes against the bundled fixture — a correct provider carrying different
data fails 89 of them.

The reference in-memory provider is [`ecl/providertest/fixture.go`](ecl/providertest/fixture.go) — read it to see the expected semantics.

### SCG + MRCM

```go
import (
    "context"

    "github.com/gofhir/ecl/mrcm"
    "github.com/gofhir/ecl/scg"
)

ctx := context.Background()

expr, err := scg.Parse("22298006 : { 363698007 = 74281007 }")
if err != nil { /* handle parse error */ }

model, err := mrcm.LoadFromBytes(mrcmJSON) // or mrcm.LoadFromJSON(reader)
if err != nil { /* handle load error */ }

res, err := mrcm.Validate(ctx, expr, model, provider)
if err != nil { /* handle validation error */ }

for _, issue := range res.Issues {
    fmt.Printf("%s at %s: %s\n", issue.Kind, issue.Path, issue.Message)
}
```

Every snippet in this README is backed by a runnable `Example` in the package it
documents ([`ecl/example_test.go`](ecl/example_test.go),
[`mrcm/example_test.go`](mrcm/example_test.go)), so CI fails if the API drifts
from the docs.

## CLI: `gofhir-ecl`

```
gofhir-ecl <command> [flags] [args]

Commands:
  validate     parse an ECL expression; report syntax errors
  explain      parse and pretty-print the AST
  eval         evaluate against an in-memory YAML fixture
  conformance  run the bundled v2.2 conformance suite
  version      print the build version
```

Exit codes: `0` success (including `-h`), `1` runtime error, `2` usage error,
`3` invalid ECL syntax, `4` feature not supported by this build. Results go to
stdout, diagnostics to stderr.

### `validate`

```bash
$ gofhir-ecl validate "<< 404684003"
OK
$ gofhir-ecl validate "<< invalid!!!"
error: invalid ECL: syntax error at 1:6: ...
```

### `explain`

```bash
$ gofhir-ecl explain "<< 404684003 : 363698007 = 74281007"
Refined :
  focus:
    DescendantOrSelfOf <<
      ConceptRef 404684003
  refinement: <0 groups, 0 conjunction, 0 disjunction>
  attributes:
    Attribute =
      name:
        ConceptRef 363698007
      value:
        ConceptRef 74281007
```

### `eval`

```bash
$ gofhir-ecl eval --fixture ecl/providertest/testdata/fixtures/standard.yaml "<< 73211009"
11687002
73211009
```

The fixture is a YAML file describing concepts, parents, descriptions, relationships, refsets, dialects, member fields, and history associations. See [`ecl/providertest/testdata/fixtures/standard.yaml`](ecl/providertest/testdata/fixtures/standard.yaml) for the schema.

### `conformance`

```bash
$ gofhir-ecl conformance
116 passed, 0 failed, 0 skipped, 116 total

$ gofhir-ecl conformance -filter '^memberOf'
PASS  ECL v2.2 features (Top, Bottom, AltIdentifier, ^R, MemberOf) :: memberOf refset

1 passed, 0 failed, 0 skipped, 1 total
```

Useful in CI to prove your `DataProvider` implementation matches the spec.

## Conformance suite

The bundled suite lives in [`ecl/providertest/testdata/`](ecl/providertest/testdata/) and is **embedded in the binary**, so `gofhir-ecl conformance` works from any directory, including after `go install`. It currently covers 116 cases across 9 areas of the spec, including a suite of expressions that must be REJECTED.

| Area | Cases | Spec section |
|---|---|---|
| Hierarchy operators (incl. Top/Bottom on non-closed sets) | 10 | 5.1 |
| Compound expressions | 4 | 5.2 |
| Primitives | 4 | 5.0 |
| Refinements (disjunction, groups, cardinality, `!=`, dot, reverse) | 25 | 5.3 |
| Filters (term, type, language, dialectId, module, definitionStatus, active, memberField) | 21 | 5.4 |
| History supplements (MIN/MOD/MAX) | 6 | 5.5 |
| Concrete values | 4 | 5.3.4 |
| v2.2 (Top, Bottom, AltIdentifier, `^R`) | 7 | v2.2 |
| **Errors** — input that must be rejected, not truncated | 14 | grammar |

Each suite is a YAML file with cases of the form:

```yaml
- name: descendantOrSelfOf root
  expression: "<< 138875005"
  expectedIds: ["138875005", "404684003", "22298006", ...]
```

Cases can also assert error paths via `expectError: true`. The runner is reusable as a Go package ([`internal/conformance`](internal/conformance/)) so you can drive it against your own fixtures or your own provider.

## Project layout

```
ecl/                       ECL parser, AST, evaluator, set, DataProvider interface
ecl/grammar/               Generated ANTLR4 grammar (run `make generate`, do not edit)
ecl/ast/                   AST node types
ecl/providertest/          Conformance runner + in-memory fixture provider, for
                           verifying your own DataProvider
ecl/providertest/testdata/ Bundled suites + fixtures (embedded in the binary)
scg/                       Compositional Grammar (SCG) parser + validator
mrcm/                      MRCM loader + validator (uses ecl/ for constraints)
sctid/                     SCTID Verhoeff checksum + partition validation
cmd/gofhir-ecl/            CLI binary
```

## Versioning

Semantic versioning. Releases are managed by [release-please](https://github.com/googleapis/release-please) from Conventional Commits. Breaking changes in the `DataProvider` interface bump the major.

## Contributing

Issues and PRs welcome. Before submitting:

```bash
make test    # go test -race ./...
make lint    # golangci-lint run
```

The conformance suite is the source of truth for expected behavior — extending the suite is the cheapest way to lock in a fix or land a new ECL feature.

## License

[Apache License 2.0](LICENSE) — Copyright © 2026 Roberto Araneda Espinoza.

## See also

- [SNOMED CT ECL specification](https://confluence.ihtsdotools.org/display/DOCECL)
- [Snowstorm](https://github.com/IHTSDO/snowstorm) — the reference Java implementation; ECL grammar and edge-case behavior follow Snowstorm
- [FHIR Terminology Service](https://www.hl7.org/fhir/terminology-service.html) — `ValueSet/$expand` with `compose.include.filter` (`property=constraint, op==`) is the FHIR mechanism for using ECL
