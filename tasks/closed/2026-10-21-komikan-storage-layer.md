# Storage layer for players, rooms, and games

- Date: 2026-10-21
- Area: persistence
- Dependencies: tasks/open/2026-10-21-komikan-server-layout-and-skeleton.md, issues/2026-10-21-komikan-persistence-choices.md

## Outcome

A `internal/store` package persists players, rooms, seats, games, and move
history in SQLite, with atomic move application and a state version that
rejects stale writes.

## Scope

- A `Store` interface with the operations the API needs: create player, get
  player by token, create room, get room, list rooms for a player, join room
  (take seat), load game state, apply move (atomic), list move history.
- A SQLite implementation using `modernc.org/sqlite` (verified to build and
  run on this device and to cross-compile for `linux/amd64` with
  `CGO_ENABLED=0`).
- Schema per the decision in the dependency issue: players, rooms, seats,
  games, moves; schema version tracked with `PRAGMA user_version`; WAL mode
  and a busy timeout.
- Game state stored as a JSON blob with a `version` column.
- Move application in a single transaction that checks the expected version
  and rejects a mismatch, so two concurrent submissions for the same turn
  cannot both succeed.
- Database path from configuration, defaulting to a file path that works for
  local runs and for the VPS deployment.

## Acceptance criteria

- `go test ./internal/store/...` passes on this device against a temporary
  database file.
- A room can be created, joined by a second player, have a move applied, and
  be reloaded with identical state after reopening the store.
- Applying a move with a stale version returns a distinguishable error
  (`ErrConflict` or similar) and does not change stored state.
- Two concurrent move applications against the same room result in exactly
  one success.
- The store exposes no SQL details above the interface; the API layer sees
  domain errors only.
- No schema changes are needed to add a move-history row or a seat role
  beyond what the schema already provides.

## Verification

- `go test ./internal/store/...` on the device.
- Manual check: build the binary, point it at a scratch database, and confirm
  the file is created and reused across restarts.

## Notes and open points

- The engine state type used in the JSON blob is owned by the rules engine
  task. If that task has not landed yet, store a minimal placeholder state
  here and switch to the real type when it exists, keeping the interface
  stable.
- Multi-process deployment is out of scope; the version check inside the
  transaction is what keeps a single-process server correct and would also
  survive a second process.

## Completion notes

- **Status: completed on 2026-10-21.**
- `internal/store` exposes a `Store` interface and domain errors
  (`ErrNotFound`, `ErrConflict`, `ErrInvalid`). The API layer sees no SQL.
- The SQLite implementation uses `modernc.org/sqlite` (pure Go, no cgo), so
  the same code builds on the phone and cross-compiles to `linux/amd64` with
  `CGO_ENABLED=0`. The dependency is now in `go.mod`.
- Schema (`internal/store/schema.sql`): `players`, `rooms`, `seats`, `games`,
  `moves`; applied at startup and tracked with `PRAGMA user_version` (version
  1). WAL mode, `busy_timeout=5000`, `foreign_keys=ON`, and a single
  connection so SQLite never sees two writers.
- The game state is a JSON blob next to a `version` column. `store.State` is
  `json.RawMessage` for now, so the rules engine can replace the document
  shape without touching the schema or the interface.
- `ApplyMove` checks the expected version inside the transaction and writes
  the state and the history row together, so a stale writer is rejected with
  `ErrConflict` and leaves the stored state untouched.
- `JoinRoom` takes the free seat, restores an existing seat when the same
  player returns, refuses a third player with `ErrConflict`, and assigns the
  two roles at random once the room is full.
- `store.NewID()` returns a 128-bit random hex id, used for players and
  rooms.
- Verified on the device: the store tests pass against a temporary database
  file; eight concurrent move applications for the same version produce
  exactly one success; a closed and reopened store returns the same state and
  history.
