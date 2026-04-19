#!/usr/bin/env python3
import argparse
import json
import subprocess
import sys
from pathlib import Path


def send(proc, request):
    body = json.dumps(request).encode("utf-8")
    header = f"Content-Length: {len(body)}\r\n\r\n".encode("utf-8")
    proc.stdin.write(header + body)
    proc.stdin.flush()


def recv(proc):
    content_length = None
    while True:
        line = b""
        while not line.endswith(b"\n"):
            chunk = proc.stdout.read(1)
            if not chunk:
                raise RuntimeError("EOF while reading headers")
            line += chunk
        line_s = line.decode("utf-8").strip()
        if line_s == "":
            break
        if line_s.lower().startswith("content-length:"):
            content_length = int(line_s.split(":", 1)[1].strip())

    if content_length is None:
        raise RuntimeError("missing Content-Length in MCP response")

    payload = proc.stdout.read(content_length)
    if not payload:
        raise RuntimeError("empty MCP payload")
    return json.loads(payload.decode("utf-8"))


def call(proc, request):
    send(proc, request)
    return recv(proc)


def print_step(title, response):
    print(f"\n=== {title} ===")
    print(json.dumps(response, indent=2))


def resolve_input_path(raw_path, repo_root):
    path = Path(raw_path)
    if path.is_absolute():
        return path
    if path.exists():
        return path.resolve()
    candidate = repo_root / path
    if candidate.exists():
        return candidate.resolve()
    return candidate.resolve()


def resolve_snapshot_file(path):
    if path.is_file():
        return path
    if path.is_dir():
        matches = sorted(path.glob("snapshot_*.json"))
        if matches:
            return matches[-1]
    return path


def main():
    script_dir = Path(__file__).resolve().parent
    repo_root = script_dir.parent

    default_bin = repo_root / "bin" / "blast"
    default_older = repo_root / "snapshots" / "sources" / "my_service" / "analytics" / "public" / "snapshot_1776536091.json"
    default_newer = repo_root / "snapshots" / "sources" / "my_service" / "analytics" / "public" / "snapshot_1776536138.json"

    parser = argparse.ArgumentParser(description="Blast Radius MCP demo")
    parser.add_argument("--bin", default=str(default_bin), help="Path to blast binary")
    parser.add_argument(
        "--older",
        default=str(default_older),
        help="Older snapshot path",
    )
    parser.add_argument(
        "--newer",
        default=str(default_newer),
        help="Newer snapshot path",
    )
    args = parser.parse_args()

    bin_path = resolve_input_path(args.bin, repo_root)
    older = resolve_snapshot_file(resolve_input_path(args.older, repo_root))
    newer = resolve_snapshot_file(resolve_input_path(args.newer, repo_root))

    if not bin_path.exists():
        print(f"Blast binary not found: {bin_path}", file=sys.stderr)
        print("Build first: go build -o bin/blast ./cmd", file=sys.stderr)
        return 1
    if not older.exists() or not newer.exists() or not older.is_file() or not newer.is_file():
        print("Snapshot files do not exist.", file=sys.stderr)
        print(f"older: {older}", file=sys.stderr)
        print(f"newer: {newer}", file=sys.stderr)
        print("Pass --older <path> and --newer <path> to snapshot JSON files or directories containing snapshot_*.json.", file=sys.stderr)
        return 1

    proc = subprocess.Popen(
        [str(bin_path), "mcp"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )

    try:
        init_resp = call(
            proc,
            {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2025-03-26",
                    "capabilities": {},
                    "clientInfo": {"name": "blast-demo", "version": "0.1.0"},
                },
            },
        )
        print_step("1) initialize", init_resp)

        tools_resp = call(proc, {"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
        print_step("2) tools/list", tools_resp)

        ingest_resp = call(
            proc,
            {
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/call",
                "params": {
                    "name": "blast.ingest.event",
                    "arguments": {
                        "type": "table.updated",
                        "source_fqn": "my_service.analytics.public",
                        "entity_fqn": "my_service.analytics.public.events",
                        "payload": {"change": "description-updated", "owner": "demo"},
                    },
                },
            },
        )
        print_step("3) blast.ingest.event", ingest_resp)

        compare_resp = call(
            proc,
            {
                "jsonrpc": "2.0",
                "id": 4,
                "method": "tools/call",
                "params": {
                    "name": "blast.snapshot.compare",
                    "arguments": {
                        "older_snapshot": str(older),
                        "newer_snapshot": str(newer),
                    },
                },
            },
        )
        print_step("4) blast.snapshot.compare", compare_resp)

        impact_resp = call(
            proc,
            {
                "jsonrpc": "2.0",
                "id": 5,
                "method": "tools/call",
                "params": {
                    "name": "blast.impact.analyze",
                    "arguments": {
                        "older_snapshot": str(older),
                        "newer_snapshot": str(newer),
                    },
                },
            },
        )
        print_step("5) blast.impact.analyze", impact_resp)

        print("\nDemo complete.")
        print("Event log: snapshots/ingestion/events.ndjson")
        return 0
    finally:
        proc.terminate()


if __name__ == "__main__":
    raise SystemExit(main())
