package board

import (
	"fmt"
	"sort"
	"strings"
)

// The ASCII renderer reproduces the sketch style used in docs/board.md: four
// spaces of margin, a six character pitch between nodes, three dashes between
// horizontally connected nodes, and connector rows between the node rows.
const (
	asciiMargin = 4
	asciiPitch  = 6
)

// RenderASCII draws the board as an ASCII sketch. Occupied nodes are drawn
// as "x" and empty nodes as "o", so the output matches the diagram in
// docs/board.md.
func (b *Board) RenderASCII(pos Position) string {
	out, _ := b.renderASCII(pos, func(p Piece) rune {
		if p == PieceEmpty {
			return 'o'
		}
		return 'x'
	})
	return out
}

// RenderASCIIWithPieces draws the board as an ASCII sketch with distinct
// glyphs per piece kind: "P" for the Pangui, "T" for a Trewa, "o" for an
// empty node.
func (b *Board) RenderASCIIWithPieces(pos Position) string {
	out, _ := b.renderASCII(pos, func(p Piece) rune {
		switch p {
		case PiecePangui:
			return 'P'
		case PieceTrewa:
			return 'T'
		default:
			return 'o'
		}
	})
	return out
}

// renderASCII draws the board and reports how many edges were drawn, so a
// test can prove that no edge is left out of the picture.
func (b *Board) renderASCII(pos Position, glyph func(Piece) rune) (string, int) {
	rows := b.rows()

	width := asciiMargin + asciiPitch*maxX(rows) + 1
	c := newCanvas(width, 2*len(rows)-1)

	drawn := 0
	for _, row := range rows {
		y := row.y
		for _, node := range row.nodes {
			c.set(asciiMargin+asciiPitch*node.Coord.X, 2*y, glyph(pos.At(node.ID)))
		}
		// Horizontal edges between nodes of the same row.
		for i := 0; i+1 < len(row.nodes); i++ {
			a, bNode := row.nodes[i], row.nodes[i+1]
			if b.connected(a.ID, bNode.ID) {
				for k := 2; k < asciiPitch-1; k++ {
					c.set(asciiMargin+asciiPitch*a.Coord.X+k, 2*y, '-')
				}
				drawn++
			}
		}
	}

	// Vertical and diagonal edges between adjacent rows.
	for i := 0; i+1 < len(rows); i++ {
		top, bottom := rows[i], rows[i+1]
		topByX := map[int]Node{}
		for _, n := range top.nodes {
			topByX[n.Coord.X] = n
		}
		bottomByX := map[int]Node{}
		for _, n := range bottom.nodes {
			bottomByX[n.Coord.X] = n
		}
		for x, topNode := range topByX {
			if bottomNode, ok := bottomByX[x]; ok && b.connected(topNode.ID, bottomNode.ID) {
				c.set(asciiMargin+asciiPitch*x, 2*top.y+1, '|')
				drawn++
			}
		}
		for x := 0; x <= maxX(rows); x++ {
			topLeft, okL := topByX[x]
			topRight, okR := topByX[x+1]
			bottomLeft, okBL := bottomByX[x]
			bottomRight, okBR := bottomByX[x+1]
			mid := asciiMargin + asciiPitch*x + 3
			if okL && okBR && b.connected(topLeft.ID, bottomRight.ID) {
				c.set(mid, 2*top.y+1, '\\')
				drawn++
			}
			if okR && okBL && b.connected(topRight.ID, bottomLeft.ID) {
				c.set(mid, 2*top.y+1, '/')
				drawn++
			}
		}
	}

	return strings.Join(c.lines(), "\n"), drawn
}

// RenderSVG draws the board as an SVG element. The result can be embedded in
// an HTML document; add an XML declaration when writing it to a file as a
// standalone document. The Pangui and the Trewa are drawn differently so the
// position can be reviewed at a glance.
func (b *Board) RenderSVG(pos Position) string {
	const (
		cell   = 64
		margin = 48
	)

	nodes := b.Nodes()
	maxX, maxY := 0, 0
	for _, n := range nodes {
		if n.Coord.X > maxX {
			maxX = n.Coord.X
		}
		if n.Coord.Y > maxY {
			maxY = n.Coord.Y
		}
	}

	width := 2*margin + maxX*cell
	height := 2*margin + maxY*cell

	px := func(id NodeID) (int, int) {
		n, _ := b.Node(id)
		return margin + n.Coord.X*cell, margin + n.Coord.Y*cell
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="sans-serif">`+"\n", width, height, width, height))
	sb.WriteString("  <title>Komikan board</title>\n")
	sb.WriteString(fmt.Sprintf("  <desc>%d nodes, %d edges, starting position.</desc>\n", len(nodes), len(b.Edges())))
	sb.WriteString(fmt.Sprintf(`  <rect width="%d" height="%d" fill="#fdfaf4"/>`+"\n", width, height))

	sb.WriteString("  <g stroke=\"#475569\" stroke-width=\"3\" stroke-linecap=\"round\">\n")
	for _, e := range b.Edges() {
		x1, y1 := px(e[0])
		x2, y2 := px(e[1])
		sb.WriteString(fmt.Sprintf(`    <line x1="%d" y1="%d" x2="%d" y2="%d"/>`+"\n", x1, y1, x2, y2))
	}
	sb.WriteString("  </g>\n")

	// The pieces are drawn as emoji on a dark disc: the Pangui is a leopard
	// on deep green, the Trewa are dogs on deep red. The discs are dark so
	// the warm emoji colours read clearly against them.
	for _, n := range nodes {
		x, y := px(n.ID)
		switch pos.At(n.ID) {
		case PiecePangui:
			sb.WriteString(fmt.Sprintf(`    <circle cx="%d" cy="%d" r="20" fill="#14532d" stroke="#4ade80" stroke-width="3"/>`+"\n", x, y))
			sb.WriteString(fmt.Sprintf(`    <text x="%d" y="%d" font-size="32" text-anchor="middle">&#128006;</text>`+"\n", x, y+11))
		case PieceTrewa:
			sb.WriteString(fmt.Sprintf(`    <circle cx="%d" cy="%d" r="16" fill="#7f1d1d" stroke="#f87171" stroke-width="3"/>`+"\n", x, y))
			sb.WriteString(fmt.Sprintf(`    <text x="%d" y="%d" font-size="25" text-anchor="middle">&#128021;</text>`+"\n", x, y+9))
		default:
			sb.WriteString(fmt.Sprintf(`    <circle cx="%d" cy="%d" r="6" fill="#ffffff" stroke="#94a3b8" stroke-width="2"/>`+"\n", x, y))
		}
	}

	sb.WriteString("</svg>\n")
	return sb.String()
}

type boardRow struct {
	y     int
	nodes []Node
}

func (b *Board) rows() []boardRow {
	nodes := b.Nodes()
	sorted := append([]Node(nil), nodes...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Coord.Y != sorted[j].Coord.Y {
			return sorted[i].Coord.Y < sorted[j].Coord.Y
		}
		return sorted[i].Coord.X < sorted[j].Coord.X
	})

	var out []boardRow
	for _, n := range sorted {
		if len(out) == 0 || out[len(out)-1].y != n.Coord.Y {
			out = append(out, boardRow{y: n.Coord.Y})
		}
		out[len(out)-1].nodes = append(out[len(out)-1].nodes, n)
	}
	return out
}

func (b *Board) connected(a, c NodeID) bool {
	for _, id := range b.Neighbors(a) {
		if id == c {
			return true
		}
	}
	return false
}

func maxX(rows []boardRow) int {
	max := 0
	for _, r := range rows {
		for _, n := range r.nodes {
			if n.Coord.X > max {
				max = n.Coord.X
			}
		}
	}
	return max
}

type canvas struct {
	cells [][]rune
}

func newCanvas(width, height int) *canvas {
	c := &canvas{cells: make([][]rune, height)}
	for y := range c.cells {
		c.cells[y] = []rune(strings.Repeat(" ", width))
	}
	return c
}

func (c *canvas) set(x, y int, r rune) {
	if y < 0 || y >= len(c.cells) {
		return
	}
	if x < 0 || x >= len(c.cells[y]) {
		return
	}
	c.cells[y][x] = r
}

func (c *canvas) lines() []string {
	out := make([]string, 0, len(c.cells))
	for _, row := range c.cells {
		out = append(out, strings.TrimRight(string(row), " "))
	}
	return out
}
