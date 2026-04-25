# Blast Radius Demo Snippets

**Quick Setup**: Build the binary first with:
```bash
go build -o bin/blast ./cmd
```

---

## 1. Validate OpenMetadata Connectivity

Check that Blast Radius can connect to OpenMetadata:

```bash
# Table format (default)
./bin/blast validate

# JSON output
./bin/blast validate --output json
```

**What it does**: Verifies OpenMetadata base URL is accessible, counts visible database services, and optionally checks visibility of a specific database.

---

## 2. Explore Services

List all available OpenMetadata database services:

```bash
# Table format
./bin/blast services

# JSON output
./bin/blast services --output json
```

**What it does**: Lists all database service integrations visible in OpenMetadata with their types (Postgres, MySQL, Snowflake, etc.).

---

## 3. Explore Databases

List all databases within a service:

```bash
./bin/blast databases --service my_service

# JSON output
./bin/blast databases --service my_service --output json
```

**What it does**: Lists all databases within a specific OpenMetadata database service.

---

## 4. Explore Tables & Catalog

List all tables in a database:

```bash
# Using database FQN (Fully Qualified Name)
./bin/blast tables my_service.my_database.public

# Short alias
./bin/blast ls my_service.my_database.public

# JSON output
./bin/blast catalog my_service.my_database.public --output json
```

**What it does**: Lists all tables in a database with their metadata, column info, and descriptions.

---

## 5. Get Table Lineage

Fetch upstream and downstream lineage for a table:

```bash
./bin/blast lineage my_service.my_database.public.my_table

# JSON output
./bin/blast lineage my_service.my_database.public.my_table --output json
```

**What it does**: Shows upstream (where data comes from) and downstream (where data flows to) dependencies for any table or column.

---

## 6. Snapshot Metadata

Capture a point-in-time snapshot of metadata from OpenMetadata:

```bash
# Using config/environment variables
./bin/blast snapshot

# Using explicit FQN
./bin/blast snapshot --source my_service.my_database.public

# Saves to: snapshots/sources/my_service/my_database/public/snapshot_<timestamp>.json
```

**What it does**: Creates a JSON snapshot of all tables, columns, tags, ownership, descriptions, and lineage for a database. Snapshots are stored hierarchically by table for easy comparison later.

---

## 7. Compare Two Snapshots

Detect what changed between two snapshots:

```bash
./bin/blast compare \
  snapshots/sources/my_service/my_database/public/snapshot_OLD.json \
  snapshots/sources/my_service/my_database/public/snapshot_NEW.json

# JSON output (shows detailed change objects)
./bin/blast compare snapshot_OLD.json snapshot_NEW.json --output json
```

**What it does**: Compares two snapshots and reports:
- **Added**: New tables, columns, tags  
- **Deleted**: Removed tables, columns, tags  
- **Modified**: Changed descriptions, owners, column types  
- **Severity**: INFO, WARNING, or CRITICAL for each change

---

## 8. Analyze Impact of Changes

Compute downstream risk from snapshot changes:

```bash
./bin/blast impact \
  snapshots/sources/my_service/my_database/public/snapshot_OLD.json \
  snapshots/sources/my_service/my_database/public/snapshot_NEW.json

# JSON output
./bin/blast impact snapshot_OLD.json snapshot_NEW.json --output json
```

**What it does**: Analyzes changes and returns:
- **Risk Level**: LOW, MEDIUM, HIGH, CRITICAL  
- **Impacted Assets**: Tables, dashboards, queries affected by changes  
- **Recommended Actions**: What to do next (e.g., "notify downstream teams", "enable column masking")

---

## 9. Web Ingestion (Public API → Metadata)

Fetch records from a public web API, load to SQLite, and sync metadata to OpenMetadata:

```bash
# Interactive wizard (recommended for demos)
./bin/blast ingest wizard

# Non-interactive version
./bin/blast ingest web \
  --url https://api.github.com/events \
  --count 100 \
  --table github_events \
  --service my_service \
  --database webdb \
  --schema public

# Run the same command again to detect drift in the API schema
./bin/blast ingest web \
  --url https://api.github.com/events \
  --count 100 \
  --table github_events \
  --service my_service \
  --database webdb \
  --schema public
```

**What it does**: 
1. Fetches JSON records from a web URL  
2. Auto-infers column types (string, int, bool, etc.)  
3. Stores in local SQLite database  
4. Publishes metadata to OpenMetadata  
5. Captures a snapshot for drift detection  
6. On repeat runs, compares with previous snapshot to detect API schema changes

---

## 10. Drift Detection with Policies

Detect metadata drift and fail if violations occur:

```bash
# Simple drift check (all changes reported as WARNING or CRITICAL)
./bin/blast guard drift \
  snapshots/sources/my_service/analytics/public/snapshot_OLD.json \
  snapshots/sources/my_service/analytics/public/snapshot_NEW.json

# Fail only on CRITICAL severity changes
./bin/blast guard drift \
  snapshot_OLD.json snapshot_NEW.json \
  --fail-on critical

# Fail if total changes exceed 5
./bin/blast guard drift \
  snapshot_OLD.json snapshot_NEW.json \
  --fail-on warning \
  --max-total 5

# Use a policy profile (recommended)
./bin/blast guard drift \
  snapshot_OLD.json snapshot_NEW.json \
  --profile profiles/guard/analytics.guard.yaml

# Exit code: 0 = pass, 1 = fail due to violations
echo "Exit code: $?"
```

**What it does**: Compares snapshots and enforces policies like:
- Fail only on CRITICAL changes
- Allow up to N total changes
- Required fields that cannot be removed

---

## 11. Contract Readiness Checks

Score table metadata for completeness (ownership, description, tags):

```bash
# Check a snapshot against contracts
./bin/blast guard contracts \
  snapshots/sources/my_service/analytics/public/snapshot_CURRENT.json

# Output: score (0-100), violations, and metadata quality recommendations
```

**What it does**: Scores metadata quality across all tables:
- **Score**: 0-100 based on description presence, owner assignment, tag coverage  
- **Violations**: Missing required metadata  
- **Recommendations**: What metadata to add

---

## 12. Release Safety Gate

Combined drift + impact + contracts check for CI/CD pipelines:

```bash
# Basic release check (will fail on policy violations)
./bin/blast release-check \
  --baseline snapshots/sources/my_service/analytics/public/snapshot_OLD.json \
  --current snapshots/sources/my_service/analytics/public/snapshot_NEW.json

# With strict policy profile
./bin/blast release-check \
  --baseline snapshot_OLD.json \
  --current snapshot_NEW.json \
  --profile profiles/guard/analytics.guard.yaml \
  --report-file release_report.json \
  --markdown-file release_report.md

# With custom policy (allow only LOW or MEDIUM risk)
./bin/blast release-check \
  --baseline snapshot_OLD.json \
  --current snapshot_NEW.json \
  --max-risk medium \
  --require-table-description=true \
  --report-file report.json \
  --markdown-file report.md

# In CI/CD (check exit code)
if ./bin/blast release-check ...; then
  echo "Release approved"
  git tag release
else
  echo "Release blocked - review artifacts"
  cat release_report.md
  exit 1
fi
```

**What it does**: All-in-one pre-release check:
- Compares baseline vs current snapshots  
- Detects drift and assigns severity  
- Analyzes impact on downstream assets  
- Checks metadata contracts (descriptions, owners)  
- Generates JSON report + Markdown summary  
- Exits 0 (pass) or 1 (fail)

---

## 13. Table Operations

Create, update, delete, and query tables:

```bash
# Create a new table
./bin/blast tables create \
  --name my_new_table \
  --schema my_service.my_database.my_schema

# Get table details
./bin/blast tables get my_service.my_database.schema.table

# Add sample data (patch JSON)
./bin/blast tables add-data my_service.my_database.schema.table \
  --body-file examples/payloads/table-sample-data.patch.json

# Update table metadata
./bin/blast tables update my_service.my_database.schema.table \
  --body-file table-update-metadata.json

# Delete a table
./bin/blast tables delete my_service.my_database.schema.table
```

**What it does**: CRUD operations for tables in OpenMetadata. The `--body-file` expects a JSON patch following OpenMetadata's API format.

---

## 14. Glossary Management

Create glossary terms for business metadata:

```bash
# Create a glossary term
./bin/blast glossary create \
  --term "Customer_ID" \
  --description "Unique identifier for a customer" \
  --glossary-name business_terms

# Assign term to table columns
# (via direct API call - see section 15)
```

**What it does**: Creates reusable business metadata terms that can be tagged to columns across multiple tables.

---

## 15. Direct OpenMetadata API Calls

For operations not yet wrapped by dedicated blast commands:

```bash
# GET request
./bin/blast api \
  --method GET \
  --path /services/databaseServices

# POST to create a service
./bin/blast api \
  --method POST \
  --path /services/databaseServices \
  --body '{"name":"my_service","serviceType":"Postgres"}'

# PATCH to update an entity
./bin/blast api \
  --method PATCH \
  --path /tables/by_name/my_service.my_db.schema.table \
  --body-file update.json

# Query parameters
./bin/blast api \
  --method GET \
  --path /services/databaseServices \
  --query "limit=50" \
  --query "offset=0"

# Save complex response
./bin/blast api \
  --method GET \
  --path /lineage/by_name/my_service.db.schema.table \
  --output json > lineage_response.json
```

**What it does**: Execute any OpenMetadata REST API call directly, useful for advanced operations or scripting.

---

## 16. MCP Server for AI Agents

Run Blast Radius as a Model Context Protocol server for AI agents:

```bash
# Terminal 1: Start MCP server
./bin/blast mcp \
  --ingestion-dir ./snapshots/ingestion \
  --config blast-radius.yaml

# Terminal 2: Run demo client (in examples/)
python3 examples/mcp_demo.py \
  --run-web-sync \
  --service my_service \
  --database webdb

# Or call MCP tools directly from Claude/Claude:
# - blast_snapshot: Capture metadata snapshot  
# - blast_compare: Compare two snapshots  
# - blast_impact: Analyze downstream impact  
# - blast_web_ingest_sync: Fetch web data and sync metadata  
# - blast_service_discovery: List available services  
```

**What it does**: 
- Exposes all Blast Radius capabilities over MCP protocol  
- Allows AI agents to autonomously:
  - Discover metadata schemas  
  - Ingest external data  
  - Detect drift  
  - Analyze impact  
  - Create release reports  
- Results saved to `./snapshots/ingestion/` for persistence

---

## 17. Create Services (OpenMetadata Onboarding)

Register a new database service in OpenMetadata:

```bash
# Using create command
./bin/blast create service \
  --name "my_postgres_prod" \
  --type "Postgres" \
  --host "postgres.example.com" \
  --port 5432 \
  --database "prod_db"

# Or direct API call
./bin/blast api \
  --method POST \
  --path /services/databaseServices \
  --body-file examples/payloads/create-service.postgres.json
```

**What it does**: Registers a new database connection in OpenMetadata so tables can be discovered and tracked.

---

## 18. Demo Playbook (End-to-End)

Complete demo sequence showcasing all features:

```bash
#!/bin/bash
set -euo pipefail

# Build
go build -o bin/blast ./cmd

# 1. Validate connectivity
echo "=== Step 1: Validate OpenMetadata ===" 
./bin/blast validate

# 2. Explore services
echo "=== Step 2: List Services ==="
./bin/blast services

# 3. Explore tables
echo "=== Step 3: List Tables ==="
./bin/blast tables my_service.my_database.public

# 4. Get lineage
echo "=== Step 4: Lineage Analysis ==="
./bin/blast lineage my_service.my_database.public.my_table

# 5. Capture baseline snapshot
echo "=== Step 5: Capture Baseline Snapshot ==="
./bin/blast snapshot --source my_service.my_database.public

# 6. Ingest web data (simulates API schema change)
echo "=== Step 6: Web Ingestion ==="
./bin/blast ingest web \
  --url https://api.github.com/events \
  --count 50 \
  --table github_events \
  --service my_service \
  --database webdb \
  --schema public

# 7. Capture new snapshot to show drift
echo "=== Step 7: Capture Current Snapshot ==="
./bin/blast snapshot --source my_service.webdb.public

# 8. Compare snapshots
echo "=== Step 8: Compare Snapshots ==="
BASELINE=$(find snapshots/sources/my_service/my_database/public -name "*.json" | head -1)
CURRENT=$(find snapshots/sources/my_service/webdb/public -name "*.json" | head -1)
./bin/blast compare "$BASELINE" "$CURRENT"

# 9. Analyze impact
echo "=== Step 9: Impact Analysis ==="
./bin/blast impact "$BASELINE" "$CURRENT"

# 10. Run drift detection with policy
echo "=== Step 10: Drift Detection with Policy ==="
./bin/blast guard drift "$BASELINE" "$CURRENT" --fail-on critical || echo "Drift detected (expected)"

# 11. Run release check
echo "=== Step 11: Release Safety Gate ==="
./bin/blast release-check \
  --baseline "$BASELINE" \
  --current "$CURRENT" \
  --profile profiles/guard/analytics.guard.yaml \
  --report-file release_demo.json \
  --markdown-file release_demo.md || echo "Release check failed (expected with drift)"

echo "=== Demo Complete ==="
echo "Generated artifacts:"
echo "  - release_demo.json (report)"
echo "  - release_demo.md (markdown summary)"
```

---

## Environment Setup

All commands use these environment variables (or config file):

```bash
export OM_BASE_URL="http://localhost:8585"        # OpenMetadata base URL
export OM_JWT_TOKEN="<your-jwt-token>"            # OpenMetadata auth token
export BR_DATABASE_FQN="my_service.db.public"     # Default database for snapshot
export BR_CONFIG="blast-radius.yaml"              # Config file path
```

Or use config file `blast-radius.yaml`:
```yaml
openmetadata:
  baseUrl: http://localhost:8585
  jwtToken: <token>
  
database:
  fqn: my_service.my_database.public

snapshot:
  directory: ./snapshots

output:
  format: table  # or json
```

---

## Quick Copy-Paste Template

Use this as a starting point for custom demos:

```bash
#!/bin/bash
set -euo pipefail

# Configuration
SERVICE="my_service"
DATABASE="my_db"
SCHEMA="public"
OM_BASE_URL="${OM_BASE_URL:-http://localhost:8585}"
SNAPSHOT_DIR="./snapshots"

# Build
go build -o bin/blast ./cmd

# Your demo sequence here
./bin/blast validate
./bin/blast services
./bin/blast snapshot --source "$SERVICE.$DATABASE.$SCHEMA"

echo "✓ Demo complete"
```

---

## Testing Release Check

Pre-built demo artifacts in `artifacts/`:

```bash
# Run release-check with pre-captured snapshots showing real drift
./bin/blast release-check \
  --baseline artifacts/release-check-fail.json \
  --current artifacts/release-check-pass.json \
  --profile profiles/guard/analytics.guard.yaml

# View generated reports
cat release_report.md
cat release_report.json
```

---

## Tips for Great Demos

1. **Start with validate**: Builds confidence that setup works
2. **Use ingest wizard**: Most interactive and engaging feature
3. **Show drift detection**: Capture baseline → ingest data → show differences
4. **Release gate is the finale**: Shows business value (CI/CD integration)
5. **Use `--output json`**: Great for scripting and showing structure
6. **Keep snapshots small**: Demo faster with limited data
7. **Pre-record times**: Cache baseline snapshots to save time during live demos

