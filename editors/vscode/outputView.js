// The "Ḍād" panel next to the terminal: runs a .dad file and shows its output right to left,
// in Ḍād Mono, until the next run. VS Code's terminal can't lay out Arabic right to left
// and closes when a program ends, so output gets its own view.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");
const crypto = require("crypto");
const path = require("path");

const VIEW_ID = "dad.output";
const MAX_OUTPUT = 1 << 20; // stop showing output after 1 MB, so a runaway loop can't freeze VS Code

class OutputView {
  constructor(context) {
    this.context = context;
    this.view = null;
    this.ready = null;
    this.child = null;
    this.last = null; // { exe, file } of the last run, for "Run again"
    this.readyResolve = null;
    this.newReady();
  }

  newReady() {
    this.ready = new Promise((resolve) => (this.readyResolve = resolve));
  }

  static register(context) {
    const view = new OutputView(context);
    const reg = vscode.window.registerWebviewViewProvider(VIEW_ID, view, {
      webviewOptions: { retainContextWhenHidden: true },
    });
    return { view, disposable: reg };
  }

  resolveWebviewView(webviewView) {
    this.view = webviewView;
    const fonts = vscode.Uri.joinPath(this.context.extensionUri, "fonts");
    const media = vscode.Uri.joinPath(this.context.extensionUri, "media");
    webviewView.webview.options = { enableScripts: true, localResourceRoots: [fonts, media] };
    webviewView.webview.html = this.html(webviewView.webview, fonts, media);
    webviewView.webview.onDidReceiveMessage((msg) => {
      if (msg.type === "ready") this.readyResolve();
      else if (msg.type === "rerun" && this.last) this.run(this.last.exe, this.last.file);
      else if (msg.type === "stop") this.stop();
    });
    webviewView.onDidDispose(() => {
      this.view = null;
      this.newReady();
    });
  }

  post(msg) {
    if (this.view) this.view.webview.postMessage(msg);
  }

  stop() {
    if (this.child) this.child.kill();
  }

  /** Runs `dad run file` and streams its output into the panel. */
  async run(exe, file) {
    this.stop();
    this.last = { exe, file };
    if (!this.view) await vscode.commands.executeCommand(`${VIEW_ID}.focus`);
    else this.view.show(true);
    await this.ready;
    const started = Date.now();
    let shown = 0;
    let truncated = false;
    this.post({ type: "start", file: path.basename(file) });
    let child;
    try {
      // The program runs in its own folder, so a relative path to dad (a setting
      // like "./bin/dad") is resolved first, the same way «Check» finds it.
      const cmd = /[\\/]/.test(exe) && !path.isAbsolute(exe) ? path.resolve(exe) : exe;
      child = cp.spawn(cmd, ["run", file], { cwd: path.dirname(file), windowsHide: true });
    } catch (err) {
      this.post({ type: "chunk", stream: "err", text: err.message + "\n" });
      this.post({ type: "done", code: -1, ms: 0 });
      return;
    }
    this.child = child;
    const send = (stream) => (data) => {
      if (this.child !== child) return;
      if (shown > MAX_OUTPUT) {
        if (!truncated) {
          truncated = true;
          this.post({ type: "chunk", stream: "err", text: "\n… توقف العرض: المخرجات أطول من 1 ميغابايت\n" });
        }
        return;
      }
      shown += data.length;
      this.post({ type: "chunk", stream, text: data });
    };
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", send("out"));
    child.stderr.on("data", send("err"));
    return new Promise((resolve) => {
      let finished = false;
      const finish = (code, signal) => {
        if (finished) return;
        finished = true;
        if (this.child === child) this.child = null;
        this.post({ type: "done", code, stopped: Boolean(signal), ms: Date.now() - started });
        resolve(code);
      };
      child.on("error", (err) => {
        this.post({ type: "chunk", stream: "err", text: err.message + "\n" });
        finish(-1, null);
      });
      child.on("close", finish);
    });
  }

  html(webview, fonts, media) {
    const nonce = crypto.randomBytes(16).toString("base64");
    const uri = (base, f) => webview.asWebviewUri(vscode.Uri.joinPath(base, f));
    return `<!doctype html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; font-src ${webview.cspSource}; script-src 'nonce-${nonce}';">
<style>
@font-face { font-family: "Dad Mono"; src: url("${uri(fonts, "DadMono-Regular.ttf")}") format("truetype"); }
</style>
<link rel="stylesheet" href="${uri(media, "output.css")}">
</head>
<body>
<header>
  <span id="title">اضغط ▶ لتشغيل ملف .dad</span>
  <span id="state"></span>
  <button id="stop" type="button" hidden>■ أوقف</button>
  <button id="rerun" type="button" hidden>▶ شغّل مجدداً</button>
</header>
<pre id="out" aria-live="polite"></pre>
<script nonce="${nonce}" src="${uri(media, "output.js")}"></script>
</body>
</html>`;
  }
}

module.exports = { OutputView, VIEW_ID };
