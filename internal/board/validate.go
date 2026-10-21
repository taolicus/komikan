package board

import (
	"fmt"
	"sort"
)

// Validate checks that a board and a position are well formed: unique node
// ids, symmetric adjacency, no isolated nodes, a connected graph, and a legal
// placement of pieces.
func Validate(b *Board, pos Position) error {
	if b == nil {
		return fmt.Errorf("board: nil board")
	}

	nodes := b.Nodes()
	if len(nodes) == 0 {
		return fmt.Errorf("board: no nodes")
	}

	seen := make(map[NodeID]bool, len(nodes))
	for _, n := range nodes {
		if seen[n.ID] {
			return fmt.Errorf("board: duplicate node id %d", n.ID)
		}
		seen[n.ID] = true
	}

	for _, e := range b.Edges() {
		a, c := e[0], e[1]
		if !b.Has(a) {
			return fmt.Errorf("board: edge references unknown node %d", a)
		}
		if !b.Has(c) {
			return fmt.Errorf("board: edge references unknown node %d", c)
		}
		if !contains(b.Neighbors(a), c) {
			return fmt.Errorf("board: edge %d-%d is not reachable from %d", a, c, a)
		}
		if !contains(b.Neighbors(c), a) {
			return fmt.Errorf("board: edge %d-%d is not reachable from %d", a, c, c)
		}
	}

	for _, n := range nodes {
		if len(n.Neighbors) == 0 {
			return fmt.Errorf("board: node %d is isolated", n.ID)
		}
	}

	if err := validateConnected(b); err != nil {
		return err
	}

	return validatePosition(b, pos)
}

func validateConnected(b *Board) error {
	nodes := b.Nodes()
	start := nodes[0].ID
	visited := map[NodeID]bool{start: true}
	stack := []NodeID{start}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, next := range b.Neighbors(cur) {
			if !visited[next] {
				visited[next] = true
				stack = append(stack, next)
			}
		}
	}
	if len(visited) != len(nodes) {
		var unreachable []int
		for _, n := range nodes {
			if !visited[n.ID] {
				unreachable = append(unreachable, int(n.ID))
			}
		}
		sort.Ints(unreachable)
		return fmt.Errorf("board: graph is not connected; unreachable nodes %v", unreachable)
	}
	return nil
}

func validatePosition(b *Board, pos Position) error {
	for id, piece := range pos {
		if !b.Has(id) {
			return fmt.Errorf("board: position refers to unknown node %d", id)
		}
		if piece == PieceEmpty {
			return fmt.Errorf("board: position lists node %d as empty; omit empty nodes instead", id)
		}
	}
	pangui, trewa := pos.Counts()
	if pangui != 1 {
		return fmt.Errorf("board: expected exactly one Pangui, found %d", pangui)
	}
	if trewa != 12 {
		return fmt.Errorf("board: expected exactly twelve Trewa, found %d", trewa)
	}
	return nil
}

func contains(ids []NodeID, want NodeID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
