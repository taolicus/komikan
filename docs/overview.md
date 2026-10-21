# Komikan — Game Rules and Implementation Specification

## 1. Overview

Komikan is a traditional Mapuche abstract strategy board game for two players. One player controls a Pangui (puma); the other controls twelve Pu Trewa (dogs).

The Pangui hunts and captures Trewa by jumping over them. The Pu Trewa cooperate to surround the Pangui and prevent it from moving.

Implement the game as a graph-based board game: positions are nodes, legal movement paths are edges, and pieces occupy nodes.

Komikan has regional and modern rule variants. This specification defines one playable variant. Keep variant-dependent rules configurable rather than assuming this is the only historical ruleset.

## 2. Players and Pieces

- Pangui player: controls exactly one Pangui piece.
- Pu Trewa player: controls twelve Trewa pieces.
- Each board node can contain at most one piece.
- Empty nodes may be occupied by a legal move.

Represent pieces using an enum or equivalent:

- `PANGUI`
- `TREWA`

Represent empty positions through the position data structure rather than treating `EMPTY` as a piece, unless the implementation consistently chooses explicit occupancy values.

Keep piece identity separate from node identity. Nodes represent positions; pieces represent entities occupying those positions.

## 3. Board Representation

Represent the board as an undirected graph.

- `Node`: A valid position where a piece can stand.
- `Edge`: A connection between two nodes along which a piece may move.
- `Board`: The complete set of nodes and connections.
- `Position`: The current assignment of pieces to nodes.
- `InitialPosition`: The starting assignment of pieces to nodes.

The board layout, including node identifiers and connections, is defined in a separate board specification file. Load this data rather than duplicating the board definition in the rules engine.

Requirements:

- Every node must have a unique, stable identifier.
- Every edge must be bidirectional.
- The board must explicitly define all valid connections, including diagonals.
- Shared intersections must be represented by one node, not duplicated.
- Movement must follow the graph; do not infer legal movement from geometric proximity.
- The board layout must provide a complete, valid initial position containing one Pangui and twelve Trewa.
- Keep board topology, initial position, current position, and game rules separate.

Do not hardcode node numbering or board dimensions into the movement engine.

## 4. Initial Setup

1. Load the selected board layout and its initial-position data.
2. Place one Pangui on its designated starting node.
3. Place twelve Trewa on their designated starting nodes.
4. Set the current player according to the selected ruleset.
5. Initialize the captured-Trewa counter to zero.
6. Validate that every starting node exists and no two pieces occupy the same node.
7. Reject invalid configurations before starting the game.

The board specification is responsible for defining the topology and initial placement. The rules engine must validate and consume that data without embedding board-specific assumptions.

## 5. Turn Order

Players alternate turns.

Default configuration:

1. The Pangui moves first.
2. The Pu Trewa player makes one move.
3. The Pangui player makes one move.
4. Continue alternating until a victory condition is reached.

A normal move changes the position of one piece. A capture sequence, if enabled, may involve several Pangui jumps during the same turn.

A multiple-capture sequence counts as one turn, not multiple alternating turns.

## 6. Normal Movement

### Pangui

On its turn, the Pangui may move from its current node to any directly connected, empty node.

A normal move:

- Must follow an edge in the board graph.
- Must end at an empty node.
- Must not pass through another node or piece.
- Must not capture a Trewa unless the move is a valid capture jump.

### Pu Trewa

On their turn, the Pu Trewa player selects one Trewa and moves it to a directly connected, empty node.

A normal move:

- Must follow an edge in the board graph.
- Must end at an empty node.
- Moves exactly one Trewa.
- Does not capture the Pangui.
- Does not jump over other pieces.

In this variant, Trewa may move in either direction along valid edges. If a historical or regional variant restricts backward movement, implement that restriction as a configurable rule.

## 7. Pangui Captures

The Pangui captures Trewa by jumping over them.

A capture is legal when all of the following conditions hold:

1. A Trewa occupies a node directly connected to the Pangui.
2. A valid destination node exists beyond that Trewa along the same straight board line.
3. The destination node is empty.
4. The board layout permits the jump through its geometry or an explicitly defined capture pattern.

Do not assume that every pair of consecutive edges constitutes a valid straight-line jump. The existence of two graph edges alone does not necessarily establish that the nodes form a valid capture pattern.

Represent capture patterns explicitly when they cannot be derived reliably from the board layout. A pattern should identify the origin node, intermediate node, destination node, and any applicable restrictions.

When a capture occurs:

1. Move the Pangui to the destination node.
2. Remove the jumped Trewa from the position.
3. Increment the captured-Trewa counter.
4. Update the game state.
5. Check victory conditions.

Reject captures when the destination is occupied, nonexistent, or invalid for the selected board layout.

### Multiple Captures

The default variant allows consecutive captures during one Pangui turn.

After a capture:

1. Recalculate all legal captures from the Pangui's new position.
2. If another capture is available, apply the selected continuation rule.
3. Each jump removes exactly one Trewa.
4. End the turn when no further capture is available, or when stopping is permitted and the player chooses to stop.

For a deterministic default implementation, make additional captures mandatory whenever available. This behavior must remain configurable because historical rules vary.

A multiple-capture sequence counts as one turn. Do not switch players while a mandatory capture remains available.

## 8. Pu Trewa Strategy and Restrictions

The Pu Trewa cannot capture the Pangui.

The Pu Trewa player wins by coordinating its pieces to eliminate every legal Pangui move. Trewa may block the Pangui's available routes, provided every move follows the board graph and ends at an empty node.

Do not implement a special capture action for Trewa.

Unless the selected variant explicitly requires otherwise:

- Trewa cannot jump over pieces.
- Trewa cannot occupy the Pangui's node.
- Trewa cannot move to nonexistent nodes.
- Trewa cannot move onto occupied nodes.

## 9. Victory Conditions

The game ends immediately when a configured victory condition is satisfied.

### Pangui Victory

The default configuration declares the Pangui the winner after capturing six Trewa.

Make the capture threshold configurable because published versions use different thresholds.

### Pu Trewa Victory

The Pu Trewa win when the Pangui has no legal moves.

A legal Pangui move includes either:

- A normal move to an adjacent empty node.
- A valid capture jump over a Trewa to an empty destination node.

The Pangui loses if neither type of move is available.

Some variants allow the Pu Trewa to win by occupying every designated node in the triangular area. This condition should be optional and enabled only when supported by the selected board layout and ruleset.

### Terminal State

Once the game is won:

- Reject all additional moves.
- Record the winning player.
- Preserve the final board position for review or replay.

## 10. Legal Move Generation

Implement a central move-generation engine.

Suggested interface:

`getLegalMoves(gameState, player) -> Move[]`

### Pu Trewa

1. Find every Trewa controlled by the player.
2. Inspect each adjacent node.
3. Add a normal move for every adjacent empty node that satisfies the selected ruleset.

### Pangui

1. Find the Pangui's current node.
2. Inspect adjacent nodes.
3. Add a normal move for each adjacent empty node.
4. Inspect all valid capture patterns.
5. Add a capture when the intermediate node contains a Trewa and the destination is empty.
6. Apply capture-continuation rules when resolving a multiple-capture turn.

Suggested move representation:

```json
{
  "player": "PANGUI",
  "pieceId": "pangui",
  "from": 4,
  "to": 12,
  "type": "CAPTURE",
  "capturedNode": 8
}
```

The node identifiers are illustrative only. Actual identifiers must come from the loaded board layout.

## 11. State Transitions

Implement a single authoritative state-transition function:

`applyMove(gameState, move) -> newGameState`

It must:

1. Validate the move against the current state.
2. Move the piece.
3. Remove captured Trewa, if any.
4. Update counters and capture-sequence state.
5. Determine whether the current turn is complete.
6. Check victory conditions.
7. Switch players only when the turn is complete.
8. Return the updated state without mutating historical states if undo or replay is supported.

Illegal moves must not modify the game state.

## 12. Recommended Game State

A game-state object should include:

- `boardLayoutId`
- `position`
- `currentPlayer`
- `capturedTrewa`
- `turnNumber`
- `captureSequenceActive`
- `mandatoryCaptureContinuation`
- `gameStatus` (`IN_PROGRESS`, `PANGUI_WON`, `PU_TREWA_WON`)
- `ruleset`
- `moveHistory`

The `position` maps node IDs to pieces. Empty nodes may be represented by absent map entries or explicit `EMPTY` values, provided the implementation uses one convention consistently.

The ruleset should include:

- `panguiCapturesToWin`: default `6`
- `allowMultipleCaptures`: default `true`
- `multipleCapturesMandatory`: default `true`
- `trewaCanMoveBackward`: default `true`
- `panguiMovesFirst`: default `true`
- `puTrewaWinByBlocking`: default `true`
- `puTrewaWinByOccupyingTriangle`: default `false`

Validate that all configured values are compatible with the loaded board layout.

## 13. User Interface Requirements

The interface should:

- Render nodes and connections from the loaded board graph.
- Display the Pangui and Trewa as visually distinct pieces.
- Indicate the current player.
- Highlight selectable pieces.
- Highlight legal destination nodes after a piece is selected.
- Distinguish capture moves from normal moves.
- Display the number of captured Trewa and the configured capture threshold.
- Prevent interaction with occupied or invalid destinations.
- Display the winner when the game ends.
- Support restarting a game with the same or a different ruleset.

The visual board must accurately represent the graph used by the rules engine. Every displayed connection must correspond to an edge in the graph, and every graph edge must be represented visually.

## 14. Validation and Tests

At minimum, test the following cases:

1. The loaded board contains unique node identifiers.
2. Every edge is bidirectional.
3. Shared intersections are represented by a single node.
4. The initial position contains exactly one Pangui and twelve Trewa.
5. No two pieces occupy the same node.
6. The Pangui can move to an adjacent empty node.
7. The Pangui cannot move to an occupied node.
8. A Trewa can move to an adjacent empty node.
9. Trewa cannot jump over pieces.
10. A valid Pangui jump removes exactly one Trewa.
11. A Pangui jump is rejected if the landing node is occupied.
12. A Pangui jump is rejected if the required connections or capture pattern do not exist.
13. Consecutive captures update the position and counter correctly.
14. A mandatory capture sequence does not prematurely switch players.
15. The Pangui wins when its configured capture threshold is reached.
16. The Pu Trewa win when the Pangui has no legal normal moves or captures.
17. Optional triangle-occupation victory works only when enabled.
18. Illegal moves leave the state unchanged.
19. No moves can be made after the game ends.
20. Every generated legal move can be applied successfully.
21. Every applied move satisfies board-topology and occupancy constraints.

## 15. Implementation Caveat

The board layout and initial-position data are supplied separately and are essential to the game. Rules alone are insufficient to reconstruct a particular board layout or historical ruleset.

Before claiming that the implementation reproduces a particular traditional version, verify its exact board geometry, initial setup, capture threshold, capture-continuation rules, and movement restrictions against a reliable source documenting that version.

Keep board data separate from the rules engine so additional Komikan variants can be supported without rewriting the core game logic.