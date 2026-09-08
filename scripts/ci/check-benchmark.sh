#!/bin/sh
set -eu

baseline="${1:-benchmarks/baseline.json}"
current="${2:-benchmarks/current.txt}"
threshold="${BENCHMARK_MAX_REGRESSION:-5}"

[ -s "$baseline" ] || { echo "benchmark baseline is missing: $baseline" >&2; exit 1; }
[ -s "$current" ] || { echo "benchmark output is missing: $current" >&2; exit 1; }

python3 - "$baseline" "$current" "$threshold" <<'PY'
import json
import re
import statistics
import sys

baseline_path, current_path, threshold_text = sys.argv[1:]
threshold = float(threshold_text)
with open(baseline_path, encoding="utf-8") as handle:
    baseline = json.load(handle)

if "benchmarks" in baseline:
    baseline = baseline["benchmarks"]

pattern = re.compile(r"^(?P<name>\S+)\s+(?P<iterations>\d+)\s+(?P<ns>[0-9.]+)\s+ns/op")
measurements = {}
with open(current_path, encoding="utf-8") as handle:
    for raw_line in handle:
        match = pattern.match(raw_line.strip())
        if match:
            measurements.setdefault(match.group("name").split("-")[0], []).append(float(match.group("ns")))

if not measurements:
    raise SystemExit("benchmark output contains no parseable benchmark lines")

failures = []
for name, expected in baseline.items():
    samples = measurements.get(name)
    if not samples:
        raise SystemExit("benchmark output is missing: " + name)
    expected_ns = float(expected["ns_per_op"])
    if expected_ns <= 0:
        raise SystemExit(f"baseline ns_per_op must be positive: {name}")
    observed_ns = statistics.median(samples)
    regression = (observed_ns / expected_ns - 1) * 100
    print(f"{name}: {observed_ns:.2f} ns/op vs {expected_ns:.2f} ns/op ({regression:+.2f}%)")
    if regression > threshold:
        failures.append(f"{name} {regression:.2f}% > {threshold:.2f}%")

if failures:
    raise SystemExit("benchmark regression gate failed: " + "; ".join(failures))
PY
