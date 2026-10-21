package board

import (
	"encoding/xml"
	"strings"
	"testing"
)

// The sketch from docs/board.md. The ASCII renderer must reproduce it
// exactly, connector rows included.
const sketch = `    o --- o --- o --- o --- o
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
    x --- x --- x --- x --- x`

func standard(t *testing.T) (*Board, Position) {
	t.Helper()
	b, pos, err := Standard()
	if err != nil {
		t.Fatalf("Standard(): %v", err)
	}
	return b, pos
}

func TestStandardLayoutIsValid(t *testing.T) {
	b, pos := standard(t)
	if err := Validate(b, pos); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestNodeAndEdgeCounts(t *testing.T) {
	b, _ := standard(t)
	if got := len(b.Nodes()); got != 38 {
		t.Errorf("node count = %d, want 38", got)
	}
	if got := len(b.Edges()); got != 79 {
		t.Errorf("edge count = %d, want 79", got)
	}
}

func TestNodeIDsAreUnique(t *testing.T) {
	b, _ := standard(t)
	seen := map[NodeID]bool{}
	for _, n := range b.Nodes() {
		if seen[n.ID] {
			t.Errorf("duplicate node id %d", n.ID)
		}
		seen[n.ID] = true
	}
	if len(seen) != 38 {
		t.Errorf("distinct node ids = %d, want 38", len(seen))
	}
}

func TestEdgesAreBidirectional(t *testing.T) {
	b, _ := standard(t)
	for _, e := range b.Edges() {
		if !contains(b.Neighbors(e[0]), e[1]) {
			t.Errorf("node %d does not list neighbour %d", e[0], e[1])
		}
		if !contains(b.Neighbors(e[1]), e[0]) {
			t.Errorf("node %d does not list neighbour %d", e[1], e[0])
		}
	}
}

func TestGraphIsConnected(t *testing.T) {
	b, _ := standard(t)
	visited := map[NodeID]bool{}
	stack := []NodeID{0}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		stack = append(stack, b.Neighbors(cur)...)
	}
	if len(visited) != 38 {
		t.Errorf("reachable nodes = %d, want 38", len(visited))
	}
}

func TestNoIsolatedNodes(t *testing.T) {
	b, _ := standard(t)
	for _, n := range b.Nodes() {
		if len(n.Neighbors) == 0 {
			t.Errorf("node %d is isolated", n.ID)
		}
	}
}

func TestEdgesSpanOneUnit(t *testing.T) {
	b, _ := standard(t)
	for _, e := range b.Edges() {
		pa, _ := b.Coord(e[0])
		pb, _ := b.Coord(e[1])
		dx := abs(pa.X - pb.X)
		dy := abs(pa.Y - pb.Y)
		if !((dx == 1 && dy == 0) || (dx == 0 && dy == 1) || (dx == 1 && dy == 1)) {
			t.Errorf("edge %d-%d spans (%d, %d), want a unit step", e[0], e[1], dx, dy)
		}
	}
}

func TestInitialPosition(t *testing.T) {
	b, pos := standard(t)

	pangui, trewa := pos.Counts()
	if pangui != 1 {
		t.Errorf("Pangui count = %d, want 1", pangui)
	}
	if trewa != 12 {
		t.Errorf("Trewa count = %d, want 12", trewa)
	}
	if len(pos) != 13 {
		t.Errorf("occupied nodes = %d, want 13", len(pos))
	}
	for id := range pos {
		if !b.Has(id) {
			t.Errorf("occupied node %d does not exist on the board", id)
		}
	}
	if pos.At(PanguiStart) != PiecePangui {
		t.Errorf("Pangui is not on node %d", PanguiStart)
	}
	for _, id := range TrewaStart {
		if pos.At(id) != PieceTrewa {
			t.Errorf("Trewa missing from node %d", id)
		}
	}
}

func TestRenderASCIIReproducesSketch(t *testing.T) {
	b, pos := standard(t)
	got := b.RenderASCII(pos)
	if got != sketch {
		t.Errorf("render does not match the sketch in docs/board.md\n--- got ---\n%s\n--- want ---\n%s", got, sketch)
	}
}

func TestRenderASCIIDrawsEveryEdge(t *testing.T) {
	b, pos := standard(t)
	_, drawn := b.renderASCII(pos, func(Piece) rune { return 'x' })
	if drawn != len(b.Edges()) {
		t.Errorf("drew %d edges, board has %d", drawn, len(b.Edges()))
	}
}

func TestRenderASCIIWithPiecesMarksBothKinds(t *testing.T) {
	b, pos := standard(t)
	got := b.RenderASCIIWithPieces(pos)
	if strings.Count(got, "P") != 1 {
		t.Errorf("expected exactly one Pangui glyph, got %d", strings.Count(got, "P"))
	}
	if strings.Count(got, "T") != 12 {
		t.Errorf("expected exactly twelve Trewa glyphs, got %d", strings.Count(got, "T"))
	}
}

func TestRenderASCIIIsEmptyForEmptyPosition(t *testing.T) {
	b, _ := standard(t)
	got := b.RenderASCII(Position{})
	if strings.Contains(got, "x") {
		t.Errorf("empty position rendered an occupied node:\n%s", got)
	}
}

func TestRenderSVGIsWellFormed(t *testing.T) {
	b, pos := standard(t)
	svg := b.RenderSVG(pos)
	if !strings.HasPrefix(svg, "<svg ") {
		t.Error("SVG does not start with its root element")
	}
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Error("SVG is missing its root element")
	}
	if got := strings.Count(svg, "<line "); got != len(b.Edges()) {
		t.Errorf("SVG draws %d edges, board has %d", got, len(b.Edges()))
	}
	if got := strings.Count(svg, "<circle "); got != 38+2 {
		t.Errorf("SVG draws %d circles, want %d nodes plus 2 legend entries", got, 38+2)
	}

	// The same markup, with an XML declaration, is a valid standalone
	// document. This is what boarddump writes to disk.
	var doc struct{}
	if err := xml.Unmarshal([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"+svg), &doc); err != nil {
		t.Errorf("SVG is not valid XML: %v", err)
	}
}

func TestNewRejectsBadBoards(t *testing.T) {
	tests := []struct {
		name  string
		specs []NodeSpec
		edges [][2]NodeID
	}{
		{
			name:  "duplicate node id",
			specs: []NodeSpec{{0, Point{0, 0}}, {0, Point{1, 0}}},
		},
		{
			name:  "edge to unknown node",
			specs: []NodeSpec{{0, Point{0, 0}}, {1, Point{1, 0}}},
			edges: [][2]NodeID{{0, 9}},
		},
		{
			name:  "self loop",
			specs: []NodeSpec{{0, Point{0, 0}}},
			edges: [][2]NodeID{{0, 0}},
		},
		{
			name:  "duplicate edge",
			specs: []NodeSpec{{0, Point{0, 0}}, {1, Point{1, 0}}},
			edges: [][2]NodeID{{0, 1}, {1, 0}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.specs, tc.edges); err == nil {
				t.Error("New() accepted an invalid board")
			}
		})
	}
}

func TestValidateRejectsBadPositions(t *testing.T) {
	b, pos := standard(t)

	noPangui := pos.Clone()
	delete(noPangui, PanguiStart)
	if err := Validate(b, noPangui); err == nil {
		t.Error("Validate() accepted a position without a Pangui")
	}

	unknown := pos.Clone()
	unknown[NodeID(999)] = PieceTrewa
	if err := Validate(b, unknown); err == nil {
		t.Error("Validate() accepted a piece on an unknown node")
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
