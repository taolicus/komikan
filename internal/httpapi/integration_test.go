package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/taolicus/komikan/internal/rules"
	"github.com/taolicus/komikan/internal/store"
)

// These tests cover the behaviour of the whole server: two browsers, rooms
// that must not see each other, a service that restarts, and two submissions
// that arrive at the same moment. The requirements they stand for are the
// ones in docs/spec.md §11 that the unit tests of the packages below cannot
// reach.

// testServer is a running API server together with the store it uses, so a
// test can stop it and start another one against the same database file.
type testServer struct {
	*httptest.Server
	db      store.Store
	stopped bool
}

// startTestServer starts the API against the database at path. An empty path
// opens a private in-memory database.
func startTestServer(t *testing.T, path string) *testServer {
	t.Helper()

	db, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open(%q): %v", path, err)
	}
	api, err := New(Config{Version: "test", Store: db})
	if err != nil {
		db.Close()
		t.Fatalf("httpapi.New(): %v", err)
	}

	srv := &testServer{Server: httptest.NewServer(api.Handler()), db: db}
	t.Cleanup(srv.stop)
	return srv
}

// stop closes the server and the database, the way a service restart does.
func (s *testServer) stop() {
	if s.stopped {
		return
	}
	s.stopped = true
	s.Server.Close()
	s.db.Close()
}

// playMove submits a move and returns the status code and the body.
func playMove(t *testing.T, client *testClient, roomID string, from, to int, version *int) (int, []byte) {
	t.Helper()

	body, err := json.Marshal(moveRequest{From: from, To: to, Version: version})
	if err != nil {
		t.Fatalf("encode move: %v", err)
	}
	resp, respBody := client.do(http.MethodPost, "/api/rooms/"+roomID+"/moves", bytes.NewReader(body))
	return resp.StatusCode, respBody
}

// errorCode reads the machine-readable code out of an error body.
func errorCode(t *testing.T, body []byte) string {
	t.Helper()

	var failure struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &failure); err != nil {
		t.Fatalf("decode error body: %v (body %s)", err, body)
	}
	return failure.Code
}

// listRooms returns the rooms a browser is seated in.
func listRooms(t *testing.T, client *testClient) []roomSummary {
	t.Helper()

	resp, body := client.do(http.MethodGet, "/api/rooms", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, body)
	}
	var list roomList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return list.Rooms
}

// TestRoomsDoNotAffectEachOther proves the room isolation requirement of
// docs/spec.md §4: a move in one room leaves every other room alone.
func TestRoomsDoNotAffectEachOther(t *testing.T) {
	srv := newTestAPI(t)
	first := newPlayingPair(t, srv)
	second := newPlayingPair(t, srv)

	before := second.roomFor(t, second.pangui)

	state := first.roomFor(t, first.pangui)
	if len(state.LegalMoves) == 0 {
		t.Fatal("the Pangui has no legal move")
	}
	move := state.LegalMoves[0]
	status, body := playMove(t, first.pangui, first.room.ID, move.From, move.To, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusOK, body)
	}

	after := second.roomFor(t, second.pangui)
	if after.Game.Version != before.Game.Version {
		t.Errorf("the other room is at version %d, want %d", after.Game.Version, before.Game.Version)
	}
	if after.Game.CurrentPlayer != before.Game.CurrentPlayer {
		t.Errorf("the other room is at turn %q, want %q", after.Game.CurrentPlayer, before.Game.CurrentPlayer)
	}
	if !reflect.DeepEqual(after.Game.Position, before.Game.Position) {
		t.Errorf("the other room's position changed: %v, want %v", after.Game.Position, before.Game.Position)
	}
	if after.Game.LastMove != nil {
		t.Errorf("the other room reports a move: %+v", after.Game.LastMove)
	}
	if after.Status != before.Status {
		t.Errorf("the other room is %q, want %q", after.Status, before.Status)
	}

	// Each player only sees the rooms they are seated in.
	rooms := listRooms(t, second.pangui)
	if len(rooms) != 1 || rooms[0].ID != second.room.ID {
		t.Errorf("the player is seated in %v, want only %s", rooms, second.room.ID)
	}
}

// TestTheSeatIsRestoredOnReload proves the requirement of docs/spec.md §4:
// reopening the room URL with the same browser token restores the seat
// instead of taking a new one.
func TestTheSeatIsRestoredOnReload(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	joiner := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)

	_, body = joiner.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)
	joined := decodeRoom(t, body)

	// The joiner reloads the page.
	resp, body := joiner.do(http.MethodGet, "/api/rooms/"+room.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, body)
	}
	seen := decodeRoom(t, body)

	if seen.You == nil {
		t.Fatal("the seat was lost on reload")
	}
	if seen.You.Seat != joined.You.Seat {
		t.Errorf("seat = %d, want %d", seen.You.Seat, joined.You.Seat)
	}
	if seen.You.Role != joined.You.Role {
		t.Errorf("role = %q, want %q", seen.You.Role, joined.You.Role)
	}
	if len(seen.Seats) != 2 {
		t.Errorf("seats = %d, want 2", len(seen.Seats))
	}
}

// TestTheGameSurvivesARestartOfTheServer proves the persistence and recovery
// requirement of docs/spec.md §7 and §11: the seats and the position are
// intact after the service comes back, and the game carries on.
func TestTheGameSurvivesARestartOfTheServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "komikan.db")
	srv := startTestServer(t, path)
	pair := newPlayingPair(t, srv.Server)

	state := pair.roomFor(t, pair.pangui)
	if len(state.LegalMoves) == 0 {
		t.Fatal("the Pangui has no legal move")
	}
	move := state.LegalMoves[0]
	status, body := playMove(t, pair.pangui, pair.room.ID, move.From, move.To, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusOK, body)
	}
	played := decodeRoom(t, body)

	// The service restarts against the same database file.
	srv.stop()
	restarted := startTestServer(t, path)
	pair.pangui.pointAt(restarted.URL)
	pair.trewa.pointAt(restarted.URL)

	// The same browser still holds the same seat...
	seen := pair.roomFor(t, pair.pangui)
	if seen.You == nil {
		t.Fatal("the seat was lost in the restart")
	}
	if seen.You.Seat != played.You.Seat || seen.You.Role != played.You.Role {
		t.Errorf("you = %+v, want seat %d with role %s", seen.You, played.You.Seat, played.You.Role)
	}

	// ... and the game is where it was left.
	if seen.Game.Version != played.Game.Version {
		t.Errorf("version = %d, want %d", seen.Game.Version, played.Game.Version)
	}
	if !reflect.DeepEqual(seen.Game.Position, played.Game.Position) {
		t.Errorf("position = %v, want %v", seen.Game.Position, played.Game.Position)
	}

	// The game carries on: it is the other player's turn, and the move the
	// server refused before the restart is still refused.
	stale := played.Game.Version - 1
	status, body = playMove(t, pair.trewa, pair.room.ID, 0, 0, &stale)
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusConflict, body)
	}
	if code := errorCode(t, body); code != "STALE_VERSION" {
		t.Errorf("code = %q, want STALE_VERSION", code)
	}

	trewa := pair.roomFor(t, pair.trewa)
	if trewa.Game.CurrentPlayer != string(rules.PlayerTrewa) {
		t.Fatalf("current player = %q, want TREWA", trewa.Game.CurrentPlayer)
	}
	if len(trewa.LegalMoves) == 0 {
		t.Fatal("the Trewa has no legal move")
	}
	next := trewa.LegalMoves[0]
	status, body = playMove(t, pair.trewa, pair.room.ID, next.From, next.To, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusOK, body)
	}
}

// TestAPlayerCanMoveWhileTheOpponentIsAbsent proves the asynchronous play
// requirement of docs/spec.md §6: the opponent does not have to be online,
// and the server does not play on their behalf.
func TestAPlayerCanMoveWhileTheOpponentIsAbsent(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	state := pair.roomFor(t, pair.pangui)
	if len(state.LegalMoves) == 0 {
		t.Fatal("the Pangui has no legal move")
	}
	move := state.LegalMoves[0]
	status, body := playMove(t, pair.pangui, pair.room.ID, move.From, move.To, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusOK, body)
	}
	played := decodeRoom(t, body)

	// The other browser sends nothing at all. The game must wait for it:
	// the server does not play for an absent player, and there is no
	// timeout.
	for i := 0; i < 5; i++ {
		seen := pair.roomFor(t, pair.pangui)
		if seen.Game.Version != played.Game.Version {
			t.Fatalf("the game moved on by itself: version %d, want %d", seen.Game.Version, played.Game.Version)
		}
		if seen.Game.CurrentPlayer != string(rules.PlayerTrewa) {
			t.Fatalf("current player = %q, want TREWA", seen.Game.CurrentPlayer)
		}
	}

	// The absent player comes back and plays.
	trewa := pair.roomFor(t, pair.trewa)
	if len(trewa.LegalMoves) == 0 {
		t.Fatal("the Trewa has no legal move")
	}
	next := trewa.LegalMoves[0]
	status, body = playMove(t, pair.trewa, pair.room.ID, next.From, next.To, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusOK, body)
	}
	after := decodeRoom(t, body)
	if after.Game.Version != played.Game.Version+1 {
		t.Errorf("version = %d, want %d", after.Game.Version, played.Game.Version+1)
	}
}

// TestOnlyOneOfTwoSimultaneousMovesIsAccepted proves the concurrency
// requirement of docs/spec.md §11: two submissions for the same turn leave
// the game with exactly one move played.
func TestOnlyOneOfTwoSimultaneousMovesIsAccepted(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	state := pair.roomFor(t, pair.pangui)
	if len(state.LegalMoves) == 0 {
		t.Fatal("the Pangui has no legal move")
	}
	move := state.LegalMoves[0]
	version := state.Game.Version

	// Two tabs of the same browser submit the same move at the same moment.
	// The barrier makes both requests leave at once, so the race is real
	// without depending on timing.
	start := make(chan struct{})
	var wg sync.WaitGroup
	statuses := make([]int, 2)
	bodies := make([][]byte, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			body, err := json.Marshal(moveRequest{From: move.From, To: move.To, Version: &version})
			if err != nil {
				bodies[i] = []byte(err.Error())
				return
			}
			resp, respBody, err := pair.pangui.try(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
			if err != nil {
				bodies[i] = []byte(err.Error())
				return
			}
			statuses[i] = resp.StatusCode
			bodies[i] = respBody
		}(i)
	}
	close(start)
	wg.Wait()

	accepted, refused := 0, 0
	for i, status := range statuses {
		switch status {
		case http.StatusOK:
			accepted++
		case http.StatusConflict:
			if code := errorCode(t, bodies[i]); code != "STALE_VERSION" {
				t.Errorf("the refused move has code %q, want STALE_VERSION", code)
			}
			refused++
		default:
			t.Errorf("status = %d, want %d or %d (body %s)", status, http.StatusOK, http.StatusConflict, bodies[i])
		}
	}
	if accepted != 1 || refused != 1 {
		t.Errorf("accepted = %d, refused = %d, want 1 and 1", accepted, refused)
	}

	// The game moved on by exactly one move.
	after := pair.roomFor(t, pair.pangui)
	if after.Game.Version != version+1 {
		t.Errorf("version = %d, want %d", after.Game.Version, version+1)
	}
}

// TestAMoveAfterTheGameIsOverIsRefused proves the endpoint validation
// requirement of docs/spec.md §11.
func TestAMoveAfterTheGameIsOverIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	final := playOut(t, pair)
	if final.Game.Status == "IN_PROGRESS" {
		t.Fatal("the game did not end")
	}
	if final.Status != string(store.RoomFinished) {
		t.Fatalf("room status = %q, want FINISHED", final.Status)
	}

	// The player whose turn it would have been tries one more move.
	client := pair.pangui
	if final.Game.CurrentPlayer == string(rules.PlayerTrewa) {
		client = pair.trewa
	}
	status, body := playMove(t, client, pair.room.ID, 7, 2, nil)
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusConflict, body)
	}
	if code := errorCode(t, body); code != "GAME_OVER" {
		t.Errorf("code = %q, want GAME_OVER", code)
	}
}

// TestAMoveInAnUnknownRoomIsNotFound proves the endpoint validation
// requirement of docs/spec.md §11.
func TestAMoveInAnUnknownRoomIsNotFound(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	status, body := playMove(t, client, "no-such-room", 7, 2, nil)
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusNotFound, body)
	}
	if code := errorCode(t, body); code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", code)
	}
}
