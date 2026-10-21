/* Ḍād playground: runs the Go lexer, parser and interpreter (compiled to WebAssembly) in the browser. */
(function () {
  "use strict";

  // Messages in the page's language (the site has English and Arabic pages).
  var AR = document.documentElement.lang === "ar";
  var L = AR ? {
    loading: "جارٍ تحميل ضاد…",
    ready: "جاهز",
    fixFirst: "أصلح الأخطاء أولاً، ثم نسّق.",
    failed: "تعذّر تحميل ضاد. تحقق من اتصالك وأعد تحميل الصفحة.",
    press: "اضغط «شغّل» لتشغيل البرنامج.",
    uiOut: "ما يطبعه البرنامج يظهر هنا.",
    noOut: "انتهى البرنامج دون أن يطبع شيئاً."
  } : {
    loading: "Loading Ḍād…",
    ready: "Ready",
    fixFirst: "Fix the errors first, then format.",
    failed: "Ḍād couldn't load. Check your connection and reload the page.",
    press: "Press Run to run the program.",
    uiOut: "What the program prints appears here.",
    noOut: "The program finished without printing anything."
  };

  var assets = (function () {
    var s = document.currentScript;
    return s ? new URL(".", s.src).href : "";
  })();

  var SAMPLES = {
    "hello": "// أول برنامج بلغة ضاد\nثابت اسم = \"العالم\"\nاطبع(\"مرحبا يا {اسم}!\")\n\nدالة حي(اسم: نص، تحية: نص = \"أهلاً\"): نص {\n    أرجع \"{تحية} يا {اسم}\"\n}\n\nاطبع(حي(\"سارة\"))\nاطبع(حي(اسم: \"علي\"، تحية: «صباح الخير»))",
    "basics": "// المتغيرات، الشروط، الحلقات، والدوال المختصرة\nدالة مربع(س: صحيح) = س * س\n\nدالة وصف(درجة: صحيح) = إذا درجة >= 90 { \"ممتاز\" } وإلا إذا درجة >= 50 { \"ناجح\" } وإلا { \"راسب\" }\n\nمتغير مجموع = 0\nلكل ع من 1 إلى 10 {\n    مجموع += مربع(ع)\n}\nاطبع(\"مجموع المربعات: {مجموع}\")\n\nلكل ع من 0 حتى 10 بخطوة 2 { اطبع(ع) }\n\nثابت درجات = [95، 72، 40]\nلكل د في درجات {\n    اطبع(\"{د} => {وصف(د)}\")\n}\n\nثابت كبار = درجات.رشح { هو >= 50 }.حول { هو * 2 }\nثابت متوسط = درجات.اختزل(0) { أ، ب => أ + ب } ك عشري / 3.0",
    "shapes": "// أنواع القيم، التعدادات، والمطابقة الشاملة\nنوع نقطة(س: عشري، ص: عشري) {\n    دالة بعد() = جذر(س * س + ص * ص)\n}\n\nتعداد شكل {\n    دائرة(قطر: عشري)\n    مستطيل(عرض: عشري، طول: عشري)\n    مثلث(قاعدة: عشري، ارتفاع: عشري)\n}\n\nدالة مساحة(ش: شكل): عشري = طابق ش {\n    دائرة(ق) => 3.1416 * ق * ق / 4\n    مستطيل(ع، ط) => ع * ط\n    مثلث(ق، ر) => ق * ر / 2\n}\n\nثابت أشكال = [دائرة(2)، مستطيل(3، 4)، مثلث(قاعدة: 6، ارتفاع: 2)]\nلكل ش في أشكال {\n    اطبع(\"المساحة = {مساحة(ش)}\")\n    إذا ش من نوع دائرة { اطبع(\"قطرها {ش.قطر}\") }\n}\n\nثابت ن = نقطة(س: 3، ص: 4)\nاطبع(ن.بعد()) // 5\nاطبع(ن == نقطة(3، 4)) // صح\nثابت م = ن.نسخة(ص: 0)",
    "safety": "// الأمان من القيم المفقودة، الأخطاء، العقود والامتدادات\nنوع مستخدم(اسم: نص، بريد: نص؟ = عدم)\n\nدالة بريد_العرض(م: مستخدم؟): نص {\n    ثابت بريد = م؟.بريد\n    إذا بريد != عدم {\n        أرجع \"{بريد} ({بريد.طول} حرفاً)\" // تحويل ذكي: بريد هنا نص\n    }\n    أرجع م؟.بريد ؟؟ \"لا يوجد بريد\"\n}\n\nدالة اقسم(أ: صحيح، ب: صحيح): عشري {\n    إذا ب == 0 { ارفع مشكلة(\"قسمة على صفر\") }\n    أرجع أ / ب\n}\n\nثابت ناتج = حاول { اقسم(10، 0) } التقط خ {\n    اطبع(خ.رسالة)\n    0\n}\n\nعقد قابل_للعرض {\n    دالة اعرض(): نص\n}\n\nامتداد نص {\n    خاص دالة نظيف() = هذا.قص()\n    دالة مكرر(مرات: صحيح) = (من 1 إلى مرات).حول { هذا }.ضم(\"\")\n}\n\nاطبع(\"هلا \".مكرر(3))\nاطبع(بريد_العرض(مستخدم(\"هدى\"، بريد: \"huda@example.com\")))\nاطبع(بريد_العرض(عدم))",
    "counter": "// واجهة بلا خادم: مكونات بمعاملات، وحالة لكل نسخة، وحقول\nمكون عداد(اسم: نص، بداية: صحيح = 0) {\n    حالة ع = بداية\n    بطاقة {\n        عنوان(\"{اسم}: {ع}\"، مستوى: 2)\n        صف {\n            زر(\"زد\") { ع += 1 }\n            زر(\"أنقص\"، معطل: ع == 0) { ع -= 1 }\n            زر(\"صفّر\") { ع = بداية }\n        }\n    }\n}\n\nمكون الرئيسية {\n    حالة اسم = \"\"\n    حالة خطوة = 1\n    عمود {\n        عنوان(\"عدّادات\")\n        فقرة(\"لكل عداد حالته الخاصة.\")\n        عداد(\"التفاح\")\n        عداد(\"البرتقال\"، بداية: 5)\n        فاصل()\n        صف {\n            حقل(اسم، تلميح: \"ما اسمك؟\")\n            حقل(خطوة)\n        }\n        إذا اسم != \"\" { فقرة(\"أهلاً يا {اسم}، الخطوة {خطوة * 2}\") }\n        مربع_اختيار(خطوة > 1، نص: \"الخطوة أكبر من 1\")\n        رابط(\"عن ضاد\"، \"https://mtwalli.github.io/dad-lang/\")\n    }\n}",
    "pages": "// برنامج من عدة صفحات: لكل صفحة عنوانها، والروابط بينها تفتحها في مكانها\n\nجدول مهمة {\n    العنوان: نص\n    الملاحظات = \"\"\n    منجزة = خطأ\n}\n\nخادم دالة كل_المهام() = مهمة.الكل()\nخادم دالة أضف(العنوان: نص) = مهمة.أنشئ(العنوان: العنوان)\nخادم دالة احفظ(م: مهمة) = م.احفظ()\nخادم دالة احذف(م: مهمة) = م.احذف()\n\nصفحة \"/\" = قائمة_المهام()\nصفحة \"/مهمة/{رقم}\" = صفحة_المهمة(رقم)\n\nصفحة \"/عن\" {\n    الشريط()\n    عنوان(\"عن التطبيق\")\n    فقرة(\"مثال على برنامج بلغة ضاد فيه ثلاث صفحات، يتنقل بينها الزائر بالروابط وبزر الرجوع.\")\n}\n\n// الشريط في أعلى كل صفحة. الرابط إلى الصفحة المعروضة يُميَّز وحده.\nمكون الشريط() {\n    صف {\n        رابط(\"المهام\"، \"/\")\n        رابط(\"عن التطبيق\"، \"/عن\")\n    }\n    فاصل()\n}\n\nمكون قائمة_المهام() {\n    حالة الجديدة = \"\"\n    ثابت المهام = كل_المهام()\n    الشريط()\n    عمود {\n        عنوان(\"المهام ({المهام.عدد { ليس هو.منجزة }} باقية)\")\n        صف {\n            حقل(الجديدة، تلميح: \"مهمة جديدة…\")\n            زر(\"أضف\"، معطل: الجديدة.قص() == \"\") {\n                ثابت م = أضف(الجديدة.قص())\n                انتقل(\"/مهمة/{م.رقم}\")\n            }\n        }\n        لكل م في المهام {\n            ثابت العلامة = إذا م.منجزة { \"✓ \" } وإلا { \"\" }\n            رابط(العلامة + م.العنوان، \"/مهمة/{م.رقم}\")\n        }\n    }\n}\n\nمكون صفحة_المهمة(رقم: صحيح) {\n    ثابت م = مهمة.بالرقم(رقم)\n    الشريط()\n    إذا م == عدم {\n        عنوان(\"لا توجد مهمة رقم {رقم}\")\n        رابط(\"إلى المهام\"، \"/\")\n    } وإلا {\n        تفاصيل(م)\n    }\n}\n\nمكون تفاصيل(م: مهمة) {\n    حالة الملاحظات = م.الملاحظات\n    عمود {\n        عنوان(م.العنوان)\n        مربع_اختيار(م.منجزة، نص: \"منجزة\") { احفظ(م.نسخة(منجزة: هو)) }\n        حقل(الملاحظات، تلميح: \"ملاحظات…\")\n        صف {\n            زر(\"احفظ\") {\n                احفظ(م.نسخة(الملاحظات: الملاحظات))\n                انتقل(\"/\")\n            }\n            زر(\"احذف\") {\n                احذف(م)\n                انتقل(\"/\")\n            }\n        }\n    }\n}",
    "library": "// عناصر الواجهة وتخصيصها: مكتبة كتب صغيرة، بألسنة ونموذج وجدول ونافذة\n\nسمة(اللون: \"فيروزي\"، الزوايا: 12)\n\nثابت التصنيفات = [\"رواية\"، \"شعر\"، \"تاريخ\"، \"علوم\"]\nثابت الحالات = [\"أريد قراءته\"، \"أقرؤه\"، \"قرأته\"]\n\nجدول كتاب {\n    العنوان: نص\n    المؤلف: نص\n    التصنيف = \"رواية\"\n    الحالة = \"أريد قراءته\"\n    التاريخ = \"\"\n    الملاحظات = \"\"\n}\n\nخادم دالة كل_الكتب() = كتاب.الكل()\n\nخادم دالة أضف(\n    العنوان: نص،\n    المؤلف: نص،\n    التصنيف: نص،\n    الحالة: نص،\n    التاريخ: نص،\n    الملاحظات: نص\n) = كتاب.أنشئ(\n    العنوان: العنوان،\n    المؤلف: المؤلف،\n    التصنيف: التصنيف،\n    الحالة: الحالة،\n    التاريخ: التاريخ،\n    الملاحظات: الملاحظات\n)\n\nخادم دالة احذف_بالرقم(رقم: صحيح) = كتاب.بالرقم(رقم)؟.احذف()\n\nمكون الرئيسية() {\n    حالة اللسان = \"الكتب\"\n    حالة الرسالة = \"\"\n    حالة العنوان = \"\"\n    حالة المؤلف = \"\"\n    حالة التصنيف = \"رواية\"\n    حالة القراءة = \"أريد قراءته\"\n    حالة التاريخ = \"\"\n    حالة الملاحظات = \"\"\n    حالة التأكيد = خطأ\n    حالة المختار = 0\n    ثابت الكتب = كل_الكتب()\n    عمود(مسافة: 20) {\n        صف(صنف: \"رأس\") {\n            أيقونة(\"كتاب\"، حجم: \"كبير\"، لون: \"أساسي\")\n            عنوان(\"مكتبتي\")\n        }\n        إذا الرسالة != \"\" { تنبيه(الرسالة، نمط: \"نجاح\") }\n        ألسنة(اللسان) {\n            لسان(\"الكتب\"، أيقونة: \"قائمة\") {\n                جدول_عرض(\n                    الكتب،\n                    أعمدة: [\"العنوان\"، \"المؤلف\"، \"التصنيف\"، \"الحالة\"]،\n                    فارغ: \"لا كتب بعد. أضف أول كتاب.\"\n                ) { الكتاب =>\n                    زر(\"احذف\"، نمط: \"خفيف\"، حجم: \"صغير\"، أيقونة: \"حذف\") {\n                        المختار = الكتاب.رقم\n                        التأكيد = صح\n                    }\n                }\n            }\n            لسان(\"أضف كتاباً\"، أيقونة: \"أضف\") {\n                بطاقة(مسافة: 14) {\n                    حقل(العنوان، تلميح: \"العنوان\")\n                    حقل(المؤلف، تلميح: \"المؤلف\")\n                    قائمة_منسدلة(التصنيف، التصنيفات)\n                    اختيار(القراءة، الحالات)\n                    حقل(التاريخ، صيغة: \"تاريخ\")\n                    منطقة_نص(الملاحظات، تلميح: \"ملاحظات…\"، أسطر: 3)\n                    زر(\"احفظ\"، أيقونة: \"حفظ\"، ملء: صح، معطل: العنوان.قص() == \"\") {\n                        أضف(العنوان.قص()، المؤلف، التصنيف، القراءة، التاريخ، الملاحظات)\n                        الرسالة = \"أُضيف «{العنوان.قص()}» إلى مكتبتك.\"\n                        العنوان = \"\"\n                        المؤلف = \"\"\n                        الملاحظات = \"\"\n                        اللسان = \"الكتب\"\n                    }\n                }\n            }\n            لسان(\"حسب التصنيف\"، أيقونة: \"رسم_بياني\") {\n                قائمة_عناصر(التصنيفات) { ت =>\n                    صف {\n                        فقرة(ت)\n                        فقرة(\"({الكتب.عدد { هو.التصنيف == ت }})\"، لون: \"باهت\")\n                    }\n                }\n            }\n        }\n        نافذة(التأكيد، عنوان: \"حذف الكتاب\") {\n            فقرة(\"سيُحذف الكتاب من مكتبتك.\")\n            صف {\n                زر(\"احذف\"، نمط: \"خطر\") {\n                    احذف_بالرقم(المختار)\n                    التأكيد = خطأ\n                    الرسالة = \"حُذف الكتاب.\"\n                }\n                زر(\"إلغاء\"، نمط: \"ثانوي\") { التأكيد = خطأ }\n            }\n        }\n    }\n}",
    "todo": "// تطبيق مهام كامل: بيانات + خادم + واجهة في ملف واحد\n\nجدول مهمة {\n    عنوان: نص\n    منجزة = خطأ\n}\n\nخادم دالة المهام() = مهمة.الكل()\nخادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)\nخادم دالة بدل(م: مهمة) = م.نسخة(منجزة: ليس م.منجزة).احفظ()\n\n// واجهة برمجية عامة للبرامج الأخرى\nمسار جلب \"/api/مهام\" = مهمة.الكل()\n\nمكون قائمة_المهام() {\n    حالة مهام = المهام()\n    حالة جديدة = \"\"\n\n    عمود {\n        عنوان(\"مهامي ({مهام.رشح { ليس هو.منجزة }.عدد} متبقية)\")\n        صف {\n            حقل(جديدة، تلميح: \"مهمة جديدة…\")\n            زر(\"أضف\") {\n                أضف(جديدة)\n                مهام = المهام()\n                جديدة = \"\"\n            }\n        }\n        لكل م في مهام {\n            صف {\n                مربع_اختيار(م.منجزة) {\n                    بدل(م)\n                    مهام = المهام()\n                }\n                فقرة(م.عنوان)\n            }\n        }\n    }\n}",
    "mistakes": "// ثلاث مشكلات: كلمة محجوزة بلا همزة، رقم مكتوب بغير 0-9، ومحرف اتجاه خفي بعد «اسم»\nاذا عمر > ١٨ {\n  ثابت اسم\u202e = \"سارة\"\n}"
  };


  /* ---------- Syntax highlighting for the editor ----------
     The textarea's text is transparent; a <pre> behind it shows the same text in colour.
     The rules follow editors/vscode/syntaxes/dad.tmLanguage.json. */

  var KW = {};
  function kw(cls, words) { words.split(" ").forEach(function (w) { KW[w] = cls; }); }
  kw("hl-kw", "إذا وإلا طابق لكل في من إلى حتى بخطوة طالما أرجع توقف تابع حاول التقط ارفع");
  kw("hl-decl", "ثابت متغير دالة نوع تعداد عقد امتداد استورد خاص خادم مكون حالة جدول مسار");
  kw("hl-const", "صح خطأ عدم");
  kw("hl-op", "و أو ليس هو ك");
  kw("hl-this", "هذا");
  kw("hl-type", "صحيح عشري نص منطقي قائمة قاموس");
  var BUILTINS = { "اطبع": 1, "مشكلة": 1, "جذر": 1, "مطلق": 1, "أدنى": 1, "أقصى": 1, "انتقل": 1,
    "عمود": 1, "صف": 1, "بطاقة": 1, "عنوان": 1, "فقرة": 1, "زر": 1, "حقل": 1, "مربع_اختيار": 1, "صورة": 1, "رابط": 1, "فاصل": 1, "سمة": 1,
    "قائمة_منسدلة": 1, "اختيار": 1, "منطقة_نص": 1, "جدول_عرض": 1, "قائمة_عناصر": 1, "ألسنة": 1, "لسان": 1, "نافذة": 1, "تنبيه": 1, "أيقونة": 1 };
  var NAMES_TYPE = { "نوع": 1, "تعداد": 1, "عقد": 1, "امتداد": 1, "مكون": 1, "جدول": 1 };

  var WORD = /[\p{L}\p{Mn}_][\p{L}\p{Mn}0-9_]*/uy;
  var NUM = /[0-9](?:[0-9_]*[0-9])?(?:\.[0-9](?:[0-9_]*[0-9])?)?/y;
  // Only 0-9 are digits in Ḍād; Arabic-Indic and Persian digits are shown as mistakes.
  var BAD_DIGITS = /[\u0660-\u0669\u06F0-\u06F9\u066B\u066C]+/y;
  var OP = /=>|[?؟][.?؟]|\.\.|==|!=|<=|>=|[-+*\/]=|&&|\|\||[-+*\/%<>=!?؟:]/y;
  var CALL_AFTER = /\s*\(/y;
  var BIDI = /[‪-‮⁦-⁩‎‏؜]/;

  function esc(t) {
    return t.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }
  function span(cls, t) { return '<span class="' + cls + '">' + esc(t) + "</span>"; }

  function sticky(re, src, i) { re.lastIndex = i; var m = re.exec(src); return m ? m[0] : null; }

  // Highlights code; `stop` ends at an unmatched "}" (used inside {…} in strings).
  function highlightCode(src, i, stop) {
    var out = "";
    var prevWord = "";
    var depth = 0;
    while (i < src.length) {
      var c = src[i];
      var two = src.substr(i, 2);
      if (stop && c === "}" && depth === 0) break;
      if (two === "//") {
        var nl = src.indexOf("\n", i);
        if (nl < 0) nl = src.length;
        out += span("hl-com", src.slice(i, nl)); i = nl; continue;
      }
      if (two === "/*") {
        var end = src.indexOf("*/", i + 2);
        end = end < 0 ? src.length : end + 2;
        out += span("hl-com", src.slice(i, end)); i = end; continue;
      }
      if (c === '"' || c === "«") {
        var r = highlightString(src, i);
        out += r.html; i = r.end; prevWord = ""; continue;
      }
      var w = sticky(WORD, src, i);
      if (w) {
        var cls = KW[w];
        // «صفحة» is a keyword only before an address: صفحة "/المهام" = …
        if (!cls && w === "صفحة" && /^[ \t]*"/.test(src.slice(i + w.length, i + w.length + 8))) cls = "hl-decl";
        if (!cls) {
          if (prevWord === "دالة") cls = "hl-fn";
          else if (NAMES_TYPE[prevWord]) cls = "hl-type";
          else if (sticky(CALL_AFTER, src, i + w.length) !== null) cls = BUILTINS[w] ? "hl-builtin" : "hl-fn";
        }
        out += cls ? span(cls, w) : esc(w);
        prevWord = w; i += w.length; continue;
      }
      var n = sticky(NUM, src, i);
      if (n) { out += span("hl-num", n); i += n.length; prevWord = ""; continue; }
      var bd = sticky(BAD_DIGITS, src, i);
      if (bd) { out += span("hl-bad", bd); i += bd.length; prevWord = ""; continue; }
      var o = sticky(OP, src, i);
      if (o) { out += span("hl-op", o); i += o.length; continue; }
      if (BIDI.test(c)) { out += span("hl-bad", c); i++; continue; }
      if (c === "{") depth++;
      if (c === "}") depth--;
      if (!/\s/.test(c)) prevWord = "";
      out += esc(c); i++;
    }
    return { html: out, end: i };
  }

  function highlightString(src, i) {
    var triple = src.substr(i, 3) === '"""';
    var close = triple ? '"""' : src[i] === "«" ? "»" : '"';
    var open = triple ? '"""' : src[i];
    var out = span("hl-str", open);
    i += open.length;
    var buf = "";
    while (i < src.length) {
      var c = src[i];
      if (!triple && c === "\n") break;
      if (src.substr(i, close.length) === close) {
        out += span("hl-str", buf + close); return { html: out, end: i + close.length };
      }
      if (c === "\\" && i + 1 < src.length) {
        out += span("hl-str", buf); buf = "";
        out += span("hl-esc", src.substr(i, 2)); i += 2; continue;
      }
      if (c === "{") {
        out += span("hl-str", buf); buf = "";
        var inner = highlightCode(src, i + 1, true);
        out += span("hl-interp", "{") + inner.html;
        i = inner.end;
        if (src[i] === "}") { out += span("hl-interp", "}"); i++; }
        continue;
      }
      buf += c; i++;
    }
    return { html: out + span("hl-str", buf), end: i };
  }

  function highlight(src) {
    // A trailing newline needs a character after it, or the <pre> is one line shorter.
    return highlightCode(src, 0, false).html + "\n ";
  }

  var booting = null;

  function loadScript(src) {
    return new Promise(function (resolve, reject) {
      if (window.Go) return resolve();
      var el = document.createElement("script");
      el.src = src;
      el.onload = function () { resolve(); };
      el.onerror = function () { reject(new Error("script")); };
      document.head.appendChild(el);
    });
  }

  function instantiate(url, imports) {
    var viaBuffer = function () {
      return fetch(url).then(function (r) { return r.arrayBuffer(); })
        .then(function (b) { return WebAssembly.instantiate(b, imports); });
    };
    if (!WebAssembly.instantiateStreaming) return viaBuffer();
    return WebAssembly.instantiateStreaming(fetch(url), imports).catch(viaBuffer);
  }

  function boot() {
    if (booting) return booting;
    booting = loadScript(assets + "playground/wasm_exec.js").then(function () {
      var go = new window.Go();
      var ready = new Promise(function (resolve) {
        window.addEventListener("dad-ready", resolve, { once: true });
      });
      return instantiate(assets + "playground/dad.wasm", go.importObject).then(function (res) {
        go.run(res.instance);
        return ready;
      });
    });
    booting.catch(function () { booting = null; });
    return booting;
  }

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text != null) n.textContent = text;
    return n;
  }


  function renderDiagnostics(box, diags) {
    box.textContent = "";
    diags.forEach(function (d) {
      var li = el("li", d.warning ? "is-warning" : "is-error", d.message);
      li.dir = "rtl";
      box.appendChild(li);
    });
  }

  function renderTokens(box, tokens) {
    box.textContent = "";
    var line = null;
    var lineNo = 0;
    tokens.forEach(function (t) {
      if (t.line !== lineNo) {
        lineNo = t.line;
        line = el("div", "dad-play__line");
        line.appendChild(el("span", "dad-play__ln", String(t.line)));
        box.appendChild(line);
      }
      if (t.category === "newline") return;
      var shown = t.category === "string" ? t.text : (t.value || t.text);
      var chip = el("span", "tok tok--" + t.category, shown);
      chip.title = t.kind + " · " + t.line + ":" + t.col;
      line.appendChild(chip);
    });
  }

  function init() {
    var src = document.getElementById("dad-src");
    if (!src || src.dataset.ready) return;
    src.dataset.ready = "1";

    var sample = document.getElementById("dad-sample");
    var status = document.getElementById("dad-status");
    var runBtn = document.getElementById("dad-run");
    var formatBtn = document.getElementById("dad-format");
    var diags = document.getElementById("dad-diags");
    var out = document.getElementById("dad-out");
    var tokens = document.getElementById("dad-tokens");
    var tabOut = document.getElementById("dad-tab-out");
    var tabTokens = document.getElementById("dad-tab-tokens");
    var tabUI = document.getElementById("dad-tab-ui");
    var uiPane = document.getElementById("dad-ui");
    var uiFrame = document.getElementById("dad-ui-frame");
    var timer = null;
    var hl = document.getElementById("dad-hl");
    var hlPre = hl.parentNode;

    // Grow the editor with its content so the coloured copy never needs its own scrollbar.
    function paint() {
      hl.innerHTML = highlight(src.value);
      src.style.height = "auto";
      src.style.height = Math.max(src.scrollHeight + 2, hlPre.scrollHeight) + "px";
    }
    src.addEventListener("input", paint);

    function showTab(which) {
      [["out", tabOut, out], ["ui", tabUI, uiPane], ["tokens", tabTokens, tokens]].forEach(function (t) {
        if (!t[1]) return;
        t[1].setAttribute("aria-selected", String(t[0] === which));
        t[2].hidden = t[0] !== which;
      });
    }
    tabOut.addEventListener("click", function () { showTab("out"); });
    tabTokens.addEventListener("click", function () { showTab("tokens"); });
    if (tabUI) tabUI.addEventListener("click", function () { showTab("ui"); });

    // The interface runs in a frame with the same page, stylesheet and script as
    // «dad dev»; its events come back here to the interpreter in WebAssembly.
    function esc(s) {
      return String(s).replace(/[&<>"]/g, function (c) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]; });
    }
    window.dadPlaygroundEvent = function (event) {
      var reply = JSON.parse(window.dadUIEvent(JSON.stringify(event)));
      if (reply.output) {
        if (out.classList.contains("is-empty")) { out.textContent = ""; out.classList.remove("is-empty"); }
        out.appendChild(document.createTextNode(reply.output));
      }
      return reply;
    };
    function showUI(ui) {
      if (!tabUI) return;
      tabUI.hidden = false;
      uiFrame.srcdoc = '<!doctype html><html lang="ar" dir="rtl"><head><meta charset="utf-8">' +
        '<meta name="viewport" content="width=device-width, initial-scale=1">' +
        '<base href="' + esc(assets + "ui/") + '"><link rel="stylesheet" href="ui.css"><title>' + esc(ui.title) + "</title></head><body>" +
        '<main id="dad-root" data-session="playground" data-routing="none" data-version="' + ui.version + '">' + ui.html + "</main>" +
        '<div id="dad-error" role="alert"' + (ui.error ? "" : " hidden") + ">" + esc(ui.error || "") + "</div>" +
        "<script>window.dadSend = function (e) { return parent.dadPlaygroundEvent(e); };<\/script>" +
        '<script src="ui.js"><\/script></body></html>';
      showTab("ui");
    }
    function hideUI() {
      if (!tabUI) return;
      tabUI.hidden = true;
      uiFrame.removeAttribute("srcdoc");
    }

    function check() {
      var res = JSON.parse(window.dadTokenize(src.value));
      renderDiagnostics(diags, res.diagnostics);
      renderTokens(tokens, res.tokens);
    }

    function run() {
      var res = JSON.parse(window.dadRun(src.value));
      renderDiagnostics(diags, res.diagnostics);
      out.textContent = "";
      out.classList.remove("is-empty");
      if (res.output) out.appendChild(document.createTextNode(res.output));
      res.notices.forEach(function (n) { out.appendChild(el("span", "dad-play__notice", n + "\n")); });
      if (res.error) out.appendChild(el("span", "dad-play__error", res.error + "\n"));
      if (!res.output && !res.error && res.notices.length === 0) {
        out.textContent = res.ui ? L.uiOut : L.noOut;
        out.classList.add("is-empty");
      }
      if (res.ui) showUI(res.ui);
      else { hideUI(); showTab("out"); }
    }

    function load(name) {
      src.value = SAMPLES[name];
      paint();
      out.textContent = L.press;
      out.classList.add("is-empty");
      hideUI();
      showTab("out");
      check();
    }

    src.value = SAMPLES[sample.value];
    paint();
    status.textContent = L.loading;

    boot().then(function () {
      status.textContent = L.ready;
      status.classList.add("is-ready");
      runBtn.disabled = false;
      load(sample.value);
      run();
      src.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(check, 200);
      });
      // Format lays the code out like «dad fmt»: four spaces per level, Kotlin style.
      function format() {
        if (typeof window.dadFormat !== "function") return;
        var res = JSON.parse(window.dadFormat(src.value));
        if (res.error) {
          status.textContent = L.fixFirst;
          return;
        }
        if (res.text !== src.value) {
          src.value = res.text;
          src.dispatchEvent(new Event("input"));
        }
      }
      if (formatBtn) {
        formatBtn.disabled = false;
        formatBtn.addEventListener("click", format);
      }
      src.addEventListener("keydown", function (e) {
        if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
          e.preventDefault();
          run();
        } else if (e.shiftKey && e.altKey && e.code === "KeyF") {
          e.preventDefault();
          format();
        }
      });
      runBtn.addEventListener("click", run);
      sample.addEventListener("change", function () { load(sample.value); run(); });
    }).catch(function () {
      status.textContent = L.failed;
      status.classList.add("is-error");
    });
  }

  if (window.document$ && typeof window.document$.subscribe === "function") {
    window.document$.subscribe(init);
  } else if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
