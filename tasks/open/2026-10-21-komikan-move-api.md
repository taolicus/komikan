# Move submission API and authoritative game state

- Date: 2026-10-21
- Area: web
- Dependencies: tasks/open/2026-10-21-komikan-rules-engine.md, tasks/open/2026-10-21-komikan-storage-layer.md, tasks/open/2026-10-21-komikan-identity-and-rooms.md

## Outcome

A player can submit a move over HTTP, the server decides whether it is legal,
the authoritative state advances atomically, and the response tells the client
everything it needs to redraw the board.

## Scope

- `POST /api/rooms/{roomID}/moves` with a JSON body containing the move
  (`from`, `to`, and optionally the expected `version`).
- Server-side validation in this order: the caller holds a seat in this room;
  the room is active; it is the caller's turn; the move is legal under the
  rules engine; the game is not already finished.
- The move is applied through the store in one transaction, with the version
  check, so a stale submission is rejected instead of overwriting a newer
  state.
- `GET /api/rooms/{roomID}` returns the full client-visible state: board
  topology (nodes, edges, coordinates), current position, current player,
  captured count and threshold, turn number, status, winner, version, and the
  caller's role and seat. Legal moves are included only for the seat holder
  whose turn it is.
- Error responses with stable status codes and a short machine-readable code:
  `400` malformed, `403` not a seat holder or not your turn, `404` unknown
  room, `409` stale version, `422` illegal move.
- A documented request/response shape, recorded in the task outcome and in
  `features/`.

## Acceptance criteria

- A legal move from the seat holder whose turn it is is accepted, the state
  advances, and the response reflects the new position and the next player.
- An illegal move is rejected with `422` and leaves stored state unchanged.
- A move from the wrong seat, or out of turn, is rejected with `403`.
- Two concurrent submissions for the same turn yield exactly one success and
  one `409`.
- A submission carrying a stale `version` is rejected with `409`.
- A move submitted after the game ended is rejected.
- The response contains the board topology needed to render the board, and
  the rendered board is guaranteed to match the graph the rules engine used.
- `go test ./internal/httpapi/...` passes on the device.

## Verification

- `go test ./internal/httpapi/...` on the device.
- Manual check with two cookie jars: submit a move as one player and observe
  the state change from the other player's session.

## Notes and open points

- The rules engine is the only place that decides legality. The API layer
  must not reimplement any rule.
- Polling or SSE for opponent moves is a client concern and belongs to the UI
  task.
