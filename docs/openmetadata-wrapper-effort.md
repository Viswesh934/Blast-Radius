# OpenMetadata Wrapper Effort Doc

## Purpose

Blast Radius is currently an initial OpenMetadata wrapper focused on metadata discovery, snapshotting, lineage lookup, entity creation, and direct API calls. The immediate goal is to make the CLI useful for common OpenMetadata tasks without relying on a local Postgres workflow or the earlier TUI/DB-console model.

This document captures the effort, scope, and next steps so the project can evolve into a more complete product later.

## Current State

The current CLI already supports:

- Validating OpenMetadata connectivity.
- Listing services, databases, tables, and glossaries.
- Creating services, databases, schemas, tables, glossaries, and glossary terms.
- Fetching lineage for an entity.
- Capturing snapshots and comparing them locally.
- Running drift and contract-readiness guards using versioned domain profiles (`guard.yaml`).
- Calling arbitrary OpenMetadata endpoints with a raw API command.

## What This Phase Is

This phase is a thin operational wrapper around OpenMetadata.

It is meant to:

- Reduce the friction of common metadata actions.
- Hide API details behind a small CLI surface.
- Make it possible to script OpenMetadata tasks consistently.
- Provide enough surface area to learn where the real product value is.

## What This Phase Is Not

This phase is not yet:

- A full governance platform.
- A rich data catalog UI.
- A complete lifecycle manager for every OpenMetadata entity.
- A replacement for the OpenMetadata UI itself.

## Near-Term Scope

The next practical additions should be:

- Better CRUD wrappers for common entities like tables, topics, dashboards, pipelines, tags, and classifications.
- Safer create flows with schema validation and clearer API error messages.
- JSON output and machine-friendly formatting for automation.
- Import/export workflows for entity definitions.
- Shortcuts for common OpenMetadata lookups and edits.

## Longer-Term Direction

Once the wrapper is stable, the project can evolve in one or more of these directions:

- CLI-first operations with richer subcommands and recipes.
- MCP integration so AI agents can call the same OpenMetadata actions safely.
- Lightweight dashboards for inspection, search, and change review.
- Policy-aware workflows for validation and controlled updates.

## Suggested Milestones

1. Stabilize the CLI wrapper.
2. Cover the highest-value OpenMetadata entities.
3. Add structured output and better diagnostics.
4. Introduce MCP tool exposure.
5. Decide whether a dashboard is needed or whether the CLI remains the primary interface.

Effort notes:

- Effort 1 now includes profile-based guard policies and profile validation/show commands.
- Effort 2 now includes MCP exposure and a CI/CD-focused `release-check` gate command with workflow artifacts.
- Effort 3 will focus on packaging/distribution and broader connector coverage.

## Operational Notes

- Prefer OpenMetadata API payloads over local database assumptions.
- Use payload files for advanced create flows when the API schema varies by deployment.
- Keep the command surface small and explicit.
- Maintain a raw `api` escape hatch for unsupported endpoints.

## Success Criteria For This Phase

- A user can create and inspect core metadata objects from the CLI alone.
- A user does not need a local Postgres setup to work with metadata entities.
- Common OpenMetadata workflows can be scripted and repeated reliably.
- The repo clearly communicates that this is an early wrapper, not the final product.
