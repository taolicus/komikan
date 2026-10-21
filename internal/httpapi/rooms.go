package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/taolicus/komikan/internal/board"
	"github.com/taolicus/komikan/internal/store"
)

// seatInfo describes one seat of a room.
type seatInfo struct {
	Seat     int    `json:"seat"`
	PlayerID string `json:"playerId"`
	Role     string `json:"role,omitempty"`
}

// gameInfo describes the game state of a room.
type gameInfo struct {
	Version int             `json:"version"`
	Status  string          `json:"status"`
	Winner  string          `json:"winner,omitempty"`
	State   json.RawMessage `json:"state"`
}

// roomSummary is the JSON shape of a room.
type roomSummary struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Seats     []seatInfo `json:"seats"`
	Game      *gameInfo  `json:"game,omitempty"`
	// You describes the seat of the requesting player, if it has one.
	You *seatInfo `json:"you,omitempty"`
	// URL is the shareable path of the room.
	URL string `json:"url"`
}

// roomList is the JSON shape of the room list.
type roomList struct {
	Rooms []roomSummary `json:"rooms"`
}

// handleCreateRoom creates a room with the creator in seat 1.
func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errors.New("httpapi: no player on the request"))
		return
	}

	room, err := s.store.CreateRoom(r.Context(), store.NewID(), player.ID,
		store.State(`{}`), s.initialState())
	if err != nil {
		s.writeError(w, err)
		return
	}

	summary := s.summarize(r, room, player.ID)
	s.writeJSON(w, http.StatusCreated, summary)
}

// handleJoinRoom seats the player in the free seat of a room.
func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errors.New("httpapi: no player on the request"))
		return
	}

	roomID := r.PathValue("roomID")
	room, err := s.store.JoinRoom(r.Context(), roomID, player.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	summary := s.summarize(r, room, player.ID)
	s.writeJSON(w, http.StatusOK, summary)
}

// handleListRooms lists the rooms the requesting player is seated in.
func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errors.New("httpapi: no player on the request"))
		return
	}

	rooms, err := s.store.RoomsForPlayer(r.Context(), player.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	list := roomList{Rooms: make([]roomSummary, 0, len(rooms))}
	for _, room := range rooms {
		list.Rooms = append(list.Rooms, s.summarize(r, room, player.ID))
	}
	s.writeJSON(w, http.StatusOK, list)
}

// handleGetRoom returns the summary of a room to one of its players.
func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
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

	s.writeJSON(w, http.StatusOK, s.summarize(r, room, player.ID))
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
	if game, err := s.store.Game(r.Context(), room.ID); err == nil {
		summary.Game = &gameInfo{
			Version: game.Version,
			Status:  game.Status,
			Winner:  game.Winner,
			State:   game.State,
		}
	}
	return summary
}

// initialState builds the game state of a new room from the standard
// starting position. The rules engine replaces this once it lands.
func (s *Server) initialState() store.State {
	position := make(map[string]string, len(s.startPosition))
	for id, piece := range s.startPosition {
		name := "TREWA"
		if piece == board.PiecePangui {
			name = "PANGUI"
		}
		position[strconv.Itoa(int(id))] = name
	}
	data, err := json.Marshal(map[string]any{
		"position": position,
		"turn":     "PANGUI",
		"captured": 0,
	})
	if err != nil {
		// The state is built from plain values, so this cannot fail.
		panic("httpapi: build initial state: " + err.Error())
	}
	return data
}
