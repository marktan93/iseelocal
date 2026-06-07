# Architecture

`iseelocal` exposes local HTTP apps through a public VPS without requiring inbound access to the developer machine.

## Components

- Desktop app: Tauri + React control surface for mappings, tunnel state, and logs.
- Local agent: Go CLI/package set for local health checks and OpenSSH process supervision.
- Relay server: Go API and ingress proxy running on the VPS.
- Caddy: public TLS terminator and wildcard reverse proxy.
- OpenSSH: secure outbound reverse tunnel transport for the MVP.

## Request Flow

```text
User browser
  -> https://myapp.iseelocal.dev
  -> Caddy wildcard vhost
  -> relay ingress on 127.0.0.1:8080
  -> route lookup by Host header
  -> http://127.0.0.1:18080 on VPS
  -> ssh -R channel
  -> http://127.0.0.1:3000 on desktop
```

## Control Flow

The desktop creates a route through the relay API. The relay allocates a loopback-only remote port and persists the route in SQLite. The agent starts an SSH process with `-R 127.0.0.1:<remote_port>:127.0.0.1:<local_port>`, then sends heartbeats once the tunnel is alive.

The first implementation keeps the Tauri Rust layer thin. Production relay and tunnel behavior lives in Go packages so the desktop, CLI, and future protocol implementation can reuse the same logic.
