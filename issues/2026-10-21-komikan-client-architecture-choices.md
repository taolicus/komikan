# Client-side architecture choices for the MVP

- Date: 2026-10-21
- Area: web

## Problem

`docs/spec.md` fixes the server side (single Go binary, embedded web
interface, HTTP API) but leaves the browser side open. The decisions below
affect the identity, rooms, and UI tasks, so they are worth settling before
those tasks start.

## 1. How the browser identity is carried

- **(a) `HttpOnly` cookie.** The token never reaches JavaScript. Requires
  CSRF protection (double-submit token or `Origin`/`Referer` checking) and
  `SameSite` handling. Strongest against token theft via XSS.
- **(b) `localStorage` token sent in a custom header.** No CSRF exposure
  (custom headers are not sent cross-origin by default), simpler, but the
  token is reachable from any script on the page.

Proposed default: **(a)**, because the token is a full player credential and
the deployment is a public URL. CSRF protection comes along with it: reject
state-changing requests whose `Origin` does not match the request host, and
require a per-session CSRF token for cookie-authenticated writes.

## 2. Frontend build step

- **(a) No build step.** Hand-written HTML, CSS, and JavaScript embedded with
  `go:embed`. Nothing to install, deploys as part of the binary, works on the
  phone.
- **(b) Bundler or framework** (esbuild/Vite, or a framework runtime). More
  convenient for a growing UI, but adds a toolchain and a second build step
  to `deploy.sh`.

Proposed default: **(a)**, with the source of truth as plain files under
`web/` that are embedded into the binary. The MVP UI is small enough that a
build step buys nothing yet.

## 3. How a client learns the opponent moved

- **(a) Polling.** The room page polls `GET /api/rooms/{id}` every few
  seconds. Simple, robust, no connection management.
- **(b) Server-Sent Events.** Push updates; needs `proxy_buffering off` on
  nginx (the deployment notes mention SSE support) and reconnect handling.

Proposed default: **(a)** polling for the MVP, with an interval around 3-5
seconds while the page is visible. SSE is listed as a future extension in
`docs/spec.md` anyway.

## 4. Board rendering technique

- **(a) SVG.** The server sends the node coordinates and edges; the client
  draws SVG elements. Crisp at any size, easy to attach click handlers and
  highlight states, scales for mobile.
- **(b) Canvas.** More manual work for hit-testing and highlights.
- **(c) DOM elements positioned absolutely.** Simplest, least pretty.

Proposed default: **(a) SVG**, since the board data already carries 2D
coordinates and the requirement "every displayed connection must correspond to
a graph edge, and every graph edge must be represented visually" is easiest
to guarantee when the client renders straight from the graph.

## 5. Page structure

- **(a) Two pages:** a home page (room list, create room) and a room page at
  `/rooms/{roomID}` (the shareable URL the specs require).
- **(b) One page** with client-side routing.

Proposed default: **(a)**, matching the required shareable room URL.

## Next step

Confirm or override each default. If you have no strong preference, reply
"defaults" and the tasks will proceed with them; changing them later is
contained, but the identity choice in particular is cheaper to change before
the rooms task exists.
