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
Version 0.2 is the first public release. You can write a whole web app in Ḍād, with
its interface (`مكون`), server functions (`خادم`), tables in SQLite (`جدول`) and routes
(`مسار`), in one or several files. `dad dev` runs it with live reload, `dad serve` puts it
online, and `dad build` turns it into a static site. Programs are checked before they run,
with messages in Arabic, in the terminal, in VS Code and in any editor through `dad lsp`.
New to programming? Start with the [tutorial in Arabic](https://mtwalli.github.io/dad-lang/tutorial/).
What comes next is in [docs/ROADMAP.md](docs/ROADMAP.md).

- [docs/SPEC.md](docs/SPEC.md): language specification (draft)
- [docs/grammar.ebnf](docs/grammar.ebnf): grammar
- [examples/](examples): sample programs

## Install

Homebrew (macOS, Linux): `brew install mtwalli/dad/dad`. winget (Windows): `winget install mtwalli.dad`.
Debian/Ubuntu and Fedora/RHEL packages are on the [releases page](https://github.com/mtwalli/dad-lang/releases/latest).

Or with the install script, on macOS and Linux:
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
Requires Go 1.26+, and Node 18+ for the tests of compiled programs (skipped without it).

```sh
go test ./...
go run ./cmd/dad check examples/01-hello.dad     # أو: go run ./cmd/dad فحص examples/01-hello.dad
go build -o dad ./cmd/dad
```

## Releasing
Add a section to [CHANGELOG.md](CHANGELOG.md) and push a version tag.
`.github/workflows/release.yml` builds the binaries, the `.deb` and `.rpm` packages, the
VS Code extension, the Homebrew formula and the winget manifests, and publishes the
release. [tools/release/README.md](tools/release/README.md) has the one-time setup for
Homebrew, winget and the Marketplace.

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
internal/module/    loads a program of several files (استورد، خاص)
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
