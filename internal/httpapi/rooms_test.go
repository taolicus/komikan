package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/taolicus/komikan/internal/rules"
	"github.com/taolicus/komikan/internal/store"
)

// testClient is one browser: its own cookie jar, its own identity.
type testClient struct {
	t       *testing.T
	client  *http.Client
	baseURL string
}

func newTestClient(t *testing.T, baseURL string) *testClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New(): %v", err)
	}
	return &testClient{t: t, client: &http.Client{Jar: jar}, baseURL: baseURL}
}

// do performs a request and returns the response and its body. State
// changing requests carry the CSRF token, as the browser client would.
func (c *testClient) do(method, path string, body io.Reader) (*http.Response, []byte) {
	c.t.Helper()

	resp, data, err := c.try(method, path, body)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp, data
}

// try performs a request and reports what went wrong instead of failing the
// test. It is the form to use from a goroutine, where a failing test would
// not stop the test cleanly.
func (c *testClient) try(method, path string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, nil, err
	}
	if method != http.MethodGet {
		// Browsers send Origin on every write.
		req.Header.Set("Origin", c.baseURL)
		if token := c.csrfToken(); token != "" {
			req.Header.Set("X-CSRF-Token", token)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return resp, data, nil
}

// pointAt moves the browser to another server, as a browser does when the
// service comes back on a new address. The cookie jar follows the host, so
// the identity is kept.
func (c *testClient) pointAt(baseURL string) {
	c.baseURL = baseURL
}

// csrfToken returns the CSRF token this browser holds.
func (c *testClient) csrfToken() string {
	c.t.Helper()
	base, err := url.Parse(c.baseURL)
	if err != nil {
		c.t.Fatalf("parse base url: %v", err)
	}
	for _, cookie := range c.client.Jar.Cookies(base) {
		if cookie.Name == csrfCookieName {
			return cookie.Value
		}
	}
	return ""
}

// sessionToken returns the raw session token this browser holds.
func (c *testClient) sessionToken() string {
	c.t.Helper()
	base, err := url.Parse(c.baseURL)
	if err != nil {
		c.t.Fatalf("parse base url: %v", err)
	}
	for _, cookie := range c.client.Jar.Cookies(base) {
		if cookie.Name == sessionCookieName {
			return cookie.Value
		}
	}
	return ""
}

func decodeRoom(t *testing.T, data []byte) roomSummary {
	t.Helper()
	var room roomSummary
	if err := json.Unmarshal(data, &room); err != nil {
		t.Fatalf("decode room: %v (body %s)", err, data)
	}
	return room
}

// newTestAPI starts a server with an in-memory store.
func newTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := store.Open("")
	if err != nil {
		t.Fatalf("store.Open(): %v", err)
	}
	api, err := New(Config{Version: "test", Store: db})
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	srv := httptest.NewServer(api.Handler())
	t.Cleanup(func() {
		srv.Close()
		db.Close()
	})
	return srv
}

func TestFirstVisitGetsAnIdentity(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	resp, _ := client.do(http.MethodGet, "/api/rooms", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if client.sessionToken() == "" {
		t.Fatal("no session cookie was set")
	}
	if client.csrfToken() == "" {
		t.Fatal("no CSRF cookie was set")
	}

	// The session cookie must not be readable by scripts.
	for _, cookie := range resp.Cookies() {
		if cookie.Name == sessionCookieName && !cookie.HttpOnly {
			t.Error("the session cookie is not HttpOnly")
		}
	}
}

func TestIdentitySurvivesAReload(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	client.do(http.MethodGet, "/api/rooms", nil)
	token := client.sessionToken()

	client.do(http.MethodGet, "/api/rooms", nil)
	if client.sessionToken() != token {
		t.Error("the identity changed on reload")
	}
}

func TestCreateRoomSeatsTheCreator(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	resp, body := client.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusCreated, body)
	}

	room := decodeRoom(t, body)
	if room.ID == "" {
		t.Error("the room has no id")
	}
	if room.Status != string(store.RoomWaiting) {
		t.Errorf("status = %q, want %q", room.Status, store.RoomWaiting)
	}
	if len(room.Seats) != 1 {
		t.Fatalf("seats = %d, want 1", len(room.Seats))
	}
	if room.You == nil || room.You.Seat != 1 {
		t.Errorf("you = %+v, want seat 1", room.You)
	}
	if room.Game == nil {
		t.Fatal("the room has no game state")
	}
	if len(room.Game.Position) != 13 {
		t.Errorf("pieces = %d, want 13", len(room.Game.Position))
	}
	if room.Game.Position["7"] != "PANGUI" {
		t.Errorf("the Pangui is not on node 7: %v", room.Game.Position)
	}
	if room.Game.CurrentPlayer != string(rules.PlayerPangui) {
		t.Errorf("current player = %q, want PANGUI", room.Game.CurrentPlayer)
	}
	if room.Game.Threshold != 8 {
		t.Errorf("threshold = %d, want 8", room.Game.Threshold)
	}
	if room.Board == nil || len(room.Board.Nodes) != 38 {
		t.Errorf("the room does not carry the board topology")
	}
}

func TestTwoIdentitiesGetDifferentRooms(t *testing.T) {
	srv := newTestAPI(t)
	one := newTestClient(t, srv.URL)
	two := newTestClient(t, srv.URL)

	_, first := one.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	_, second := two.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))

	roomOne := decodeRoom(t, first)
	roomTwo := decodeRoom(t, second)
	if roomOne.ID == "" || roomTwo.ID == "" {
		t.Fatalf("a room was not created: %s / %s", first, second)
	}
	if roomOne.ID == roomTwo.ID {
		t.Error("two rooms share the same id")
	}
	if roomOne.You == nil || roomTwo.You == nil {
		t.Fatal("the responses do not say which seat the caller holds")
	}
	if roomOne.You.PlayerID == roomTwo.You.PlayerID {
		t.Error("two identities share the same player id")
	}
}

func TestJoinRoomAssignsRoles(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	joiner := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)

	resp, body := joiner.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusOK, body)
	}

	joined := decodeRoom(t, body)
	if joined.Status != string(store.RoomActive) {
		t.Errorf("status = %q, want %q", joined.Status, store.RoomActive)
	}
	if len(joined.Seats) != 2 {
		t.Fatalf("seats = %d, want 2", len(joined.Seats))
	}

	roles := map[string]bool{}
	for _, seat := range joined.Seats {
		roles[seat.Role] = true
	}
	if !roles[string(store.RolePangui)] || !roles[string(store.RoleTrewa)] {
		t.Errorf("roles = %v, want one Pangui and one Trewa", roles)
	}
}

func TestJoiningTwiceRestoresTheSeat(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	joiner := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)

	_, body = joiner.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)
	joined := decodeRoom(t, body)

	// The joiner comes back: the seat is restored, not a new one allocated.
	_, body = joiner.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)
	again := decodeRoom(t, body)
	if len(again.Seats) != 2 {
		t.Errorf("seats = %d, want 2", len(again.Seats))
	}
	if again.You.Seat != joined.You.Seat {
		t.Errorf("seat = %d, want %d", again.You.Seat, joined.You.Seat)
	}
}

func TestAThirdPlayerIsRefused(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	joiner := newTestClient(t, srv.URL)
	third := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)
	joiner.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)

	resp, body := third.do(http.MethodPost, "/api/rooms/"+room.ID+"/join", nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %s)", resp.StatusCode, http.StatusConflict, body)
	}
}

func TestGetRoomNeedsASeat(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	outsider := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)

	resp, _ := creator.do(http.MethodGet, "/api/rooms/"+room.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status for a seat holder = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	resp, body = outsider.do(http.MethodGet, "/api/rooms/"+room.ID, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status for an outsider = %d, want %d (body %s)", resp.StatusCode, http.StatusForbidden, body)
	}
}

func TestUnknownRoomIsNotFound(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	resp, _ := client.do(http.MethodGet, "/api/rooms/nope", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestListRoomsOnlyShowsOwnRooms(t *testing.T) {
	srv := newTestAPI(t)
	one := newTestClient(t, srv.URL)
	two := newTestClient(t, srv.URL)

	one.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	one.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	two.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))

	_, body := one.do(http.MethodGet, "/api/rooms", nil)
	var list roomList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Rooms) != 2 {
		t.Errorf("rooms = %d, want 2", len(list.Rooms))
	}

	_, body = two.do(http.MethodGet, "/api/rooms", nil)
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Rooms) != 1 {
		t.Errorf("rooms = %d, want 1", len(list.Rooms))
	}
}

func TestWritesWithoutTheCSRFHeaderAreRefused(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	// Take a cookie first, but send no CSRF header.
	client.do(http.MethodGet, "/api/rooms", nil)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/rooms", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.client.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestRoomsPageIsServed(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	for _, path := range []string{"/rooms/", "/rooms/whatever"} {
		resp, body := client.do(http.MethodGet, path, nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status for %s = %d, want %d", path, resp.StatusCode, http.StatusOK)
			continue
		}
		if !strings.Contains(string(body), "Komikan") {
			t.Errorf("the page for %s does not mention the game", path)
		}
	}
}

func TestRoomBoardNeedsASeat(t *testing.T) {
	srv := newTestAPI(t)
	creator := newTestClient(t, srv.URL)
	outsider := newTestClient(t, srv.URL)

	_, body := creator.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	room := decodeRoom(t, body)

	resp, _ := outsider.do(http.MethodGet, "/api/rooms/"+room.ID+"/board", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status for an outsider = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	resp, body = creator.do(http.MethodGet, "/api/rooms/"+room.ID+"/board", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status for a seat holder = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/svg+xml") {
		t.Errorf("content type = %q, want image/svg+xml", ct)
	}
	if !strings.Contains(string(body), "<svg") {
		t.Error("the board is not an SVG")
	}
	if !strings.Contains(string(body), "128006") {
		t.Error("the board does not show the Pangui")
	}
}

func TestTheRawTokenNeverAppearsInAResponse(t *testing.T) {
	srv := newTestAPI(t)
	client := newTestClient(t, srv.URL)

	_, body := client.do(http.MethodPost, "/api/rooms", bytes.NewReader([]byte("{}")))
	token := client.sessionToken()
	if token == "" {
		t.Fatal("no session token")
	}
	if strings.Contains(string(body), token) {
		t.Error("the raw session token appears in a response body")
	}
	if strings.Contains(string(body), token[:8]) {
		t.Error("part of the raw session token appears in a response body")
	}
}
