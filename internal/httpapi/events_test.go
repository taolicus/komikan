package httpapi

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/taolicus/komikan/internal/store"
)

// These tests cover the live updates of docs/spec.md §6: a page is told that
// the other player moved instead of asking the server again and again.

// eventStream is one open Server-Sent Events connection of a test browser.
type eventStream struct {
	resp   *http.Response
	reader *bufio.Reader
}

// openEventStream connects a browser to the events of a room.
func openEventStream(t *testing.T, client *testClient, roomID string) *eventStream {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, client.baseURL+"/api/rooms/"+roomID+"/events", nil)
	if err != nil {
		t.Fatalf("build the stream request: %v", err)
	}
	resp, err := client.client.Do(req)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type = %q, want text/event-stream", got)
	}
	if got := resp.Header.Get("X-Accel-Buffering"); got != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", got)
	}

	// The body is closed before the server, so the handler returns and the
	// test does not wait for a stream that never ends.
	t.Cleanup(func() { resp.Body.Close() })
	return &eventStream{resp: resp, reader: bufio.NewReader(resp.Body)}
}

// next waits for the next room event and decodes it. It fails the test when
// no event arrives in time.
func (s *eventStream) next(t *testing.T) roomSummary {
	t.Helper()

	type result struct {
		data []byte
		err  error
	}
	lines := make(chan result, 1)
	go func() {
		for {
			line, err := s.reader.ReadString('\n')
			if err != nil {
				lines <- result{err: err}
				return
			}
			if data, ok := strings.CutPrefix(line, "data: "); ok {
				lines <- result{data: []byte(strings.TrimSpace(data))}
				return
			}
		}
	}()

	select {
	case got := <-lines:
		if got.err != nil {
			t.Fatalf("read an event: %v", got.err)
		}
		var room roomSummary
		if err := json.Unmarshal(got.data, &room); err != nil {
			t.Fatalf("decode an event: %v (event %s)", err, got.data)
		}
		return room
	case <-time.After(5 * time.Second):
		t.Fatal("no event arrived")
		return roomSummary{}
	}
}

// TestTheStreamStartsWithTheRoom proves a page gets the room as soon as it
// connects, so it does not need a second request to fill itself in.
func TestTheStreamStartsWithTheRoom(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	stream := openEventStream(t, pair.trewa, pair.room.ID)
	first := stream.next(t)

	if first.ID != pair.room.ID {
		t.Errorf("room = %q, want %q", first.ID, pair.room.ID)
	}
	if first.You == nil || first.You.Role != string(pair.trewaSeat.Role) {
		t.Errorf("you = %+v, want the Trewa seat", first.You)
	}
	if first.Game == nil || first.Game.Version != pair.room.Game.Version {
		t.Fatalf("game = %+v, want version %d", first.Game, pair.room.Game.Version)
	}
}

// TestAMoveIsPushedToTheOtherPlayer proves the live update requirement: the
// page of the player who is waiting is told about the move, with its own
// legal moves already in it.
func TestAMoveIsPushedToTheOtherPlayer(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)

	stream := openEventStream(t, pair.trewa, pair.room.ID)
	connected := stream.next(t)

	state := pair.roomFor(t, pair.pangui)
	if len(state.LegalMoves) == 0 {
		t.Fatal("the Pangui has no legal move")
	}
	move := state.LegalMoves[0]
	status, body := playMove(t, pair.pangui, pair.room.ID, move.From, move.To, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", status, http.StatusOK, body)
	}

	pushed := stream.next(t)
	if pushed.Game == nil {
		t.Fatal("the pushed room has no game")
	}
	if pushed.Game.Version != connected.Game.Version+1 {
		t.Errorf("version = %d, want %d", pushed.Game.Version, connected.Game.Version+1)
	}
	if pushed.Game.CurrentPlayer != string(pair.trewaSeat.Role) {
		t.Errorf("current player = %q, want %q", pushed.Game.CurrentPlayer, pair.trewaSeat.Role)
	}
	// The event is rendered for the player it is sent to.
	if len(pushed.LegalMoves) == 0 {
		t.Error("the Trewa was told about the move but not given its own moves")
	}
}

// TestJoiningIsPushedToTheWaitingPlayer proves the room page of the first
// player stops waiting as soon as the second player arrives, without asking.
func TestJoiningIsPushedToTheWaitingPlayer(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	joiner := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", strings.NewReader("{}"))
	room := decodeRoom(t, body)
	if room.Status != string(store.RoomWaiting) {
		t.Fatalf("room status = %q, want WAITING", room.Status)
	}

	stream := openEventStream(t, creator, room.ID)
	waiting := stream.next(t)
	if waiting.Status != string(store.RoomWaiting) {
		t.Fatalf("status = %q, want WAITING", waiting.Status)
	}

	resp, body := joiner.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, body)
	}

	started := stream.next(t)
	if started.Status == string(store.RoomWaiting) {
		t.Error("the waiting player was not told that the game started")
	}
	if len(started.Seats) != 2 {
		t.Errorf("seats = %d, want 2", len(started.Seats))
	}
	if started.Game == nil || started.Game.Status != "IN_PROGRESS" {
		t.Errorf("game = %+v, want it in progress", started.Game)
	}
}

// TestTheStreamIsRefusedToAStranger proves the room isolation requirement of
// docs/spec.md §4: the stream of a room is only open to its players.
func TestTheStreamIsRefusedToAStranger(t *testing.T) {
	srv := newTestAPI(t)
	pair := newPlayingPair(t, srv)
	stranger := newTestClient(t, srv.URL)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/rooms/"+pair.room.ID+"/events", nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	resp, err := stranger.client.Do(req)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("content type = %q, want an error document", got)
	}
}
