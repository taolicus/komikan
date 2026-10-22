# Deploy the playable MVP to komikan.tequionet.org

- Date: 2026-10-21
- Area: deployment
- Dependencies: tasks/open/2026-10-21-komikan-integration-and-concurrency-tests.md, tasks/open/2026-10-21-komikan-web-ui.md

## Status

Everything is done except one thing that needs a person: **two people on two
devices playing a whole game**. The service is live, the state survives a
restart, and the site is served over HTTPS with a valid certificate.

What was checked, and how:

- `sh deploy.sh` succeeds from the phone, and the health check passes. The
  commit that is running is stamped into the binary and shows up in the
  service log.
- The database is at `/home/dev/komikan/data/komikan.db`, owned by `dev`.
  The unit now sets `DB_PATH` and `UMask=0027`, so the database and its
  write-ahead log are `0640`: readable by the service user and by nobody
  else. The unit file is kept in the repository as `deploy/komikan.service`.
- The game state survives `systemctl restart komikan`, checked through the
  API: a room with a played move was still at the same version, with the
  same position and the same seat, after the restart.
- The service is `enabled`, so it starts again after a reboot.
- The certificate is valid until 2027-01-17.
- The runbook is `features/komikan-deployment-runbook.md`. The backup and
  the restore were both run on the server and checked afterwards: the
  restored database opens, and the room keeps its seats, its version and its
  position.

Still to do:

- two people, on two devices, opening `https://komikan.tequionet.org/rooms/`,
  sharing a room link and playing a game to the end. Everything else can be
  checked from here; that one needs two humans and two networks.

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
  the new endpoints require it. (The room event stream was added later and
  needed no change: the vhost already proxies HTTP/1.1 with a long read
  timeout, and the stream itself turns buffering off with
  `X-Accel-Buffering: no`.)
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
- **Update 2026-10-22:** the room event stream was checked through the public
  URL with two cookie jars: a move played by one browser arrived on the
  other's open stream, and the payload carried the new threshold. The
  deployed binary reports `version=edc2b2c`.
