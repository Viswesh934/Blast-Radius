# Effort 3 Checklist: AI-Native Web Ingestion + Drift Operations

Goal: make Blast Radius agent-ready for real-world, continuous metadata change detection from public and partner web sources.

## Scope

- Expose end-to-end web ingestion workflow through MCP.
- Expose OpenMetadata service discovery through MCP.
- Keep CLI and MCP workflows behaviorally aligned.
- Document practical AI usage patterns and business outcomes.

## New MCP Tools

- `blast.openmetadata.services.list`
  - Lists available database services in OpenMetadata.
  - Lets agents pick valid service names before ingestion.
- `blast.ingest.web.sync`
  - Fetches public JSON records.
  - Infers fields from payloads.
  - Writes data to SQLite.
  - Syncs metadata to OpenMetadata.
  - Captures snapshot and compares drift with prior snapshot.

## Success Criteria

- Agent can discover services before writes.
- Agent can run full web ingest with one MCP tool call.
- Tool output includes drift summary and snapshot paths.
- Missing service behavior is actionable (choose existing or allow creation).

## Suggested MCP Call Sequence

1. `blast.openmetadata.services.list`
2. `blast.ingest.web.sync` with selected service
3. Repeat step 2 later for drift comparison
4. Optional: `blast.snapshot.compare` and `blast.impact.analyze` for additional review

## Real-World Usefulness

- Data monitoring: detect schema/field drift from public APIs before pipelines fail.
- Vendor risk: catch payload contract changes from external integrations early.
- Release safety: compare snapshots between runs to gate deployments when metadata changes.
- Agent automation: AI copilots can ingest, catalog, and assess change impact without manual scripting.
- Team visibility: gives analysts and platform teams concrete drift summaries and artifacts.

## Demo Commands

Build:

- `go build -o bin/blast ./cmd`

Run MCP server:

- `./bin/blast mcp --ingestion-dir ./snapshots/ingestion`

CLI parity run:

- `blast ingest web --url https://api.github.com/events --count 100 --table github_events --service <existing_service> --database webdb --schema public --source <existing_service>.webdb.public`



## Exit Definition

Effort 3 is complete when an AI agent can:

- discover valid OpenMetadata services,
- run ingestion from a live web endpoint,
- produce snapshots and drift summary,
- and provide a recommendation based on detected change risk.


Add this section to the end of effort3-ai-ingestion-mcp-checklist.md.


### Proposed implementation items
- Add configurable workers:
  - default from CPU count
  - override via flag, for example --workers
- Parallelize:
  - table normalization
  - lineage fetch
  - hash generation
- Add safe aggregation:
  - collect worker outputs through channels
  - single writer merges final snapshot maps
- Add cancellation and timeout:
  - stop all workers on fatal error
  - context-aware worker shutdown

### MCP feature additions
- Add ingest status streaming events for long-running operations.
- Add tool for dry-run mode:
  - infer fields and show planned schema without writing.
- Add tool for resume mode:
  - continue failed ingest from last successful checkpoint.
- Add tool for drift summary only:
  - return concise change counts and top risky entities.

### Bubble Tea CLI interface plan
- Add new command: blast ui
- Screens:
  - connection check and environment status
  - source URL input and preview
  - target selection for service database schema table
  - record count and run options
  - live ingest progress view
  - drift result and impact summary view
  - recent runs and snapshot history
- UX goals:
  - keyboard-first flow
  - clear error handling with suggested fixes
  - copyable command preview for reproducibility
