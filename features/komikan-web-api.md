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
| GET | `/` | the home page: the board, and the action to start a game |
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

The board SVG is transparent and drawn for the dark page; see
`features/komikan-board-layout.md` for the colours.

Status codes: `400` malformed, `403` not a seat holder, out of turn, or a
failed CSRF check, `404` unknown room, `409` full room, a stale version, a
room that has not started, or a game that is over, `422` an illegal move.

Every error body carries a message and a machine-readable code:
`MALFORMED`, `FORBIDDEN`, `NOT_YOUR_TURN`, `CSRF_FAILED`, `NOT_FOUND`,
`STALE_VERSION`, `ROOM_NOT_ACTIVE`, `GAME_OVER`, `ILLEGAL_MOVE`.

`ROOM_NOT_ACTIVE` means the room has not started yet: the second seat is
still empty. `GAME_OVER` means the game is over and the room is finished.

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

## The two pages

`web/index.html` is the home page and `web/rooms.html` is the page of a
room; both are plain scripts in static files, embedded in the binary.

- The home page starts a game with a button (`POST /api/rooms`) and lists the
  games of the identity (`GET /api/rooms`), with the state of each one and
  the role of the player. The list is also the first call of the page, so it
  is the call that gives the browser its session and its CSRF token: the
  button waits for the list before it creates a game, and sends the token in
  the `X-CSRF-Token` header. A write that arrives with a session and without
  that header is refused with `CSRF_FAILED`.
- The list of rooms at `/rooms/` is the same page script in its other shape:
  it reads the list, and its "Crear sala" button waits for that read for the
  same reason.
- The room page reads the room (`GET /api/rooms/{roomID}`), draws the board,
  and plays the moves (`POST /api/rooms/{roomID}/moves`). While the room
  waits for the second player it shows the link of the room, with a field
  holding the whole URL, a share button (only where the browser has the
  share API) and a copy button. Once the second player has joined, the link
  is gone: the game is on.

## What a click means

The room page draws the legal moves of the player whose turn it is. A click on
a piece picks it and shows the points it can reach; a click on one of those
points plays that move.

The picked piece decides which move a point means. With the twelve dogs the
same point is often reachable by more than one of them, and the move that is
played has to be the one of the picked dog. A click on a point the picked
piece cannot reach plays nothing.

## The tests of the pages

The page scripts are tested on their own: `web/rooms.test.js` and
`web/index.test.js` run them against the fake DOM of `web/fakedom.js` and a
stubbed fetch, and look at what the page drew and at the requests it made.

```sh
node --test "web/**/*.test.js"   # run the client tests
```

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
