# Harden the public endpoint before more people use it

- Date: 2026-10-22
- Area: deployment

## Problem

The service is public at `https://komikan.tequionet.org` and is about to be
used by more than the people who built it. It was written as a small service
for a handful of players, and the parts that matter once a site is open to
the world are not there.

Checked on 2026-10-22 against the running service and the code:

- **No security headers.** A request for `https://komikan.tequionet.org/`
  comes back with `server`, `date`, `content-type` and `cache-control`, and
  nothing else. There is no `Content-Security-Policy`, no
  `X-Content-Type-Options`, no `X-Frame-Options`, no `Referrer-Policy` and
  no `Strict-Transport-Security`. The pages carry a session cookie, so the
  transport should be pinned.
- **No limit on the body a request may carry.** `handleSubmitMove` decodes
  the body with `json.NewDecoder(r.Body)` and no `http.MaxBytesReader`
  around it (`internal/httpapi/game.go`). nginx caps the body at its
  default 1 MB, so the public path is covered today, but the server should
  not depend on the proxy for that.
- **No timeouts on the body.** The server sets `ReadHeaderTimeout` and
  nothing else (`cmd/server/main.go`), so a client that opens a connection
  and dribbles a body slowly holds that connection for as long as it
  likes.
- **No limit on how many rooms a visitor may create.** Every
  `POST /api/rooms` writes a room, a seat, a game and a player row. Nothing
  in the store ever deletes anything, so a script can fill the database in
  a minute and the rows stay there.
- **The pages load a third-party stylesheet.** Both pages pull the Cinzel
  font from `fonts.googleapis.com`, which tells a third party who visited
  and fails outright on a network that blocks it. The CSS has a fallback
  stack, so the page still reads, but the request still goes out.

## Scope

- Response headers on every response:
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: no-referrer`
  - `X-Frame-Options: DENY`
  - `Strict-Transport-Security`, set by nginx, pinning HTTPS
  - a `Content-Security-Policy` the pages actually satisfy. Both pages
    carry their CSS and their JavaScript inline, so a policy that forbids
    inline script is not possible without moving them into their own files
    under `web/`. Moving them is in scope if that is what it takes to get a
    policy worth having; a policy full of `'unsafe-inline'` is not.
- `http.MaxBytesReader` around every request body, with a limit that is
  generous for a move (a few kilobytes) and small enough to matter.
- `ReadTimeout`, `WriteTimeout` and `IdleTimeout` on the server, so a slow
  client cannot hold a connection open. `WriteTimeout` must not cut the
  room event streams (`internal/httpapi/events.go`): either leave it at zero
  and rely on `ReadTimeout` and `IdleTimeout`, or give the stream handler a
  deadline of its own with `http.ResponseController`. `IdleTimeout` only
  applies between requests, so it is safe for a stream.
- A limit on room creation per identity, answered with `429 Too Many
  Requests`. The limit should be high enough for a real person (say a few
  dozen rooms) and low enough to stop a script.
- The Google Fonts dependency either removed (the system font stack the
  CSS already falls back to is fine) or reduced to one origin that the
  policy allows.

## Acceptance criteria

- A move submission with a body larger than the limit is refused with `413`
  before the body is read.
- The server has read, write and idle timeouts, and a slow request is cut
  off.
- Every response carries the headers above, and the pages still load and
  play with the policy in place.
- An identity that creates more rooms than the limit in a window gets
  `429`, and a normal player never reaches it.
- A whole game still plays end to end on the public URL after the change,
  including the event stream (a `Content-Security-Policy` needs
  `connect-src 'self'` for it, and a `WriteTimeout` must not cut it).

## Verification

- `go test ./...` and `node --test "web/**/*.test.js"` on the phone.
- A test that asserts the headers on a response, and one that asserts a
  large body is refused.
- `sh deploy.sh`, then `curl -fsS https://komikan.tequionet.org/healthz`.
- `curl -fsSI https://komikan.tequionet.org/` shows the headers.

## Notes and open points

- The nginx configuration is not in the repository; the headers that belong
  to the proxy (`Strict-Transport-Security`) have to be added there and
  checked on the host.
- The room limit is a blunt instrument: it cannot tell a person from a
  script. It only has to be good enough to keep the database from filling
  up on its own.
