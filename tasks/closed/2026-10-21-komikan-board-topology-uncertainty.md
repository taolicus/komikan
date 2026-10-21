# Board topology is not pinned down yet

- Date: 2026-10-21
- Area: game-rules
- Resolved: 2026-10-21

## Resolution

Settled by the project owner on 2026-10-21:

- **Board:** the 38-point layout drawn in `docs/board.md` — a 5x5 Alquerque
  square (25 points) plus a 13-point triangular region above it, the
  triangle's bottom vertex shared with the square's top row.
- **Triangle lines:** the adjacency implied by the sketch, as written out
  exactly in `features/komikan-board-layout.md`.
- **Pangi start:** the middle of the second row, node 7.

The pinned definition — node table, full 79-edge list, initial placement, and
the checks it was verified against — now lives in
`features/komikan-board-layout.md`, which is the authoritative version of
`docs/board.md`.

Research that informed the decision:

- Wikipedia (en and es) and Board and Pieces confirm Komikan is played on an
  expanded Alquerque board with a triangular section, and that several
  triangular variants exist, so the project pins one board rather than
  claiming a canonical one.
- Adrian Cheuquepan's account (cited by Wikipedia) confirms the rules in
  `docs/overview.md` and places the Pangi in the triangle and the Trewa in
  the square, which the adopted layout matches.
- Chilean sources describe a board of 38 puntos; the adopted layout has
  exactly 38 nodes.

The interpretation was verified before being written down: 38 nodes, 79
bidirectional edges, connected, no isolated nodes, every edge one unit long,
and a render that reproduces the sketch in `docs/board.md` exactly.

## Problem as found

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

## Where it looked ambiguous (as first written)

Preserved as written. Point 1 was the weakest part of this analysis: the
40-point hypothesis rested on an unverified recollection and was weakened once
Chilean sources describing a board of *38 puntos* turned up.

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

## What I asked (answered as option b)

- **(a) Supply a reference.** A photo, scan, or link to a board you trust
  (community or published account). I transcribe it into the node/edge list
  and record the source in `features/`.
- **(b) Approve the sketch as the project board.** We treat `docs/board.md`
  as canonical for Komikan even if it differs from the traditional board.
- **(c) Approve the classic 40-point layout** (triangle of 1/2/3/4/5 sharing
  its base row with the 5x5 square) and I implement that, treating the
  sketch as a drafting error.

**Answered as (b)**, with the sketch's reading as the source of the exact
adjacency.

## Interim handling

`tasks/open/2026-10-21-komikan-board-data-and-renderer.md` proceeds with the
square region as drawn and a documented provisional top region, and renders
the result so the topology can be eyeballed. Because the rules engine only
consumes board data, revising the layout later is a data change rather than
an engine rewrite.

## Next step

Implemented as a transcription of `features/komikan-board-layout.md` by
`tasks/open/2026-10-21-komikan-board-data-and-renderer.md`, whose renderer
must reproduce the sketch in `docs/board.md`.
