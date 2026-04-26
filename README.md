# go-ecl

[![CI](https://github.com/gofhir/ecl/actions/workflows/ci.yml/badge.svg)](https://github.com/gofhir/ecl/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/gofhir/ecl.svg)](https://pkg.go.dev/github.com/gofhir/ecl)
[![Release](https://img.shields.io/github/v/release/gofhir/ecl)](https://github.com/gofhir/ecl/releases)

Embeddable parser and evaluator for the SNOMED CT **Expression Constraint Language (ECL) v2.2** in pure Go. Comes with parsers for **SNOMED Compositional Grammar (SCG)** and a **Machine Readable Concept Model (MRCM)** validator, plus a `gofhir-ecl` CLI and a 44-case conformance suite.

```go
import "github.com/gofhir/ecl/ecl"

ast, _   := ecl.Parse("<< 404684003 |Clinical finding| {{ D term = wild: \"Diabet*\" }}")
result, _ := ecl.Evaluate(ctx, ast, yourProvider)
// result.Slice() == []string{"73211009", ...}
```

## Why

ECL is the standard query language for SNOMED CT. Until now, evaluating it from Go meant standing up [Snowstorm](https://github.com/IHTSDO/snowstorm) (Java + Elasticsearch) over HTTP. `go-ecl` is the first Go-native implementation, designed to embed inside FHIR servers, ValueSet authoring tools, CDS pipelines, edge devices, and CI validators — without the JVM.

## Status

- ✅ **ECL v2.2 evaluator** — hierarchy, compound, refinements (with cardinality, reverse `R`, attribute groups), dot notation, filters (term/type/language/dialect/active/module/effectiveTime/definitionStatus/memberField), history supplements, concrete values (integer/decimal/string/boolean), Top/Bottom, MemberOf, RefsetContainingAny (`^R`), AltIdentifier
- ✅ **SCG** parser + validator
- ✅ **MRCM** loader + validator (uses ECL evaluator internally)
- ✅ **SCTID** Verhoeff checksum + partition validation
- ✅ **44/44** bundled conformance cases pass
- 📦 Latest release: see [GitHub Releases](https://github.com/gofhir/ecl/releases)

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

You implement [`ecl.DataProvider`](ecl/provider.go) against your storage (PostgreSQL closure tables, in-memory maps, Elasticsearch, an HTTP terminology server, …). The 18 methods are batch-shaped to avoid N+1 patterns and split into:

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

For tests and examples, the in-memory provider in [`internal/conformance/fixture.go`](internal/conformance/fixture.go) implements all 18 against a YAML fixture — read it to see the expected semantics.

### SCG + MRCM

```go
import (
    "github.com/gofhir/ecl/scg"
    "github.com/gofhir/ecl/mrcm"
)

scgExpr, _ := scg.Parse("404684003 : 363698007 = 74281007")
model, _   := mrcm.LoadModel("mrcm.json")
err := mrcm.NewValidator(model, provider).Validate(ctx, scgExpr)
```

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
  refinement: <1 ungrouped, 0 groups, 0 conjunction, 0 disjunction>
```

### `eval`

```bash
$ gofhir-ecl eval --fixture testdata/conformance/fixtures/standard.yaml "<< 73211009"
11687002
73211009
```

The fixture is a YAML file describing concepts, parents, descriptions, relationships, refsets, and history associations. See [`testdata/conformance/fixtures/standard.yaml`](testdata/conformance/fixtures/standard.yaml) for the schema.

### `conformance`

```bash
$ gofhir-ecl conformance
44 passed, 0 failed, 0 skipped, 44 total

$ gofhir-ecl conformance -filter '^memberOf'
PASS  ECL v2.2 features (Top, Bottom, AltIdentifier, ^R, MemberOf) :: memberOf refset

1 passed, 0 failed, 0 skipped, 1 total
```

Useful in CI to prove your `DataProvider` implementation matches the spec.

## Conformance suite

The bundled suite lives in [`testdata/conformance/`](testdata/conformance/) and currently covers 44 cases across 8 areas of the spec.

| Area | Cases | Spec section |
|---|---|---|
| Hierarchy operators | 8 | 5.1 |
| Compound expressions | 4 | 5.2 |
| Primitives | 4 | 5.0 |
| Refinements (groups, dot, reverse) | 6 | 5.3 |
| Filters (term/wild, language, negation) | 8 | 5.4 |
| History supplements | 3 | 5.5 |
| Concrete values | 4 | 5.3.4 |
| v2.2 (Top, Bottom, AltIdentifier, `^R`) | 7 | v2.2 |

Each suite is a YAML file with cases of the form:

```yaml
- name: descendantOrSelfOf root
  expression: "<< 138875005"
  expectedIds: ["138875005", "404684003", "22298006", ...]
```

Cases can also assert error paths via `expectError: true`. The runner is reusable as a Go package ([`internal/conformance`](internal/conformance/)) so you can drive it against your own fixtures or your own provider.

## Project layout

```
ecl/                  ECL parser, AST, evaluator, set, DataProvider interface
ecl/grammar/          Generated ANTLR4 grammar (do not edit)
ecl/ast/              AST node types
scg/                  Compositional Grammar (SCG) parser + validator
mrcm/                 MRCM loader + validator (uses ecl/ for constraints)
sctid/                SCTID Verhoeff checksum + partition validation
cmd/gofhir-ecl/       CLI binary
internal/conformance/ YAML fixture loader, in-memory provider, suite runner
testdata/conformance/ Bundled fixtures + cases
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

See `LICENSE`. (Add one if missing — the project does not yet declare a license.)

## See also

- [SNOMED CT ECL specification](https://confluence.ihtsdotools.org/display/DOCECL)
- [Snowstorm](https://github.com/IHTSDO/snowstorm) — the reference Java implementation; ECL grammar and edge-case behavior follow Snowstorm
- [FHIR Terminology Service](https://www.hl7.org/fhir/terminology-service.html) — `ValueSet/$expand` with `compose.include.filter` (`property=constraint, op==`) is the FHIR mechanism for using ECL
