# Board data and a renderer for review

- Date: 2026-10-21
- Area: game-rules
- Dependencies: tasks/closed/2026-10-21-komikan-board-topology-uncertainty.md

## Outcome

The Komikan board exists as data in Go: nodes, bidirectional edges, display
coordinates, and the initial piece placement. A rendering of that data can be
produced and eyeballed on the phone so the topology can be checked against the
sketch it was derived from.

## Scope

- New package `internal/board` with the types from `docs/board.md`:
  `NodeID`, `Node` (id, neighbours, coordinates), `Board`, `Piece`
  (`PANGUI`, `TREWA`), and `Position` (node id to piece; absent means empty).
- The standard layout transcribed from `features/komikan-board-layout.md`:
  38 nodes, 79 edges, node ids 0..37, the shared vertex at node 15, the
  Pangui on node 7, and the twelve Trewa on nodes 23, 27..37.
- Display coordinates for every node, used for rendering and for capture
  geometry, never for movement legality.
- A validator that checks unique ids, symmetric adjacency, existing nodes, and
  a legal initial placement.
- A way to dump the board for review: a small `go run` command or a test that
  writes an ASCII/SVG rendering to a file.

## Acceptance criteria

- `go build ./...` and `go test ./...` succeed on this device.
- Every edge appears in both directions (test).
- Node ids are unique (test).
- The board has exactly 38 nodes and 79 edges (test).
- The graph is connected and has no isolated nodes (test).
- The initial position contains exactly one Pangui and twelve Trewa, no two
  pieces share a node, and every piece sits on an existing node (test).
- The Pangui is on node 7 and the Trewa on nodes 23, 27..37 (test).
- The renderer's ASCII output reproduces the sketch in `docs/board.md`
  exactly, including the connector rows.
- Node and edge counts are stated in the task outcome and already recorded in
  `features/komikan-board-layout.md`.

## Verification

- `go test ./internal/board/...` on the device.
- Compare the renderer's output against the sketch in `docs/board.md` and the
  diagram in `features/komikan-board-layout.md`.

## Notes and open points

- The topology is settled: this task is a transcription of
  `features/komikan-board-layout.md`, not an interpretation. The decision and
  its provenance are recorded in the closed dependency.
- The renderer is deliberately part of this task: it is the cheapest way to
  confirm that the transcription matches the sketch it came from.
