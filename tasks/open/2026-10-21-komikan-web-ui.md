# Web interface: home page and room board

- Date: 2026-10-21
- Area: web
- Dependencies: tasks/closed/2026-10-21-komikan-move-api.md, tasks/closed/2026-10-21-komikan-board-data-and-renderer.md, issues/2026-10-21-komikan-client-architecture-choices.md

## Status

**Parked on 2026-10-21 at the user's request.** The room work was enough for
this round: the rooms exist, a browser can create one and share it, the
second player joins it, and the page says so. The rest of this task is
picked up later.

What is already there, in `web/rooms.html`:

- the room list at `/rooms/`, with a create action and the role of each
  room;
- the room page at `/rooms/{roomID}`: the board, whose turn it is, the
  captured count against the threshold, the winner at the end, and the link to
  share with the second player;
- the board drawn in the browser as SVG from the topology the server sends,
  with the puma and the dogs drawn distinctly, the last move marked, and the
  legal destinations highlighted when a piece is picked;
- clicking a destination submits the move; the other player's move is
  pushed to the page over a Server-Sent Events stream, so the page does not
  poll.

And in `web/index.html`, the home page:

- a primary button that creates a game and takes the player straight to it;
- the list of the games of the browser, right away, with the state of each
  one and the role of the player;
- a short description of the game under the board.

The game screen was reshaped on the user's word on 2026-10-22: the title and
the subtitle are gone, the turn and the score sit under the board, the colour
legend and the table of seats are gone, the link to share is a field with the
whole URL plus a share button and a copy button above the board, visible only
while the room waits for the second player, and "Volver" is a link to the
home page.

What is still to do:

- ~~the phone layout was never looked at by a person~~ — **checked by the
  owner on 2026-10-22: the phone layout is fine.**
- ~~the room page polls every two seconds and rebuilds the whole board on
  every poll~~ — **done on 2026-10-22:** the room is pushed over
  `GET /api/rooms/{roomID}/events`, the page keeps one `EventSource`, and a
  fallback poll only runs if the browser cannot keep a stream. The board is
  still redrawn on each update, but an update now only happens when the room
  actually changed.
- ~~the waiting view is plain~~ — with live updates, the share field
  disappears and the board starts as soon as the second player joins.
- the game page has no move history, no captured pieces shown on the board,
  and no way to see the opponent's last move other than the marked node;
- the home page lists the games of the browser, but it does not say who the
  opponent is, and it does not refresh while it stays open.

The last two are deliberately out of scope for this round: on 2026-10-22 the
owner picked live updates as the only interface item to address now. The rest
stays parked here.

## Outcome

Two embedded pages replace the coming-soon screen: a home page listing the
player's rooms with a create action, and a room page that renders the board,
accepts moves, shows whose turn it is, and reports the result.

## Scope

- Static assets under `web/`, embedded into the binary, no build step
  (per the client architecture decision).
- Home page:
  - lists the rooms for the current identity, with status and the opponent's
    presence;
  - "create room" action, including the role choice if role assignment is
    not fixed;
  - a link back to the room list without losing progress.
- Room page at `/rooms/{roomID}`:
  - waiting view while only one seat is taken, with a copy-link button;
  - board rendered as SVG from the topology the server sends, so every drawn
    edge corresponds to a graph edge and vice versa;
  - pieces drawn distinctly for Pangui and Trewa;
  - selecting a movable piece highlights its legal destinations; captures are
    visually distinguished from normal moves;
  - clicking a destination submits the move; occupied and invalid
    destinations are not clickable, and an illegal submission shows a clear
    message;
  - indicators for the current player, whose seat you are, the number of
    captured Trewa against the threshold, and the winner when the game ends;
  - opponent moves are pushed over the room's event stream (per the
    live-update task), with a poll only as a fallback.
- The interface is usable on a phone screen as well as a desktop.
- The room page works from a page reload and from a fresh browser with the
  same identity.

## Acceptance criteria

- The pages are served from the embedded binary; no separate frontend server
  or runtime is required.
- The rendered board shows every node and every edge of the server's graph,
  and the count of drawn edges matches the count in the response.
- Clicking a legal destination advances the board after the server accepts the
  move; an illegal move leaves the board unchanged and shows a message.
- The interface distinguishes the Pangui from the Trewa, shows whose turn it
  is, shows the captured count against the threshold, and shows the winner at
  game end.
- Reloading the page restores the current position and turn from the server.
- The layout is usable at a narrow viewport (phone) and a wide one. Checked
  by the owner on a phone on 2026-10-22.
- The coming-soon page is no longer served at `/`.

## Verification

- Manual use on the phone browser against a local server, with two browser
  profiles to simulate two players.
- A test that fetches `/` and `/rooms/{id}` and asserts they return the
  expected assets.

## Notes and open points

- The UI reads only what the API returns; it does not compute legal moves.
  If the server includes legal moves only for the player to move, the client
  must handle the other player's turn by simply disabling input.
- No build step means no bundling: keep scripts small and avoid dependencies
  that would need one.
