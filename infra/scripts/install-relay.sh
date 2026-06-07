#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: install-relay.sh /path/to/iseelocal-relay" >&2
  exit 1
fi

install -d -m 755 /etc/iseelocal
id iseelocal >/dev/null 2>&1 || useradd --system --home /var/lib/iseelocal --shell /usr/sbin/nologin iseelocal
install -d -m 750 -o iseelocal -g iseelocal /var/lib/iseelocal
install -m 755 "$1" /usr/local/bin/iseelocal-relay
install -m 644 infra/systemd/iseelocal-relay.service /etc/systemd/system/iseelocal-relay.service

if [[ ! -f /etc/iseelocal/relay.env ]]; then
  cat >/etc/iseelocal/relay.env <<'ENV'
ISEELOCAL_API_TOKEN=replace-with-a-long-random-token
ISEELOCAL_BASE_DOMAIN=iseelocal.dev
ISEELOCAL_SSH_HOST=152.42.204.9
ISEELOCAL_SSH_USER=tunnel
ISEELOCAL_DATABASE=/var/lib/iseelocal/iseelocal.db
ISEELOCAL_API_ADDR=127.0.0.1:8081
ISEELOCAL_INGRESS_ADDR=127.0.0.1:8080
ISEELOCAL_REMOTE_PORT_START=18080
ISEELOCAL_REMOTE_PORT_END=18999
ENV
  chmod 600 /etc/iseelocal/relay.env
fi

systemctl daemon-reload
systemctl enable --now iseelocal-relay
systemctl status iseelocal-relay --no-pager
