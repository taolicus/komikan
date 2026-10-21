# Cross-cutting integration, persistence, and concurrency tests

- Date: 2026-10-21
- Area: testing
- Dependencies: tasks/open/2026-10-21-komikan-storage-layer.md, tasks/open/2026-10-21-komikan-move-api.md

## Outcome

An automated test suite exercises the whole server the way two players and a
restarting machine would, covering the requirements in `docs/spec.md` §11 that
unit tests cannot reach.

## Scope

- Two simulated browsers with separate cookie jars, playing a game across
  separate sessions:
  - create, join, play alternating moves, reach a terminal state;
  - a player can move while the opponent is absent.
- Room isolation: activity in one room does not change another room's state.
- Identity persistence: the same browser token restores the same seat after
  a reload and after a server restart.
- Persistence and recovery: close the store, reopen it against the same
  database file, and confirm rooms, seats, and game state are intact.
- Concurrency: simultaneous move submissions for the same turn yield
  exactly one accepted move; a stale version is rejected.
- Endpoint validation: malformed bodies, unknown rooms, wrong seats, out of
  turn moves, and moves after game end all return the documented status
  codes.
- The suite runs on this device with `go test ./...` and needs no external
  services.

## Acceptance criteria

- `go test ./...` passes on the device and covers each bullet above with at
  least one test.
- Tests are deterministic: no sleeps where a synchronisation point will do.
- Each test is independent and can run alone with `go test -run`.
- Failures name the requirement they break, so a regression is traceable to
  the spec.
- The suite does not require the deployed VPS or any network access.

## Verification

- `go test ./...` on the device.
- Optional on a laptop: `go test -race ./...`, since the race detector may
  not be available for `android/arm64`.

## Notes and open points

- This task covers the cross-cutting behaviour; rules, board, and store unit
  tests live with their own tasks.
- If the identity mechanism is still undecided, keep these tests written
  against a helper that hides how the token is carried, so a switch from
  cookie to header does not rewrite the suite.
