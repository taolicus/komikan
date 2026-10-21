package rules

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"

	"github.com/taolicus/komikan/internal/board"
)

// testEngine returns an engine for the canonical board with the default
// ruleset.
func testEngine(t *testing.T) *Engine {
	t.Helper()
	b, _, err := board.Standard()
	if err != nil {
		t.Fatalf("board.Standard(): %v", err)
	}
	engine, err := NewEngine(b, DefaultRuleset())
	if err != nil {
		t.Fatalf("NewEngine(): %v", err)
	}
	return engine
}

// stateWith builds a state from a position.
func stateWith(pieces map[board.NodeID]board.Piece, current Player) GameState {
	position := make(Position, len(pieces))
	for id, piece := range pieces {
		position[id] = piece
	}
	return GameState{
		BoardLayoutID: LayoutStandard,
		Position:      position,
		CurrentPlayer: current,
		TurnNumber:    1,
		GameStatus:    StatusInProgress,
		Ruleset:       DefaultRuleset(),
		MoveHistory:   []Move{},
	}
}

// hasMove reports whether the list holds a move from one node to another.
func hasMove(moves []Move, from, to board.NodeID, kind MoveType) bool {
	for _, move := range moves {
		if move.From == from && move.To == to && move.Type == kind {
			return true
		}
	}
	return false
}

func TestInitialPosition(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()

	pangui, trewa := state.Position.Counts()
	if pangui != 1 || trewa != 12 {
		t.Errorf("pieces = %d Pangui and %d Trewa, want 1 and 12", pangui, trewa)
	}
	if node, ok := state.Position.PanguiNode(); !ok || node != board.PanguiStart {
		t.Errorf("the Pangui is on %v, want %d", node, board.PanguiStart)
	}
	if state.CurrentPlayer != PlayerPangui {
		t.Errorf("current player = %q, want %q", state.CurrentPlayer, PlayerPangui)
	}
	if state.GameStatus != StatusInProgress {
		t.Errorf("status = %q, want %q", state.GameStatus, StatusInProgress)
	}
	if state.TurnNumber != 1 {
		t.Errorf("turn = %d, want 1", state.TurnNumber)
	}
}

func TestPanguiCanMoveToAnAdjacentEmptyNode(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()

	moves := engine.LegalMoves(state, PlayerPangui)
	if !hasMove(moves, 7, 6, MoveNormal) {
		t.Errorf("the Pangui cannot move from 7 to 6: %v", moves)
	}
}

func TestPanguiCannotMoveToAnOccupiedNode(t *testing.T) {
	engine := testEngine(t)
	// The Pangui at 7 is surrounded by Trewa on two of its neighbours.
	state := stateWith(map[board.NodeID]board.Piece{
		7:  board.PiecePangui,
		6:  board.PieceTrewa,
		8:  board.PieceTrewa,
		11: board.PieceTrewa,
	}, PlayerPangui)

	for _, move := range engine.LegalMoves(state, PlayerPangui) {
		if move.To == 6 || move.To == 8 || move.To == 11 {
			t.Errorf("the Pangui may not move onto %d", move.To)
		}
	}
}

func TestTrewaCanMoveToAnAdjacentEmptyNode(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()

	// The bottom rows are full, so the Trewa on 30 moves up to 25.
	moves := engine.LegalMoves(state, PlayerTrewa)
	if !hasMove(moves, 30, 25, MoveNormal) {
		t.Errorf("the Trewa on 30 cannot move to 25: %v", moves)
	}
}

func TestTrewaCannotJumpOverPieces(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()

	for _, move := range engine.LegalMoves(state, PlayerTrewa) {
		if move.From == move.To {
			t.Errorf("the Trewa may not stay on its node: %v", move)
		}
		if state.Position.At(move.To) != board.PieceEmpty {
			t.Errorf("the Trewa may not move onto an occupied node: %v", move)
		}
		if !engine.board.Has(move.From) || !engine.board.Has(move.To) {
			t.Errorf("the move leaves the board: %v", move)
		}
	}
}

func TestCaptureRemovesExactlyOneTrewa(t *testing.T) {
	engine := testEngine(t)
	// The Pangui on 7 jumps over the Trewa on 6 and lands on 5.
	state := stateWith(map[board.NodeID]board.Piece{
		7: board.PiecePangui,
		6: board.PieceTrewa,
	}, PlayerPangui)

	moves := engine.LegalMoves(state, PlayerPangui)
	if !hasMove(moves, 7, 5, MoveCapture) {
		t.Fatalf("the Pangui cannot capture from 7 to 5: %v", moves)
	}

	next, err := engine.ApplyMove(state, Move{
		Player:       PlayerPangui,
		From:         7,
		To:           5,
		Type:         MoveCapture,
		CapturedNode: 6,
	})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}

	if next.Position.At(5) != board.PiecePangui {
		t.Error("the Pangui did not land on 5")
	}
	if next.Position.At(6) != board.PieceEmpty {
		t.Error("the jumped Trewa is still on 6")
	}
	if next.Position.At(7) != board.PieceEmpty {
		t.Error("the Pangui is still on 7")
	}
	if next.CapturedTrewa != 1 {
		t.Errorf("captured = %d, want 1", next.CapturedTrewa)
	}
	if _, trewa := next.Position.Counts(); trewa != 0 {
		t.Errorf("trewa = %d, want 0", trewa)
	}
}

func TestCaptureIsRejectedWhenTheLandingNodeIsOccupied(t *testing.T) {
	engine := testEngine(t)
	state := stateWith(map[board.NodeID]board.Piece{
		7: board.PiecePangui,
		6: board.PieceTrewa,
		5: board.PieceTrewa,
	}, PlayerPangui)

	for _, move := range engine.LegalMoves(state, PlayerPangui) {
		if move.Type == MoveCapture && move.To == 5 {
			t.Error("the Pangui may not capture onto an occupied node")
		}
	}
}

func TestCaptureIsRejectedWhenTheLineBends(t *testing.T) {
	engine := testEngine(t)
	// 7 -> 11 -> 10 bends, so it is not a jump. The same Trewa can still
	// be jumped straight down the column: 7 -> 11 -> 15.
	state := stateWith(map[board.NodeID]board.Piece{
		7:  board.PiecePangui,
		11: board.PieceTrewa,
	}, PlayerPangui)

	for _, move := range engine.LegalMoves(state, PlayerPangui) {
		if move.Type == MoveCapture && move.To == 10 {
			t.Errorf("the Pangui may not jump a line that bends: %v", move)
		}
	}
	if !hasMove(engine.LegalMoves(state, PlayerPangui), 7, 15, MoveCapture) {
		t.Errorf("the straight jump from 7 over 11 to 15 is missing: %v", engine.LegalMoves(state, PlayerPangui))
	}
}

func TestCaptureIsRejectedOverAnEmptyNode(t *testing.T) {
	engine := testEngine(t)
	state := stateWith(map[board.NodeID]board.Piece{
		7: board.PiecePangui,
	}, PlayerPangui)

	for _, move := range engine.LegalMoves(state, PlayerPangui) {
		if move.Type == MoveCapture {
			t.Errorf("the Pangui may not jump over an empty node: %v", move)
		}
	}
}

func TestConsecutiveCapturesUpdateThePositionAndTheCounter(t *testing.T) {
	engine := testEngine(t)
	// The Pangui on 20 jumps over 19 to 18, then over 23 to 28.
	state := stateWith(map[board.NodeID]board.Piece{
		20: board.PiecePangui,
		19: board.PieceTrewa,
		23: board.PieceTrewa,
	}, PlayerPangui)

	first, err := engine.ApplyMove(state, Move{
		Player: PlayerPangui, From: 20, To: 18, Type: MoveCapture, CapturedNode: 19,
	})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if first.CapturedTrewa != 1 {
		t.Errorf("captured = %d, want 1", first.CapturedTrewa)
	}
	if !first.CaptureSequenceActive {
		t.Error("the capture sequence should still be active")
	}
	if first.CurrentPlayer != PlayerPangui {
		t.Errorf("current player = %q, want the Pangui to keep playing", first.CurrentPlayer)
	}

	second, err := engine.ApplyMove(first, Move{
		Player: PlayerPangui, From: 18, To: 28, Type: MoveCapture, CapturedNode: 23,
	})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if second.CapturedTrewa != 2 {
		t.Errorf("captured = %d, want 2", second.CapturedTrewa)
	}
	if second.CaptureSequenceActive {
		t.Error("the capture sequence should be over")
	}
	if second.CurrentPlayer != PlayerTrewa {
		t.Errorf("current player = %q, want the Trewa", second.CurrentPlayer)
	}
	if second.Position.At(28) != board.PiecePangui {
		t.Error("the Pangui did not land on 28")
	}
	if _, trewa := second.Position.Counts(); trewa != 0 {
		t.Errorf("trewa = %d, want 0", trewa)
	}
}

func TestMandatoryCaptureDoesNotSwitchPlayers(t *testing.T) {
	engine := testEngine(t)
	state := stateWith(map[board.NodeID]board.Piece{
		20: board.PiecePangui,
		19: board.PieceTrewa,
		23: board.PieceTrewa,
	}, PlayerPangui)

	next, err := engine.ApplyMove(state, Move{
		Player: PlayerPangui, From: 20, To: 18, Type: MoveCapture, CapturedNode: 19,
	})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if next.CurrentPlayer != PlayerPangui {
		t.Errorf("current player = %q, want the Pangui", next.CurrentPlayer)
	}
	if next.TurnNumber != state.TurnNumber {
		t.Errorf("turn = %d, want it to stay at %d", next.TurnNumber, state.TurnNumber)
	}
}

func TestCaptureIsPreferredOverANormalMove(t *testing.T) {
	engine := testEngine(t)
	state := stateWith(map[board.NodeID]board.Piece{
		7: board.PiecePangui,
		6: board.PieceTrewa,
	}, PlayerPangui)

	moves := engine.LegalMoves(state, PlayerPangui)
	for _, move := range moves {
		if move.Type == MoveNormal {
			t.Errorf("the Pangui must capture when it can: %v", moves)
		}
	}
	if !hasMove(moves, 7, 5, MoveCapture) {
		t.Errorf("the capture from 7 to 5 is missing: %v", moves)
	}

	// With the preference off, the normal move is legal again.
	ruleset := DefaultRuleset()
	ruleset.CapturePreferred = false
	other, err := NewEngine(engine.board, ruleset)
	if err != nil {
		t.Fatalf("NewEngine(): %v", err)
	}
	moves = other.LegalMoves(state, PlayerPangui)
	if !hasMove(moves, 7, 2, MoveNormal) {
		t.Errorf("without the preference the Pangui may move normally: %v", moves)
	}
}

func TestPanguiWinsAtTheThreshold(t *testing.T) {
	b, _, err := board.Standard()
	if err != nil {
		t.Fatalf("board.Standard(): %v", err)
	}
	ruleset := DefaultRuleset()
	ruleset.PanguiCapturesToWin = 1
	engine, err := NewEngine(b, ruleset)
	if err != nil {
		t.Fatalf("NewEngine(): %v", err)
	}

	state := stateWith(map[board.NodeID]board.Piece{
		7: board.PiecePangui,
		6: board.PieceTrewa,
	}, PlayerPangui)

	next, err := engine.ApplyMove(state, Move{
		Player: PlayerPangui, From: 7, To: 5, Type: MoveCapture, CapturedNode: 6,
	})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if next.GameStatus != StatusPanguiWon {
		t.Errorf("status = %q, want %q", next.GameStatus, StatusPanguiWon)
	}
	if next.Winner != PlayerPangui {
		t.Errorf("winner = %q, want %q", next.Winner, PlayerPangui)
	}
}

func TestTrewaWinWhenThePanguiIsBlocked(t *testing.T) {
	engine := testEngine(t)
	// The Pangui on 0 has only two neighbours, 1 and 5. Both hold Trewa,
	// and the node beyond 1 holds a Trewa too, so no jump is possible.
	state := stateWith(map[board.NodeID]board.Piece{
		0: board.PiecePangui,
		1: board.PieceTrewa,
		2: board.PieceTrewa,
		5: board.PieceTrewa,
	}, PlayerTrewa)

	evaluated := engine.Evaluate(state)
	if evaluated.GameStatus != StatusTrewaWon {
		t.Errorf("status = %q, want %q", evaluated.GameStatus, StatusTrewaWon)
	}
	if evaluated.Winner != PlayerTrewa {
		t.Errorf("winner = %q, want %q", evaluated.Winner, PlayerTrewa)
	}
}

func TestTriangleVictoryIsNotApplicable(t *testing.T) {
	// The canonical board has no designated triangle area, so the rule
	// stays off and the engine does not implement it. A variant that
	// enables it needs a board with that area defined first.
	ruleset := DefaultRuleset()
	if ruleset.PuTrewaWinByOccupyingTriangle {
		t.Error("the triangle rule should be off by default")
	}
}

func TestIllegalMoveLeavesTheStateUnchanged(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()
	before := state.Clone()

	// The Pangui may not jump over an empty node.
	if _, err := engine.ApplyMove(state, Move{
		Player: PlayerPangui, From: 7, To: 5, Type: MoveCapture, CapturedNode: 6,
	}); err == nil {
		t.Error("an illegal capture was accepted")
	}

	// The Trewa may not move on the Pangui's turn.
	if _, err := engine.ApplyMove(state, Move{
		Player: PlayerTrewa, From: 30, To: 35, Type: MoveNormal,
	}); err == nil {
		t.Error("a move out of turn was accepted")
	}

	if !reflect.DeepEqual(before, state) {
		t.Error("the state changed after an illegal move")
	}
}

func TestApplyMoveDoesNotMutateTheState(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()
	before := state.Clone()

	moves := engine.LegalMoves(state, PlayerPangui)
	if len(moves) == 0 {
		t.Fatal("no legal moves")
	}
	if _, err := engine.ApplyMove(state, moves[0]); err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if !reflect.DeepEqual(before, state) {
		t.Error("the state was modified by ApplyMove")
	}
}

func TestNoMovesAfterTheGameEnds(t *testing.T) {
	engine := testEngine(t)
	state := stateWith(map[board.NodeID]board.Piece{
		7: board.PiecePangui,
		6: board.PieceTrewa,
	}, PlayerPangui)

	next, err := engine.ApplyMove(state, Move{
		Player: PlayerPangui, From: 7, To: 5, Type: MoveCapture, CapturedNode: 6,
	})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}

	if moves := engine.LegalMoves(next, PlayerPangui); len(moves) != 0 {
		t.Errorf("the game is over but there are still moves: %v", moves)
	}
	if _, err := engine.ApplyMove(next, Move{
		Player: PlayerPangui, From: 5, To: 6, Type: MoveNormal,
	}); err == nil {
		t.Error("a move after the end of the game was accepted")
	}
}

func TestEveryLegalMoveCanBeApplied(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()

	for _, move := range engine.LegalMoves(state, PlayerPangui) {
		next, err := engine.ApplyMove(state, move)
		if err != nil {
			t.Errorf("ApplyMove(%v): %v", move, err)
			continue
		}
		if next.Position.At(move.To) != board.PiecePangui {
			t.Errorf("the Pangui is not on %d after %v", move.To, move)
		}
	}

	trewaTurn := state
	trewaTurn.CurrentPlayer = PlayerTrewa
	for _, move := range engine.LegalMoves(trewaTurn, PlayerTrewa) {
		next, err := engine.ApplyMove(trewaTurn, move)
		if err != nil {
			t.Errorf("ApplyMove(%v): %v", move, err)
			continue
		}
		if next.Position.At(move.To) != board.PieceTrewa {
			t.Errorf("the Trewa is not on %d after %v", move.To, move)
		}
	}
}

func TestTrewaMayNotMoveBackwardWhenTheRuleSaysSo(t *testing.T) {
	b, _, err := board.Standard()
	if err != nil {
		t.Fatalf("board.Standard(): %v", err)
	}
	ruleset := DefaultRuleset()
	ruleset.TrewaCanMoveBackward = false
	engine, err := NewEngine(b, ruleset)
	if err != nil {
		t.Fatalf("NewEngine(): %v", err)
	}

	state := engine.NewGame()
	for _, move := range engine.LegalMoves(state, PlayerTrewa) {
		from, ok1 := engine.board.Coord(move.From)
		to, ok2 := engine.board.Coord(move.To)
		if !ok1 || !ok2 {
			t.Fatalf("move %v leaves the board", move)
		}
		if to.Y >= from.Y {
			t.Errorf("the Trewa moved backwards: %v", move)
		}
	}
}

func TestRandomPlayoutsReachATerminalState(t *testing.T) {
	engine := testEngine(t)
	random := rand.New(rand.NewSource(20261021))

	for game := 0; game < 200; game++ {
		state := engine.NewGame()
		steps := 0
		for !state.GameStatus.Over() {
			moves := engine.LegalMoves(state, state.CurrentPlayer)
			if len(moves) == 0 {
				// A player with no legal move ends the game.
				state = engine.Evaluate(state)
				if !state.GameStatus.Over() {
					t.Fatalf("game %d: no legal move but the game is not over: %+v", game, state)
				}
				break
			}
			move := moves[random.Intn(len(moves))]
			next, err := engine.ApplyMove(state, move)
			if err != nil {
				t.Fatalf("game %d: ApplyMove(%v): %v", game, move, err)
			}
			state = next
			steps++
			if steps > 500 {
				t.Fatalf("game %d: the game did not end", game)
			}
		}

		if !state.GameStatus.Over() {
			t.Fatalf("game %d: the game ended without a status", game)
		}
		if len(state.MoveHistory) != steps {
			t.Errorf("game %d: %d moves played, %d in the history", game, steps, len(state.MoveHistory))
		}
		if pangui, trewa := state.Position.Counts(); pangui != 1 {
			t.Errorf("game %d: %d Pangui on the board", game, pangui)
		} else if trewa > 12 {
			t.Errorf("game %d: %d Trewa on the board", game, trewa)
		}
	}
}

func TestStateJSONRoundTrip(t *testing.T) {
	engine := testEngine(t)
	state := engine.NewGame()

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}

	var decoded GameState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	if !reflect.DeepEqual(state, decoded) {
		t.Errorf("the state did not survive the round trip:\n%v\n%v", state, decoded)
	}

	// The position is stored as a readable map from node id to piece name.
	var raw struct {
		Position map[string]string `json:"position"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	if raw.Position["7"] != "PANGUI" {
		t.Errorf("position[7] = %q, want PANGUI", raw.Position["7"])
	}
	if raw.Position["23"] != "TREWA" {
		t.Errorf("position[23] = %q, want TREWA", raw.Position["23"])
	}
}
