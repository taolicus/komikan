package rules

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/taolicus/komikan/internal/board"
)

// Player is one of the two sides.
type Player string

const (
	// PlayerPangui is the puma.
	PlayerPangui Player = "PANGUI"
	// PlayerTrewa is the pack of dogs.
	PlayerTrewa Player = "TREWA"
)

// Valid reports whether the player is one of the two sides.
func (p Player) Valid() bool {
	return p == PlayerPangui || p == PlayerTrewa
}

// Opponent returns the other side.
func (p Player) Opponent() Player {
	if p == PlayerPangui {
		return PlayerTrewa
	}
	return PlayerPangui
}

// Status is the state of a game.
type Status string

const (
	// StatusInProgress means the game is still being played.
	StatusInProgress Status = "IN_PROGRESS"
	// StatusPanguiWon means the Pangui reached the capture threshold.
	StatusPanguiWon Status = "PANGUI_WON"
	// StatusTrewaWon means the Trewa left the Pangui with no legal move.
	StatusTrewaWon Status = "PU_TREWA_WON"
	// StatusDraw means the game ended without a winner.
	StatusDraw Status = "DRAW"
)

// Over reports whether the game has ended.
func (s Status) Over() bool {
	return s != StatusInProgress
}

// MoveType is the kind of a move.
type MoveType string

const (
	// MoveNormal is a move along one edge to an empty node.
	MoveNormal MoveType = "NORMAL"
	// MoveCapture is a jump over a Trewa to the node beyond it.
	MoveCapture MoveType = "CAPTURE"
	// MoveEndTurn ends a capture sequence that may be stopped. It only
	// exists in variants where stopping a chain is allowed.
	MoveEndTurn MoveType = "END_TURN"
)

// Move is a single move of a game.
type Move struct {
	Player Player       `json:"player"`
	From   board.NodeID `json:"from"`
	To     board.NodeID `json:"to"`
	Type   MoveType     `json:"type"`
	// CapturedNode is the Trewa that was jumped over, for a capture.
	CapturedNode board.NodeID `json:"capturedNode,omitempty"`
}

// String describes the move in a readable way.
func (m Move) String() string {
	switch m.Type {
	case MoveCapture:
		return fmt.Sprintf("%s captures %d -> %d over %d", m.Player, m.From, m.To, m.CapturedNode)
	case MoveEndTurn:
		return fmt.Sprintf("%s ends the turn at %d", m.Player, m.From)
	default:
		return fmt.Sprintf("%s moves %d -> %d", m.Player, m.From, m.To)
	}
}

// Position maps the nodes that hold a piece to that piece. Nodes that are
// absent are empty.
type Position map[board.NodeID]board.Piece

// Clone returns a copy of the position.
func (p Position) Clone() Position {
	out := make(Position, len(p))
	for id, piece := range p {
		out[id] = piece
	}
	return out
}

// At returns the piece on a node, or board.PieceEmpty.
func (p Position) At(id board.NodeID) board.Piece {
	return p[id]
}

// Set places a piece on a node, or clears it when the piece is empty.
func (p Position) Set(id board.NodeID, piece board.Piece) {
	if piece == board.PieceEmpty {
		delete(p, id)
		return
	}
	p[id] = piece
}

// Nodes returns the occupied nodes, in order.
func (p Position) Nodes() []board.NodeID {
	ids := make([]board.NodeID, 0, len(p))
	for id := range p {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// PanguiNode returns the node of the Pangui.
func (p Position) PanguiNode() (board.NodeID, bool) {
	for id, piece := range p {
		if piece == board.PiecePangui {
			return id, true
		}
	}
	return 0, false
}

// Counts returns how many pieces of each kind are on the board.
func (p Position) Counts() (pangui, trewa int) {
	for _, piece := range p {
		switch piece {
		case board.PiecePangui:
			pangui++
		case board.PieceTrewa:
			trewa++
		}
	}
	return pangui, trewa
}

// MarshalJSON writes the position as a map from node id to piece name, the
// shape the API and the stored state use.
func (p Position) MarshalJSON() ([]byte, error) {
	out := make(map[string]string, len(p))
	for id, piece := range p {
		out[strconv.Itoa(int(id))] = pieceName(piece)
	}
	return json.Marshal(out)
}

// UnmarshalJSON reads a position written by MarshalJSON.
func (p *Position) UnmarshalJSON(data []byte) error {
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(Position, len(raw))
	for key, name := range raw {
		id, err := strconv.Atoi(key)
		if err != nil {
			return fmt.Errorf("rules: node id %q: %w", key, err)
		}
		piece, err := pieceFromName(name)
		if err != nil {
			return err
		}
		if piece == board.PieceEmpty {
			continue
		}
		out[board.NodeID(id)] = piece
	}
	*p = out
	return nil
}

// pieceName returns the name of a piece.
func pieceName(piece board.Piece) string {
	switch piece {
	case board.PiecePangui:
		return "PANGUI"
	case board.PieceTrewa:
		return "TREWA"
	default:
		return ""
	}
}

// pieceFromName returns the piece with that name.
func pieceFromName(name string) (board.Piece, error) {
	switch name {
	case "PANGUI":
		return board.PiecePangui, nil
	case "TREWA":
		return board.PieceTrewa, nil
	case "":
		return board.PieceEmpty, nil
	default:
		return board.PieceEmpty, fmt.Errorf("rules: unknown piece %q", name)
	}
}

// GameState is the whole state of a game.
type GameState struct {
	// BoardLayoutID names the board the state belongs to.
	BoardLayoutID string `json:"boardLayoutId"`
	// Position holds the pieces on the board.
	Position Position `json:"position"`
	// CurrentPlayer is the side to move.
	CurrentPlayer Player `json:"currentPlayer"`
	// CapturedTrewa is how many Trewa the Pangui has captured.
	CapturedTrewa int `json:"capturedTrewa"`
	// TurnNumber counts the turns of the game.
	TurnNumber int `json:"turnNumber"`
	// CaptureSequenceActive is true while the Pangui is in the middle of a
	// chain of captures and may continue it.
	CaptureSequenceActive bool `json:"captureSequenceActive"`
	// GameStatus is the status of the game.
	GameStatus Status `json:"gameStatus"`
	// Winner is the side that won, when the game is over.
	Winner Player `json:"winner,omitempty"`
	// Ruleset is the variant being played.
	Ruleset Ruleset `json:"ruleset"`
	// MoveHistory lists the moves of the game in order.
	MoveHistory []Move `json:"moveHistory"`
}

// Clone returns a deep copy of the state. Applying a move to a state never
// touches the state it came from.
func (s GameState) Clone() GameState {
	out := s
	out.Position = s.Position.Clone()
	if s.MoveHistory != nil {
		out.MoveHistory = make([]Move, len(s.MoveHistory))
		copy(out.MoveHistory, s.MoveHistory)
	}
	return out
}
