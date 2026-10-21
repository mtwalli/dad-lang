---
icon: lucide/map
---

# Roadmap

Ḍād 0.2 is the first public release. A whole web app can be written in it today: the
interface, server functions, tables in SQLite and routes, in one file or several. It runs
with `dad dev`, goes online with `dad serve`, and becomes a static site with `dad build`.
This page shows how it is built, what is done, and what comes next.

## Architecture

```text
*.dad ─► lexer ─► parser ─► syntax tree ─► checker ─┬─► interpreter ─► dad run · dad dev · dad serve · playground (WebAssembly)
                                                     └─► JavaScript compiler ─► dad build (static site, or app.js for Node)
                       tables ─► SQLite (migrations from table changes)
```

- **One program, written in Go.** `dad` is a single binary for each system, with no
  dependencies: SQLite is built in (`modernc.org/sqlite`, pure Go), and so are the page
  script, the stylesheet and the Kufi font.
- **A hand-written parser,** recursive descent with Pratt parsing for expressions, so that
  error messages can be precise and in Arabic.
- **A checker** that reports only what is certain: undefined names with suggestions,
  arguments, declared types, members, and `طابق` that misses cases. A program with mistakes
  doesn't run.
- **Two ways to run, one behaviour.** The interpreter is the reference. The JavaScript
  compiler is tested against it: every example prints the same output and the same errors,
  and pages render the same HTML after the same clicks.
- **Pages render on the server** in `dad dev` and `dad serve` (like Phoenix LiveView), and in
  the browser after `dad build`.

## Phases

| Phase | Goal | Status |
|---|---|---|
| **0. Specification** | [The specification](SPEC.md), [the grammar](grammar.md), sample programs, the lexer | ✅ Done |
| **1. Core language** | Parser, checker, interpreter, programs in several files, JavaScript compiler, Arabic errors | ✅ Done. Next: smart casts for optional values, generic types, unused-variable warnings |
| **2. Interface** | `مكون`, `حالة`, the elements, right to left by default | ✅ In `dad dev`, the playground and `dad build`. Next: more elements, several pages, updating only what changed |
| **3. Server and data** | `خادم` functions, `جدول` in SQLite, `مسار`, `dad dev`, `dad serve` | ✅ Done. Next: users and sign-in, permissions, requests in parallel |
| **4. Tooling** | VS Code extension, `dad fmt`, the playground, the `dad lsp` language server | ✅ Done |
| **5. Distribution** | Homebrew, winget, `.deb` and `.rpm`, install scripts, the tutorial in English and Arabic | ✅ Done in 0.2 |

## What comes next

These are the next steps, roughly in order:

1. **Users and sign-in**, and permissions on server functions and routes, so that programs
   with private data can go online.
2. **Requests in parallel** in `dad serve`, for busier sites.
3. **More elements:** lists, selects, text areas and dialogs. Also **several pages** in one
   program, each with its own address.
4. **Updating only what changed** on a page (signals), instead of laying out the whole
   component again.
5. **A sharper checker:** smart casts after `إذا ن != عدم`, generic types, warnings for unused
   names.
6. **Signed installers** for macOS and Windows.

## Done so far

### Phase 1: the core language

1. ✅ The syntax tree (`internal/ast`) and the parser, with Arabic errors that give the line
   and column (`internal/parser`)
2. ✅ `dad run`: the interpreter (`internal/interp`), in the terminal and in the
   [playground](playground.md)
3. ✅ Golden tests: each example has its expected output in `examples/output/`
4. ✅ [Programs in several files](modules.md) (`internal/module`): `استورد` and `خاص`,
   checked across files, and followed by every command and by the editor
5. ✅ The checker (`internal/check`). Programs with mistakes don't run
6. ✅ The JavaScript compiler (`internal/codegen/js`) and [`dad build`](build.md)

### Phase 2: the interface

1. ✅ Components with state, and the elements `عمود`، `صف`، `بطاقة`، `عنوان`، `فقرة`، `زر`،
   `حقل`، `مربع_اختيار`، `صورة`، `رابط`، `فاصل`
2. ✅ Pages in `dad dev`: rendered on the server, updated in place, reloaded on save. They use
   a right-to-left Kufi stylesheet with light and dark themes
3. ✅ The playground shows the interface (in WebAssembly, tables in memory)
4. ✅ Static sites from `dad build`: components render in the browser, and tables live in its
   storage

### Phase 3: server and data

1. ✅ Tables behind a `Store` interface: in memory for the playground and the tests, and in
   SQLite for `dad run`, `dad dev` and `dad serve`
2. ✅ Migrations when a table changes, and simple `حيث` conditions run as SQL
3. ✅ `dad dev`: routes with `{params}`, server functions at `/_dad/call/<name>`, a developer
   page, and live reload that keeps the data
4. ✅ [`dad serve`](deploy.md): the production server

### Phases 4 and 5: tooling and distribution

1. ✅ The VS Code extension, with a right-to-left editor, and the Ḍād Mono font
2. ✅ `dad fmt`, in the Kotlin style
3. ✅ `dad lsp`, for definitions, references, rename, hover, completion and the outline in
   any editor
4. ✅ Release builds for macOS, Windows and Linux on Intel and ARM: Homebrew, winget, `.deb`,
   `.rpm` and install scripts
5. ✅ The [tutorial](tutorial/index.md) in English and Arabic, and this site in both
   languages

## Costs to know about

- A signed and notarized macOS installer needs the Apple Developer Program, at $99 a year.
  Until then, Homebrew and the install script cover macOS.
- A signed Windows installer needs a code-signing certificate. Until then, winget and the
  zip archive cover Windows.
