# iseelocal

`iseelocal` is a private, GUI-controlled reverse tunnel MVP. It uses a Tauri + React desktop app, a Go local agent, a Go relay server, OpenSSH reverse forwarding, and Caddy on an Ubuntu VPS.

## MVP Architecture

```text
Browser
  -> https://myapp.iseelocal.dev
  -> Caddy wildcard HTTPS
  -> iseelocal relay ingress
  -> 127.0.0.1:18080 on the VPS
  -> ssh -R tunnel
  -> local agent
  -> 127.0.0.1:3000
```

The first implementation intentionally uses OpenSSH reverse tunnels instead of a custom tunnel protocol. Tunnel ports stay bound to the VPS loopback interface, and Caddy is the only public listener.

## Workspace

```text
apps/desktop              Tauri + React desktop app
cmd/iseelocal-agent   Go local agent CLI
cmd/iseelocal-relay   Go relay API and ingress server
internal/agent            Agent packages
internal/relay            Relay packages
internal/shared           Shared contracts and validation
infra                     Caddy, SSH, systemd, and install examples
docs                      Architecture, security, and runbooks
```

## Quick Start

Install dependencies:

```bash
pnpm install
go mod download
```

Run tests:

```bash
go test ./...
pnpm --filter @iseelocal/desktop test
```

Run the relay locally:

```bash
ISEELOCAL_API_TOKEN=dev-token \
ISEELOCAL_BASE_DOMAIN=localhost \
ISEELOCAL_SSH_HOST=152.42.204.9 \
go run ./cmd/iseelocal-relay
```

Run the desktop UI:

```bash
pnpm desktop:dev
```

See `docs/mvp-runbook.md` for the full VPS flow.
