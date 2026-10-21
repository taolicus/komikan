# Board data and a renderer for review

- Date: 2026-10-21
- Area: game-rules
- Dependencies: issues/2026-10-21-komikan-board-topology-uncertainty.md

## Outcome

The Komikan board exists as data in Go: nodes, bidirectional edges, display
coordinates, and the initial piece placement. A rendering of that data can be
produced and eyeballed on the phone so the topology can be checked against a
reference before the rules engine is built on top of it.

## Scope

- New package `internal/board` with the types from `docs/board.md`:
  `NodeID`, `Node` (id, neighbours, coordinates), `Board`, `Piece`
  (`PANGUI`, `TREWA`), and `Position` (node id to piece; absent means empty).
- The standard layout as a data file in Go source, built from an explicit
  edge list (no node numbering or dimensions hardcoded in any consumer).
- Display coordinates for every node, used for rendering and for capture
  geometry, never for movement legality.
- Initial position: one Pangui and twelve Trewa, on distinct nodes.
- A validator that checks unique ids, symmetric adjacency, existing nodes, and
  a legal initial placement.
- A way to dump the board for review: a small `go run` command or a test that
  writes an ASCII/SVG rendering to a file.

## Acceptance criteria

- `go build ./...` and `go test ./...` succeed on this device.
- Every edge appears in both directions (test).
- Node ids are unique (test).
- The initial position contains exactly one Pangui and twelve Trewa, no two
  pieces share a node, and every piece sits on an existing node (test).
- Node and edge counts are stated in the task outcome and in
  `features/komikan-board-layout.md`.
- A rendering of the board can be produced and reviewed; the file or the
  rendered output is attached to the task outcome.
- The interpretation used to read the ASCII sketch is written down explicitly
  (which row is which, which nodes connect diagonally, where the shared
  triangle/square vertex is), so it can be corrected in one place.

## Verification

- `go test ./internal/board/...` on the device.
- Read the rendered board against the sketch in `docs/board.md` and, if
  available, against the reference discussed in the dependency issue.

## Notes and open points

- The dependency issue is unresolved, so this task ships a *provisional*
  layout. The rules engine consumes the board as data, so a later correction
  is a data change.
- The renderer is deliberately part of this task: it is the cheapest way for
  the board shape to be reviewed before it is baked into gameplay.
