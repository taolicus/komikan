# Persistence technology and schema choices

- Date: 2026-10-21
- Area: persistence

## Problem

`docs/spec.md` asks for "a storage layer for rooms, player identities, and
game states. The initial implementation may use SQLite." The MVP needs a
database that works on the phone (for local testing) and cross-compiles to
linux/amd64 with `CGO_ENABLED=0` for deployment.

## Verified on this device (2026-10-21)

- `modernc.org/sqlite` (pure Go, no cgo) builds and runs on this phone
  (Termux, `android/arm64`): a trivial `create table` / insert round trip
  worked.
- The same module cross-compiles to `linux/amd64` with `CGO_ENABLED=0`, which
  is what `deploy.sh` needs.
- `github.com/mattn/go-sqlite3` was not tested: it needs cgo, and the
  deployment build path disables cgo.

So the driver question is effectively settled; what remains is the schema and
the concurrency model.

## 1. Storage engine

- **(a) SQLite via `modernc.org/sqlite`.** One file, transactional, matches
  the spec wording, verified above.
- **(b) JSON files on disk.** No dependency, trivially readable, but
  concurrency and crash-safety are hand-rolled.
- **(c) bbolt.** A single embedded key-value file, but it is not a relational
  store and the spec leans toward SQLite.

Proposed default: **(a)**.

## 2. How game state is stored

- **(a) One JSON blob per game row.** The whole `GameState` (position, turn,
  counters, move history) serialised into a single column. Flexible, and the
  rules engine already produces a serialisable state.
- **(b) Normalised rows** for position, pieces, and moves. Queryable, but
  much more code and no MVP feature needs the queries.

Proposed default: **(a)**, with a `version` integer column alongside it.

## 3. Concurrency control for moves

- **(a) Optimistic versioning.** The client sends the `version` it saw; the
  server updates the row only where the version still matches, and returns
  `409 Conflict` otherwise. Fits the "state version or revision number" in
  the spec.
- **(b) Serialise per room in the process** with a mutex, and rely on the
  database transaction for durability. Simple, correct for a single-process
  server.
- **(c) Both.** Optimistic version check inside the transaction, which
  already excludes the second writer.

Proposed default: **(c)** — the version check inside the write transaction
covers a single-process server and a future multi-process one. Concurrent
submissions for the same turn must result in exactly one accepted move.

## 4. Schema sketch

- `players(id, token_hash, created_at)`
- `rooms(id, created_at, updated_at, status, ruleset_json, role_assign)`
- `seats(room_id, player_id, seat, role)` — or fold seats into `rooms`
- `games(room_id, state_json, version, status, winner, updated_at)`
- `moves(room_id, seq, player_id, payload_json, created_at)`

Proposed default: keep `seats` as its own table so a player can hold seats in
several rooms without duplicating room columns, and keep `moves` as its own
table rather than a JSON array inside the game blob, so history can grow
without rewriting the state blob on every move.

## 5. Operational details

- Database path: a `-db` flag / `DB_PATH` environment variable, defaulting to
  a file next to the binary on the VPS (`/home/dev/komikan/data/komikan.db`),
  owned by the `dev` user.
- `PRAGMA journal_mode=WAL` and a busy timeout, so a reader never blocks the
  writer.
- Schema versioning: a single `schema.sql` applied at startup, tracked with
  `PRAGMA user_version`, so upgrades do not need a migration framework for
  the MVP.

## Next step

Confirm or override. If you have no preference, reply "defaults" and the
storage task proceeds with SQLite, a JSON state blob plus a version column, and
a version check inside the write transaction.

**Answered on 2026-10-21: the defaults were applied.** The storage task is
done (`tasks/closed/2026-10-21-komikan-storage-layer.md`): SQLite through
`modernc.org/sqlite`, one JSON blob per game with a `version` column, the
version check inside the write transaction, the schema sketch above (with
`seats` and `moves` as their own tables), and the database at
`data/komikan.db` relative to the working directory, overridable with `-db`
or `DB_PATH`.
