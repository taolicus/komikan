package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/taolicus/komikan/internal/board"
	"github.com/taolicus/komikan/internal/rules"
	"github.com/taolicus/komikan/internal/store"
)

// nodeView is one node of the board, with the coordinates a client draws it
// at.
type nodeView struct {
	ID int `json:"id"`
	X  int `json:"x"`
	Y  int `json:"y"`
}

// boardView is the topology of the board. The client draws the board from
// this, so the picture on the screen is the graph the rules engine uses.
type boardView struct {
	Nodes []nodeView `json:"nodes"`
	Edges [][2]int   `json:"edges"`
}

// moveView is a legal move in the shape a client needs.
type moveView struct {
	From         int    `json:"from"`
	To           int    `json:"to"`
	Type         string `json:"type"`
	CapturedNode *int   `json:"capturedNode,omitempty"`
}

// gameView is the client-visible state of the game in a room.
type gameView struct {
	Version       int               `json:"version"`
	Status        string            `json:"status"`
	Winner        string            `json:"winner,omitempty"`
	TurnNumber    int               `json:"turnNumber"`
	CurrentPlayer string            `json:"currentPlayer"`
	CapturedTrewa int               `json:"capturedTrewa"`
	Threshold     int               `json:"captureThreshold"`
	Position      map[string]string `json:"position"`
	// CaptureChain is true while the Pangui may continue a chain of
	// captures.
	CaptureChain bool `json:"captureSequenceActive"`
	// LastMove is the move that was played last, when there is one.
	LastMove *moveView `json:"lastMove,omitempty"`
}

// summarize turns a room into its JSON shape for a given player.
func (s *Server) summarize(r *http.Request, room store.Room, playerID string) roomSummary {
	summary := roomSummary{
		ID:        room.ID,
		Status:    string(room.Status),
		CreatedAt: room.CreatedAt,
		UpdatedAt: room.UpdatedAt,
		Seats:     make([]seatInfo, 0, len(room.Seats)),
		URL:       "/rooms/" + room.ID,
		Board:     s.boardView(),
	}
	for _, seat := range room.Seats {
		summary.Seats = append(summary.Seats, seatInfo{
			Seat:     seat.Seat,
			PlayerID: seat.PlayerID,
			Role:     string(seat.Role),
		})
	}
	if seat, ok := room.Seat(playerID); ok {
		you := seatInfo{
			Seat:     seat.Seat,
			PlayerID: seat.PlayerID,
			Role:     string(seat.Role),
		}
		summary.You = &you
	}

	game, err := s.store.Game(r.Context(), room.ID)
	if err != nil {
		return summary
	}
	state, err := s.decodeState(game.State)
	if err != nil {
		s.log.Error("decode game state", "room", room.ID, "error", err)
		return summary
	}

	summary.Game = &gameView{
		Version:       game.Version,
		Status:        string(state.GameStatus),
		Winner:        string(state.Winner),
		TurnNumber:    state.TurnNumber,
		CurrentPlayer: string(state.CurrentPlayer),
		CapturedTrewa: state.CapturedTrewa,
		Threshold:     state.Ruleset.PanguiCapturesToWin,
		Position:      s.positionView(state),
		CaptureChain:  state.CaptureSequenceActive,
	}
	if len(state.MoveHistory) > 0 {
		last := moveViewOf(state.MoveHistory[len(state.MoveHistory)-1])
		summary.Game.LastMove = &last
	}

	// The legal moves belong to the player whose turn it is. Nobody else
	// needs them.
	if summary.You != nil && summary.You.Role != "" &&
		rules.Player(summary.You.Role) == state.CurrentPlayer &&
		!state.GameStatus.Over() {
		for _, move := range s.engine.LegalMoves(state, state.CurrentPlayer) {
			summary.LegalMoves = append(summary.LegalMoves, moveViewOf(move))
		}
	}
	return summary
}

// moveViewOf turns a move into its JSON shape.
func moveViewOf(move rules.Move) moveView {
	view := moveView{
		From: int(move.From),
		To:   int(move.To),
		Type: string(move.Type),
	}
	if move.Type == rules.MoveCapture {
		captured := int(move.CapturedNode)
		view.CapturedNode = &captured
	}
	return view
}

// positionView returns the position as a map from node id to piece name.
func (s *Server) positionView(state rules.GameState) map[string]string {
	out := make(map[string]string, len(state.Position))
	for id, piece := range state.Position {
		out[fmt.Sprintf("%d", id)] = pieceName(piece)
	}
	return out
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

// boardView returns the topology of the board.
func (s *Server) boardView() *boardView {
	nodes := s.board.Nodes()
	view := &boardView{
		Nodes: make([]nodeView, 0, len(nodes)),
		Edges: make([][2]int, 0, len(s.board.Edges())),
	}
	for _, node := range nodes {
		view.Nodes = append(view.Nodes, nodeView{ID: int(node.ID), X: node.Coord.X, Y: node.Coord.Y})
	}
	for _, edge := range s.board.Edges() {
		view.Edges = append(view.Edges, [2]int{int(edge[0]), int(edge[1])})
	}
	return view
}

// decodeState reads the game state of a room.
func (s *Server) decodeState(raw store.State) (rules.GameState, error) {
	var state rules.GameState
	if len(raw) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return rules.GameState{}, fmt.Errorf("httpapi: decode game state: %w", err)
	}
	return state, nil
}

// encodeState writes a game state.
func encodeState(state rules.GameState) (store.State, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("httpapi: encode game state: %w", err)
	}
	return store.State(data), nil
}

// errNotYourTurn means the player tried to move out of turn.
var errNotYourTurn = &apiError{
	status: http.StatusForbidden,
	code:   "NOT_YOUR_TURN",
	msg:    "it is not your turn",
}

// errIllegalMove means the move is not legal in the current position.
var errIllegalMove = &apiError{
	status: http.StatusUnprocessableEntity,
	code:   "ILLEGAL_MOVE",
	msg:    "illegal move",
}

// errStaleVersion means the client worked from an older state.
var errStaleVersion = &apiError{
	status: http.StatusConflict,
	code:   "STALE_VERSION",
	msg:    "the game moved on",
}

// errRoomNotActive means the room has not started yet.
var errRoomNotActive = &apiError{
	status: http.StatusConflict,
	code:   "ROOM_NOT_ACTIVE",
	msg:    "the game has not started",
}

// errGameOver means the game is finished.
var errGameOver = &apiError{
	status: http.StatusConflict,
	code:   "GAME_OVER",
	msg:    "the game is over",
}

// errMalformed means the request body could not be read.
var errMalformed = &apiError{
	status: http.StatusBadRequest,
	code:   "MALFORMED",
	msg:    "malformed request",
}

// apiError is an error that knows its status code and its machine-readable
// code.
type apiError struct {
	status int
	code   string
	msg    string
}

func (e *apiError) Error() string {
	return e.msg
}

// moveRequest is the body of a move submission.
type moveRequest struct {
	From int `json:"from"`
	To   int `json:"to"`
	// Version is the state version the client saw. When it is given and
	// does not match, the move is rejected.
	Version *int `json:"version"`
}

// handleSubmitMove plays a move submitted by a seat holder.
func (s *Server) handleSubmitMove(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errForbidden)
		return
	}
	ctx := r.Context()
	roomID := r.PathValue("roomID")

	room, err := s.store.GetRoom(ctx, roomID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	seat, seated := room.Seat(player.ID)
	if !seated {
		s.writeError(w, errForbidden)
		return
	}
	switch room.Status {
	case store.RoomFinished:
		// The game is over: say so, rather than claiming the room never
		// started.
		s.writeError(w, errGameOver)
		return
	case store.RoomActive:
	default:
		s.writeError(w, errRoomNotActive)
		return
	}

	var request moveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, errMalformed)
		return
	}

	game, err := s.store.Game(ctx, room.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if request.Version != nil && *request.Version != game.Version {
		s.writeError(w, errStaleVersion)
		return
	}

	state, err := s.decodeState(game.State)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if state.GameStatus.Over() {
		s.writeError(w, errGameOver)
		return
	}
	if rules.Player(seat.Role) != state.CurrentPlayer {
		s.writeError(w, errNotYourTurn)
		return
	}

	// The rules engine decides which moves are legal. The client only
	// says which node it wants to move to.
	var chosen *rules.Move
	for _, move := range s.engine.LegalMoves(state, state.CurrentPlayer) {
		if int(move.From) == request.From && int(move.To) == request.To {
			candidate := move
			chosen = &candidate
			break
		}
	}
	if chosen == nil {
		s.writeError(w, errIllegalMove)
		return
	}

	next, err := s.engine.ApplyMove(state, *chosen)
	if err != nil {
		s.writeError(w, errIllegalMove)
		return
	}

	encoded, err := encodeState(next)
	if err != nil {
		s.writeError(w, err)
		return
	}
	payload, err := json.Marshal(chosen)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if _, err := s.store.ApplyMove(ctx, room.ID, game.Version, player.ID, store.State(payload),
		store.GameResult{
			State:    encoded,
			Status:   string(next.GameStatus),
			Winner:   string(next.Winner),
			Finished: next.GameStatus.Over(),
		}); err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.writeError(w, errStaleVersion)
			return
		}
		s.writeError(w, err)
		return
	}

	room, err = s.store.GetRoom(ctx, room.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, s.summarize(r, room, player.ID))
}
