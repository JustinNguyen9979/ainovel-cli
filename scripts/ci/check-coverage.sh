#!/bin/sh
set -eu

profile="${1:-coverage.out}"
minimum="${COVERAGE_MINIMUM:-80.0}"

if [ ! -s "$profile" ]; then
  echo "coverage profile is missing or empty: $profile" >&2
  exit 1
fi

actual="$(go tool cover -func="$profile" | awk '/^total:/ { print $3 }' | tr -d '%')"
if [ -z "$actual" ]; then
  echo "unable to read total coverage from $profile" >&2
  exit 1
fi

awk -v actual="$actual" -v minimum="$minimum" 'BEGIN {
  if (actual + 0 < minimum + 0) {
    printf "coverage %.1f%% is below required %.1f%%\n", actual, minimum
    exit 1
  }
  printf "coverage %.1f%% meets required %.1f%%\n", actual, minimum
}'
