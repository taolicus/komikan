# Player identity and room lifecycle

- Date: 2026-10-21
- Area: web
- Dependencies: tasks/open/2026-10-21-komikan-server-layout-and-skeleton.md, tasks/open/2026-10-21-komikan-storage-layer.md, issues/2026-10-21-komikan-client-architecture-choices.md, issues/2026-10-21-komikan-rules-and-role-assignment-questions.md

## Outcome

A browser gets a durable identity on first visit, can create and share a
room, can join a room from its URL, and can list the rooms it belongs to.
Seats and roles are bound to the browser identity and rooms stay isolated
from each other.

## Scope

- Token generation: cryptographically random, at least 128 bits, from
  `crypto/rand`; stored server-side as a hash, never logged or echoed.
- Token delivery per the decision in the client architecture issue
  (provisional: `HttpOnly` cookie plus CSRF protection for writes).
- Endpoints:
  - `POST /api/rooms` — create a room; the creator takes seat 1 and the
    confirmed role.
  - `POST /api/rooms/{roomID}/join` — take the free seat, or restore the
    existing seat when the same identity returns.
  - `GET /api/rooms` — list rooms for the requesting identity.
  - `GET /api/rooms/{roomID}` — room summary for a seat holder.
- Room identifiers: at least 128 bits of randomness, not sequential.
- Role assignment per the confirmed answer (provisional: the creator picks
  their role at creation, the joiner takes the other one).
- Room status values: `WAITING` (one seat), `ACTIVE` (two seats),
  `FINISHED` (game over).
- Errors: `404` for an unknown room, `403` for a request from a non-seat
  holder, `409` for a full room, `400` for a malformed request.

## Acceptance criteria

- A first-time client receives an identity that survives a reload and a
  restart of the server.
- Two different identities create two different rooms that do not share any
  state.
- A second identity joining a room binds seat 2; a third identity is refused
  with `409`.
- Re-opening a room URL from the browser that already holds a seat restores
  that seat instead of allocating another one.
- `GET /api/rooms` returns only rooms the requesting identity is seated in.
- Room ids are not guessable: two rooms created back to back have unrelated
  ids.
- A test asserts that the raw token never appears in a response body, a URL,
  or a log line.

## Verification

- `go test ./internal/httpapi/...` on the device, using two separate cookie
  jars to simulate two browsers.
- Manual check against the running local server.

## Notes and open points

- The move endpoint is not part of this task; it is the next task.
- The room URL path (`/rooms/{roomID}`) is defined here because the UI task
  links to it.

## Completion notes

- **Status: completed on 2026-10-21.**
- **Role assignment: random on join**, as the user decided. The creator
  takes seat 1 without a role; when the second player joins, both seats get
  a role at random (one `PANGUI`, one `TREWA`) and the room becomes
  `ACTIVE`.
- The browser identity is a 256-bit token from `crypto/rand`, delivered as
  an `HttpOnly` cookie named `komikan_session`. Only its SHA-256 hash is
  stored, so a restart of the server keeps every identity, and the raw token
  never reaches a response body or a log line.
- A second cookie, `komikan_csrf`, holds a token derived from the session
  token (HMAC-SHA256). The client echoes it in the `X-CSRF-Token` header on
  writes. A write from a browser that already has a session must carry that
  header; a first contact (no session yet) must carry an `Origin` header
  that matches the host, which browsers always send on a write.
- Endpoints implemented: `POST /api/rooms`, `POST /api/rooms/{roomID}/join`,
  `GET /api/rooms`, `GET /api/rooms/{roomID}`, and
  `GET /api/rooms/{roomID}/board` (the board of a room rendered as SVG).
  Errors: `404` unknown room, `403` not a seat holder, `409` full room.
- Room ids are 128 random bits, so two rooms created back to back have
  unrelated ids.
- `web/rooms.html` is a small page that fills itself from the API: it lists
  the player's rooms, creates one, joins one, shows the seats, the roles and
  the board. It lives at `/rooms/` and `/rooms/{roomID}`; the coming-soon
  page at `/` is untouched.
- Verified on the device: `go test ./internal/httpapi/...` passes with two
  cookie jars standing in for two browsers, and a manual run against the
  real binary confirmed create, join, role assignment, the board endpoint
  and the `403` for an outsider.
