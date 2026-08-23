# Official ECL examples — vendored, do not edit

Every `.txt` file under this directory is a **verbatim copy** from the SNOMED
International reference repository for the Expression Constraint Language:

| | |
|---|---|
| Source | <https://github.com/IHTSDO/snomed-expression-constraint-language> |
| Path | `examples/` |
| Commit | `b0e07105ae395821bcc953f3d6084b57dc7bef2c` (2026-01-21) |
| Licence | Apache License 2.0 — the same licence this repository uses |
| Copyright | SNOMED International (IHTSDO) |

## Why they are here

They are the closest thing to a **spec-authored** test corpus that exists for
ECL. The specification itself gives prose and examples without expected results,
so it cannot settle a semantic question; what this corpus does settle is the
*syntax*: these 121 expressions are declared valid by the body that defines the
language, so every one of them must parse. That is a claim nobody on this side of
the fence gets to interpret.

`TestParse_OfficialExamples` in [`../../parse_official_test.go`](../../parse_official_test.go)
walks this tree and parses each file as a single expression (they are one
expression each, some spanning several lines). It is the only test in the repo
whose expectations were not written here.

What it does **not** cover: semantics. No file states what a given expression
should return, so a build that parses all 121 and evaluates every one of them
wrongly would still be green. The bundled conformance suite in
`ecl/providertest/testdata/` covers evaluation, and its expectations are this
project's reading of the spec, not SNOMED International's.

## Updating

Re-copy the whole `examples/` tree, update the commit hash above, and run the
tests. A new file that fails to parse is a genuine finding — either the grammar
moved or our parser has a gap; do not delete the file to make CI green.

The grammar in `ecl/grammar/ECL.g4` comes from `syntax/ECL.g4` of the same
repository and carries one deliberate local fix, documented in its header. The
`grammar-upstream` CI workflow watches for drift.
