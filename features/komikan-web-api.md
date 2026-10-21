# Komikan web API, identity, and rooms

Durable knowledge about the HTTP surface of the Komikan server: how a browser
is identified, how rooms work, and what the endpoints answer.

## Layout

- `internal/httpapi` owns routing, identity, and the JSON shapes.
- `internal/store` owns persistence; the API never sees SQL.
- `web/` holds the static pages, embedded with `go:embed`.

## Identity

- A browser gets a 256-bit token from `crypto/rand` on its first request.
  It is delivered as an `HttpOnly` cookie named `komikan_session` and only
  its SHA-256 hash is stored, so identities survive a restart of the server
  and the raw token never appears in a response or a log line.
- A second cookie, `komikan_csrf`, holds `HMAC-SHA256(session, "komikan-csrf")`.
  It is readable by scripts on purpose: the client echoes it in the
  `X-CSRF-Token` header on writes, and the server compares it with the value
  it derives from the session it holds.
- `Secure` is set when the request arrives over TLS or with
  `X-Forwarded-Proto: https`. The nginx vhost sets that header, so the
  public site gets secure cookies while a local run over plain HTTP does not.
- A write (anything but GET/HEAD) must come from our own `Origin`; when the
  browser already has a session it must also carry the CSRF header. A first
  contact has no token yet, so a matching `Origin` is the only proof, and
  browsers always send `Origin` on a write.

## Rooms

- A room id is 128 random bits (`store.NewID`), never sequential.
- The creator takes seat 1 with no role. When the second player joins, both
  seats are given a role **at random** (one `PANGUI`, one `TREWA`) and the
  room becomes `ACTIVE`. A third player gets `409`.
- Room status: `WAITING` (one seat), `ACTIVE` (two seats), `FINISHED`.
- Re-opening a room URL with the same identity restores the existing seat.
- The room stores the ruleset document, but every room starts from the same
  default ruleset for now.

## Endpoints

| Method | Path | What it does |
|--------|------|--------------|
| GET | `/` | the coming-soon page |
| GET | `/rooms/` | the rooms page: list, create, join |
| GET | `/rooms/{roomID}` | the same page, focused on one room |
| GET | `/healthz` | `200 ok`, used by the deploy script |
| GET | `/favicon.svg` | the leopard favicon |
| POST | `/api/rooms` | create a room, returns `201` and the summary |
| POST | `/api/rooms/{roomID}/join` | take the free seat, or restore the seat |
| GET | `/api/rooms` | the rooms the identity is seated in |
| GET | `/api/rooms/{roomID}` | the room summary, for a seat holder |
| POST | `/api/rooms/{roomID}/moves` | play a move, returns the new summary |
| GET | `/api/rooms/{roomID}/board` | the board of the room as `image/svg+xml` |

Status codes: `400` malformed, `403` not a seat holder, out of turn, or a
failed CSRF check, `404` unknown room, `409` full room, a stale version, a
room that has not started, or a game that is over, `422` an illegal move.

Every error body carries a message and a machine-readable code:
`MALFORMED`, `FORBIDDEN`, `NOT_YOUR_TURN`, `CSRF_FAILED`, `NOT_FOUND`,
`STALE_VERSION`, `ROOM_NOT_ACTIVE`, `GAME_OVER`, `ILLEGAL_MOVE`.

## Playing a move

`POST /api/rooms/{roomID}/moves` takes the node the piece stands on and the
node it goes to:

```json
{"from": 7, "to": 11, "version": 3}
```

The client says where it wants to go; the rules engine decides what that
means. The version is optional: when it is given and does not match, the
move is refused with `409 STALE_VERSION`. The response is the whole room
summary, so the client can redraw without a second request.

The summary carries the board topology (38 nodes with their coordinates and
79 edges), so the board the browser draws is the same graph the rules engine
plays on. It also carries the legal moves, but only for the seat holder
whose turn it is.

## The game state document

The state is a JSON blob in the `games` row, next to a `version` column that
rejects stale writes. It is the rules engine's own state document:

```json
{
  "boardLayoutId": "komikan-standard",
  "position": {"7": "PANGUI", "23": "TREWA"},
  "currentPlayer": "PANGUI",
  "capturedTrewa": 0,
  "turnNumber": 1,
  "captureSequenceActive": false,
  "gameStatus": "IN_PROGRESS",
  "ruleset": {"...": "..."},
  "moveHistory": []
}
```

The position is stored as a map from node id to piece name, so it reads well
in the database and in a log line. The schema and the store interface did
not change when the rules engine landed.
