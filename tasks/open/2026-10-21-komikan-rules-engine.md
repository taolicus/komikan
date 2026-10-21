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
