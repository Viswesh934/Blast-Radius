#!/usr/bin/env python3

"""
DEMO SCRIPT 2: Interactive Feature Showcase

This script provides an interactive menu to demonstrate individual features
of Blast Radius. Great for learning and hands-on exploration.

Usage:
  python3 examples/demo-interactive.py
  
  OR set environment first:
  export OM_BASE_URL="http://localhost:8585"
  export OM_JWT_TOKEN="your-token"
  python3 examples/demo-interactive.py
"""

import subprocess
import sys
import os
import json
from pathlib import Path
from typing import Optional, List, Tuple

class BlastDemo:
    def __init__(self, blast_bin: str = "./bin/blast"):
        self.blast_bin = blast_bin
        self.service = os.getenv("DEMO_SERVICE", "my_service")
        self.database = os.getenv("DEMO_DATABASE", "my_db")
        self.schema = os.getenv("DEMO_SCHEMA", "public")
        
        if not self.check_binary():
            print("❌ Blast binary not found. Building...")
            self.build()
    
    def check_binary(self) -> bool:
        return Path(self.blast_bin).exists()
    
    def build(self):
        """Build the blast binary"""
        result = subprocess.run(["go", "build", "-o", self.blast_bin, "./cmd"], cwd=".")
        if result.returncode != 0:
            print("Failed to build blast binary")
            sys.exit(1)
        print("✓ Build complete")
    
    def run_command(self, *args, **kwargs) -> Tuple[int, str, str]:
        """Run a blast command and return (exit_code, stdout, stderr)"""
        cmd = [self.blast_bin] + list(args)
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            **kwargs
        )
        return result.returncode, result.stdout, result.stderr
    
    def pretty_print(self, data):
        """Pretty print JSON or text"""
        try:
            obj = json.loads(data)
            print(json.dumps(obj, indent=2))
        except:
            print(data)
    
    # ========== Individual Demo Functions ==========
    
    def demo_validate(self):
        """Demo: Validate connectivity"""
        print("\n" + "="*60)
        print("DEMO 1: Validate OpenMetadata Connectivity")
        print("="*60)
        print("Command: ./bin/blast validate")
        print("")
        
        code, out, err = self.run_command("validate", "--output", "json")
        
        if code == 0:
            print("✓ Success")
            self.pretty_print(out)
        else:
            print("✗ Failed")
            print(err)
    
    def demo_services(self):
        """Demo: List services"""
        print("\n" + "="*60)
        print("DEMO 2: Discover Database Services")
        print("="*60)
        print("Command: ./bin/blast services")
        print("")
        
        code, out, err = self.run_command("services", "--output", "json")
        
        if code == 0:
            print("✓ Success")
            data = json.loads(out)
            print(f"\nFound {data.get('service_count', 0)} services:")
            for svc in data.get("services", [])[:3]:
                print(f"  - {svc.get('name')} ({svc.get('serviceType')})")
            if len(data.get("services", [])) > 3:
                print(f"  ... and {len(data['services']) - 3} more")
        else:
            print("✗ Failed")
            print(err)
    
    def demo_tables(self):
        """Demo: List tables"""
        print("\n" + "="*60)
        print("DEMO 3: Explore Table Catalog")
        print("="*60)
        fqn = f"{self.service}.{self.database}.{self.schema}"
        print(f"Command: ./bin/blast tables {fqn}")
        print("")
        
        code, out, err = self.run_command("tables", fqn, "--output", "json")
        
        if code == 0:
            print("✓ Success")
            data = json.loads(out)
            print(f"\nDatabase: {data.get('database_fqn')}")
            print(f"Tables found: {data.get('table_count', 0)}")
            for table in data.get("tables", [])[:3]:
                print(f"  - {table.get('name')}")
            if len(data.get("tables", [])) > 3:
                print(f"  ... and {len(data['tables']) - 3} more")
        else:
            print("✗ Not found or error")
            # This is expected if the database/schema doesn't exist
    
    def demo_snapshot(self):
        """Demo: Capture snapshot"""
        print("\n" + "="*60)
        print("DEMO 4: Capture Metadata Snapshot")
        print("="*60)
        fqn = f"{self.service}.{self.database}.{self.schema}"
        print(f"Command: ./bin/blast snapshot --source {fqn}")
        print("")
        
        code, out, err = self.run_command("snapshot", "--source", fqn)
        
        if code == 0:
            print("✓ Success")
            print(out)
            
            # Find the snapshot file
            snapshots = list(Path("snapshots").rglob("snapshot_*.json"))
            if snapshots:
                latest = max(snapshots, key=lambda p: p.stat().st_mtime)
                print(f"\nSnapshot saved to: {latest}")
                print(f"Size: {latest.stat().st_size} bytes")
        else:
            print("✗ Failed")
            print(err)
    
    def demo_compare(self):
        """Demo: Compare two snapshots"""
        print("\n" + "="*60)
        print("DEMO 5: Compare Two Snapshots")
        print("="*60)
        
        # Find two snapshots
        snapshots = sorted(list(Path("snapshots").rglob("snapshot_*.json")))
        
        if len(snapshots) < 2:
            print("⚠ Need at least 2 snapshots to compare")
            print("  Capture a snapshot first with Demo 4")
            return
        
        snap1 = str(snapshots[-2])
        snap2 = str(snapshots[-1])
        
        print(f"Command: ./bin/blast compare {snap1} {snap2}")
        print("")
        
        code, out, err = self.run_command("compare", snap1, snap2, "--output", "json")
        
        if code == 0:
            print("✓ Success")
            data = json.loads(out)
            summary = data.get("diff", {}).get("summary", {})
            print(f"\nChanges detected:")
            print(f"  Added: {summary.get('added', 0)}")
            print(f"  Deleted: {summary.get('deleted', 0)}")
            print(f"  Modified: {summary.get('modified', 0)}")
        else:
            print("✗ Failed")
            print(err)
    
    def demo_impact(self):
        """Demo: Analyze impact"""
        print("\n" + "="*60)
        print("DEMO 6: Analyze Downstream Impact")
        print("="*60)
        
        snapshots = sorted(list(Path("snapshots").rglob("snapshot_*.json")))
        
        if len(snapshots) < 2:
            print("⚠ Need at least 2 snapshots for impact analysis")
            return
        
        snap1 = str(snapshots[-2])
        snap2 = str(snapshots[-1])
        
        print(f"Command: ./bin/blast impact {snap1} {snap2}")
        print("")
        
        code, out, err = self.run_command("impact", snap1, snap2, "--output", "json")
        
        if code == 0:
            print("✓ Success")
            data = json.loads(out)
            impact = data.get("impact", {})
            print(f"\nImpact Analysis:")
            print(f"  Risk Level: {impact.get('risk_level', 'unknown').upper()}")
            print(f"  Impacted Assets: {len(impact.get('impacted_assets', []))}")
            print(f"  Recommended Actions: {len(impact.get('recommended_actions', []))}")
            
            for i, action in enumerate(impact.get("recommended_actions", [])[:2], 1):
                print(f"    {i}. {action}")
        else:
            print("✗ Failed")
            print(err)
    
    def demo_drift_check(self):
        """Demo: Drift detection"""
        print("\n" + "="*60)
        print("DEMO 7: Drift Detection with Policy")
        print("="*60)
        
        snapshots = sorted(list(Path("snapshots").rglob("snapshot_*.json")))
        
        if len(snapshots) < 2:
            print("⚠ Need at least 2 snapshots")
            return
        
        snap1 = str(snapshots[-2])
        snap2 = str(snapshots[-1])
        
        print(f"Command: ./bin/blast guard drift {snap1} {snap2} --fail-on critical")
        print("")
        
        code, out, err = self.run_command("guard", "drift", snap1, snap2, "--fail-on", "critical", "--output", "json")
        
        print(f"Exit Code: {code} {'(PASS ✓)' if code == 0 else '(FAIL ✗)'}")
        self.pretty_print(out)
    
    def demo_lineage(self):
        """Demo: Fetch lineage"""
        print("\n" + "="*60)
        print("DEMO 8: Metadata Lineage")
        print("="*60)
        
        table_fqn = f"{self.service}.{self.database}.{self.schema}.example_table"
        print(f"Command: ./bin/blast lineage {table_fqn}")
        print(f"(Using example table: {table_fqn})")
        print("")
        
        code, out, err = self.run_command("lineage", table_fqn, "--output", "json")
        
        if code == 0:
            print("✓ Success")
            data = json.loads(out)
            upstream = data.get("upstream", [])
            downstream = data.get("downstream", [])
            print(f"\nUpstream dependencies: {len(upstream)}")
            for u in upstream[:2]:
                print(f"  ← {u}")
            print(f"\nDownstream consumers: {len(downstream)}")
            for d in downstream[:2]:
                print(f"  → {d}")
        else:
            print("✗ Not found (table may not have lineage)")
    
    def demo_release_check(self):
        """Demo: Release safety gate"""
        print("\n" + "="*60)
        print("DEMO 9: Release Safety Gate")
        print("="*60)
        
        snapshots = sorted(list(Path("snapshots").rglob("snapshot_*.json")))
        
        if len(snapshots) < 2:
            print("⚠ Need at least 2 snapshots")
            return
        
        snap1 = str(snapshots[-2])
        snap2 = str(snapshots[-1])
        
        print(f"Command: ./bin/blast release-check \\")
        print(f"  --baseline {snap1} \\")
        print(f"  --current {snap2} \\")
        print(f"  --max-risk medium \\")
        print(f"  --report-file release_report.json")
        print("")
        
        code, out, err = self.run_command(
            "release-check",
            "--baseline", snap1,
            "--current", snap2,
            "--max-risk", "medium",
            "--report-file", "release_report.json",
            "--markdown-file", "release_report.md"
        )
        
        print(f"Exit Code: {code}")
        print(f"Status: {'PASS ✓ Release approved' if code == 0 else 'FAIL ✗ Release blocked'}")
        print(out)
        print(err) if err else None
    
    def show_menu(self):
        """Display interactive menu"""
        while True:
            print("\n" + "="*60)
            print("BLAST RADIUS - INTERACTIVE DEMO")
            print("="*60)
            print("\nChoose a feature to demo:")
            print("  1. Validate OpenMetadata connectivity")
            print("  2. Discover database services")
            print("  3. Explore table catalog")
            print("  4. Capture metadata snapshot")
            print("  5. Compare two snapshots")
            print("  6. Analyze downstream impact")
            print("  7. Drift detection with policy")
            print("  8. Metadata lineage analysis")
            print("  9. Release safety gate")
            print("  10. Run all demos")
            print("  Q. Quit")
            print("")
            
            choice = input("Select (1-10, Q): ").strip().upper()
            
            if choice == 'Q':
                print("Goodbye! 👋")
                break
            elif choice == '1':
                self.demo_validate()
            elif choice == '2':
                self.demo_services()
            elif choice == '3':
                self.demo_tables()
            elif choice == '4':
                self.demo_snapshot()
            elif choice == '5':
                self.demo_compare()
            elif choice == '6':
                self.demo_impact()
            elif choice == '7':
                self.demo_drift_check()
            elif choice == '8':
                self.demo_lineage()
            elif choice == '9':
                self.demo_release_check()
            elif choice == '10':
                self.run_all()
            else:
                print("Invalid choice")
            
            input("\nPress Enter to continue...")
    
    def run_all(self):
        """Run all demos in sequence"""
        print("\n🚀 Running all demos...\n")
        self.demo_validate()
        self.demo_services()
        self.demo_tables()
        self.demo_snapshot()
        self.demo_compare()
        self.demo_impact()
        self.demo_drift_check()
        self.demo_lineage()
        self.demo_release_check()
        print("\n✓ All demos complete!")

def main():
    """Main entry point"""
    try:
        demo = BlastDemo()
        demo.show_menu()
    except KeyboardInterrupt:
        print("\n\nInterrupted")
        sys.exit(0)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
