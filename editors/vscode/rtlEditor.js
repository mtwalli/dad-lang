// A right-to-left editor for .dad files. VS Code's text editor always lays lines out
// left to right, so .dad files open in this custom editor instead. It edits VS Code's own
// document, so saving, undo, git, the dirty indicator and the Problems panel all work.
// Navigation (go to definition, references, rename, symbols, back) comes from dad lsp,
// through the same client as the text editor, and leads across the files of a program.
"use strict";

const vscode = require("vscode");
const crypto = require("crypto");
const path = require("path");

const VIEW_TYPE = "dad.rtlEditor";

const EXPECT_MS = 5000; // how long a jump into a file not open yet waits for its editor

class RtlEditorProvider {
  constructor(context, runFile, formatDocument, lsp) {
    this.context = context;
    this.runFile = runFile;
    this.formatDocument = formatDocument;
    this.lsp = lsp;
    this.editors = new Set(); // { document, panel, post, caret }
    this.active = null; // the editor in front, for the commands of the context menu
    this.expected = null; // { uri, range, until }: where to put the caret when a file's editor opens
    this.history = []; // { uri, line, character }: places to go back to
  }

  static register(context, runFile, formatDocument, lsp) {
    const provider = new RtlEditorProvider(context, runFile, formatDocument, lsp);
    if (lsp) lsp.onJump = (uri, range) => provider.expect(uri, range);
    const command = (id, fn) => vscode.commands.registerCommand(id, () => provider.active && fn.call(provider, provider.active));
    const disposable = vscode.Disposable.from(
      vscode.window.registerCustomEditorProvider(VIEW_TYPE, provider, {
        webviewOptions: { retainContextWhenHidden: true },
        supportsMultipleEditorsPerDocument: true,
      }),
      command("dad.rtl.definition", provider.goToDefinition),
      command("dad.rtl.references", provider.findReferences),
      command("dad.rtl.rename", provider.renameSymbol),
      command("dad.rtl.symbol", provider.goToSymbol),
      command("dad.rtl.workspaceSymbol", provider.goToWorkspaceSymbol),
      vscode.commands.registerCommand("dad.rtl.back", () => provider.back())
    );
    disposable.provider = provider;
    return disposable;
  }

  /** Whether a file is open in a right-to-left editor. */
  has(uri) {
    for (const e of this.editors) if (e.document.uri.toString() === uri.toString()) return true;
    return false;
  }

  resolveCustomTextEditor(document, panel) {
    const media = vscode.Uri.joinPath(this.context.extensionUri, "media");
    const fonts = vscode.Uri.joinPath(this.context.extensionUri, "fonts");
    panel.webview.options = { enableScripts: true, localResourceRoots: [media, fonts] };
    panel.webview.html = this.html(panel.webview, media, fonts);

    const post = (msg) => panel.webview.postMessage(msg);
    const editor = { document, panel, post, caret: new vscode.Position(0, 0), ready: false };
    this.editors.add(editor);
    this.active = editor;
    const sendText = () => post({ type: "update", text: document.getText() });
    const sendDiagnostics = () => post({ type: "diagnostics", diagnostics: diagnosticsFor(document.uri) });

    const subs = [
      vscode.workspace.onDidChangeTextDocument((e) => {
        if (e.document.uri.toString() === document.uri.toString() && e.contentChanges.length) sendText();
      }),
      vscode.languages.onDidChangeDiagnostics((e) => {
        if (e.uris.some((u) => u.toString() === document.uri.toString())) sendDiagnostics();
      }),
      panel.webview.onDidReceiveMessage((msg) => this.onMessage(editor, msg, sendText, sendDiagnostics)),
    ];
    if (panel.onDidChangeViewState) {
      subs.push(panel.onDidChangeViewState(() => {
        if (panel.active) {
          this.active = editor;
          this.revealExpected(editor);
        }
      }));
    }
    panel.onDidDispose(() => {
      subs.forEach((s) => s.dispose());
      this.editors.delete(editor);
      if (this.active === editor) this.active = null;
    });
  }

  async onMessage(editor, msg, sendText, sendDiagnostics) {
    const { document, panel } = editor;
    const pos = () => new vscode.Position(msg.line, msg.character);
    switch (msg.type) {
      case "ready":
        sendText();
        sendDiagnostics();
        editor.ready = true;
        this.revealExpected(editor);
        break;
      case "edit":
        if (msg.text !== document.getText()) await replaceAll(document, msg.text);
        break;
      case "caret":
        editor.caret = pos();
        this.active = editor;
        break;
      case "definition":
        editor.caret = pos();
        await this.goToDefinition(editor);
        break;
      case "references":
        editor.caret = pos();
        await this.findReferences(editor);
        break;
      case "rename":
        editor.caret = pos();
        await this.renameSymbol(editor);
        break;
      case "run":
        await this.runFile(document.uri);
        break;
      case "format":
        if (this.formatDocument) await this.formatDocument(document);
        break;
      case "undo":
      case "redo":
        await vscode.commands.executeCommand(msg.type);
        break;
      case "preview":
        await vscode.commands.executeCommand("dad.preview", document.uri);
        break;
      case "openText":
        await vscode.commands.executeCommand("vscode.openWith", document.uri, "default", panel.viewColumn);
        break;
    }
  }

  // ---------- navigation ----------

  say(text) {
    vscode.window.setStatusBarMessage("$(info) " + text, 4000);
  }

  async goToDefinition(editor) {
    const loc = this.lsp && (await this.lsp.definition(editor.document, editor.caret));
    if (!loc) return this.say("لا يوجد تعريف هنا");
    await this.go(editor, loc.uri, loc.range);
  }

  async findReferences(editor) {
    const locs = this.lsp ? await this.lsp.references(editor.document, editor.caret) : [];
    if (!locs.length) return this.say("لا توجد مراجع هنا");
    const items = [];
    for (const loc of locs) {
      const text = await lineText(loc.uri, loc.range.start.line);
      items.push({
        label: text.trim() || " ",
        description: `${path.basename(loc.uri.fsPath || loc.uri.path || "")}:${loc.range.start.line + 1}`,
        loc,
      });
    }
    const here = editor.document.uri.toString();
    const pick = await vscode.window.showQuickPick(items, {
      title: `${locs.length} ${locs.length === 1 ? "reference" : "references"}`,
      matchOnDescription: true,
      // preview places in this file as they are selected
      onDidSelectItem: (item) => item.loc.uri.toString() === here && editor.post(reveal(item.loc.range, false)),
    });
    if (pick) await this.go(editor, pick.loc.uri, pick.loc.range);
    else editor.post(reveal(new vscode.Range(editor.caret, editor.caret), true));
  }

  async renameSymbol(editor) {
    if (!this.lsp) return;
    const { document, caret } = editor;
    const prep = await this.lsp.prepareRename(document, caret);
    if (!prep) return this.say("لا يوجد اسم هنا يمكن تغييره");
    const name = await vscode.window.showInputBox({
      title: "Rename Symbol",
      value: prep.placeholder,
      valueSelection: [0, prep.placeholder.length],
      prompt: "الاسم الجديد، في كل الملفات التي تذكره",
    });
    if (!name || name === prep.placeholder) return;
    try {
      const edit = await this.lsp.rename(document, caret, name);
      if (edit) await vscode.workspace.applyEdit(edit);
    } catch (e) {
      vscode.window.showErrorMessage(e.message);
    }
  }

  async goToSymbol(editor) {
    const syms = this.lsp ? await this.lsp.symbols(editor.document) : [];
    const items = [];
    const walk = (list, container) => {
      for (const s of list) {
        items.push({ label: s.name, description: container || s.detail || "", range: rangeOf(s.selectionRange) });
        walk(s.children || [], s.name);
      }
    };
    walk(syms, "");
    if (!items.length) return this.say("لا توجد تعريفات في هذا الملف");
    const pick = await vscode.window.showQuickPick(items, {
      title: "Go to Symbol in File",
      matchOnDescription: true,
      onDidSelectItem: (item) => editor.post(reveal(item.range, false)),
    });
    if (pick) await this.go(editor, editor.document.uri, pick.range);
    else editor.post(reveal(new vscode.Range(editor.caret, editor.caret), true));
  }

  async goToWorkspaceSymbol(editor) {
    if (!this.lsp) return;
    const qp = vscode.window.createQuickPick();
    qp.title = "Go to Symbol in Workspace";
    qp.placeholder = "اسم نوع أو دالة أو مكون…";
    qp.matchOnDescription = true;
    let asked = 0;
    const search = async (query) => {
      const n = ++asked;
      qp.busy = true;
      const syms = await this.lsp.workspaceSymbols(query);
      if (n !== asked) return;
      qp.items = syms.map((s) => ({
        label: s.name,
        description: s.containerName || "",
        uri: vscode.Uri.parse(s.location.uri),
        range: rangeOf(s.location.range),
      }));
      qp.busy = false;
    };
    qp.onDidChangeValue(search);
    const picked = new Promise((resolve) => {
      qp.onDidAccept(() => resolve(qp.selectedItems[0]));
      qp.onDidHide(() => resolve(null));
    });
    qp.show();
    search("");
    const item = await picked;
    qp.dispose();
    if (item) await this.go(editor, item.uri, item.range);
  }

  /** Goes to a place, remembering where the caret was so that «back» returns there. */
  async go(from, uri, range) {
    this.history.push({ uri: from.document.uri, position: from.caret });
    if (this.history.length > 50) this.history.shift();
    vscode.commands.executeCommand("setContext", "dad.rtlCanGoBack", true);
    await this.show(uri, range, from);
  }

  async back() {
    const place = this.history.pop();
    vscode.commands.executeCommand("setContext", "dad.rtlCanGoBack", this.history.length > 0);
    if (place) await this.show(place.uri, new vscode.Range(place.position, place.position), this.active);
  }

  /** Shows a place: in the editor it is already open in, or by opening the file right to left. */
  async show(uri, range, from) {
    const key = uri.toString();
    let target = from && from.document.uri.toString() === key ? from : null;
    if (!target) {
      for (const e of this.editors) if (e.document.uri.toString() === key) target = e;
    }
    if (target) {
      if (target !== from && target.panel.reveal) target.panel.reveal(target.panel.viewColumn);
      this.active = target;
      target.caret = range.start;
      target.post(reveal(range, true));
      return;
    }
    this.expect(uri, range);
    await vscode.commands.executeCommand("vscode.openWith", uri, VIEW_TYPE, from && from.panel ? { viewColumn: from.panel.viewColumn } : undefined);
  }

  /** Asks the editor of a file about to open to put its caret at range. */
  expect(uri, range) {
    this.expected = { uri: uri.toString(), range, until: Date.now() + EXPECT_MS };
  }

  revealExpected(editor) {
    const x = this.expected;
    if (!x || !editor.ready || x.uri !== editor.document.uri.toString()) return;
    this.expected = null;
    if (Date.now() > x.until) return;
    editor.caret = x.range.start;
    editor.post(reveal(x.range, true));
  }

  html(webview, media, fonts) {
    const nonce = crypto.randomBytes(16).toString("base64");
    const uri = (base, file) => webview.asWebviewUri(vscode.Uri.joinPath(base, file));
    return `<!doctype html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; font-src ${webview.cspSource}; script-src 'nonce-${nonce}';">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
@font-face { font-family: "Dad Mono"; src: url("${uri(fonts, "DadMono-Regular.ttf")}") format("truetype"); font-weight: 400; }
@font-face { font-family: "Dad Mono"; src: url("${uri(fonts, "DadMono-Bold.ttf")}") format("truetype"); font-weight: 700; }
</style>
<link rel="stylesheet" href="${uri(media, "rtl-editor.css")}">
</head>
<body>
<div id="scroller">
  <div class="stack">
    <div id="gutter" aria-hidden="true"></div>
    <div class="code">
      <div id="current"></div>
      <pre id="hl" aria-hidden="true"></pre>
      <div id="link" aria-hidden="true"></div>
      <textarea id="src" dir="rtl" wrap="off" spellcheck="false" autocapitalize="off" autocomplete="off" aria-label="Ḍād code"></textarea>
    </div>
  </div>
</div>
<footer>
  <span id="status" role="status"></span>
  <button id="preview" type="button" title="Preview the interface beside the code">معاينة</button>
  <button id="run" type="button" title="Run this file (Ctrl/Cmd+Enter)">▶ شغّل</button>
  <button id="to-text" type="button" title="Open in VS Code's left-to-right text editor">Open as text</button>
</footer>
<script nonce="${nonce}" src="${uri(media, "rtl-editor.js")}"></script>
</body>
</html>`;
  }
}

/** The message that moves the caret of a right-to-left editor to a range (and selects it when select). */
function reveal(range, focus, select = true) {
  return {
    type: "reveal",
    line: range.start.line,
    character: range.start.character,
    endLine: select ? range.end.line : range.start.line,
    endCharacter: select ? range.end.character : range.start.character,
    focus,
  };
}

const rangeOf = (r) => new vscode.Range(r.start.line, r.start.character, r.end.line, r.end.character);

/** The text of a line of a file, for showing references. */
async function lineText(uri, line) {
  try {
    const doc = await vscode.workspace.openTextDocument(uri);
    return line < doc.lineCount ? doc.lineAt(line).text : "";
  } catch {
    return "";
  }
}

/** Replaces the whole document with text, as one undoable edit. */
async function replaceAll(document, text) {
  const edit = new vscode.WorkspaceEdit();
  const end = document.lineAt(Math.max(document.lineCount - 1, 0)).range.end;
  edit.replace(document.uri, new vscode.Range(new vscode.Position(0, 0), end), text);
  return vscode.workspace.applyEdit(edit);
}

/** VS Code's problems for a file, in the shape the webview draws. */
function diagnosticsFor(uri) {
  return vscode.languages.getDiagnostics(uri).map((d) => ({
    line: d.range.start.line,
    start: d.range.start.character,
    endLine: d.range.end.line,
    end: d.range.end.character,
    severity: d.severity,
    message: d.message,
  }));
}

module.exports = { RtlEditorProvider, VIEW_TYPE, replaceAll, diagnosticsFor };
