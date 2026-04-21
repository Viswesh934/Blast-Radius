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
