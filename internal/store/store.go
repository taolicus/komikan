// Package store persists players, rooms, seats, games, and moves.
//
// The package exposes a domain interface and domain errors. Nothing above it
// sees SQL: the API layer deals in rooms, seats, and games.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Domain errors. The API layer maps these onto HTTP status codes.
var (
	// ErrNotFound means the requested record does not exist.
	ErrNotFound = errors.New("store: not found")
	// ErrConflict means the record exists but the request contradicts its
	// current state, such as a stale version or a full room.
	ErrConflict = errors.New("store: conflict")
	// ErrInvalid means the request was malformed.
	ErrInvalid = errors.New("store: invalid")
)

// State is the persisted game state. Until the rules engine lands it is an
// opaque JSON document, so the storage schema and the interface do not have
// to change when the real state type arrives.
type State = json.RawMessage

// Role is the side a seat plays.
type Role string

const (
	// RolePangui is the puma.
	RolePangui Role = "PANGUI"
	// RoleTrewa is the pack of dogs.
	RoleTrewa Role = "TREWA"
)

// Valid reports whether the role is one of the known roles.
func (r Role) Valid() bool {
	return r == RolePangui || r == RoleTrewa
}

// Player is a browser identity.
type Player struct {
	ID        string
	TokenHash string
	CreatedAt time.Time
}

// RoomStatus is the lifecycle state of a room.
type RoomStatus string

const (
	// RoomWaiting means one seat is taken and the game has not started.
	RoomWaiting RoomStatus = "WAITING"
	// RoomActive means both seats are taken and the game is running.
	RoomActive RoomStatus = "ACTIVE"
	// RoomFinished means the game is over.
	RoomFinished RoomStatus = "FINISHED"
)

// Seat is a player's place in a room.
type Seat struct {
	RoomID   string
	PlayerID string
	Seat     int
	// Role is empty until both seats are taken. Roles are assigned when the
	// second player joins.
	Role Role
}

// Room is a game room.
type Room struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    RoomStatus
	Ruleset   State
	Seats     []Seat
}

// Seat returns the seat held by a player, if any.
func (r Room) Seat(playerID string) (Seat, bool) {
	for _, s := range r.Seats {
		if s.PlayerID == playerID {
			return s, true
		}
	}
	return Seat{}, false
}

// Game is the persisted state of the game in a room.
type Game struct {
	RoomID    string
	State     State
	Version   int
	Status    string
	Winner    string
	UpdatedAt time.Time
}

// Move is one recorded move.
type Move struct {
	RoomID    string
	Seq       int
	PlayerID  string
	Payload   State
	CreatedAt time.Time
}

// GameResult carries the state of a game after a move.
type GameResult struct {
	State  State
	Status string
	Winner string
	// Finished tells the store that the game is over, so the room is
	// marked as finished as well.
	Finished bool
}

// Store is the persistence interface the API layer uses.
type Store interface {
	// CreatePlayer records a new browser identity.
	CreatePlayer(ctx context.Context, tokenHash string) (Player, error)
	// PlayerByTokenHash finds the identity behind a token hash.
	PlayerByTokenHash(ctx context.Context, tokenHash string) (Player, error)

	// CreateRoom creates a room with one seat for the creator and an initial
	// game state.
	CreateRoom(ctx context.Context, roomID, creatorID string, ruleset State, initial State) (Room, error)
	// GetRoom returns a room with its seats.
	GetRoom(ctx context.Context, roomID string) (Room, error)
	// RoomsForPlayer lists the rooms a player is seated in.
	RoomsForPlayer(ctx context.Context, playerID string) ([]Room, error)
	// JoinRoom seats a player in the free seat, or returns their existing
	// seat. Roles are assigned once both seats are taken.
	JoinRoom(ctx context.Context, roomID, playerID string) (Room, error)

	// Game returns the game state of a room.
	Game(ctx context.Context, roomID string) (Game, error)
	// ApplyMove stores a move and the resulting state atomically. It fails
	// with ErrConflict when the expected version is stale.
	ApplyMove(ctx context.Context, roomID string, expectedVersion int, playerID string, payload State, result GameResult) (Game, error)
	// Moves lists the moves of a room in order.
	Moves(ctx context.Context, roomID string) ([]Move, error)

	// Close releases the database.
	Close() error
}
