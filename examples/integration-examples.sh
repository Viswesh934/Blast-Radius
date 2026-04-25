#!/bin/bash

################################################################################
# DEMO SCRIPT 3: Blast Radius - REST API Integration Examples
#
# This script shows how to integrate Blast Radius commands into external systems
# via shell scripts, CI/CD pipelines, and API calls.
#
# These examples are meant to be copy-pasted and modified for specific use cases.
################################################################################

################################################################################
# ===== PART 1: Shell Script Integration =====
################################################################################

# Example 1.1: Simple drift detection in a CI pipeline
detect_drift_ci() {
  local baseline="$1"
  local current="$2"
  local policy_profile="$3"
  
  echo "[CI] Running drift check..."
  
  if ./bin/blast guard drift "$baseline" "$current" \
      --fail-on critical \
      --profile "$policy_profile" \
      --output json > drift_result.json; then
    echo "[CI] ✓ Drift check PASSED"
    return 0
  else
    echo "[CI] ✗ Drift check FAILED"
    cat drift_result.json
    return 1
  fi
}

# Example 1.2: Automated snapshot capture and comparison
compare_metadata_versions() {
  local db_fqn="$1"
  local run_name="${2:-auto_check}"
  
  echo "[SNAPSHOT] Capturing baseline..."
  BASELINE=$(./bin/blast snapshot --source "$db_fqn" 2>&1 | tail -1)
  
  # Simulate some time passing / changes happening
  echo "[SNAPSHOT] Waiting for changes..."
  sleep 2
  
  echo "[SNAPSHOT] Capturing current state..."
  CURRENT=$(./bin/blast snapshot --source "$db_fqn" 2>&1 | tail -1)
  
  echo "[SNAPSHOT] Comparing..."
  ./bin/blast compare "$BASELINE" "$CURRENT" --output json > "${run_name}_comparison.json"
  
  echo "[SNAPSHOT] Report saved to ${run_name}_comparison.json"
}

# Example 1.3: Conditional release approval
release_gate_workflow() {
  local baseline="$1"
  local current="$2"
  local profile="$3"
  
  echo "========================================="
  echo "RELEASE GATE WORKFLOW"
  echo "========================================="
  
  # Run release check
  echo "Step 1: Running release safety gate..."
  if ! ./bin/blast release-check \
      --baseline "$baseline" \
      --current "$current" \
      --profile "$profile" \
      --report-file release_report.json \
      --markdown-file release_report.md; then
    echo "Release BLOCKED ❌"
    echo ""
    echo "Report:"
    cat release_report.md
    return 1
  fi
  
  echo "Release APPROVED ✅"
  return 0
}

# Example 1.4: Orchestrate full metadata health check
metadata_health_check() {
  local service="$1"
  local db="$2"
  local schema="$3"
  
  local fqn="${service}.${db}.${schema}"
  local report_dir="health-check-$(date +%s)"
  
  mkdir -p "$report_dir"
  
  echo "[HEALTH] Starting metadata health check for $fqn"
  
  # 1. Validate connectivity
  echo "[HEALTH] Checking OpenMetadata connectivity..."
  if ! ./bin/blast validate > "$report_dir/connectivity.txt" 2>&1; then
    echo "[HEALTH] ✗ OpenMetadata not accessible"
    return 1
  fi
  echo "[HEALTH] ✓ Connected"
  
  # 2. Capture snapshot
  echo "[HEALTH] Capturing metadata snapshot..."
  if ! ./bin/blast snapshot --source "$fqn" > "$report_dir/snapshot.txt" 2>&1; then
    echo "[HEALTH] ✗ Could not capture snapshot"
    return 1
  fi
  echo "[HEALTH] ✓ Snapshot captured"
  
  # 3. Check contracts (completeness)
  echo "[HEALTH] Checking metadata completeness..."
  if ! ./bin/blast guard contracts "$(find snapshots -name "*.json" -type f | head -1)" \
      --output json > "$report_dir/contracts.json" 2>&1; then
    echo "[HEALTH] ⚠ Some metadata issues found"
  else
    echo "[HEALTH] ✓ Metadata contracts satisfied"
  fi
  
  echo "[HEALTH] Report saved to: $report_dir"
  return 0
}

################################################################################
# ===== PART 2: GitHub Actions Integration =====
################################################################################

# Save this as .github/workflows/metadata-check.yml
github_actions_example() {
  cat << 'EOF'
name: Metadata Release Gate

on:
  pull_request:
    paths:
      - 'data/schemas/**'

jobs:
  metadata-check:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build Blast
        run: go build -o bin/blast ./cmd
      
      - name: Capture baseline snapshot
        env:
          OM_BASE_URL: ${{ secrets.OM_BASE_URL }}
          OM_JWT_TOKEN: ${{ secrets.OM_JWT_TOKEN }}
        run: |
          ./bin/blast snapshot --source my_service.my_db.public \
            > baseline.log 2>&1
          SNAPSHOT=$(find snapshots -name "*.json" | head -1)
          echo "BASELINE=$SNAPSHOT" >> $GITHUB_ENV
      
      - name: Run release gate
        env:
          OM_BASE_URL: ${{ secrets.OM_BASE_URL }}
          OM_JWT_TOKEN: ${{ secrets.OM_JWT_TOKEN }}
        run: |
          ./bin/blast release-check \
            --baseline "${{ env.BASELINE }}" \
            --current "${{ env.BASELINE }}" \
            --profile profiles/guard/analytics.guard.yaml \
            --report-file report.json \
            --markdown-file report.md
      
      - name: Comment PR with results
        if: failure()
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const report = fs.readFileSync('report.md', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: report
            });
EOF
}

################################################################################
# ===== PART 3: GitLab CI Integration =====
################################################################################

# Save this as .gitlab-ci.yml
gitlab_ci_example() {
  cat << 'EOF'
stages:
  - check
  - gate

metadata-drift-check:
  stage: check
  image: golang:1.21
  script:
    - go build -o bin/blast ./cmd
    - ./bin/blast validate
    - ./bin/blast snapshot --source my_service.my_db.public
  artifacts:
    paths:
      - snapshots/
    reports:
      dotenv: metadata.env
  only:
    - merge_requests

release-gate:
  stage: gate
  image: golang:1.21
  script:
    - go build -o bin/blast ./cmd
    - |
      ./bin/blast release-check \
        --baseline snapshots/baseline.json \
        --current snapshots/current.json \
        --profile profiles/guard/analytics.guard.yaml \
        --report-file report.json \
        --markdown-file report.md
    - cat report.md
  artifacts:
    paths:
      - report.json
      - report.md
    reports:
      sast: report.json
  allow_failure: true
EOF
}

################################################################################
# ===== PART 4: Terraform Integration =====
################################################################################

# Example: Run drift check as part of Terraform deployment
terraform_integration_example() {
  cat << 'EOF'
# main.tf - Run metadata check before Terraform apply

resource "null_resource" "metadata_validation" {
  provisioner "local-exec" {
    command = <<-EOT
      ./bin/blast release-check \
        --baseline snapshots/tf-baseline.json \
        --current snapshots/tf-current.json \
        --profile profiles/guard/analytics.guard.yaml \
        --report-file release_report.json \
        --markdown-file release_report.md
      
      if [ $? -ne 0 ]; then
        echo "Metadata check failed - Terraform apply blocked"
        exit 1
      fi
    EOT
  }
}

resource "aws_eks_cluster" "my_cluster" {
  depends_on = [null_resource.metadata_validation]
  # ... rest of cluster config
}
EOF
}

################################################################################
# ===== PART 5: Kubernetes Integration =====
################################################################################

# Save this as k8s-metadata-check-job.yaml
kubernetes_integration_example() {
  cat << 'EOF'
apiVersion: batch/v1
kind: CronJob
metadata:
  name: blast-radius-drift-check
  namespace: metadata-tools
spec:
  schedule: "0 * * * *"  # Every hour
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: blast
            image: golang:1.21
            command:
            - /bin/sh
            - -c
            - |
              git clone https://github.com/Viswesh934/blast-radius.git
              cd blast-radius
              go build -o bin/blast ./cmd
              
              export OM_BASE_URL=$OM_BASE_URL
              export OM_JWT_TOKEN=$OM_JWT_TOKEN
              
              ./bin/blast snapshot --source my_service.my_db.public
              
              # Store result for analysis
              cp snapshots/*/public/*.json /output/snapshot-$(date +%s).json
            
            env:
            - name: OM_BASE_URL
              valueFrom:
                configMapKeyRef:
                  name: blast-config
                  key: om-base-url
            - name: OM_JWT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: blast-secrets
                  key: om-jwt-token
            
            volumeMounts:
            - name: output
              mountPath: /output
          
          volumes:
          - name: output
            persistentVolumeClaim:
              claimName: blast-snapshots-pvc
          
          restartPolicy: OnFailure
EOF
}

################################################################################
# ===== PART 6: Docker Integration =====
################################################################################

# Save this as Dockerfile
dockerfile_example() {
  cat << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o bin/blast ./cmd

FROM alpine:latest
RUN apk add --no-cache bash jq curl
COPY --from=builder /app/bin/blast /usr/local/bin/blast
COPY --from=builder /app/examples /examples
COPY --from=builder /app/profiles /profiles

ENV OM_BASE_URL=${OM_BASE_URL}
ENV OM_JWT_TOKEN=${OM_JWT_TOKEN}

ENTRYPOINT ["blast"]
EOF

  # Usage:
  cat << 'EOF'
# Build image
docker build -t blast-radius .

# Run snapshot
docker run -e OM_BASE_URL=http://openmetadata:8585 \
           -e OM_JWT_TOKEN=token \
           blast-radius snapshot --source my_service.my_db.public

# Run CI job
docker run -e OM_BASE_URL=http://openmetadata:8585 \
           -e OM_JWT_TOKEN=token \
           -v /data/snapshots:/app/snapshots \
           blast-radius release-check \
           --baseline /app/snapshots/baseline.json \
           --current /app/snapshots/current.json
EOF
}

################################################################################
# ===== PART 7: Airflow DAG Integration =====
################################################################################

# Save this as dags/metadata_drift_check_dag.py
airflow_dag_example() {
  cat << 'EOF'
from datetime import datetime, timedelta
from airflow import DAG
from airflow.operators.bash import BashOperator
from airflow.operators.python import PythonOperator
import json

default_args = {
    'owner': 'data-platform',
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

dag = DAG(
    'metadata_drift_check',
    default_args=default_args,
    description='Hourly metadata drift detection',
    schedule_interval='@hourly',
    start_date=datetime(2024, 1, 1),
    catchup=False,
)

# Task 1: Capture snapshot
capture_snapshot = BashOperator(
    task_id='capture_snapshot',
    bash_command='''
    export PATH=/opt/blast:$PATH
    blast snapshot --source my_service.my_db.public
    ''',
    dag=dag,
)

# Task 2: Compare with previous
compare_snapshots = BashOperator(
    task_id='compare_snapshots',
    bash_command='''
    export PATH=/opt/blast:$PATH
    BASELINE=$(find snapshots -name "*.json" | tail -2 | head -1)
    CURRENT=$(find snapshots -name "*.json" | tail -1)
    blast compare $BASELINE $CURRENT --output json > /tmp/comparison.json
    ''',
    dag=dag,
)

# Task 3: Alert if critical changes
def check_drift():
    with open('/tmp/comparison.json', 'r') as f:
        result = json.load(f)
    
    if result['diff']['summary']['critical'] > 0:
        raise Exception(f"Critical drift detected: {result['diff']['summary']}")

alert_on_drift = PythonOperator(
    task_id='alert_on_drift',
    python_callable=check_drift,
    dag=dag,
)

capture_snapshot >> compare_snapshots >> alert_on_drift
EOF
}

################################################################################
# ===== PART 8: Event-Driven (Kafka/SNS) =====
################################################################################

# Example: Trigger Blast on OpenMetadata change events
event_driven_example() {
  cat << 'EOF'
#!/usr/bin/env python3
# Script to listen for OpenMetadata events and trigger drift check

import json
import subprocess
from kafka import KafkaConsumer

def on_table_change(event):
    """Triggered when an OpenMetadata table is updated"""
    
    table_fqn = event['entity']['fqn']
    print(f"Change detected in table: {table_fqn}")
    
    # Capture current state
    result = subprocess.run([
        './bin/blast', 'snapshot',
        '--source', table_fqn.rsplit('.', 1)[0]
    ], capture_output=True)
    
    if result.returncode != 0:
        print(f"Failed to capture snapshot: {result.stderr}")
        return
    
    # Run drift check
    subprocess.run([
        './bin/blast', 'guard', 'drift',
        'snapshots/baseline.json',
        'snapshots/current.json',
        '--fail-on', 'critical'
    ])

# Listen to OpenMetadata events
consumer = KafkaConsumer(
    'metadata-events',
    bootstrap_servers=['kafka:9092'],
    value_deserializer=lambda m: json.loads(m.decode('utf-8'))
)

for event in consumer:
    if event['value']['eventType'] == 'ENTITY_UPDATED':
        on_table_change(event['value'])
EOF
}

################################################################################
# ===== PART 9: Python Script Integration =====
################################################################################

# Save this as scripts/monitor_metadata.py
python_integration_example() {
  cat << 'EOF'
#!/usr/bin/env python3

import subprocess
import json
from pathlib import Path
from datetime import datetime
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class BlastRadiusClient:
    def __init__(self, binary_path="./bin/blast"):
        self.binary = binary_path
    
    def run_command(self, *args):
        """Execute a blast command"""
        result = subprocess.run(
            [self.binary, *args, "--output", "json"],
            capture_output=True,
            text=True
        )
        return json.loads(result.stdout) if result.returncode == 0 else None
    
    def validate(self):
        """Check connectivity"""
        return self.run_command("validate")
    
    def snapshot(self, source_fqn):
        """Capture snapshot"""
        return self.run_command("snapshot", "--source", source_fqn)
    
    def compare(self, snap1, snap2):
        """Compare two snapshots"""
        return self.run_command("compare", snap1, snap2)
    
    def release_check(self, baseline, current, profile):
        """Run release gate"""
        result = subprocess.run([
            self.binary, "release-check",
            "--baseline", baseline,
            "--current", current,
            "--profile", profile,
            "--report-file", "report.json"
        ])
        return result.returncode == 0

# Usage
def main():
    client = BlastRadiusClient()
    
    # Validate setup
    if not client.validate():
        logger.error("OpenMetadata not accessible")
        return
    
    # Capture baseline
    logger.info("Capturing baseline...")
    client.snapshot("my_service.my_db.public")
    
    # Simulate change
    logger.info("Simulating metadata change...")
    # ... make changes to OpenMetadata ...
    
    # Capture current
    logger.info("Capturing current state...")
    client.snapshot("my_service.my_db.public")
    
    # Compare
    snap1 = list(Path("snapshots").rglob("*.json"))[-2]
    snap2 = list(Path("snapshots").rglob("*.json"))[-1]
    
    logger.info(f"Comparing {snap1} vs {snap2}")
    diff = client.compare(str(snap1), str(snap2))
    
    logger.info(f"Changes: {json.dumps(diff['diff']['summary'], indent=2)}")

if __name__ == "__main__":
    main()
EOF
}

################################################################################
# ===== DEMO ORCHESTRATION =====
################################################################################

# Main demo menu
demo_menu() {
  echo "======================================================="
  echo "BLAST RADIUS - INTEGRATION PATTERNS"
  echo "======================================================="
  echo ""
  echo "Choose an integration example:"
  echo ""
  echo "Shell Scripts:"
  echo "  1. Show CI drift detection"
  echo "  2. Show snapshot comparison workflow"
  echo "  3. Show release gate workflow"
  echo "  4. Show metadata health check"
  echo ""
  echo "CI/CD:"
  echo "  5. Show GitHub Actions example"
  echo "  6. Show GitLab CI example"
  echo "  7. Show Terraform integration"
  echo ""
  echo "Orchestration:"
  echo "  8. Show Kubernetes CronJob"
  echo "  9. Show Docker usage"
  echo "  10. Show Airflow DAG"
  echo ""
  echo "Event-Driven:"
  echo "  11. Show Kafka/SNS integration"
  echo "  12. Show Python client library"
  echo ""
  echo "  Q. Quit"
  echo ""
  
  read -p "Select (1-12, Q): " choice
  
  case $choice in
    1) github_actions_example ;;
    2) gitlab_ci_example ;;
    3) terraform_integration_example ;;
    4) kubernetes_integration_example ;;
    5) dockerfile_example ;;
    6) airflow_dag_example ;;
    7) event_driven_example ;;
    8) python_integration_example ;;
    Q|q) exit 0 ;;
    *) echo "Invalid choice" ;;
  esac
  
  echo ""
  echo "File created above. Copy to your project!"
}

# If running as script, show menu; if sourced, functions available
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  demo_menu
fi
