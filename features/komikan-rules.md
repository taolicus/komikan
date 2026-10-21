# The rules of Komikan, as implemented

Durable knowledge about the game rules in `internal/rules`, and the decisions
behind them.

## Where the rules live

- `internal/rules/ruleset.go` — every value that varies between variants.
- `internal/rules/state.go` — the state, the move, the position.
- `internal/rules/engine.go` — legal moves, applying a move, victory.

The engine holds the board, not the state: a state is a plain document that
can be stored and sent to a browser, and the engine is built once for a
board and a ruleset.

```go
engine, err := rules.NewEngine(board, rules.DefaultRuleset())
state := engine.NewGame()
moves := engine.LegalMoves(state, state.CurrentPlayer)
next, err := engine.ApplyMove(state, moves[0])
```

## The default variant

| Setting | Value | Note |
|---|---|---|
| `panguiCapturesToWin` | 6 | the usual "capture six prey" account |
| `allowMultipleCaptures` | true | the puma may chain jumps in one turn |
| `multipleCapturesMandatory` | true | once a jump is made, it must continue if it can |
| `trewaCanMoveBackward` | true | the dogs may move along any edge |
| `panguiMovesFirst` | true | |
| `puTrewaWinByBlocking` | true | the puma with no legal move loses |
| `puTrewaWinByOccupyingTriangle` | false | the canonical board has no triangle area |
| `capturePreferred` | true | the puma must capture when it can |
| `drawWhenTrewaCannotMove` | true | the dogs with no legal move end the game in a draw |
| `thresholdBeforeBlocking` | true | the capture that wins the game is not cancelled by the position it leaves |

These are the defaults proposed in the open rule issues, not yet endorsed by
the user. They all live in one struct, so a change is a change of values.

## Capture geometry

A capture is exactly a pair of edges that form a collinear, evenly spaced
triple of nodes:

- the puma stands on `from`,
- a Trewa stands on `over`, which is a neighbour of `from`,
- the node `to` beyond it is empty, is a neighbour of `over`, and `over` is
  the exact midpoint of `from` and `to`.

The midpoint test covers horizontal, vertical, and diagonal lines at once,
because every edge on the canonical board spans exactly one unit. A triple
that bends (`7 -> 11 -> 10`) is not a jump.

The puma never jumps over an empty node, never lands on an occupied node,
and never clears two Trewa at once: a chain of jumps is a chain of single
jumps.

## Turn flow

- The puma moves first.
- After a jump, the puma keeps playing while another jump is available
  (`captureSequenceActive`). The turn only ends when no jump is left.
- A chain of jumps counts as one turn.
- `MoveEndTurn` exists only for variants where stopping a chain is allowed.

## Victory

Checked after every move, in this order:

1. the capture threshold (the puma wins),
2. the puma has no legal move (the dogs win),
3. the dogs have no legal move (a draw).

`Engine.Evaluate` applies the same checks to a position that was set up
outside the engine.

## What the random playouts show

300 games with random legal moves: every game ended, all of them with the
puma reaching six captures, and the average game ran about 73 moves. The
dogs never blocked the puma, because blocking needs the dogs to coordinate
and random play does not. A rules balance question, not an engine bug: the
dogs have to be played, not wandered.

## Still open

The rule values above are proposals. The user's answers are still to come
(see `issues/2026-10-21-komikan-rules-and-role-assignment-questions.md` and
`issues/2026-10-21-komikan-capture-geometry-uncertainty.md`).
