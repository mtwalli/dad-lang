---
hide:
  - navigation
  - toc
---

<div class="dad-hero" markdown>
# ضاد · Ḍād { .dad-title }
<p class="dad-tagline">لغة برمجة عربية حديثة لبناء تطبيقات الويب كاملة — بلغة واحدة</p>
<p class="dad-sub">A modern Arabic programming language for complete web apps: UI, server and data in one language.</p>
</div>

!!! warning "Early design stage"
    Ḍād is in **Phase 0**: the specification is a draft and only the lexer exists. Everything
    here can still change. See the [roadmap](ROADMAP.md).

## A whole app in one file

```dad
جدول مهمة {
  عنوان: نص
  منجزة = خطأ
}

خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)

مكون عداد() {
  حالة ع = ٠
  زر("زد {ع}") { ع += ١ }
}
```

`جدول` defines a database table, `خادم دالة` a function that runs on the server (the UI can call
it directly), and `مكون` a reactive UI component.

## Design principles

- **Less code, no ceremony.** Types are inferred, there are no semicolons, and functions can be
  one-line expressions. Lambdas go after the call, and strings fill in values directly:
  `"مرحبا {اسم}"`.
- **Precise and safe.** Values are immutable by default, optional values are explicit
  (`نص؟`), and `طابق` must handle every case.
- **One language for the whole stack.** The compiler splits client, server and database code
  for you, with no `async`/`await` noise.
- **Arabic first.** Arabic keywords, digits `٠١٢٣` and punctuation `، ؛ ؟`, with
  diacritic-insensitive names and Arabic error messages.
- **Small core.** 39 keywords. UI elements and HTTP verbs are library functions.

## Explore

- [Language specification](SPEC.md): the full design, from the lexical rules to the full-stack layer
- [Grammar](grammar.md): the formal grammar in EBNF
- [Examples](examples.md): five sample programs
- [Roadmap](ROADMAP.md): architecture and phases

## Try the toolchain

The compiler is written in Go. So far it can tokenize Ḍād source:

```sh
go run ./cmd/dad tokens examples/01-hello.dad    # or: go run ./cmd/dad رموز …
```
