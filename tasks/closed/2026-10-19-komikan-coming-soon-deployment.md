# Komikan coming-soon page deployment

- Date: 2026-10-19
- Area: deployment

## Outcome

Completed. https://komikan.tequionet.org serves the coming-soon page
("Pr&oacute;ximamente...", black background, centered white text).

## What was done

- Created `komikan/` in this repository: a minimal Go HTTP server with the
  HTML page embedded via `go:embed` (see `features/tao-vps-go-service-deployment.md`
  for the deployment pattern used).
- Cross-compiled from Termux (arm64) to linux/amd64 with
  `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` and deployed the binary to
  `/home/dev/komikan/komikan` on tao.
- Added and enabled `komikan.service` (systemd, user `dev`, listens on
  `127.0.0.1:18082`).
- Added the nginx site `komikan.tequionet.org` (reverse proxy to 18082) and
  obtained a Let's Encrypt certificate via `certbot --nginx` with HTTPS
  redirect and automatic renewal.

## Verified

- Page serves over HTTPS from outside the VPS with a valid certificate.
- HTTP redirects to HTTPS.
- Service enabled (survives reboot) and running.
