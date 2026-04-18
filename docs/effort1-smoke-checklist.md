# Effort 1 Smoke Checklist

Run this checklist against a working OpenMetadata environment.

## Prerequisites

- `OM_BASE_URL` points to `/api/v1`.
- `OM_JWT_TOKEN` is set when auth is enabled.
- CLI is built: `go build -o bin/blast ./cmd`.

## Connectivity

- `./bin/blast validate`
- `./bin/blast validate --output json`

Expected:
- Command succeeds.
- JSON mode returns valid JSON.

## Discovery

- `./bin/blast services`
- `./bin/blast databases --service <service>`
- `./bin/blast tables <service.database.schema>`
- `./bin/blast glossary list`

Expected:
- Commands succeed.
- Results are non-empty when entities exist.

## Create Flows

- `./bin/blast create service --payload-file examples/payloads/create-service.postgres.json`
- `./bin/blast create database --name analytics --service <service>`
- `./bin/blast create schema --name public --database <service.analytics>`
- `./bin/blast create table --payload-file examples/payloads/create-table.minimal.json`
- `./bin/blast glossary create --name finance`
- `./bin/blast glossary term create --glossary finance --name pii`

Expected:
- Command returns created IDs/FQNs.

## Table Lifecycle

- `./bin/blast tables get <service.database.schema.table>`
- `./bin/blast tables update <service.database.schema.table> --body-file examples/payloads/update-table-description.patch.json`
- `./bin/blast tables delete <service.database.schema.table>`

Expected:
- Get returns table details.
- Update returns updated table.
- Delete succeeds.

## Raw API Escape Hatch

- `./bin/blast api --method GET --path /services/databaseServices`
- `./bin/blast api --method GET --path /glossaries --output json`

Expected:
- Calls succeed and JSON output is valid.

## Error Quality

Try one invalid call:
- `./bin/blast create database --name bad --service does-not-exist`

Expected:
- Error includes action + status + OpenMetadata message.

## Guard Profiles (Effort 1)

- `./bin/blast guard profile validate profiles/guard/analytics.guard.yaml`
- `./bin/blast guard profile show profiles/guard/analytics.guard.yaml --output json`
- `./bin/blast guard drift <baseline.json> <current.json> --profile profiles/guard/analytics.guard.yaml`
- `./bin/blast guard contracts --source <service.database.schema> --profile profiles/guard/analytics.guard.yaml`

Expected:
- Profile validate succeeds for valid `guard.yaml` files.
- Profile show prints normalized profile content.
- Drift/contracts commands apply policy from profile and return CI-friendly pass/fail exit codes.
