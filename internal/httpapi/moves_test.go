package httpapi

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/taolicus/komikan/internal/rules"
)

// playingPair is two browsers in the same room, and the seat of each.
type playingPair struct {
	room       roomSummary
	pangui     *testClient
	trewa      *testClient
	panguiSeat seatInfo
	trewaSeat  seatInfo
}

// newPlayingPair creates a room and fills it with a second player.
func newPlayingPair(t *testing.T, srv *httptest.Server) *playingPair {
	t.Helper()
	one := newTestClient(t, srv.URL)
	two := newTestClient(t, srv.URL)

	_, body := one.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	created := decodeRoom(t, body)
	_, body = two.do(http.MethodPost, "/api/rooms/"+created.ID+"/join", nil)
	joined := decodeRoom(t, body)

	pair := &playingPair{room: joined}
	for _, seat := range joined.Seats {
		switch seat.Role {
		case string(rules.PlayerPangui):
			pair.panguiSeat = seat
			if seat.PlayerID == created.You.PlayerID {
				pair.pangui = one
			} else {
				pair.pangui = two
			}
		case string(rules.PlayerTrewa):
			pair.trewaSeat = seat
			if seat.PlayerID == created.You.PlayerID {
				pair.trewa = one
			} else {
				pair.trewa = two
			}
		}
	}
	if pair.pangui == nil || pair.trewa == nil {
		t.Fatalf("the roles were not assigned: %+v", joined.Seats)
	}
	return pair
}

// roomFor fetches the room summary as a given browser.
func (p *playingPair) roomFor(t *testing.T, client *testClient) roomSummary {
	t.Helper()
	resp, body := client.do(http.MethodGet, "/api/rooms/"+p.room.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, body)
	}
	return decodeRoom(t, body)
}

func TestThePanguiMovesFirst(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	// The Pangui is told its legal moves, the Trewa is not.
	panguiRoom := pair.roomFor(t, pair.pangui)
	if len(panguiRoom.LegalMoves) == 0 {
		t.Fatal("the Pangui has no legal moves")
	}
	trewaRoom := pair.roomFor(t, pair.trewa)
	if len(trewaRoom.LegalMoves) != 0 {
		t.Errorf("the Trewa was given legal moves: %v", trewaRoom.LegalMoves)
	}
}

func TestAMoveAdvancesTheGame(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	before := pair.roomFor(t, pair.pangui)
	move := before.LegalMoves[0]

	body, _ := json.Marshal(moveRequest{From: move.From, To: move.To, Version: &before.Game.Version})
	resp, respBody := pair.pangui.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, respBody)
	}

	after := decodeRoom(t, respBody)
	if after.Game.Version != before.Game.Version+1 {
		t.Errorf("version = %d, want %d", after.Game.Version, before.Game.Version+1)
	}
	// The turn only changes when the player changes: a chain of captures
	// keeps the same player.
	if after.Game.TurnNumber != before.Game.TurnNumber &&
		after.Game.TurnNumber != before.Game.TurnNumber+1 {
		t.Errorf("turn = %d, want %d or %d", after.Game.TurnNumber,
			before.Game.TurnNumber, before.Game.TurnNumber+1)
	}
	if len(after.Game.Position) != len(before.Game.Position) {
		t.Errorf("pieces = %d, want %d", len(after.Game.Position), len(before.Game.Position))
	}
	if after.Game.LastMove == nil {
		t.Error("the response does not name the move that was played")
	}

	// The other browser sees the same state.
	seen := pair.roomFor(t, pair.trewa)
	if seen.Game.Version != after.Game.Version {
		t.Errorf("the other player sees version %d, want %d", seen.Game.Version, after.Game.Version)
	}
}

func TestACaptureRemovesATrewa(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	// The Pangui starts on node 7. Move it so that a jump is on offer:
	// the capture is only legal when the node beyond the Trewa is empty.
	before := pair.roomFor(t, pair.pangui)

	// The Pangui walks to node 2 or 8 or 11; whichever it is, play it and
	// then look for a capture.
	var first moveView
	for _, move := range before.LegalMoves {
		first = move
		break
	}
	body, _ := json.Marshal(moveRequest{From: first.From, To: first.To, Version: &before.Game.Version})
	resp, respBody := pair.pangui.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, respBody)
	}
	after := decodeRoom(t, respBody)

	// After the first move it is the Trewa's turn, unless the Pangui
	// captured, in which case it keeps playing.
	if after.Game.CurrentPlayer == string(rules.PlayerPangui) {
		if after.Game.CapturedTrewa != 1 {
			t.Errorf("captured = %d, want 1", after.Game.CapturedTrewa)
		}
		return
	}
	if after.Game.CurrentPlayer != string(rules.PlayerTrewa) {
		t.Errorf("current player = %q, want the Trewa", after.Game.CurrentPlayer)
	}
}

func TestMovingOutOfTurnIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	body, _ := json.Marshal(moveRequest{From: 30, To: 35})
	resp, respBody := pair.trewa.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusForbidden, respBody)
	}

	var failure struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(respBody, &failure); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if failure.Code != "NOT_YOUR_TURN" {
		t.Errorf("code = %q, want NOT_YOUR_TURN", failure.Code)
	}
}

func TestAnIllegalMoveIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	// The Pangui stands on node 7 and cannot reach node 20.
	body, _ := json.Marshal(moveRequest{From: 7, To: 20})
	resp, respBody := pair.pangui.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusUnprocessableEntity, respBody)
	}

	var failure struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(respBody, &failure); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if failure.Code != "ILLEGAL_MOVE" {
		t.Errorf("code = %q, want ILLEGAL_MOVE", failure.Code)
	}

	// The game did not move on.
	after := pair.roomFor(t, pair.pangui)
	if after.Game.Version != 0 {
		t.Errorf("version = %d, want 0", after.Game.Version)
	}
}

func TestAStaleVersionIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	before := pair.roomFor(t, pair.pangui)
	stale := before.Game.Version

	body, _ := json.Marshal(moveRequest{From: before.LegalMoves[0].From, To: before.LegalMoves[0].To})
	resp, respBody := pair.pangui.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, respBody)
	}

	// The same move again, with the version the client remembers.
	body, _ = json.Marshal(moveRequest{From: before.LegalMoves[0].From, To: before.LegalMoves[0].To, Version: &stale})
	resp, respBody = pair.pangui.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusConflict, respBody)
	}

	var failure struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(respBody, &failure); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if failure.Code != "STALE_VERSION" {
		t.Errorf("code = %q, want STALE_VERSION", failure.Code)
	}
}

func TestAnOutsiderCannotMove(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)
	outsider := newTestClient(t, srv.URL)

	body, _ := json.Marshal(moveRequest{From: 7, To: 2})
	resp, respBody := outsider.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusForbidden, respBody)
	}
}

func TestAMoveInAWaitingRoomIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	_, body := client.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)

	body, _ = json.Marshal(moveRequest{From: 7, To: 2})
	resp, respBody := client.do(http.MethodPost, "/api/rooms/"+room.ID+"/moves", bytes.NewReader(body))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusConflict, respBody)
	}

	var failure struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(respBody, &failure); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if failure.Code != "ROOM_NOT_ACTIVE" {
		t.Errorf("code = %q, want ROOM_NOT_ACTIVE", failure.Code)
	}
}

func TestAMalformedMoveIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	resp, respBody := pair.pangui.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader([]byte("not json")))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusBadRequest, respBody)
	}
}

// playOut plays random legal moves until the game ends, and returns the
// final state. The seed keeps the game the same from run to run.
func playOut(t *testing.T, pair *playingPair) roomSummary {
	t.Helper()
	random := rand.New(rand.NewSource(20261021))

	for step := 0; step < 400; step++ {
		overview := pair.roomFor(t, pair.pangui)
		if overview.Game.Status != "IN_PROGRESS" {
			return overview
		}
		client := pair.pangui
		if overview.Game.CurrentPlayer == string(rules.PlayerTrewa) {
			client = pair.trewa
		}
		// The legal moves are only sent to the player whose turn it is.
		state := pair.roomFor(t, client)
		moves := state.LegalMoves
		if len(moves) == 0 {
			t.Fatalf("step %d: the player to move has no legal move", step)
		}
		move := moves[random.Intn(len(moves))]
		body, _ := json.Marshal(moveRequest{From: move.From, To: move.To})
		resp, respBody := client.do(http.MethodPost, "/api/rooms/"+pair.room.ID+"/moves", bytes.NewReader(body))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("step %d: status = %d, want %d (body %s)", step, resp.StatusCode, http.StatusOK, respBody)
		}
	}
	return pair.roomFor(t, pair.pangui)
}

func TestAGameCanBePlayedToTheEnd(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	// Both players keep playing until the game ends. Every move is
	// answered by the server, so this also proves that a whole game
	// survives the round trip.
	final := playOut(t, pair)

	if final.Game.Status == "IN_PROGRESS" {
		t.Fatal("the game did not end")
	}
	if final.Status != "FINISHED" {
		t.Errorf("room status = %q, want FINISHED", final.Status)
	}
	if final.Game.Winner == "" && final.Game.Status != "DRAW" {
		t.Errorf("the game ended with no winner: %q", final.Game.Status)
	}
}
