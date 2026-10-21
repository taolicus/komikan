package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/taolicus/komikan/internal/board"
	"github.com/taolicus/komikan/internal/store"
)

// handleRoomBoard renders the board of a room in its current position.
func (s *Server) handleRoomBoard(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errors.New("httpapi: no player on the request"))
		return
	}

	room, err := s.store.GetRoom(r.Context(), r.PathValue("roomID"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	if _, seated := room.Seat(player.ID); !seated {
		s.writeError(w, errForbidden)
		return
	}

	game, err := s.store.Game(r.Context(), room.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	position, err := s.positionFromState(game.State)
	if err != nil {
		s.writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write([]byte(s.board.RenderSVG(position))); err != nil {
		s.log.Error("write board", "error", err)
	}
}

// positionFromState reads the position out of a game state document.
//
// The shape of the state is a placeholder until the rules engine lands: it
// carries the position as a map from node id to piece name.
func (s *Server) positionFromState(state store.State) (board.Position, error) {
	var doc struct {
		Position map[string]string `json:"position"`
	}
	if err := json.Unmarshal(state, &doc); err != nil {
		return nil, fmt.Errorf("httpapi: decode game state: %w", err)
	}

	position := board.Position{}
	for key, name := range doc.Position {
		id, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		switch name {
		case "PANGUI":
			position[board.NodeID(id)] = board.PiecePangui
		case "TREWA":
			position[board.NodeID(id)] = board.PieceTrewa
		}
	}
	return position, nil
}
