// The browser half of a Ḍād page. The program runs on the server (dad dev);
// this script sends clicks and typing to it as events and updates the page in
// place with what it renders, keeping focus and the text being typed.
//
// The page is <main id="dad-root" data-session="…" data-version="…">. A host can
// set window.dadSend(event) → Promise<reply> to deliver events another way
// (the web playground runs the interpreter in the browser).
(function () {
  "use strict";
  var root = document.getElementById("dad-root");
  if (!root) return;
  var session = root.getAttribute("data-session");
  var version = Number(root.getAttribute("data-version"));
  var errorBox = document.getElementById("dad-error");
  var queue = Promise.resolve();
  var pending = new Map(); // input element → timer for typing not sent yet

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
        showError("انقطع الاتصال بالبرنامج. هل ما زال «ضاد تطوير» يعمل؟");
      });
    });
    return queue;
  }

  function apply(reply) {
    if (!reply) return;
    if (reply.reload) { location.reload(); return; }
    showError(reply.error || "");
    if (typeof reply.html === "string") {
      morph(root, reply.html);
      version = reply.version;
    }
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
    var el = e.target.closest("[data-click]");
    if (!el || el.disabled) return;
    flushTyping();
    send(el, "data-click", null);
  });

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
    if (!el.hasAttribute || !el.hasAttribute("data-toggle")) return;
    flushTyping();
    send(el, "data-toggle", el.checked);
  });

  // Enter in a field presses the first button next to it, as in a form.
  root.addEventListener("keydown", function (e) {
    var el = e.target;
    if (e.key !== "Enter" || el.tagName !== "INPUT" || el.type === "checkbox" || e.isComposing) return;
    var button = el.parentElement && el.parentElement.querySelector("button[data-click]");
    if (!button) return;
    e.preventDefault();
    button.click();
  });

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
        else morphChildren(x, y);
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

  function syncInput(x, y) {
    if (x.type === "checkbox") {
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
