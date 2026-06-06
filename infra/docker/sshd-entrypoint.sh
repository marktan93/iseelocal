#!/usr/bin/env sh
set -eu

mkdir -p /etc/ssh/host_keys /home/tunnel/.ssh /run/sshd
passwd -d tunnel >/dev/null

if [ ! -f /etc/ssh/host_keys/ssh_host_ed25519_key ]; then
  ssh-keygen -t ed25519 -f /etc/ssh/host_keys/ssh_host_ed25519_key -N ''
fi

if [ ! -f /etc/ssh/host_keys/ssh_host_rsa_key ]; then
  ssh-keygen -t rsa -b 4096 -f /etc/ssh/host_keys/ssh_host_rsa_key -N ''
fi

if [ -f /authorized_keys/authorized_keys ]; then
  cp /authorized_keys/authorized_keys /home/tunnel/.ssh/authorized_keys
else
  echo "warning: /authorized_keys/authorized_keys is missing; tunnel SSH logins will fail" >&2
  : > /home/tunnel/.ssh/authorized_keys
fi

chown -R tunnel:tunnel /home/tunnel/.ssh
chmod 700 /home/tunnel/.ssh
chmod 600 /home/tunnel/.ssh/authorized_keys
chmod 600 /etc/ssh/host_keys/ssh_host_*_key

exec /usr/sbin/sshd -D -e -f /etc/ssh/sshd_config
