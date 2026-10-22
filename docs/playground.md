---
icon: lucide/play
title: الملعب
hide:
  - toc
---

# الملعب

اكتب ضاد وشغّله. كل شيء يعمل في متصفحك، بمفسّر ضاد الحقيقي بعد ترجمته إلى WebAssembly.
اضغط **شغّل**، أو <span dir="ltr"><kbd>Ctrl</kbd>+<kbd>Enter</kbd></span> (<span dir="ltr"><kbd>⌘</kbd>+<kbd>Enter</kbd></span> على Mac).
وزر **نسّق** (<span dir="ltr"><kbd>Shift</kbd>+<kbd>Alt</kbd>+<kbd>F</kbd></span>، و<span dir="ltr"><kbd>Shift</kbd>+<kbd>Option</kbd>+<kbd>F</kbd></span>
على Mac) يرتّب الكود كما يرتّبه [`dad fmt`](editor.md#formatting).

<div class="dad-play" id="dad-play">
  <div class="dad-play__bar">
    <label for="dad-sample">ابدأ من</label>
    <select id="dad-sample">
      <option value="hello">مرحبا</option>
      <option value="basics">الحلقات والدوال المجهولة</option>
      <option value="shapes">التعدادات والمطابقة</option>
      <option value="safety">القيم الاختيارية والأخطاء</option>
      <option value="counter">واجهة: عدّادات</option>
      <option value="pages">واجهة: عدة صفحات</option>
      <option value="library">واجهة: العناصر والسمة</option>
      <option value="signup">واجهة: نموذج يفحص حقوله</option>
      <option value="todo">تطبيق مهام بجدول</option>
      <option value="mistakes">ملف فيه أخطاء</option>
    </select>
    <button type="button" class="md-button md-button--primary dad-play__run" id="dad-run" disabled>شغّل</button>
    <button type="button" class="md-button dad-play__run" id="dad-format" disabled title="نسّق الكود (Shift+Alt+F)">نسّق</button>
    <span class="dad-play__status" id="dad-status" role="status">جارٍ تحميل ضاد…</span>
  </div>
  <div class="dad-play__panes">
    <div class="dad-play__code">
      <pre class="dad-play__hl" dir="rtl" lang="ar" aria-hidden="true"><code id="dad-hl"></code></pre>
      <textarea id="dad-src" class="dad-play__editor" dir="rtl" lang="ar" spellcheck="false"
                autocapitalize="off" autocomplete="off" aria-label="كود ضاد"></textarea>
    </div>
    <div class="dad-play__output">
      <ul class="dad-play__diags" id="dad-diags" aria-label="الأخطاء والتنبيهات"></ul>
      <div class="dad-play__tabs" role="tablist">
        <button type="button" role="tab" id="dad-tab-out" aria-selected="true" aria-controls="dad-out">الناتج</button>
        <button type="button" role="tab" id="dad-tab-ui" aria-selected="false" aria-controls="dad-ui" hidden>الواجهة</button>
        <button type="button" role="tab" id="dad-tab-tokens" aria-selected="false" aria-controls="dad-tokens">الرموز</button>
      </div>
      <pre class="dad-play__out" id="dad-out" role="tabpanel" dir="rtl" aria-live="polite"></pre>
      <div class="dad-play__ui" id="dad-ui" role="tabpanel" hidden><iframe title="الواجهة" id="dad-ui-frame"></iframe></div>
      <div class="dad-play__tokens" id="dad-tokens" role="tabpanel" dir="rtl" hidden></div>
    </div>
  </div>
  <p class="dad-play__legend">
    يعرض تبويب «الرموز» كيف يقرأ ضاد الكود:
    <span class="tok tok--keyword">كلمة محجوزة</span>
    <span class="tok tok--name">اسم</span>
    <span class="tok tok--number">عدد</span>
    <span class="tok tok--string">نص</span>
    <span class="tok tok--symbol">رمز</span>.
    مرّر المؤشر فوق رمز لترى نوعه وموضعه.
  </p>
</div>
