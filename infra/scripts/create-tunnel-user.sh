#!/usr/bin/env bash
set -euo pipefail

useradd --system --create-home --shell /usr/sbin/nologin tunnel || true
install -d -m 700 -o tunnel -g tunnel /home/tunnel/.ssh
touch /home/tunnel/.ssh/authorized_keys
chown tunnel:tunnel /home/tunnel/.ssh/authorized_keys
chmod 600 /home/tunnel/.ssh/authorized_keys

echo "Add the desktop public key to /home/tunnel/.ssh/authorized_keys using infra/ssh/authorized_keys.example."
echo "Append infra/ssh/sshd_config.snippet to /etc/ssh/sshd_config, then run: systemctl reload ssh"
