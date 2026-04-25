# 🎯 Start Demo Here!

**Welcome!** Everything you need to demo Blast Radius is organized below.

## 🚀 Quick Start (Pick One)

### Option 1: **Let Me Explore Interactively** (15-30 min, hands-on)
```bash
python3 examples/demo-interactive.py
```
✨ Menu-driven interface. Pick features to demo in any order.

### Option 2: **Run Full Automated Demo** (5-10 min, presentation)
```bash
bash examples/full-demo.sh
```
📹 Walks through all features in sequence. Great for recording or group demos.

### Option 3: **Just Show Release Gate** (2-3 min, CI/CD focused)
```bash
bash examples/release_gate_demo.sh
```
⚡ Fast demo using pre-captured snapshots. Show the business value immediately.

---

## 📚 Demo Resources

### 📖 **For Reading First**
| File | Purpose | Length |
|------|---------|--------|
| [DEMO_RESOURCES.md](DEMO_RESOURCES.md) | Overview of all demos & how to use | 5 min |
| [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md) | One-page command cheat sheet | 2 min |
| [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md) | 18 copy-paste ready examples | 10 min |

### 🎬 **Demo Scripts to Run**
| File | Type | Duration | Audience |
|------|------|----------|----------|
| `examples/full-demo.sh` | Bash | 5-10 min | Everyone (presentation) |
| `examples/demo-interactive.py` | Python | Self-paced | Hands-on learners |
| `examples/release_gate_demo.sh` | Bash | 2-3 min | CI/CD focused |
| `examples/integration-examples.sh` | Bash | Reference | Developers/engineers |

### 💻 **Code Integration Examples**
See `examples/integration-examples.sh` for:
- Shell script patterns
- GitHub Actions workflow
- GitLab CI pipeline  
- Terraform integration
- Kubernetes CronJob
- Docker setup
- Airflow DAG
- Kafka/event-driven
- Python client library

---

## 📋 Command Cheat Sheet

### Connectivity & Discovery
```bash
./bin/blast validate                 # Check setup works
./bin/blast services                 # List databases
./bin/blast tables my_svc.db.schema  # List tables
./bin/blast lineage <table_fqn>      # Show lineage
```

### Snapshots & Comparison
```bash
./bin/blast snapshot --source my_svc.db.schema
./bin/blast compare snap_old.json snap_new.json
./bin/blast impact snap_old.json snap_new.json
```

### Release Gate (What to Show in CI/CD!)
```bash
./bin/blast release-check \
  --baseline snap_old.json \
  --current snap_new.json \
  --profile profiles/guard/analytics.guard.yaml
```

### Ingest & Drift
```bash
./bin/blast ingest wizard                    # Interactive
./bin/blast guard drift snap1.json snap2.json --fail-on critical
```

→ See [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md) for full reference

---

## 🎯 Demo Scenarios

### "I have 2 minutes"
```bash
./bin/blast validate
```
Just show connectivity works.

### "I have 5 minutes"
```bash
bash examples/release_gate_demo.sh
```
Show the complete release gate feature with pre-recorded snapshots.

### "I have 15 minutes"
```bash
python3 examples/demo-interactive.py
# Choose: 1, 2, 4, 5, 9
```
Walks through: validate → services → ingest → compare → release gate

### "I have 30 minutes"
```bash
bash examples/full-demo.sh
```
Everything: validation, snapshots, ingestion, drift, impact, release gates

### "I want to teach people"
```bash
python3 examples/demo-interactive.py
```
Menu-driven so people explore at their own pace, or live code through features.

---

## 🛠️ Setup Required

**One time only:**
```bash
# 1. Build binary
go build -o bin/blast ./cmd

# 2. Set environment (or use config file)
export OM_BASE_URL="http://localhost:8585"
export OM_JWT_TOKEN="your-jwt-token"

# Optional: customize where data comes from
export DEMO_SERVICE="my_service"
export DEMO_DATABASE="my_db"
export DEMO_SCHEMA="public"
```

Then run any demo script above!

---

## 💡 Pro Tips

1. **Start with `demo-interactive.py`** - Most forgiving, shows all features
2. **Read `DEMO_QUICK_REFERENCE.md`** first - Know what commands exist
3. **Pre-capture snapshots** for live demos (faster, no network delays)
4. **Use policy profiles** - Shows governance/enforcement capability  
5. **Show Markdown reports** - Business-friendly output for non-technical viewers

---

## ✅ What Each Demo Shows

### Interactive Demo (`demo-interactive.py`)
- ✅ Validate connectivity
- ✅ Discover services & tables
- ✅ Capture snapshots
- ✅ Compare changes
- ✅ Analyze impact
- ✅ Detect drift
- ✅ Fetch lineage
- ✅ Release gate

### Full Demo (`full-demo.sh`)
- ✅ Everything above, plus:
- ✅ Ingest from web API
- ✅ Save reports to disk
- ✅ Show structured output
- ✅ All logs & artifacts

### Release Gate Demo (`release_gate_demo.sh`)
- ✅ Two policy scenarios (fail/pass)
- ✅ Shows structured reports
- ✅ Markdown report generation
- ✅ Exit codes matter

---

## 🎬 Demo Videos

To record your demo:

```bash
# Using asciinema
asciinema rec my_demo.cast
bash examples/full-demo.sh
# Press Ctrl-D to stop recording
# Upload: asciinema upload my_demo.cast

# Or use your screen recorder
# Just run: bash examples/full-demo.sh
```

---

## 📁 File Organization

```
Blast-Radius/
  DEMO_START_HERE.md ..................... ← You are here!
  DEMO_RESOURCES.md ..................... Detailed guide
  DEMO_QUICK_REFERENCE.md .............. One-page cheat sheet
  DEMO_SNIPPETS.md ..................... Copy-paste examples
  
  examples/
    full-demo.sh ....................... 5-10 min walkthrough
    demo-interactive.py ............... Self-paced menu
    release_gate_demo.sh .............. Fast CI/CD demo
    integration-examples.sh ........... Integration patterns
    
    mcp_demo.py ....................... AI agent demo
    release_gate_demo.sh .............. (original)
```

---

## ❓ FAQ

**Q: Where's the best place to start?**
A: Read `DEMO_QUICK_REFERENCE.md` (2 min), then run `demo-interactive.py` (15 min)

**Q: What if I don't have OpenMetadata running?**
A: Use `release_gate_demo.sh` which uses pre-captured snapshots

**Q: Can I modify these for my use case?**
A: Yes! Everything is open source. Edit the scripts as needed.

**Q: Which demo shows the most value?**
A: `release_gate_demo.sh` - Shows how it prevents bad releases (ROI!)

**Q: Can I use these in presentations?**
A: Yes! Record them with `asciinema` or your screen recorder

**Q: What's the quickest way to show connectivity?**
A: `./bin/blast validate`

**Q: How do I show ingestion?**
A: `python3 examples/demo-interactive.py` → Choose option #4

---

## 🚀 Ready? Let's Go!

```bash
# Pick your path:

# Path 1: Explore interactively
python3 examples/demo-interactive.py

# Path 2: Automated walkthrough
bash examples/full-demo.sh

# Path 3: Fast CI/CD demo
bash examples/release_gate_demo.sh

# Path 4: Quick verify
./bin/blast validate
```

**Questions?** Check the files listed above or look at the README.md for more context.

**Good luck with your demo! 🎉**

