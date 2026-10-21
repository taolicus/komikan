package board

// The canonical Komikan board: 38 nodes, 79 edges.
//
// This data is a transcription of features/komikan-board-layout.md, which is
// the authoritative description of the layout. Node ids are stable for the
// life of the project because persisted game states refer to them.
//
// Coordinates use grid units, X across the columns 0..4 and Y down the rows
// 0..7, matching the sketch in docs/board.md. Every edge spans exactly one
// unit horizontally, vertically, or diagonally.

// PanguiStart is the node the Pangui occupies at the start of a game: the
// middle of the second row, inside the triangle.
const PanguiStart NodeID = 7

// TrewaStart lists the nodes the twelve Trewa occupy at the start of a game:
// the two outer nodes of the square's third row from the bottom, plus the
// bottom two rows in full.
var TrewaStart = []NodeID{23, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37}

// SharedVertex is the node shared by the triangle and the square: the middle
// of the square's top row. It is the busiest junction on the board.
const SharedVertex NodeID = 15

var standardNodes = []NodeSpec{
	// Row 1.
	{0, Point{0, 0}}, {1, Point{1, 0}}, {2, Point{2, 0}}, {3, Point{3, 0}}, {4, Point{4, 0}},
	// Row 2.
	{5, Point{0, 1}}, {6, Point{1, 1}}, {7, Point{2, 1}}, {8, Point{3, 1}}, {9, Point{4, 1}},
	// Row 3.
	{10, Point{1, 2}}, {11, Point{2, 2}}, {12, Point{3, 2}},
	// Row 4, the top row of the square.
	{13, Point{0, 3}}, {14, Point{1, 3}}, {15, Point{2, 3}}, {16, Point{3, 3}}, {17, Point{4, 3}},
	// Row 5.
	{18, Point{0, 4}}, {19, Point{1, 4}}, {20, Point{2, 4}}, {21, Point{3, 4}}, {22, Point{4, 4}},
	// Row 6.
	{23, Point{0, 5}}, {24, Point{1, 5}}, {25, Point{2, 5}}, {26, Point{3, 5}}, {27, Point{4, 5}},
	// Row 7.
	{28, Point{0, 6}}, {29, Point{1, 6}}, {30, Point{2, 6}}, {31, Point{3, 6}}, {32, Point{4, 6}},
	// Row 8.
	{33, Point{0, 7}}, {34, Point{1, 7}}, {35, Point{2, 7}}, {36, Point{3, 7}}, {37, Point{4, 7}},
}

var standardHorizontalEdges = [][2]NodeID{
	{0, 1}, {1, 2}, {2, 3}, {3, 4},
	{5, 6}, {6, 7}, {7, 8}, {8, 9},
	{10, 11}, {11, 12},
	{13, 14}, {14, 15}, {15, 16}, {16, 17},
	{18, 19}, {19, 20}, {20, 21}, {21, 22},
	{23, 24}, {24, 25}, {25, 26}, {26, 27},
	{28, 29}, {29, 30}, {30, 31}, {31, 32},
	{33, 34}, {34, 35}, {35, 36}, {36, 37},
}

var standardVerticalEdges = [][2]NodeID{
	{0, 5}, {1, 6}, {2, 7}, {3, 8}, {4, 9},
	{6, 10}, {7, 11}, {8, 12},
	{11, 15},
	{13, 18}, {14, 19}, {15, 20}, {16, 21}, {17, 22},
	{18, 23}, {19, 24}, {20, 25}, {21, 26}, {22, 27},
	{23, 28}, {24, 29}, {25, 30}, {26, 31}, {27, 32},
	{28, 33}, {29, 34}, {30, 35}, {31, 36}, {32, 37},
}

var standardDiagonalEdges = [][2]NodeID{
	{5, 10}, {9, 12},
	{10, 15}, {12, 15},
	{13, 19}, {15, 19}, {15, 21}, {17, 21},
	{19, 23}, {19, 25}, {21, 25}, {21, 27},
	{23, 29}, {25, 29}, {25, 31}, {27, 31},
	{29, 33}, {29, 35}, {31, 35}, {31, 37},
}

// Standard returns the canonical board and its starting position.
func Standard() (*Board, Position, error) {
	edges := make([][2]NodeID, 0, len(standardHorizontalEdges)+len(standardVerticalEdges)+len(standardDiagonalEdges))
	edges = append(edges, standardHorizontalEdges...)
	edges = append(edges, standardVerticalEdges...)
	edges = append(edges, standardDiagonalEdges...)

	b, err := New(standardNodes, edges)
	if err != nil {
		return nil, nil, err
	}

	pos := make(Position, len(TrewaStart)+1)
	pos[PanguiStart] = PiecePangui
	for _, id := range TrewaStart {
		pos[id] = PieceTrewa
	}

	return b, pos, nil
}
