#!/bin/sh
# Build the server binary and deploy it to the production VPS.
#
# Deployment is deliberately just "copy the binary": systemd
# (komikan.service) and nginx (komikan.tequionet.org) are already set
# up on the host.
#
# POSIX sh only: works with sh, dash, bash, or ash.
set -eu

HOST="${DEPLOY_HOST:-tao}"
REMOTE_DIR="${DEPLOY_DIR:-/home/dev/komikan}"
SERVICE="komikan"
HEALTH_URL="${HEALTH_URL:-https://komikan.tequionet.org/healthz}"

cd "$(dirname "$0")"

bin="$(mktemp)"
trap 'rm -f "$bin"' EXIT

rev="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"

echo "==> building for linux/amd64 ($rev)"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w -X main.version=$rev" -o "$bin" ./cmd/server

echo "==> copying to $HOST:$REMOTE_DIR"
scp -q "$bin" "$HOST:$REMOTE_DIR/komikan.new"
# Replace by rename so a running binary is never written in place.
# The data directory holds the SQLite file and must belong to the service user.
ssh "$HOST" "chmod 0755 '$REMOTE_DIR/komikan.new' && mv -f '$REMOTE_DIR/komikan.new' '$REMOTE_DIR/komikan' && install -d -o dev -g dev -m 0750 '$REMOTE_DIR/data' && systemctl restart $SERVICE"

echo "==> checking $HEALTH_URL"
curl -fsS --retry 10 --retry-delay 1 --retry-all-errors -o /dev/null "$HEALTH_URL"
echo "==> deployed"
