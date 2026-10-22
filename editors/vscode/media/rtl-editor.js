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
  let changes = []; // lines changed since git last saw the file: { kind, start, end, old }
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
  const BUILTINS = new Set(["اطبع", "مشكلة", "جذر", "مطلق", "أدنى", "أقصى", "انتقل", "سمة",
    "عمود", "صف", "بطاقة", "عنوان", "فقرة", "زر", "حقل", "مربع_اختيار", "صورة", "رابط", "فاصل",
    "قائمة_منسدلة", "اختيار", "منطقة_نص", "جدول_عرض", "قائمة_عناصر", "ألسنة", "لسان", "نافذة", "تنبيه", "أيقونة", "نموذج"]);
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
          // «صفحة» is a keyword only before an address: صفحة "/المهام" = …
          if (!cls && w === "صفحة" && /^[ \t]*"/.test(text.slice(i + w.length, i + w.length + 8))) cls = "decl";
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
    // git: a bar beside added and changed lines, a mark where lines were removed
    const changed = new Map();
    changes.forEach((c, i) => {
      if (c.kind === "deleted") changed.set(Math.min(c.start, starts.length - 1), [c.start >= starts.length ? "g-del-end" : "g-del", i]);
      else for (let n = c.start; n < c.end; n++) changed.set(n, [c.kind === "added" ? "g-add" : "g-mod", i]);
    });
    gutter.innerHTML = starts.map((_, n) => {
      const sev = errorLines.get(n);
      let cls = sev === undefined ? "" : sev === 0 ? " err" : " warn";
      const g = changed.get(n);
      if (g) cls += " " + g[0];
      return `<div class="ln${cls}"${g ? ` data-change="${g[1]}" title="${CHANGE_TITLE[g[0]]}"` : ""}>${n + 1}</div>`;
    }).join("");
    const count = document.getElementById("changes");
    count.hidden = changes.length === 0;
    count.textContent = "± " + changes.length;
    count.title = changes.length + (changes.length === 1 ? " change" : " changes") + " since the last commit — click for the next (Alt+F5)";
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

  // ---------- positions ----------

  /** The line and UTF-16 column of an offset, as VS Code counts them. */
  function position(offset) {
    const before = src.value.slice(0, offset);
    const line = before.split("\n").length - 1;
    return { line, character: offset - (before.lastIndexOf("\n") + 1) };
  }

  /** Tells the extension where the caret is, for the commands of the context menu. */
  function postCaret() {
    vscode.postMessage(Object.assign({ type: "caret" }, position(src.selectionStart)));
  }

  /** The offset in the text under a point of the window, or null. */
  function offsetAt(x, y) {
    // The coloured layer lies under the textarea: look through to it.
    src.style.pointerEvents = "none";
    hl.style.pointerEvents = "auto";
    let node = null, off = 0;
    try {
      if (document.caretPositionFromPoint) {
        const p = document.caretPositionFromPoint(x, y);
        if (p) { node = p.offsetNode; off = p.offset; }
      } else if (document.caretRangeFromPoint) {
        const r = document.caretRangeFromPoint(x, y);
        if (r) { node = r.startContainer; off = r.startOffset; }
      }
    } finally {
      src.style.pointerEvents = "";
      hl.style.pointerEvents = "";
    }
    if (!node || !hl.contains(node)) return null;
    if (node.nodeType !== Node.TEXT_NODE) {
      // between elements: the start of the child at off
      const child = node.childNodes[off];
      if (!child) return src.value.length;
      node = child; off = 0;
      while (node.firstChild) node = node.firstChild;
    }
    let total = 0;
    const walker = document.createTreeWalker(hl, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      if (walker.currentNode === node) return Math.min(total + off, src.value.length);
      total += walker.currentNode.length;
    }
    return null;
  }

  /** A DOM range over the coloured layer for [a, b) of the text. */
  function domRange(a, b) {
    const r = document.createRange();
    let total = 0, started = false;
    const walker = document.createTreeWalker(hl, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) {
      const n = walker.currentNode, len = n.length;
      if (!started && a <= total + len) { r.setStart(n, a - total); started = true; }
      if (started && b <= total + len) { r.setEnd(n, b - total); return r; }
      total += len;
    }
    return null;
  }

  // ---------- going to definitions ----------

  const NAME_CHAR = /[\p{L}\p{Mn}0-9_]/u;
  const link = document.getElementById("link");

  /** What a Cmd/Ctrl+click at an offset would follow: a name, or the file an «استورد» names. */
  function linkAt(offset) {
    const text = src.value;
    const ls = text.lastIndexOf("\n", offset - 1) + 1;
    const le = text.indexOf("\n", offset) < 0 ? text.length : text.indexOf("\n", offset);
    const line = text.slice(ls, le);
    const col = offset - ls;
    if (/^\s*استورد\s/.test(line)) {
      const m = /"[^"]*"|«[^»]*»/.exec(line.slice(line.indexOf(" من ") + 1));
      if (m) {
        const start = line.indexOf(m[0], line.indexOf(" من "));
        if (col >= start && col <= start + m[0].length) return [ls + start, ls + start + m[0].length];
      }
    }
    let a = offset, b = offset;
    while (a > ls && NAME_CHAR.test(text[a - 1])) a--;
    while (b < le && NAME_CHAR.test(text[b])) b++;
    if (a === b || /^[0-9]/.test(text.slice(a, b))) return null;
    const word = text.slice(a, b);
    if (KW[word] || (word === "صفحة" && /^[ \t]*"/.test(text.slice(b, b + 8)))) return null;
    return [a, b];
  }

  function showLink(x, y) {
    const offset = offsetAt(x, y);
    const span = offset === null ? null : linkAt(offset);
    const r = span && domRange(span[0], span[1]);
    const rect = r && r.getBoundingClientRect();
    if (!rect || !rect.width) return hideLink();
    const box = link.parentElement.getBoundingClientRect();
    Object.assign(link.style, {
      display: "block",
      top: rect.bottom - box.top - 2 + "px",
      left: rect.left - box.left + "px",
      width: rect.width + "px",
    });
    src.classList.add("linking");
  }

  function hideLink() {
    link.style.display = "none";
    src.classList.remove("linking");
  }

  let mouse = null;
  src.addEventListener("mousemove", (e) => {
    mouse = { x: e.clientX, y: e.clientY };
    if (e.metaKey || e.ctrlKey) showLink(e.clientX, e.clientY);
    else hideLink();
  });
  src.addEventListener("mouseleave", () => { mouse = null; hideLink(); });
  window.addEventListener("keydown", (e) => { if ((e.key === "Meta" || e.key === "Control") && mouse) showLink(mouse.x, mouse.y); });
  window.addEventListener("keyup", (e) => { if (e.key === "Meta" || e.key === "Control") hideLink(); });
  window.addEventListener("blur", hideLink);

  src.addEventListener("click", (e) => {
    if (!(e.metaKey || e.ctrlKey) || e.button !== 0) return;
    const offset = offsetAt(e.clientX, e.clientY);
    if (offset === null || !linkAt(offset)) return;
    e.preventDefault();
    hideLink();
    src.setSelectionRange(offset, offset);
    caretMoved();
    flush();
    vscode.postMessage(Object.assign({ type: "definition" }, position(offset)));
  });

  // A right click puts the caret where it points, so the menu's commands act there.
  src.addEventListener("contextmenu", (e) => {
    const offset = offsetAt(e.clientX, e.clientY);
    if (offset !== null && (offset < src.selectionStart || offset > src.selectionEnd)) {
      src.setSelectionRange(offset, offset);
      caretMoved();
    }
    flush();
    postCaret();
  });

  // ---------- git changes ----------

  const CHANGE_TITLE = { "g-add": "سطر جديد", "g-mod": "سطر تغيّر", "g-del": "حُذفت أسطر هنا", "g-del-end": "حُذفت أسطر هنا" };
  const peek = document.getElementById("peek");
  let peeked = -1;

  /** Shows what a change replaced, under it, with buttons to undo it or open the diff. */
  function openPeek(i) {
    const c = changes[i];
    if (!c) return closePeek();
    peeked = i;
    const label = c.kind === "added" ? "أسطر جديدة منذ آخر حفظ في git" : c.kind === "deleted" ? "أسطر حُذفت:" : "كانت هذه الأسطر:";
    peek.innerHTML =
      `<div class="peek-head"><span>${label}</span>` +
      `<button data-do="prev" title="Previous change (Shift+Alt+F5)">▲</button>` +
      `<button data-do="next" title="Next change (Alt+F5)">▼</button>` +
      `<button data-do="revert" title="Put the old lines back">↩ تراجع</button>` +
      `<button data-do="diff" title="Open the file's changes">الفرق</button>` +
      `<button data-do="close" title="Close (Esc)">✕</button></div>` +
      (c.old.length ? `<pre>${c.old.map((l) => esc(l) || " ").join("\n")}</pre>` : "");
    const lh = LINE();
    const pad = parseFloat(getComputedStyle(src).paddingTop);
    peek.style.top = pad + Math.max(c.end, c.start) * lh + "px";
    peek.hidden = false;
    peek.querySelectorAll("button").forEach((b) => (b.onclick = (e) => {
      e.stopPropagation();
      const what = b.dataset.do;
      if (what === "close") closePeek();
      else if (what === "next" || what === "prev") goToChange(what === "next" ? 1 : -1, peeked);
      else if (what === "revert") { flush(); vscode.postMessage({ type: "revertChange", kind: c.kind, start: c.start, end: c.end }); closePeek(); }
      else if (what === "diff") vscode.postMessage({ type: "openChanges" });
    }));
    const r = peek.getBoundingClientRect(), box = scroller.getBoundingClientRect();
    if (r.bottom > box.bottom) scroller.scrollTop += r.bottom - box.bottom + lh;
  }

  function closePeek() {
    peek.hidden = true;
    peeked = -1;
  }

  /** Moves to the next (by = 1) or previous (by = -1) change after the caret, or after change from. */
  function goToChange(by, from = -1) {
    if (!changes.length) return;
    let i;
    if (from >= 0) i = (from + by + changes.length) % changes.length;
    else {
      const line = caretLine();
      i = by > 0 ? changes.findIndex((c) => c.start > line) : changes.map((c) => c.start < line).lastIndexOf(true);
      if (i < 0) i = by > 0 ? 0 : changes.length - 1;
    }
    const starts = lineStarts(src.value);
    const pos = starts[Math.min(changes[i].start, starts.length - 1)];
    src.focus();
    src.setSelectionRange(pos, pos);
    caretMoved();
    postCaret();
    openPeek(i);
  }

  gutter.addEventListener("click", (e) => {
    const ln = e.target.closest(".ln[data-change]");
    if (!ln) return;
    const i = Number(ln.dataset.change);
    if (i === peeked) closePeek();
    else openPeek(i);
  });
  document.getElementById("changes").addEventListener("click", () => goToChange(1));

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

  // ---------- indentation ----------
  const UNIT = "    "; // four spaces, as dad fmt
  const CLOSERS = { "}": "{", ")": "(", "]": "[" };

  /** Replaces src.value[start:end] with text (undoably) and selects [selStart, selEnd]. */
  function replace(start, end, text, selStart, selEnd = selStart) {
    src.setSelectionRange(start, end);
    insert(text);
    src.setSelectionRange(selStart, selEnd);
    caretMoved();
  }

  function lineStartAt(pos) { return src.value.lastIndexOf("\n", pos - 1) + 1; }
  function lineEndAt(pos) { const i = src.value.indexOf("\n", pos); return i < 0 ? src.value.length : i; }
  function indentOf(line) { return (line.match(/^ */) || [""])[0]; }

  /** The indentation of the line holding the bracket that ch at pos would close. */
  function openerIndent(pos, ch) {
    const want = CLOSERS[ch];
    let depth = 0;
    for (let i = pos - 1; i >= 0; i--) {
      const c = src.value[i];
      if (c in CLOSERS) depth++;
      else if (c === "{" || c === "(" || c === "[") {
        if (depth === 0) return c === want ? indentOf(src.value.slice(lineStartAt(i), lineEndAt(i))) : null;
        depth--;
      }
    }
    return null;
  }

  /** Indents (by = 1) or outdents (by = -1) every line touched by the selection. */
  function shiftLines(by) {
    const a = src.selectionStart, b = src.selectionEnd;
    const start = lineStartAt(a);
    const end = lineEndAt(b > a && src.value[b - 1] === "\n" ? b - 1 : b);
    const lines = src.value.slice(start, end).split("\n");
    let firstDelta = 0, total = 0;
    const out = lines.map((l, i) => {
      let next;
      if (by > 0) next = l.trim() === "" ? l : UNIT + l;
      else next = l.replace(new RegExp("^ {1," + UNIT.length + "}"), "");
      if (i === 0) firstDelta = next.length - l.length;
      total += next.length - l.length;
      return next;
    });
    const text = out.join("\n");
    if (text === src.value.slice(start, end)) return;
    const selStart = Math.max(start, a + firstDelta);
    replace(start, end, text, a === b ? selStart : start, a === b ? selStart : start + text.length);
  }

  src.addEventListener("keydown", (e) => {
    const mod = e.metaKey || e.ctrlKey;
    const a = src.selectionStart, b = src.selectionEnd;
    const ls = lineStartAt(a);
    const before = src.value.slice(ls, a);

    if (e.key === "Tab" && !mod) {
      e.preventDefault();
      if (e.shiftKey) shiftLines(-1);
      else if (src.value.slice(a, b).includes("\n")) shiftLines(1);
      else insert(UNIT.slice(0, UNIT.length - (before.length % UNIT.length) || UNIT.length));
      return;
    }
    if (mod && (e.key === "]" || e.key === "[")) { e.preventDefault(); shiftLines(e.key === "]" ? 1 : -1); return; }

    if (e.key === "Enter" && !mod && !e.altKey) {
      e.preventDefault();
      // Keep the line's indentation, one level more after an opening bracket.
      // Between "{" and "}" the closing bracket goes on its own line.
      const after = src.value.slice(b, lineEndAt(b));
      const head = before.replace(/ +$/, "");
      let indent = indentOf(before);
      const opens = /[{(\[]$/.test(head);
      if (opens) indent += UNIT;
      const closer = after.trimStart()[0];
      if (opens && closer && CLOSERS[closer] === head[head.length - 1]) {
        const back = indent.slice(UNIT.length);
        const text = "\n" + indent + "\n" + back;
        const start = ls + head.length;
        replace(start, b + (after.length - after.trimStart().length), text, start + 1 + indent.length);
      } else {
        replace(ls + head.length, b, "\n" + indent, ls + head.length + 1 + indent.length);
      }
      return;
    }

    if (e.key in CLOSERS && !mod && a === b && /^ *$/.test(before)) {
      // A closing bracket at the start of a line lines up with its opening line.
      const target = openerIndent(a, e.key);
      if (target !== null && target !== before) {
        e.preventDefault();
        replace(ls, a, target + e.key, ls + target.length + 1);
        return;
      }
    }

    if (e.key === "Backspace" && !mod && a === b && before.length > 0 && /^ +$/.test(before)) {
      // In the indentation, Backspace goes back one level.
      e.preventDefault();
      const n = before.length % UNIT.length || UNIT.length;
      replace(a - n, a, "", a - n);
      return;
    }

    if ((e.shiftKey && e.altKey && e.code === "KeyF") || (mod && e.shiftKey && e.code === "KeyI")) {
      e.preventDefault(); flush(); vscode.postMessage({ type: "format" }); return;
    }
    if (mod && e.key === "Enter") { e.preventDefault(); vscode.postMessage({ type: "run" }); return; }
    if (e.key === "F12" && !mod && !e.altKey) {
      e.preventDefault(); flush();
      vscode.postMessage(Object.assign({ type: e.shiftKey ? "references" : "definition" }, position(src.selectionStart)));
      return;
    }
    if (e.key === "F5" && e.altKey && !mod) { e.preventDefault(); goToChange(e.shiftKey ? -1 : 1); return; }
    if (e.key === "Escape" && !peek.hidden) { e.preventDefault(); closePeek(); return; }
    if (e.key === "F2" && !mod && !e.shiftKey && !e.altKey) {
      e.preventDefault(); flush();
      vscode.postMessage(Object.assign({ type: "rename" }, position(src.selectionStart)));
      return;
    }
    if (mod && !e.shiftKey && e.key.toLowerCase() === "z") { e.preventDefault(); flush(); vscode.postMessage({ type: "undo" }); return; }
    if (mod && (e.key.toLowerCase() === "y" || (e.shiftKey && e.key.toLowerCase() === "z"))) { e.preventDefault(); flush(); vscode.postMessage({ type: "redo" }); return; }
    if (mod && e.key.toLowerCase() === "s") { flush(); }
  });
  ["keyup", "click", "select", "focus"].forEach((ev) => src.addEventListener(ev, () => { caretMoved(); postCaret(); }));
  window.addEventListener("resize", size);

  function flush() {
    if (sendTimer) { clearTimeout(sendTimer); sendTimer = null; lastSent = src.value; vscode.postMessage({ type: "edit", text: src.value }); }
  }

  document.getElementById("to-text").addEventListener("click", () => { flush(); vscode.postMessage({ type: "openText" }); });
  document.getElementById("run").addEventListener("click", () => { flush(); vscode.postMessage({ type: "run" }); });
  document.getElementById("preview").addEventListener("click", () => { flush(); vscode.postMessage({ type: "preview" }); });

  window.addEventListener("message", (event) => {
    const msg = event.data;
    if (msg.type === "update") {
      if (msg.text !== src.value && msg.text !== lastSent) {
        keepCaret(src.value, msg.text);
      }
      lastSent = null;
      render();
    } else if (msg.type === "diagnostics") {
      diagnostics = msg.diagnostics;
      render();
    } else if (msg.type === "changes") {
      changes = msg.changes;
      closePeek();
      render();
    } else if (msg.type === "reveal") {
      // Go to a place (a definition, a reference, a symbol), selecting it and
      // showing its line a third of the way down.
      const starts = lineStarts(src.value);
      const at = (line, ch) => Math.min((starts[line] ?? src.value.length) + ch, src.value.length);
      const a = at(msg.line, msg.character);
      const b = msg.endLine === undefined ? a : at(msg.endLine, msg.endCharacter);
      if (msg.focus !== false) src.focus();
      src.setSelectionRange(a, b);
      const top = parseFloat(getComputedStyle(src).paddingTop) + msg.line * LINE();
      scroller.scrollTop = Math.max(0, top - scroller.clientHeight / 3);
      caretMoved();
      postCaret();
    }
  });

  /** Sets the new text, keeping the caret on its line and after the same code. */
  function keepCaret(oldText, newText) {
    const pos = src.selectionStart;
    const oldStarts = lineStarts(oldText);
    let line = oldText.slice(0, pos).split("\n").length - 1;
    const oldLine = oldText.slice(oldStarts[line], pos);
    const typed = oldLine.replace(/\s+/g, "").length; // code characters before the caret
    src.value = newText;
    const starts = lineStarts(newText);
    line = Math.min(line, starts.length - 1);
    const lineText = newText.slice(starts[line], newText.indexOf("\n", starts[line]) < 0 ? newText.length : newText.indexOf("\n", starts[line]));
    let col = 0;
    if (typed === 0) col = indentOf(lineText).length;
    for (let seen = 0; seen < typed && col < lineText.length; col++) {
      if (!/\s/.test(lineText[col])) seen++;
    }
    const caret = starts[line] + col;
    src.setSelectionRange(caret, caret);
  }

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
