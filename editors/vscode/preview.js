// The interface preview: a panel beside the editor that shows the program's
// interface while you write it, before running it. «dad preview» compiles the
// program to JavaScript, as «dad build» does, and the page runs it: buttons,
// fields, pages and server functions work, and tables are kept in the panel.
// It follows the code as you type; when the code has errors it keeps showing
// the last version that worked, with the errors listed on top. The program runs
// in a frame inside the panel's page, which has a phone-width button.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");
const crypto = require("crypto");
const path = require("path");

const DELAY_MS = 350; // after the last key
const TIMEOUT_MS = 15000;
const INSTALL_URL = "https://mtwalli.github.io/dad-lang/install/";

/** Runs dad with args, writing input to it; resolves to { code, stdout, stderr }. */
function runDad(exe, args, input) {
  return new Promise((resolve) => {
    let child;
    try {
      child = cp.spawn(exe, args, { windowsHide: true });
    } catch (e) {
      resolve({ code: -1, stdout: "", stderr: e.message });
      return;
    }
    let stdout = "";
    let stderr = "";
    const timer = setTimeout(() => child.kill(), TIMEOUT_MS);
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (d) => (stdout += d));
    child.stderr.on("data", (d) => (stderr += d));
    child.on("error", (e) => {
      clearTimeout(timer);
      resolve({ code: -1, stdout, stderr: e.message });
    });
    child.on("close", (code) => {
      clearTimeout(timer);
      resolve({ code, stdout, stderr });
    });
    child.stdin.on("error", () => {});
    child.stdin.end(input || "", "utf8");
  });
}

const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

/** What runs inside the program's frame: it keeps the page on screen across updates. */
const FRAME_SCRIPT =
  "try{var r=parent.dadRoute;if(r&&r!==\"/\")location.hash=r;" +
  "addEventListener(\"hashchange\",function(){try{parent.dadSaveRoute(decodeURIComponent(location.hash.replace(/^#/,\"\"))||\"/\")}catch(e){}})}catch(e){}";

/**
 * The preview panel's own page: the program runs in a frame, so its styles and
 * its width are its own (the phone button makes the frame as wide as a phone,
 * and the program's layout for phones applies). Around it: a small toolbar,
 * the problems (a click goes to the code), and a note when there is nothing to
 * show. The extension sends it messages: page, note, problems.
 */
function shell(nonce, csp) {
  return `<!doctype html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="${esc(csp)}">
<style>
  html, body { margin: 0; height: 100%; overflow: hidden; background: var(--vscode-editor-background); color: var(--vscode-foreground);
    font: 13px/1.6 var(--vscode-font-family, system-ui), sans-serif; }
  #app { display: block; width: 100%; height: 100%; border: 0; background: #fff; }
  #stage { height: 100%; }
  body.phone-width #stage { display: grid; place-items: center; background: var(--vscode-sideBar-background, #e9e9ee); }
  body.phone-width #app { width: 390px; max-width: calc(100% - 24px); height: min(844px, calc(100% - 64px)); border-radius: 22px;
    box-shadow: 0 0 0 9px #1c1c22, 0 10px 30px rgba(0,0,0,.35); }
  #note { position: fixed; inset: 0; display: grid; place-items: center; padding: 16px; text-align: center; background: var(--vscode-editor-background); }
  #note h1 { font-size: 1.15rem; margin: 0 0 .4rem; }
  #note p { margin: 0 0 1rem; opacity: .85; }
  #note code { font-family: var(--vscode-editor-font-family, monospace); }
  #note button { font: inherit; padding: 4px 14px; border: 0; border-radius: 3px; cursor: pointer;
    background: var(--vscode-button-background); color: var(--vscode-button-foreground); }
  [hidden], .tools:not(:has(button:not([hidden]))) { display: none !important; }
  .bar { position: fixed; inset-inline-start: 10px; bottom: 10px; z-index: 10; display: flex; flex-direction: column; align-items: flex-end; gap: 6px; }
  .tools { display: flex; gap: 2px; padding: 3px; border-radius: 8px; background: var(--vscode-editorWidget-background, #f3f3f3);
    color: var(--vscode-editorWidget-foreground, #333); box-shadow: 0 2px 8px rgba(0,0,0,.3); }
  .tools button { font: inherit; color: inherit; background: none; border: 0; cursor: pointer; padding: 3px 8px; border-radius: 5px; white-space: nowrap; }
  .tools button:hover, .tools button[aria-pressed="true"] { background: var(--vscode-toolbar-hoverBackground, rgba(127,127,127,.2)); }
  button:focus-visible { outline: 1px solid var(--vscode-focusBorder, #0078d4); }
  .problems { max-width: min(560px, calc(100vw - 20px)); max-height: 40vh; overflow: auto; border-radius: 8px;
    background: var(--vscode-editorWidget-background, #fff); color: var(--vscode-editorWidget-foreground, #333);
    border: 1px solid var(--vscode-editorError-foreground, #e51400); box-shadow: 0 2px 8px rgba(0,0,0,.3); }
  .problems p { margin: 0; padding: 6px 10px; border-bottom: 1px solid rgba(127,127,127,.2); }
  .problems button { display: block; width: 100%; text-align: start; font: inherit; color: inherit; background: none; border: 0;
    border-bottom: 1px solid rgba(127,127,127,.12); padding: 6px 10px; cursor: pointer; }
  .problems button:hover { background: var(--vscode-list-hoverBackground, rgba(127,127,127,.12)); }
  .where { opacity: .7; margin-inline-end: 6px; }
  .err { color: var(--vscode-editorError-foreground, #e51400); }
</style>
</head>
<body>
<div id="stage"><iframe id="app" title="معاينة الواجهة" hidden></iframe></div>
<div id="note" hidden></div>
<div class="bar">
  <div class="problems" hidden></div>
  <div class="tools">
    <button class="count err" hidden title="المعاينة تعرض آخر نسخة بلا أخطاء"></button>
    <button class="phone" title="عرض الهاتف / Phone width">📱</button>
    <button class="reset" title="امسح بيانات الجداول وأعد التحميل / Clear the tables and reload">↺</button>
  </div>
</div>
<script nonce="${nonce}">
(function () {
  "use strict";
  const vscode = acquireVsCodeApi();
  const state = vscode.getState() || {};
  const save = () => vscode.setState(state);
  const $ = (s) => document.querySelector(s);
  const frame = $("#app"), note = $("#note"), list = $(".problems"), count = $(".count");
  let html = "", problems = [];
  // the frame's page asks for the page to show, and says when it changes
  window.dadRoute = state.route || "/";
  window.dadSaveRoute = (route) => { state.route = window.dadRoute = route; save(); };
  const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);

  function show() {
    const errors = problems.filter((p) => !p.warning);
    count.hidden = errors.length === 0;
    count.textContent = "⚠ " + errors.length;
    list.hidden = errors.length === 0 || state.hideProblems;
    list.innerHTML = errors.length
      ? "<p>" + (html ? "تعرض المعاينة آخر نسخة بلا أخطاء. أصلح هذه لتتحدث:" : "أصلح هذه لتظهر المعاينة:") + "</p>" +
        errors.map((p, i) => '<button data-i="' + i + '"><span class="where">' + esc(p.where) + "</span>" + esc(p.message) + "</button>").join("")
      : "";
    list.querySelectorAll("button").forEach((b) => {
      const p = errors[Number(b.dataset.i)];
      b.onclick = () => vscode.postMessage({ type: "goto", path: p.path, line: p.line, col: p.col });
    });
  }
  function phone() {
    document.body.classList.toggle("phone-width", !!state.phone);
    $(".phone").setAttribute("aria-pressed", String(!!state.phone));
  }
  function load() {
    frame.removeAttribute("srcdoc");
    frame.srcdoc = html;
  }
  count.onclick = () => { state.hideProblems = !state.hideProblems; save(); show(); };
  $(".phone").onclick = () => { state.phone = !state.phone; save(); phone(); };
  $(".reset").onclick = () => {
    try { Object.keys(localStorage).filter((k) => k.startsWith("dad:")).forEach((k) => localStorage.removeItem(k)); } catch (e) {}
    window.dadSaveRoute("/");
    if (html) load();
  };
  window.addEventListener("message", (e) => {
    const m = e.data || {};
    if (m.problems) {
      problems = m.problems;
      if (problems.some((p) => !p.warning)) state.hideProblems = false;
    }
    if (m.type === "page") {
      html = m.html;
      note.hidden = true;
      frame.hidden = false;
      $(".phone").hidden = $(".reset").hidden = false;
      load();
    } else if (m.type === "note") {
      html = "";
      frame.hidden = true;
      frame.removeAttribute("srcdoc");
      $(".phone").hidden = $(".reset").hidden = true;
      note.innerHTML = "<div><h1>" + esc(m.title) + "</h1><p>" + m.text + "</p>" +
        (m.actions || []).map((a) => '<button data-action="' + esc(a[0]) + '">' + esc(a[1]) + "</button>").join(" ") + "</div>";
      note.querySelectorAll("[data-action]").forEach((b) => (b.onclick = () => vscode.postMessage({ type: b.dataset.action })));
      note.hidden = false;
    }
    show();
  });
  phone();
  show();
  vscode.postMessage({ type: "ready" });
})();
</script>
</body>
</html>`;
}

class Preview {
  constructor(context, dadPath, reveal) {
    this.context = context;
    this.dadPath = dadPath;
    this.reveal = reveal;
    this.panels = new Map(); // document uri → { uri, panel, timer, version, html, nonce }
    this.assets = new Map(); // dad path → Promise<boolean>: its assets are written
  }

  /** Registers «Ḍād: Preview Interface»; activeUri() is the .dad file in front. */
  static register(context, dadPath, activeUri, reveal) {
    const p = new Preview(context, dadPath, reveal);
    const all = (delay) => p.panels.forEach((pv) => p.schedule(pv, delay));
    context.subscriptions.push(
      vscode.commands.registerCommand("dad.preview", (uri) => {
        const target = uri instanceof vscode.Uri ? uri : activeUri();
        if (target) return p.open(target);
        vscode.window.showInformationMessage("Ḍād: open a .dad file to preview its interface.");
      }),
      vscode.workspace.onDidChangeTextDocument((e) => {
        const pv = p.panels.get(e.document.uri.toString());
        if (pv && e.contentChanges.length) p.schedule(pv, DELAY_MS);
      }),
      // the files a program imports, and its stylesheet, are read from the disk
      vscode.workspace.onDidSaveTextDocument((doc) => {
        if (doc.languageId === "dad" || /\.css$/i.test(doc.uri.fsPath || "")) all(50);
      })
    );
    if (vscode.workspace.createFileSystemWatcher) {
      const w = vscode.workspace.createFileSystemWatcher("**/*.{css,dad}");
      context.subscriptions.push(w, w.onDidChange(() => all(200)), w.onDidCreate(() => all(200)));
    }
    return p;
  }

  get assetsDir() {
    return vscode.Uri.joinPath(this.context.globalStorageUri, "preview-assets");
  }

  /** Opens the preview of a file beside it, or shows the one already open. */
  async open(uri) {
    const key = uri.toString();
    const open = this.panels.get(key);
    if (open) {
      open.panel.reveal(undefined, true);
      return open;
    }
    const panel = vscode.window.createWebviewPanel(
      "dad.preview",
      "معاينة · " + path.basename(uri.fsPath || uri.path || "", ".dad"),
      { viewColumn: vscode.ViewColumn.Beside, preserveFocus: true },
      { enableScripts: true, retainContextWhenHidden: true, localResourceRoots: [this.assetsDir] }
    );
    panel.iconPath = vscode.Uri.joinPath(this.context.extensionUri, "images", "file.svg");
    const pv = { uri, panel, timer: null, version: 0, html: "", nonce: crypto.randomBytes(16).toString("base64"), shown: null, ready: false };
    this.panels.set(key, pv);
    panel.webview.html = shell(pv.nonce, this.csp(pv));
    panel.onDidDispose(() => {
      clearTimeout(pv.timer);
      this.panels.delete(key);
    });
    panel.webview.onDidReceiveMessage((msg) => this.onMessage(pv, msg));
    await this.render(pv);
    return pv;
  }

  schedule(pv, delay) {
    clearTimeout(pv.timer);
    pv.timer = setTimeout(() => this.render(pv), delay);
  }

  /** Sends what the panel shows; it is sent again if the panel's page starts over. */
  send(pv, msg) {
    if (msg.type !== "problems") pv.shown = msg;
    else if (pv.shown) pv.shown = Object.assign({}, pv.shown, { problems: msg.problems });
    if (pv.ready) pv.panel.webview.postMessage(msg);
  }

  async onMessage(pv, msg) {
    switch (msg.type) {
      case "ready":
        pv.ready = true;
        if (pv.shown) pv.panel.webview.postMessage(pv.shown);
        break;
      case "goto": {
        const uri = msg.path ? vscode.Uri.file(msg.path) : pv.uri;
        const pos = new vscode.Position(Math.max(msg.line - 1, 0), Math.max(msg.col - 1, 0));
        if (this.reveal) await this.reveal(uri, new vscode.Range(pos, pos));
        break;
      }
      case "reload":
        pv.html = "";
        await this.render(pv);
        break;
      case "run":
        await vscode.commands.executeCommand("dad.run", pv.uri);
        break;
      case "install":
        vscode.env.openExternal(vscode.Uri.parse(INSTALL_URL));
        break;
    }
  }

  /** Writes ui.js, ui.css, the fonts and the runtime where the pages can load them, once per dad. */
  ensureAssets(exe) {
    if (!this.assets.has(exe)) {
      const p = runDad(exe, ["preview", "--assets", this.assetsDir.fsPath], "").then((r) => r.code === 0);
      this.assets.set(exe, p);
      p.then((ok) => ok || this.assets.delete(exe)); // try again next time
    }
    return this.assets.get(exe);
  }

  /** The pages' Content-Security-Policy: their own assets, and only scripts with the panel's nonce. */
  csp(pv) {
    const src = pv.panel.webview.cspSource;
    return `default-src 'none'; img-src ${src} https: data: blob:; media-src ${src} https: data:; font-src ${src} data:; ` +
      `style-src ${src} 'unsafe-inline'; script-src 'nonce-${pv.nonce}'; frame-src ${src} 'self' about:; connect-src https:;`;
  }

  async render(pv) {
    const version = ++pv.version;
    const { panel } = pv;
    const webview = panel.webview;
    const exe = this.dadPath();
    if (!exe) {
      return this.message(pv, "لم يُعثر على برنامج ضاد", "ثبّت ضاد، أو اضبط الإعداد <code>dad.path</code>، ثم أعد فتح المعاينة.", [["install", "كيف أثبّته"]]);
    }
    if (!(await this.ensureAssets(exe))) {
      return this.message(pv, "المعاينة تحتاج ضاد أحدث", "نسختك من <code>dad</code> لا تعرف الأمر <code>dad preview</code>. حدّثها إلى آخر إصدار.", [["install", "كيف أحدّثها"]]);
    }
    let doc;
    try {
      doc = await vscode.workspace.openTextDocument(pv.uri);
    } catch {
      return;
    }
    const base = webview.asWebviewUri(this.assetsDir).toString().replace(/\/?$/, "/");
    const res = await runDad(exe, ["preview", "--path", doc.uri.fsPath, "--base", base, "--nonce", pv.nonce, "--csp", this.csp(pv), "-"], doc.getText());
    if (version !== pv.version || !this.panels.has(pv.uri.toString())) return; // a newer render started, or the panel closed
    let out;
    try {
      out = JSON.parse(res.stdout);
    } catch {
      return this.message(pv, "تعذّرت المعاينة", esc(res.stderr.trim() || "لم يُجب ضاد."), []);
    }
    const here = path.resolve(doc.uri.fsPath);
    const problems = (out.diagnostics || []).map((d) => ({
      path: d.path,
      line: d.line,
      col: d.col,
      message: d.message,
      warning: d.warning,
      where: (d.path && path.resolve(d.path) !== here ? path.basename(d.path) + " · " : "") + "السطر " + d.line,
    }));
    const errors = problems.filter((p) => !p.warning);
    if (errors.length) {
      // keep the last version that worked on screen, with the errors over it
      if (pv.html) return this.send(pv, { type: "problems", problems });
      return this.message(pv, "في الكود أخطاء", "تظهر المعاينة حين تُصلح.", [], problems);
    }
    if (!out.site) {
      return this.message(pv, "لا واجهة في هذا البرنامج",
        "المعاينة لما يعرضه <code>مكون الرئيسية()</code> أو <code>صفحة</code>. أما هذا البرنامج فشغّله لترى ما يطبعه.", [["run", "▶ شغّل"]]);
    }
    // In the frame, links such as «#/عن» stay in the frame's own page.
    const html = out.html
      .replace("<head>\n", () => '<head>\n<base href="about:srcdoc">\n')
      .replace("<body>\n", () => `<body>\n<script nonce="${pv.nonce}">${FRAME_SCRIPT}</script>\n`);
    if (html === pv.html) return this.send(pv, { type: "problems", problems });
    pv.html = html;
    this.send(pv, { type: "page", html, problems });
  }

  /** A note instead of the interface. text is HTML; actions are [message type, label] buttons. */
  message(pv, title, text, actions, problems = []) {
    pv.html = "";
    this.send(pv, { type: "note", title, text, actions, problems });
  }
}

module.exports = { Preview, runDad, shell, FRAME_SCRIPT };
