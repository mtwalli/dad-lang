---
title: Playground
hide:
  - toc
---

# Playground

Type Ḍād code and see how the real Ḍād lexer reads it: every token, plus errors and
warnings in Arabic. Everything runs in your browser. Running programs comes in Phase 1.

<div class="dad-play" id="dad-play">
  <div class="dad-play__bar">
    <label for="dad-sample">Start from</label>
    <select id="dad-sample">
      <option value="hello">Hello</option>
      <option value="shapes">Enums and matching</option>
      <option value="todo">To-do app</option>
      <option value="mistakes">A file with mistakes</option>
    </select>
    <span class="dad-play__status" id="dad-status" role="status">Loading the Ḍād lexer…</span>
  </div>
  <div class="dad-play__panes">
    <textarea id="dad-src" class="dad-play__editor" dir="rtl" lang="ar" spellcheck="false"
              aria-label="Ḍād source code"></textarea>
    <div class="dad-play__output">
      <ul class="dad-play__diags" id="dad-diags" aria-label="Errors and warnings"></ul>
      <div class="dad-play__tokens" id="dad-tokens" dir="rtl" aria-label="Tokens"></div>
    </div>
  </div>
  <p class="dad-play__legend">
    <span class="tok tok--keyword">keyword</span>
    <span class="tok tok--name">name</span>
    <span class="tok tok--number">number</span>
    <span class="tok tok--string">text</span>
    <span class="tok tok--symbol">symbol</span>
    Hover a token to see its kind and position. Names are shown the way Ḍād stores them,
    without diacritics or tatweel.
  </p>
</div>
