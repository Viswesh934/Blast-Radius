# Blast Radius

Blast Radius is an OpenMetadata-focused CLI for metadata snapshotting, lineage lookup, change comparison, and downstream impact analysis.

## What It Does

- Connects directly to OpenMetadata APIs.
- Captures metadata snapshots for a database FQN.
- Compares two snapshots to identify added, deleted, and modified entities.
- Computes downstream impact and risk level from detected changes.
- Fetches lineage for a live OpenMetadata entity.
- Creates OpenMetadata services, databases, and schemas from CLI.
- Creates OpenMetadata tables, glossaries, and glossary terms from CLI.
- Supports direct OpenMetadata API calls for advanced operations.

## Commands

```bash
blast validate
blast validate --output json
blast services
blast databases --service my_service
blast tables [database-fqn]
blast tables create --name events --schema service.database.schema
blast tables get service.database.schema.table
blast tables add-data service.database.schema.table --body-file examples/payloads/table-sample-data.patch.json
blast tables delete service.database.schema.table
blast tables update service.database.schema.table --body-file table-update.json
blast snapshot
blast guard drift <baseline-snapshot.json> <current-snapshot.json>
blast guard drift <baseline-snapshot.json> --source service.database.schema
blast guard contracts [snapshot.json]
blast guard contracts --source service.database.schema
blast guard profile validate profiles/guard/analytics.guard.yaml
blast guard profile show profiles/guard/analytics.guard.yaml
blast lineage <entity-fqn>
blast compare <older-snapshot.json> <newer-snapshot.json>
blast impact <older-snapshot.json> <newer-snapshot.json>
blast release-check --baseline snapshots/sources/my_service/analytics/public/snapshot_1776536091.json --source my_service.analytics.public --profile profiles/guard/analytics.guard.yaml
blast mcp
blast create service --name my_service --type Postgres
blast create database --name analytics --service my_service
blast create schema --name public --database my_service.analytics
blast create table --name events --schema my_service.analytics.public
blast create table --payload-file examples/payloads/create-table.minimal.json
blast api --method GET --path /services/databaseServices
blast glossary list
blast glossary create --name finance --description "Finance definitions"
blast glossary term create --glossary finance --name pii --description "Personally identifiable information"
```

Command aliases:

- `blast tables` also supports `blast table`, `blast ls`, and `blast catalog`.
- `blast tables get` also supports `show`, `describe`, and `inspect`.
- `blast tables delete` also supports `rm` and `remove`.
- `blast tables update` also supports `edit` and `patch`.
- `blast tables add-data` updates OpenMetadata sample/metadata payloads, not physical source DB rows.
- `blast glossary` also supports `blast glossaries`.
- `blast glossary term` also supports `blast glossary terms`.
- `blast create table` supports `--payload` and `--payload-file` for OpenMetadata table JSON.
- Most wrapper commands support `--output json` for machine-friendly output.

## Examples

Use these patterns when you want to stay entirely inside OpenMetadata:

```bash
# Create a service, then create a database and schema under it
blast create service --name analytics_service --type Postgres
blast create database --name analytics --service analytics_service
blast create schema --name public --database analytics_service.analytics

# Discover metadata
blast services
blast databases --service analytics_service
blast tables create --name events --schema analytics_service.analytics.public
blast tables analytics_service.analytics.public
blast tables get analytics_service.analytics.public.events
blast tables add-data analytics_service.analytics.public.events --body-file examples/payloads/table-sample-data.patch.json
blast tables update analytics_service.analytics.public.events --body-file table-update.json
blast tables --output json analytics_service.analytics.public

# Work with glossaries
blast glossary create --name finance --description "Finance definitions"
blast glossary term create --glossary finance --name pii --description "Sensitive personal data"
blast glossary list
blast glossary list --glossary finance

# Call an OpenMetadata endpoint directly
blast api --method GET --path /glossaries

# Use bundled payload examples
blast create service --payload-file examples/payloads/create-service.postgres.json
blast create table --payload-file examples/payloads/create-table.minimal.json
blast tables update analytics_service.analytics.public.events --body-file examples/payloads/update-table-description.patch.json
```

## Glossary

- `service`: an OpenMetadata database service, such as Postgres or Snowflake.
- `database`: a logical database owned by a service in OpenMetadata.
- `schema`: a database schema that belongs to a database.
- `glossary`: a top-level vocabulary container for business terms.
- `glossary term`: a term inside a glossary, optionally nested under a parent term.
- `entity FQN`: a fully qualified OpenMetadata name, such as `service.database.schema`.

## Configuration

Use either environment variables or a local config file.

Required for OpenMetadata workflows:

- `OM_BASE_URL` (should include `/api/v1`)
- `OM_JWT_TOKEN` (if your OpenMetadata deployment requires auth)
- `BR_DATABASE_FQN` (default database FQN used by `snapshot`, `tables`, and `validate`)

Optional:

- `BR_SNAPSHOT_DIRECTORY` (default: `./snapshots`)
- `BR_OUTPUT_FORMAT` (`table` or `json`)

Example files:

- `.env.example`
- `blast-radius.yaml.example`

## Build and Run

```bash
go mod tidy
go build -o bin/blast ./cmd
./bin/blast --help
```

Optional local alias if you still build `bin/blast-radius`:

```bash
ln -sf "$PWD/bin/blast-radius" "$HOME/.local/bin/blast"
```

## Typical Workflow

```bash
# 1. Validate connectivity and config
./bin/blast validate

# 2. List tables visible in OpenMetadata
./bin/blast tables

# 2b. Discover existing services and databases
./bin/blast services
./bin/blast databases --service my_service

# 2c. Create entities in OpenMetadata (no Postgres dependency)
./bin/blast create service --name my_service --type Postgres
./bin/blast create database --name analytics --service my_service
./bin/blast create schema --name public --database my_service.analytics
./bin/blast create table --name events --schema my_service.analytics.public

# 2d. Create glossary content
./bin/blast glossary create --name finance --description "Finance definitions"
./bin/blast glossary term create --glossary finance --name pii --description "Sensitive personal data"

# 3. Capture snapshots over time
./bin/blast snapshot
# or target a different source without changing .env
./bin/blast snapshot --source my_service.analytics.public
# ... later
./bin/blast snapshot

# 3b. Run metadata drift and contract readiness guards
./bin/blast guard drift snapshots/sources/my_service/analytics/public/snapshot_older.json snapshots/sources/my_service/analytics/public/snapshot_newer.json
./bin/blast guard drift snapshots/sources/my_service/analytics/public/snapshot_older.json --source my_service.analytics.public
./bin/blast guard contracts snapshots/sources/my_service/analytics/public/snapshot_newer.json
./bin/blast guard contracts --source my_service.analytics.public --min-score 70

# 3c. Use versioned guard profiles per domain
./bin/blast guard profile validate profiles/guard/analytics.guard.yaml
./bin/blast guard profile show profiles/guard/analytics.guard.yaml
./bin/blast guard drift snapshots/sources/my_service/analytics/public/snapshot_older.json snapshots/sources/my_service/analytics/public/snapshot_newer.json --profile profiles/guard/analytics.guard.yaml
./bin/blast guard contracts --source my_service.analytics.public --profile profiles/guard/analytics.guard.yaml

# 4. Compare two snapshots
./bin/blast compare snapshots/sources/my_service/analytics/public/snapshot_older.json snapshots/sources/my_service/analytics/public/snapshot_newer.json

# 5. Analyze impact from differences
./bin/blast impact snapshots/older.json snapshots/newer.json

# 5b. Run a CI-friendly release gate check
./bin/blast release-check \
	--baseline snapshots/sources/my_service/analytics/public/snapshot_1776536091.json \
	--source my_service.analytics.public \
	--profile profiles/guard/analytics.guard.yaml \
	--max-risk high \
	--report-file artifacts/release-check.json \
	--markdown-file artifacts/release-check.md

# 6. Inspect lineage for an entity directly from OpenMetadata
./bin/blast lineage service.db.schema.table_name

# 7. Inspect, update, or delete a table
./bin/blast tables create --name events --schema analytics_service.analytics.public
./bin/blast tables get analytics_service.analytics.public.events
./bin/blast tables add-data analytics_service.analytics.public.events --body-file examples/payloads/table-sample-data.patch.json
./bin/blast tables update analytics_service.analytics.public.events --body-file table-update.json
./bin/blast tables delete analytics_service.analytics.public.events

# 8. Run as an MCP server for AI agents and ingestion clients
./bin/blast mcp --ingestion-dir ./snapshots/ingestion
```

## MCP Server + Ingestion Pipeline

Blast Radius can run as an MCP server over stdio so external agents and automation can trigger metadata workflows in real time.

Start server:

```bash
./bin/blast mcp --ingestion-dir ./snapshots/ingestion
```

Exposed MCP tools:

- `blast.snapshot.capture`: create a new snapshot from OpenMetadata.
- `blast.snapshot.compare`: compare two snapshot files.
- `blast.impact.analyze`: compute downstream impact between snapshots.
- `blast.ingest.event`: append a real-time event into the ingestion queue.
- `blast.ingest.flush`: flush queued events for a source, capture a snapshot, compare against a baseline, and emit an impact report.

Ingestion outputs:

- Event log: `snapshots/ingestion/events.ndjson`
- Flush reports: `snapshots/ingestion/reports/report_<timestamp>.json`

Notes:

- `blast.ingest.event` requires `type` and `source_fqn`.
- `blast.ingest.flush` can take an explicit `baseline_snapshot`; if omitted, it uses the latest prior snapshot for that source.
- If no baseline snapshot exists, flush still captures a fresh snapshot and returns a message indicating compare/impact was skipped.

## CI/CD Release Gate

Use `release-check` to gate deploys and pull requests with one deterministic command.

Example:

```bash
./bin/blast release-check \
	--baseline snapshots/sources/my_service/analytics/public/snapshot_1776536091.json \
	--source my_service.analytics.public \
	--profile profiles/guard/analytics.guard.yaml \
	--max-risk high \
	--max-impacted-assets 20 \
	--report-file artifacts/release-check.json \
	--markdown-file artifacts/release-check.md
```

Behavior:

- Runs drift policy checks (`fail_on`, `max_total`).
- Runs contract readiness checks (`min_score`, owner/description policy).
- Runs impact checks (`max_risk`, optional `max_impacted_assets`).
- Exits non-zero when any policy check fails.

GitHub Actions workflow:

- See `.github/workflows/metadata-release-gate.yml`.
- Configure repository secrets: `OM_BASE_URL`, `OM_JWT_TOKEN`.
- Configure repository variables: `BR_DATABASE_FQN`, `BASELINE_SNAPSHOT`.

## Notes

- Snapshot and lineage retrieval are OpenMetadata API-driven.
- `guard drift` is CI-friendly and can compare two snapshot files or compare a baseline file to a live OpenMetadata source (`--source`).
- `guard contracts` scores metadata contract readiness (descriptions, columns, datatypes) from a snapshot file or live source.
- `guard profile validate` and `guard profile show` let teams verify and inspect versioned `guard.yaml` policy files.
- `guard drift --profile` and `guard contracts --profile` apply domain policy from a versioned profile file.
- `snapshot` writes one aggregate snapshot under `snapshots/sources/<source-fqn-path>/` and one per-table snapshot under `snapshots/tables/<table-fqn-path>/`.
- Use `snapshot --source <fqn>` to snapshot a different source at runtime without changing environment variables.
- Compare and impact operate on saved snapshot JSON files.
- If `BR_OUTPUT_FORMAT=json`, compare and impact commands print JSON output.
- `create service` supports `--payload` and `--payload-file` for full OpenMetadata service JSON.
- `create table` supports `--payload` and `--payload-file` for full OpenMetadata table JSON.
- `tables create` supports `--payload` and `--payload-file` for full OpenMetadata table JSON.
- `tables add-data` supports `--body` and `--body-file` for sample data metadata updates.
- `tables update` supports `--body` and `--body-file` so you can use the exact OpenMetadata update payload your deployment expects.
- Use `--output json` on wrapper commands when scripting automation.
- Use `blast api` for operations that are not yet wrapped by dedicated commands.
- The effort and roadmap discussion lives in [docs/openmetadata-wrapper-effort.md](docs/openmetadata-wrapper-effort.md).
- The effort-1 validation checklist lives in [docs/effort1-smoke-checklist.md](docs/effort1-smoke-checklist.md).
- The effort-2 release gate checklist lives in [docs/effort2-release-gate-checklist.md](docs/effort2-release-gate-checklist.md).
