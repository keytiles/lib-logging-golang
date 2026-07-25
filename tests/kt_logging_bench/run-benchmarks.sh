#!/usr/bin/env bash
# Runs kt_logging emit benchmarks and prints a clear per-scenario summary at the end.
#
# Defaults: -benchtime=3s (about 3s of work per scenario). No -count by default —
# re-run the script a few times for a feel; use -count=N when you want averaged rows.
#
# Usage (from anywhere):
#   ./tests/kt_logging_bench/run-benchmarks.sh
#   ./tests/kt_logging_bench/run-benchmarks.sh -count=3
#   ./tests/kt_logging_bench/run-benchmarks.sh -benchtime=5s
#
# Extra args are passed through to go test.
#
# Unit tests live in ./tests/kt_logging/ — this package is benchmarks only.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${REPO_ROOT}"

BENCH_PKG="./tests/kt_logging_bench/"
RAW_LOG="$(mktemp -t kt-logging-bench.XXXXXX.log)"
trap 'rm -f "${RAW_LOG}"' EXIT

# Default bench duration; skipped if the caller already passes -benchtime=...
BENCHTIME_ARGS=(-benchtime=3s)
for arg in "$@"; do
  case "${arg}" in
    -benchtime=*) BENCHTIME_ARGS=() ;;
  esac
done

echo "Running emit benchmarks in ${BENCH_PKG} ..."
echo "(benchmarks-only package; default ${BENCHTIME_ARGS[*]:-(custom -benchtime from args)}; full log → ${RAW_LOG})"
echo

# -v so --- SKIP lines (and skip reasons) appear in the log for the summary.
# -run=^$ is still set so any future Test* in this package would not run with benches.
go test -v -run='^$' -bench='BenchmarkEmit' -benchmem "${BENCHTIME_ARGS[@]}" "${BENCH_PKG}" "$@" | tee "${RAW_LOG}"

echo
echo "============================================================"
echo " Benchmark summary (one row per scenario)"
echo "============================================================"

# Collect samples: space-separated lists per scenario (supports -count=N → mean).
declare -A SAMPLES_NS=()
declare -A SAMPLES_B=()
declare -A SAMPLES_A=()

while IFS= read -r line; do
  case "${line}" in
    BenchmarkEmit_*)
      ;;
    *)
      continue
      ;;
  esac
  if [[ "${line}" != *"ns/op"* ]]; then
    continue
  fi

  name="$(echo "${line}" | awk '{print $1}' | sed -E 's/-[0-9]+$//')"
  ns="$(echo "${line}" | sed -nE 's/.*[[:space:]]([0-9.]+) ns\/op.*/\1/p')"
  bytes="$(echo "${line}" | sed -nE 's/.*[[:space:]]([0-9.]+) B\/op.*/\1/p')"
  allocs="$(echo "${line}" | sed -nE 's/.*[[:space:]]([0-9.]+) allocs\/op.*/\1/p')"
  scenario="${name#BenchmarkEmit_}"

  SAMPLES_NS["${scenario}"]="${SAMPLES_NS[${scenario}]:-} ${ns}"
  SAMPLES_B["${scenario}"]="${SAMPLES_B[${scenario}]:-} ${bytes}"
  SAMPLES_A["${scenario}"]="${SAMPLES_A[${scenario}]:-} ${allocs}"
done < "${RAW_LOG}"

mean_of() {
  # $1 = space-separated numbers → printed mean (1 decimal), or empty if none
  echo "$1" | awk '{
    s=0; n=0;
    for (i=1; i<=NF; i++) { s+=$i; n++ }
    if (n>0) printf "%.1f", s/n
  }'
}

sample_count_of() {
  echo "$1" | awk '{print NF}'
}

# scenario -> skip reason (from "--- SKIP:" + preceding t.Skip message)
declare -A SKIPPED=()
prev_reason=""
while IFS= read -r line; do
  if [[ "${line}" =~ benchmark_test\.go:[0-9]+:[[:space:]]*(.*)$ ]]; then
    prev_reason="${BASH_REMATCH[1]}"
    continue
  fi
  if [[ "${line}" =~ ^---\ SKIP:\ BenchmarkEmit_([A-Za-z0-9_]+) ]]; then
    scenario="${BASH_REMATCH[1]}"
    if [[ -n "${prev_reason}" ]]; then
      SKIPPED["${scenario}"]="${prev_reason}"
    else
      SKIPPED["${scenario}"]="skipped"
    fi
    prev_reason=""
  fi
done < "${RAW_LOG}"

# Detect if any scenario has multiple samples (i.e. -count>1)
MAX_SAMPLES=1
for scenario in "${!SAMPLES_NS[@]}"; do
  c="$(sample_count_of "${SAMPLES_NS[${scenario}]}")"
  if (( c > MAX_SAMPLES )); then
    MAX_SAMPLES="${c}"
  fi
done
if (( MAX_SAMPLES > 1 )); then
  echo "(ns/op, B/op, allocs/op are means over ${MAX_SAMPLES} runs per scenario)"
fi
echo
printf "%-28s  %12s  %10s  %12s  %s\n" "SCENARIO" "ns/op" "B/op" "allocs/op" "CONFIG / NOTE"
printf "%-28s  %12s  %10s  %12s  %s\n" "----------------------------" "------------" "----------" "------------" "-------------"

# Fixed order matching benchmark_test.go — always print every scenario.
SCENARIOS=(FileJson StdoutJson StdoutPlain FilePlain FileRollingJson)

config_for() {
  case "$1" in
    FileJson)        echo "testdata/bench-handler-file-json.yaml" ;;
    StdoutJson)      echo "testdata/bench-handler-stdout-json.yaml" ;;
    StdoutPlain)     echo "testdata/bench-handler-stdout-plain.yaml" ;;
    FilePlain)       echo "testdata/bench-handler-file-plain.yaml" ;;
    FileRollingJson) echo "testdata/bench-handler-file-rolling-json.yaml" ;;
    *)               echo "?" ;;
  esac
}

for scenario in "${SCENARIOS[@]}"; do
  config="$(config_for "${scenario}")"
  if [[ -n "${SAMPLES_NS[${scenario}]+x}" ]]; then
    ns="$(mean_of "${SAMPLES_NS[${scenario}]}")"
    bytes="$(mean_of "${SAMPLES_B[${scenario}]}")"
    allocs="$(mean_of "${SAMPLES_A[${scenario}]}")"
    printf "%-28s  %12s  %10s  %12s  %s\n" "${scenario}" "${ns}" "${bytes}" "${allocs}" "${config}"
  elif [[ -n "${SKIPPED[${scenario}]+x}" ]]; then
    printf "%-28s  %12s  %10s  %12s  %s\n" "${scenario}" "SKIPPED" "-" "-" "${config} — ${SKIPPED[${scenario}]}"
  else
    if [[ "${scenario}" == "FileRollingJson" && "$(go env GOOS)" == "windows" ]]; then
      printf "%-28s  %12s  %10s  %12s  %s\n" "${scenario}" "SKIPPED" "-" "-" "${config} — rolling file not reliable on Windows (see CHANGELOG)"
    else
      printf "%-28s  %12s  %10s  %12s  %s\n" "${scenario}" "?" "-" "-" "${config} — no result in go test output"
    fi
  fi
done

echo
echo "Done."
