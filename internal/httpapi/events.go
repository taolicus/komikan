package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/taolicus/komikan/internal/store"
)

// eventHub tells the listeners of a room that the room changed.
//
// It carries no game data: a listener that wakes up reads the room again and
// renders it for the player it serves. So a move costs one read per connected
// player and never depends on how fast a browser is.
//
// The hub lives in the process. The service runs as a single process against
// one SQLite file and every write goes through it, so the hub sees every
// change; a deployment with more than one process would need a shared bus
// instead.
type eventHub struct {
	mu     sync.Mutex
	topics map[string]map[chan struct{}]struct{}
}

// newEventHub returns an empty hub.
func newEventHub() *eventHub {
	return &eventHub{topics: make(map[string]map[chan struct{}]struct{})}
}

// subscribe registers a listener for a room and returns the channel it is
// woken on, together with the function that removes it again.
//
// The channel holds one pending signal: a listener that is still working
// through the previous change is not queued twice, because reading the room
// once is enough to see the latest state.
func (h *eventHub) subscribe(roomID string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)

	h.mu.Lock()
	if h.topics[roomID] == nil {
		h.topics[roomID] = make(map[chan struct{}]struct{})
	}
	h.topics[roomID][ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		listeners := h.topics[roomID]
		delete(listeners, ch)
		if len(listeners) == 0 {
			delete(h.topics, roomID)
		}
		h.mu.Unlock()
	}
}

// publish tells every listener of a room that something changed. It never
// blocks.
func (h *eventHub) publish(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.topics[roomID] {
		select {
		case ch <- struct{}{}:
		default:
			// A signal is already pending, and one is enough.
		}
	}
}

// eventPingPeriod is how often an idle stream sends a comment. A comment
// keeps a proxy from treating the stream as a dead connection.
const eventPingPeriod = 25 * time.Second

// handleRoomEvents streams the state of a room whenever it changes.
//
// It is a Server-Sent Events stream. The payload of every event is the same
// JSON document the GET of the room returns, so the client has one shape to
// render whether the state arrived as a response or as an event.
//
// The first event is sent as soon as the client is connected, so a page never
// has to make a second request to get what it is missing.
func (s *Server) handleRoomEvents(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errForbidden)
		return
	}
	roomID := r.PathValue("roomID")
	room, err := s.store.GetRoom(r.Context(), roomID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if _, seated := room.Seat(player.ID); !seated {
		s.writeError(w, errForbidden)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, errors.New("httpapi: the response cannot be streamed"))
		return
	}

	// Subscribe before reading the room: a change that lands between the
	// first read and the subscription would otherwise be lost.
	events, unsubscribe := s.hub.subscribe(roomID)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	// nginx buffers a proxied response by default, which would hold the
	// events back. The header turns that off for this response only.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if !s.sendRoomEvent(w, r, player.ID, roomID) {
		return
	}
	flusher.Flush()

	ping := time.NewTicker(eventPingPeriod)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-events:
			if !s.sendRoomEvent(w, r, player.ID, roomID) {
				return
			}
			flusher.Flush()
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// sendRoomEvent writes the current state of a room as one event. It returns
// false when the stream should end: the room is gone, the player lost their
// seat, or the write failed.
func (s *Server) sendRoomEvent(w http.ResponseWriter, r *http.Request, playerID, roomID string) bool {
	room, err := s.store.GetRoom(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Tell the page the room is gone rather than leaving it waiting
			// for an event that will never come.
			fmt.Fprint(w, "event: gone\ndata: {}\n\n")
			return false
		}
		s.log.Error("read room for an event", "room", roomID, "error", err)
		return false
	}
	if _, seated := room.Seat(playerID); !seated {
		return false
	}

	data, err := json.Marshal(s.summarize(r, room, playerID))
	if err != nil {
		s.log.Error("encode room event", "room", roomID, "error", err)
		return false
	}
	// A JSON document has no newline in it, so it fits one data line.
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return false
	}
	return true
}
