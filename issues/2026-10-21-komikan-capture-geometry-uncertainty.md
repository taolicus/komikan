# Capture jump geometry and mandatory-capture rules need decisions

- Date: 2026-10-21
- Area: game-rules

## Problem

`docs/overview.md` defines captures in prose ("a valid destination node
exists beyond that Trewa along the same straight board line") but the board is
stored as a graph, and the doc explicitly warns that two consecutive edges do
not necessarily form a legal jump. The rules engine needs a concrete,
implementable definition, plus answers to a few related gaps.

**Update 2026-10-21:** the board is now settled
(`features/komikan-board-layout.md`) and every edge spans exactly one unit
horizontally, vertically, or diagonally. The board is a unit lattice, so
question 1 below is no longer a judgement call between a geometry-derived rule
and an explicit pattern list: a capture is exactly a pair of edges forming a
collinear, evenly spaced triple of nodes. No separate capture-pattern table is
needed. The remaining questions still stand.

## Open questions

1. **How is "straight line" determined?**
   - (a) *Geometry-derived:* use the per-node display coordinates; a jump
     `from -> over -> to` is legal when `from`, `over`, `to` are collinear,
     evenly spaced, and both edges exist in the graph. Works for horizontal,
     vertical, and diagonal lines without extra data.
   - (b) *Explicit pattern list:* the board file enumerates every legal
     (from, over, to) triple. Precise, but long and needs maintenance
     whenever the layout changes.
   - (c) *Hybrid:* derive candidates by geometry, then filter against an
     explicit allow/deny list in the board data.

   Proposed default: **(a)**. It is simple, and the board data task already
   produces coordinates for rendering. Coordinates stay display data for
   topology purposes but are used only for capture geometry.

2. **Can a jump clear two Trewa in a line and land beyond them?**
   Some accounts of the game describe a double jump. `docs/overview.md`
   assumes one jump captures exactly one Trewa.

   **Answered on 2026-10-22: no.** A jump takes exactly one Trewa; a chain
   of jumps is a chain of single jumps.

3. **Must the Pangui capture when a capture is available?**
   The doc makes *continuation* after a capture mandatory, but does not say the
   Pangui may not prefer a normal move when a capture is available at the start
   of its turn. Traditional accounts of the game say the puma must capture
   when it can.

   Proposed default: **yes** — if a capture is available at the start of the
   Pangui's turn, only captures are legal for that turn. This is a bigger
   behavioural difference than it looks, so please confirm.

   **Answered on 2026-10-22: no.** The puma is not made to eat. A capture
   on offer does not make the normal moves illegal, and the player chooses.
   `rules.DefaultRuleset()` has `CapturePreferred: false`.

   Chain continuation was a separate question and is also answered: a chain
   of jumps **may be stopped** at any point (`MultipleCapturesMandatory:
   false`). Once the puma has jumped it keeps the turn, but it may end the
   turn instead of jumping again. The page offers a "Terminar la captura"
   button while a chain is running.

4. **Does a capture require the jumped Trewa specifically?**
   Obviously it does in this game (the Trewa never captures), but the engine
   should still reject jumping over an empty node or over the Pangui itself.

   Proposed default: reject, with a distinct error reason for each case.

5. **Which terminal conditions exist besides the two documented ones?**
   - Pangui wins at the capture threshold (now 8, raised from 6 on
     2026-10-22).
   - Pu Trewa win when the Pangui has no legal normal move and no capture.
   - Undefined in the docs: what happens when the Trewa player has no legal
     move at all? A draw? A Pangui win? Continuing play?

   **Answered on 2026-10-22: a draw.** Draw when the Trewa player has no
   legal move and the Pangui has not reached its threshold. `DRAW` is a game
   status.

6. **Order of victory checks.**
   If the Pangui's last capture both reaches the threshold and leaves it
   blocked, which result wins?

   Proposed default: **threshold first** (the capture that wins the game is
   not cancelled by the resulting position).

## Next step

Confirm or override the proposed defaults. The rules engine task implements
whatever is decided here; the values live in the `Ruleset` struct so a variant
can differ without engine changes.

**Update 2026-10-21:** the engine is written and shipped with the proposed
defaults, so the game is playable.

**Update 2026-10-22: closed.** Every question has an answer, and the code
matches it.

How each question landed in the code:

| Question | Answer | Where it lives |
|---|---|---|
| 1. straight line | (a) geometry | `internal/rules`: the node beyond the Trewa must be empty, and the Trewa must be the exact midpoint of the two nodes. One test for every direction, so no pattern table. |
| 2. double jump | no | A jump always takes exactly one Trewa; a chain of jumps is a chain of single jumps. |
| 3. must capture | **no** | `Ruleset.CapturePreferred` is off: a capture on offer does not make the normal moves illegal, so the puma may decline it. |
| 3b. must continue a chain | **no** | `Ruleset.MultipleCapturesMandatory` is off: while a chain runs, `MoveEndTurn` is offered and ends the turn. |
| 4. only a Trewa may be jumped | yes | The engine only jumps over a node holding a Trewa; an empty node or the Pangui is not a capture. |
| 5. the Trewa are stuck | draw | `Ruleset.DrawWhenTrewaCannotMove`, and `Status` has a `DRAW` value. |
| 6. order of the victory checks | threshold first | `Ruleset.ThresholdBeforeBlocking`. |
