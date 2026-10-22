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
| `panguiCapturesToWin` | 8 | raised from 6 on 2026-10-22 to give the dogs room |
| `allowMultipleCaptures` | true | the puma may chain jumps in one turn |
| `multipleCapturesMandatory` | false | a chain may be ended at any point (answered 2026-10-22) |
| `trewaCanMoveBackward` | true | the dogs may move along any edge |
| `panguiMovesFirst` | true | |
| `puTrewaWinByBlocking` | true | the puma with no legal move loses |
| `puTrewaWinByOccupyingTriangle` | false | the canonical board has no triangle area |
| `capturePreferred` | false | the puma is never made to eat: it may move normally instead |
| `drawWhenTrewaCannotMove` | true | the dogs with no legal move end the game in a draw |
| `thresholdBeforeBlocking` | true | the capture that wins the game is not cancelled by the position it leaves |

Every value is now the answer of the owner of the game or a property of the
board; none of them is a provisional proposal. They all live in one struct,
so a change is a change of values.

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
- The puma is not made to hunt. A capture on offer does not make the normal
  moves illegal: the player chooses between them (`capturePreferred` is
  off).
- After a jump, the puma may keep jumping while another jump is available,
  but it is not obliged to (`multipleCapturesMandatory` is off). A capture
  chain that is ended counts as one turn, like one that is played out.
- `MoveEndTurn` is offered exactly while a capture chain is running. It has
  `From == To` (the puma stays where it is) and the API accepts it like any
  other move. The page shows it as a button rather than as a point of the
  board, because the puma's own point is also how a move is picked.
- While a chain runs, `captureSequenceActive` is true and the turn belongs to
  the puma; it ends either by continuing or by `MoveEndTurn`.

## Victory

Checked after every move, in this order:

1. the capture threshold (the puma wins),
2. the puma has no legal move (the dogs win),
3. the dogs have no legal move (a draw).

`Engine.Evaluate` applies the same checks to a position that was set up
outside the engine.

## What the random playouts show

200 games with random legal moves under the current defaults (threshold 8,
chains optional, the puma free to decline a capture): every game ended, the
average game ran about 220 moves, and the longest ran 618. The puma won all
200.

The numbers under the older defaults, for comparison:

| Defaults | Average | Longest | Puma wins |
|---|---|---|---|
| threshold 6, capture mandatory, chain mandatory | ~73 | — | 300/300 |
| threshold 6, capture optional, chain mandatory | ~153 | 552 | 199/200 |
| threshold 8, capture optional, chain optional | ~220 | 618 | 200/200 |

The single dog win at threshold 6 was noise: changing the rules changes the
whole random trajectory, so one game in two hundred proves nothing. Random
dogs never coordinate, and blocking the puma is exactly what needs
coordination. Playouts can show that a rule set terminates and roughly how
long a game runs; they cannot say whether the game is balanced. Only play
between people can answer that, so the threshold is a design choice (the
puma now has to catch eight of the twelve), not a measured result.

## Decisions

Every rule question has been answered by the owner of the game on
2026-10-22; this document is the record of the answers and the values in
`rules.DefaultRuleset()` are the implementation. What remains is
play-testing, not rule-setting.
