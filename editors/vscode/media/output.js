// The Ḍād output panel: shows a program's output, right to left, until the next run.
(function () {
  "use strict";
  const vscode = acquireVsCodeApi();
  const out = document.getElementById("out");
  const title = document.getElementById("title");
  const state = document.getElementById("state");
  const stop = document.getElementById("stop");
  const rerun = document.getElementById("rerun");

  stop.addEventListener("click", () => vscode.postMessage({ type: "stop" }));
  rerun.addEventListener("click", () => vscode.postMessage({ type: "rerun" }));

  function append(text, cls) {
    const atBottom = window.innerHeight + window.scrollY >= document.body.scrollHeight - 4;
    const last = out.lastChild;
    if (last && last.nodeType === 1 && last.className === cls) last.textContent += text;
    else {
      const span = document.createElement("span");
      span.className = cls;
      span.textContent = text;
      out.appendChild(span);
    }
    if (atBottom) window.scrollTo(0, document.body.scrollHeight);
  }

  window.addEventListener("message", (event) => {
    const msg = event.data;
    if (msg.type === "start") {
      out.textContent = "";
      title.textContent = "▶ " + msg.file;
      state.textContent = "يعمل…";
      state.className = "running";
      stop.hidden = false;
      rerun.hidden = true;
    } else if (msg.type === "chunk") {
      append(msg.text, msg.stream === "err" ? "err" : "out");
    } else if (msg.type === "done") {
      const secs = (msg.ms / 1000).toFixed(msg.ms < 10000 ? 2 : 1);
      if (msg.stopped) state.textContent = "أُوقف";
      else if (msg.code === 0) state.textContent = `انتهى في ${secs} ث`;
      else state.textContent = `انتهى بخطأ (رمز ${msg.code})`;
      state.className = msg.code === 0 && !msg.stopped ? "ok" : "fail";
      if (!out.textContent) append("لم يطبع البرنامج شيئاً.", "note");
      stop.hidden = true;
      rerun.hidden = false;
    }
  });
  vscode.postMessage({ type: "ready" });
})();
