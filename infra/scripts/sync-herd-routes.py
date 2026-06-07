#!/usr/bin/env python3
"""Discover local Laravel Herd projects and sync them to an iseelocal VPS."""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import signal
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any


DEFAULT_EXCLUDES = {"iseelocal", "iseelocal.worktrees", "node_modules", "vendor"}
PROJECT_MARKERS = ("public", "artisan", "composer.json", "package.json")


def main() -> int:
    args = parse_args()
    if args.watch_seconds <= 0:
        result = sync_once(args, previous_signature="")
        print_summary(result)
        return 0

    print(f"watching Herd projects every {args.watch_seconds}s", flush=True)
    previous_signature = ""
    stop = False

    def handle_stop(_signum: int, _frame: object) -> None:
        nonlocal stop
        stop = True

    signal.signal(signal.SIGINT, handle_stop)
    signal.signal(signal.SIGTERM, handle_stop)

    while not stop:
        try:
            result = sync_once(args, previous_signature=previous_signature)
            previous_signature = result["signature"]
            print_summary(result)
        except Exception as exc:  # noqa: BLE001 - top-level watcher must keep reporting failures.
            print(f"sync failed: {exc}", file=sys.stderr, flush=True)
        for _ in range(args.watch_seconds):
            if stop:
                break
            time.sleep(1)

    return 0


def parse_args() -> argparse.Namespace:
    home = Path.home()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--vps-host", default=os.getenv("ISEELOCAL_VPS_HOST", "152.42.204.9"))
    parser.add_argument("--vps-user", default=os.getenv("ISEELOCAL_VPS_USER", "root"))
    parser.add_argument("--ssh-key", default=os.getenv("ISEELOCAL_SSH_KEY", ""))
    parser.add_argument("--tunnel-user", default=os.getenv("ISEELOCAL_TUNNEL_USER", "tunnel"))
    parser.add_argument("--tunnel-ssh-port", type=int, default=int(os.getenv("ISEELOCAL_TUNNEL_SSH_PORT", "2222")))
    parser.add_argument("--base-domain", default=os.getenv("ISEELOCAL_BASE_DOMAIN", "iseelocal.dev"))
    parser.add_argument("--public-scheme", choices=("http", "https"), default=os.getenv("ISEELOCAL_PUBLIC_SCHEME", "https"))
    parser.add_argument("--remote-port-start", type=int, default=int(os.getenv("ISEELOCAL_REMOTE_PORT_START", "18080")))
    parser.add_argument("--remote-port-end", type=int, default=int(os.getenv("ISEELOCAL_REMOTE_PORT_END", "18999")))
    parser.add_argument(
        "--herd-config",
        default=str(home / "Library/Application Support/Herd/config/valet/config.json"),
    )
    parser.add_argument("--watch-seconds", type=int, default=0, help="rerun forever at this interval")
    parser.add_argument("--exclude", action="append", default=[], help="folder name to ignore; can be repeated")
    parser.add_argument("--keep-direct", action="store_true", help="do not remove direct localhost routes")
    parsed = parser.parse_args()
    if not parsed.ssh_key:
        parser.error("--ssh-key or ISEELOCAL_SSH_KEY is required")
    return parsed


def sync_once(args: argparse.Namespace, previous_signature: str) -> dict[str, Any]:
    projects = discover_projects(Path(args.herd_config), set(args.exclude) | DEFAULT_EXCLUDES)
    result = sync_remote_routes(args, projects)
    signature = route_signature(result["routes"])
    should_restart = signature != previous_signature or not tunnel_process_running(args)
    if should_restart:
        restart_tunnel(args, result["routes"])
    result["signature"] = signature
    result["tunnel_restarted"] = should_restart
    return result


def discover_projects(config_path: Path, excludes: set[str]) -> list[dict[str, str]]:
    with config_path.open("r", encoding="utf-8") as file:
        config = json.load(file)

    tld = str(config.get("tld") or "test").strip(".") or "test"
    discovered: dict[str, dict[str, str]] = {}
    for raw_root in config.get("paths") or []:
        root = Path(str(raw_root)).expanduser()
        if not root.is_dir():
            continue
        try:
            children = sorted(root.iterdir(), key=lambda item: item.name.lower())
        except OSError as err:
            print(f"skipping unreadable Herd path {root}: {err}", file=sys.stderr)
            continue
        for child in children:
            if child.name.startswith(".") or child.name in excludes or not is_dir(child):
                continue
            if not has_project_marker(child):
                continue
            name = normalize_subdomain(child.name)
            if not name:
                print(f"skipping invalid Herd site name: {child.name}", file=sys.stderr)
                continue
            discovered.setdefault(
                name,
                {
                    "subdomain": name,
                    "project_name": name,
                    "project_path": str(child.resolve()),
                    "upstream_host": f"{name}.{tld}",
                },
            )
    return list(discovered.values())


def is_dir(path: Path) -> bool:
    try:
        return path.is_dir()
    except OSError:
        return False


def has_project_marker(path: Path) -> bool:
    return any((path / marker).exists() for marker in PROJECT_MARKERS)


def normalize_subdomain(value: str) -> str:
    normalized = re.sub(r"[^a-z0-9-]+", "-", value.strip().lower()).strip("-")
    if len(normalized) < 2 or len(normalized) > 63:
        return ""
    if not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]*[a-z0-9])?", normalized):
        return ""
    return normalized


def sync_remote_routes(args: argparse.Namespace, projects: list[dict[str, str]]) -> dict[str, Any]:
    payload = {
        "projects": projects,
        "base_domain": args.base_domain,
        "public_scheme": args.public_scheme,
        "remote_port_start": args.remote_port_start,
        "remote_port_end": args.remote_port_end,
        "prune_direct": not args.keep_direct,
    }
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", delete=False) as file:
        json.dump(payload, file)
        payload_path = file.name
    remote_payload = "/tmp/iseelocal-herd-sync.json"
    try:
        run(["scp", *ssh_key_args(args), payload_path, f"{args.vps_user}@{args.vps_host}:{remote_payload}"])
        completed = run(
            [
                "ssh",
                *ssh_key_args(args),
                f"{args.vps_user}@{args.vps_host}",
                "python3 - <<'PY'\n" + REMOTE_SYNC_SCRIPT + "\nPY",
            ],
            capture=True,
        )
        return json.loads(completed.stdout)
    finally:
        Path(payload_path).unlink(missing_ok=True)


REMOTE_SYNC_SCRIPT = r"""
import datetime
import json
import secrets
import sqlite3

payload = json.load(open("/tmp/iseelocal-herd-sync.json", "r", encoding="utf-8"))
path = "/var/lib/docker/volumes/iseelocal_relay-data/_data/iseelocal.db"
now = datetime.datetime.now(datetime.UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")
base_domain = payload["base_domain"].strip(".").lower()
public_scheme = payload["public_scheme"]
port_start = int(payload["remote_port_start"])
port_end = int(payload["remote_port_end"])
projects = payload["projects"]

con = sqlite3.connect(path)
with con:
    if payload.get("prune_direct"):
        con.execute("delete from routes where coalesce(upstream_host, '') = ''")

    used = {row[0] for row in con.execute("select remote_port from routes")}
    next_port = port_start

    for project in sorted(projects, key=lambda item: item["subdomain"]):
        subdomain = project["subdomain"]
        upstream_host = project["upstream_host"]
        public_host = f"{subdomain}.{base_domain}"
        public_url = f"{public_scheme}://{public_host}"
        row = con.execute("select id from routes where subdomain = ?", (subdomain,)).fetchone()
        if row:
            con.execute(
                '''
                update routes
                set public_host = ?, public_url = ?, project_name = ?, project_path = ?,
                    local_host = ?, local_port = ?, upstream_host = ?, status = ?,
                    updated_at = ?, last_heartbeat_at = ?
                where subdomain = ?
                ''',
                (
                    public_host,
                    public_url,
                    project["project_name"],
                    project["project_path"],
                    "127.0.0.1",
                    80,
                    upstream_host,
                    "online",
                    now,
                    now,
                    subdomain,
                ),
            )
            continue

        while next_port in used and next_port <= port_end:
            next_port += 1
        if next_port > port_end:
            raise RuntimeError("no remote ports available")

        used.add(next_port)
        con.execute(
            '''
            insert into routes (
                id, subdomain, public_host, public_url, project_name, project_path,
                local_host, local_port, upstream_host, remote_host, remote_port,
                protocol, status, created_at, updated_at, last_heartbeat_at
            ) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ''',
            (
                "route_" + secrets.token_hex(8),
                subdomain,
                public_host,
                public_url,
                project["project_name"],
                project["project_path"],
                "127.0.0.1",
                80,
                upstream_host,
                "127.0.0.1",
                next_port,
                "http",
                "online",
                now,
                now,
                now,
            ),
        )

routes = [
    {
        "subdomain": row[0],
        "local_host": row[1],
        "local_port": row[2],
        "remote_port": row[3],
        "upstream_host": row[4],
    }
    for row in con.execute(
        '''
        select subdomain, local_host, local_port, remote_port, upstream_host
        from routes
        where coalesce(upstream_host, '') != ''
        order by remote_port
        '''
    )
]
print(json.dumps({"projects": len(projects), "routes": routes}, sort_keys=True))
"""


def route_signature(routes: list[dict[str, Any]]) -> str:
    compact = [
        [route["subdomain"], route["remote_port"], route["local_host"], route["local_port"]]
        for route in routes
    ]
    return json.dumps(compact, sort_keys=True)


def tunnel_process_running(args: argparse.Namespace) -> bool:
    completed = run(["ps", "-axo", "command="], capture=True)
    destination = f"{args.tunnel_user}@{args.vps_host}"
    return any(destination in line and " -R " in f" {line} " for line in completed.stdout.splitlines())


def restart_tunnel(args: argparse.Namespace, routes: list[dict[str, Any]]) -> None:
    stop_existing_tunnels(args)
    if not routes:
        return

    command = [
        "ssh",
        *ssh_key_args(args),
        "-p",
        str(args.tunnel_ssh_port),
        "-o",
        "ExitOnForwardFailure=yes",
        "-o",
        "ServerAliveInterval=30",
        "-o",
        "ServerAliveCountMax=3",
        "-o",
        "StrictHostKeyChecking=accept-new",
        "-f",
        "-N",
    ]
    for route in routes:
        command.extend(
            [
                "-R",
                f"127.0.0.1:{route['remote_port']}:{route['local_host']}:{route['local_port']}",
            ]
        )
    command.append(f"{args.tunnel_user}@{args.vps_host}")
    run(command)


def stop_existing_tunnels(args: argparse.Namespace) -> None:
    completed = run(["ps", "-axo", "pid=,command="], capture=True)
    destination = f"{args.tunnel_user}@{args.vps_host}"
    for line in completed.stdout.splitlines():
        parts = line.strip().split(maxsplit=1)
        if len(parts) != 2:
            continue
        pid, command = parts
        if destination in command and " -R " in f" {command} ":
            try:
                os.kill(int(pid), signal.SIGTERM)
            except ProcessLookupError:
                pass


def ssh_key_args(args: argparse.Namespace) -> list[str]:
    return ["-i", args.ssh_key, "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"]


def run(command: list[str], capture: bool = False) -> subprocess.CompletedProcess[str]:
    printable = " ".join(shlex.quote(part) for part in command)
    completed = subprocess.run(
        command,
        check=False,
        text=True,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
    )
    if completed.returncode != 0:
        detail = completed.stderr.strip() if completed.stderr else ""
        raise RuntimeError(f"command failed ({completed.returncode}): {printable}\n{detail}")
    return completed


def print_summary(result: dict[str, Any]) -> None:
    print(
        f"synced {result['projects']} Herd projects, {len(result['routes'])} VPS routes; "
        f"tunnel_restarted={result['tunnel_restarted']}",
        flush=True,
    )


if __name__ == "__main__":
    raise SystemExit(main())
