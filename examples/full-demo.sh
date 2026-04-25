#!/bin/bash

################################################################################
# DEMO SCRIPT 1: Blast Radius - Complete Feature Walkthrough
# 
# This script demonstrates all major features of Blast Radius in sequence.
# Prerequisites: OpenMetadata running, config set via env vars or config file
################################################################################

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

step() {
  echo -e "${BLUE}=== $1 ===${NC}"
}

success() {
  echo -e "${GREEN}✓ $1${NC}"
}

info() {
  echo -e "${YELLOW}ℹ $1${NC}"
}

error() {
  echo -e "${RED}✗ $1${NC}"
}

# Configuration
SERVICE="${DEMO_SERVICE:-my_service}"
DATABASE="${DEMO_DATABASE:-my_db}"
SCHEMA="${DEMO_SCHEMA:-public}"
WEB_URL="${DEMO_WEB_URL:-https://api.github.com/events}"
RECORD_COUNT="${DEMO_RECORD_COUNT:-50}"
BLAST_BIN="${BLAST_BIN:-./bin/blast}"

# Paths
DEMO_OUTPUT_DIR="demo-output"
BASELINE_SNAPSHOT=""
CURRENT_SNAPSHOT=""

# ===== Setup =====
setup() {
  step "Setup"
  
  # Build if needed
  if [[ ! -f "$BLAST_BIN" ]]; then
    info "Building blast binary..."
    go build -o "$BLAST_BIN" ./cmd
    success "Binary built"
  fi
  
  # Create demo output directory
  mkdir -p "$DEMO_OUTPUT_DIR"
  success "Demo output directory ready: $DEMO_OUTPUT_DIR"
}

# ===== Feature 1: Validate =====
demo_validate() {
  step "Feature 1: Validate OpenMetadata Connectivity"
  
  info "Running validation..."
  if $BLAST_BIN validate --output json > "$DEMO_OUTPUT_DIR/validate.json" 2>&1; then
    success "OpenMetadata is accessible"
    cat "$DEMO_OUTPUT_DIR/validate.json" | jq .
  else
    error "OpenMetadata validation failed - check config"
    return 1
  fi
}

# ===== Feature 2: Services =====
demo_services() {
  step "Feature 2: Discover Database Services"
  
  info "Listing available services..."
  if $BLAST_BIN services --output json > "$DEMO_OUTPUT_DIR/services.json" 2>&1; then
    success "Services discovered"
    cat "$DEMO_OUTPUT_DIR/services.json" | jq '.services[] | {name, serviceType}' 
  else
    error "Could not list services"
    return 1
  fi
}

# ===== Feature 3: Databases =====
demo_databases() {
  step "Feature 3: Explore Database Structure"
  
  info "Listing databases in $SERVICE..."
  if $BLAST_BIN databases --service "$SERVICE" --output json > "$DEMO_OUTPUT_DIR/databases.json" 2>&1; then
    success "Databases enumerated"
    cat "$DEMO_OUTPUT_DIR/databases.json" | jq '.databases[0:3]' 
  else
    info "Could not list databases (may not be available in OpenMetadata)"
  fi
}

# ===== Feature 4: Tables =====
demo_tables() {
  step "Feature 4: Catalog Tables"
  
  local SOURCE_FQN="$SERVICE.$DATABASE.$SCHEMA"
  info "Listing tables in $SOURCE_FQN..."
  
  if $BLAST_BIN tables "$SOURCE_FQN" --output json > "$DEMO_OUTPUT_DIR/tables.json" 2>&1; then
    success "Tables cataloged"
    cat "$DEMO_OUTPUT_DIR/tables.json" | jq '.tables[0:3]' 
  else
    info "Could not list tables (database may not be in OpenMetadata)"
  fi
}

# ===== Feature 5: Lineage =====
demo_lineage() {
  step "Feature 5: Metadata Lineage"
  
  local TABLE_FQN="$SERVICE.$DATABASE.$SCHEMA.example_table"
  info "Fetching lineage for $TABLE_FQN..."
  
  if $BLAST_BIN lineage "$TABLE_FQN" --output json > "$DEMO_OUTPUT_DIR/lineage.json" 2>&1; then
    success "Lineage retrieved"
    cat "$DEMO_OUTPUT_DIR/lineage.json" | jq .
  else
    info "No lineage found (table may not exist)"
  fi
}

# ===== Feature 6: Snapshot =====
demo_snapshot() {
  step "Feature 6: Capture Metadata Snapshot"
  
  local SOURCE_FQN="$SERVICE.$DATABASE.$SCHEMA"
  info "Capturing baseline snapshot of $SOURCE_FQN..."
  
  if $BLAST_BIN snapshot --source "$SOURCE_FQN" > "$DEMO_OUTPUT_DIR/snapshot1.log" 2>&1; then
    success "Snapshot captured"
    
    # Find the captured snapshot
    BASELINE_SNAPSHOT=$(find snapshots -name "*.json" -type f 2>/dev/null | head -1)
    
    if [[ -n "$BASELINE_SNAPSHOT" ]]; then
      info "Snapshot saved to: $BASELINE_SNAPSHOT"
      info "Snapshot size: $(du -h "$BASELINE_SNAPSHOT" | cut -f1)"
      cat "$BASELINE_SNAPSHOT" | jq 'keys' 
    fi
  else
    error "Could not capture snapshot"
    return 1
  fi
}

# ===== Feature 7: Web Ingest =====
demo_web_ingest() {
  step "Feature 7: Web API Ingestion"
  
  info "Ingesting from: $WEB_URL"
  info "Records to fetch: $RECORD_COUNT"
  
  if $BLAST_BIN ingest web \
    --url "$WEB_URL" \
    --count "$RECORD_COUNT" \
    --table demo_events \
    --service "$SERVICE" \
    --database webdb \
    --schema public \
    > "$DEMO_OUTPUT_DIR/ingest.log" 2>&1; then
    
    success "Web ingestion completed"
    
    # Find the newly created snapshot
    CURRENT_SNAPSHOT=$(find snapshots -name "*.json" -type f -newer "$DEMO_OUTPUT_DIR/ingest.log" 2>/dev/null | head -1)
    
    if [[ -n "$CURRENT_SNAPSHOT" ]]; then
      info "Ingestion snapshot: $CURRENT_SNAPSHOT"
    fi
  else
    error "Web ingestion failed"
    return 1
  fi
}

# ===== Feature 8: Compare =====
demo_compare() {
  step "Feature 8: Compare Snapshots"
  
  if [[ -z "$BASELINE_SNAPSHOT" ]]; then
    info "Skipping comparison (no baseline snapshot)"
    return 0
  fi
  
  if [[ -z "$CURRENT_SNAPSHOT" ]]; then
    info "Skipping comparison (no current snapshot)"
    return 0
  fi
  
  info "Comparing:"
  info "  Baseline: $BASELINE_SNAPSHOT"
  info "  Current: $CURRENT_SNAPSHOT"
  
  if $BLAST_BIN compare "$BASELINE_SNAPSHOT" "$CURRENT_SNAPSHOT" \
    --output json > "$DEMO_OUTPUT_DIR/compare.json" 2>&1; then
    
    success "Comparison complete"
    cat "$DEMO_OUTPUT_DIR/compare.json" | jq '.diff.summary'
  else
    error "Comparison failed"
    return 1
  fi
}

# ===== Feature 9: Impact =====
demo_impact() {
  step "Feature 9: Impact Analysis"
  
  if [[ -z "$BASELINE_SNAPSHOT" ]] || [[ -z "$CURRENT_SNAPSHOT" ]]; then
    info "Skipping impact analysis (need two snapshots)"
    return 0
  fi
  
  info "Analyzing impact of changes..."
  
  if $BLAST_BIN impact "$BASELINE_SNAPSHOT" "$CURRENT_SNAPSHOT" \
    --output json > "$DEMO_OUTPUT_DIR/impact.json" 2>&1; then
    
    success "Impact analysis complete"
    cat "$DEMO_OUTPUT_DIR/impact.json" | jq '.impact | {risk_level, impacted_assets: (.impacted_assets | length)}'
  else
    error "Impact analysis failed"
    return 1
  fi
}

# ===== Feature 10: Drift Detection =====
demo_drift() {
  step "Feature 10: Drift Detection with Policies"
  
  if [[ -z "$BASELINE_SNAPSHOT" ]] || [[ -z "$CURRENT_SNAPSHOT" ]]; then
    info "Skipping drift detection (need two snapshots)"
    return 0
  fi
  
  info "Running drift check (fail on CRITICAL only)..."
  
  # Allow non-zero exit
  set +e
  $BLAST_BIN guard drift "$BASELINE_SNAPSHOT" "$CURRENT_SNAPSHOT" \
    --fail-on critical \
    --output json > "$DEMO_OUTPUT_DIR/drift.json" 2>&1
  DRIFT_EXIT=$?
  set -e
  
  if [[ $DRIFT_EXIT -eq 0 ]]; then
    success "No critical drift detected"
  else
    info "Drift detected or policy violated (exit code: $DRIFT_EXIT)"
  fi
  
  cat "$DEMO_OUTPUT_DIR/drift.json" | jq . 2>/dev/null || true
}

# ===== Feature 11: Release Check =====
demo_release_check() {
  step "Feature 11: Release Safety Gate"
  
  if [[ -z "$BASELINE_SNAPSHOT" ]] || [[ -z "$CURRENT_SNAPSHOT" ]]; then
    info "Skipping release-check (need two snapshots)"
    return 0
  fi
  
  info "Running release safety gate..."
  
  # Allow non-zero exit
  set +e
  $BLAST_BIN release-check \
    --baseline "$BASELINE_SNAPSHOT" \
    --current "$CURRENT_SNAPSHOT" \
    --max-risk high \
    --report-file "$DEMO_OUTPUT_DIR/release_report.json" \
    --markdown-file "$DEMO_OUTPUT_DIR/release_report.md" \
    > "$DEMO_OUTPUT_DIR/release_check.log" 2>&1
  RELEASE_EXIT=$?
  set -e
  
  if [[ $RELEASE_EXIT -eq 0 ]]; then
    success "Release check PASSED ✓"
  else
    info "Release check FAILED ✗ (exit code: $RELEASE_EXIT)"
  fi
  
  info "Generated artifacts:"
  ls -lh "$DEMO_OUTPUT_DIR"/release_report.* 2>/dev/null || echo "  (no artifacts)"
}

# ===== Feature 12: MCP Server =====
demo_mcp() {
  step "Feature 12: MCP Server (AI Agent Integration)"
  
  info "MCP server would run with:"
  echo "  $BLAST_BIN mcp --ingestion-dir ./snapshots/ingestion"
  echo ""
  info "This enables:"
  echo "  - AI agents (Claude, etc.) to run Blast Radius tools"
  echo "  - Autonomous metadata discovery"
  echo "  - Programmatic drift detection"
  echo ""
  info "To test, run in another terminal and connect Claude:"
  echo "  python3 examples/mcp_demo.py --run-web-sync --service $SERVICE"
}

# ===== Summary =====
summary() {
  step "Demo Summary"
  
  info "All demo artifacts saved to: $DEMO_OUTPUT_DIR"
  echo ""
  echo "Generated files:"
  ls -lh "$DEMO_OUTPUT_DIR"/ 2>/dev/null | tail -n +2 || echo "  (none)"
  
  echo ""
  success "Demo complete! 🎉"
}

# ===== Main Flow =====
main() {
  setup
  demo_validate && \
  demo_services && \
  demo_databases && \
  demo_tables && \
  demo_lineage && \
  demo_snapshot && \
  demo_web_ingest && \
  demo_compare && \
  demo_impact && \
  demo_drift && \
  demo_release_check && \
  demo_mcp && \
  summary
}

# Run
main
