# Roadmap

## MVP

- Tauri React desktop control surface.
- Go local agent with health checks and SSH process supervision.
- Go relay API with SQLite route storage and token auth.
- Go ingress proxy that routes wildcard hosts to loopback remote ports.
- Caddy wildcard HTTPS in front of relay ingress.

## Next

- Desktop settings persistence wired into the Tauri app.
- Relay API integration from Tauri commands instead of the current local shell stub.
- SSH control master support for adding and removing mappings without restarting every tunnel.
- Password-protected public routes.
- Heartbeat automation from the agent.
- Installable macOS app bundle and signed binaries.

## Later

- Custom Go tunnel protocol over TLS/WebSocket or gRPC bidirectional streams.
- Raw TCP tunnel mode.
- Multi-user accounts, teams, quotas, billing, and custom domains.
