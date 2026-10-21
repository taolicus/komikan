# Server layout, configuration, and skeleton

- Date: 2026-10-21
- Area: server

## Outcome

The project has a package layout that the rules, storage, API, and UI tasks
can build on, and a server binary that starts, reports health, and shuts down
cleanly.

## Scope

- Reshape `cmd/server` into a thin entry point over internal packages:
  - `internal/board` (board data)
  - `internal/rules` (game logic)
  - `internal/store` (persistence, added by a later task)
  - `internal/httpapi` (HTTP handlers, added by a later task)
  - `web/` (static assets, embedded; the coming-soon page moves here)
- Configuration from the environment and flags: `ADDR` (existing default
  `127.0.0.1:18082`), and a placeholder for the database path.
- A `/healthz` endpoint returning `200` with a small body.
- Graceful shutdown on `SIGTERM`/`SIGINT` so a systemd restart does not cut
  requests in flight.
- Keep serving the current coming-soon page at `/` until the UI task
  replaces it, so the deployed site does not break in the meantime.
- A `-version` flag or equivalent that prints the build version.

## Acceptance criteria

- `go build ./...` and `go test ./...` succeed on this device.
- The server starts locally, `/healthz` returns `200`, and `/` still serves
  the coming-soon page.
- The server exits cleanly on `SIGTERM` within a second or two, and the exit
  code is zero.
- No global mutable state in the server wiring; handlers receive their
  dependencies explicitly.
- The embed of static assets lives in one place that the UI task extends.

## Verification

- `go run ./cmd/server` on the device, then `curl` `/healthz` and `/`.
- Send `SIGTERM` and confirm a clean exit.

## Notes and open points

- This task is deliberately small and unblocks the parallel work of the
  storage and API tasks. It makes no behavioural change to the deployed site.
