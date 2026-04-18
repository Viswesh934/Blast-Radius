# Metadata Drift Guard + Contract Readiness Checker

This repository now includes guard commands that support a local-first workflow and treat OpenMetadata as an optional feature.

## Why This Matches Your Direction

The core product can run from saved snapshots and CI checks.
OpenMetadata is used as an enrichment source through `--source` when you want live metadata.

That means the product is not only an OpenMetadata wrapper.
It is a metadata quality and release-safety layer that can plug into OpenMetadata.

## Feature Set Implemented

- Drift Guard:
  - `blast guard drift <baseline.json> <current.json>`
  - `blast guard drift <baseline.json> --source service.database.schema`
  - Policies: `--fail-on`, `--max-total`
  - Exit code fails CI when policy fails
- Contract Readiness Checker:
  - `blast guard contracts [snapshot.json]`
  - `blast guard contracts --source service.database.schema`
  - Policies: `--min-score`, `--require-owner`, `--require-table-description`
  - Exit code fails CI when score threshold fails

## Mapping To Paradox Tracks

### Paradox #T-01 MCP Ecosystem & AI Agents

- Expose guard commands through an MCP server as tools.
- Add an AI assistant flow that explains guard failures and suggests fix patches.
- Add natural language query mode over snapshot files, with optional OpenMetadata context.

### Paradox #T-02 Data Observability

- Turn guard outputs into trend metrics (readiness score over time, drift frequency).
- Add anomaly detection for sudden spikes in critical drift.
- Build a dashboard from guard JSON outputs.

### Paradox #T-03 Connectors & Ingestion

- Add ingestion from other metadata formats into `snapshot.StateSnapshot`.
- Build adapters for warehouse catalogs and schema registries.
- Keep OpenMetadata as one adapter among many.

### Paradox #T-04 Developer Tooling & CI/CD

- Use guard commands in GitHub Actions and pull-request checks.
- Gate deploys by contract score and severity policy.
- Add pre-merge bot comments with failed entities.

### Paradox #T-05 Community & Comms Apps

- Post guard failures to Slack/Teams.
- Publish daily readiness status by domain/team.
- Route ownership-based notifications for contract regressions.

### Paradox #T-06 Governance & Classification

- Add policy checks for ownership, descriptions, PII flags, tags, and glossary coverage.
- Add control packs (SOX, HIPAA, PCI templates) as guard profiles.
- Combine readiness score with governance controls for release approvals.

## Near-Term Extension Plan

1. Add guard profile files (`guard.yaml`) for reusable policy bundles.
2. Add baseline auto-discovery by source and timestamp.
3. Add rule plugins (ownership, tags, glossary, lineage completeness).
4. Add MCP server wrapper for AI-agent execution.
5. Add GitHub Action to run drift and contracts on every PR.
