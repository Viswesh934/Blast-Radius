# Blast Radius - Quick Reference Card

**Print this out or bookmark it during demos!**

## Setup
```bash
go build -o bin/blast ./cmd
export OM_BASE_URL="http://localhost:8585"
export OM_JWT_TOKEN="your-jwt-token"
```

## Quick Command Reference

### ✅ Connectivity & Discovery
| Command | What It Does | When to Use |
|---------|-------------|-----------|
| `validate` | Check OpenMetadata accessibility | Start here - confirms setup works |
| `services` | List database services | See what data sources are available |
| `databases --service X` | List databases in a service | Browse schema structure |
| `tables <fqn>` | List tables in database | Explore catalog |
| `lineage <table>` | Fetch upstream/downstream | Show data flow |

### 📸 Snapshots (Point-in-time capture)
| Command | What It Does | When to Use |
|---------|-------------|-----------|
| `snapshot --source X` | Capture metadata state | Create baseline for comparison |
| `snapshot` (repeated) | Capture again later | Show drift detection |

### 🔍 Analysis (Compare & Impact)
| Command | What It Does | When to Use |
|---------|-------------|-----------|
| `compare snap1 snap2` | Show what changed | Identify added/deleted/modified entities |
| `impact snap1 snap2` | Compute downstream risk | Assess business impact |
| `guard drift snap1 snap2` | Detect drift with policy | CI/CD metadata validation |

### 🚀 Ingestion (External Data → Metadata)
| Command | What It Does | When to Use |
|---------|-------------|-----------|
| `ingest wizard` | Interactive guide | Best for live demos - user-friendly |
| `ingest web --url X` | Fetch JSON API, sync metadata | Ingest public data sources |

### 🛡️ Release Gate (CI/CD)
| Command | What It Does | When to Use |
|---------|-------------|-----------|
| `release-check` | All-in-one safety gate | Pre-release approval workflow |
| Shows drift, impact, metadata completeness | Outputs JSON + Markdown | Generate audit trail |

### 🔧 Advanced
| Command | What It Does | When to Use |
|---------|-------------|-----------|
| `api --path X` | Direct OpenMetadata API | Advanced operations |
| `mcp` | AI agent server | Let Claude/GPT control Blast Radius |
| `glossary create` | Create business terms | Add semantic metadata |

---

## 🎬 Demo Flows

### **Flow 1: "Show Drift Detection" (5 min)**
1. `snapshot --source db.schema.public` → Baseline
2. `ingest web --url ... ` → Ingest new data (fakes schema change)
3. `compare snapshot_OLD snapshot_NEW` → Show differences
4. `impact snapshot_OLD snapshot_NEW` → Show risk
5. `guard drift snapshot_OLD snapshot_NEW` → Policy enforcement

### **Flow 2: "Release Gate CI/CD" (3 min)**
1. Have two snapshots ready (old + new)
2. `release-check --baseline snap1 --current snap2 --profile guards/analytics.guard.yaml --report-file report.json --markdown-file report.md`
3. Show artifacts (JSON + Markdown report)
4. Explain exit code (0=pass, 1=fail)

### **Flow 3: "Data Lineage" (2 min)**
1. `lineage <table_fqn>`
2. Show upstream sources
3. Show downstream consumers

### **Flow 4: "Web Ingestion Wizard" (5 min)**
1. `ingest wizard` → Walks through everything interactively

---

## 📊 Common Use Cases

### "Show connection works"
```bash
./bin/blast validate --output json
```

### "Show catalogue"
```bash
./bin/blast tables my_service.my_db.public --output json
```

### "Capture state before change"
```bash
./bin/blast snapshot --source my_service.my_db.public
```

### "Detect what broke"
```bash
./bin/blast compare snapshot_before.json snapshot_after.json
```

### "Show impact to business"
```bash
./bin/blast impact snapshot_before.json snapshot_after.json
```

### "Block bad release"
```bash
./bin/blast release-check --baseline before --current after --profile my.guard.yaml
```

---

## 🎯 Output Formats

### Human-readable (default)
```bash
./bin/blast tables my_service.db.schema
```

### Machine-readable (scripts)
```bash
./bin/blast tables my_service.db.schema --output json | jq .
```

---

## 📁 Directory Structure

```
snapshots/
  sources/           # Full DB snapshots
    my_service/
      my_db/public/
        snapshot_TIMESTAMP.json
  tables/            # Per-table snapshots  
    my_service/
      my_db.public.table/
        snapshot_TIMESTAMP.json

profiles/guard/
  analytics.guard.yaml    # Drift policies
  finance.guard.yaml      # Finance policies

artifacts/
  release-check-fail.json # Demo output
  release-check-fail.md
```

---

## 🚀 Demo Scripts

**Bash - Full automated demo:**
```bash
bash examples/full-demo.sh
```

**Python - Interactive menu:**
```bash
python3 examples/demo-interactive.py
```

**Shell - Original demo:**
```bash
bash examples/release_gate_demo.sh
```

---

## 💡 Pro Tips

1. **Start with `validate`** - Confirms setup before anything else
2. **Use `ingest wizard`** - Most engaging for audiences
3. **Pre-capture snapshots** - Speeds up live demos
4. **Use `--output json`** - Great for showing structure  
5. **Show the Markdown report** - Business-friendly output
6. **Explain exit codes** - 0=pass (green), 1=fail (red)
7. **Use policy profiles** - Shows enforcement capability

---

## 🔗 Next Steps

- **For your slides**: Copy snippets from `DEMO_SNIPPETS.md`
- **For automation**: Use `full-demo.sh` or customize
- **For learning**: Run `demo-interactive.py` and explore
- **For CI/CD**: Integrate `release-check` into GitHub Actions/GitLab CI

---

## Questions?

- **OpenMetadata connectivity issues?** → Run `validate` first
- **No data in tables?** → Use `ingest wizard` to add some
- **Want to publish changes to OpenMetadata?** → Snapshots auto-publish when captured
- **Need real drift to show?** → Run `ingest web` twice with same URL
- **Want custom policies?** → Edit `profiles/guard/*.guard.yaml`

