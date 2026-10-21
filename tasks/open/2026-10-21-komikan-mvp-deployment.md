# Deploy the playable MVP to komikan.tequionet.org

- Date: 2026-10-21
- Area: deployment
- Dependencies: tasks/open/2026-10-21-komikan-integration-and-concurrency-tests.md, tasks/open/2026-10-21-komikan-web-ui.md

## Outcome

The MVP is live at `https://komikan.tequionet.org`: two players can create and
share a room, play a full game across separate devices, and the state survives
a service restart.

## Scope

- Database location on the VPS: a data directory the `dev` user can write,
  with the path passed to the service through the environment or a flag.
- systemd unit for `komikan.service`: the database path and address set, the
  service still runs as `dev` with `Restart=always`.
- `deploy.sh` updated for the new binary and the data path; keep the atomic
  copy and the health check.
- nginx: confirm the existing site still proxies correctly; only change it if
  the new endpoints require it (SSE is not used in the MVP).
- A short runbook in `features/` covering: where the database lives, how to
  back it up, how to restart, and how to check health.

## Acceptance criteria

- `sh deploy.sh` succeeds from the phone and the health check passes.
- A room can be created in one browser, shared, and joined in a second
  browser; a full game can be played.
- The game state survives `systemctl restart komikan`.
- The service starts automatically after a reboot.
- The database file is owned by the service user and is not world-readable.
- The site continues to serve over HTTPS with a valid certificate.

## Verification

- `sh deploy.sh` from the phone.
- `curl -fsS https://komikan.tequionet.org/healthz` from outside the VPS.
- Two browsers on separate networks, or one phone and one desktop, playing a
  game.
- `systemctl restart komikan` followed by reloading the room page and
  confirming the position is unchanged.

## Notes and open points

- The deployment pattern is already documented in
  `features/tao-vps-go-service-deployment.md`; this task adds the
  komikan-specific parts rather than repeating the general pattern.
- The site currently serves a coming-soon page, so this is the first
  deployment that changes what the URL shows.
