# go-ecl

[![CI](https://github.com/gofhir/ecl/actions/workflows/ci.yml/badge.svg)](https://github.com/gofhir/ecl/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/gofhir/ecl.svg)](https://pkg.go.dev/github.com/gofhir/ecl)
[![Release](https://img.shields.io/github/v/release/gofhir/ecl)](https://github.com/gofhir/ecl/releases)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Embeddable parser and evaluator for the SNOMED CT **Expression Constraint Language (ECL) v2.2** in pure Go. Comes with parsers for **SNOMED Compositional Grammar (SCG)** and a **Machine Readable Concept Model (MRCM)** validator, plus a `gofhir-ecl` CLI and a 136-case conformance suite.

```go
// The path repeats "ecl" because the module is github.com/gofhir/ecl and the
// package lives in ecl/. Correcting it would mean a new module path, i.e. a
// major version, so it stays.
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
- ✅ **MRCM** loader + validator (uses ECL evaluator internally) — domain, range, grouped, and both cardinalities including **per relationship group**
- ✅ **SCTID** Verhoeff checksum + partition validation
- ✅ **136/136** bundled conformance cases pass, all executed by CI
- ✅ **121/121** of SNOMED International's [official ECL examples](ecl/testdata/official-examples/) parse — the one test suite this project did not write
- ✅ **Fuzzed parser** with bounded input, so an expression from a URL cannot stall the process
- 📦 Latest release: see [GitHub Releases](https://github.com/gofhir/ecl/releases)

### Known limitations

Each of these returns `ecl.ErrUnsupportedFeature` rather than a silently wrong
result, so you can classify it with `errors.Is` and answer 501 instead of serving
bad data:

| Construct | Why |
|---|---|
| `{{ D term != … }}`, `{{ D language != … }}`, `{{ D type != … }}`, **unless** the provider implements `NegatingDescriptionProvider` | Negating a description filter is a per-ROW operation: a concept with both an FSN and a Spanish synonym satisfies `language != es` through its FSN, so set subtraction is wrong. Negated **concept** filters (`{{ C … != … }}`) always work. |
| `{{ D dialect = en-gb }}` (alias form), **unless** the provider implements `DialectAliasResolver` | Mapping a dialect alias to a language reference set's SCTID is terminology data — only the international English aliases are universal, and the same alias can name different refsets in different editions. With the capability this works; `{{ D dialectId = 900000000000508004 }}` always does. |
| `^[field]` projection | `Set` carries concept IDs only. Use a `{{ M … }}` member filter. |
| `{{ D id = … }}` | The parser models it, but `DescriptionFilterOpts` has no field to carry the ids to the provider. |
| A **negated** term filter with a set of terms — `{{ D term != ("a" "b") }}` | One description row has to fail every value, so it cannot be decomposed: intersecting per-term negations would accept a concept with one row failing the first term and a different row failing the second. The positive set form works. |
| A cardinality or `!=` on a reverse (`R`) attribute, **unless** the provider implements `InboundRelationshipsProvider` | `RelationshipTargets` returns a `Set`, so it loses the inbound count and the per-type total. With the capability these work. |
| A reverse (`R`) attribute **inside an attribute group** — `{ R attr = value }` | Braces assert that the clauses share one relationship group *of the focus concept*, and a reverse relationship belongs to the source, so the focus has no group to constrain. Write it outside the braces. See the note below. |
| An MRCM in-group cardinality **minimum** across groups that do not use the attribute | The specification says how many times an attribute *"can be"* assigned a value in a group, which settles the maximum and leaves the minimum open. Enforced only within groups that contain the attribute, so `1..1` behaves like `0..1`. See `validateInGroupCardinality` for why that direction was chosen. |
| Subsumption-**redundant** values collapsed before counting cardinality | Both cardinality fields count *"distinct (non-redundant)"* values. Distinctness is enforced; collapsing values where one subsumes the other needs subsumption testing, so it is left to the caller. It can only over-report, never under-report. |

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

Some things cannot be expressed through `DataProvider` as it stands, and widening
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
| `DialectAliasResolver` | `{{ D dialect = en-gb }}`, the alias form, whose alias-to-reference-set mapping is terminology data rather than something a parser can compute. |

The reference provider in [`ecl/providertest/fixture.go`](ecl/providertest/fixture.go)
implements all five; read it for a worked example. `providertest.VerifyContract`
has a check per capability, asserting that it agrees with the required method it
replaces — a batch that disagrees with its per-concept equivalent would silently
decide every answer.

#### A note on reverse attributes inside a group

`{ R attr = value }` returns `ErrUnsupportedFeature`. Write `R attr = value`
without the braces instead.

The reason is that grouping only ever says one thing: *these clauses hold in the
same relationship group of the focus concept*. That is what distinguishes a heart
infarct from a concept carrying heart-site in one group and infarct-morphology in
another. A reverse clause's relationship is not the focus concept's — it belongs to
the source — so the focus has nothing to group, and only two readings remain:

- `{ R attr = value }` on its own: the braces add nothing. Measured on real SNOMED
  before this release, the result was identical to the ungrouped form.
- `{ R attr = value, other = x }`: the group has to belong to *someone*, and it can
  only be the source, so `other = x` silently constrains the **source** rather than
  the focus. `* : { R 363698007 = 22298006, 116676008 = 55641003 }` returned
  74281007, which has no relationships of its own — what was checked was
  22298006's morphology.

Redundant or misleading, never useful. The specification neither permits nor
forbids it — §6.2 describes reverse attributes and attribute groups separately,
§6.3 defines reverse cardinality only for an ungrouped refinement, and the
[official example set](https://github.com/IHTSDO/snomed-expression-constraint-language)
never places `R` inside braces — and Ontoserver rejects it with *"Cannot reverse an
attribute inside a group"*. This library now agrees.

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

`providertest.VerifyFixture(t)` is the other half. It runs the 136 bundled cases
against the bundled fixture, so it verifies the **evaluator** rather than your
provider: 107 of them pin concrete concept IDs, which a correct provider carrying
different data cannot satisfy. Use it to check this library, not your storage.

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

The validator reports seven kinds of issue: `domain_violation`,
`range_violation`, `cardinality_violation`, `in_group_cardinality_violation`,
`grouped_violation`, `ungrouped_violation`, `unknown_attribute` — plus
`invalid_rule` for a rule in the *model* that could not be applied, which is an
issue rather than an error so one broken rule cannot hide the violations already
found.

The two cardinalities constrain different things and an expression can satisfy one
while breaking the other. Three finding sites spread over three groups are fine
under a concept cardinality of `0..*` *and* an in-group cardinality of `0..1`; the
same three in one group break the second only. Both count **distinct values**, per
the specification's *"distinct (non-redundant) value"* — the same value asserted
twice is one value, not two.

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
136 passed, 0 failed, 0 skipped, 136 total

$ gofhir-ecl conformance -filter '^memberOf'
PASS  ECL v2.2 features (Top, Bottom, AltIdentifier, ^R, MemberOf) :: memberOf refset

1 passed, 0 failed, 0 skipped, 1 total
```

Useful in CI to prove your `DataProvider` implementation matches the spec.

## Testing against the specification

Three different things get called "spec compliance", and they are worth keeping
apart:

| | What it proves | Who wrote the expectations |
|---|---|---|
| [`ecl/grammar/ECL.g4`](ecl/grammar/ECL.g4) | The syntax accepted is the published grammar. It is SNOMED International's file, carrying one local fix for an upstream typo that stops the grammar generating (documented in its header). CI regenerates the parser from it and diffs byte for byte. | SNOMED International |
| [`ecl/testdata/official-examples/`](ecl/testdata/official-examples/) | The 121 expressions upstream publishes as **valid ECL** all parse. `TestParse_OfficialExamples` walks the tree; a failure is this project's defect by construction, since upstream declares them valid. | SNOMED International |
| [`ecl/providertest/testdata/`](ecl/providertest/testdata/) | 136 expressions **evaluate** to specific concept sets against a bundled fixture. | This project |

The third row was the honest weak spot: no official corpus states what an
expression should *return* — the specification gives prose and examples without
expected results — so those expectations are this project's reading of it, and a
misreading yields a green test. Several bugs fixed in v1.2.0 had a passing test
asserting the wrong behaviour.

`make oracle` is the answer to that. It runs a **differential test** against a real
FHIR terminology server ([`internal/oracle`](internal/oracle/)): the server
supplies the facts — the descendants of X, the targets of attribute A on set S, the
relationship groups of a concept — and also evaluates the whole expression itself,
so the only thing under test is what this library composes out of those facts.
That is where every semantic bug found so far actually lived.

```
$ make oracle
28 agreed (0 of them vacuously, on the empty set), 1 skipped, of 29 cases;
60 HTTP requests in 28s
```

The one skip is reverse cardinality, which needs `InboundRelationshipsProvider`;
the harness does not implement it, so this library correctly reports the construct
unsupported instead of guessing. Agreement on an empty set is counted separately
and never as evidence — two implementations that both return nothing agree for any
reason at all.

It is not part of `make check`: it needs network, and a divergence needs triage
rather than a red build. The server is another implementation, not the
specification.

`make check-upstream` diffs both vendored artefacts against upstream, and a
scheduled workflow runs it weekly — otherwise a new grammar version could land
without anyone noticing.

## Parsing untrusted input

`Parse` is usually reached from a URL — a `?ecl=` parameter, a ValueSet
`compose.include.filter` — so it is treated as a hostile-input boundary.

**Two-stage parsing.** ANTLR's default ALL(\*) prediction is exact but, on this
grammar, quadratic. Measured before the change:

| expression | size | before | after |
|---|---|---|---|
| 3 clauses (the size real queries are) | 170 B | 31 ms | **0.09 ms** |
| 50 refinement clauses | 1.1 KB | 4.5 s | **0.6 ms** |
| 100 refinement clauses | 2.2 KB | 7.6 s | **1.1 ms** |
| 200 refinement clauses | 4.4 KB | 27.5 s | **2.1 ms** |

A long conjunction is not adversarial — a query builder emits them — and it was
seconds. Allocations for the 100-clause case went from 92 million to 18 thousand.
`Parse` now tries linear SLL prediction first and falls back to full ALL(\*) only
for input SLL cannot decide, which produces the identical tree and the identical
error messages.

**Bounded input.** `ecl.MaxInputBytes` (1 MiB) and `ecl.MaxNestingDepth` (100) are
checked before parsing starts. They exist because ANTLR offers no way to interrupt
a parse in progress: a `context` deadline would let a caller stop *waiting* while
the goroutine kept burning CPU. Nesting depth is the one axis that stayed
superlinear, so it is capped rather than fixed — at depth 100 a parse costs 0.5 ms,
and the deepest expression among the 121 official examples and 136 conformance
cases nests **4** levels. Over the limit you get a `*ecl.ParseError`, the same type
as a syntax error.

**Fuzzed.** `FuzzParse` asserts that no input panics, that a nil tree never comes
back with a nil error, and that no parse runs away. Run it with `make fuzz`; CI
runs a 60-second pass per pull request and uploads any crasher. It earned its keep
twelve seconds after being written by finding `* {{ D term = "C:\temp" }}` — 26
bytes, an invalid escape someone types by accident — which grew the heap past 5 GB
without bound. Crashers are committed under
[`ecl/testdata/fuzz/`](ecl/testdata/fuzz/) and replay as ordinary tests.

## Conformance suite

The bundled suite lives in [`ecl/providertest/testdata/`](ecl/providertest/testdata/) and is **embedded in the binary**, so `gofhir-ecl conformance` works from any directory, including after `go install`. It currently covers 136 cases across 9 areas of the spec, including a suite of expressions that must be REJECTED.

| Area | Cases | Spec section |
|---|---|---|
| Hierarchy operators (incl. Top/Bottom on non-closed sets) | 10 | 5.1 |
| Compound expressions | 4 | 5.2 |
| Primitives | 4 | 5.0 |
| Refinements (disjunction, groups, cardinality, `!=`, dot, reverse) | 31 | 5.3 |
| Filters (term, type, language, dialect and dialectId, module, definitionStatus, active, effectiveTime, memberField) | 53 | 5.4 |
| History supplements (MIN/MOD/MAX) | 6 | 5.5 |
| Concrete values | 4 | 5.3.4 |
| v2.2 (Top, Bottom, AltIdentifier, `^R`) | 7 | v2.2 |
| **Errors** — input that must be rejected, not truncated | 17 | grammar |

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

Semantic versioning, driven by [release-please](https://github.com/googleapis/release-please)
from Conventional Commits — see [docs/RELEASING.md](docs/RELEASING.md).

Widening `DataProvider` would be a breaking change, so it does not happen: new
capabilities arrive as [optional interfaces](#optional-capabilities) the evaluator
type-asserts for, which is why v1.2.0 delivered eight fixes and five new
capabilities without a major. For a Go module a major means a new import path and a
migration for every consumer, so it is a decision rather than a side effect, and CI
fails a pull request that would trigger one unless it is labelled `major-release`.

## Contributing

Issues and PRs welcome. Before submitting:

```bash
make check           # lint, -race with coverage, conformance, tidy, vet
make check-upstream  # optional: the vendored SNOMED artefacts vs upstream
make oracle          # optional: differential test against a real server
```

The conformance suite is the source of truth for expected behavior — extending the
suite is the cheapest way to lock in a fix or land a new ECL feature. If a change
touches semantics, `make oracle` is worth the 30 seconds: it is the only check
whose expectations this project did not write.

## License

[Apache License 2.0](LICENSE) — Copyright © 2026 Roberto Araneda Espinoza.

## See also

- [SNOMED CT ECL specification](https://confluence.ihtsdotools.org/display/DOCECL)
- [Snowstorm](https://github.com/IHTSDO/snowstorm) — the reference Java implementation; ECL grammar and edge-case behavior follow Snowstorm
- [FHIR Terminology Service](https://www.hl7.org/fhir/terminology-service.html) — `ValueSet/$expand` with `compose.include.filter` (`property=constraint, op==`) is the FHIR mechanism for using ECL
