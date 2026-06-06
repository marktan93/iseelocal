# syntax=docker/dockerfile:1.7

FROM alpine:3.22

RUN apk add --no-cache openssh-server

COPY infra/docker/sshd_config /etc/ssh/sshd_config
COPY infra/docker/sshd-entrypoint.sh /usr/local/bin/sshd-entrypoint

RUN adduser -D -h /home/tunnel -s /sbin/nologin tunnel \
    && mkdir -p /home/tunnel/.ssh /etc/ssh/host_keys /run/sshd \
    && chown -R tunnel:tunnel /home/tunnel/.ssh \
    && chmod 700 /home/tunnel/.ssh \
    && chmod +x /usr/local/bin/sshd-entrypoint

EXPOSE 22

ENTRYPOINT ["/usr/local/bin/sshd-entrypoint"]
