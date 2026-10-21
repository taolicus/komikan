package rules

import (
	"errors"
	"fmt"
	"sort"

	"github.com/taolicus/komikan/internal/board"
)

// Engine plays one variant of Komikan on one board.
//
// The board is held by the engine instead of the state, so a state stays a
// plain document that can be stored and sent to a browser.
type Engine struct {
	board   *board.Board
	ruleset Ruleset
}

// NewEngine builds an engine for a board and a ruleset.
func NewEngine(b *board.Board, ruleset Ruleset) (*Engine, error) {
	if b == nil {
		return nil, errors.New("rules: a board is required")
	}
	if err := ruleset.Validate(); err != nil {
		return nil, err
	}
	return &Engine{board: b, ruleset: ruleset}, nil
}

// Ruleset returns the variant the engine plays.
func (e *Engine) Ruleset() Ruleset {
	return e.ruleset
}

// Board returns the board the engine plays on.
func (e *Engine) Board() *board.Board {
	return e.board
}

// NewGame returns the starting state of a game.
func (e *Engine) NewGame() GameState {
	position := make(Position, len(board.TrewaStart)+1)
	position[board.PanguiStart] = board.PiecePangui
	for _, id := range board.TrewaStart {
		position[id] = board.PieceTrewa
	}

	first := PlayerPangui
	if !e.ruleset.PanguiMovesFirst {
		first = PlayerTrewa
	}

	return GameState{
		BoardLayoutID: LayoutStandard,
		Position:      position,
		CurrentPlayer: first,
		TurnNumber:    1,
		GameStatus:    StatusInProgress,
		Ruleset:       e.ruleset,
		MoveHistory:   []Move{},
	}
}

// LegalMoves returns the moves the given player may play in the given state.
//
// The result is sorted, so the same state always produces the same list.
func (e *Engine) LegalMoves(state GameState, player Player) []Move {
	if state.GameStatus.Over() {
		return nil
	}
	switch player {
	case PlayerPangui:
		return e.panguiMoves(state)
	case PlayerTrewa:
		return e.trewaMoves(state)
	default:
		return nil
	}
}

// panguiMoves returns the moves of the puma.
func (e *Engine) panguiMoves(state GameState) []Move {
	from, ok := state.Position.PanguiNode()
	if !ok {
		return nil
	}

	captures := e.captureMoves(state, from)

	// In the middle of a chain of captures, the puma may only continue
	// capturing. When stopping is allowed, it may also end the turn.
	if state.CaptureSequenceActive {
		if e.ruleset.AllowMultipleCaptures && e.ruleset.MultipleCapturesMandatory {
			return captures
		}
		moves := append([]Move(nil), captures...)
		moves = append(moves, Move{Player: PlayerPangui, From: from, To: from, Type: MoveEndTurn})
		return sortMoves(moves)
	}

	// Traditional accounts say the puma must capture when it can.
	if e.ruleset.CapturePreferred && len(captures) > 0 {
		return captures
	}

	normals := e.normalMoves(state, from, PlayerPangui)
	return sortMoves(append(normals, captures...))
}

// trewaMoves returns the moves of the dogs.
func (e *Engine) trewaMoves(state GameState) []Move {
	var moves []Move
	for _, from := range state.Position.Nodes() {
		if state.Position.At(from) != board.PieceTrewa {
			continue
		}
		moves = append(moves, e.normalMoves(state, from, PlayerTrewa)...)
	}
	return sortMoves(moves)
}

// normalMoves returns the moves along a single edge to an empty node.
func (e *Engine) normalMoves(state GameState, from board.NodeID, player Player) []Move {
	var moves []Move
	for _, to := range e.board.Neighbors(from) {
		if state.Position.At(to) != board.PieceEmpty {
			continue
		}
		if player == PlayerTrewa && !e.ruleset.TrewaCanMoveBackward && !e.isForward(from, to) {
			continue
		}
		moves = append(moves, Move{Player: player, From: from, To: to, Type: MoveNormal})
	}
	return moves
}

// captureMoves returns the jumps the puma can make from a node.
//
// A jump needs three nodes in a straight line at even spacing: the puma, a
// Trewa, and the empty node beyond it. The two edges must exist on the
// board, and the Trewa must be exactly in the middle.
func (e *Engine) captureMoves(state GameState, from board.NodeID) []Move {
	var moves []Move
	for _, over := range e.board.Neighbors(from) {
		if state.Position.At(over) != board.PieceTrewa {
			continue
		}
		for _, to := range e.board.Neighbors(over) {
			if to == from {
				continue
			}
			if state.Position.At(to) != board.PieceEmpty {
				continue
			}
			if !e.isStraightLine(from, over, to) {
				continue
			}
			moves = append(moves, Move{
				Player:       PlayerPangui,
				From:         from,
				To:           to,
				Type:         MoveCapture,
				CapturedNode: over,
			})
		}
	}
	return sortMoves(moves)
}

// isStraightLine reports whether the middle node is the exact midpoint of
// the other two.
func (e *Engine) isStraightLine(from, over, to board.NodeID) bool {
	a, ok := e.board.Coord(from)
	if !ok {
		return false
	}
	b, ok := e.board.Coord(over)
	if !ok {
		return false
	}
	c, ok := e.board.Coord(to)
	if !ok {
		return false
	}
	return a.X+c.X == 2*b.X && a.Y+c.Y == 2*b.Y
}

// isForward reports whether a move goes towards the triangle, which is the
// direction the Trewa advance in.
func (e *Engine) isForward(from, to board.NodeID) bool {
	a, ok := e.board.Coord(from)
	if !ok {
		return false
	}
	c, ok := e.board.Coord(to)
	if !ok {
		return false
	}
	return c.Y < a.Y
}

// Evaluate returns the state with the terminal conditions applied, without
// playing a move. It tells whether a position that was set up outside of
// the engine is already decided.
func (e *Engine) Evaluate(state GameState) GameState {
	next := state.Clone()
	e.checkVictory(&next)
	return next
}

// ApplyMove applies a move to a state and returns the new state.
//
// The state that came in is never modified, and an illegal move returns an
// error and leaves the state exactly as it was.
func (e *Engine) ApplyMove(state GameState, move Move) (GameState, error) {
	if state.GameStatus.Over() {
		return state, fmt.Errorf("rules: the game is over (%s)", state.GameStatus)
	}
	if !move.Player.Valid() {
		return state, fmt.Errorf("rules: unknown player %q", move.Player)
	}
	if move.Player != state.CurrentPlayer {
		return state, fmt.Errorf("rules: it is %s's turn, not %s's", state.CurrentPlayer, move.Player)
	}
	if !e.isLegal(state, move) {
		return state, fmt.Errorf("rules: illegal move: %s", move)
	}

	next := state.Clone()
	switch move.Type {
	case MoveNormal:
		piece := next.Position.At(move.From)
		next.Position.Set(move.From, board.PieceEmpty)
		next.Position.Set(move.To, piece)
	case MoveCapture:
		piece := next.Position.At(move.From)
		next.Position.Set(move.From, board.PieceEmpty)
		next.Position.Set(move.CapturedNode, board.PieceEmpty)
		next.Position.Set(move.To, piece)
		next.CapturedTrewa++
	case MoveEndTurn:
		// The board does not change.
	default:
		return state, fmt.Errorf("rules: unknown move type %q", move.Type)
	}
	next.MoveHistory = append(next.MoveHistory, move)

	// The turn continues while the puma has a capture to chain.
	next.CaptureSequenceActive = false
	if move.Type == MoveCapture && e.ruleset.AllowMultipleCaptures {
		if len(e.captureMoves(next, move.To)) > 0 {
			next.CaptureSequenceActive = true
		}
	}
	if !next.CaptureSequenceActive {
		next.CurrentPlayer = state.CurrentPlayer.Opponent()
		next.TurnNumber++
	}

	e.checkVictory(&next)
	return next, nil
}

// isLegal reports whether a move is among the legal moves of a state.
func (e *Engine) isLegal(state GameState, move Move) bool {
	for _, legal := range e.LegalMoves(state, move.Player) {
		if legal == move {
			return true
		}
	}
	return false
}

// checkVictory applies the terminal conditions to a state.
func (e *Engine) checkVictory(state *GameState) {
	thresholdReached := e.ruleset.PanguiCapturesToWin > 0 &&
		state.CapturedTrewa >= e.ruleset.PanguiCapturesToWin
	blocked := e.ruleset.PuTrewaWinByBlocking &&
		len(e.LegalMoves(*state, PlayerPangui)) == 0

	// The order of the two checks is a rule of its own: by default the
	// capture that wins the game is not cancelled by the position it
	// leaves behind.
	if e.ruleset.ThresholdBeforeBlocking {
		if thresholdReached {
			e.win(state, StatusPanguiWon, PlayerPangui)
			return
		}
		if blocked {
			e.win(state, StatusTrewaWon, PlayerTrewa)
			return
		}
	} else {
		if blocked {
			e.win(state, StatusTrewaWon, PlayerTrewa)
			return
		}
		if thresholdReached {
			e.win(state, StatusPanguiWon, PlayerPangui)
			return
		}
	}

	// The dogs are stuck: the game ends without a winner.
	if e.ruleset.DrawWhenTrewaCannotMove && len(e.LegalMoves(*state, PlayerTrewa)) == 0 {
		state.GameStatus = StatusDraw
		state.Winner = ""
	}
}

// win marks the game as finished.
func (e *Engine) win(state *GameState, status Status, winner Player) {
	state.GameStatus = status
	state.Winner = winner
}

// sortMoves puts moves in a stable order.
func sortMoves(moves []Move) []Move {
	sort.Slice(moves, func(i, j int) bool {
		if moves[i].From != moves[j].From {
			return moves[i].From < moves[j].From
		}
		if moves[i].To != moves[j].To {
			return moves[i].To < moves[j].To
		}
		return moves[i].Type < moves[j].Type
	})
	return moves
}
