# Tao VPS: Go service deployment pattern

Durable knowledge about the web server (tao, root@tao.cl / 50.28.104.248) and
how single-binary Go projects are deployed behind its nginx reverse proxy.

## Access and environment

- SSH: `ssh tao` (root, `~/.ssh/config`), dev user via `ssh dev`.
- OS: Ubuntu; nginx at `/usr/sbin/nginx`; certbot via `/snap/bin/certbot`.
- Go is not installed on the VPS. Projects are cross-compiled externally:
  Termux arm64 -> linux/amd64 with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`
  (Termux `go build` fails on cross-compile with cgo enabled).

## Per-project layout (see kxp, asteroids, komikan)

- Binary: `/home/dev/<name>/<name>`, root-owned 0755 (copied by root via
  deploy script), also `/var/www/<name>` may hold an older copy for some
  projects. The directory itself is `dev:dev`.
- Deployment: source lives locally (module `github.com/taolicus/<name>`,
  binary in `cmd/server/`); `deploy.sh` cross-builds, copies the binary
  atomically (`.new` + rename), restarts the service, and health-checks the
  public URL. Run with `sh deploy.sh` from the phone (shared storage is
  noexec).
- systemd: `/etc/systemd/system/<name>.service`, `User=dev`,
  `WorkingDirectory=/home/dev/<name>`, `Restart=always`. Typical service
  environment: `ADDR=127.0.0.1:<port>` and/or `-addr` flag.
- nginx: `/etc/nginx/sites-available/<vhost>`, symlinked into
  `sites-enabled`, `proxy_pass http://127.0.0.1:<port>` with standard
  `X-Real-IP` / `X-Forwarded-*` headers; SSE endpoints add
  `proxy_buffering off`.
- TLS: `certbot --nginx -d <vhost>` (certbot account already registered;
  auto-renewal configured). Then `systemctl reload nginx`.

## Port allocation (localhost only)

- 18080: kxp (`kxp.tao.cl`, `/home/dev/kxp/kxp`, flag `-addr`)
- 18081: asteroids (`asteroids.pifia.net`, `/etc/asteroids.env` has LOBBY_KEY;
  service.d drop-in exists)
- 18082: komikan (`komikan.tequionet.org`, env `ADDR`)

Next free port: 18083. Ports 80xx are docker-proxy mappings.

## DNS

- `komikan.tequionet.org` is a CNAME to `tequionet.org`, which points at the
  VPS (50.28.104.248); the subdomain needs no DNS change when adding sites.
- The `pifia.net` certificate also covers the `tequionet.org` vhost.
