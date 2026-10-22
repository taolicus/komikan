# Push room updates instead of polling

- Date: 2026-10-22
- Area: web
- Resolved: 2026-10-22
- Dependencies: `features/komikan-web-api.md`, `tasks/closed/2026-10-21-komikan-move-api.md`

## Outcome

A room page is told about a change instead of asking for it. The server
exposes `GET /api/rooms/{roomID}/events`, a Server-Sent Events stream open to
a seat holder, which sends the room summary once on connect and again
whenever the room changes. A move and a join publish to it after they commit.
The page keeps one `EventSource` and does not poll.

The change also carried the two rule decisions of the same day that touch the
page: the capture threshold is 8, and a capture chain may be stopped, which
adds a "Terminar la captura" button and an `END_TURN` move.

## Scope as carried out

- `internal/httpapi/events.go`: an in-process hub keyed by room id, and the
  streaming handler. The event is a signal, not a copy of the game: each
  listener re-reads the room and renders it for its own player, so legal
  moves and the seat are never sent to the wrong person.
- `POST /api/rooms/{roomID}/join` and `POST /api/rooms/{roomID}/moves`
  publish after they commit.
- `X-Accel-Buffering: no` on the stream, and a `: ping` comment every 25
  seconds, so nginx does not hold the events back or drop an idle stream.
- Shutdown: `cmd/server/main.go` gives the HTTP server a base context and
  cancels it before `Shutdown`, so the open streams end instead of making
  every restart wait out the 5-second timeout.
- `web/rooms.html`: one `EventSource` per open room, a five-second poll only
  if the browser has no `EventSource` or the stream gives up, the end-turn
  button, and no `END_TURN` destination drawn on the board (the puma's own
  point is where a move is picked, not where it goes).

## Acceptance criteria

- Opening a room connects the stream and receives the room at once.
- A move by the other player, and the second player joining, reach the open
  page without a poll.
- A browser that is not seated in the room is refused.
- The page makes no periodic request while the stream is open.
- A deploy does not wait on open streams.

## Verification

- `go test ./...` — `internal/httpapi/events_test.go` covers the first
  event, a pushed move (with the receiver's own legal moves), a pushed join,
  and the refusal of a stranger. `TestPanguiMayEndACaptureChain` covers the
  rules side of stopping a chain.
- `node --test "web/**/*.test.js"` — the page opens one stream, does not
  poll, renders a pushed event, falls back only when the stream closes, and
  plays the end-turn move.
- Manual run against a local binary: a second browser's move arrived on an
  open `curl -N` stream, and `SIGTERM` with the stream open exited in 75 ms.
- Deployed with the rest of the change; see the commit that carries
  `events.go`.

## Notes

- The hub is per process. That is correct while the service is one process
  against one SQLite file; a multi-process deployment would need a shared
  bus, and the comment in `events.go` says so.
- The event carries the whole room summary, board topology included. It is
  about a kilobyte, and the board never changes, so it is a candidate for
  trimming if a room ever gets busy.
- nginx still has to be trusted to pass the stream through. The header is on
  the response; if a future proxy ignores it, the stream would arrive in
  bursts.
