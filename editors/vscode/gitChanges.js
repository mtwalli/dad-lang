// What changed in a file since git last saw it, for the right-to-left editor's
// gutter (VS Code's text editor shows its own). The original is the file as
// git's index has it (what VS Code's text editor compares with too), read with
// «git show :./file»; the editor's current text is compared with it line by
// line. A file that git doesn't know (untracked, or not in a repository) has no
// changes to show.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");
const path = require("path");

const MAX_LINES = 20000; // past this, changes are found by the common start and end only

/**
 * The differences between two lists of lines, as hunks: lines [start, end) of
 * the new text replace lines [oldStart, oldEnd) of the old one. Myers' O(ND)
 * algorithm, after taking off the lines both share at the start and the end.
 */
function diffLines(a, b) {
  let pre = 0;
  while (pre < a.length && pre < b.length && a[pre] === b[pre]) pre++;
  let suf = 0;
  while (suf < a.length - pre && suf < b.length - pre && a[a.length - 1 - suf] === b[b.length - 1 - suf]) suf++;
  const A = a.slice(pre, a.length - suf);
  const B = b.slice(pre, b.length - suf);
  if (!A.length && !B.length) return [];
  if (!A.length || !B.length || A.length + B.length > MAX_LINES) {
    return [{ oldStart: pre, oldEnd: pre + A.length, start: pre, end: pre + B.length }];
  }
  // Myers: the furthest x reached on each diagonal k, for each number of edits d
  const N = A.length, M = B.length, max = N + M, off = max;
  const v = new Int32Array(2 * max + 2);
  const trace = [];
  let found = false;
  for (let d = 0; d <= max && !found; d++) {
    trace.push(v.slice());
    for (let k = -d; k <= d; k += 2) {
      let x = k === -d || (k !== d && v[off + k - 1] < v[off + k + 1]) ? v[off + k + 1] : v[off + k - 1] + 1;
      let y = x - k;
      while (x < N && y < M && A[x] === B[y]) { x++; y++; }
      v[off + k] = x;
      if (x >= N && y >= M) { found = true; break; }
    }
  }
  // walk back, marking which lines of A were removed and which of B added
  const removed = new Uint8Array(N), added = new Uint8Array(M);
  let x = N, y = M;
  for (let d = trace.length - 1; d > 0; d--) {
    const vd = trace[d];
    const k = x - y;
    const prevK = k === -d || (k !== d && vd[off + k - 1] < vd[off + k + 1]) ? k + 1 : k - 1;
    const prevX = vd[off + prevK], prevY = prevX - prevK;
    while (x > prevX && y > prevY) { x--; y--; }
    if (x === prevX) added[--y] = 1; // went down: a line of B
    else removed[--x] = 1; // went right: a line of A
  }
  // group the marks into hunks
  const hunks = [];
  let i = 0, j = 0;
  while (i < N || j < M) {
    if (i < N && j < M && !removed[i] && !added[j]) { i++; j++; continue; }
    const h = { oldStart: pre + i, start: pre + j };
    while (i < N && removed[i]) i++;
    while (j < M && added[j]) j++;
    // a change can be removals and additions in either order before the next shared line
    while ((i < N && removed[i]) || (j < M && added[j])) {
      while (i < N && removed[i]) i++;
      while (j < M && added[j]) j++;
    }
    h.oldEnd = pre + i;
    h.end = pre + j;
    hunks.push(h);
  }
  return hunks;
}

const splitLines = (text) => text.replace(/\r\n?/g, "\n").split("\n");

/** Hunks with their kind and the old lines, as the editor shows them. */
function changes(original, text) {
  const a = splitLines(original);
  const b = splitLines(text);
  return diffLines(a, b).map((h) => ({
    kind: h.oldStart === h.oldEnd ? "added" : h.start === h.end ? "deleted" : "modified",
    start: h.start,
    end: h.end,
    old: a.slice(h.oldStart, h.oldEnd),
  }));
}

class GitChanges {
  constructor() {
    this.originals = new Map(); // file path → Promise<string | null>
    this.emitter = new vscode.EventEmitter();
    /** Fires when git's view of the files may have changed: a commit, staging, a checkout. */
    this.onDidChange = this.emitter.event;
    this.gitPath = "git";
  }

  /** Watches git, through VS Code's git extension when it is there, and the .git folder. */
  static register(context) {
    const g = new GitChanges();
    const changed = () => g.refresh();
    if (vscode.workspace.createFileSystemWatcher) {
      const w = vscode.workspace.createFileSystemWatcher("**/.git/{index,HEAD}");
      context.subscriptions.push(w, w.onDidChange(changed), w.onDidCreate(changed), w.onDidDelete(changed));
    }
    context.subscriptions.push(
      vscode.workspace.onDidSaveTextDocument((doc) => g.originals.delete(doc.uri.fsPath)),
      g.emitter
    );
    const ext = vscode.extensions && vscode.extensions.getExtension && vscode.extensions.getExtension("vscode.git");
    if (ext) {
      Promise.resolve(ext.isActive ? ext.exports : ext.activate())
        .then((exports) => {
          const api = exports && exports.getAPI && exports.getAPI(1);
          if (!api) return;
          if (api.git && api.git.path) g.gitPath = api.git.path;
          const watch = (repo) => context.subscriptions.push(repo.state.onDidChange(changed));
          api.repositories.forEach(watch);
          context.subscriptions.push(api.onDidOpenRepository(watch));
        })
        .catch(() => {});
    }
    return g;
  }

  refresh() {
    this.originals.clear();
    this.emitter.fire();
  }

  /** The file as git's index has it, or null when git doesn't know it. */
  original(file) {
    if (!this.originals.has(file)) {
      this.originals.set(file, new Promise((resolve) => {
        cp.execFile(this.gitPath, ["show", ":./" + path.basename(file)], {
          cwd: path.dirname(file), encoding: "utf8", maxBuffer: 64 * 1024 * 1024, windowsHide: true,
        }, (err, stdout) => resolve(err ? null : stdout));
      }));
    }
    return this.originals.get(file);
  }

  /** The changes in a document since git last saw it: [] when there are none or git doesn't know the file. */
  async changes(document) {
    if (document.uri.scheme !== "file") return [];
    const original = await this.original(document.uri.fsPath);
    return original === null ? [] : changes(original, document.getText());
  }
}

/**
 * Undoes one change: puts the old lines back in place of the new ones.
 * Lines are VS Code's lines of the document, as the hunk counted them.
 */
function revertEdit(document, hunk) {
  const n = document.lineCount;
  const eol = document.eol === 2 ? "\r\n" : "\n"; // EndOfLine.CRLF is 2
  const edit = new vscode.WorkspaceEdit();
  const { start, end, old } = hunk;
  if (end < n) {
    edit.replace(document.uri, new vscode.Range(start, 0, end, 0), old.map((l) => l + eol).join(""));
  } else if (start > 0) {
    // the change reaches the last line: replace from the end of the line before it
    const from = document.lineAt(start - 1).range.end;
    const to = document.lineAt(n - 1).range.end;
    edit.replace(document.uri, new vscode.Range(from, to), old.map((l) => eol + l).join(""));
  } else {
    edit.replace(document.uri, new vscode.Range(0, 0, n - 1, document.lineAt(n - 1).text.length), old.join(eol));
  }
  return edit;
}

module.exports = { GitChanges, diffLines, changes, revertEdit };
