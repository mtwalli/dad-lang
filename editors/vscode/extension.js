// VS Code support for Ḍād (ضاد): errors as you type, a Run button, keyword hovers and completion.
// It talks to the installed `dad` program; there are no npm dependencies.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { KEYWORDS, BUILTINS } = require("./keywords");

const INSTALL_URL = "https://mtwalli.github.io/dad-lang/install/";
const CHECK_TIMEOUT_MS = 5000;

let diagnostics;
let cachedPath = null;
let warnedMissing = false;
let warnedOld = false;
let runTerminal = null;
const timers = new Map(); // document uri → debounce timer
const pending = new Map(); // document uri → running check process

// ---------- finding the dad program ----------

function candidates() {
  const exe = process.platform === "win32" ? "dad.exe" : "dad";
  const list = [];
  for (const dir of (process.env.PATH || "").split(path.delimiter)) {
    if (dir) list.push(path.join(dir, exe));
  }
  // VS Code started from the Dock or Start menu may not see the shell's PATH,
  // so also look where the installers put dad.
  list.push(path.join(os.homedir(), ".dad", "bin", exe));
  if (process.env.LOCALAPPDATA) list.push(path.join(process.env.LOCALAPPDATA, "dad", "bin", exe));
  if (process.platform !== "win32") list.push("/usr/local/bin/dad", "/opt/homebrew/bin/dad");
  return list;
}

function isFile(p) {
  try {
    return fs.statSync(p).isFile();
  } catch {
    return false;
  }
}

/** Returns the path of the dad program, or null when it can't be found. */
function dadPath() {
  const configured = vscode.workspace.getConfiguration("dad").get("path", "").trim();
  if (configured) return configured.replace(/^~(?=$|[\\/])/, os.homedir());
  if (cachedPath && isFile(cachedPath)) return cachedPath;
  cachedPath = candidates().find(isFile) || null;
  return cachedPath;
}

function warnMissing() {
  if (warnedMissing) return;
  warnedMissing = true;
  vscode.window
    .showWarningMessage(
      "Ḍād: couldn't find the `dad` program. Install it to see errors and run files, or set its path.",
      "How to install",
      "Set path"
    )
    .then((choice) => {
      if (choice === "How to install") vscode.env.openExternal(vscode.Uri.parse(INSTALL_URL));
      if (choice === "Set path") vscode.commands.executeCommand("workbench.action.openSettings", "dad.path");
    });
}

// ---------- positions ----------

/**
 * dad reports 1-based lines and columns counted in characters (code points);
 * VS Code counts UTF-16 units. Converts one to the other.
 */
function toPosition(doc, line, col) {
  const lineIdx = Math.min(Math.max(line - 1, 0), Math.max(doc.lineCount - 1, 0));
  const text = doc.lineAt(lineIdx).text;
  let units = 0;
  let chars = 1;
  while (chars < col && units < text.length) {
    const code = text.codePointAt(units);
    units += code > 0xffff ? 2 : 1;
    chars++;
  }
  return new vscode.Position(lineIdx, units);
}

function toRange(doc, line, col) {
  const start = toPosition(doc, line, col);
  const word = doc.getWordRangeAtPosition(start);
  if (word && word.start.isEqual(start)) return word;
  const text = doc.lineAt(start.line).text;
  if (start.character >= text.length) {
    // Problems at the end of a line (for example an unclosed bracket): mark the last character.
    return new vscode.Range(start.line, Math.max(text.length - 1, 0), start.line, text.length);
  }
  const width = text.codePointAt(start.character) > 0xffff ? 2 : 1;
  return new vscode.Range(start, start.translate(0, width));
}

/** Turns the output of `dad check --json` into VS Code diagnostics. */
function parseCheckOutput(doc, stdout) {
  const data = JSON.parse(stdout);
  return (data.diagnostics || []).map((d) => {
    const diag = new vscode.Diagnostic(
      toRange(doc, d.line, d.col),
      d.message,
      d.warning ? vscode.DiagnosticSeverity.Warning : vscode.DiagnosticSeverity.Error
    );
    diag.source = "ضاد";
    return diag;
  });
}

// ---------- checking ----------

function check(doc) {
  if (doc.languageId !== "dad") return;
  const exe = dadPath();
  if (!exe) {
    warnMissing();
    return;
  }
  const key = doc.uri.toString();
  const version = doc.version;
  const running = pending.get(key);
  if (running) running.kill();

  let child;
  try {
    child = cp.spawn(exe, ["check", "--json", "-"], { windowsHide: true });
  } catch {
    warnMissing();
    return;
  }
  pending.set(key, child);
  let out = "";
  child.stdout.setEncoding("utf8");
  child.stdout.on("data", (chunk) => (out += chunk));
  child.stderr.on("data", () => {});
  const timer = setTimeout(() => child.kill(), CHECK_TIMEOUT_MS);
  child.on("error", (err) => {
    clearTimeout(timer);
    if (pending.get(key) === child) pending.delete(key);
    if (err.code === "ENOENT") {
      cachedPath = null;
      warnMissing();
    }
  });
  child.on("close", () => {
    clearTimeout(timer);
    if (pending.get(key) !== child) return; // a newer check replaced this one
    pending.delete(key);
    if (doc.isClosed || doc.version !== version) return;
    try {
      diagnostics.set(doc.uri, parseCheckOutput(doc, out));
    } catch {
      warnOld();
    }
  });
  child.stdin.on("error", () => {});
  child.stdin.end(doc.getText(), "utf8");
}

function warnOld() {
  if (warnedOld) return;
  warnedOld = true;
  vscode.window
    .showWarningMessage(
      "Ḍād: your `dad` is too old to report errors to the editor. Update it to v0.1.4 or newer.",
      "How to update"
    )
    .then((choice) => {
      if (choice) vscode.env.openExternal(vscode.Uri.parse(INSTALL_URL));
    });
}

function scheduleCheck(doc, delay) {
  const key = doc.uri.toString();
  clearTimeout(timers.get(key));
  timers.set(
    key,
    setTimeout(() => {
      timers.delete(key);
      check(doc);
    }, delay)
  );
}

// ---------- running ----------

async function fileToRun(uri) {
  if (uri instanceof vscode.Uri && uri.scheme === "file") {
    const open = vscode.workspace.textDocuments.find((d) => d.uri.toString() === uri.toString());
    if (open && open.isDirty && vscode.workspace.getConfiguration("dad").get("saveBeforeRun", true)) {
      await open.save();
    }
    return uri.fsPath;
  }
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "dad") {
    vscode.window.showInformationMessage("Ḍād: open a .dad file to run it.");
    return null;
  }
  const doc = editor.document;
  if (doc.isUntitled) {
    // Run unsaved scratch files from a temporary copy.
    const tmp = path.join(os.tmpdir(), "dad-untitled.dad");
    fs.writeFileSync(tmp, doc.getText(), "utf8");
    return tmp;
  }
  if (doc.isDirty && vscode.workspace.getConfiguration("dad").get("saveBeforeRun", true)) {
    await doc.save();
  }
  return doc.uri.fsPath;
}

async function run(uri) {
  const file = await fileToRun(uri);
  if (!file) return;
  const exe = dadPath();
  if (!exe) {
    warnedMissing = false;
    warnMissing();
    return;
  }
  // The terminal runs dad directly (no shell), so file names never need quoting.
  if (runTerminal) runTerminal.dispose();
  runTerminal = vscode.window.createTerminal({
    name: `ضاد: ${path.basename(file)}`,
    shellPath: exe,
    shellArgs: ["run", file],
    cwd: path.dirname(file),
    iconPath: new vscode.ThemeIcon("play"),
    isTransient: true,
  });
  runTerminal.show(true);
}

function runCheckCommand() {
  const editor = vscode.window.activeTextEditor;
  if (!editor || editor.document.languageId !== "dad") return;
  check(editor.document);
  vscode.commands.executeCommand("workbench.actions.view.problems");
}

function showVersion() {
  const exe = dadPath();
  if (!exe) {
    warnedMissing = false;
    warnMissing();
    return;
  }
  cp.execFile(exe, ["version"], { windowsHide: true }, (err, stdout) => {
    if (err) vscode.window.showErrorMessage(`Ḍād: couldn't run ${exe}: ${err.message}`);
    else vscode.window.showInformationMessage(`${stdout.trim()} — ${exe}`);
  });
}

// ---------- hovers and completion ----------

function describe(word) {
  const entry = KEYWORDS[word] || BUILTINS[word];
  if (!entry) return null;
  const [english, text] = entry;
  const kind = KEYWORDS[word] ? "keyword" : "built-in";
  const md = new vscode.MarkdownString(`**${word}** · \`${english}\` · _${kind}_\n\n${text}`);
  md.supportHtml = false;
  return md;
}

const hoverProvider = {
  provideHover(doc, position) {
    const range = doc.getWordRangeAtPosition(position);
    if (!range) return null;
    const md = describe(doc.getText(range));
    return md ? new vscode.Hover(md, range) : null;
  },
};

const completionProvider = {
  provideCompletionItems() {
    const items = [];
    for (const [word, [english]] of Object.entries(KEYWORDS)) {
      const item = new vscode.CompletionItem(word, vscode.CompletionItemKind.Keyword);
      item.detail = english;
      item.documentation = describe(word);
      item.filterText = `${word} ${english}`;
      items.push(item);
    }
    for (const [word, [english]] of Object.entries(BUILTINS)) {
      const isType = /^[A-Z]/.test(english);
      const item = new vscode.CompletionItem(
        word,
        isType ? vscode.CompletionItemKind.Class : vscode.CompletionItemKind.Function
      );
      item.detail = english;
      item.documentation = describe(word);
      item.filterText = `${word} ${english}`;
      items.push(item);
    }
    return items;
  },
};

// ---------- activation ----------

function activate(context) {
  diagnostics = vscode.languages.createDiagnosticCollection("dad");
  const selector = { language: "dad" };
  const cfg = () => vscode.workspace.getConfiguration("dad");

  context.subscriptions.push(
    diagnostics,
    vscode.commands.registerCommand("dad.run", run),
    vscode.commands.registerCommand("dad.check", runCheckCommand),
    vscode.commands.registerCommand("dad.showVersion", showVersion),
    vscode.languages.registerHoverProvider(selector, hoverProvider),
    vscode.languages.registerCompletionItemProvider(selector, completionProvider),
    vscode.workspace.onDidOpenTextDocument((doc) => check(doc)),
    vscode.workspace.onDidSaveTextDocument((doc) => check(doc)),
    vscode.workspace.onDidChangeTextDocument((e) => {
      if (e.document.languageId === "dad" && cfg().get("checkOnType", true)) scheduleCheck(e.document, 300);
    }),
    vscode.window.onDidCloseTerminal((t) => {
      if (t === runTerminal) runTerminal = null;
    }),
    vscode.workspace.onDidCloseTextDocument((doc) => {
      diagnostics.delete(doc.uri);
      clearTimeout(timers.get(doc.uri.toString()));
    }),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("dad.path")) {
        cachedPath = null;
        warnedMissing = false;
        warnedOld = false;
        vscode.workspace.textDocuments.forEach(check);
      }
    })
  );
  vscode.workspace.textDocuments.forEach(check);
}

function deactivate() {
  for (const child of pending.values()) child.kill();
}

module.exports = { activate, deactivate, _test: { toPosition, toRange, parseCheckOutput, candidates } };
