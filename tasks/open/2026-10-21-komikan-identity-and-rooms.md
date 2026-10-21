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
