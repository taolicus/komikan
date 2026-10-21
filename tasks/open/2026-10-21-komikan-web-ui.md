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
- the room page at `/rooms/{roomID}`, with the seats, the roles, whose turn
  it is, the captured count against the threshold, the winner at the end,
  and a copy-link button;
- the board drawn in the browser as SVG from the topology the server sends,
  with the puma and the dogs drawn distinctly, the last move marked, and the
  legal destinations highlighted when a piece is picked;
- clicking a destination submits the move, and the board is refreshed by
  polling every two seconds.

What is still to do:

- the phone layout was never looked at by a person; the widths, the touch
  targets and the scroll on a narrow screen need a real look;
- the waiting view (only one seat taken) is plain: no invitation screen, no
  copy-link prominence, no indication that the second player has arrived
  beyond the status line;
- the game page has no move history, no captured pieces shown on the board,
  and no way to see the opponent's last move other than the marked node;
- the coming-soon page is still served at `/`, and this task's scope says it
  should be replaced by the home page listing the player's rooms. The user
  asked for the coming-soon page to stay, so it stays until they say
  otherwise.

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
  - opponent moves are picked up by polling (per the client decision).
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
- The layout is usable at a narrow viewport (phone) and a wide one.
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
