// A DOM small enough to run the pages of the site outside a browser.
//
// The pages are plain scripts inside static HTML files, so a test can run
// them against this fake document and a stubbed fetch, and look at what the
// page did.
//
// This file is a helper for the tests: it is not embedded in the binary and
// it is never served.

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
    this.value = "";
    this.hidden = false;
    this.disabled = false;
    const classes = new Set();
    this.classList = {
      add: (name) => classes.add(name),
      remove: (name) => classes.delete(name),
      contains: (name) => classes.has(name),
      toggle: (name, force) => {
        const on = force === undefined ? !classes.has(name) : force;
        if (on) classes.add(name);
        else classes.delete(name);
        return on;
      },
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

  // Setting the inner HTML replaces the children: a page empties the board
  // before it draws it again.
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

// extractScript returns the script of an HTML page.
function extractScript(source) {
  const match = source.match(/<script>([\s\S]*?)<\/script>/);
  if (!match) throw new Error("the page has no script");
  return match[1];
}

// tick waits for the promises of the page to settle.
function tick() {
  return new Promise((resolve) => setImmediate(resolve));
}

module.exports = {FakeElement, FakeDocument, extractScript, tick};
