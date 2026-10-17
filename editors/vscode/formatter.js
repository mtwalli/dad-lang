// Formatting for .dad files: Format Document, format on save, and Shift+Alt+F in the
// right-to-left editor. The work is done by `dad fmt -`, so every editor formats alike.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");

const TIMEOUT_MS = 5000;

/**
 * Formats text with `dad fmt -`. Resolves to { text } or { error }; a file with
 * syntax errors gives { error, syntax: true } and is left as it is.
 */
function formatText(exe, text) {
  return new Promise((resolve) => {
    let child;
    try {
      child = cp.spawn(exe, ["fmt", "-"], { windowsHide: true });
    } catch (err) {
      resolve({ error: err.message });
      return;
    }
    let out = "", err = "";
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (c) => (out += c));
    child.stderr.on("data", (c) => (err += c));
    const timer = setTimeout(() => child.kill(), TIMEOUT_MS);
    child.on("error", (e) => {
      clearTimeout(timer);
      resolve({ error: e.code === "ENOENT" ? "dad was not found" : e.message });
    });
    child.on("close", (code) => {
      clearTimeout(timer);
      if (code === 0) resolve({ text: out });
      else if (/أمر غير معروف/.test(err)) resolve({ error: "This dad is too old to format files. Update it to the latest version.", old: true });
      else resolve({ error: err.trim() || `dad fmt exited with code ${code}`, syntax: /أخطاء/.test(err) });
    });
    child.stdin.end(text);
  });
}

/**
 * The smallest edit that turns the document into text: lines that didn't change
 * at the start and end are left alone, so the cursor and folds stay put.
 */
function minimalEdit(document, text) {
  const oldLines = document.getText().split("\n");
  const newLines = text.split("\n");
  let top = 0;
  while (top < oldLines.length && top < newLines.length && oldLines[top] === newLines[top]) top++;
  if (top === oldLines.length && top === newLines.length) return null;
  let bottom = 0;
  while (
    bottom < oldLines.length - top && bottom < newLines.length - top &&
    oldLines[oldLines.length - 1 - bottom] === newLines[newLines.length - 1 - bottom]
  ) bottom++;
  const endLine = oldLines.length - bottom; // exclusive
  const start = new vscode.Position(top, 0);
  const end = endLine >= document.lineCount
    ? document.lineAt(document.lineCount - 1).range.end
    : new vscode.Position(endLine, 0);
  let replacement = newLines.slice(top, newLines.length - bottom).join("\n");
  if (endLine < document.lineCount && newLines.length - bottom > top) replacement += "\n";
  return vscode.TextEdit.replace(new vscode.Range(start, end), replacement);
}

/**
 * Registers the formatter. dadPath() finds the dad program; activeUri() is the
 * .dad file in front, in either editor.
 */
function register(context, dadPath, activeUri) {
  const say = (msg) => vscode.window.setStatusBarMessage("$(warning) Ḍād: " + msg, 5000);

  async function edits(document) {
    const exe = dadPath();
    if (!exe) return [];
    const res = await formatText(exe, document.getText());
    if (res.error) {
      say(res.syntax ? "not formatted — fix the errors first" : res.error);
      return [];
    }
    const edit = minimalEdit(document, res.text);
    return edit ? [edit] : [];
  }

  async function formatDocument(document) {
    const list = await edits(document);
    if (!list.length) return false;
    const we = new vscode.WorkspaceEdit();
    we.set(document.uri, list);
    return vscode.workspace.applyEdit(we);
  }

  context.subscriptions.push(
    vscode.languages.registerDocumentFormattingEditProvider({ language: "dad" }, { provideDocumentFormattingEdits: edits }),
    vscode.commands.registerCommand("dad.format", async (uri) => {
      const target = uri instanceof vscode.Uri ? uri : activeUri();
      if (!target) return;
      const doc = await vscode.workspace.openTextDocument(target);
      if (doc.languageId === "dad") await formatDocument(doc);
    }),
    // Format on save works in both editors: VS Code's own "format on save" doesn't
    // run for custom editors such as the right-to-left one.
    vscode.workspace.onWillSaveTextDocument((e) => {
      if (e.document.languageId !== "dad") return;
      if (!vscode.workspace.getConfiguration("dad", e.document.uri).get("formatOnSave", true)) return;
      if (e.reason === vscode.TextDocumentSaveReason.AfterDelay) return; // not while auto-saving mid-typing
      e.waitUntil(edits(e.document));
    })
  );
  return { formatDocument };
}

module.exports = { register, formatText, minimalEdit };
