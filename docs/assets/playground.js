/* Ḍād playground: runs the Go lexer, parser and interpreter (compiled to WebAssembly) in the browser. */
(function () {
  "use strict";

  var assets = (function () {
    var s = document.currentScript;
    return s ? new URL(".", s.src).href : "";
  })();

  var SAMPLES = {
    "hello": "// أول برنامج بلغة ضاد\nثابت اسم = \"العالم\"\nاطبع(\"مرحبا يا {اسم}!\")\n\nدالة حي(اسم: نص، تحية: نص = \"أهلاً\"): نص {\n  أرجع \"{تحية} يا {اسم}\"\n}\n\nاطبع(حي(\"سارة\"))\nاطبع(حي(اسم: \"علي\"، تحية: «صباح الخير»))",
    "basics": "// المتغيرات، الشروط، الحلقات، والدوال المختصرة\nدالة مربع(س: صحيح) = س * س\n\nدالة وصف(درجة: صحيح) = إذا درجة >= ٩٠ { \"ممتاز\" } وإلا إذا درجة >= ٥٠ { \"ناجح\" } وإلا { \"راسب\" }\n\nمتغير مجموع = ٠\nلكل ع من ١ إلى ١٠ {\n  مجموع += مربع(ع)\n}\nاطبع(\"مجموع المربعات: {مجموع}\")\n\nلكل ع من ٠ حتى ١٠ بخطوة ٢ { اطبع(ع) }\n\nثابت درجات = [٩٥، 72، ٤٠]\nلكل د في درجات {\n  اطبع(\"{د} => {وصف(د)}\")\n}\n\nثابت كبار = درجات.رشح { هو >= ٥٠ }.حول { هو * ٢ }\nثابت متوسط = درجات.اختزل(٠) { أ، ب => أ + ب } ك عشري / ٣٫٠\nاطبع(\"كبار:\", كبار)\nاطبع(\"المتوسط:\", متوسط)",
    "shapes": "// أنواع القيم، التعدادات، والمطابقة الشاملة\nنوع نقطة(س: عشري، ص: عشري) {\n  دالة بعد() = جذر(س * س + ص * ص)\n}\n\nتعداد شكل {\n  دائرة(قطر: عشري)\n  مستطيل(عرض: عشري، طول: عشري)\n  مثلث(قاعدة: عشري، ارتفاع: عشري)\n}\n\nدالة مساحة(ش: شكل): عشري = طابق ش {\n  دائرة(ق) => ٣٫١٤١٦ * ق * ق / ٤\n  مستطيل(ع، ط) => ع * ط\n  مثلث(ق، ر) => ق * ر / ٢\n}\n\nثابت أشكال = [دائرة(٢)، مستطيل(٣، ٤)، مثلث(قاعدة: ٦، ارتفاع: ٢)]\nلكل ش في أشكال {\n  اطبع(\"المساحة = {مساحة(ش)}\")\n  إذا ش هو دائرة { اطبع(\"قطرها {ش.قطر}\") }\n}\n\nثابت ن = نقطة(س: ٣، ص: ٤)\nاطبع(ن.بعد())              // ٥\nاطبع(ن == نقطة(٣، ٤))     // صح\nثابت م = ن.نسخة(ص: ٠)",
    "safety": "// الأمان من القيم المفقودة، الأخطاء، العقود والامتدادات\nنوع مستخدم(اسم: نص، بريد: نص؟ = عدم)\n\nدالة بريد_العرض(م: مستخدم؟): نص {\n  ثابت بريد = م؟.بريد\n  إذا بريد != عدم {\n    أرجع \"{بريد} ({بريد.طول} حرفاً)\"     // تحويل ذكي: بريد هنا نص\n  }\n  أرجع م؟.بريد ؟؟ \"لا يوجد بريد\"\n}\n\nدالة اقسم(أ: صحيح، ب: صحيح): صحيح {\n  إذا ب == ٠ { ارفع مشكلة(\"قسمة على صفر\") }\n  أرجع أ / ب\n}\n\nثابت ناتج = حاول { اقسم(١٠، ٠) } التقط خ { اطبع(خ.رسالة)؛ ٠ }\n\nعقد قابل_للعرض {\n  دالة اعرض(): نص\n}\n\nامتداد نص {\n  خاص دالة نظيف() = هذا.قص()\n  دالة مكرر(مرات: صحيح) = (من ١ إلى مرات).حول { هذا }.ضم(\"\")\n}\n\nاطبع(\"هلا \".مكرر(3))\nاطبع(بريد_العرض(مستخدم(\"هدى\"، بريد: \"huda@example.com\")))\nاطبع(بريد_العرض(عدم))",
    "mistakes": "// ثلاث مشكلات: كلمة محجوزة بلا همزة، أرقام مختلطة، ومحرف اتجاه خفي بعد «اسم»\nاذا عمر > ١8 {\n  ثابت اسم\u202e = \"سارة\"\n}"
};

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

  function toArabicDigits(n) {
    return String(n).replace(/[0-9]/g, function (d) { return "٠١٢٣٤٥٦٧٨٩"[d]; });
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
        line.appendChild(el("span", "dad-play__ln", toArabicDigits(t.line)));
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
    var diags = document.getElementById("dad-diags");
    var out = document.getElementById("dad-out");
    var tokens = document.getElementById("dad-tokens");
    var tabOut = document.getElementById("dad-tab-out");
    var tabTokens = document.getElementById("dad-tab-tokens");
    var timer = null;

    function showTab(which) {
      var isOut = which === "out";
      tabOut.setAttribute("aria-selected", String(isOut));
      tabTokens.setAttribute("aria-selected", String(!isOut));
      out.hidden = !isOut;
      tokens.hidden = isOut;
    }
    tabOut.addEventListener("click", function () { showTab("out"); });
    tabTokens.addEventListener("click", function () { showTab("tokens"); });

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
        out.textContent = "انتهى البرنامج دون أن يطبع شيئاً.";
        out.classList.add("is-empty");
      }
      showTab("out");
    }

    function load(name) {
      src.value = SAMPLES[name];
      out.textContent = "اضغط «Run» لتشغيل البرنامج.";
      out.classList.add("is-empty");
      check();
    }

    src.value = SAMPLES[sample.value];
    status.textContent = "Loading Ḍād…";

    boot().then(function () {
      status.textContent = "Ready";
      status.classList.add("is-ready");
      runBtn.disabled = false;
      load(sample.value);
      run();
      src.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(check, 200);
      });
      src.addEventListener("keydown", function (e) {
        if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
          e.preventDefault();
          run();
        }
      });
      runBtn.addEventListener("click", run);
      sample.addEventListener("change", function () { load(sample.value); run(); });
    }).catch(function () {
      status.textContent = "Ḍād couldn't load. Check your connection and reload the page.";
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
