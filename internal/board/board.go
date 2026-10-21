// Package board holds the Komikan board topology and the pieces placed on it.
//
// The topology is data: nodes with grid coordinates and the edges between
// them. Nothing outside this package may assume node ids, node counts, or
// board dimensions. The canonical layout is documented in
// features/komikan-board-layout.md.
package board

import (
	"fmt"
	"sort"
)

// NodeID identifies a node on the board.
type NodeID int

// Point is a board coordinate in grid units. X is the column, Y is the row
// counted from the top of the board.
type Point struct {
	X int
	Y int
}

// Piece is the kind of occupant of a node. A position that does not mention
// a node is empty; PieceEmpty exists so that a lookup can return a value.
type Piece uint8

const (
	// PieceEmpty is an unoccupied node.
	PieceEmpty Piece = iota
	// PiecePangui is the puma, the single hunting piece.
	PiecePangui
	// PieceTrewa is one of the twelve dogs.
	PieceTrewa
)

func (p Piece) String() string {
	switch p {
	case PiecePangui:
		return "Pangui"
	case PieceTrewa:
		return "Trewa"
	default:
		return "empty"
	}
}

// Node is a place where a piece can stand.
type Node struct {
	ID        NodeID
	Coord     Point
	Neighbors []NodeID
}

// NodeSpec describes a node before adjacency is computed.
type NodeSpec struct {
	ID    NodeID
	Coord Point
}

// Board is the immutable topology of a Komikan board.
type Board struct {
	nodes map[NodeID]Node
	order []NodeID
	edges [][2]NodeID
}

// New builds a board from node specifications and an undirected edge list.
// It rejects duplicate ids, self loops, and edges that reference unknown
// nodes. Adjacency is derived from the edge list, so it is bidirectional by
// construction.
func New(specs []NodeSpec, edges [][2]NodeID) (*Board, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("board: no nodes")
	}

	b := &Board{nodes: make(map[NodeID]Node, len(specs))}
	for _, s := range specs {
		if _, dup := b.nodes[s.ID]; dup {
			return nil, fmt.Errorf("board: duplicate node id %d", s.ID)
		}
		b.nodes[s.ID] = Node{ID: s.ID, Coord: s.Coord}
		b.order = append(b.order, s.ID)
	}

	seen := make(map[[2]NodeID]bool, len(edges))
	for _, e := range edges {
		a, c := e[0], e[1]
		if a == c {
			return nil, fmt.Errorf("board: self loop on node %d", a)
		}
		if _, ok := b.nodes[a]; !ok {
			return nil, fmt.Errorf("board: edge references unknown node %d", a)
		}
		if _, ok := b.nodes[c]; !ok {
			return nil, fmt.Errorf("board: edge references unknown node %d", c)
		}
		key := [2]NodeID{a, c}
		if key[0] > key[1] {
			key = [2]NodeID{c, a}
		}
		if seen[key] {
			return nil, fmt.Errorf("board: duplicate edge %d-%d", key[0], key[1])
		}
		seen[key] = true
	}

	for _, e := range edges {
		a, c := e[0], e[1]
		na, nc := b.nodes[a], b.nodes[c]
		na.Neighbors = append(na.Neighbors, c)
		nc.Neighbors = append(nc.Neighbors, a)
		b.nodes[a] = na
		b.nodes[c] = nc
		b.edges = append(b.edges, e)
	}

	for id, n := range b.nodes {
		sort.Slice(n.Neighbors, func(i, j int) bool { return n.Neighbors[i] < n.Neighbors[j] })
		b.nodes[id] = n
	}
	sort.Slice(b.edges, func(i, j int) bool {
		if b.edges[i][0] != b.edges[j][0] {
			return b.edges[i][0] < b.edges[j][0]
		}
		return b.edges[i][1] < b.edges[j][1]
	})

	return b, nil
}

// Nodes returns every node in stable order.
func (b *Board) Nodes() []Node {
	out := make([]Node, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, b.nodes[id])
	}
	return out
}

// Node returns the node with the given id.
func (b *Board) Node(id NodeID) (Node, bool) {
	n, ok := b.nodes[id]
	return n, ok
}

// Has reports whether the board contains the node.
func (b *Board) Has(id NodeID) bool {
	_, ok := b.nodes[id]
	return ok
}

// Neighbors returns the nodes reachable from id in one step.
func (b *Board) Neighbors(id NodeID) []NodeID {
	n, ok := b.nodes[id]
	if !ok {
		return nil
	}
	return n.Neighbors
}

// Coord returns the grid coordinate of a node.
func (b *Board) Coord(id NodeID) (Point, bool) {
	n, ok := b.nodes[id]
	if !ok {
		return Point{}, false
	}
	return n.Coord, true
}

// Edges returns the undirected edge list.
func (b *Board) Edges() [][2]NodeID {
	return b.edges
}

// Position maps occupied nodes to the piece standing on them. Nodes that are
// absent from the map are empty.
type Position map[NodeID]Piece

// At returns the piece on a node, or PieceEmpty when it is unoccupied.
func (p Position) At(id NodeID) Piece {
	if piece, ok := p[id]; ok {
		return piece
	}
	return PieceEmpty
}

// Set places a piece on a node.
func (p Position) Set(id NodeID, piece Piece) {
	p[id] = piece
}

// Clone returns an independent copy of the position.
func (p Position) Clone() Position {
	out := make(Position, len(p))
	for id, piece := range p {
		out[id] = piece
	}
	return out
}

// Counts returns how many pieces of each kind are on the board.
func (p Position) Counts() (pangui, trewa int) {
	for _, piece := range p {
		switch piece {
		case PiecePangui:
			pangui++
		case PieceTrewa:
			trewa++
		}
	}
	return pangui, trewa
}
