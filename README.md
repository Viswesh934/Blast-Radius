# Blast Radius

Blast Radius is a terminal-first data impact analysis tool.

This version provides a Bubble Tea TUI scaffold with:

- Home screen with actions
- Snapshot history list with filter/search
- Compare flow (select two snapshots)
- Impact analysis screen with risk badge, changes, impacted assets, and recommendations
- Loading and message bars
- Help overlay
- Reusable component library (badges, alerts, table, stat boxes, buttons, progress, timeline, dialog)
- Workflow abstractions for snapshot, comparison, and impact execution progress

OpenMetadata integration is intentionally deferred so UI and workflows can be finalized first.

## Project Layout

```text
cmd/
  main.go
internal/
  config/
  impact/
  openmetadata/
  snapshot/
  tui/

Key TUI files:

- internal/tui/components.go
- internal/tui/workflow.go
```

## Configuration

Use either environment variables or a local config file.

Environment variables:

- OM_BASE_URL
- OM_JWT_TOKEN
- BR_DATABASE_FQN
- BR_SNAPSHOT_DIRECTORY
- BR_OUTPUT_FORMAT

Example files:

- .env.example
- blast-radius.yaml.example

## Build and Run

```bash
go mod tidy
go build -o bin/blast-radius ./cmd
./bin/blast-radius
```

## Keyboard

- up/down or j/k: navigate
- enter: select
- esc: back
- /: filter snapshot list
- ?: toggle help
- q or ctrl+c: quit

## Current Behavior

- Taking a snapshot creates a local placeholder snapshot file in the snapshot directory.
- Compare and impact run locally using saved snapshot JSON files.
- OpenMetadata client code exists as a scaffold and will be connected in the next integration step.
