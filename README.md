# Blast Radius 🎯

> **Continuous metadata operations for safer, smarter data releases**

Blast Radius is a CLI + MCP toolkit that captures metadata snapshots, detects drift, analyzes impact, and enforces policies—enabling teams to catch schema changes *before* they break production jobs.

**Use it to:**
- 📸 Capture point-in-time metadata snapshots from OpenMetadata
- 🔍 Compare snapshots to detect schema drift automatically
- 📊 Analyze downstream impact of metadata changes
- 🛡️ Enforce release policies before deployments 
- 🤖 Expose workflows to AI agents via MCP protocol
- 🔄 Ingest public APIs, auto-infer schema, sync to OpenMetadata
- 📈 Maintain lineage and governance across metadata lifecycle

---

## 🤔 The Problem

**Data teams face three core challenges:**

| Challenge | Effect | Blast Radius Solution |
|-----------|--------|----------------------|
| **Silent API changes** | External APIs evolve without notice, breaking downstream jobs | Snapshot + compare to detect changes immediately |
| **Late drift discovery** | Schema breaks go unnoticed until dashboards/pipelines fail in prod | Drift detection gates catch issues before release |
| **Manual metadata ops** | Onboarding, governance, and lineage tracking require manual effort | Automation + policies ensure consistency |
| **AI + metadata gap** | AI agents can reason about code but can't safely execute metadata workflows | MCP protocol enables governed agent access |

**Result:** Unplanned downtime, broken dashboards, failed ETL jobs, and emergency hotfixes.

**Blast Radius Solution:** Deterministic, policy-driven metadata validation at every stage.

---

## 🎯 What It Does (Core Features)

### 1. **Metadata Snapshots**
Capture a point-in-time JSON snapshot of all tables, columns, descriptions, ownership, tags, and lineage for a database.
```bash
./bin/blast snapshot --source my_service.my_db.public
# Saves to: snapshots/sources/my_service/my_db/public/snapshot_<timestamp>.json
```

### 2. **Drift Detection**
Compare two snapshots to identify added, deleted, modified tables/columns with severity levels.
```bash
./bin/blast compare snapshot_old.json snapshot_new.json
# Shows: added=2, deleted=1, modified=3 changes with CRITICAL/WARNING/INFO severity
```

### 3. **Impact Analysis**
Compute downstream risk: which tables/dashboards/pipelines are affected by detected changes.
```bash
./bin/blast impact snapshot_old.json snapshot_new.json
# Returns: risk_level, impacted_assets, recommended_actions
```

### 4. **Release Safety Gate**
All-in-one pre-deployment check: drift + impact + metadata completeness with exit codes for CI/CD.
```bash
./bin/blast release-check --baseline snap1.json --current snap2.json --profile guard.yaml
# Exits 0 (pass/approved) or 1 (fail/blocked) for pipeline gates
```

### 5. **Web API Ingestion**
Fetch records from public JSON APIs, auto-infer column types, load to SQLite, publish to OpenMetadata, and detect schema drift on repeat runs.
```bash
./bin/blast ingest web --url https://api.github.com/events --count 100 --service web_service --database webdb --schema public
```

### 6. **Policy Enforcement**
Define metadata guards (YAML profiles) that fail releases on violations: drift severity, change counts, missing metadata.
```yaml
# profiles/guard/analytics.guard.yaml
drift:
  failOn: critical        # Fail only on CRITICAL severity
  maxTotal: 5             # Allow max 5 total changes
contracts:
  minScore: 80            # Metadata completeness score 0-100
  requireOwner: true      # Owner field required
  requireTableDescription: true
```

### 7. **Lineage Inspection**
Fetch upstream (sources) and downstream (consumers) dependencies for any table.
```bash
./bin/blast lineage my_service.my_db.public.my_table
```

### 8. **AI Agent Access (MCP)**
Expose all workflows via Model Context Protocol for AI agents to autonomously manage metadata.
```bash
./bin/blast mcp --ingestion-dir ./snapshots/ingestion
# Agents can: discover services, ingest data, detect drift, analyze impact
```

---

## 🏗️ Architecture & Tech Stack

### **Technology Choices**
- **Language:** Go (fast, single binary, easy deployment)
- **CLI:** Cobra framework (battle-tested command parsing)
- **OpenMetadata Client:** Direct HTTP REST API calls + JWT auth
- **Storage:** SQLite 3 (web ingestion data) + JSON files (snapshots)
- **Config:** Viper (env vars + YAML config files)
- **AI Integration:** Model Context Protocol (stdio-based)

### **System Architecture**

```
┌─────────────────────────────────────────────────────────┐
│                   USER / AI AGENT                       │
└──────────────────────┬──────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
    ┌────────┐  ┌────────────┐  ┌────────┐
    │  CLI   │  │  MCP Server│  │ Config │
    │Commands│  │ (AI Agents)│  │ Files  │
    └────┬───┘  └─────┬──────┘  └────────┘
         │            │
         └─────┬──────┘
               │
               ▼
    ┌──────────────────────────┐
    │   Blast Radius Core      │
    │  ─────────────────────   │
    │ • Snapshot Handler       │
    │ • Diff/Compare Engine    │
    │ • Impact Analyzer        │
    │ • Policy Enforcer        │
    │ • Web Ingest Orchestrator│
    └────────┬─────────────────┘
             │
    ┌────────┴──────────────┐
    │                       │
    ▼                       ▼
┌──────────────┐    ┌────────────────────┐
│ OpenMetadata │    │  SQLite / JSON     │
│   (remote)   │    │  (local storage)   │
└──────────────┘    └────────────────────┘
```

### **Command Flow Example: Release Gate**

```
User: ./bin/blast release-check --baseline snap1.json --current snap2.json

1. Load Snapshots
   ├─ Parse snap1.json (baseline state)
   └─ Parse snap2.json (current state)

2. Detect Drift
   ├─ Compare tables/columns/descriptions
   ├─ Assign severity (INFO/WARNING/CRITICAL)
   └─ Count changes by type

3. Analyze Impact
   ├─ Fetch lineage for affected entities
   ├─ Compute risk level (LOW/MEDIUM/HIGH/CRITICAL)
   └─ Identify downstream assets

4. Check Contracts
   ├─ Count metadata completeness scores
   ├─ Verify required fields (owner, description)
   └─ Calculate quality percentage

5. Decide Release
   ├─ Apply policy profile rules
   ├─ Compare against thresholds
   └─ Generate report (JSON + Markdown)

6. Exit with Code
   ├─ 0 = PASS (release approved)
   └─ 1 = FAIL (release blocked)
```

### **Data Flow: Web Ingestion**

```
Public JSON API → Fetch Records → Infer Schema
                                      ↓
                            ┌─────────────────┐
                            │  SQLite DB      │
                            │  (local copy)   │
                            └────────┬────────┘
                                     ↓
                        Auto-Create Metadata
                        (tables, columns, types)
                                     ↓
                            ┌─────────────────┐
                            │ OpenMetadata    │
                            │ (sync metadata) │
                            └────────┬────────┘
                                     ↓
                          Capture Snapshot
                            (point-in-time)
                                     ↓
                        Compare vs Previous
                          (detect drift)
```

---

## 🚀 Getting Started (5 Minutes)

### Prerequisites
- Go 1.21+
- Access to OpenMetadata instance (v1.0+)
- OpenMetadata JWT token (for authentication)

### 1. Build

```bash
git clone https://github.com/Viswesh934/blast-radius.git
cd blast-radius
go build -o bin/blast ./cmd
```

### 2. Configure

Set environment variables or create `.env` file:

```bash
export OM_BASE_URL="http://localhost:8585/api/v1"  # OpenMetadata endpoint
export OM_JWT_TOKEN="your-jwt-token"                # Auth token
export BR_DATABASE_FQN="my_service.my_db.public"   # Default database
```

Or create `blast-radius.yaml`:

```yaml
openmetadata:
  baseurl: "http://localhost:8585/api/v1"
  jwt_token: "your-jwt-token"

database:
  fqn: "my_service.my_db.public"

snapshot:
  directory: "./snapshots"

output:
  format: "table"
```

### 3. Verify Setup

```bash
./bin/blast validate
# Output: OpenMetadata connectivity verified ✓
```

### 4. Run Your First Demo

```bash
# Option A: Interactive menu (explore features at your pace)
python3 examples/demo-interactive.py

# Option B: Automated complete walkthrough (5-10 min)
bash examples/full-demo.sh

# Option C: Quick release gate demo (2-3 min)
bash examples/release_gate_demo.sh
```

---

## 📖 Usage Examples

### **Capture Metadata Snapshot**
```bash
./bin/blast snapshot --source my_service.my_db.public
# Creates: snapshots/sources/my_service/my_db/public/snapshot_<timestamp>.json
```

### **Compare Two Snapshots**
```bash
./bin/blast compare \
  snapshots/sources/my_service/my_db/public/snapshot_OLD.json \
  snapshots/sources/my_service/my_db/public/snapshot_NEW.json

# Output:
# Change summary:
#   added: 3
#   deleted: 1
#   modified: 5
#   affected_tables: 2
```

### **Analyze Downstream Impact**
```bash
./bin/blast impact snapshot_old.json snapshot_new.json

# Output:
# risk_level: HIGH
# impacted_assets: 7
#   - my_dashboard (DASHBOARD) - column type changed
#   - etl_pipeline_v2 (PIPELINE) - upstream dependency
# recommended_actions:
#   - Notify dashboard owners
#   - Run data quality checks
#   - Update downstream schemas
```

### **Release Gate (CI/CD Integration)**
```bash
./bin/blast release-check \
  --baseline snapshots/baseline.json \
  --current snapshots/current.json \
  --profile profiles/guard/analytics.guard.yaml \
  --report-file release_report.json \
  --markdown-file release_report.md

# Exit code 0 = PASS (approved), 1 = FAIL (blocked)
cat release_report.md  # Business-friendly summary
```

### **Ingest from Public API**
```bash
# Interactive wizard (recommended)
./bin/blast ingest wizard

# OR non-interactive
./bin/blast ingest web \
  --url https://api.github.com/events \
  --count 100 \
  --table github_events \
  --service web_service \
  --database webdb \
  --schema public

# Run again to detect schema drift
./bin/blast ingest web \
  --url https://api.github.com/events \
  --count 100 \
  --table github_events \
  --service web_service \
  --database webdb \
  --schema public
```

### **Drift Detection with Policies**
```bash
./bin/blast guard drift \
  baseline_snapshot.json \
  current_snapshot.json \
  --fail-on critical \
  --max-total 5 \
  --profile profiles/guard/analytics.guard.yaml

# Exit 0 if no violations, 1 if policy broken
```

### **Fetch Metadata Lineage**
```bash
./bin/blast lineage my_service.my_db.public.my_table

# Output:
# upstream:
#   - source_system.raw_db.raw_table
# downstream:
#   - analytics.marts.customer_summary
#   - reporting_dashboard.views.top_customers
```

### **Discover Catalog**
```bash
./bin/blast services                                    # List all services
./bin/blast databases --service my_service             # List databases in service
./bin/blast tables my_service.my_db.public             # List tables in database
./bin/blast tables get my_service.my_db.public.events  # Get table details
```

---

## 🤖 AI Integration (MCP Protocol)

Run Blast Radius as an MCP server for AI agents (Claude, etc.):

```bash
# Terminal 1: Start MCP server
./bin/blast mcp --ingestion-dir ./snapshots/ingestion

# Terminal 2: Test with demo client
python3 examples/mcp_demo.py --run-web-sync --service my_service
```

**Exposed Tools:**
- `blast_snapshot` - Capture metadata snapshot
- `blast_compare` - Compare two snapshots
- `blast_impact` - Analyze impact
- `blast_web_ingest_sync` - Ingest API data + detect drift
- `blast_service_discovery` - List available services
- `blast_release_check` - Run safety gate

**What agents can do:**
- Autonomously detect metadata drift
- Ingest external data and sync to OpenMetadata
- Recommend remediation actions
- Generate release approval reports
- Monitor metadata health continuously

---

## 🛠️ Complete Command Reference

For detailed command reference, see [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md)

**Quick examples:**
```bash
# Core operations
./bin/blast validate                                    # Check connectivity
./bin/blast snapshot --source db.schema                # Capture metadata
./bin/blast compare snap1.json snap2.json              # Find differences
./bin/blast impact snap1.json snap2.json               # Assess risk
./bin/blast release-check --baseline ... --current ... # Pre-release gate

# Discovery
./bin/blast services                                    # List services
./bin/blast databases --service X                      # List databases
./bin/blast tables my_service.db.schema                # List tables
./bin/blast lineage my_service.db.schema.table         # Show lineage

# Ingestion
./bin/blast ingest wizard                              # Interactive guide
./bin/blast ingest web --url ... --service ...         # Web API ingest

# Governance
./bin/blast guard drift snap1.json snap2.json          # Drift check
./bin/blast guard contracts snapshot.json              # Metadata quality

# Advanced
./bin/blast api --method GET --path /services          # Direct API calls
./bin/blast mcp                                        # Start MCP server
```

---

## 📚 Demos & Resources

### Quick Navigation
- **Getting started?** → Start with `python3 examples/demo-interactive.py`
- **Need a reference?** → See [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md)
- **Showing to others?** → Run `bash examples/full-demo.sh`
- **Learning patterns?** → Check [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md) (18 examples)
- **Integration patterns?** → See `examples/integration-examples.sh` (GitHub Actions, K8s, Airflow, etc.)

### Demo Scripts

| Script | Duration | Use Case |
|--------|----------|----------|
| `examples/demo-interactive.py` | 15-30 min | Hands-on learning, menu-driven exploration |
| `examples/full-demo.sh` | 5-10 min | Automated walkthrough, presentations |
| `examples/release_gate_demo.sh` | 2-3 min | Fast demo with pre-captured snapshots |
| `examples/integration-examples.sh` | Reference | CI/CD integrations (GitHub, GitLab, K8s, etc.) |

### Complete Resource Guide
See [DEMO_RESOURCES.md](DEMO_RESOURCES.md) for detailed breakdown of all demos and how to customize them.

---

## 📋 Configuration

### Environment Variables (Highest Priority)

```bash
# Required
OM_BASE_URL="http://localhost:8585/api/v1"    # OpenMetadata endpoint (MUST include /api/v1)
OM_JWT_TOKEN="your-jwt-token"                  # Auth token

# Optional (defaults below)
BR_DATABASE_FQN="postgres_default.public"      # Default database
BR_SNAPSHOT_DIRECTORY="./snapshots"            # Where to store snapshots
BR_OUTPUT_FORMAT="table"                       # "table" or "json"
```

### Config File (`.yaml` or `.yaml.example`)

```yaml
openmetadata:
  baseurl: "http://localhost:8585/api/v1"
  jwt_token: "your-jwt-token"

database:
  fqn: "postgres_default.public"

snapshot:
  directory: "./snapshots"

output:
  format: "table"  # or "json"
```

### Priority Order
1. Command-line flags
2. Environment variables (`OM_*`, `BR_*`)
3. Config file (`blast-radius.yaml`)
4. Defaults

---

## 📂 Project Structure

```
Blast-Radius/
├── cmd/
│   ├── main.go                          # Entry point
│   └── commands/                        # CLI commands
│       ├── snapshot.go                  # Snapshot capture
│       ├── compare.go                   # Snapshot comparison
│       ├── impact.go                    # Impact analysis
│       ├── guard.go                     # Drift detection & policies
│       ├── release_check.go             # Release gate
│       ├── ingest_web.go                # Web API ingestion
│       ├── lineage.go                   # Lineage fetch
│       ├── mcp.go                       # MCP server
│       ├── services.go, tables.go, ...  # Discovery commands
│       └── root.go                      # Root command setup
│
├── internal/
│   ├── config/
│   │   └── config.go                    # Config loading (env + file)
│   ├── openmetadata/
│   │   ├── client.go                    # HTTP REST API client
│   │   └── entities.go                  # Entity structures
│   ├── snapshot/
│   │   ├── snapshot.go                  # Snapshot creation
│   │   ├── diff.go                      # Comparison logic
│   │   └── storage.go                   # File I/O
│   ├── impact/
│   │   └── analyzer.go                  # Impact computation
│   └── mcp/
│       ├── server.go                    # MCP protocol handler
│       ├── protocol.go                  # MCP messages
│       └── ingestion.go                 # Ingestion tools
│
├── pkg/
│   └── netfetch/
│       └── client.go                    # HTTP fetcher for web APIs
│
├── examples/
│   ├── full-demo.sh                     # Complete walkthrough
│   ├── demo-interactive.py              # Interactive menu
│   ├── release_gate_demo.sh             # Fast demo
│   ├── integration-examples.sh          # Integration patterns
│   ├── mcp_demo.py                      # AI agent demo
│   └── payloads/                        # Example OpenMetadata payloads
│
├── profiles/guard/                      # Policy profiles (YAML)
│   ├── analytics.guard.yaml             # Example: analytics policies
│   └── finance.guard.yaml               # Example: finance policies
│
├── artifacts/                           # Demo artifacts
├── snapshots/                           # Saved metadata snapshots
│
├── DEMO_START_HERE.md                   # Quick start guide
├── DEMO_RESOURCES.md                    # Complete demo guide
├── DEMO_QUICK_REFERENCE.md              # One-page cheat sheet
├── DEMO_SNIPPETS.md                     # Copy-paste examples
│
├── go.mod && go.sum                     # Go dependencies
├── blast-radius.yaml.example            # Config template
├── .env.example                         # Env vars template
└── README.md                            # This file
```

---

## 🎓 Learning & Growth

### For Users

**Getting Started:**
1. Read: [DEMO_START_HERE.md](DEMO_START_HERE.md)
2. Run: `python3 examples/demo-interactive.py`
3. Reference: [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md)
4. Copy: [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md)

**Advanced Usage:**
- Create custom policy profiles (profiles/guard/*.yaml)
- Write integration scripts (see examples/integration-examples.sh)
- Extend MCP tools for AI agents
- Build CI/CD pipelines using release-check

### For Developers

**Key Concepts to Understand:**
- OpenMetadata entity model (services → databases → schemas → tables)
- Snapshot format (JSON structure with lineage + metadata)
- Risk scoring (how impact analyzer works)
- Policy evaluation (guard profile syntax)
- MCP protocol (stdio-based tool discovery)

**Code Entry Points:**
- `cmd/commands/root.go` - CLI command registration
- `internal/openmetadata/client.go` - API integration
- `internal/snapshot/snapshot.go` - Snapshot creation logic
- `internal/snapshot/diff.go` - Comparison engine
- `internal/impact/analyzer.go` - Impact computation
- `internal/mcp/server.go` - MCP protocol

**Adding New Features:**
1. Create new command in `cmd/commands/your_feature.go`
2. Register in `cmd/commands/root.go`
3. Implement business logic in `internal/`
4. Add tests in `*_test.go` files
5. Document in [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md)

### Roadmap Ideas

- [ ] Support more metadata sources (Collate, Apache Atlas, etc.)
- [ ] Extended policy language (conditional rules, state machines)
- [ ] Scheduled drift checks (background service mode)
- [ ] Metadata diff visualization (web UI)
- [ ] Slack/email notifications on release block
- [ ] Historical drift analytics / trends
- [ ] Terraform provider for policy management
- [ ] Multi-environment metadata sync

### Contributing

Pull requests welcome! Areas of interest:
- New OpenMetadata entity types
- Additional policy rule types
- Integration examples
- Documentation improvements
- Performance optimizations

---

## 🆘 Troubleshooting

### **Error: "invalid character '<' looking for beginning of value"**
**Cause:** `OM_BASE_URL` missing `/api/v1` endpoint

**Fix:**
```bash
# Wrong:
export OM_BASE_URL="http://localhost:8585"

# Correct:
export OM_BASE_URL="http://localhost:8585/api/v1"
```

### **Error: "unauthorized" or "invalid token"**
**Cause:** JWT token expired or invalid

**Fix:**
1. Generate new token in OpenMetadata UI (Settings → Integrations → Ingestion)
2. Update `OM_JWT_TOKEN` environment variable

### **Error: "database FQN not found"**
**Cause:** Database doesn't exist in OpenMetadata

**Fix:**
1. Verify database name: `./bin/blast databases --service <service_name>`
2. Create missing database: `./bin/blast create database --name <db> --service <service>`

### **No snapshots being created**
**Cause:** Snapshot directory doesn't exist or permissions issue

**Fix:**
```bash
mkdir -p ./snapshots
chmod 755 ./snapshots
./bin/blast snapshot --source my_service.my_db.public  # Try again
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file

---

## 🙋 Support & Feedback

- **Issues:** GitHub Issues
- **Discussions:** GitHub Discussions
- **Examples:** See [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md)
- **Integration help:** See [examples/integration-examples.sh](examples/integration-examples.sh)
