---
title: Playground
hide:
  - toc
---

# Playground

Write Ḍād and run it. Everything runs in your browser with the real Ḍād interpreter,
compiled to WebAssembly. Press **Run**, or <kbd>Ctrl</kbd>+<kbd>Enter</kbd>
(<kbd>⌘</kbd>+<kbd>Enter</kbd> on a Mac).

<div class="dad-play" id="dad-play">
  <div class="dad-play__bar">
    <label for="dad-sample">Start from</label>
    <select id="dad-sample">
      <option value="hello">Hello</option>
      <option value="basics">Loops and lambdas</option>
      <option value="shapes">Enums and matching</option>
      <option value="safety">Optionals and errors</option>
      <option value="mistakes">A file with mistakes</option>
    </select>
    <button type="button" class="md-button md-button--primary dad-play__run" id="dad-run" disabled>Run</button>
    <span class="dad-play__status" id="dad-status" role="status">Loading Ḍād…</span>
  </div>
  <div class="dad-play__panes">
    <div class="dad-play__code">
      <pre class="dad-play__hl" dir="rtl" lang="ar" aria-hidden="true"><code id="dad-hl"></code></pre>
      <textarea id="dad-src" class="dad-play__editor" dir="rtl" lang="ar" spellcheck="false"
                autocapitalize="off" autocomplete="off" aria-label="Ḍād source code"></textarea>
    </div>
    <div class="dad-play__output">
      <ul class="dad-play__diags" id="dad-diags" aria-label="Errors and warnings"></ul>
      <div class="dad-play__tabs" role="tablist">
        <button type="button" role="tab" id="dad-tab-out" aria-selected="true" aria-controls="dad-out">Output</button>
        <button type="button" role="tab" id="dad-tab-tokens" aria-selected="false" aria-controls="dad-tokens">Tokens</button>
      </div>
      <pre class="dad-play__out" id="dad-out" role="tabpanel" dir="rtl" aria-live="polite"></pre>
      <div class="dad-play__tokens" id="dad-tokens" role="tabpanel" dir="rtl" hidden></div>
    </div>
  </div>
  <p class="dad-play__legend">
    The Tokens tab shows how Ḍād reads your code:
    <span class="tok tok--keyword">keyword</span>
    <span class="tok tok--name">name</span>
    <span class="tok tok--number">number</span>
    <span class="tok tok--string">text</span>
    <span class="tok tok--symbol">symbol</span>.
    Hover a token to see its kind and position.
  </p>
</div>
