# Ḍād Roadmap

Strategy: build a **thin vertical slice** first (the to-do app in `examples/05-todo.dad`
working end-to-end, crudely), then deepen each layer.

## Architecture

```text
*.dad ─► lexer ─► parser ─► AST ─► resolver/type checker ─► IR ─┬─► client JS (browser, RTL UI runtime)
                                                                 ├─► server JS (embedded engine) / later native
                                                                 └─► SQLite schema + migrations
```

- **Compiler:** Go, single static binary per OS (`GOOS/GOARCH` cross-compilation).
- **Parser:** hand-written recursive descent with Pratt parsing for expressions — no parser
  generator, so error messages can be precise and in Arabic.
- **Types:** bidirectional type inference (Kotlin/Swift-style local inference), smart casts,
  exhaustiveness checking for `طابق`.
- **Client output:** ES2022 modules + a small reactive runtime (signals), embedded with `go:embed`.
- **Server in dev:** generated server JS runs inside the `dad` binary on an embedded pure-Go
  JS engine (goja) — no Node install required. **Production:** `ضاد ابنِ` emits a bundle for
  Node/Bun; a native backend is a later option if server performance demands it.
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no cgo → trivial cross-compiling).

## Phases

| Phase | Goal | Done when |
|---|---|---|
| **0. Specification** | `docs/SPEC.md`, `docs/grammar.ebnf`, sample programs, lexer | Keywords agreed; all examples tokenize ✅ (lexer done) |
| **1. Core language** | Parser, AST, type checker, JS codegen for logic; `dad run`, REPL, Arabic errors | `01`–`04` examples run and print correct output ✅ (`dad run`) |
| **2. UI** | `مكون`, `حالة`, element library, reactive runtime, RTL defaults | Counter + list examples render in a browser |
| **3. Server & data** | `خادم` functions (auto RPC), `جدول` → SQLite, `مسار`, `dad dev` with live reload | `05-todo.dad` works end-to-end (server half ✅ with SQLite; the UI half next) |
| **4. Tooling** | Language server (Go), VS Code extension, formatter `dad fmt`, WASM web playground | Editing in VS Code with completion, errors, RTL display verified (VS Code extension and playground ✅, language server next) |
| **5. Distribution** | GoReleaser + GitHub Actions; Homebrew, winget, `.deb`/`.rpm`, `install.sh`; Arabic tutorial site | One-line install on macOS, Windows, Linux |

## Phase 1 breakdown (in progress)
1. ✅ AST node definitions (`internal/ast`)
2. ✅ Parser for declarations/statements/expressions + Arabic error reporting with line/column
   (`internal/parser`)
3. ✅ `dad run file.dad` — a tree-walking interpreter (`internal/interp`) so programs run
   today, in the terminal and in the [playground](playground.md)
4. ✅ Golden tests: each example has expected output in `examples/output/`
5. Name resolution and type checker (`internal/check`)
6. JS code generator (`internal/codegen/js`); `dad run` switches to it once it passes the
   same golden tests. The interpreter stays as the reference implementation.

## Phase 3 breakdown (in progress)
1. ✅ Parse `جدول`, `خادم` and `مسار`
2. ✅ Tables in the interpreter, behind a `Store` interface: memory for the playground and
   tests, SQLite for `dad run` and `dad dev`
3. ✅ `dad dev`: routes with `{params}`, server functions at `/_dad/call/<name>`, a dev page,
   live reload that keeps the data
4. ✅ SQLite store (`modernc.org/sqlite`, `.dad/<program>.db`): migrations from table changes,
   simple `حيث` lambdas compiled to SQL, earlier JSON data imported
5. Typed client calls from `مكون` code once Phase 2 lands

## Costs to know about
- macOS signed/notarized installer: Apple Developer Program, $99/year (can postpone via Homebrew).
- Windows signed installer: code-signing certificate (can postpone via winget / zip).
