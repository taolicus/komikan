# Board topology is not pinned down yet

- Date: 2026-10-21
- Area: game-rules

## Problem

`docs/board.md` defines the board as an ASCII sketch. The sketch is not precise
enough to derive a node/edge list without interpretation, and the shape it
suggests does not obviously match the traditional Komikan board. Every other
part of the MVP (rules engine, renderer, tests) consumes this data, so it is
the first thing worth settling.

## What the current sketch says

Reading `docs/board.md` literally, the sketch has eight rows of nodes:

| row | nodes | contents in the sketch |
|-----|-------|------------------------|
| 1 | 5 | all empty |
| 2 | 5 | one `x` in the middle (presumably the Pangui) |
| 3 | 3 | all empty |
| 4 | 5 | all empty |
| 5 | 5 | all empty |
| 6 | 5 | `x` at both ends |
| 7 | 5 | all `x` |
| 8 | 5 | all `x` |

That is 38 nodes and 13 occupied starting nodes (1 Pangui + 12 Trewa), so the
piece counts are internally consistent.

## Where it is ambiguous or probably wrong

1. **The top region is not a triangle.** Rows of 5, 5, and 3 nodes do not form
   a triangle. The traditional board is usually drawn as a triangle of 1, 2,
   3, 4, 5 points whose base row is the top row of a 5x5 square, giving 40
   points in total. I could not verify this from an external source while
   writing this issue (web search was unavailable), so treat it as a strong
   suspicion rather than as fact.
2. **The "shared bottom vertex" is only a vertex in the sketch.** If the
   traditional base *row* is shared, the top region and the square share five
   nodes, not one. The sketch shows a single shared vertex plus an extra
   5-node row above the square, which is a different board.
3. **Pangui start node.** The sketch puts the Pangui in the middle of the
   second row. In the traditional account the puma starts on the apex of the
   triangle. If the shape is wrong, this start node is wrong too.
4. **Column alignment of row 3.** The three-node row is indented in the
   sketch, but the exact columns are not marked, so which row-2 nodes connect
   to which row-3 nodes is a guess.
5. **Diagonal connectors in the top region** are loose `\`, `|`, `/`
   characters; the resulting adjacency set is not uniquely determined.
6. **The square region** (rows 4-8, five wide, X-pattern diagonals) is
   unambiguous enough to adopt as is, and matches the traditional drawing
   style.

## What I need from you

Pick one:

- **(a) Supply a reference.** A photo, scan, or link to a board you trust
  (community or published account). I transcribe it into the node/edge list
  and record the source in `features/`.
- **(b) Approve the sketch as the project board.** We treat `docs/board.md`
  as canonical for Komikan even if it differs from the traditional board.
- **(c) Approve the classic 40-point layout** (triangle of 1/2/3/4/5 sharing
  its base row with the 5x5 square) and I implement that, treating the
  sketch as a drafting error.

## Proposed default if you have no preference

Option (c), with the sketch's piece placement adapted: Pangui on the triangle
apex, Trewa on the bottom two rows of the square plus the two outer nodes of
the row above them (12 nodes, which matches the sketch's occupied count).

## Interim handling

`tasks/open/2026-10-21-komikan-board-data-and-renderer.md` proceeds with the
square region as drawn and a documented provisional top region, and renders
the result so the topology can be eyeballed. Because the rules engine only
consumes board data, revising the layout later is a data change rather than
an engine rewrite.

## Next step

Answer (a), (b), or (c). Until then, the board data task records its
interpretation explicitly so it can be corrected in one place.
