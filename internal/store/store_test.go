package store

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func newStore(t *testing.T) Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "komikan.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newPlayer(t *testing.T, s Store) Player {
	t.Helper()
	p, err := s.CreatePlayer(context.Background(), "hash-"+NewID())
	if err != nil {
		t.Fatalf("CreatePlayer(): %v", err)
	}
	return p
}

func TestCreatePlayerAndLookup(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	p := newPlayer(t, s)
	got, err := s.PlayerByTokenHash(ctx, p.TokenHash)
	if err != nil {
		t.Fatalf("PlayerByTokenHash(): %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("player id = %q, want %q", got.ID, p.ID)
	}

	if _, err := s.PlayerByTokenHash(ctx, "unknown"); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestCreateRoomSeatsTheCreator(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{"panguiCapturesToWin":6}`), State(`{"turn":1}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}
	if room.Status != RoomWaiting {
		t.Errorf("status = %q, want %q", room.Status, RoomWaiting)
	}
	if len(room.Seats) != 1 {
		t.Fatalf("seats = %d, want 1", len(room.Seats))
	}
	if room.Seats[0].PlayerID != creator.ID || room.Seats[0].Seat != 1 {
		t.Errorf("seat = %+v, want the creator in seat 1", room.Seats[0])
	}
	if room.Seats[0].Role != "" {
		t.Errorf("role = %q, want it unset until the second player joins", room.Seats[0].Role)
	}

	game, err := s.Game(ctx, room.ID)
	if err != nil {
		t.Fatalf("Game(): %v", err)
	}
	if string(game.State) != `{"turn":1}` {
		t.Errorf("state = %s, want the initial state", game.State)
	}
}

func TestJoinRoomAssignsBothRoles(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)
	joiner := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}

	room, err = s.JoinRoom(ctx, room.ID, joiner.ID)
	if err != nil {
		t.Fatalf("JoinRoom(): %v", err)
	}
	if room.Status != RoomActive {
		t.Errorf("status = %q, want %q", room.Status, RoomActive)
	}
	if len(room.Seats) != 2 {
		t.Fatalf("seats = %d, want 2", len(room.Seats))
	}

	roles := map[Role]bool{}
	for _, seat := range room.Seats {
		if !seat.Role.Valid() {
			t.Errorf("seat %d has role %q, want a known role", seat.Seat, seat.Role)
		}
		roles[seat.Role] = true
	}
	if !roles[RolePangui] || !roles[RoleTrewa] {
		t.Errorf("roles = %v, want one Pangui and one Trewa", roles)
	}
}

func TestJoinRoomRestoresAnExistingSeat(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)
	joiner := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}
	if _, err := s.JoinRoom(ctx, room.ID, joiner.ID); err != nil {
		t.Fatalf("JoinRoom(): %v", err)
	}

	// The creator comes back to the room: their seat is restored, not a new
	// one is allocated.
	room, err = s.JoinRoom(ctx, room.ID, creator.ID)
	if err != nil {
		t.Fatalf("JoinRoom() by the creator: %v", err)
	}
	if len(room.Seats) != 2 {
		t.Errorf("seats = %d, want 2", len(room.Seats))
	}
	if seat, ok := room.Seat(creator.ID); !ok || seat.Seat != 1 {
		t.Errorf("creator seat = %+v, want seat 1", seat)
	}
}

func TestJoinRoomRejectsAThirdPlayer(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)
	joiner := newPlayer(t, s)
	third := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}
	if _, err := s.JoinRoom(ctx, room.ID, joiner.ID); err != nil {
		t.Fatalf("JoinRoom(): %v", err)
	}

	if _, err := s.JoinRoom(ctx, room.ID, third.ID); !errors.Is(err, ErrConflict) {
		t.Errorf("error = %v, want ErrConflict", err)
	}
}

func TestRoomsForPlayerOnlyListsItsOwnRooms(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	one := newPlayer(t, s)
	two := newPlayer(t, s)

	first, err := s.CreateRoom(ctx, NewID(), one.ID, State(`{}`), State(`{}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}
	second, err := s.CreateRoom(ctx, NewID(), one.ID, State(`{}`), State(`{}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}

	rooms, err := s.RoomsForPlayer(ctx, one.ID)
	if err != nil {
		t.Fatalf("RoomsForPlayer(): %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("rooms = %d, want 2", len(rooms))
	}
	seen := map[string]bool{first.ID: true, second.ID: true}
	for _, r := range rooms {
		if !seen[r.ID] {
			t.Errorf("unexpected room %s", r.ID)
		}
	}

	rooms, err = s.RoomsForPlayer(ctx, two.ID)
	if err != nil {
		t.Fatalf("RoomsForPlayer(): %v", err)
	}
	if len(rooms) != 0 {
		t.Errorf("rooms = %d, want 0 for a player with no seats", len(rooms))
	}
}

func TestApplyMoveUpdatesStateAndHistory(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{"turn":1}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}

	game, err := s.ApplyMove(ctx, room.ID, 0, creator.ID,
		State(`{"from":7,"to":6}`),
		GameResult{State: State(`{"turn":2}`), Status: "ACTIVE"})
	if err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if game.Version != 1 {
		t.Errorf("version = %d, want 1", game.Version)
	}
	if string(game.State) != `{"turn":2}` {
		t.Errorf("state = %s, want the new state", game.State)
	}

	moves, err := s.Moves(ctx, room.ID)
	if err != nil {
		t.Fatalf("Moves(): %v", err)
	}
	if len(moves) != 1 {
		t.Fatalf("moves = %d, want 1", len(moves))
	}
	if string(moves[0].Payload) != `{"from":7,"to":6}` {
		t.Errorf("payload = %s, want the move that was applied", moves[0].Payload)
	}
}

func TestApplyMoveWithAStaleVersionConflicts(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{"turn":1}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}
	if _, err := s.ApplyMove(ctx, room.ID, 0, creator.ID, State(`{"a":1}`),
		GameResult{State: State(`{"turn":2}`), Status: "ACTIVE"}); err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}

	// The same version again: the second writer must be rejected and the
	// stored state must not change.
	if _, err := s.ApplyMove(ctx, room.ID, 0, creator.ID, State(`{"a":2}`),
		GameResult{State: State(`{"turn":3}`), Status: "ACTIVE"}); !errors.Is(err, ErrConflict) {
		t.Errorf("error = %v, want ErrConflict", err)
	}

	game, err := s.Game(ctx, room.ID)
	if err != nil {
		t.Fatalf("Game(): %v", err)
	}
	if game.Version != 1 {
		t.Errorf("version = %d, want 1", game.Version)
	}
	if string(game.State) != `{"turn":2}` {
		t.Errorf("state = %s, want the state of the first move", game.State)
	}
}

func TestConcurrentMovesOnlyOneWins(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	creator := newPlayer(t, s)

	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{"turn":1}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}

	const writers = 8
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.ApplyMove(ctx, room.ID, 0, creator.ID,
				State(`{"n":`+string(rune('0'+i))+`}`),
				GameResult{State: State(`{"turn":2}`), Status: "ACTIVE"})
			switch {
			case err == nil:
				mu.Lock()
				succeeded++
				mu.Unlock()
			case errors.Is(err, ErrConflict):
			default:
				t.Errorf("ApplyMove(): unexpected error %v", err)
			}
		}(i)
	}
	wg.Wait()

	if succeeded != 1 {
		t.Errorf("succeeded = %d, want exactly 1", succeeded)
	}
	game, err := s.Game(ctx, room.ID)
	if err != nil {
		t.Fatalf("Game(): %v", err)
	}
	if game.Version != 1 {
		t.Errorf("version = %d, want 1", game.Version)
	}
	moves, err := s.Moves(ctx, room.ID)
	if err != nil {
		t.Fatalf("Moves(): %v", err)
	}
	if len(moves) != 1 {
		t.Errorf("moves = %d, want 1", len(moves))
	}
}

func TestStateSurvivesReopeningTheStore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "komikan.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	creator := newPlayer(t, s)
	room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{"turn":1}`))
	if err != nil {
		t.Fatalf("CreateRoom(): %v", err)
	}
	if _, err := s.ApplyMove(ctx, room.ID, 0, creator.ID, State(`{"from":7}`),
		GameResult{State: State(`{"turn":2}`), Status: "ACTIVE"}); err != nil {
		t.Fatalf("ApplyMove(): %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("Open() again: %v", err)
	}
	defer reopened.Close()

	game, err := reopened.Game(ctx, room.ID)
	if err != nil {
		t.Fatalf("Game(): %v", err)
	}
	if game.Version != 1 || string(game.State) != `{"turn":2}` {
		t.Errorf("game = %+v, want version 1 and the stored state", game)
	}
	moves, err := reopened.Moves(ctx, room.ID)
	if err != nil {
		t.Fatalf("Moves(): %v", err)
	}
	if len(moves) != 1 {
		t.Errorf("moves = %d, want 1", len(moves))
	}
}

func TestRolesAreAssignedAtRandom(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		creator := newPlayer(t, s)
		joiner := newPlayer(t, s)
		room, err := s.CreateRoom(ctx, NewID(), creator.ID, State(`{}`), State(`{}`))
		if err != nil {
			t.Fatalf("CreateRoom(): %v", err)
		}
		room, err = s.JoinRoom(ctx, room.ID, joiner.ID)
		if err != nil {
			t.Fatalf("JoinRoom(): %v", err)
		}
		seat1, _ := room.Seat(creator.ID)
		seen[string(seat1.Role)] = true
	}

	if !seen[string(RolePangui)] || !seen[string(RoleTrewa)] {
		t.Errorf("seat 1 roles seen = %v, want both roles to come up", seen)
	}
}

func TestUnknownRoomIsNotFound(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if _, err := s.GetRoom(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}
