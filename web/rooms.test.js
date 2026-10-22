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

// ---- a DOM small enough to run the page in ---------------------------

class FakeElement {
  constructor(tag, registry) {
    this.tagName = tag;
    this.registry = registry;
    this.attributes = {};
    this.children = [];
    this.listeners = {};
    this.textContent = "";
    this.className = "";
    this.href = "";
    this.hidden = false;
    this.disabled = false;
    const classes = new Set();
    this.classList = {
      add: (name) => classes.add(name),
      remove: (name) => classes.delete(name),
      contains: (name) => classes.has(name),
    };
    this._innerHTML = "";
  }

  setAttribute(name, value) {
    this.attributes[name] = String(value);
  }

  getAttribute(name) {
    return this.attributes[name];
  }

  appendChild(child) {
    this.children.push(child);
    return child;
  }

  addEventListener(name, handler) {
    if (!this.listeners[name]) this.listeners[name] = [];
    this.listeners[name].push(handler);
  }

  // Setting the inner HTML replaces the children: the page empties the
  // board before it draws it again.
  set innerHTML(value) {
    this._innerHTML = value;
    if (value === "") {
      for (const child of this.children) removeFromRegistry(this.registry, child);
      this.children = [];
    }
  }

  get innerHTML() {
    return this._innerHTML;
  }
}

// removeFromRegistry takes an element and everything inside it out of the
// registry of the document.
function removeFromRegistry(registry, element) {
  for (const child of element.children) removeFromRegistry(registry, child);
  const index = registry.indexOf(element);
  if (index >= 0) registry.splice(index, 1);
}

class FakeDocument {
  constructor() {
    this.registry = [];
    this.byId = new Map();
    this.cookie = "komikan_csrf=test-token";
  }

  getElementById(id) {
    if (!this.byId.has(id)) {
      const element = new FakeElement("div", this.registry);
      this.registry.push(element);
      this.byId.set(id, element);
    }
    return this.byId.get(id);
  }

  createElement(tag) {
    const element = new FakeElement(tag, this.registry);
    this.registry.push(element);
    return element;
  }

  createElementNS(namespace, tag) {
    return this.createElement(tag);
  }
}

// ---- the page under test ---------------------------------------------

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
      captureThreshold: 6,
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
    href: "https://example.test/rooms/testroom",
  };
  const navigator = {clipboard: {writeText: async () => {}}};
  // The page polls the server every two seconds; the test does not.
  const setInterval = () => 0;

  const source = fs.readFileSync(PAGE, "utf8");
  const script = source.match(/<script>([\s\S]*?)<\/script>/)[1];
  eval(script);

  return {document, calls};
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

const tick = () => new Promise((resolve) => setImmediate(resolve));

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
