# Rules engine: moves, captures, turns, victory

- Date: 2026-10-21
- Area: game-rules
- Dependencies: tasks/open/2026-10-21-komikan-board-data-and-renderer.md, issues/2026-10-21-komikan-capture-geometry-uncertainty.md, issues/2026-10-21-komikan-rules-and-role-assignment-questions.md

## Outcome

A `internal/rules` package implements the game independently of HTTP and the
browser: legal move generation, capture detection, turn handling, and victory
detection, all driven by a configurable `Ruleset`.

## Scope

- `Ruleset` struct with the configurable values from `docs/overview.md` §12
  and the defaults confirmed in the dependency issues.
- `GameState` with position, current player, captured count, turn number,
  capture-sequence state, status, and move history.
- `Move` type carrying player, piece, from, to, move type, and captured node.
- `LegalMoves(state, player)` — normal moves to adjacent empty nodes, plus
  captures, filtered by the ruleset (mandatory capture preference, mandatory
  continuation, backward movement for the Trewa).
- `ApplyMove(state, move) (GameState, error)` — validates, applies, updates
  counters and capture-sequence state, checks victory, switches player only
  when the turn is complete, and never mutates the previous state.
- Capture detection per the decision in the capture geometry issue
  (provisional: geometry-derived from board coordinates).
- Victory conditions: Pangui capture threshold, Pangui blocked, and the
  agreed handling of a draw.
- No dependency on `net/http`, `embed`, or any web asset.

## Acceptance criteria

- The rules engine package imports nothing from the HTTP or UI layers.
- Tests exist for the cases listed in `docs/overview.md` §14. Where a case is
  not applicable under the confirmed ruleset, the test file records why
  rather than dropping it silently.
- An illegal move returns an error and leaves the state byte-for-byte
  unchanged (deep comparison in the test).
- Every move returned by `LegalMoves` can be applied successfully.
- A random-playout test runs many games to completion without panics,
  infinite loops, or illegal states.
- Capture chains update the position and the captured counter correctly, and
  the turn does not switch while a mandatory continuation is available.
- A game ends with a terminal status, and further moves are rejected after
  that.
- `go test ./internal/rules/...` passes on this device.

## Verification

- `go test ./internal/rules/...` on the device.
- `go test -race` is optional: the race detector may not be available for
  `android/arm64`, so treat it as a laptop-only extra rather than a
  requirement here.

## Notes and open points

- The rules engine starts from the defaults in the dependency issues. If the
  answers change, the change should land in the `Ruleset` defaults or in the
  board data, not in scattered conditionals.
- `features/` should record the confirmed rule set once the answers land.

## Completion notes

- **Status: completed on 2026-10-21.**
- `internal/rules` imports `internal/board` and nothing else: no HTTP, no
  embed, no web assets.
- The engine holds the board and the ruleset; the state stays a plain
  document that can be stored and sent to a browser. The suggested
  signatures are kept: `LegalMoves(state, player)` and
  `ApplyMove(state, move)`.
- Every rule that varies between variants is a field of `Ruleset`
  (`rules.DefaultRuleset()` holds the defaults proposed in the dependency
  issues). The engine has no scattered conditionals for a variant.
- The capture is a collinear, evenly spaced triple: the node beyond the
  Trewa must be empty and the Trewa must be the exact midpoint. The
  midpoint test covers every direction at once, so no pattern table is
  needed.
- `ApplyMove` never modifies the state it receives: it clones, applies, and
  returns the new state. An illegal move returns an error and the original
  state, byte for byte.
- `Engine.Evaluate` applies the victory checks to a position that was set
  up outside the engine, which is how a blocked puma is recognised.
- Tests cover the cases in `docs/overview.md` §14: the initial position,
  normal moves, occupied destinations, jumps that bend, jumps onto occupied
  nodes, chains of captures, the capture threshold, the blocking win, the
  unchanged state after an illegal move, no moves after the end of the
  game, and every legal move being applicable. The triangle-occupation win
  is recorded as not applicable: the canonical board has no triangle area,
  so the rule stays off.
- A random playout test runs 200 games to a terminal state with no panic,
  no infinite loop and no illegal move.
- The rules are written up in `features/komikan-rules.md`.
