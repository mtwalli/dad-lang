---
template: home.html
title: Home
hide:
  - navigation
  - toc
---

## A whole app in one file

<div class="dad-showcase" markdown>

<div class="dad-showcase__code" markdown>

```dad
جدول مهمة {
  عنوان: نص
  منجزة = خطأ
}

خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)

مكون قائمة_المهام() {
  حالة جديدة = ""
  عمود {
    حقل(جديدة، تلميح: "مهمة جديدة…")
    زر("أضف") { أضف(جديدة) }
  }
}
```

</div>

<dl class="dad-legend">
  <dt lang="ar">جدول</dt>
  <dd>A database table. Ḍād creates the SQLite schema and a typed API for it.</dd>
  <dt lang="ar">خادم دالة</dt>
  <dd>A function that runs on the server. The interface calls it like any local function.</dd>
  <dt lang="ar">مكون</dt>
  <dd>A reactive interface component, right-to-left by default.</dd>
</dl>

</div>

## Principles, one keyword each

<dl class="dad-principles">
  <div>
    <dt lang="ar">ثابت</dt>
    <dd><strong>Immutable by default.</strong> Values never change unless you declare them with <code>متغير</code>.</dd>
  </div>
  <div>
    <dt lang="ar">؟</dt>
    <dd><strong>No surprise nulls.</strong> A value that may be missing says so in its type, as in <code>نص؟</code>.</dd>
  </div>
  <div>
    <dt lang="ar">طابق</dt>
    <dd><strong>Every case handled.</strong> The compiler rejects a match that forgets a case.</dd>
  </div>
  <div>
    <dt lang="ar">خادم</dt>
    <dd><strong>No fetch, no async.</strong> Server calls look local; the compiler adds the waiting.</dd>
  </div>
  <div>
    <dt lang="ar">،؛؟</dt>
    <dd><strong>Arabic first.</strong> Arabic punctuation works everywhere, diacritics never break a name, and numbers stay plain 0-9.</dd>
  </div>
  <div>
    <dt>39</dt>
    <dd><strong>A small core.</strong> 39 keywords in total. Interface elements are ordinary functions.</dd>
  </div>
</dl>

## Where it stands

<ol class="dad-timeline">
  <li class="is-done"><span>Specification</span><small>Keywords, grammar and the Arabic-aware lexer</small></li>
  <li class="is-next"><span>Core language</span><small>Parser and <code>dad run</code> done; type checker next</small></li>
  <li><span>Interface</span><small>Components and reactive state</small></li>
  <li class="is-next"><span>Server and data</span><small>Tables, server functions, routes and <a href="server/"><code>dad dev</code></a> work, with SQLite; the interface next</small></li>
  <li class="is-next"><span>Tooling</span><small><a href="https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang">VS Code extension</a> and <a href="playground/">web playground</a> done; language server next</small></li>
  <li><span>Installers</span><small>macOS, Windows and Linux</small></li>
</ol>

[Read the full roadmap](ROADMAP.md){ .md-button } [Read the specification](SPEC.md){ .md-button }
