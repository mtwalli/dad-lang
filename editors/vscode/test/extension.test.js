// Tests the extension against a stand-in for the VS Code API and a real `dad` program.
// Run: DAD=/path/to/dad node test/extension.test.js
"use strict";

const Module = require("module");
const assert = require("assert");
const path = require("path");
const fs = require("fs");
const os = require("os");

const DAD = process.env.DAD;

// A throwaway home folder, so installing the font never touches a real font folder.
const HOME = require("fs").mkdtempSync(require("path").join(require("os").tmpdir(), "dad-home-"));
process.env.HOME = HOME;
delete process.env.XDG_DATA_HOME;
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
  static joinPath(base, ...parts) {
    return new Uri(path.join(base.fsPath, ...parts), base.scheme);
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
const answers = []; // replies to the next information messages, in order
const config = { path: DAD, checkOnType: true, saveBeforeRun: true };
const commands = {};
const executed = []; // commands the extension ran, with their arguments
const applied = []; // workspace edits the extension applied
let customProvider = null;
let viewProvider = null; // the Ḍād output panel's provider
let outputPanel = null; // its fake webview view, once VS Code "opens" the panel

function fakeWebviewView() {
  const v = {
    posted: [],
    handlers: [],
    shown: 0,
    webview: {
      options: null,
      html: "",
      cspSource: "vscode-resource:",
      asWebviewUri: (u) => u,
      postMessage: (m) => (v.posted.push(m), Promise.resolve(true)),
      onDidReceiveMessage: (fn) => (v.handlers.push(fn), { dispose() {} }),
    },
    onDidDispose: () => ({ dispose() {} }),
    show() {
      v.shown++;
    },
    send: (m) => v.handlers.forEach((fn) => fn(m)),
    text: () => v.posted.filter((m) => m.type === "chunk").map((m) => m.text).join(""),
  };
  return v;
}
const diagnosticsByUri = new Map();

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
    lineAt: (i) => ({ text: lines[i], range: new Range(i, 0, i, lines[i].length) }),
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
  WorkspaceEdit: class {
    constructor() {
      this.edits = [];
    }
    replace(uri, range, text) {
      this.edits.push({ uri, range, text });
    }
  },
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
  EventEmitter: class {
    constructor() {
      this.listeners = [];
      this.event = (fn) => (this.listeners.push(fn), { dispose() {} });
    }
    fire(v) {
      this.listeners.forEach((fn) => fn(v));
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
    getDiagnostics: (uri) => diagnosticsByUri.get(uri.toString()) || [],
    onDidChangeDiagnostics: on("diagnosticsChanged"),
  },
  commands: {
    registerCommand: (id, fn) => ((commands[id] = fn), { dispose() {} }),
    executeCommand: (...args) => {
      executed.push(args);
      // Focusing the panel makes VS Code create its view; the page then says it is ready.
      if (args[0] === "dad.output.focus" && viewProvider && !outputPanel) {
        outputPanel = fakeWebviewView();
        viewProvider.resolveWebviewView(outputPanel);
        outputPanel.send({ type: "ready" });
      }
      return Promise.resolve();
    },
  },
  window: {
    activeTextEditor: null,
    showWarningMessage: (m) => (messages.push(m), Promise.resolve()),
    showInformationMessage: (m) => (messages.push(m), Promise.resolve(answers.shift())),
    showErrorMessage: (m) => (messages.push(m), Promise.resolve()),
    createTerminal: (opts) => {
      const t = { opts, shown: false, disposed: false, show() { this.shown = true; }, dispose() { this.disposed = true; } };
      terminals.push(t);
      return t;
    },
    onDidCloseTerminal: on("closeTerminal"),
    registerWebviewViewProvider: (id, provider) => ((viewProvider = provider), { dispose() {} }),
    registerCustomEditorProvider: (viewType, provider) => ((customProvider = { viewType, provider }), { dispose() {} }),
  },
  workspace: {
    textDocuments: [],
    getConfiguration: () => ({ get: (k, d) => (k in config ? config[k] : d) }),
    onDidOpenTextDocument: on("open"),
    onDidSaveTextDocument: on("save"),
    onDidChangeTextDocument: on("change"),
    onDidCloseTextDocument: on("close"),
    onDidChangeConfiguration: on("config"),
    applyEdit: (edit) => (applied.push(edit), Promise.resolve(true)),
  },
};

const resolveOrig = Module._resolveFilename;
Module._resolveFilename = function (req, ...rest) {
  return req === "vscode" ? "vscode" : resolveOrig.call(this, req, ...rest);
};
require.cache.vscode = { id: "vscode", filename: "vscode", loaded: true, exports: vscode };

const ext = require("../extension");
const state = {};
const context = {
  subscriptions: [],
  extensionPath: path.join(__dirname, ".."),
  extensionUri: new Uri(path.join(__dirname, "..")),
  globalState: { get: (k) => state[k], update: (k, v) => ((state[k] = v), Promise.resolve()) },
};
ext.activate(context);
const font = require("../font");

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
  for (const id of ["dad.run", "dad.check", "dad.showVersion", "dad.installFont"]) assert.ok(commands[id], id);
});

test("a clean file has no problems", async () => {
  const d = await diagnosticsFor(makeDoc(fs.readFileSync(path.join(__dirname, "../../../examples/01-hello.dad"), "utf8"), "/tmp/a.dad"));
  assert.deepStrictEqual(d, []);
});

test("errors and warnings land on the right words", async () => {
  const doc = makeDoc("// مثال\nاذا عمر > \u0661\u0668 {\n}", "/tmp/b.dad"); // ١٨: digits must be 0-9
  const d = await diagnosticsFor(doc);
  assert.strictEqual(d.length, 2);
  assert.strictEqual(d[0].severity, 1); // warning: «اذا» without hamza
  assert.deepStrictEqual([d[0].range.start.line, d[0].range.start.character, d[0].range.end.character], [1, 0, 3]);
  assert.strictEqual(doc.getText(d[0].range), "اذا");
  assert.strictEqual(d[1].severity, 0); // error: mixed digits
  assert.strictEqual(doc.getText(d[1].range), "\u0661\u0668");
  assert.match(d[1].message, /0-9: «١٨» تُكتب «18»/);
  assert.strictEqual(d[1].source, "ضاد");
});

test("a syntax error is reported", async () => {
  const d = await diagnosticsFor(makeDoc("اطبع(1 + )", "/tmp/c.dad"));
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

test("run shows the output in the Ḍād panel and keeps it", async () => {
  const file = path.join(os.tmpdir(), "مثال تجريبي.dad");
  fs.writeFileSync(file, 'اطبع("مرحبا")\nاطبع(1 + 2)\n');
  vscode.window.activeTextEditor = { document: makeDoc('اطبع("مرحبا")', file) };
  await commands["dad.run"]();
  assert.ok(outputPanel, "the panel opens");
  assert.match(outputPanel.webview.html, /<html lang="ar" dir="rtl">/);
  assert.match(outputPanel.webview.html, /DadMono-Regular\.ttf/);
  const start = outputPanel.posted.find((m) => m.type === "start");
  assert.strictEqual(start.file, "مثال تجريبي.dad");
  assert.strictEqual(outputPanel.text(), "مرحبا\n3\n");
  const done = outputPanel.posted.at(-1);
  assert.strictEqual(done.type, "done");
  assert.strictEqual(done.code, 0);
  assert.strictEqual(terminals.length, 0, "no terminal is used");
  // Running again clears and refills the same panel.
  const before = outputPanel.posted.length;
  outputPanel.send({ type: "rerun" });
  const rerunDone = () => outputPanel.posted.slice(before).some((m) => m.type === "done");
  for (let i = 0; i < 50 && !rerunDone(); i++) await wait(50);
  assert.ok(outputPanel.posted.slice(before).some((m) => m.type === "start"), "run again starts a new run");
  assert.strictEqual(outputPanel.posted.at(-1).code, 0);
});

test("run shows errors and the exit code", async () => {
  const file = path.join(os.tmpdir(), "خطأ.dad");
  fs.writeFileSync(file, "اطبع(1 / 0)\n");
  vscode.window.activeTextEditor = { document: makeDoc("اطبع(1 / 0)", file) };
  const before = outputPanel.posted.length;
  await commands["dad.run"]();
  const msgs = outputPanel.posted.slice(before);
  assert.ok(msgs.some((m) => m.type === "chunk" && m.stream === "err" && /قسمة على صفر/.test(m.text)));
  assert.strictEqual(msgs.at(-1).code, 1);
});

test("a running program can be stopped", async () => {
  const file = path.join(os.tmpdir(), "حلقة.dad");
  fs.writeFileSync(file, "متغير ن = 0\nطالما صح { ن += 1 }\n");
  vscode.window.activeTextEditor = { document: makeDoc("طالما صح {}", file) };
  const before = outputPanel.posted.length;
  const running = commands["dad.run"]();
  await wait(150);
  outputPanel.send({ type: "stop" });
  await running;
  const done = outputPanel.posted.slice(before).find((m) => m.type === "done");
  assert.ok(done.stopped || done.code !== 0, "stopped");
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
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = 1", "/tmp/d.dad")));
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = 1", "/tmp/e.dad")));
  await wait(300);
  assert.strictEqual(messages.filter((m) => /couldn't find/.test(m)).length, 1);
  config.path = DAD;
});

test("the font is set for .dad files", () => {
  const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, "../package.json"), "utf8"));
  assert.match(pkg.contributes.configurationDefaults["[dad]"]["editor.fontFamily"], /^'Dad Mono'/);
  for (const f of font.FILES) assert.ok(fs.existsSync(path.join(__dirname, "../fonts", f)), f);
});

test("offers Ḍād Mono once and installs it for the user", async () => {
  if (process.platform === "win32") return; // would write to the real registry
  font._reset();
  const before = messages.length;
  answers.push("Install font", undefined); // install, then dismiss the "reload" message
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = 1", "/tmp/f1.dad")));
  await wait(200);
  const dest = font.userFontDir();
  assert.ok(dest.startsWith(HOME), "installs under the test home folder");
  for (const f of font.FILES) assert.ok(fs.existsSync(path.join(dest, f)), "installed " + f);
  assert.ok(font.isInstalled());
  const asked = messages.slice(before).filter((m) => /Install Ḍād Mono\?/.test(m));
  assert.strictEqual(asked.length, 1);
  assert.ok(messages.slice(before).some((m) => /is installed in/.test(m)));
  events.open.forEach((fn) => fn(makeDoc("ثابت ب = 2", "/tmp/f2.dad")));
  await wait(100);
  assert.strictEqual(messages.slice(before).filter((m) => /Install Ḍād Mono\?/.test(m)).length, 1);
});

test("\"Don't ask again\" is remembered", async () => {
  if (process.platform === "win32") return;
  fs.rmSync(font.userFontDir(), { recursive: true, force: true });
  font._reset();
  answers.push("Don't ask again");
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = 1", "/tmp/f3.dad")));
  await wait(100);
  assert.strictEqual(state["dad.fontPrompt"], "never");
  font._reset();
  const before = messages.length;
  events.open.forEach((fn) => fn(makeDoc("ثابت أ = 1", "/tmp/f4.dad")));
  await wait(100);
  assert.strictEqual(messages.length, before, "no second prompt");
});

function fakePanel() {
  const panel = {
    posted: [],
    handlers: [],
    viewColumn: 1,
    webview: {
      options: null,
      html: "",
      cspSource: "vscode-resource:",
      asWebviewUri: (u) => u,
      postMessage: (m) => (panel.posted.push(m), Promise.resolve(true)),
      onDidReceiveMessage: (fn) => (panel.handlers.push(fn), { dispose() {} }),
    },
    onDidDispose: () => ({ dispose() {} }),
    send: (m) => Promise.all(panel.handlers.map((fn) => fn(m))),
  };
  return panel;
}

test("the right-to-left editor is registered for .dad files", () => {
  const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, "../package.json"), "utf8"));
  const ed = pkg.contributes.customEditors.find((e) => e.viewType === "dad.rtlEditor");
  assert.ok(ed && ed.priority === "default" && ed.selector[0].filenamePattern === "*.dad");
  assert.strictEqual(customProvider.viewType, "dad.rtlEditor");
  for (const f of ["media/rtl-editor.js", "media/rtl-editor.css"]) assert.ok(fs.existsSync(path.join(__dirname, "..", f)), f);
});

test("the right-to-left editor shows the document and its problems", async () => {
  const doc = makeDoc("ثابت أ = 1\nاطبع(أ)", "/tmp/rtl1.dad");
  diagnosticsByUri.set(doc.uri.toString(), [new Diagnostic(new Range(1, 0, 1, 4), "مشكلة تجريبية", 0)]);
  const panel = fakePanel();
  customProvider.provider.resolveCustomTextEditor(doc, panel);
  assert.ok(panel.webview.options.enableScripts);
  assert.match(panel.webview.html, /<html lang="ar" dir="rtl">/);
  assert.match(panel.webview.html, /Content-Security-Policy/);
  assert.match(panel.webview.html, /DadMono-Regular\.ttf/);
  await panel.send({ type: "ready" });
  const update = panel.posted.find((m) => m.type === "update");
  assert.strictEqual(update.text, "ثابت أ = 1\nاطبع(أ)");
  const diags = panel.posted.find((m) => m.type === "diagnostics").diagnostics;
  assert.deepStrictEqual(diags[0], { line: 1, start: 0, endLine: 1, end: 4, severity: 0, message: "مشكلة تجريبية" });
});

test("edits in the right-to-left editor go into VS Code's document", async () => {
  const doc = makeDoc("ثابت أ = 1", "/tmp/rtl2.dad");
  const panel = fakePanel();
  customProvider.provider.resolveCustomTextEditor(doc, panel);
  await panel.send({ type: "edit", text: "ثابت أ = 2" });
  const edit = applied.at(-1).edits[0];
  assert.strictEqual(edit.text, "ثابت أ = 2");
  assert.strictEqual(edit.uri.toString(), doc.uri.toString());
  const before = applied.length;
  await panel.send({ type: "edit", text: "ثابت أ = 1" }); // same as the document: nothing to apply
  assert.strictEqual(applied.length, before);
  await panel.send({ type: "openText" });
  assert.deepStrictEqual(executed.at(-1).slice(0, 3), ["vscode.openWith", doc.uri, "default"]);
  await panel.send({ type: "undo" });
  assert.deepStrictEqual(executed.at(-1), ["undo"]);
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
