# MVP Runbook

## 1. DNS

Create DNS records pointing to the VPS:

```text
A     yourdomain.com       <vps-ip>
A     *.yourdomain.com     <vps-ip>
A     api.yourdomain.com   <vps-ip>
```

## 2. Build Relay And Agent

From the project root:

```bash
go build -o dist/iseelocal-relay ./cmd/iseelocal-relay
go build -o dist/iseelocal-agent ./cmd/iseelocal-agent
```

## 3. VPS Setup

Install Caddy and OpenSSH server on Ubuntu. Copy `dist/iseelocal-relay` to the VPS, then run:

```bash
sudo useradd --system --home /var/lib/iseelocal --shell /usr/sbin/nologin iseelocal || true
sudo ./infra/scripts/create-tunnel-user.sh
sudo ./infra/scripts/install-relay.sh ./dist/iseelocal-relay
```

Copy `infra/caddy/Caddyfile.example` into `/etc/caddy/Caddyfile`, replace `yourdomain.com`, then reload Caddy:

```bash
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

For a Docker-based VPS setup with Caddy, relay, and a restricted tunnel SSHD in Compose, see `docs/docker-vps.md`.

## 4. Create A Route

```bash
curl -sS -X POST https://api.yourdomain.com/api/routes \
  -H "Authorization: Bearer $ISEELOCAL_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"subdomain":"myapp","local_host":"127.0.0.1","local_port":3000,"protocol":"http"}'
```

The response contains `remote_port`, `ssh_user`, and `ssh_host`.

## 5. Start The Tunnel

On the desktop, with a local app running on port `3000`:

```bash
./dist/iseelocal-agent check --host 127.0.0.1 --port 3000
./dist/iseelocal-agent run-ssh \
  --ssh-user tunnel \
  --ssh-host your-vps.com \
  --ssh-port 22 \
  --remote-port 18080 \
  --local-port 3000
```

Then mark the route online:

```bash
curl -sS -X POST https://api.yourdomain.com/api/routes/route_id/heartbeat \
  -H "Authorization: Bearer $ISEELOCAL_API_TOKEN"
```

Open:

```text
https://myapp.yourdomain.com
```
