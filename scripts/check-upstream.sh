#!/usr/bin/env bash
#
# Reports drift between the vendored SNOMED International artefacts and upstream.
#
# Two things in this repository are copies of files SNOMED International owns:
#
#   ecl/grammar/ECL.g4              <- syntax/ECL.g4
#   ecl/testdata/official-examples/ <- examples/
#
# Neither has any mechanism that would notice upstream moving. The regeneration
# gate in ci.yml regenerates from OUR copy of the grammar, so it proves
# reproducibility, not fidelity; and the example corpus is a plain file tree.
# This script closes that gap. It is advisory by nature — upstream changing is
# news, not a defect — so run it on a schedule, not on every pull request.
#
# Exit codes: 0 in sync, 1 drift found, 2 could not check.
set -uo pipefail

UPSTREAM_URL="https://github.com/IHTSDO/snomed-expression-constraint-language.git"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUR_GRAMMAR="$REPO_ROOT/ecl/grammar/ECL.g4"
OUR_EXAMPLES="$REPO_ROOT/ecl/testdata/official-examples"

# The single deliberate local fix, as an exact diff hunk. Upstream references the
# rule as `memberOf` on line 13 but defines it as `memberof` on line 14, so the
# grammar does not generate as published. See the header of ECL.g4.
readonly EXPECTED_PATCH='< refsetOperator : memberOf | refsetContainingAny;
> refsetOperator : memberof | refsetContainingAny;'

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "==> cloning $UPSTREAM_URL"
if ! git clone -q --depth 1 "$UPSTREAM_URL" "$tmp/upstream"; then
	echo "error: could not clone upstream; nothing was checked" >&2
	exit 2
fi
pinned_commit="$(git -C "$tmp/upstream" rev-parse HEAD)"
echo "    upstream HEAD is $pinned_commit"

drift=0

# ── Grammar ──────────────────────────────────────────────────────────────────
# Our copy carries a provenance header that upstream does not have, so strip the
# leading comment block before diffing. Only the leading block: the grammar body
# has no comments, and stripping every // line would hide a real change.
echo "==> ECL.g4"
awk 'BEGIN{h=1} h && (/^\/\// || /^[[:space:]]*$/) {next} {h=0; print}' \
	"$OUR_GRAMMAR" >"$tmp/ours.g4"

actual_patch="$(diff "$tmp/upstream/syntax/ECL.g4" "$tmp/ours.g4" | grep -E '^[<>]' || true)"
if [[ "$actual_patch" == "$EXPECTED_PATCH" ]]; then
	echo "    in sync (carrying the one known local fix)"
else
	echo "    DRIFT: the diff against upstream is no longer just the memberof fix." >&2
	echo "    Expected:" >&2
	echo "$EXPECTED_PATCH" | sed 's/^/      /' >&2
	echo "    Got:" >&2
	echo "${actual_patch:-      (identical — the local fix is gone; did upstream fix it?)}" | sed 's/^/      /' >&2
	drift=1
fi

# Whether the recorded provenance commit still matches is separate from whether
# the content matches: content can be identical while the pin is stale, which is
# only a bookkeeping problem, so it is reported without setting drift.
if ! grep -q "$pinned_commit" "$OUR_GRAMMAR"; then
	echo "    note: the commit recorded in the ECL.g4 header is not upstream HEAD."
	echo "          Content above is what matters; refresh the pin when convenient."
fi

# ── Examples ─────────────────────────────────────────────────────────────────
echo "==> examples/"
if diff -qr "$tmp/upstream/examples" "$OUR_EXAMPLES" \
	--exclude=PROVENANCE.md >"$tmp/examples.diff" 2>&1; then
	echo "    in sync ($(find "$OUR_EXAMPLES" -name '*.txt' | wc -l | tr -d ' ') files)"
else
	echo "    DRIFT:" >&2
	sed 's/^/      /' "$tmp/examples.diff" >&2
	echo >&2
	echo "    Re-copy the tree, update PROVENANCE.md and wantOfficialExamples, and run" >&2
	echo "    the tests. A new example that does not parse is a finding: do not delete" >&2
	echo "    the file to make CI green." >&2
	drift=1
fi

exit "$drift"
