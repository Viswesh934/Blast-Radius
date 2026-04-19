#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BASELINE="snapshots/sources/my_service/analytics/public/snapshot_1776536091.json"
CURRENT="snapshots/sources/my_service/analytics/public/snapshot_1776536138.json"

FAIL_JSON="artifacts/release-check-fail.json"
FAIL_MD="artifacts/release-check-fail.md"
PASS_JSON="artifacts/release-check-pass.json"
PASS_MD="artifacts/release-check-pass.md"

echo "== Build binary =="
go build -o bin/blast ./cmd

mkdir -p artifacts

echo
echo "== Demo 1: strict policy (expected FAIL) =="
set +e
./bin/blast release-check \
  --baseline "$BASELINE" \
  --current "$CURRENT" \
  --profile profiles/guard/analytics.guard.yaml \
  --report-file "$FAIL_JSON" \
  --markdown-file "$FAIL_MD"
FAIL_EXIT=$?
set -e

if [[ "$FAIL_EXIT" -ne 0 ]]; then
  echo "Result: FAIL path works (non-zero exit: $FAIL_EXIT)"
else
  echo "Result: unexpected PASS in strict policy"
fi

echo
echo "Strict policy summary:"
sed -n '1,40p' "$FAIL_MD"

echo
echo "== Demo 2: relaxed policy (expected PASS) =="
./bin/blast release-check \
  --baseline "$BASELINE" \
  --current "$CURRENT" \
  --min-score 0 \
  --require-table-description=false \
  --max-risk high \
  --report-file "$PASS_JSON" \
  --markdown-file "$PASS_MD"

echo "Result: PASS path works (zero exit)"

echo
echo "Relaxed policy summary:"
sed -n '1,40p' "$PASS_MD"

echo
echo "Artifacts generated:"
echo "- $FAIL_JSON"
echo "- $FAIL_MD"
echo "- $PASS_JSON"
echo "- $PASS_MD"
