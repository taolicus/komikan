# The wrong dog moves when a point is reachable by several of them

- Date: 2026-10-22
- Area: web

## Status

**Done on 2026-10-22.** The room page now plays the move of the piece the
player picked, and the page script has a test of its own.

## The problem

Reported from a live game: "sometimes the wrong piece moves when playing the
dogs".

The room page decided which move a click means in `onNodeClick`:

```js
const move = (room.legalMoves || []).find((m) => m.to === id);
```

It looked for the first legal move that ends on the clicked point, whatever
piece that move starts from. With twelve dogs the same point is often
reachable by more than one of them, so the dog that moved was the first in the
sorted list, not the one the player had picked. The points on the board were
highlighted for the picked dog, but the move that was sent belonged to
another one: the picture said one thing and the game did another.

The server was not at fault. It plays the move it is given, and the move it
was given named the wrong piece.

In the starting position the point 24 is reachable by the dog on 23 and by the
dog on 29. Picking the dog on 29 and clicking 24 played the move 23 -> 24.

## Scope

- A click on a destination plays the move of the piece that was picked, and
  only that move.
- A click on a point the picked piece cannot reach plays nothing.
- The page script gets a test of its own, so the rule stays true.

## Outcome

- `web/rooms.html`: `onNodeClick` looks for the destination among the moves
  of the picked piece, not among all the legal moves. A click without a
  picked piece plays nothing.
- `web/rooms.test.js`: the page script runs against a fake DOM and a stubbed
  fetch, and the test looks at the move request that reaches the server. The
  first version of the test failed with `23 !== 29`, which is the bug the
  player reported.

The server side needed no change: it already plays the move it is given.
