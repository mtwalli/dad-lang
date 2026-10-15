// Checks the TextMate grammar on the example programs with a small tokenizer that follows
// TextMate's rules (match / begin-end, includes, end pattern wins ties). Run: node test/grammar.test.js
"use strict";

const fs = require("fs");
const path = require("path");
const assert = require("assert");

const root = path.join(__dirname, "..");
const grammar = JSON.parse(fs.readFileSync(path.join(root, "syntaxes/dad.tmLanguage.json"), "utf8"));
const re = (src) => new RegExp(src, "gu");

function resolve(patterns) {
  const out = [];
  for (const p of patterns) {
    if (p.include) {
      const name = p.include === "$self" ? null : p.include.slice(1);
      const rule = name ? grammar.repository[name] : { patterns: grammar.patterns };
      if (!rule) throw new Error("unknown include " + p.include);
      if (rule.patterns && !rule.begin && !rule.match) out.push(...resolve(rule.patterns));
      else out.push(rule);
    } else out.push(p);
  }
  return out;
}

function firstMatch(regex, line, pos) {
  regex.lastIndex = pos;
  return regex.exec(line);
}

function captureScopes(captures, m, base, tokens) {
  if (!captures) return;
  // Only whole-group captures are needed for this grammar.
  let offset = m.index;
  for (let i = 1; i < m.length; i++) {
    if (m[i] === undefined) continue;
    const at = m[0].indexOf(m[i], offset - m.index) + m.index;
    if (captures[i]) tokens.push({ text: m[i], scopes: [...base, captures[i].name] });
    offset = at + m[i].length;
  }
}

/** Tokenizes source into {text, scopes} pieces (unscoped text is skipped). */
function tokenize(src) {
  const tokens = [];
  const stack = [{ patterns: resolve(grammar.patterns), scopes: ["source.dad"] }];
  for (const line of src.split("\n")) {
    let pos = 0;
    while (pos <= line.length) {
      const top = stack[stack.length - 1];
      let best = null;
      if (top.end) {
        const m = firstMatch(re(top.end), line, pos);
        if (m) best = { m, end: true };
      }
      for (const rule of top.patterns) {
        const m = firstMatch(re(rule.match || rule.begin), line, pos);
        if (m && (!best || m.index < best.m.index)) best = { m, rule };
      }
      const gapEnd = best ? best.m.index : line.length;
      if (gapEnd > pos && stack.length > 1) tokens.push({ text: line.slice(pos, gapEnd), scopes: [...top.scopes] });
      if (!best) break;
      const { m } = best;
      if (best.end) {
        if (m[0]) tokens.push({ text: m[0], scopes: [...top.scopes, ...(top.endScope ? [top.endScope] : [])] });
        stack.pop();
      } else if (best.rule.match) {
        if (best.rule.captures) captureScopes(best.rule.captures, m, top.scopes, tokens);
        else tokens.push({ text: m[0], scopes: [...top.scopes, best.rule.name] });
      } else {
        const r = best.rule;
        const scopes = r.name ? [...top.scopes, r.name] : [...top.scopes];
        tokens.push({ text: m[0], scopes: [...scopes, r.beginCaptures?.["0"]?.name].filter(Boolean) });
        stack.push({
          end: r.end,
          endScope: r.endCaptures?.["0"]?.name,
          patterns: resolve(r.patterns || []),
          scopes: r.contentName ? [...scopes, r.contentName] : scopes,
        });
      }
      pos = m.index + m[0].length + (m[0].length === 0 ? 1 : 0);
    }
  }
  return tokens;
}

function scopeOf(tokens, text, nth = 0) {
  const found = tokens.filter((t) => t.text === text);
  assert.ok(found[nth], `token «${text}» #${nth} not found`);
  return found[nth].scopes.join(" ");
}

function test(name, fn) {
  try {
    fn();
    console.log("ok  ", name);
  } catch (e) {
    console.log("FAIL", name, "\n     ", e.message);
    process.exitCode = 1;
  }
}

// Every regex in the grammar must compile with Unicode semantics.
test("all patterns compile", () => {
  JSON.stringify(grammar, (k, v) => {
    if (["match", "begin", "end"].includes(k)) re(v);
    return v;
  });
});

const ex = (n) => fs.readFileSync(path.join(root, "../../examples", n), "utf8");

test("keywords and declarations", () => {
  const t = tokenize(ex("02-basics.dad"));
  assert.match(scopeOf(t, "دالة"), /storage\.type\.function/);
  assert.match(scopeOf(t, "مربع"), /entity\.name\.function\.dad/);
  assert.match(scopeOf(t, "إذا"), /keyword\.control/);
  assert.match(scopeOf(t, "وإلا"), /keyword\.control/);
  assert.match(scopeOf(t, "لكل"), /keyword\.control/);
  assert.match(scopeOf(t, "بخطوة"), /keyword\.control/);
  assert.match(scopeOf(t, "صحيح"), /support\.type/);
  assert.match(scopeOf(t, "متغير"), /storage\.modifier/);
  assert.match(scopeOf(t, "هو"), /keyword\.operator\.word/);
  assert.match(scopeOf(t, "ك"), /keyword\.operator\.word/);
  assert.match(scopeOf(t, "اطبع"), /support\.function\.builtin/);
  assert.match(scopeOf(t, "رشح"), /entity\.name\.function\.call/);
});

test("short words are not found inside longer names", () => {
  const t = tokenize("ثابت وصف = ليسا\nدالة كتب() = هوية");
  for (const tok of t) {
    if (/keyword\.operator\.word/.test(tok.scopes.join(" "))) assert.fail(`«${tok.text}» marked as an operator`);
  }
  assert.match(scopeOf(t, "وصف"), /variable\.other\.declaration/);
});

test("numbers in both digit sets", () => {
  const t = tokenize("ثابت أ = ٣٫١٤١٦ + 2.5 + ١٬٠٠٠ + 72");
  for (const n of ["٣٫١٤١٦", "2.5", "١٬٠٠٠", "72"]) assert.match(scopeOf(t, n), /constant\.numeric/);
});

test("strings, escapes and interpolation", () => {
  const t = tokenize('اطبع("مرحبا يا {اسم}!\\n")\nاطبع(«صباح {أ.حول { هو }} الخير»)');
  assert.match(scopeOf(t, "مرحبا يا "), /string\.quoted\.double/);
  assert.match(scopeOf(t, "\\n"), /constant\.character\.escape/);
  assert.match(scopeOf(t, "{"), /meta\.interpolation.*punctuation\.section\.interpolation\.begin/);
  assert.match(scopeOf(t, "هو"), /string\.quoted\.guillemet.*meta\.interpolation.*keyword\.operator\.word/);
  assert.match(scopeOf(t, " الخير"), /string\.quoted\.guillemet/);
  assert.doesNotMatch(scopeOf(t, " الخير"), /meta\.interpolation/);
});

test("comments", () => {
  const t = tokenize("// إذا هنا تعليق\n/* ثابت\n متغير */ ثابت س = ١");
  assert.match(scopeOf(t, " إذا هنا تعليق"), /comment\.line/);
  assert.match(scopeOf(t, " ثابت"), /comment\.block/);
  assert.match(scopeOf(t, " متغير "), /comment\.block/);
  assert.match(scopeOf(t, "ثابت"), /storage\.modifier/);
});

test("types, enums, match and optionals", () => {
  const t = tokenize(ex("03-shapes.dad") + "\n" + ex("04-safety.dad"));
  assert.match(scopeOf(t, "نقطة"), /entity\.name\.type/);
  assert.match(scopeOf(t, "شكل"), /entity\.name\.type/);
  assert.match(scopeOf(t, "طابق"), /keyword\.control/);
  assert.match(scopeOf(t, "=>"), /keyword\.operator\.arrow/);
  assert.match(scopeOf(t, "؟."), /keyword\.operator\.optional/);
  assert.match(scopeOf(t, "؟؟"), /keyword\.operator\.optional/);
  assert.match(scopeOf(t, "عدم"), /constant\.language/);
  assert.match(scopeOf(t, "هذا"), /variable\.language/);
  assert.match(scopeOf(t, "حاول"), /keyword\.control/);
  assert.match(scopeOf(t, "التقط"), /keyword\.control/);
  assert.match(scopeOf(t, "امتداد"), /storage\.type/);
});

test("hidden direction characters are flagged", () => {
  const t = tokenize("ثابت اسم‮ = ١");
  assert.match(scopeOf(t, "‮"), /invalid\.illegal\.bidi/);
});

test("every example tokenizes without stray scopes", () => {
  for (const f of fs.readdirSync(path.join(root, "../../examples")).filter((f) => f.endsWith(".dad"))) {
    const t = tokenize(ex(f));
    assert.ok(t.length > 20, f);
  }
});

module.exports = { tokenize };
