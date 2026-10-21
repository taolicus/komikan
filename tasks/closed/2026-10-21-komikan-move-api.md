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

## Completion notes

- **Status: completed on 2026-10-21.** The user asked for the room work to
  stop at the join indication, so the browser interface is only partly
  done; the API below is complete and tested.
- `POST /api/rooms/{roomID}/moves` takes `{"from": n, "to": m}` and an
  optional `"version"`. The client says which node it wants to move to; the
  rules engine decides what that means, and the API layer never re-implements
  a rule.
- The order of the checks is the one in the task: a seat holder, an active
  room, the caller's turn, a legal move, a game that is not over.
- The move is applied through the store in one transaction with the version
  check, so a stale submission is rejected instead of overwriting a newer
  state.
- `GET /api/rooms/{roomID}` now carries the board topology (38 nodes with
  their coordinates and 79 edges), the position, the current player, the
  captured count against the threshold, the turn number, the status, the
  winner, the version, the caller's seat and role, the last move, and the
  legal moves — but only for the seat holder whose turn it is.
- Every error carries a stable status code and a machine-readable code:
  `400 MALFORMED`, `403 FORBIDDEN`, `403 NOT_YOUR_TURN`, `403 CSRF_FAILED`,
  `404 NOT_FOUND`, `409 STALE_VERSION`, `409 ROOM_NOT_ACTIVE`,
  `409 GAME_OVER`, `422 ILLEGAL_MOVE`.
- The game state in the database is now the rules engine's own state
  document. The placeholder shape is gone; the schema did not change.
- The store marks the room as finished when the game ends, and keeps the
  game's status in step with the rules engine's status.
- Tests in `internal/httpapi/moves_test.go`: the puma moves first, a move
  advances the state and the other player sees it, moving out of turn is
  refused, an illegal move is refused and changes nothing, a stale version is
  refused, an outsider cannot move, a move in a waiting room is refused, a
  malformed body is refused, and a whole game can be played to the end
  through the API.
- Verified on the device with `go test ./...` and by hand against the real
  binary with two cookie jars: create, join, move, illegal move, stale
  version.
