# Komikan board layout

Authoritative definition of the MVP board: **38 nodes, 79 undirected edges**,
initial placement of one Pangui and twelve Trewa.

The rules engine, the renderer, and the tests all read the board from this
definition. Nothing outside `internal/board` may assume node ids, node counts,
or board dimensions.

## Decision and provenance

Settled on 2026-10-21 by the project owner.

Komikan is played on an *expanded Alquerque* board: an Alquerque square with a
triangular patterned section attached along one side. There is no single
canonical version — the triangular section has several published variants
(Culin 1898 "Solitario", Matus 1920, Alcina Franch 1980) — so the project
picks one specific board and implements it exactly.

The board adopted here is the 38-point layout sketched in `docs/board.md`:

- a 5x5 Alquerque square of 25 points with the alternating diagonal lattice,
- a triangular region of 13 points above it,
- the triangle's bottom vertex shared with the square's top row.

Supporting facts:

- Chilean sources describe a Komikan board of **38 puntos**; this layout has
  exactly 38 nodes.
- Adrián Cheuquepan's account (the source cited by Wikipedia) places the
  Pangi in the triangle ("la montaña") and the Trewa in the square ("el
  valle"), which this layout's initial position matches.
- The rules in `docs/overview.md` agree with the same source: Pangi moves
  first, both sides move one segment along the board lines in any direction,
  only the Pangi captures (by jumping over a Trewa), consecutive captures are
  allowed, and the win conditions are six captures or immobilising the Pangi.

`docs/board.md` remains the sketch this was read from. This file is the
authoritative version of it.

## Coordinate convention

`x` is the column, 0 (left) to 4 (right). `y` is the row, 0 at the top of the
board and 7 at the bottom, matching the sketch top to bottom. Adjacent nodes
are always one unit apart, so a unit grid preserves every drawn length.

```
    o --- o --- o --- o --- o        y=0   ids 0..4
    |     |     |     |     |
    o --- o --- x --- o --- o        y=1   ids 5..9    (x at id 7 = Pangui)
       \  |     |     |  /
          o --- o --- o             y=2   ids 10..12  (columns 1..3)
             \  |  /
    o --- o --- o --- o --- o        y=3   ids 13..17  (square top row)
    |  \  |  /  |  \  |  /  |
    o --- o --- o --- o --- o        y=4   ids 18..22
    |  /  |  \  |  /  |  \  |
    x --- o --- o --- o --- x        y=5   ids 23..27
    |  \  |  /  |  \  |  /  |
    x --- x --- x --- x --- x        y=6   ids 28..32
    |  /  |  \  |  /  |  \  |
    x --- x --- x --- x --- x        y=7   ids 33..37
```

## Nodes

| Row | y | Columns | Node ids | Notes |
|-----|---|---------|----------|-------|
| 1 | 0 | 0..4 | 0..4 | top row of the triangle region |
| 2 | 1 | 0..4 | 5..9 | Pangi starts at id 7 |
| 3 | 2 | 1..3 | 10..12 | narrowing to the shared vertex |
| 4 | 3 | 0..4 | 13..17 | square top row; id 15 is the shared vertex |
| 5 | 4 | 0..4 | 18..22 | |
| 6 | 5 | 0..4 | 23..27 | Trewa at ids 23 and 27 |
| 7 | 6 | 0..4 | 28..32 | all Trewa |
| 8 | 7 | 0..4 | 33..37 | all Trewa |

Node ids are assigned row by row from the top, left to right. They are stable
for the life of the project: they are stored in persisted game states, so
renumbering them is a breaking change.

## Edges

79 edges. Every edge is bidirectional; the lists below name each edge once.

### Horizontal (30)

```
0-1, 1-2, 2-3, 3-4,
5-6, 6-7, 7-8, 8-9,
10-11, 11-12,
13-14, 14-15, 15-16, 16-17,
18-19, 19-20, 20-21, 21-22,
23-24, 24-25, 25-26, 26-27,
28-29, 29-30, 30-31, 31-32,
33-34, 34-35, 35-36, 36-37
```

### Vertical (29)

```
0-5, 1-6, 2-7, 3-8, 4-9,
6-10, 7-11, 8-12,
11-15,
13-18, 14-19, 15-20, 16-21, 17-22,
18-23, 19-24, 20-25, 21-26, 22-27,
23-28, 24-29, 25-30, 26-31, 27-32,
28-33, 29-34, 30-35, 31-36, 32-37
```

### Diagonal (20)

```
5-10, 9-12,
10-15, 12-15,
13-19, 15-19, 15-21, 17-21,
19-23, 19-25, 21-25, 21-27,
23-29, 25-29, 25-31, 27-31,
29-33, 29-35, 31-35, 31-37
```

The square's diagonal lattice alternates by row band, which is what gives the
Alquerque its diamond pattern: the band between rows 3 and 4 runs
`13-19, 15-19, 15-21, 17-21`, the next band runs the other way, and so on.

## Initial position

- **Pangui:** node 7 — the middle of the second row, inside the triangle.
- **Trewa:** nodes 23, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37 — the two
  outer nodes of the square's third row from the bottom, plus the bottom two
  rows in full.
- All other nodes are empty.

## Consequences for the rules engine

- Every edge spans exactly one unit horizontally, vertically, or diagonally.
  The board is a unit lattice, so the "same straight line" condition for a
  capture is exactly: `from`, `over`, and `to` are collinear, evenly spaced,
  and both `from-over` and `over-to` are edges. No separate capture-pattern
  table is needed.
- Node 15, the shared vertex, has degree 8: four neighbours from the triangle
  (10, 11, 12) and five from the square (14, 16, 19, 20, 21). It is the
  busiest junction on the board and the natural place for a Trewa to shut down
  the route between the two regions.
- The Pangui's start node 7 has degree 4 (neighbours 2, 6, 8, 11).

## Verification

The interpretation was checked programmatically before being written down:

- 38 nodes, 79 edges, every edge present in both directions.
- The graph is connected; every node is reachable from node 0.
- No isolated nodes; degrees range from 2 (the top corners, ids 0 and 4) to 8
  (ids 15, 19, 21, 25, 29, 31).
- Every edge spans one unit horizontally, vertically, or diagonally.
- Rendering the graph back to ASCII reproduces the sketch in `docs/board.md`
  exactly, including every connector row.

`internal/board` re-establishes these properties with its own tests, so the
checks survive independently of this document.

## Implementation

- `internal/board` holds the transcription. `board.Standard()` returns the
  board and the starting position; `board.Validate(board, position)` checks
  node uniqueness, symmetric adjacency, connectivity, and a legal placement.
- `cmd/boarddump` prints the ASCII sketch and can write an SVG rendering:

  ```
  go run ./cmd/boarddump
  go run ./cmd/boarddump -svg docs/board.svg
  ```

  `docs/board.svg` is the committed SVG rendering of the starting position.
  Regenerate it whenever the layout changes.

- The coming-soon page renders the board inline from this data at request
  time, so the board on the site is always the real board. The SVG renderer
  returns an SVG element without an XML declaration, so it can be embedded in
  HTML; `cmd/boarddump` adds the declaration when writing a standalone file.

## Rendering

The site is dark, and the board is drawn **on the page background**: the SVG
has no background rectangle, and the page gives it no panel, no border, and
no shadow. The board sits on the page as if it were part of it.

The colours are chosen for that dark background:

| Part | Colour | Note |
|---|---|---|
| board lines | `#6f6656` | warm grey, light enough to read on `#0c0b09` |
| empty node | `#332e26` fill, `#6f6656` stroke | a dim dot, not a white one |
| Pangui disc | `#14532d` | deep green, no border |
| Trewa disc | `#7f1d1d` | deep red, no border |
| last move | `#c9a227` | gold ring |
| legal destination | `#3b82f6` | blue ring and dot |
| capture destination | `#eab308` | amber ring and dot |
| selected piece | `#60a5fa` | light blue ring |

There are two renderers and they use the same palette:

- `internal/board.RenderSVG` writes the colours as inline attributes, because
  its output has to stand on its own as an SVG file. It feeds the coming-soon
  page, `GET /api/rooms/{roomID}/board`, and `cmd/boarddump`.
- the game page draws the board in the browser from the board graph the API
  sends, and takes its colours from CSS classes in `web/rooms.html`.

There is no theme system: the dark colours are the colours.

## Caveats

- This is one specific variant, chosen because it is documented and internally
  consistent, not because it is proven to be the oldest or most widespread.
  Other triangular variants exist and could be added later as additional board
  layouts, since the rules engine consumes the board as data.
- The 38-point count and the Pangi-in-triangle placement are attested, but the
  exact triangle line pattern was read off the sketch in `docs/board.md`. If a
  physical or published board turns out to differ, only `internal/board`
  changes.
