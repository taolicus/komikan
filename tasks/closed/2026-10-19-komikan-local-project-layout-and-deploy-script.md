# Komikan: local project layout and deploy script

- Date: 2026-10-19
- Area: deployment

## Outcome

Completed. The komikan Go project now lives entirely in this repository,
matching the asteroids project layout, and is deployed with a script.

## What was done

- Moved the project from `komikan/` to the repository root:
  `go.mod` (module `github.com/taolicus/komikan`), `cmd/server/main.go`,
  `cmd/server/index.html`.
- Added `deploy.sh` (POSIX sh, modeled on `../asteroids/deploy.sh` but
  simpler): builds `./cmd/server` for linux/amd64, copies the binary to
  `/home/dev/komikan/` via atomic rename, restarts `komikan.service`, and
  health-checks `https://komikan.tequionet.org/`.
- Deployed the site with the script; the manually copied binary from the
  previous deployment was replaced by the script-managed one. The VPS holds
  no source code (never did) — only the binary, plus the systemd and nginx
  configuration that stays as host setup.

## Verified

- `sh deploy.sh` run end to end from the phone (storage is noexec, so
  `./deploy.sh` fails; use `sh deploy.sh`).
- Service restarted with the new binary; HTTPS returns 200 from outside.
