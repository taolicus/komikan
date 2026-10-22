// The room page decides which move a click means. With twelve dogs a point
// can be reached by more than one of them, so the piece the player picked
// has to decide.
//
// This test drives the page script of web/rooms.html with a fake DOM and a
// stubbed fetch, and looks at the move request that reaches the server.
//
// Run it with: node --test "web/**/*.test.js"

const test = require("node:test");
const assert = require("node:assert");
const fs = require("node:fs");
const path = require("node:path");

const {FakeDocument, extractScript, tick} = require("./fakedom.js");

const PAGE = path.join(__dirname, "rooms.html");
const CELL = 64;
const MARGIN = 48;

// The canonical board, as the server sends it: [id, x, y] and [from, to].
const NODES = [
  [0,0,0],
  [1,1,0],
  [2,2,0],
  [3,3,0],
  [4,4,0],
  [5,0,1],
  [6,1,1],
  [7,2,1],
  [8,3,1],
  [9,4,1],
  [10,1,2],
  [11,2,2],
  [12,3,2],
  [13,0,3],
  [14,1,3],
  [15,2,3],
  [16,3,3],
  [17,4,3],
  [18,0,4],
  [19,1,4],
  [20,2,4],
  [21,3,4],
  [22,4,4],
  [23,0,5],
  [24,1,5],
  [25,2,5],
  [26,3,5],
  [27,4,5],
  [28,0,6],
  [29,1,6],
  [30,2,6],
  [31,3,6],
  [32,4,6],
  [33,0,7],
  [34,1,7],
  [35,2,7],
  [36,3,7],
  [37,4,7]
];
const EDGES = [
  [0,1],
  [0,5],
  [1,2],
  [1,6],
  [2,3],
  [2,7],
  [3,4],
  [3,8],
  [4,9],
  [5,6],
  [5,10],
  [6,7],
  [6,10],
  [7,8],
  [7,11],
  [8,9],
  [8,12],
  [9,12],
  [10,11],
  [10,15],
  [11,12],
  [11,15],
  [12,15],
  [13,14],
  [13,18],
  [13,19],
  [14,15],
  [14,19],
  [15,16],
  [15,19],
  [15,20],
  [15,21],
  [16,17],
  [16,21],
  [17,21],
  [17,22],
  [18,19],
  [18,23],
  [19,20],
  [19,23],
  [19,24],
  [19,25],
  [20,21],
  [20,25],
  [21,22],
  [21,25],
  [21,26],
  [21,27],
  [22,27],
  [23,24],
  [23,28],
  [23,29],
  [24,25],
  [24,29],
  [25,26],
  [25,29],
  [25,30],
  [25,31],
  [26,27],
  [26,31],
  [27,31],
  [27,32],
  [28,29],
  [28,33],
  [29,30],
  [29,33],
  [29,34],
  [29,35],
  [30,31],
  [30,35],
  [31,32],
  [31,35],
  [31,36],
  [31,37],
  [32,37],
  [33,34],
  [34,35],
  [35,36],
  [36,37]
];

// The starting position: the puma on node 7, the twelve dogs below it.
const PIECES = {
  7: "PANGUI",
  23: "TREWA", 27: "TREWA", 28: "TREWA", 29: "TREWA", 30: "TREWA", 31: "TREWA",
  32: "TREWA", 33: "TREWA", 34: "TREWA", 35: "TREWA", 36: "TREWA", 37: "TREWA",
};

// The moves the dogs may play in that position. Node 24 is reachable by the
// dog on 23 and by the dog on 29; node 25 by three of them.
const TREWA_MOVES = [
  {from: 23, to: 18, type: "NORMAL"},
  {from: 23, to: 19, type: "NORMAL"},
  {from: 23, to: 24, type: "NORMAL"},
  {from: 27, to: 21, type: "NORMAL"},
  {from: 27, to: 22, type: "NORMAL"},
  {from: 27, to: 26, type: "NORMAL"},
  {from: 29, to: 24, type: "NORMAL"},
  {from: 29, to: 25, type: "NORMAL"},
  {from: 30, to: 25, type: "NORMAL"},
  {from: 31, to: 25, type: "NORMAL"},
  {from: 31, to: 26, type: "NORMAL"},
];

// The moves the puma may play in that position.
const PANGUI_MOVES = [
  {from: 7, to: 2, type: "NORMAL"},
  {from: 7, to: 6, type: "NORMAL"},
  {from: 7, to: 8, type: "NORMAL"},
  {from: 7, to: 11, type: "NORMAL"},
];

// ---- the page under test ---------------------------------------------

// FakeEventSource is the stream the page opens for a room. The test keeps the
// instances, so it can push an event the way the server would.
class FakeEventSource {
  constructor(url) {
    this.url = url;
    this.readyState = FakeEventSource.OPEN;
    this.onmessage = null;
    this.onerror = null;
    FakeEventSource.instances.push(this);
  }

  close() {
    this.readyState = FakeEventSource.CLOSED;
  }
}
FakeEventSource.CONNECTING = 0;
FakeEventSource.OPEN = 1;
FakeEventSource.CLOSED = 2;
FakeEventSource.instances = [];

// summary builds the room the server would send to a player.
function summary(role, legalMoves) {
  const other = role === "PANGUI" ? "TREWA" : "PANGUI";
  return {
    id: "testroom",
    status: "ACTIVE",
    seats: [
      {seat: 1, playerId: "player-one", role},
      {seat: 2, playerId: "player-two", role: other},
    ],
    you: {seat: 1, playerId: "player-one", role},
    game: {
      version: 3,
      status: "IN_PROGRESS",
      turnNumber: 3,
      currentPlayer: role,
      capturedTrewa: 0,
      captureThreshold: 8,
      position: PIECES,
      captureSequenceActive: false,
    },
    board: {
      nodes: NODES.map(([id, x, y]) => ({id, x, y})),
      edges: EDGES,
    },
    legalMoves,
    url: "/rooms/testroom",
  };
}

// loadPage runs the script of the room page against a fake DOM and returns
// the document it drew on and the requests it made.
function loadPage(data) {
  const document = new FakeDocument();
  const calls = [];
  const fetch = async (path, options) => {
    calls.push({
      path,
      body: options && options.body ? JSON.parse(options.body) : null,
    });
    return {status: 200, text: async () => JSON.stringify(data)};
  };
  const location = {
    pathname: "/rooms/testroom",
    origin: "https://example.test",
    href: "https://example.test/rooms/testroom",
  };
  const navigator = {clipboard: {writeText: async () => {}}};
  // The page listens for the room instead of asking for it; the test pushes
  // the events itself.
  FakeEventSource.instances = [];
  const EventSource = FakeEventSource;
  const intervals = [];
  const setInterval = (fn, ms) => {
    intervals.push(ms);
    return intervals.length;
  };

  const source = fs.readFileSync(PAGE, "utf8");
  const script = extractScript(source);
  eval(script);

  return {document, calls, streams: FakeEventSource.instances, intervals};
}

// loadListPage runs the script of the room page as the list of rooms, the
// page at /rooms/.
//
// The list is the first call of the page, and it is the call that gives the
// browser its identity. It is held back until the test releases it, so the
// order of the calls can be looked at.
function loadListPage() {
  const document = new FakeDocument();
  const calls = [];
  let releaseList = () => {};
  const listReady = new Promise((resolve) => {
    releaseList = resolve;
  });
  const fetch = async (path, options) => {
    const method = options && options.method ? options.method : "GET";
    calls.push({
      path,
      method,
      headers: (options && options.headers) || {},
      body: options && options.body ? JSON.parse(options.body) : null,
    });
    if (method === "POST") {
      return {status: 201, text: async () => JSON.stringify({id: "newroom"})};
    }
    await listReady;
    return {status: 200, text: async () => JSON.stringify({rooms: []})};
  };
  const location = {
    pathname: "/rooms/",
    origin: "https://example.test",
    href: "https://example.test/rooms/",
  };
  const navigator = {clipboard: {writeText: async () => {}}};
  const setInterval = () => 0;

  const source = fs.readFileSync(PAGE, "utf8");
  const script = extractScript(source);
  eval(script);

  return {document, calls, releaseList: () => releaseList()};
}

// clickNode clicks the point of the board.
function clickNode(document, id) {
  const node = NODES.find(([nodeID]) => nodeID === id);
  const cx = MARGIN + node[1] * CELL;
  const cy = MARGIN + node[2] * CELL;
  const hits = document.registry.filter(
    (element) =>
      element.attributes.class === "hit" &&
      Number(element.attributes.cx) === cx &&
      Number(element.attributes.cy) === cy
  );
  assert.strictEqual(hits.length, 1, "the board has one point on node " + id);
  for (const handler of hits[0].listeners.click) handler();
}

// moveRequests returns the moves the page sent to the server.
function moveRequests(calls) {
  return calls.filter((call) => call.path.endsWith("/moves"));
}

// ---- the tests --------------------------------------------------------

test("the dog that was picked is the dog that moves", async () => {
  const page = loadPage(summary("TREWA", TREWA_MOVES));
  await tick();

  clickNode(page.document, 29);
  clickNode(page.document, 24); // also reachable by the dog on node 23
  await tick();

  const moves = moveRequests(page.calls);
  assert.strictEqual(moves.length, 1);
  assert.strictEqual(moves[0].body.from, 29);
  assert.strictEqual(moves[0].body.to, 24);
});

test("a point the picked dog cannot reach plays nothing", async () => {
  const page = loadPage(summary("TREWA", TREWA_MOVES));
  await tick();

  clickNode(page.document, 29); // the dog on 29 reaches 24 and 25
  clickNode(page.document, 18); // only the dog on 23 reaches 18
  await tick();

  assert.strictEqual(moveRequests(page.calls).length, 0);
});

test("picking a dog plays no move yet", async () => {
  const page = loadPage(summary("TREWA", TREWA_MOVES));
  await tick();

  clickNode(page.document, 29);
  await tick();

  assert.strictEqual(moveRequests(page.calls).length, 0);
});

test("the puma plays the move that was picked", async () => {
  const page = loadPage(summary("PANGUI", PANGUI_MOVES));
  await tick();

  clickNode(page.document, 7);
  clickNode(page.document, 11);
  await tick();

  const moves = moveRequests(page.calls);
  assert.strictEqual(moves.length, 1);
  assert.strictEqual(moves[0].body.from, 7);
  assert.strictEqual(moves[0].body.to, 11);
});

test("the link is shared while the room waits for the second player", async () => {
  const data = summary("TREWA", TREWA_MOVES);
  data.status = "WAITING";
  const page = loadPage(data);
  await tick();

  const share = page.document.getElementById("share");
  assert.strictEqual(share.classList.contains("hidden"), false);
  assert.strictEqual(
    page.document.getElementById("share-url").value,
    "https://example.test/rooms/testroom"
  );
});

test("the link is gone once the second player has joined", async () => {
  const page = loadPage(summary("TREWA", TREWA_MOVES));
  await tick();

  const share = page.document.getElementById("share");
  assert.strictEqual(share.classList.contains("hidden"), true);
});

test("a room of the list waits for the identity of the browser", async () => {
  const page = loadListPage();

  // The button is pressed before the list has arrived.
  for (const handler of page.document.getElementById("create").listeners.click) handler();
  await tick();
  assert.strictEqual(page.calls.filter((call) => call.method === "POST").length, 0);

  page.releaseList();
  await tick();
  await tick();

  const posts = page.calls.filter((call) => call.method === "POST");
  assert.strictEqual(posts.length, 1);
  assert.strictEqual(posts[0].path, "/api/rooms");
  assert.strictEqual(posts[0].headers["X-CSRF-Token"], "test-token");
});

test("the room is pushed instead of polled", async () => {
  const page = loadPage(summary("TREWA", TREWA_MOVES));
  await tick();

  // The page asks once for the room it opened, and then listens.
  assert.deepStrictEqual(page.calls.map((call) => call.path), ["/api/rooms/testroom"]);
  assert.strictEqual(page.intervals.length, 0, "the page does not poll");

  assert.strictEqual(page.streams.length, 1);
  assert.strictEqual(page.streams[0].url, "/api/rooms/testroom/events");

  // The opponent moves: the server pushes the room to the page.
  const moved = summary("TREWA", TREWA_MOVES);
  moved.game.version = 4;
  moved.game.currentPlayer = "PANGUI";
  page.streams[0].onmessage({data: JSON.stringify(moved)});
  await tick();

  assert.strictEqual(
    page.document.getElementById("room-status").textContent,
    "Turno de Pangui 🐆"
  );
  assert.strictEqual(page.streams.length, 1, "the page keeps one stream");
});

test("a page that loses the stream goes back to asking", async () => {
  const page = loadPage(summary("TREWA", TREWA_MOVES));
  await tick();

  const stream = page.streams[0];
  stream.readyState = FakeEventSource.CLOSED;
  stream.onerror();

  assert.deepStrictEqual(page.intervals, [5000]);
});

test("the puma may end a capture chain from the page", async () => {
  const data = summary("PANGUI", [
    {from: 7, to: 11, type: "CAPTURE", capturedNode: 6},
    {from: 7, to: 7, type: "END_TURN"},
  ]);
  data.game.captureSequenceActive = true;
  const page = loadPage(data);
  await tick();

  const row = page.document.getElementById("end-turn-row");
  assert.strictEqual(row.classList.contains("hidden"), false);

  for (const handler of page.document.getElementById("end-turn").listeners.click) handler();
  await tick();

  const sent = moveRequests(page.calls);
  assert.strictEqual(sent.length, 1);
  assert.deepStrictEqual(sent[0].body, {from: 7, to: 7, version: 3});
});

test("ending a chain is only offered while one runs", async () => {
  const page = loadPage(summary("PANGUI", PANGUI_MOVES));
  await tick();

  const row = page.document.getElementById("end-turn-row");
  assert.strictEqual(row.classList.contains("hidden"), true);
});

test("the end of a chain is a button, not a point of the board", async () => {
  const data = summary("PANGUI", [
    {from: 7, to: 7, type: "END_TURN"},
    {from: 7, to: 11, type: "NORMAL"},
  ]);
  data.game.captureSequenceActive = true;
  const page = loadPage(data);
  await tick();

  clickNode(page.document, 7); // pick the puma

  // Its own point is not drawn as somewhere to move to.
  const node = NODES.find(([id]) => id === 7);
  const cx = MARGIN + node[1] * CELL;
  const cy = MARGIN + node[2] * CELL;
  const targets = page.document.registry.filter(
    (element) =>
      (element.attributes.class === "target" || element.attributes.class === "target-capture") &&
      Number(element.attributes.cx) === cx &&
      Number(element.attributes.cy) === cy
  );
  assert.strictEqual(targets.length, 0);
});
