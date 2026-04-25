# 📚 Blast Radius Demo Resources

Everything you need to demo Blast Radius features to any audience!

## 📖 Start Here

**First time?** Read this in order:
1. [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md) - 2 min overview of all commands
2. [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md) - Copy-paste ready examples for each feature
3. Pick a demo script below based on your needs

---

## 🎬 Demo Scripts

### 1. **Full Automated Demo** (Bash)
📄 **File:** `examples/full-demo.sh`
- ⏱️ **Duration:** 5-10 minutes
- 📝 **Format:** Automated walkthrough with colored output
- 🎯 **Best for:** Presenting to groups, recording videos
- 🚀 **Features Shown:** All 12 core features in sequence

**Run it:**
```bash
bash examples/full-demo.sh
```

**What it shows:**
- Validate connectivity
- List services & tables
- Capture snapshots
- Detect drift
- Analyze impact
- Run release gate
- Generate reports

### 2. **Interactive Demo Menu** (Python)
📄 **File:** `examples/demo-interactive.py`
- ⏱️ **Duration:** Self-paced
- 📝 **Format:** Interactive menu (choose what to demo)
- 🎯 **Best for:** Hands-on learning, exploring features
- 🚀 **Features:** Pick individual demos or run all

**Run it:**
```bash
python3 examples/demo-interactive.py
```

**Features:**
- Menu-driven interface
- Run individual demos in any order
- All output formatted nicely
- No pre-recorded snapshots needed

### 3. **Original Release Gate Demo** (Bash)
📄 **File:** `examples/release_gate_demo.sh`
- ⏱️ **Duration:** 3-5 minutes
- 📝 **Format:** Focused demo of release gate feature
- 🎯 **Best for:** Showing CI/CD integration
- 🚀 **Features:** Uses pre-captured snapshots (fast!)

**Run it:**
```bash
bash examples/release_gate_demo.sh
```

**What it shows:**
- Strict policy (fails)
- Relaxed policy (passes)
- Generated report artifacts

### 4. **Integration Patterns** (Bash/Code Examples)
📄 **File:** `examples/integration-examples.sh`
- ⏱️ **Duration:** Reference material
- 📝 **Format:** 12 different integration examples
- 🎯 **Best for:** Developers building with Blast Radius
- 🚀 **Features:** GitHub Actions, GitLab CI, Kubernetes, Airflow, etc.

**Browse examples:**
```bash
bash examples/integration-examples.sh
```

**Includes:**
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

## 📋 Copy-Paste Reference

### **Quick Command Reference**
📄 **File:** `DEMO_QUICK_REFERENCE.md`

One-page cheat sheet with:
- Every command in a table
- Use cases for each
- Common demo flows
- Setup instructions
- Pro tips

**Perfect for:** Keeping on your clipboard during live demos!

### **Complete Snippet Library**
📄 **File:** `DEMO_SNIPPETS.md`

18 sections covering:
1. Validate OpenMetadata connectivity
2. Discover services
3. Explore databases
4. Browse catalog
5. Fetch lineage
6. Capture snapshots
7. Compare snapshots
8. Analyze impact
9. Web ingestion
10. Drift detection with policies
11. Contract checks
12. Release safety gates
13. Table operations
14. Glossary management
15. Direct API calls
16. MCP server for AI agents
17. Service creation
18. Complete end-to-end demo

**Perfect for:** Finding the exact command you need!

---

## 🎯 Quick Navigation

**"Show me connectivity works"**
→ Run: `./bin/blast validate`

**"Show the data catalogue"**
→ Run: `./bin/blast tables my_service.my_db.public`

**"Show we can ingest external data"**
→ Run: `python3 examples/demo-interactive.py` → Choose #4

**"Show drift detection"**
→ Run: `bash examples/full-demo.sh` (shows everything)

**"Show release gate for CI/CD"**
→ Run: `bash examples/release_gate_demo.sh`

**"Let users explore themselves"**
→ Run: `python3 examples/demo-interactive.py` (menu-driven)

---

## 🛠️ Setup

All scripts require Blast Radius to be built first:

```bash
# Build once
go build -o bin/blast ./cmd

# Set environment (or use config file)
export OM_BASE_URL="http://localhost:8585"
export OM_JWT_TOKEN="your-jwt-token"
export DEMO_SERVICE="my_service"
export DEMO_DATABASE="my_db"
export DEMO_SCHEMA="public"

# Then run any script above
```

---

## 📊 Demo Flow Recommendations

### **For Executives (5 min)**
1. `./bin/blast validate` → Shows connection works
2. `./bin/blast services` → Show available data
3. `bash examples/release_gate_demo.sh` → Show business value (prevents bad releases)

### **For Data Engineers (15 min)**
1. `python3 examples/demo-interactive.py` → Menu #1-3 (explore)
2. Menu #4 → Ingest demo
3. Menu #5-7 → Drift & impact
4. Menu #9 → Release gate

### **For Platform Engineers (20 min)**
1. `bash examples/full-demo.sh` → Full walkthrough
2. `bash examples/integration-examples.sh` → Integration patterns
3. Show `DEMO_SNIPPETS.md` → Copy-paste for their stack

### **For Hands-On Workshop (45 min)**
1. `python3 examples/demo-interactive.py` → Guided exploration
2. Participants run commands at their own pace
3. `DEMO_QUICK_REFERENCE.md` → Distributed to attendees
4. `examples/integration-examples.sh` → Homework inspiration

---

## 🎬 Recording Tips

To record a demo video:

```bash
# Terminal 1: Start recording with asciinema
asciinema rec demo-recording.cast

# Terminal 2: Run the demo
bash examples/full-demo.sh

# Export as MP4
asciinema upload demo-recording.cast
```

Or use the interactive demo for screen capture:
```bash
# Screen recording tool of choice
bash examples/full-demo.sh  # or python3 examples/demo-interactive.py
```

---

## 🔧 Customization

### Use Your Own Database
Edit environment variables before running:

```bash
export DEMO_SERVICE="your_service"
export DEMO_DATABASE="your_database"
export DEMO_SCHEMA="your_schema"
export DEMO_WEB_URL="https://your-api.com/data"

bash examples/full-demo.sh
```

### Use Custom Policies
Create your own guard profile:

```bash
# Create profiles/guard/custom.guard.yaml
cat > profiles/guard/custom.guard.yaml << EOF
drift:
  failOn: critical
  maxTotal: 10

contracts:
  minScore: 80
  requireOwner: true
  requireTableDescription: true
EOF

# Use it in demos
./bin/blast release-check ... --profile profiles/guard/custom.guard.yaml
```

### Modify Demo Scripts
All demo scripts are commented and easy to customize. Just edit:
- `examples/full-demo.sh` → 200 lines, clear structure
- `examples/demo-interactive.py` → 400 lines, object-oriented
- `DEMO_SNIPPETS.md` → Copy sections as needed

---

## 🚀 Advanced Usage

### Pre-capture Snapshots for Speed
```bash
# Before demo, capture baseline
./bin/blast snapshot --source my_service.my_db.public
find snapshots -name "*.json" | head -1 > BASELINE.txt

# In demo, use pre-captured:
./bin/blast compare $(cat BASELINE.txt) snapshots/current.json
```

### Generate Static Reports
```bash
# Generate markdown reports for slides/docs
./bin/blast release-check \
  --baseline snap1.json \
  --current snap2.json \
  --report-file report.json \
  --markdown-file report.md

# Embed report.md in your presentation
cat report.md  # Copy to slides
```

### Chain Multiple Demos
```bash
# Run validation → snapshot → drift in sequence
./bin/blast validate && \
./bin/blast snapshot --source db.schema.public && \
./bin/blast snapshot --source db.schema.public && \
./bin/blast guard drift \
  $(find snapshots -name "*.json" | tail -2 | head -1) \
  $(find snapshots -name "*.json" | tail -1)
```

---

## ❓ FAQ

**Q: Which demo should I run first?**
A: Start with `examples/demo-interactive.py` - it's menu-driven and most forgiving.

**Q: Can I use this with a live OpenMetadata instance?**
A: Yes! Set up environment variables pointing to your instance and all demos work.

**Q: What if OpenMetadata isn't available?**
A: The interactive demo handles failures gracefully. Use pre-captured snapshots from `release_gate_demo.sh`.

**Q: How long does each demo take?**
A: 3-5 min (release gate), 5-10 min (full demo), 15-30 min (interactive), 45+ min (workshop).

**Q: Can I modify the demo scripts?**
A: Yes! They're all MIT licensed and meant to be customized.

**Q: What's the best way to show this to a large audience?**
A: Record `full-demo.sh` or use `interactive.py` with screen sharing and walk through features.

---

## 📞 Need Help?

- **Blast Radius docs:** See [README.md](../README.md)
- **Code snippets:** Check [DEMO_SNIPPETS.md](DEMO_SNIPPETS.md)
- **Quick reference:** [DEMO_QUICK_REFERENCE.md](DEMO_QUICK_REFERENCE.md)
- **Integration patterns:** `bash examples/integration-examples.sh`

---

## 🎉 Ready to Demo!

Pick your demo based on your audience and time:

| Audience | Time | Script |
|----------|------|--------|
| Executives | 5 min | `validate` + `services` + `release_gate_demo.sh` |
| Developers | 15 min | `demo-interactive.py` |
| Engineers | 20 min | `full-demo.sh` + `integration-examples.sh` |
| Workshop | 45+ min | `demo-interactive.py` + hands-on exploration |

**Go forth and demo! 🚀**

