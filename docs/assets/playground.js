/* Ḍād playground: runs the Go lexer (compiled to WebAssembly) in the browser. */
(function () {
  "use strict";

  var assets = (function () {
    var s = document.currentScript;
    return s ? new URL(".", s.src).href : "";
  })();

  var SAMPLES = {
    hello: [
      "// أول برنامج بلغة ضاد",
      'ثابت اسم = "العالم"',
      'اطبع("مرحبا يا {اسم}!")',
      "",
      'دالة حي(اسم: نص، تحية: نص = "أهلاً"): نص {',
      '  أرجع "{تحية} يا {اسم}"',
      "}",
      "",
      'اطبع(حي(اسم: "علي"، تحية: «صباح الخير»))',
    ].join("\n"),
    shapes: [
      "تعداد شكل {",
      "  دائرة(قطر: عشري)",
      "  مستطيل(عرض: عشري، طول: عشري)",
      "}",
      "",
      "دالة مساحة(ش: شكل): عشري = طابق ش {",
      "  دائرة(ق) => ٣٫١٤١٦ * ق * ق / ٤",
      "  مستطيل(ع، ط) => ع * ط",
      "}",
    ].join("\n"),
    todo: [
      "جدول مهمة {",
      "  عنوان: نص",
      "  منجزة = خطأ",
      "}",
      "",
      "خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)",
      "",
      "مكون عداد() {",
      "  حالة ع = ٠",
      '  زر("زد {ع}") { ع += ١ }',
      "}",
    ].join("\n"),
    mistakes: [
      "// ثلاث مشكلات: كلمة محجوزة بلا همزة، أرقام مختلطة، ومحرف اتجاه خفي بعد «اسم»",
      "اذا عمر > ١8 {",
      '  ثابت اسم‮ = "سارة"',
      "}",
    ].join("\n"),
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

  function boot() {
    if (booting) return booting;
    booting = loadScript(assets + "playground/wasm_exec.js").then(function () {
      var go = new window.Go();
      var url = assets + "playground/dad.wasm";
      var ready = new Promise(function (resolve) {
        window.addEventListener("dad-ready", resolve, { once: true });
      });
      var load = WebAssembly.instantiateStreaming
        ? WebAssembly.instantiateStreaming(fetch(url), go.importObject).catch(function () {
            return fetch(url).then(function (r) { return r.arrayBuffer(); })
              .then(function (b) { return WebAssembly.instantiate(b, go.importObject); });
          })
        : fetch(url).then(function (r) { return r.arrayBuffer(); })
            .then(function (b) { return WebAssembly.instantiate(b, go.importObject); });
      return load.then(function (res) {
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

  function render(src, diagsBox, tokensBox) {
    var res = JSON.parse(window.dadTokenize(src.value));

    diagsBox.textContent = "";
    if (res.diagnostics.length === 0) {
      diagsBox.appendChild(el("li", "dad-play__ok", "✓ لا أخطاء"));
    }
    res.diagnostics.forEach(function (d) {
      var li = el("li", d.warning ? "is-warning" : "is-error", d.message);
      li.dir = "rtl";
      diagsBox.appendChild(li);
    });

    tokensBox.textContent = "";
    var line = null;
    var lineNo = 0;
    res.tokens.forEach(function (t) {
      if (t.line !== lineNo) {
        lineNo = t.line;
        line = el("div", "dad-play__line");
        line.appendChild(el("span", "dad-play__ln", toArabicDigits(t.line)));
        tokensBox.appendChild(line);
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
    var diags = document.getElementById("dad-diags");
    var tokens = document.getElementById("dad-tokens");
    var timer = null;

    src.value = SAMPLES[sample.value];
    status.textContent = "Loading the Ḍād lexer…";

    boot().then(function () {
      status.textContent = "Ready. Results update as you type.";
      status.classList.add("is-ready");
      render(src, diags, tokens);
      src.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(function () { render(src, diags, tokens); }, 150);
      });
      sample.addEventListener("change", function () {
        src.value = SAMPLES[sample.value];
        render(src, diags, tokens);
      });
    }).catch(function () {
      status.textContent = "The lexer couldn't load. Check your connection and reload the page.";
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
