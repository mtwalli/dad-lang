// A right-to-left editor for .dad files. VS Code's text editor always lays lines out
// left to right, so .dad files open in this custom editor instead. It edits VS Code's own
// document, so saving, undo, git, the dirty indicator and the Problems panel all work.
"use strict";

const vscode = require("vscode");
const crypto = require("crypto");

const VIEW_TYPE = "dad.rtlEditor";

class RtlEditorProvider {
  constructor(context, runFile, formatDocument) {
    this.context = context;
    this.runFile = runFile;
    this.formatDocument = formatDocument;
  }

  static register(context, runFile, formatDocument) {
    return vscode.window.registerCustomEditorProvider(VIEW_TYPE, new RtlEditorProvider(context, runFile, formatDocument), {
      webviewOptions: { retainContextWhenHidden: true },
      supportsMultipleEditorsPerDocument: true,
    });
  }

  resolveCustomTextEditor(document, panel) {
    const media = vscode.Uri.joinPath(this.context.extensionUri, "media");
    const fonts = vscode.Uri.joinPath(this.context.extensionUri, "fonts");
    panel.webview.options = { enableScripts: true, localResourceRoots: [media, fonts] };
    panel.webview.html = this.html(panel.webview, media, fonts);

    const post = (msg) => panel.webview.postMessage(msg);
    const sendText = () => post({ type: "update", text: document.getText() });
    const sendDiagnostics = () => post({ type: "diagnostics", diagnostics: diagnosticsFor(document.uri) });

    const subs = [
      vscode.workspace.onDidChangeTextDocument((e) => {
        if (e.document.uri.toString() === document.uri.toString() && e.contentChanges.length) sendText();
      }),
      vscode.languages.onDidChangeDiagnostics((e) => {
        if (e.uris.some((u) => u.toString() === document.uri.toString())) sendDiagnostics();
      }),
      panel.webview.onDidReceiveMessage((msg) => this.onMessage(document, panel, msg, sendText, sendDiagnostics)),
    ];
    panel.onDidDispose(() => subs.forEach((s) => s.dispose()));
  }

  async onMessage(document, panel, msg, sendText, sendDiagnostics) {
    switch (msg.type) {
      case "ready":
        sendText();
        sendDiagnostics();
        break;
      case "edit":
        if (msg.text !== document.getText()) await replaceAll(document, msg.text);
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
      case "openText":
        await vscode.commands.executeCommand("vscode.openWith", document.uri, "default", panel.viewColumn);
        break;
    }
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
      <textarea id="src" dir="rtl" wrap="off" spellcheck="false" autocapitalize="off" autocomplete="off" aria-label="Ḍād code"></textarea>
    </div>
  </div>
</div>
<footer>
  <span id="status" role="status"></span>
  <button id="run" type="button" title="Run this file (Ctrl/Cmd+Enter)">▶ شغّل</button>
  <button id="to-text" type="button" title="Open in VS Code's left-to-right text editor">Open as text</button>
</footer>
<script nonce="${nonce}" src="${uri(media, "rtl-editor.js")}"></script>
</body>
</html>`;
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
