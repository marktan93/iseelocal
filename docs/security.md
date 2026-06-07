# Security

The MVP defaults to a private single-user deployment.

## Defaults

- Relay API requires `Authorization: Bearer <token>`.
- SSH tunnel user is restricted and has no shell.
- Remote forwarded ports bind to `127.0.0.1` on the VPS.
- Caddy is the only public listener.
- Local target validation allows loopback HTTP targets only.
- Sensitive local ports are blocked by default: `22`, `3306`, `5432`, `6379`, `27017`.
- Relay ingress limits request bodies to 10 MiB.
- Relay ingress writes access logs with host, method, path, status, and duration.

## VPS SSH Notes

Keep `GatewayPorts no` for the tunnel user. Public exposure should happen through Caddy and the Go ingress proxy, not direct `0.0.0.0` remote binds.

Use a dedicated SSH key for `tunnel@152.42.204.9`. Do not reuse a personal admin key.

## Before Public SaaS

Add per-user auth, per-route random secrets, optional public basic auth, quotas, rate limits backed by shared storage, request audit retention, custom domains, and abuse handling.
