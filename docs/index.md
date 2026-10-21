---
icon: lucide/house
template: home.html
title: الرئيسية
hide:
  - navigation
  - toc
---

## تطبيق كامل في ملف واحد

<div class="dad-showcase" markdown>

<div class="dad-showcase__code" markdown>

```dad
جدول مهمة {
    العنوان: نص
    منجزة = خطأ
}

خادم دالة أضف(العنوان: نص) = مهمة.أنشئ(العنوان: العنوان)

مكون الرئيسية {
    حالة الجديدة = ""
    عمود {
        حقل(الجديدة، تلميح: "مهمة جديدة…")
        زر("أضف") { أضف(الجديدة) }
    }
}
```

</div>

<dl class="dad-legend">
  <dt><span class="dad-ico dad-ico--database" aria-hidden="true"></span>جدول</dt>
  <dd>جدول في قاعدة البيانات. ينشئ ضاد بنيته في SQLite، وطريقة مضبوطة الأنواع للتعامل معه.</dd>
  <dt><span class="dad-ico dad-ico--server" aria-hidden="true"></span>خادم دالة</dt>
  <dd>دالة تعمل على الخادم، وتستدعيها الواجهة كأي دالة أخرى.</dd>
  <dt><span class="dad-ico dad-ico--layout-panel-top" aria-hidden="true"></span>مكون</dt>
  <dd>قطعة من الواجهة تتحدّث وحدها حين تتغير حالتها، ومن اليمين إلى اليسار من الأساس.</dd>
</dl>

</div>

## مبادئ، لكل منها كلمة

<dl class="dad-principles">
  <div>
    <dt>ثابت</dt>
    <dd><strong>الثبات هو الأصل.</strong> لا تتغير قيمة إلا إذا عرّفتها بـ<code>متغير</code>.</dd>
  </div>
  <div>
    <dt>؟</dt>
    <dd><strong>لا مفاجآت مع «عدم».</strong> القيمة التي قد تكون مفقودة يقول نوعها ذلك، مثل <code>نص؟</code>.</dd>
  </div>
  <div>
    <dt>طابق</dt>
    <dd><strong>لكل حالة حسابها.</strong> يرفض المترجم «طابق» تنسى حالة.</dd>
  </div>
  <div>
    <dt>خادم</dt>
    <dd><strong>بلا fetch ولا async.</strong> استدعاء الخادم يبدو كاستدعاء محلي، والمترجم يتولّى الانتظار.</dd>
  </div>
  <div>
    <dt>،؛؟</dt>
    <dd><strong>العربية أولاً.</strong> علامات الترقيم العربية تعمل في كل مكان، والتشكيل لا يغيّر الأسماء، والأعداد بأرقام 0-9.</dd>
  </div>
  <div>
    <dt>39</dt>
    <dd><strong>نواة صغيرة.</strong> 39 كلمة محجوزة فقط، وعناصر الواجهة دوال عادية.</dd>
  </div>
</dl>

## أين وصلنا

<ol class="dad-timeline">
  <li class="is-done"><span>المواصفات</span><small>الكلمات المحجوزة والقواعد النحوية، ومحلل نصوص يفهم العربية</small></li>
  <li class="is-done"><span>نواة اللغة</span><small><code>dad run</code>، والفاحص، و<a href="modules/">البرامج متعددة الملفات</a>، والترجمة إلى JavaScript</small></li>
  <li class="is-next"><span>الواجهة</span><small><a href="ui/">المكونات والحالة والعناصر</a> في <code>dad dev</code> والملعب، و<a href="build/"><code>dad build</code></a> يصنع مواقع ثابتة، و<a href="ui/#pages">صفحات متعددة</a>، و21 عنصراً، و<a href="styling/">سمات</a>. التالي: نماذج تفحص حقولها</small></li>
  <li class="is-next"><span>الخادم والبيانات</span><small>جداول في SQLite، ودوال الخادم، والمسارات، و<a href="server/"><code>dad dev</code></a> و<a href="deploy/"><code>dad serve</code></a>. التالي: المستخدمون وتسجيل الدخول</small></li>
  <li class="is-done"><span>الأدوات</span><small><a href="https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang">إضافة VS Code</a>، و<a href="playground/">الملعب</a>، و<code>dad fmt</code>، و<a href="editor/#the-language-server">خادم اللغة</a></small></li>
  <li class="is-done"><span>التثبيت</span><small><a href="install/">Homebrew وwinget و.deb و.rpm</a>، و<a href="tutorial/">دليل للمبتدئين</a> بالعربية والإنجليزية</small></li>
</ol>

[خارطة الطريق كاملة](ROADMAP.md){ .md-button } [مواصفات اللغة](SPEC.md){ .md-button }
