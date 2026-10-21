package httpapi

import (
	"errors"
	"net/http"

	"github.com/taolicus/komikan/internal/board"
)

// handleRoomBoard renders the board of a room in its current position.
//
// The game page draws the board in the browser from the board graph, so this
// endpoint is the fallback for a plain image of the position.
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
	state, err := s.decodeState(game.State)
	if err != nil {
		s.writeError(w, err)
		return
	}

	position := board.Position{}
	for id, piece := range state.Position {
		position[id] = piece
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write([]byte(s.board.RenderSVG(position))); err != nil {
		s.log.Error("write board", "error", err)
	}
}
