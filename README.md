# ضاد · Ḍād

**لغة برمجة عربية حديثة لبناء تطبيقات الويب كاملة — الواجهة والخادم والبيانات — بلغة واحدة.**

A modern Arabic-keyword programming language for building complete web apps (UI, server and
data) in one language. It's concise and precise in the spirit of Kotlin and Swift, and the
compiler is written in Go.

```
جدول مهمة { عنوان: نص ؛ منجزة = خطأ }

خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)

مكون عداد() {
  حالة ع = ٠
  زر("زد {ع}") { ع += ١ }
}
```

## Status
Phase 0: the specification and lexer are done. See [docs/ROADMAP.md](docs/ROADMAP.md).

- [docs/SPEC.md](docs/SPEC.md): language specification (draft)
- [docs/grammar.ebnf](docs/grammar.ebnf): grammar
- [examples/](examples): sample programs

## Build & test
Requires Go 1.22+.

```sh
go test ./...
go run ./cmd/dad tokens examples/01-hello.dad     # أو: go run ./cmd/dad رموز examples/01-hello.dad
go build -o dad ./cmd/dad
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
internal/token/     token kinds and keywords
internal/lexer/     Arabic-aware lexer + tests
docs/               spec, grammar, roadmap (also the docs site source)
tools/pygments-dad/ syntax highlighting for .dad code on the docs site
examples/           .dad sample programs
```
