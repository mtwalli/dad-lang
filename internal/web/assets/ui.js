// The browser half of a Ḍād page. The program runs on the server (dad dev);
// this script sends clicks and typing to it as events and updates the page in
// place with what it renders, keeping focus and the text being typed.
//
// The page is <main id="dad-root" data-session="…" data-version="…">. A host can
// set window.dadSend(event) → Promise<reply> to deliver events another way
// (the web playground runs the interpreter in the browser).
//
// Pages: a link to one of the program's pages has data-nav, and opens the page in
// place with a «nav» event. data-routing says how the address is kept:
//   "history" (dad dev, dad serve): the real address, with the back button;
//   "hash" (dad build): «#/المهام», so a static site works from any host or from disk;
//   "none" (the playground): not kept.
(function () {
  "use strict";
  var root = document.getElementById("dad-root");
  if (!root) return;
  var session = root.getAttribute("data-session");
  var version = Number(root.getAttribute("data-version"));
  var errorBox = document.getElementById("dad-error");
  var queue = Promise.resolve();
  var pending = new Map(); // input element → timer for typing not sent yet
  var routing = root.getAttribute("data-routing") || "history";
  var shown = currentPath(); // the address of the page on screen

  function currentPath() {
    if (routing === "hash") return decodeURIComponent(location.hash.replace(/^#/, "")) || "/";
    if (routing === "history") return decodeURIComponent(location.pathname);
    return "/";
  }

  function post(event) {
    if (typeof window.dadSend === "function") return Promise.resolve(window.dadSend(event));
    return fetch("/_dad/ui/event", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(event),
    }).then(function (r) { return r.json(); });
  }

  function showError(msg) {
    if (!errorBox) return;
    errorBox.textContent = msg || "";
    errorBox.hidden = !msg;
  }

  // send queues an event. The handler id is read when the event goes out, since
  // the previous reply may have renumbered the page.
  function send(el, attr, value) {
    queue = queue.then(function () {
      if (!el.isConnected) return;
      var id = el.getAttribute(attr);
      if (!id) return;
      var v = typeof value === "function" ? value() : value;
      return post({ s: session, v: version, id: id, value: v }).then(apply, function () {
        showError(root.hasAttribute("data-live")
          ? "انقطع الاتصال بالبرنامج. هل ما زال «ضاد تطوير» يعمل؟"
          : "انقطع الاتصال بالخادم. أعد تحميل الصفحة بعد قليل.");
      });
    });
    return queue;
  }

  function apply(reply) {
    if (!reply) return;
    if (reply.reload) { location.reload(); return; }
    if (reply.location) {
      if (routing === "none") window.open(reply.location, "_blank", "noopener"); // the playground's frame stays
      else location.href = reply.location;
      return;
    }
    showError(reply.error || "");
    if (typeof reply.html === "string") {
      morph(root, reply.html);
      version = reply.version;
      focusDialog();
    }
    if (reply.title) document.title = reply.title;
    if (typeof reply.path === "string" && reply.path !== shown) moved(reply.path);
  }

  // moved records that another page is on screen: in the address bar, and at the top.
  function moved(path) {
    shown = path;
    if (routing === "history") history.pushState(null, "", path);
    else if (routing === "hash") location.hash = "#" + path;
    window.scrollTo(0, 0);
    // Screen readers start reading the new page from its top.
    root.setAttribute("tabindex", "-1");
    root.focus({ preventScroll: true });
  }

  // open shows the page at an address: a link to a page, or the back button.
  function open(path, push) {
    flushTyping();
    queue = queue.then(function () {
      return post({ s: session, v: version, nav: path }).then(function (reply) {
        if (reply && reply.reload) { location.href = path; return; }
        if (!push && reply) shown = reply.path; // already in the address bar
        apply(reply);
      }, function () {
        location.href = path; // no connection: let the browser load it
      });
    });
    return queue;
  }

  // Typing is sent shortly after the last key; anything else sends it at once first.
  function flushTyping() {
    pending.forEach(function (timer, input) {
      clearTimeout(timer);
      pending.delete(input);
      send(input, "data-input", function () { return input.value; });
    });
  }

  root.addEventListener("click", function (e) {
    var link = e.target.closest("a[data-nav]");
    if (link && routing !== "hash") {
      // a click that opens a new tab or window, or downloads, stays the browser's
      if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
      e.preventDefault();
      var path = decodeURIComponent(link.getAttribute("href"));
      if (path !== shown) open(path, true);
      return;
    }
    // a click outside an open dialog closes it, like its × button
    if (e.target.classList && e.target.classList.contains("dad-backdrop")) {
      var x = e.target.querySelector(".dad-close");
      if (x) x.click();
      return;
    }
    var el = e.target.closest("[data-click]");
    if (!el || el.disabled) return;
    flushTyping();
    send(el, "data-click", null);
  });

  // Back and forward show the page of that address again.
  if (routing === "history") {
    window.addEventListener("popstate", function () {
      var path = currentPath();
      if (path !== shown) { shown = path; open(path, false); }
    });
  } else if (routing === "hash") {
    window.addEventListener("hashchange", function () {
      var path = currentPath();
      if (path !== shown) { shown = path; open(path, false); }
    });
  }

  root.addEventListener("input", function (e) {
    var el = e.target;
    if (!el.hasAttribute || !el.hasAttribute("data-input")) return;
    clearTimeout(pending.get(el));
    pending.set(el, setTimeout(function () {
      pending.delete(el);
      send(el, "data-input", function () { return el.value; });
    }, 150));
  });

  root.addEventListener("change", function (e) {
    var el = e.target;
    if (!el.hasAttribute) return;
    if (el.hasAttribute("data-toggle")) {
      flushTyping();
      send(el, "data-toggle", el.checked);
    } else if (el.hasAttribute("data-choose")) {
      // a drop-down list or a radio button: the number of the choice
      flushTyping();
      send(el, "data-choose", el.value);
    }
  });

  root.addEventListener("keydown", function (e) {
    var el = e.target;
    // Esc closes a dialog.
    if (e.key === "Escape") {
      var x = root.querySelector(".dad-dialog .dad-close");
      if (x) { e.preventDefault(); x.click(); }
      return;
    }
    // The arrow keys move between tabs; the page is right to left, so ← is the next one.
    if (el.getAttribute && el.getAttribute("role") === "tab" && (e.key === "ArrowLeft" || e.key === "ArrowRight")) {
      var tabs = Array.prototype.slice.call(el.parentElement.querySelectorAll('[role="tab"]'));
      var step = (e.key === "ArrowLeft") === (getComputedStyle(el).direction === "rtl") ? 1 : -1;
      var next = tabs[(tabs.indexOf(el) + step + tabs.length) % tabs.length];
      e.preventDefault();
      next.focus();
      next.click();
      return;
    }
    // Enter in a field presses the first button next to it, as in a form.
    if (e.key !== "Enter" || el.tagName !== "INPUT" || el.type === "checkbox" || el.type === "radio" || e.isComposing) return;
    var button = el.parentElement && el.parentElement.querySelector("button[data-click]");
    if (!button) return;
    e.preventDefault();
    button.click();
  });

  // A dialog that just opened takes the focus: its first field, or the dialog.
  function focusDialog() {
    var d = root.querySelector(".dad-dialog");
    if (!d || d.contains(document.activeElement)) return;
    var first = d.querySelector("input, select, textarea, button:not(.dad-close), a[href]");
    (first || d).focus();
  }

  // ---------- updating the page in place ----------

  function morph(el, html) {
    var t = document.createElement("template");
    t.innerHTML = html;
    morphChildren(el, t.content);
  }

  function same(a, b) {
    return a.nodeType === b.nodeType && a.nodeName === b.nodeName &&
      (a.nodeType !== 1 || a.tagName !== "INPUT" || a.type === b.getAttribute("type"));
  }

  function morphChildren(a, b) {
    var olds = Array.prototype.slice.call(a.childNodes);
    var news = Array.prototype.slice.call(b.childNodes);
    for (var i = 0; i < news.length; i++) {
      var x = olds[i], y = news[i];
      if (!x) { a.appendChild(y); continue; }
      if (!same(x, y)) { a.replaceChild(y, x); continue; }
      if (x.nodeType === 3) {
        if (x.nodeValue !== y.nodeValue) x.nodeValue = y.nodeValue;
      } else if (x.nodeType === 1) {
        syncAttributes(x, y);
        if (x.tagName === "INPUT") syncInput(x, y);
        else if (x.tagName === "TEXTAREA") syncText(x, y);
        else {
          morphChildren(x, y);
          if (x.tagName === "SELECT") syncSelect(x, y);
        }
      }
    }
    for (var j = olds.length - 1; j >= news.length; j--) a.removeChild(olds[j]);
  }

  function syncAttributes(x, y) {
    var i, at;
    for (i = x.attributes.length - 1; i >= 0; i--) {
      at = x.attributes[i];
      if (!y.hasAttribute(at.name)) x.removeAttribute(at.name);
    }
    for (i = 0; i < y.attributes.length; i++) {
      at = y.attributes[i];
      if (x.getAttribute(at.name) !== at.value) x.setAttribute(at.name, at.value);
    }
  }

  function syncText(x, y) {
    var v = y.value; // a <template>'s textarea holds its text as its value
    if (pending.has(x)) return;
    if (x.value !== v) x.value = v;
  }

  function syncSelect(x, y) {
    var chosen = y.querySelector("option[selected]");
    var v = chosen ? chosen.getAttribute("value") : "";
    if (x.value !== v) x.value = v;
  }

  function syncInput(x, y) {
    if (x.type === "checkbox" || x.type === "radio") {
      x.checked = y.hasAttribute("checked");
      return;
    }
    var v = y.getAttribute("value") || "";
    // Don't overwrite what is being typed right now.
    if (pending.has(x)) return;
    if (x.value !== v) x.value = v;
  }

  // ---------- reloading when the program changes (dad dev) ----------

  var live = Number(root.getAttribute("data-live"));
  if (live) {
    setInterval(function () {
      fetch("/_dad/version").then(function (r) { return r.json(); }).then(function (v) {
        if (v.version !== live) location.reload();
        else if (v.error) showError(v.error);
      }, function () {});
    }, 1000);
  }
})();
