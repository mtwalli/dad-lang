// The right-to-left editor for .dad files, running inside a VS Code webview.
// A transparent textarea (native right-to-left editing, caret, selection, input methods)
// sits on top of a coloured copy of the same text; both share every layout metric.
// The text lives in VS Code's document: edits are sent to the extension, and changes
// made elsewhere (undo, git, another editor) come back as "update" messages.
(function () {
  "use strict";
  const vscode = acquireVsCodeApi();
  const src = document.getElementById("src");
  const hl = document.getElementById("hl");
  const gutter = document.getElementById("gutter");
  const scroller = document.getElementById("scroller");
  const current = document.getElementById("current");
  const LINE = () => parseFloat(getComputedStyle(src).lineHeight);
  let diagnostics = [];
  let lastSent = null;
  let sendTimer = null;

  // ---------- highlighting (same rules as the VS Code grammar) ----------
  const KW = {};
  const kw = (cls, words) => words.split(" ").forEach((w) => (KW[w] = cls));
  kw("kw", "إذا وإلا طابق لكل في من إلى حتى بخطوة طالما أرجع توقف تابع حاول التقط ارفع");
  kw("decl", "ثابت متغير دالة نوع تعداد عقد امتداد استورد خاص خادم مكون حالة جدول مسار");
  kw("const", "صح خطأ عدم");
  kw("op", "و أو ليس هو ك");
  kw("this", "هذا");
  kw("type", "صحيح عشري نص منطقي قائمة قاموس");
  const BUILTINS = new Set(["اطبع", "مشكلة", "جذر", "مطلق", "أدنى", "أقصى"]);
  const NAMES_TYPE = new Set(["نوع", "تعداد", "عقد", "امتداد", "مكون", "جدول"]);
  const WORD = /[\p{L}\p{Mn}_][\p{L}\p{Mn}0-9_]*/uy;
  const NUM = /[0-9](?:[0-9_]*[0-9])?(?:\.[0-9](?:[0-9_]*[0-9])?)?/y;
  const BAD_DIGITS = /[٠-٩۰-۹٫٬]+/y;
  const OP = /=>|[?؟][.?؟]|\.\.|==|!=|<=|>=|[-+*/]=|&&|\|\||[-+*/%<>=!?؟:]/y;
  const BIDI = /[‪-‮⁦-⁩‎‏؜]/;

  const esc = (t) => t.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  const sticky = (re, s, i) => { re.lastIndex = i; const m = re.exec(s); return m ? m[0] : null; };

  // Tokens are produced as [start, end, class] over the whole text.
  function tokens(text) {
    const out = [];
    let i = 0, prev = "";
    const push = (a, b, c) => { if (b > a) out.push([a, b, c]); };
    function code(stopAtBrace) {
      let depth = 0;
      while (i < text.length) {
        const c = text[i];
        if (stopAtBrace && c === "}" && depth === 0) return;
        if (text.startsWith("//", i)) { let e = text.indexOf("\n", i); if (e < 0) e = text.length; push(i, e, "com"); i = e; continue; }
        if (text.startsWith("/*", i)) { let e = text.indexOf("*/", i + 2); e = e < 0 ? text.length : e + 2; push(i, e, "com"); i = e; continue; }
        if (c === '"' || c === "«") { string(); prev = ""; continue; }
        const w = sticky(WORD, text, i);
        if (w) {
          let cls = KW[w];
          if (!cls) {
            if (prev === "دالة") cls = "fn";
            else if (NAMES_TYPE.has(prev)) cls = "type";
            else if (/^\s*\(/.test(text.slice(i + w.length, i + w.length + 8))) cls = BUILTINS.has(w) ? "builtin" : "fn";
          }
          if (cls) push(i, i + w.length, cls);
          prev = w; i += w.length; continue;
        }
        const n = sticky(NUM, text, i);
        if (n) { push(i, i + n.length, "num"); i += n.length; prev = ""; continue; }
        const bd = sticky(BAD_DIGITS, text, i);
        if (bd) { push(i, i + bd.length, "bad"); i += bd.length; continue; }
        const o = sticky(OP, text, i);
        if (o) { push(i, i + o.length, "op"); i += o.length; continue; }
        if (BIDI.test(c)) { push(i, i + 1, "bad"); i++; continue; }
        if (c === "{") depth++;
        if (c === "}") depth--;
        if (!/\s/.test(c)) prev = "";
        i++;
      }
    }
    function string() {
      const triple = text.startsWith('"""', i);
      const close = triple ? '"""' : text[i] === "«" ? "»" : '"';
      let start = i;
      i += triple ? 3 : 1;
      while (i < text.length) {
        const c = text[i];
        if (!triple && c === "\n") break;
        if (text.startsWith(close, i)) { i += close.length; push(start, i, "str"); return; }
        if (c === "\\" && i + 1 < text.length) { push(start, i, "str"); push(i, i + 2, "esc"); i += 2; start = i; continue; }
        if (c === "{") {
          push(start, i, "str"); push(i, i + 1, "interp"); i++;
          code(true);
          if (text[i] === "}") { push(i, i + 1, "interp"); i++; }
          start = i; continue;
        }
        i++;
      }
      push(start, i, "str");
    }
    code(false);
    return out;
  }

  // ---------- rendering ----------
  function lineStarts(text) {
    const starts = [0];
    for (let i = 0; i < text.length; i++) if (text[i] === "\n") starts.push(i + 1);
    return starts;
  }

  function render() {
    const text = src.value;
    const starts = lineStarts(text);
    // Problems become ranges in the text, so they can be drawn as wavy underlines.
    const marks = diagnostics.map((d) => {
      const a = (starts[d.line] ?? text.length) + d.start;
      const b = Math.max((starts[d.endLine] ?? text.length) + d.end, a + 1);
      return [a, Math.min(b, text.length), d.severity === 0 ? "err" : "warn", d.message];
    });
    const toks = tokens(text);
    // Cut the text at every token and mark boundary, then wrap each piece.
    const cuts = new Set([0, text.length]);
    toks.forEach(([a, b]) => { cuts.add(a); cuts.add(b); });
    marks.forEach(([a, b]) => { cuts.add(a); cuts.add(b); });
    const points = [...cuts].sort((x, y) => x - y);
    let html = "", ti = 0;
    for (let p = 0; p < points.length - 1; p++) {
      const a = points[p], b = points[p + 1];
      while (ti < toks.length && toks[ti][1] <= a) ti++;
      const tok = ti < toks.length && toks[ti][0] <= a && toks[ti][1] >= b ? toks[ti][2] : "";
      const mark = marks.find(([x, y]) => x <= a && y >= b);
      const cls = [tok, mark ? mark[2] : ""].filter(Boolean).join(" ");
      const piece = esc(text.slice(a, b));
      html += cls ? `<span class="${cls}">${piece}</span>` : piece;
    }
    hl.innerHTML = html + "\n ";
    const errorLines = new Map();
    diagnostics.forEach((d) => errorLines.set(d.line, Math.min(errorLines.get(d.line) ?? 9, d.severity)));
    gutter.innerHTML = starts.map((_, n) => {
      const sev = errorLines.get(n);
      const cls = sev === undefined ? "" : sev === 0 ? " err" : " warn";
      return `<div class="ln${cls}">${n + 1}</div>`;
    }).join("");
    size();
    caretMoved();
  }

  // The textarea grows with its content, so both layers scroll together in one scroller.
  function size() {
    src.style.height = "0px";
    src.style.width = "0px";
    const w = Math.max(src.scrollWidth + 24, scroller.clientWidth - gutter.offsetWidth);
    const h = Math.max(src.scrollHeight, scroller.clientHeight);
    src.style.width = hl.style.width = w + "px";
    src.style.height = hl.style.height = h + "px";
  }

  function caretLine() {
    return src.value.slice(0, src.selectionStart).split("\n").length - 1;
  }

  function caretMoved() {
    const line = caretLine();
    const lh = LINE();
    const top = parseFloat(getComputedStyle(src).paddingTop) + line * lh;
    current.style.top = top + "px";
    current.style.height = lh + "px";
    gutter.querySelectorAll(".ln.cur").forEach((e) => e.classList.remove("cur"));
    const ln = gutter.children[line];
    if (ln) ln.classList.add("cur");
    // Keep the caret in view, since the textarea itself never scrolls.
    if (top < scroller.scrollTop) scroller.scrollTop = top - lh;
    else if (top + lh > scroller.scrollTop + scroller.clientHeight) scroller.scrollTop = top + 2 * lh - scroller.clientHeight;
    const d = diagnostics.find((x) => x.line === line);
    document.getElementById("status").textContent = d ? d.message : "";
    document.getElementById("status").className = d ? (d.severity === 0 ? "err" : "warn") : "";
    vscode.setState({ caret: src.selectionStart, scroll: scroller.scrollTop });
  }

  // ---------- editing ----------
  function edited() {
    render();
    clearTimeout(sendTimer);
    sendTimer = setTimeout(() => {
      lastSent = src.value;
      vscode.postMessage({ type: "edit", text: src.value });
    }, 120);
  }

  function insert(text) {
    // execCommand keeps the textarea's own undo stack consistent with typing.
    if (!document.execCommand("insertText", false, text)) {
      src.setRangeText(text, src.selectionStart, src.selectionEnd, "end");
    }
    edited();
  }

  src.addEventListener("input", edited);
  src.addEventListener("keydown", (e) => {
    const mod = e.metaKey || e.ctrlKey;
    if (e.key === "Tab" && !mod) { e.preventDefault(); insert("  "); return; }
    if (e.key === "Enter" && !mod && !e.altKey) {
      // Keep the indentation of the current line, one level more after "{".
      const before = src.value.slice(0, src.selectionStart);
      const lineText = before.slice(before.lastIndexOf("\n") + 1);
      let indent = (lineText.match(/^\s*/) || [""])[0];
      if (/[{(\[]\s*$/.test(lineText)) indent += "  ";
      e.preventDefault(); insert("\n" + indent); return;
    }
    if (mod && e.key === "Enter") { e.preventDefault(); vscode.postMessage({ type: "run" }); return; }
    if (mod && !e.shiftKey && e.key.toLowerCase() === "z") { e.preventDefault(); flush(); vscode.postMessage({ type: "undo" }); return; }
    if (mod && (e.key.toLowerCase() === "y" || (e.shiftKey && e.key.toLowerCase() === "z"))) { e.preventDefault(); flush(); vscode.postMessage({ type: "redo" }); return; }
    if (mod && e.key.toLowerCase() === "s") { flush(); }
  });
  ["keyup", "click", "select", "focus"].forEach((ev) => src.addEventListener(ev, caretMoved));
  window.addEventListener("resize", size);

  function flush() {
    if (sendTimer) { clearTimeout(sendTimer); sendTimer = null; lastSent = src.value; vscode.postMessage({ type: "edit", text: src.value }); }
  }

  document.getElementById("to-text").addEventListener("click", () => { flush(); vscode.postMessage({ type: "openText" }); });
  document.getElementById("run").addEventListener("click", () => { flush(); vscode.postMessage({ type: "run" }); });

  window.addEventListener("message", (event) => {
    const msg = event.data;
    if (msg.type === "update") {
      if (msg.text !== src.value && msg.text !== lastSent) {
        const caret = Math.min(src.selectionStart, msg.text.length);
        src.value = msg.text;
        src.setSelectionRange(caret, caret);
      }
      lastSent = null;
      render();
    } else if (msg.type === "diagnostics") {
      diagnostics = msg.diagnostics;
      render();
    } else if (msg.type === "reveal") {
      const starts = lineStarts(src.value);
      const pos = (starts[msg.line] ?? 0) + msg.character;
      src.focus();
      src.setSelectionRange(pos, pos);
      caretMoved();
    }
  });

  const saved = vscode.getState();
  render();
  if (saved) {
    src.setSelectionRange(saved.caret, saved.caret);
    scroller.scrollTop = saved.scroll;
  }
  src.focus();
  // Measure again once Ḍād Mono has loaded: widths change with the font.
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(render);
  vscode.postMessage({ type: "ready" });
})();
