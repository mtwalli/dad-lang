// Tests the extension against a stand-in for the VS Code API and a real `dad` program.
// Run: DAD=/path/to/dad node test/extension.test.js
"use strict";

const Module = require("module");
const assert = require("assert");
const path = require("path");
const fs = require("fs");
const os = require("os");

const DAD = process.env.DAD;
if (!DAD) {
  console.error("Set DAD to the path of a dad binary built from this repository.");
  process.exit(2);
}

// ---------- a small fake of the vscode module ----------

class Position {
  constructor(line, character) {
    this.line = line;
    this.character = character;
  }
  isEqual(o) {
    return o.line === this.line && o.character === this.character;
  }
  translate(dl, dc) {
    return new Position(this.line + dl, this.character + dc);
  }
}
class Range {
  constructor(a, b, c, d) {
    if (a instanceof Position) [this.start, this.end] = [a, b];
    else [this.start, this.end] = [new Position(a, b), new Position(c, d)];
  }
}
class Uri {
  constructor(fsPath, scheme = "file") {
    this.fsPath = fsPath;
    this.scheme = scheme;
  }
  toString() {
    return this.scheme + "://" + this.fsPath;
  }
  static parse(s) {
    return new Uri(s, "https");
  }
}
class Diagnostic {
  constructor(range, message, severity) {
    Object.assign(this, { range, message, severity });
  }
}

const events = {};
const on = (name) => (fn) => {
  (events[name] = events[name] || []).push(fn);
  return { dispose() {} };
};
const diagSets = new Map();
const terminals = [];
const messages = [];
const config = { path: DAD, checkOnType: true, saveBeforeRun: true };
const commands = {};

function makeDoc(text, file = "/tmp/test.dad") {
  const lines = text.split("\n");
  return {
    uri: new Uri(file),
    languageId: "dad",
    version: 1,
    isClosed: false,
    isDirty: false,
    isUntitled: false,
    lineCount: lines.length,
    getText: (r) => (r ? lines[r.start.line].slice(r.start.character, r.end.character) : text),
    lineAt: (i) => ({ text: lines[i] }),
    getWordRangeAtPosition(pos) {
      const line = lines[pos.line];
      const word = /[\p{L}\p{Mn}\p{Nd}_]/u;
      let s = pos.character;
      let e = pos.character;
      while (s > 0 && word.test(line[s - 1])) s--;
      while (e < line.length && word.test(line[e])) e++;
      return s === e ? undefined : new Range(pos.line, s, pos.line, e);
    },
    save: async () => true,
  };
}

const vscode = {
  Position,
  Range,
  Uri,
  Diagnostic,
  DiagnosticSeverity: { Error: 0, Warning: 1 },
  CompletionItemKind: { Keyword: 13, Function: 2, Class: 6 },
  CompletionItem: class {
    constructor(label, kind) {
      Object.assign(this, { label, kind });
    }
  },
  MarkdownString: class {
    constructor(value) {
      this.value = value;
    }
  },
  Hover: class {
    constructor(contents, range) {
      Object.assign(this, { contents, range });
    }
  },
  ThemeIcon: class {
    constructor(id) {
      this.id = id;
    }
  },
  env: { openExternal() {} },
  languages: {
    createDiagnosticCollection: () => ({
      set: (uri, d) => diagSets.set(uri.toString(), d),
      delete: (uri) => diagSets.delete(uri.toString()),
      dispose() {},
    }),
    registerHoverProvider: (sel, p) => ((vscode._hover = p), { dispose() {} }),
    registerCompletionItemProvider: (sel, p) => ((vscode._completion = p), { dispose() {} }),
  },
  commands: {
    registerCommand: (id, fn) => ((commands[id] = fn), { dispose() {} }),
    executeCommand() {},
  },
  window: {
    activeTextEditor: null,
    showWarningMessage: (m) => (messages.push(m), Promise.resolve()),
    showInformationMessage: (m) => (messages.push(m), Promise.resolve()),
    showErrorMessage: (m) => (messages.push(m), Promise.resolve()),
    createTerminal: (opts) => {
      const t = { opts, shown: false, disposed: false, show() { this.shown = true; }, dispose() { this.disposed = true; } };
      terminals.push(t);
      return t;
    },
    onDidCloseTerminal: on("closeTerminal"),
  },
  workspace: {
    textDocuments: [],
    getConfiguration: () => ({ get: (k, d) => (k in config ? config[k] : d) }),
    onDidOpenTextDocument: on("open"),
    onDidSaveTextDocument: on("save"),
    onDidChangeTextDocument: on("change"),
    onDidCloseTextDocument: on("close"),
    onDidChangeConfiguration: on("config"),
  },
};

const resolveOrig = Module._resolveFilename;
Module._resolveFilename = function (req, ...rest) {
  return req === "vscode" ? "vscode" : resolveOrig.call(this, req, ...rest);
};
require.cache.vscode = { id: "vscode", filename: "vscode", loaded: true, exports: vscode };

const ext = require("../extension");
ext.activate({ subscriptions: [] });

// ---------- tests ----------

const wait = (ms) => new Promise((r) => setTimeout(r, ms));
async function diagnosticsFor(doc) {
  events.open.forEach((fn) => fn(doc));
  for (let i = 0; i < 50 && !diagSets.has(doc.uri.toString()); i++) await wait(50);
  return diagSets.get(doc.uri.toString());
}

const tests = [];
const test = (name, fn) => tests.push([name, fn]);

test("registers the commands", () => {
  for (const id of ["dad.run", "dad.check", "dad.showVersion"]) assert.ok(commands[id], id);
});

test("a clean file has no problems", async () => {
  const d = await diagnosticsFor(makeDoc(fs.readFileSync(path.join(__dirname, "../../../examples/01-hello.dad"), "utf8"), "/tmp/a.dad"));
  assert.deepStrictEqual(d, []);
});

test("errors and warnings land on the right words", async () => {
  const doc = makeDoc("// مثال\nاذا عمر > ١8 {\n}", "/tmp/b.dad");
  const d = await diagnosticsFor(doc);
  assert.strictEqual(d.length, 2);
  assert.strictEqual(d[0].severity, 1); // warning: «اذا» without hamza
  assert.deepStrictEqual([d[0].range.start.line, d[0].range.start.character, d[0].range.end.character], [1, 0, 3]);
  assert.strictEqual(doc.getText(d[0].range), "اذا");
  assert.strictEqual(d[1].severity, 0); // error: mixed digits
  assert.strictEqual(doc.getText(d[1].range), "١8");
  assert.match(d[1].message, /الأرقام العربية/);
  assert.strictEqual(d[1].source, "ضاد");
});

test("a syntax error is reported", async () => {
  const d = await diagnosticsFor(makeDoc("اطبع(١ + )", "/tmp/c.dad"));
  assert.strictEqual(d.length, 1);
  assert.match(d[0].message, /تعبير غير متوقع/);
  assert.strictEqual(d[0].range.start.character, 9);
});

test("columns after characters outside the BMP are converted to UTF-16", () => {
  const doc = makeDoc('ثابت ن = "😀" + خطا');
  const pos = ext._test.toPosition(doc, 1, 16); // the character after the emoji string
  assert.strictEqual(pos.character, 16);
  assert.strictEqual(doc.lineAt(0).text.slice(pos.character), "خطا");
});

test("run opens a terminal that runs dad directly", async () => {
  const file = path.join(os.tmpdir(), "مثال تجريبي.dad");
  fs.writeFileSync(file, 'اطبع("مرحبا")\n');
  const doc = makeDoc('اطبع("مرحبا")', file);
  vscode.window.activeTextEditor = { document: doc };
  await commands["dad.run"]();
  const t = terminals.at(-1);
  assert.strictEqual(t.opts.shellPath, DAD);
  assert.deepStrictEqual(t.opts.shellArgs, ["run", file]);
  assert.strictEqual(t.opts.cwd, os.tmpdir());
  assert.ok(t.shown);
  await commands["dad.run"]();
  assert.ok(t.disposed, "the previous run terminal is replaced");
});

test("hover explains keywords in English", () => {
  const doc = makeDoc("طالما صح { توقف }");
  const h = vscode._hover.provideHover(doc, new Position(0, 2));
  assert.match(h.contents.value, /`while`/);
  assert.strictEqual(vscode._hover.provideHover(doc, new Position(0, 9)), null);
});

test("completion offers keywords and built-ins", () => {
  const items = vscode._completion.provideCompletionItems();
  const labels = items.map((i) => i.label);
  for (const w of ["إذا", "طابق", "اطبع", "صحيح"]) assert.ok(labels.includes(w), w);
});

test("a missing dad shows one helpful warning", async () => {
  config.path = "/nonexistent/dad";
  events.config.forEach((fn) => fn({ affectsConfiguration: () => true }));
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = ١", "/tmp/d.dad")));
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = ١", "/tmp/e.dad")));
  await wait(300);
  assert.strictEqual(messages.filter((m) => /couldn't find/.test(m)).length, 1);
  config.path = DAD;
});

(async () => {
  for (const [name, fn] of tests) {
    try {
      await fn();
      console.log("ok  ", name);
    } catch (e) {
      console.log("FAIL", name, "\n     ", e.message);
      process.exitCode = 1;
    }
  }
  ext.deactivate();
})();
