# Effort 2 Checklist: Release Gate + MCP Operationalization

Goal: move Blast Radius from wrapper prototype to a CI/CD-ready tool with a demoable operator workflow.

## Scope For Today

- Stabilize `release-check` as the primary CI gate command.
- Validate deterministic fail/pass behavior from snapshot files.
- Keep MCP demo path operational for agent/tooling integrations.
- Add a GitHub Actions gate workflow template.
- Produce demo commands anyone can run locally.

## Success Criteria

- `release-check` returns non-zero on policy violations.
- `release-check` returns zero when policy thresholds allow release.
- JSON and Markdown artifacts are produced for both outcomes.
- Workflow file exists and references required secrets/variables.
- Demo scripts run without manual code edits.

## Demo Runbook

### 1. Build

- `go build -o bin/blast ./cmd`

### 2. Strict policy (expected FAIL)

- `./bin/blast release-check --baseline snapshots/sources/my_service/analytics/public/snapshot_1776536091.json --current snapshots/sources/my_service/analytics/public/snapshot_1776536138.json --profile profiles/guard/analytics.guard.yaml --report-file artifacts/release-check-fail.json --markdown-file artifacts/release-check-fail.md`

Expected:
- Exit code non-zero.
- `artifacts/release-check-fail.json` and `artifacts/release-check-fail.md` exist.

### 3. Relaxed policy (expected PASS)

- `./bin/blast release-check --baseline snapshots/sources/my_service/analytics/public/snapshot_1776536091.json --current snapshots/sources/my_service/analytics/public/snapshot_1776536138.json --min-score 0 --require-table-description=false --max-risk high --report-file artifacts/release-check-pass.json --markdown-file artifacts/release-check-pass.md`

Expected:
- Exit code zero.
- `artifacts/release-check-pass.json` and `artifacts/release-check-pass.md` exist.

### 4. One-command stage demo

- `./examples/release_gate_demo.sh`

Expected:
- Script prints both FAIL and PASS outcomes.
- Script shows human-readable summaries.

### 5. MCP smoke demo (optional)

- `python3 examples/mcp_demo.py --newer /workspaces/Blast-Radius/snapshots/sources/my_service/analytics/public`

Expected:
- Initialize + tools/list + ingest + compare + impact complete.

## CI/CD Wiring Checklist

- Workflow exists: `.github/workflows/metadata-release-gate.yml`.
- Secrets configured in GitHub repo:
  - `OM_BASE_URL`
  - `OM_JWT_TOKEN`
- Variables configured in GitHub repo:
  - `BR_DATABASE_FQN`
  - `BASELINE_SNAPSHOT`

## Out Of Scope For Effort 2

- Packaging/distribution (brew/winget/docker release automation).
- Hosted dashboard.
- AI explanation features.

## Exit Definition

Effort 2 is complete when a teammate can clone the repo, run the stage demo, and explain:

- why a release was blocked,
- how to tune policy for controlled rollout,
- and how the same command plugs into CI.
