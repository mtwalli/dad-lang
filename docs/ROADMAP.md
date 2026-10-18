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
| **2. UI** | `مكون`, `حالة`, element library, reactive runtime, RTL defaults | Counter + list examples render in a browser ✅ (`dad dev`, the playground, and static sites from `dad build`) |
| **3. Server & data** | `خادم` functions (auto RPC), `جدول` → SQLite, `مسار`, `dad dev` with live reload | `05-todo.dad` works end-to-end (server half ✅ with SQLite; the UI half next) |
| **4. Tooling** | Language server (Go), VS Code extension, formatter `dad fmt`, WASM web playground | Editing in VS Code with completion, errors, RTL display verified (VS Code extension, `dad fmt`, playground and the `dad lsp` language server ✅) |
| **5. Distribution** | GoReleaser + GitHub Actions; Homebrew, winget, `.deb`/`.rpm`, `install.sh`; Arabic tutorial site | One-line install on macOS, Windows, Linux |

## Phase 1 breakdown (in progress)
1. ✅ AST node definitions (`internal/ast`)
2. ✅ Parser for declarations/statements/expressions + Arabic error reporting with line/column
   (`internal/parser`)
3. ✅ `dad run file.dad` — a tree-walking interpreter (`internal/interp`) so programs run
   today, in the terminal and in the [playground](playground.md)
4. ✅ Golden tests: each example has expected output in `examples/output/`
5. ✅ Name resolution and a first type checker (`internal/check`): undefined names with
   suggestions, arguments, declared types, members, exhaustive `طابق`; programs with errors
   don't run. Next: smart casts for optional values, generic types, unused-variable warnings
6. ✅ JS code generator (`internal/codegen/js`) and `dad build`: a static site for programs
   with an interface, `app.js` for Node otherwise. Every example and test program prints the
   same output and errors as the interpreter, which stays the reference implementation for
   `dad run`, `dad dev` and the playground

## Phase 2 breakdown (in progress)
1. ✅ Parse `مكون` and `حالة`
2. ✅ Components and per-instance state in the interpreter; elements `عمود`، `صف`، `بطاقة`،
   `عنوان`، `فقرة`، `زر`، `حقل`، `مربع_اختيار`، `صورة`، `رابط`، `فاصل`
3. ✅ Pages in `dad dev`: render on the server, events from the browser, updates in place,
   live reload; a right-to-left Kufi stylesheet with light and dark themes
4. ✅ The playground shows the interface (in WebAssembly, tables in memory)
5. ✅ Compiled pages work without a server (`dad build`): components render in the browser,
   tables live in its storage. Next: update only what changed (signals) instead of
   re-rendering the component tree
6. More elements: lists, selects, text areas, dialogs, navigation between pages

## Phase 3 breakdown (in progress)
1. ✅ Parse `جدول`, `خادم` and `مسار`
2. ✅ Tables in the interpreter, behind a `Store` interface: memory for the playground and
   tests, SQLite for `dad run` and `dad dev`
3. ✅ `dad dev`: routes with `{params}`, server functions at `/_dad/call/<name>`, a dev page,
   live reload that keeps the data
4. ✅ SQLite store (`modernc.org/sqlite`, `.dad/<program>.db`): migrations from table changes,
   simple `حيث` lambdas compiled to SQL, earlier JSON data imported
5. ✅ Components call server functions and tables directly (they run on the server for now)

## Costs to know about
- macOS signed/notarized installer: Apple Developer Program, $99/year (can postpone via Homebrew).
- Windows signed installer: code-signing certificate (can postpone via winget / zip).
