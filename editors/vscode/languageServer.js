// A small client for Ḍād's language server («dad lsp»), with no npm dependencies.
// It starts dad, keeps the server's copy of each open .dad file up to date, and
// answers VS Code's requests for definitions, references, rename, hover,
// member completion and the outline. Errors still come from «dad check», and
// formatting from «dad fmt»; the server is only asked for what needs it.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");

const TIMEOUT_MS = 4000;

class LanguageServer {
  constructor(exe, log) {
    this.exe = exe;
    this.log = log || (() => {});
    this.nextId = 1;
    this.pending = new Map(); // id → { resolve, timer }
    this.buffer = Buffer.alloc(0);
    this.ready = null;
    this.dead = false;
  }

  /** Starts dad lsp; resolves to true when it answered «initialize». */
  start() {
    if (this.ready) return this.ready;
    this.ready = new Promise((resolve) => {
      try {
        this.child = cp.spawn(this.exe, ["lsp"], { windowsHide: true });
      } catch {
        this.dead = true;
        resolve(false);
        return;
      }
      this.child.on("error", () => { this.dead = true; resolve(false); });
      this.child.on("exit", () => {
        this.dead = true;
        for (const p of this.pending.values()) { clearTimeout(p.timer); p.resolve(null); }
        this.pending.clear();
      });
      this.child.stderr.on("data", (d) => this.log(String(d)));
      this.child.stdout.on("data", (chunk) => this.receive(chunk));
      // VS Code shows keywords and built-ins from the extension itself.
      this.request("initialize", { processId: process.pid, rootUri: null, capabilities: {}, initializationOptions: { keywords: false } })
        .then((res) => {
          if (!res || !res.capabilities) { this.dead = true; resolve(false); return; } // an older dad without «lsp»
          this.notify("initialized", {});
          resolve(true);
        });
    });
    return this.ready;
  }

  stop() {
    if (this.child && !this.dead) {
      this.request("shutdown", null).then(() => this.notify("exit", null));
      setTimeout(() => { try { this.child.kill(); } catch {} }, 500);
    }
  }

  send(msg) {
    if (this.dead || !this.child) return;
    const body = Buffer.from(JSON.stringify(Object.assign({ jsonrpc: "2.0" }, msg)), "utf8");
    this.child.stdin.write(`Content-Length: ${body.length}\r\n\r\n`);
    this.child.stdin.write(body);
  }

  request(method, params) {
    if (this.dead) return Promise.resolve(null);
    const id = this.nextId++;
    return new Promise((resolve) => {
      const timer = setTimeout(() => { this.pending.delete(id); resolve(null); }, TIMEOUT_MS);
      this.pending.set(id, { resolve, timer });
      this.send({ id, method, params });
    });
  }

  notify(method, params) {
    this.send({ method, params });
  }

  receive(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);
    for (;;) {
      const headerEnd = this.buffer.indexOf("\r\n\r\n");
      if (headerEnd < 0) return;
      const header = this.buffer.slice(0, headerEnd).toString("ascii");
      const m = /Content-Length:\s*(\d+)/i.exec(header);
      if (!m) { this.buffer = this.buffer.slice(headerEnd + 4); continue; }
      const length = Number(m[1]);
      if (this.buffer.length < headerEnd + 4 + length) return;
      const body = this.buffer.slice(headerEnd + 4, headerEnd + 4 + length).toString("utf8");
      this.buffer = this.buffer.slice(headerEnd + 4 + length);
      let msg;
      try { msg = JSON.parse(body); } catch { continue; }
      if (msg.id !== undefined && this.pending.has(msg.id)) {
        const p = this.pending.get(msg.id);
        this.pending.delete(msg.id);
        clearTimeout(p.timer);
        p.resolve(msg.error ? { error: msg.error } : msg.result);
      }
      // Notifications (such as diagnostics) are not used: «dad check» provides them.
    }
  }
}

// ---------- converting between the protocol and VS Code ----------

const toRange = (r) => new vscode.Range(r.start.line, r.start.character, r.end.line, r.end.character);
const at = (doc, pos) => ({ textDocument: { uri: doc.uri.toString() }, position: { line: pos.line, character: pos.character } });

const completionKind = {
  2: vscode.CompletionItemKind.Method, 3: vscode.CompletionItemKind.Function, 5: vscode.CompletionItemKind.Field,
  6: vscode.CompletionItemKind.Variable, 7: vscode.CompletionItemKind.Class, 10: vscode.CompletionItemKind.Property,
  13: vscode.CompletionItemKind.Enum, 20: vscode.CompletionItemKind.EnumMember, 21: vscode.CompletionItemKind.Constant,
  22: vscode.CompletionItemKind.Struct,
};

/**
 * Connects the language server to VS Code. dadPath() finds dad. Returns a
 * disposable; the providers answer nothing when dad has no «lsp» command.
 */
function register(context, dadPath) {
  let server = null;
  const synced = new Map(); // uri → version sent

  async function get() {
    if (server && !server.dead) return (await server.ready) ? server : null;
    const exe = dadPath();
    if (!exe) return null;
    server = new LanguageServer(exe);
    synced.clear();
    return (await server.start()) ? server : null;
  }

  // The server needs the current text before answering about it.
  async function ready(doc) {
    const s = await get();
    if (!s) return null;
    const uri = doc.uri.toString();
    if (!synced.has(uri)) {
      s.notify("textDocument/didOpen", { textDocument: { uri, languageId: "dad", version: doc.version, text: doc.getText() } });
    } else if (synced.get(uri) !== doc.version) {
      s.notify("textDocument/didChange", { textDocument: { uri, version: doc.version }, contentChanges: [{ text: doc.getText() }] });
    }
    synced.set(uri, doc.version);
    return s;
  }

  const ask = async (doc, method, params) => {
    const s = await ready(doc);
    if (!s) return null;
    const res = await s.request(method, params);
    return res && res.error ? null : res;
  };

  const selector = { language: "dad" };
  const disposables = [
    vscode.languages.registerDefinitionProvider(selector, {
      async provideDefinition(doc, pos) {
        const loc = await ask(doc, "textDocument/definition", at(doc, pos));
        return loc ? new vscode.Location(doc.uri, toRange(loc.range)) : null;
      },
    }),
    vscode.languages.registerReferenceProvider(selector, {
      async provideReferences(doc, pos) {
        const locs = await ask(doc, "textDocument/references", at(doc, pos));
        return (locs || []).map((l) => new vscode.Location(doc.uri, toRange(l.range)));
      },
    }),
    vscode.languages.registerDocumentHighlightProvider(selector, {
      async provideDocumentHighlights(doc, pos) {
        const hs = await ask(doc, "textDocument/documentHighlight", at(doc, pos));
        return (hs || []).map((h) => new vscode.DocumentHighlight(toRange(h.range), h.kind === 3 ? vscode.DocumentHighlightKind.Write : vscode.DocumentHighlightKind.Read));
      },
    }),
    vscode.languages.registerRenameProvider(selector, {
      async prepareRename(doc, pos) {
        const r = await ask(doc, "textDocument/prepareRename", at(doc, pos));
        if (!r) throw new Error("لا يوجد اسم هنا يمكن تغييره");
        return { range: toRange(r.range), placeholder: r.placeholder };
      },
      async provideRenameEdits(doc, pos, newName) {
        const s = await ready(doc);
        if (!s) return null;
        const res = await s.request("textDocument/rename", Object.assign(at(doc, pos), { newName }));
        if (!res) return null;
        if (res.error) throw new Error(res.error.message);
        const edit = new vscode.WorkspaceEdit();
        for (const e of res.changes[doc.uri.toString()] || []) edit.replace(doc.uri, toRange(e.range), e.newText);
        return edit;
      },
    }),
    vscode.languages.registerHoverProvider(selector, {
      async provideHover(doc, pos) {
        const h = await ask(doc, "textDocument/hover", at(doc, pos));
        if (!h) return null;
        return new vscode.Hover(new vscode.MarkdownString(h.contents.value), h.range ? toRange(h.range) : undefined);
      },
    }),
    vscode.languages.registerCompletionItemProvider(selector, {
      async provideCompletionItems(doc, pos) {
        const items = await ask(doc, "textDocument/completion", at(doc, pos));
        return (items || []).map((it) => {
          const item = new vscode.CompletionItem(it.label, completionKind[it.kind] || vscode.CompletionItemKind.Text);
          item.detail = it.detail;
          item.sortText = "0" + it.label; // the program's own names before keywords
          return item;
        });
      },
    }, ".", "؟"),
    vscode.languages.registerDocumentSymbolProvider(selector, {
      async provideDocumentSymbols(doc) {
        const syms = await ask(doc, "textDocument/documentSymbol", { textDocument: { uri: doc.uri.toString() } });
        const convert = (s) => {
          const ds = new vscode.DocumentSymbol(s.name, s.detail || "", Math.max((s.kind || 13) - 1, 0), toRange(s.range), toRange(s.selectionRange));
          ds.children = (s.children || []).map(convert);
          return ds;
        };
        return (syms || []).map(convert);
      },
    }),
    vscode.workspace.onDidCloseTextDocument((doc) => {
      const uri = doc.uri.toString();
      if (server && synced.has(uri)) server.notify("textDocument/didClose", { textDocument: { uri } });
      synced.delete(uri);
    }),
    { dispose: () => server && server.stop() },
  ];
  context.subscriptions.push(...disposables);
  return {
    restart() {
      if (server) server.stop();
      server = null;
    },
  };
}

/** Whether the cursor follows «x.»: then only members make sense, not keywords. */
function afterDot(doc, pos) {
  const line = doc.lineAt(pos.line).text.slice(0, pos.character);
  return /[\p{L}\p{Mn}_0-9)\]](?:\.|[؟?]\.)[\p{L}\p{Mn}_0-9]*$/u.test(line);
}

module.exports = { register, afterDot, LanguageServer };
