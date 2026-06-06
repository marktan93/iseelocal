# AGENTS.md

Guidance for AI and automation agents working in this repository.

## Project overview

`iseelocal` exposes local HTTP apps through a public VPS. It contains:

- `cmd/iseelocal-relay`: Go relay API and ingress server for the VPS.
- `cmd/iseelocal-agent`: Go local agent CLI for health checks and SSH reverse tunnels.
- `internal/relay`: API, auth, ingress proxy, port allocation, and SQLite store packages.
- `internal/agent`: local healthcheck, SSH command construction, and supervision packages.
- `internal/shared`: shared contracts and validation.
- `apps/desktop`: Tauri + React desktop UI.
- `infra`: Caddy, systemd, SSH, and install examples.
- `infra/docker`: Docker Compose deployment for Caddy, relay, and tunnel SSHD.
- `docs`: architecture, security, roadmap, and MVP runbook.

## Setup

Use the pinned package manager and existing toolchain configuration.

```bash
pnpm install --frozen-lockfile
go mod download
```

Do not commit generated dependency directories, local databases, build outputs, logs, or secrets.

## Required checks

Run the checks that match the files you changed. For broad or release-facing changes, run all of these:

```bash
go test ./...
pnpm --filter @iseelocal/desktop test
go build -o dist/iseelocal-relay ./cmd/iseelocal-relay
go build -o dist/iseelocal-agent ./cmd/iseelocal-agent
pnpm desktop:build
```

If `dist/` is created only for local validation, do not commit it.

## Task completion workflow

At the end of every completed task, commit and push the work to the current remote branch.

1. Run the relevant validation commands first and make sure they pass.
2. Check `git status --short --untracked-files=all`.
3. Do not commit secrets, local env files, local databases, dependency directories, build outputs, or session artifacts.
4. If there are source, documentation, or configuration changes, stage them, create a concise commit, and include this trailer:

   ```text
   Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
   ```

5. Push to the branch currently being worked on. If the branch has no upstream, push with:

   ```bash
   git push -u origin HEAD
   ```

6. If there are no repository changes after validation, do not create an empty commit; report that nothing needed committing.

## Local validation flow

Use local addresses and temporary state when validating without a VPS.

1. Start a local HTTP app on a loopback port, for example `127.0.0.1:3000`.
2. Start the relay with development-only values:

   ```bash
   ISEELOCAL_API_TOKEN=dev-token \
   ISEELOCAL_BASE_DOMAIN=localhost \
   ISEELOCAL_SSH_HOST=localhost \
   ISEELOCAL_DATABASE=/tmp/iseelocal-dev.db \
   go run ./cmd/iseelocal-relay
   ```

3. Verify the API:

   ```bash
   curl -sS -H "Authorization: Bearer dev-token" http://127.0.0.1:8081/api/health
   curl -sS -X POST http://127.0.0.1:8081/api/routes \
     -H "Authorization: Bearer dev-token" \
     -H "Content-Type: application/json" \
     -d '{"subdomain":"myapp","local_host":"127.0.0.1","local_port":3000,"protocol":"http"}'
   ```

4. Verify the agent against the local app:

   ```bash
   go run ./cmd/iseelocal-agent check --host 127.0.0.1 --port 3000
   go run ./cmd/iseelocal-agent ssh-args --ssh-host localhost --remote-port 18080 --local-port 3000
   ```

5. Mark a route online only after the tunnel is actually running. For local-only tests without SSH, verify offline and error responses explicitly instead of pretending the public tunnel works.

## VPS smoke test

For a real public test, follow `docs/mvp-runbook.md` and verify:

- DNS has root, wildcard, and API records pointing to the VPS.
- Caddy is the only public HTTP(S) listener.
- Relay API and ingress bind to loopback.
- The tunnel SSH user only allows remote port forwarding and does not allow shell, TTY, agent forwarding, or X11 forwarding.
- A public URL such as `https://myapp.yourdomain.com` reaches the local HTTP app only after the route exists, the SSH reverse tunnel is running, and heartbeat has marked the route online.

For Docker VPS deployments, use `docs/docker-vps.md`. The default tunnel SSH host port is `2222` to avoid conflicting with the VPS admin SSH service on `22`.

## Safety rules

- Never commit API tokens, SSH private keys, production domains with secrets, local databases, or generated certificates.
- Keep tunnel remote binds on `127.0.0.1`; Caddy should be the public entry point.
- Preserve loopback-only local target validation unless a task explicitly changes the security model.
- Do not broaden SSH permissions beyond the MVP requirements.
- Do not swallow errors with silent fallbacks; surface failures clearly in CLI, API, and UI paths.
- Prefer existing packages and validation helpers before adding new abstractions.

## Desktop notes

The current desktop/Tauri layer is still MVP/demo-oriented. Before claiming GUI-driven public tunneling works, verify that desktop commands create real relay routes and start real SSH tunnel supervision, not just local demo state.
