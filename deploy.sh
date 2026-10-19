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
HEALTH_URL="${HEALTH_URL:-https://komikan.tequionet.org/}"

cd "$(dirname "$0")"

bin="$(mktemp)"
trap 'rm -f "$bin"' EXIT

echo "==> building for linux/amd64"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o "$bin" ./cmd/server

echo "==> copying to $HOST:$REMOTE_DIR"
scp -q "$bin" "$HOST:$REMOTE_DIR/komikan.new"
# Replace by rename so a running binary is never written in place.
ssh "$HOST" "chmod 0755 '$REMOTE_DIR/komikan.new' && mv -f '$REMOTE_DIR/komikan.new' '$REMOTE_DIR/komikan' && systemctl restart $SERVICE"

echo "==> checking $HEALTH_URL"
curl -fsS --retry 10 --retry-delay 1 --retry-all-errors -o /dev/null "$HEALTH_URL"
echo "==> deployed"
