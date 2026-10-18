# ضاد · Ḍād

[![VS Code extension](https://img.shields.io/visual-studio-marketplace/v/mtwalli.dad-lang?label=VS%20Code%20extension)](https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang)

**لغة برمجة عربية حديثة لبناء تطبيقات الويب كاملة — الواجهة والخادم والبيانات — بلغة واحدة.**

A modern Arabic-keyword programming language for building complete web apps (UI, server and
data) in one language. It's concise and precise in the spirit of Kotlin and Swift, and the
compiler is written in Go.

```
جدول مهمة { عنوان: نص ؛ منجزة = خطأ }

خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)

مكون عداد() {
  حالة ع = 0
  زر("زد {ع}") { ع += 1 }
}
```

## Status
Phase 1: the parser and interpreter are done, so `dad run file.dad` runs programs. Phase 3 has started: tables (`جدول`, stored in SQLite), server functions (`خادم`) and routes (`مسار`) work, and `dad dev file.dad` serves them with live reload. Phase 2 has started too: components (`مكون`) with state (`حالة`) and elements such as `زر` and `حقل` render as live, right-to-left pages in `dad dev`, so `examples/05-todo.dad` works end to end. Programs are checked before they run: undefined names (with suggestions), wrong arguments and types, missing fields and `طابق` cases are reported in Arabic by `dad check`, VS Code and the playground. `dad lsp` gives any editor go to definition, references, rename and completion. `dad build file.dad` compiles a program to JavaScript: a page becomes a static site that needs no server, and the tests check that it prints exactly what the interpreter prints. See [docs/ROADMAP.md](docs/ROADMAP.md).

- [docs/SPEC.md](docs/SPEC.md): language specification (draft)
- [docs/grammar.ebnf](docs/grammar.ebnf): grammar
- [examples/](examples): sample programs

## Install

macOS and Linux:
```sh
curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
```

Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.ps1 | iex
```

With Go: `go install github.com/mtwalli/dad-lang/cmd/dad@latest`

Then run a program: `dad run examples/01-hello.dad` (or `ضاد شغل …`). No install needed for the
[browser playground](https://mtwalli.github.io/dad-lang/playground/).
Full guide: [mtwalli.github.io/dad-lang/install](https://mtwalli.github.io/dad-lang/install/).

For VS Code, install [Ḍād from the Marketplace](https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang)
(search "Ḍād" in the Extensions view). See [editors/vscode](editors/vscode) and the [editor guide](https://mtwalli.github.io/dad-lang/editor/).

## Build & test
Requires Go 1.24+.

```sh
go test ./...
go run ./cmd/dad check examples/01-hello.dad     # أو: go run ./cmd/dad فحص examples/01-hello.dad
go build -o dad ./cmd/dad
```

## Releasing
Push a version tag; `.github/workflows/release.yml` builds binaries for macOS, Linux and
Windows (amd64 and arm64) and the VS Code extension (`dad-lang.vsix`), publishes them with `checksums.txt` as a GitHub release, and the
install scripts pick up the newest release automatically.

```sh
git tag v0.2.0 && git push origin v0.2.0
```

## Documentation site
The docs are built with [MkDocs](https://www.mkdocs.org/) + [Material](https://squidfunk.github.io/mkdocs-material/)
and published to GitHub Pages by `.github/workflows/docs.yml` on every push to `main`.

```sh
pip install -r requirements-docs.txt
mkdocs serve        # preview at http://127.0.0.1:8000
```

## Layout
```
cmd/dad/            CLI (ضاد / dad)
cmd/wasm/           the toolchain compiled to WebAssembly for the docs playground
internal/token/     token kinds and keywords
internal/lexer/     Arabic-aware lexer + tests
internal/ast/       syntax tree
internal/parser/    parser with Arabic error messages
internal/interp/    interpreter behind `dad run` and `dad dev`, tables and storage
internal/codegen/js/ compiler to JavaScript and its runtime, behind `dad build`
internal/sqlitestore/ SQLite storage and migrations for tables
internal/check/     the checker: names, arguments, types, exhaustive طابق
internal/lsp/       the language server behind `dad lsp`
internal/format/    the formatter behind `dad fmt`
internal/web/       the browser side of pages: page shell, stylesheet, fonts, script
tools/font/         builds Ḍād Mono, the coding font (python3 tools/font/build.py)
fonts/, fonts-src/  the built font and its sources
editors/vscode/     VS Code extension (tests: node test/grammar.test.js; DAD=… node test/extension.test.js)
docs/               spec, grammar, roadmap (also the docs site source)
tools/pygments-dad/ syntax highlighting for .dad code on the docs site
examples/           .dad sample programs
install.sh, install.ps1  installers that download the latest GitHub release
```

## License
[MIT](LICENSE)
