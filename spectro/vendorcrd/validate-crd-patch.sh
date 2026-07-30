#!/usr/bin/env bash
# Validate that spectro/vendorcrd/crd-patch.yaml covers every fork-only CRD schema addition.
#
# The fork-only additions are `git diff <upstream-tag> -- config/crd/bases/` (the fork's generated
# CRDs vs the upstream release tag). Every added property name and enum value must be referenced in
# crd-patch.yaml, or palette vendorcrd-sync would silently drop that fork field on the next CAPZ bump.
# Run after `make generate` on every CAPZ upgrade; update crd-patch.yaml until this passes.
#
# Usage: spectro/vendorcrd/validate-crd-patch.sh [upstream-tag]   (default: v1.26.0)
set -euo pipefail

TAG="${1:-v1.26.0}"
PATCH="spectro/vendorcrd/crd-patch.yaml"
cd "$(git rev-parse --show-toplevel)"

[ -f "$PATCH" ] || { echo "FAIL: $PATCH not found"; exit 1; }

diff="$(git diff "$TAG" -- config/crd/bases/)"
[ -n "$diff" ] || { echo "FAIL: no fork delta vs $TAG (wrong tag, or CRDs not regenerated with 'make generate')"; exit 1; }

missing=0

# Added property names: '+  <name>:' (skip generic schema keys that always appear in the patch).
while IFS= read -r key; do
  case "$key" in type|description|properties|items|enum|""|format|default) continue;; esac
  grep -q "/$key" "$PATCH" || grep -qE "(^| )$key:" "$PATCH" || { echo "MISSING property in $PATCH: $key"; missing=1; }
done < <(printf '%s\n' "$diff" | grep -E '^\+[[:space:]]+[a-zA-Z][a-zA-Z0-9]*:' | sed -E 's/^\+[[:space:]]+([a-zA-Z0-9]+):.*/\1/' | sort -u)

# Added enum scalar values: '+  - <value>'.
while IFS= read -r val; do
  [ -n "$val" ] || continue
  grep -qE "value: $val( |$)" "$PATCH" || { echo "MISSING enum value in $PATCH: $val"; missing=1; }
done < <(printf '%s\n' "$diff" | grep -E '^\+[[:space:]]+- [a-z][a-z0-9-]*$' | sed -E 's/^\+[[:space:]]+- //' | sort -u)

if [ "$missing" -ne 0 ]; then
  echo "crd-patch.yaml is INCOMPLETE vs $TAG — add the missing JSON6902 ops."
  exit 1
fi
echo "OK: crd-patch.yaml covers all fork-only CRD additions vs $TAG"
