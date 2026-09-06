#!/bin/sh
# Generate deterministic release notes from git commits.
# Usage: .github/scripts/gen-changelog.sh [previous_tag]
set -eu

PREV_TAG="${1:-$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || true)}"
CURR_TAG="$(git describe --tags --abbrev=0 HEAD 2>/dev/null || echo HEAD)"

if [ -n "$PREV_TAG" ]; then
    RANGE="${PREV_TAG}..${CURR_TAG}"
else
    RANGE="${CURR_TAG}"
fi

printf '%s\n\n' "## Changes in ${CURR_TAG}"
if [ -n "$PREV_TAG" ]; then
    git log "$RANGE" --pretty=format:'- %s' --no-merges
else
    git log -50 --pretty=format:'- %s' --no-merges
fi
printf '\n'
