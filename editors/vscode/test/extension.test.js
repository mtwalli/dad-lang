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
  // As VS Code writes them: file:///C:/… on Windows, with forward slashes.
  toString() {
    if (this.scheme !== "file") return this.scheme + "://" + this.fsPath;
    const p = this.fsPath.replace(/\\/g, "/");
    return "file://" + (p.startsWith("/") ? "" : "/") + p;
  }
  static parse(s) {
    const m = /^file:\/\/(.*)$/.exec(s);
    if (!m) return new Uri(s, "https");
    let p = decodeURIComponent(m[1]);
    if (/^\/[A-Za-z]:/.test(p)) p = p.slice(1); // /C:/… → C:/…
    return new Uri(path.normalize(p));
  }
  static file(p) {
    return new Uri(p);
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
const statusMessages = [];
const answers = []; // replies to the next information messages, in order
const config = { path: DAD, checkOnType: true, saveBeforeRun: true, formatOnSave: true };
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

const webviewPanels = []; // panels the extension created, such as the preview
const quickPicks = []; // what the extension offered to choose from, answered by pickAnswers
const pickAnswers = [];
const inputAnswers = [];
const vscode = {
  Position,
  Range,
  ViewColumn: { Active: -1, Beside: -2, One: 1, Two: 2 },
  Disposable: { from: (...ds) => ({ dispose: () => ds.forEach((d) => d && d.dispose && d.dispose()) }) },
  DocumentLink: class {
    constructor(range, target) {
      Object.assign(this, { range, target });
    }
  },
  SymbolInformation: class {
    constructor(name, kind, containerName, location) {
      Object.assign(this, { name, kind, containerName, location });
    }
  },
  WorkspaceEdit: class {
    constructor() {
      this.edits = [];
    }
    replace(uri, range, text) {
      this.edits.push({ uri, range, text });
    }
    set(uri, edits) {
      for (const e of edits) this.edits.push({ uri, range: e.range, text: e.newText });
    }
  },
  TextEdit: { replace: (range, newText) => ({ range, newText }) },
  TextDocumentSaveReason: { Manual: 1, AfterDelay: 2, FocusOut: 3 },
  Uri,
  Diagnostic,
  Location: class {
    constructor(uri, range) {
      Object.assign(this, { uri, range });
    }
  },
  DocumentHighlight: class {
    constructor(range, kind) {
      Object.assign(this, { range, kind });
    }
  },
  DocumentHighlightKind: { Text: 0, Read: 1, Write: 2 },
  DocumentSymbol: class {
    constructor(name, detail, kind, range, selectionRange) {
      Object.assign(this, { name, detail, kind, range, selectionRange, children: [] });
    }
  },
  DiagnosticSeverity: { Error: 0, Warning: 1 },
  CompletionItemKind: { Keyword: 13, Function: 2, Class: 6, Method: 1, Field: 4, Variable: 5, Property: 9, Enum: 12, EnumMember: 19, Constant: 20, Struct: 21, Text: 0 },
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
    registerHoverProvider: (sel, p) => (((p.provideHover.constructor.name === "AsyncFunction" ? (vscode._lspHover = p) : (vscode._hover = p))), { dispose() {} }),
    registerCompletionItemProvider: (sel, p) => (((p.provideCompletionItems.constructor.name === "AsyncFunction" ? (vscode._lspCompletion = p) : (vscode._completion = p))), { dispose() {} }),
    registerDefinitionProvider: (sel, p) => ((vscode._definition = p), { dispose() {} }),
    registerReferenceProvider: (sel, p) => ((vscode._references = p), { dispose() {} }),
    registerDocumentHighlightProvider: (sel, p) => ((vscode._highlights = p), { dispose() {} }),
    registerRenameProvider: (sel, p) => ((vscode._rename = p), { dispose() {} }),
    registerDocumentSymbolProvider: (sel, p) => ((vscode._symbols = p), { dispose() {} }),
    registerDocumentFormattingEditProvider: (sel, p) => ((vscode._formatter = p), { dispose() {} }),
    registerDocumentLinkProvider: (sel, p) => ((vscode._links = p), { dispose() {} }),
    registerWorkspaceSymbolProvider: (p) => ((vscode._workspaceSymbols = p), { dispose() {} }),
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
    visibleTextEditors: [],
    createWebviewPanel: (viewType, title, column, options) => {
      const panel = fakePanel();
      Object.assign(panel, { viewType, title, column, options, revealed: 0, disposeFns: [] });
      panel.webview.options = options;
      panel.reveal = () => panel.revealed++;
      panel.onDidDispose = (fn) => (panel.disposeFns.push(fn), { dispose() {} });
      panel.dispose = () => panel.disposeFns.forEach((fn) => fn());
      webviewPanels.push(panel);
      return panel;
    },
    setStatusBarMessage: (m) => (statusMessages.push(m), { dispose() {} }),
    showQuickPick: async (items, opts) => {
      quickPicks.push({ items, opts });
      const answer = pickAnswers.shift();
      return answer === undefined ? undefined : items[answer];
    },
    showInputBox: async () => inputAnswers.shift(),
  },
  workspace: {
    textDocuments: [],
    getConfiguration: () => ({ get: (k, d) => (k in config ? config[k] : d) }),
    onDidOpenTextDocument: on("open"),
    onDidSaveTextDocument: on("save"),
    onDidChangeTextDocument: on("change"),
    onDidCloseTextDocument: on("close"),
    onDidChangeConfiguration: on("config"),
    onWillSaveTextDocument: on("willSave"),
    applyEdit: (edit) => (applied.push(edit), Promise.resolve(true)),
    workspaceFolders: undefined,
    createFileSystemWatcher: () => ({ onDidChange: () => ({ dispose() {} }), onDidCreate: () => ({ dispose() {} }), dispose() {} }),
    openTextDocument: async (uri) => makeDoc(fs.readFileSync(uri.fsPath, "utf8"), uri.fsPath),
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
  globalStorageUri: new Uri(fs.mkdtempSync(path.join(os.tmpdir(), "dad-storage-"))),
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

// Applies TextEdits (from the formatter) to a document's text.
function applyEdits(doc, edits) {
  const lines = doc.getText().split("\n");
  const offset = (p) => lines.slice(0, p.line).reduce((n, l) => n + l.length + 1, 0) + p.character;
  let text = doc.getText();
  for (const e of [...edits].reverse()) {
    text = text.slice(0, offset(e.range.start)) + e.newText + text.slice(offset(e.range.end));
  }
  return text;
}

const messy = "دالة مربع(س: صحيح) {\nأرجع س*س\n      }\nاطبع( مربع(3) )\n";
const tidy = "دالة مربع(س: صحيح) {\n    أرجع س * س\n}\n\nاطبع(مربع(3))\n";

test("Format Document lays out the file with dad fmt", async () => {
  const doc = makeDoc(messy, "/tmp/f1.dad");
  const edits = await vscode._formatter.provideDocumentFormattingEdits(doc);
  assert.strictEqual(edits.length, 1);
  assert.strictEqual(applyEdits(doc, edits), tidy);
  // Lines that don't change are not part of the edit.
  assert.strictEqual(edits[0].range.start.line, 1);
  assert.deepStrictEqual(await vscode._formatter.provideDocumentFormattingEdits(makeDoc(tidy, "/tmp/f2.dad")), []);
});

test("a file with errors is not formatted", async () => {
  const before = statusMessages.length;
  const edits = await vscode._formatter.provideDocumentFormattingEdits(makeDoc("دالة (\n", "/tmp/f3.dad"));
  assert.deepStrictEqual(edits, []);
  assert.ok(statusMessages.slice(before).some((m) => /fix the errors/.test(m)));
});

test("saving formats the file, unless turned off or auto-saving", async () => {
  const save = (reason) => {
    const waits = [];
    events.willSave.forEach((fn) => fn({ document: makeDoc(messy, "/tmp/f4.dad"), reason, waitUntil: (p) => waits.push(p) }));
    return waits;
  };
  const waits = save(vscode.TextDocumentSaveReason.Manual);
  assert.strictEqual(waits.length, 1);
  assert.strictEqual(applyEdits(makeDoc(messy), await waits[0]), tidy);
  assert.strictEqual(save(vscode.TextDocumentSaveReason.AfterDelay).length, 0);
  config.formatOnSave = false;
  assert.strictEqual(save(vscode.TextDocumentSaveReason.Manual).length, 0);
  config.formatOnSave = true;
});

test("Shift+Alt+F in the right-to-left editor formats the document", async () => {
  const doc = makeDoc(messy, "/tmp/rtl3.dad");
  const panel = fakePanel();
  customProvider.provider.resolveCustomTextEditor(doc, panel);
  const before = applied.length;
  await panel.send({ type: "format" });
  assert.strictEqual(applied.length, before + 1);
  const e = applied.at(-1).edits[0];
  assert.strictEqual(applyEdits(doc, [{ range: e.range, newText: e.text }]), tidy);
});

const lspSrc = "نوع نقطة(س: عشري، ص: عشري) {\n    دالة بعد() = جذر(س * س + ص * ص)\n}\n\nدالة مربع(عدد: صحيح): صحيح = عدد * عدد\n\nثابت ن = نقطة(س: 3، ص: 4)\nاطبع(ن.بعد()، مربع(2)، مربع(عدد: 3))\n";

test("go to definition, references and highlights come from dad lsp", async () => {
  const doc = makeDoc(lspSrc, "/tmp/lsp1.dad");
  const def = await vscode._definition.provideDefinition(doc, new Position(7, 8)); // ن.بعد
  assert.ok(def, "no definition");
  assert.deepStrictEqual([def.range.start.line, def.range.start.character], [1, 9]);
  const refs = await vscode._references.provideReferences(doc, new Position(4, 6)); // مربع
  assert.strictEqual(refs.length, 3);
  const hs = await vscode._highlights.provideDocumentHighlights(doc, new Position(6, 6)); // ن: declared, then used once
  assert.strictEqual(hs.length, 2);
});

test("rename changes every use, and refuses keywords", async () => {
  const doc = makeDoc(lspSrc, "/tmp/lsp2.dad");
  const prep = await vscode._rename.prepareRename(doc, new Position(4, 11)); // عدد
  assert.strictEqual(prep.placeholder, "عدد");
  const edit = await vscode._rename.provideRenameEdits(doc, new Position(4, 11), "قيمة");
  assert.strictEqual(edit.edits.length, 4);
  assert.ok(edit.edits.every((e) => e.text === "قيمة"));
  await assert.rejects(() => vscode._rename.provideRenameEdits(doc, new Position(4, 11), "إذا"));
});

test("hover shows a declaration; completion after a dot lists members", async () => {
  const doc = makeDoc(lspSrc, "/tmp/lsp3.dad");
  const h = await vscode._lspHover.provideHover(doc, new Position(7, 15)); // مربع
  assert.ok(/دالة مربع\(عدد: صحيح\): صحيح/.test(h.contents[0] ? h.contents[0].value : h.contents.value));
  const doc2 = makeDoc(lspSrc + "اطبع(ن.)\n", "/tmp/lsp4.dad");
  const items = await vscode._lspCompletion.provideCompletionItems(doc2, new Position(8, 7));
  assert.deepStrictEqual(items.map((i) => i.label), ["بعد", "س", "ص", "نسخة"]);
  // the extension's keyword list stays out of the way after a dot
  assert.deepStrictEqual(vscode._completion.provideCompletionItems(doc2, new Position(8, 7)), []);
});

test("the outline lists declarations with their members", async () => {
  const doc = makeDoc(lspSrc, "/tmp/lsp5.dad");
  const syms = await vscode._symbols.provideDocumentSymbols(doc);
  assert.deepStrictEqual(syms.map((s) => s.name), ["نقطة", "مربع", "ن"]);
  assert.deepStrictEqual(syms[0].children.map((c) => c.name), ["س", "ص", "بعد"]);
});

// ---------- programs of several files ----------

const progDir = fs.mkdtempSync(path.join(os.tmpdir(), "dad-prog-"));
const progFiles = {
  "نماذج.dad": "نوع منتج(اسم: نص، سعر: عشري)\n\nدالة بالضريبة(سعر: عشري): عشري = سعر * 1.15\n",
  "main.dad": "استورد منتج، بالضريبة من \"./نماذج\"\n\nثابت م = منتج(\"خبز\"، 2.5)\nاطبع(م.اسم، بالضريبة(م.سعر))\n",
};
for (const [name, text] of Object.entries(progFiles)) fs.writeFileSync(path.join(progDir, name), text);
const lastReveal = (panel) => panel.posted.filter((m) => m.type === "reveal").at(-1);
const progDoc = (name) => makeDoc(progFiles[name], path.join(progDir, name));

test("definitions lead into the files a program imports", async () => {
  const doc = progDoc("main.dad");
  const def = await vscode._definition.provideDefinition(doc, new Position(3, 8)); // م.اسم
  assert.strictEqual(path.basename(def.uri.fsPath), "نماذج.dad");
  assert.deepStrictEqual([def.range.start.line, def.range.start.character], [0, 9]);
  const file = await vscode._definition.provideDefinition(doc, new Position(0, 27)); // "./نماذج"
  assert.strictEqual(path.basename(file.uri.fsPath), "نماذج.dad");
  const links = await vscode._links.provideDocumentLinks(doc);
  assert.deepStrictEqual(links.map((l) => path.basename(l.target.fsPath)), ["نماذج.dad"]);
});

test("references and rename reach every file of the program", async () => {
  const doc = progDoc("main.dad");
  const refs = await vscode._references.provideReferences(doc, new Position(3, 12)); // بالضريبة
  assert.deepStrictEqual(refs.map((r) => path.basename(r.uri.fsPath) + ":" + r.range.start.line).sort(), ["main.dad:0", "main.dad:3", "نماذج.dad:2"]);
  const edit = await vscode._rename.provideRenameEdits(doc, new Position(3, 12), "مع_الضريبة");
  assert.deepStrictEqual([...new Set(edit.edits.map((e) => path.basename(e.uri.fsPath)))].sort(), ["main.dad", "نماذج.dad"]);
  const syms = await vscode._workspaceSymbols.provideWorkspaceSymbols("منتج");
  assert.ok(syms.some((s) => s.name === "منتج" && path.basename(s.location.uri.fsPath) === "نماذج.dad"), JSON.stringify(syms.map((s) => s.name)));
});

test("the right-to-left editor's right-click menu has navigation and formatting", () => {
  const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, "../package.json"), "utf8"));
  const menu = pkg.contributes.menus["webview/context"].map((m) => m.command);
  for (const id of ["dad.rtl.definition", "dad.rtl.references", "dad.rtl.rename", "dad.rtl.symbol", "dad.rtl.workspaceSymbol", "dad.rtl.back", "dad.format", "dad.run"]) {
    assert.ok(menu.includes(id), id);
    assert.ok(id === "dad.format" || id === "dad.run" || commands[id], "no command " + id);
  }
  assert.ok(pkg.contributes.menus["webview/context"].every((m) => m.when.includes("webviewId == 'dad.rtlEditor'")));
});

test("in the right-to-left editor, go to definition moves the caret or opens the file", async () => {
  const doc = progDoc("main.dad");
  const panel = fakePanel();
  customProvider.provider.resolveCustomTextEditor(doc, panel);
  await panel.send({ type: "ready" });
  // the right-clicked place, then the menu's command
  await panel.send({ type: "caret", line: 3, character: 5 }); // م in اطبع(م.اسم…)
  await commands["dad.rtl.definition"]();
  assert.deepStrictEqual(lastReveal(panel), { type: "reveal", line: 2, character: 5, endLine: 2, endCharacter: 6, focus: true });
  // into the other file: it opens right to left, then its caret goes to the field
  await panel.send({ type: "definition", line: 3, character: 8 }); // اسم
  assert.deepStrictEqual([executed.at(-1)[0], executed.at(-1)[2]], ["vscode.openWith", "dad.rtlEditor"]);
  assert.strictEqual(path.basename(executed.at(-1)[1].fsPath), "نماذج.dad");
  const other = fakePanel();
  customProvider.provider.resolveCustomTextEditor(progDoc("نماذج.dad"), other);
  await other.send({ type: "ready" });
  assert.deepStrictEqual(lastReveal(other), { type: "reveal", line: 0, character: 9, endLine: 0, endCharacter: 12, focus: true });
  // back to where the caret was
  await commands["dad.rtl.back"]();
  assert.deepStrictEqual(lastReveal(panel), { type: "reveal", line: 3, character: 8, endLine: 3, endCharacter: 8, focus: true });
});

test("in the right-to-left editor, references, symbols and rename use pickers", async () => {
  const doc = progDoc("main.dad");
  const panel = fakePanel();
  customProvider.provider.resolveCustomTextEditor(doc, panel);
  await panel.send({ type: "ready" });
  pickAnswers.push(1);
  await panel.send({ type: "references", line: 3, character: 12 }); // بالضريبة
  const refs = quickPicks.at(-1).items;
  assert.strictEqual(refs.length, 3);
  assert.ok(refs.some((r) => /^نماذج\.dad:3$/.test(r.description)), refs.map((r) => r.description).join());
  assert.strictEqual(lastReveal(panel).type, "reveal");
  pickAnswers.push(0);
  await panel.send({ type: "caret", line: 0, character: 0 });
  await commands["dad.rtl.symbol"]();
  assert.deepStrictEqual(quickPicks.at(-1).items.map((i) => i.label), ["م"]);
  assert.deepStrictEqual(lastReveal(panel), { type: "reveal", line: 2, character: 5, endLine: 2, endCharacter: 6, focus: true });
  inputAnswers.push("الثمن");
  const before = applied.length;
  await panel.send({ type: "rename", line: 3, character: 12 });
  assert.strictEqual(applied.length, before + 1);
  assert.deepStrictEqual([...new Set(applied.at(-1).edits.map((e) => path.basename(e.uri.fsPath)))].sort(), ["main.dad", "نماذج.dad"]);
});

// ---------- the interface preview ----------

const previewDir = fs.mkdtempSync(path.join(os.tmpdir(), "dad-preview-"));
const previewFile = path.join(previewDir, "app.dad");
const previewSrc = "مكون الرئيسية() {\n    حالة ع = 0\n    عنوان(\"العدد {ع}\")\n    زر(\"زد\") { ع += 1 }\n}\n";
const until = async (fn) => {
  for (let i = 0; i < 100 && !fn(); i++) await wait(50);
  return fn();
};

const pageOf = (panel) => panel.posted.filter((m) => m.type === "page").at(-1);
const noteOf = (panel) => panel.posted.filter((m) => m.type === "note").at(-1);

test("the preview shows the interface beside the code, and follows it", async () => {
  fs.writeFileSync(previewFile, previewSrc);
  fs.writeFileSync(path.join(previewDir, "app.css"), ".خاص { color: red }\n");
  const uri = new Uri(previewFile);
  await commands["dad.preview"](uri);
  const panel = webviewPanels.at(-1);
  assert.strictEqual(panel.viewType, "dad.preview");
  assert.strictEqual(panel.column.viewColumn, vscode.ViewColumn.Beside);
  assert.ok(panel.options.enableScripts);
  // the panel's page: a frame for the program, a toolbar, and a policy
  assert.match(panel.webview.html, /<iframe id="app"/);
  assert.match(panel.webview.html, /Content-Security-Policy/);
  assert.match(panel.webview.html, /script-src &#39;nonce-/);
  assert.strictEqual(pageOf(panel), undefined); // not before the page says it is ready
  await panel.send({ type: "ready" });
  const html = pageOf(panel).html;
  assert.match(html, /<main id="dad-root"/);
  assert.match(html, /parent\.dadRoute/); // keeps the page on screen
  assert.match(html, /\.خاص \{ color: red \}/); // the program's own stylesheet
  assert.match(html, /زد/); // the compiled program
  const nonce = /nonce-([^&]+)&#39;/.exec(panel.webview.html)[1];
  assert.ok(html.includes(`<script nonce="${nonce}">`));
  const assets = context.globalStorageUri.fsPath;
  for (const f of ["ui.js", "ui.css", "dad-runtime.js"]) assert.ok(fs.existsSync(path.join(assets, "preview-assets", f)), f);
  // the same file again shows the same panel
  await commands["dad.preview"](uri);
  assert.strictEqual(webviewPanels.at(-1), panel);
  assert.strictEqual(panel.revealed, 1);

  // an error: the page stays, the problems go to it
  const pages = panel.posted.filter((m) => m.type === "page").length;
  fs.writeFileSync(previewFile, previewSrc.replace("ع += 1", "س += 1"));
  events.change.forEach((fn) => fn({ document: makeDoc("", previewFile), contentChanges: [{}] }));
  assert.ok(await until(() => panel.posted.some((m) => m.type === "problems" && m.problems.some((p) => !p.warning))), "no problems sent");
  assert.strictEqual(panel.posted.filter((m) => m.type === "page").length, pages);
  const p = panel.posted.find((m) => m.type === "problems").problems.find((x) => !x.warning);
  assert.strictEqual(p.line, 4);
  assert.match(p.where, /السطر 4/);

  // a click on the problem goes to the code
  await panel.send({ type: "goto", path: previewFile, line: 4, col: 13 });
  const open = executed.filter((e) => e[0] === "vscode.open").at(-1);
  assert.strictEqual(open[1].fsPath, previewFile);
  assert.deepStrictEqual([open[2].selection.start.line, open[2].selection.start.character], [3, 12]);

  // fixed: the page updates
  fs.writeFileSync(previewFile, previewSrc.replace("زد", "أضف"));
  events.change.forEach((fn) => fn({ document: makeDoc("", previewFile), contentChanges: [{}] }));
  assert.ok(await until(() => /أضف/.test(pageOf(panel).html)), "not updated");
  // if the panel's page starts over, it gets what it showed
  const before = panel.posted.length;
  await panel.send({ type: "ready" });
  assert.strictEqual(panel.posted.length, before + 1);
  assert.match(panel.posted.at(-1).html, /أضف/);
  panel.dispose();
});

test("the preview of a program without an interface says so", async () => {
  const file = path.join(previewDir, "plain.dad");
  fs.writeFileSync(file, "اطبع(1)\n");
  await commands["dad.preview"](new Uri(file));
  const panel = webviewPanels.at(-1);
  await panel.send({ type: "ready" });
  assert.match(noteOf(panel).title, /لا واجهة في هذا البرنامج/);
  await panel.send({ type: "run" });
  assert.deepStrictEqual(executed.at(-1).slice(0, 1), ["dad.run"]);
  // with errors from the start, it lists them
  fs.writeFileSync(file, "مكون الرئيسية() {\n    عنوان(س)\n}\n");
  await panel.send({ type: "reload" });
  assert.match(noteOf(panel).title, /في الكود أخطاء/);
  assert.match(noteOf(panel).problems[0].message, /«س»/);
  panel.dispose();
});

test("the preview is offered in the title bar, the right-click menu and the RTL editor", () => {
  const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, "../package.json"), "utf8"));
  assert.ok(pkg.contributes.menus["editor/title"].some((m) => m.command === "dad.preview"));
  assert.ok(pkg.contributes.menus["webview/context"].some((m) => m.command === "dad.preview"));
  assert.ok(pkg.contributes.keybindings.some((k) => k.command === "dad.preview"));
  const panel = fakePanel();
  customProvider.provider.resolveCustomTextEditor(makeDoc(previewSrc, previewFile), panel);
  assert.match(panel.webview.html, /id="preview"/);
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
  // VS Code disposes an extension's subscriptions when it stops; this stops dad lsp.
  context.subscriptions.forEach((d) => d && d.dispose && d.dispose());
})();
