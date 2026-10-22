// The home page starts a game and lists the games of the browser.
//
// This test drives the page script of web/index.html with a fake DOM and a
// stubbed fetch, and looks at the requests it makes and at the list it
// draws.
//
// Run it with: node --test "web/**/*.test.js"

const test = require("node:test");
const assert = require("node:assert");
const fs = require("node:fs");
const path = require("node:path");

const {FakeDocument, extractScript, tick} = require("./fakedom.js");

const PAGE = path.join(__dirname, "index.html");

// loadPage runs the script of the home page against a fake DOM and returns
// the document it drew on and the requests it made.
//
// The list of the games is the first call of the page, and it is the call
// that gives the browser its identity. It is held back until the test
// releases it, so the order of the calls can be looked at.
function loadPage(rooms, options) {
  const settings = options || {};
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
      if (settings.failCreate) return {status: 500, ok: false, json: async () => ({})};
      return {status: 201, ok: true, json: async () => ({id: "newroom"})};
    }
    await listReady;
    return {status: 200, ok: true, json: async () => ({rooms})};
  };
  const location = {
    pathname: "/",
    origin: "https://example.test",
    href: "https://example.test/",
  };

  const source = fs.readFileSync(PAGE, "utf8");
  const script = extractScript(source);
  eval(script);

  return {document, calls, location, releaseList: () => releaseList()};
}

// click presses a button of the page.
function click(document, id) {
  const button = document.getElementById(id);
  for (const handler of button.listeners.click) handler();
}

// ---- the tests --------------------------------------------------------

test("the home page lists the games of the browser", async () => {
  const page = loadPage([
    {id: "abc12345", status: "WAITING", you: {seat: 1, playerId: "p1"}},
    {id: "def67890", status: "ACTIVE", you: {seat: 2, playerId: "p1", role: "TREWA"}},
  ]);
  page.releaseList();
  await tick();

  const list = page.document.getElementById("games-list");
  assert.strictEqual(list.children.length, 2);
  assert.strictEqual(list.children[0].children[0].textContent, "abc12345");
  assert.strictEqual(list.children[0].children[0].href, "/rooms/abc12345");
  assert.strictEqual(list.children[0].children[1].textContent, "esperando al rival");
  assert.strictEqual(list.children[1].children[1].textContent, "en curso · Trewa 🐕");
  assert.strictEqual(page.document.getElementById("games").hidden, false);
});

test("the home page shows no list when there is no game", async () => {
  const page = loadPage([]);
  page.releaseList();
  await tick();

  assert.strictEqual(page.document.getElementById("games-list").children.length, 0);
  assert.strictEqual(page.document.getElementById("games").hidden, true);
});

test("the button starts a game and goes to it", async () => {
  const page = loadPage([]);
  page.releaseList();
  await tick();

  click(page.document, "create");
  await tick();

  const posts = page.calls.filter((call) => call.method === "POST");
  assert.strictEqual(posts.length, 1);
  assert.strictEqual(posts[0].path, "/api/rooms");
  assert.strictEqual(page.location.href, "/rooms/newroom");
});

test("a game that cannot be created says so", async () => {
  const page = loadPage([], {failCreate: true});
  page.releaseList();
  await tick();

  click(page.document, "create");
  await tick();

  assert.strictEqual(
    page.document.getElementById("note").textContent,
    "No se pudo crear la partida. Inténtalo de nuevo."
  );
  assert.strictEqual(page.document.getElementById("create").disabled, false);
});

test("the game waits for the identity of the browser", async () => {
  const page = loadPage([]);

  // The button is pressed before the list has arrived.
  click(page.document, "create");
  await tick();
  assert.strictEqual(page.calls.filter((call) => call.method === "POST").length, 0);

  page.releaseList();
  await tick();
  await tick();

  const posts = page.calls.filter((call) => call.method === "POST");
  assert.strictEqual(posts.length, 1);
  assert.strictEqual(posts[0].headers["X-CSRF-Token"], "test-token");
});
