# Releasing

Releases are automated by [release-please](https://github.com/googleapis/release-please).
Nobody tags by hand.

## The flow

1. Merge to `main` with [conventional commit](https://www.conventionalcommits.org)
   subjects. `.github/workflows/release.yml` runs release-please.
2. release-please opens (or updates) a **release PR** that bumps
   `.release-please-manifest.json` and writes `CHANGELOG.md` from the commit
   subjects since the last tag.
3. Merging that PR creates the tag and the GitHub Release.
4. A follow-up step in the same workflow links the hand-written notes into the
   Release body, if `docs/releases/<tag>.md` exists. See below.

## What bumps what

| Commit | Bump |
|---|---|
| `fix:`, `perf:`, `refactor:` | patch |
| `feat:` | **minor** |
| `feat!:`, `fix(scope)!:`, or a `BREAKING CHANGE:` footer | **MAJOR** |
| `docs:`, `test:`, `chore:`, `ci:`, `style:` | no release on their own |

### The two config options that do NOT protect you

`release-please-config.json` sets `bump-minor-pre-major` and
`bump-patch-for-minor-pre-major`. Both are **pre-major** options: they apply only
below `1.0.0`, and this module is past that. They are inert, and reading them as a
safeguard against a major version is a mistake.

The only thing standing between a stray `!` and `v2.0.0` is the commit message. For
a Go module, a major means a new import path (`/v2`) and a migration for every
consumer, so it is a decision, not a side effect.

The `major-bump` job in `ci.yml` therefore fails a pull request whose commits carry
a breaking-change marker. It is a prompt, not a prohibition: label the PR
`major-release` and it passes. That job exists because a `fix(ecl)!:` was written
out of habit in this repository for a change that was not breaking.

### The other way a major arrives

A clean set of commits is not enough, and this bit for real. Merging the v1.2.0
work produced a release PR titled `chore(main): release go-ecl 2.0.0`. The commits
carried no marker; the **configuration** did the damage.

`include-component-in-tag` defaults to `true` in manifest mode, so with
`package-name: go-ecl` release-please looked for a release tagged
`go-ecl-v1.1.0`. The real tags are `v1.1.0`. Finding no match it re-read the
history from the beginning and re-counted a `feat(ecl)!:` that had already shipped
in v1.0.0. The config now sets `include-component-in-tag: false`.

The lesson is where the check has to live. `major-bump` inspects a pull request's
commits and structurally cannot see this. So `release.yml` has a second guard
that reads what release-please **actually decided** — the version in the manifest
on its own branch — and fails when the major moves without a `major-release`
label. Check the decision, not the inputs to it.

A symptom worth recognising: the release PR's changelog listing commits from the
beginning of the project means release-please did not find the previous release,
and every already-shipped breaking change is back in scope.

## Hand-written release notes

release-please builds the Release body from commit **subjects**. That is enough for
a patch and nowhere near enough for a release that changes behaviour: a subject
cannot say what results move, why, or what to check before upgrading.

So a substantial release gets a file at `docs/releases/<tag>.md` —
[`v1.2.0.md`](releases/v1.2.0.md) is the worked example. Write it before merging
the release PR. The workflow reads the file from the tag and prepends a link to the
generated body; the link is pinned to the tag, so the notes always match the
version they describe. No file, no link, no failure.

What belongs in one:

- Every behaviour change, with the **before and after**, and how to tell whether
  you are affected. Someone with snapshot tests over ECL results needs to know
  which ones will move.
- The reasoning for anything that stays unsupported. A limitation without a reason
  reads as an oversight and gets re-reported.
- Measurements rather than adjectives. "27 calls → 1" survives review;
  "much faster" does not.

## Before merging the release PR

```bash
make check           # lint, -race with coverage, conformance, tidy, vet
make check-upstream  # the vendored SNOMED International artefacts vs upstream
make oracle          # differential test against a real terminology server
```

`make check` is the gate; the other two need network and report news rather than
defects, so they are deliberately outside it. Read them anyway before a release:
`check-upstream` is how you learn the grammar moved, and `oracle` is the only check
whose expectations this project did not write.

Also confirm the version release-please picked is the one the notes file is named
after. They are written by different people at different times, and a `feat:` that
lands late turns a patch into a minor.
