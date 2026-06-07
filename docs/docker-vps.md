# Docker VPS Deployment

This deployment runs Caddy, the relay, and a restricted tunnel SSH server with Docker Compose.

## Why the relay and SSHD share a network namespace

The relay stores route targets as `127.0.0.1:<remote_port>`, and the agent creates SSH reverse forwards bound to `127.0.0.1`. In Docker, separate containers have separate loopback interfaces. The Compose file therefore runs the relay with `network_mode: service:tunnel-sshd` so the relay and tunnel SSHD share the same loopback namespace.

Caddy is the only public HTTP(S) entry point. The relay listens on `0.0.0.0` only inside the private Docker network; the host publishes only Caddy ports and the tunnel SSH port.

## VPS prerequisites

- Docker Engine with the Compose plugin.
- DNS records pointing to the VPS:

  ```text
  A     yourdomain.com       <vps-ip>
  A     *.yourdomain.com     <vps-ip>
  A     api.yourdomain.com   <vps-ip>
  ```

- Public inbound ports:
  - `80/tcp` and `443/tcp` for Caddy.
  - `2222/tcp` by default for tunnel SSH.

Using `2222` avoids conflicting with the VPS admin SSH service on port `22`.

The included Docker Caddyfile serves the apex dashboard and API over HTTPS for real domains, and keeps wildcard app routes on HTTP by default. Wildcard HTTPS requires DNS-provider TLS automation or on-demand TLS safeguards.

## Setup

From the repository root on the VPS:

```bash
cp infra/docker/relay.env.example infra/docker/relay.env
cp infra/docker/ssh/authorized_keys.example infra/docker/ssh/authorized_keys
```

Edit `infra/docker/relay.env`:

```text
ISEELOCAL_API_TOKEN=<long-random-token>
ISEELOCAL_BASE_DOMAIN=iseelocal.dev
ISEELOCAL_PUBLIC_SCHEME=http
ISEELOCAL_SSH_HOST=152.42.204.9
ISEELOCAL_SSH_USER=tunnel
ISEELOCAL_SSH_PORT=2222
```

Replace the placeholder key in `infra/docker/ssh/authorized_keys` with the desktop public key.

Start the stack:

```bash
docker compose -f infra/docker/compose.yaml --env-file infra/docker/relay.env up -d --build
docker compose -f infra/docker/compose.yaml ps
```

## Create and start a route

Create a route through the public API:

```bash
curl -sS -X POST http://api.iseelocal.dev/api/routes \
  -H "Authorization: Bearer $ISEELOCAL_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"subdomain":"myapp","local_host":"127.0.0.1","local_port":3000,"protocol":"http"}'
```

The response includes `remote_port`, `ssh_user`, `ssh_host`, and `ssh_port`.

On the desktop, with the local app running:

```bash
./dist/iseelocal-agent check --host 127.0.0.1 --port 3000
./dist/iseelocal-agent run-ssh \
  --ssh-user tunnel \
  --ssh-host 152.42.204.9 \
  --ssh-port 2222 \
  --remote-port 18080 \
  --local-port 3000
```

In another shell, mark the route online:

```bash
curl -sS -X POST http://api.iseelocal.dev/api/routes/<route_id>/heartbeat \
  -H "Authorization: Bearer $ISEELOCAL_API_TOKEN"
```

Open:

```text
http://myapp.iseelocal.dev
```

## Herd sites

Laravel Herd sites usually share `127.0.0.1:80` and route by the HTTP `Host` header, for example `phpmyadmin.test` or `performance-track.test`. Expose a Herd site by creating a route with `upstream_host`:

```bash
curl -sS -X POST http://api.yourdomain.com/api/routes \
  -H "Authorization: Bearer $ISEELOCAL_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"subdomain":"phpmyadmin","project_name":"phpMyAdmin","project_path":"/Users/whoami/Desktop/scripts/phpmyadmin","local_host":"127.0.0.1","local_port":80,"upstream_host":"phpmyadmin.test","protocol":"http"}'
```

Then run the agent using the returned remote port and `--local-port 80`. Repeat with each Herd site name, for example `performance-track.test` or `lg-subscribe.test`.

To keep VPS routes in sync with local Herd projects, run the Herd sync watcher on the Mac:

```bash
python3 infra/scripts/sync-herd-routes.py \
  --ssh-key /path/to/vps-key.pem \
  --vps-host 152.42.204.9 \
  --watch-seconds 30
```

The watcher reads Herd's Valet config, creates missing `*.iseelocal.dev` routes, removes direct localhost-only routes, records each Herd folder and virtual host, and restarts one SSH reverse-forward process when the route set changes.

## Operations

View logs:

```bash
docker compose -f infra/docker/compose.yaml logs -f caddy relay tunnel-sshd
```

Restart:

```bash
docker compose -f infra/docker/compose.yaml restart
```

Stop:

```bash
docker compose -f infra/docker/compose.yaml down
```

Persistent data lives in Docker volumes: `relay-data`, `ssh-host-keys`, `caddy-data`, and `caddy-config`.
