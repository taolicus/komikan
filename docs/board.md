# Komikan Board Data Structure — Brief Specification

> **Status: superseded.** The sketch below is the source the board was read
> from. The authoritative node and edge list is
> `features/komikan-board-layout.md` (38 nodes, 79 edges); use that for
> implementation. The sketch is kept for reference and for the renderer to
> reproduce.

## 1. Board Diagram

The following ASCII diagram represents the board topology. `x` marks initially occupied nodes, and `o` marks initially empty nodes. The bottom vertex of the triangle is shared with the square region.

    o --- o --- o --- o --- o
    |     |     |     |     |
    o --- o --- x --- o --- o
       \  |     |     |  /
          o --- o --- o
             \  |  /
    o --- o --- o --- o --- o
    |  \  |  /  |  \  |  /  |
    o --- o --- o --- o --- o
    |  /  |  \  |  /  |  \  |
    x --- o --- o --- o --- x
    |  \  |  /  |  \  |  /  |
    x --- x --- x --- x --- x
    |  /  |  \  |  /  |  \  |
    x --- x --- x --- x --- x

## 2. Board Representation

Represent the Komikan board as an undirected graph.

- **Node:** An intersection where a piece can be placed.
- **Edge:** A connection between two nodes along which a piece may move.
- **Board:** A collection of nodes and their connections.
- **Position:** The pieces currently occupying nodes.

## 3. Data Types

Implement the following structures in Go:

- `NodeID`: An integer uniquely identifying a board node.
- `Node`: Contains a `NodeID` and a list of neighboring node IDs.
- `Board`: Contains all nodes and their adjacency lists.
- `Piece`: An enumeration representing an empty position, the puma, or a prey piece.
- `Position`: Maps node IDs to the pieces occupying them. Missing entries represent empty nodes.

## 4. Board Topology

- Model the upper triangular region and lower square region as one graph.
- The triangle's bottom vertex is shared with the square region and must be represented by a single node.
- Define edges explicitly, including horizontal, vertical, and diagonal connections.
- Every edge must be bidirectional: if node A connects to node B, node B must connect to node A.
- Node IDs must remain stable across game positions.

## 5. Initial Position

Define the initial position separately from the board topology.

- Nodes marked `x` in the diagram start occupied.
- Nodes marked `o` start empty.
- The initial position specifies which piece occupies each initially occupied node.
- The board topology remains unchanged throughout the game.

## 6. Movement and Game Logic

- A piece may move only along an edge connecting its current node to a destination node.
- The destination must be empty unless a specific game rule permits otherwise.
- Capture rules, turn order, and victory conditions belong to the game logic, not the board structure.
- Keep movement validation independent of rendering.

## 7. Optional Rendering Metadata

If the board must be drawn, each node may additionally store two-dimensional coordinates (`X`, `Y`). These coordinates are for display only and must not determine legal movement.