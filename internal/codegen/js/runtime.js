// The runtime of Ḍād (ضاد) programs compiled to JavaScript by «dad build».
// It follows the interpreter (internal/interp) rule for rule, so a program prints
// the same thing however it runs: صحيح values are BigInt, عشري values are
// numbers, عدم is null, and errors carry the interpreter's Arabic messages.
(function (root) {
  "use strict";

  // ---------------------------------------------------------------- values

  class List { constructor(elems) { this.elems = elems; } }
  class Range {
    constructor(from, to, step, inclusive) { Object.assign(this, { from, to, step, inclusive }); }
    values() {
      const out = [];
      if (this.step === 0n) return out;
      for (let i = this.from; rangeContinues(this, i); i += this.step) out.push(i);
      return out;
    }
  }
  function rangeContinues(r, i) {
    if (r.step > 0n) return r.inclusive ? i <= r.to : i < r.to;
    return r.inclusive ? i >= r.to : i > r.to;
  }
  class DMap {
    constructor(keys, vals) { this.keys = keys || []; this.vals = vals || []; }
    find(k) { for (let i = 0; i < this.keys.length; i++) if (equal(this.keys[i], k)) return i; return -1; }
  }
  class StructType {
    constructor(name, fields, table) { Object.assign(this, { name, fields, methods: {}, table: !!table }); }
  }
  class Struct { constructor(type, vals) { this.type = type; this.vals = vals; } }
  class EnumType { constructor(name) { this.name = name; this.variants = []; } }
  class Variant { constructor(en, name, fields) { Object.assign(this, { enum: en, name, fields }); } }
  class EnumVal { constructor(variant, vals) { this.variant = variant; this.vals = vals; } }
  class Problem { constructor(message) { this.message = message; } }
  class Component { constructor(name, params, body) { Object.assign(this, { name, params, body }); } }
  class StateRef { constructor(name, cell) { this.name = name; this.cell = cell; } }

  // Errors and signals. RuntimeError stops the program; Thrown is a مشكلة that
  // «حاول» can catch; the signals carry أرجع, توقف and تابع out of value blocks.
  class RuntimeError extends Error {
    constructor(pos, msg) {
      super(`خطأ أثناء التشغيل [السطر ${pos[0]}، العمود ${pos[1]}]: ${msg}`);
      this.pos = pos; this.msg = msg;
    }
  }
  class Thrown extends Error {
    constructor(val, pos) { super("مشكلة لم تُعالَج"); this.val = val; this.pos = pos; }
  }
  class ReturnSig { constructor(val) { this.val = val; } }
  const BREAK = { signal: "break" }, CONTINUE = { signal: "continue" };

  function fail(pos, msg) { throw new RuntimeError(pos, msg); }
  function raise(pos, msg) { throw new Thrown(new Problem(msg), pos); }

  const isInt = (v) => typeof v === "bigint";
  const isFloat = (v) => typeof v === "number";
  const isFunc = (v) => typeof v === "function";

  // ---------------------------------------------------------------- type names and formatting

  function typeName(v) {
    if (v === null || v === undefined) return "عدم";
    switch (typeof v) {
      case "boolean": return "منطقي";
      case "bigint": return "صحيح";
      case "number": return "عشري";
      case "string": return "نص";
      case "function": return "دالة";
    }
    if (v instanceof List || v instanceof Range) return "قائمة";
    if (v instanceof DMap) return "قاموس";
    if (v instanceof Component) return "مكون";
    if (v instanceof StructType) return "نوع";
    if (v instanceof Struct) return v.type.name;
    if (v instanceof EnumType) return "تعداد";
    if (v instanceof Variant) return v.enum.name;
    if (v instanceof EnumVal) return v.variant.enum.name;
    if (v instanceof Problem) return "مشكلة";
    return "؟";
  }

  function formatFloat(f) {
    if (Number.isNaN(f)) return "ليس عدداً";
    if (f === Infinity) return "∞";
    if (f === -Infinity) return "-∞";
    let s = String(f);
    if (/e/i.test(s)) s = expandExponent(s);
    if (Object.is(f, -0)) s = "-0";
    if (!s.includes(".")) s += ".0";
    return s;
  }
  // 1.5e-7 → 0.00000015 and 1e21 → 1000000000000000000000, as Go's 'f' format writes them.
  function expandExponent(s) {
    let sign = "";
    if (s[0] === "-") { sign = "-"; s = s.slice(1); }
    const [mant, expStr] = s.toLowerCase().split("e");
    const exp = Number(expStr);
    let [int, frac = ""] = mant.split(".");
    let digits = int + frac;
    let point = int.length + exp;
    if (point <= 0) return sign + "0." + "0".repeat(-point) + digits;
    if (point >= digits.length) return sign + digits + "0".repeat(point - digits.length);
    return sign + digits.slice(0, point) + "." + digits.slice(point);
  }

  function format(v) {
    if (v === null || v === undefined) return "عدم";
    switch (typeof v) {
      case "boolean": return v ? "صح" : "خطأ";
      case "bigint": return v.toString();
      case "number": return formatFloat(v);
      case "string": return v;
      case "function": return v.$name ? "<دالة " + v.$name + ">" : "<دالة>";
    }
    if (v instanceof List) return "[" + v.elems.map(nested).join("، ") + "]";
    if (v instanceof Range) return "[" + v.values().map(nested).join("، ") + "]";
    if (v instanceof DMap) {
      if (v.keys.length === 0) return "[:]";
      return "[" + v.keys.map((k, i) => nested(k) + ": " + nested(v.vals[i])).join("، ") + "]";
    }
    if (v instanceof Component) return "<مكون " + v.name + ">";
    if (v instanceof StateRef) return format(v.cell.val);
    if (v instanceof StructType) return "<نوع " + v.name + ">";
    if (v instanceof EnumType) return "<تعداد " + v.name + ">";
    if (v instanceof Variant) return v.name;
    if (v instanceof Struct) return v.type.name + "(" + fields(v.type.fields, v.vals) + ")";
    if (v instanceof EnumVal) {
      if (v.vals.length === 0) return v.variant.name;
      return v.variant.name + "(" + fields(v.variant.fields, v.vals) + ")";
    }
    if (v instanceof Problem) return "مشكلة: " + v.message;
    return "؟";
  }
  const nested = (v) => (typeof v === "string" ? "«" + v + "»" : format(v));
  const fields = (fs, vals) => vals.map((v, i) => fs[i].name + ": " + nested(v)).join("، ");

  // ---------------------------------------------------------------- equality

  function equal(a, b) {
    if (a === null || a === undefined) return b === null || b === undefined;
    if (isInt(a)) return isInt(b) ? a === b : isFloat(b) ? Number(a) === b : false;
    if (isFloat(a)) return isInt(b) ? a === Number(b) : isFloat(b) ? a === b : false;
    if (typeof a === "boolean" || typeof a === "string") return a === b;
    if (a instanceof List) return b instanceof List && equalArrays(a.elems, b.elems);
    if (a instanceof Range) {
      if (b instanceof Range) return equalArrays(a.values(), b.values());
      if (b instanceof List) return equalArrays(a.values(), b.elems);
      return false;
    }
    if (a instanceof Struct) return b instanceof Struct && a.type === b.type && equalArrays(a.vals, b.vals);
    if (a instanceof EnumVal) return b instanceof EnumVal && a.variant === b.variant && equalArrays(a.vals, b.vals);
    if (a instanceof Problem) return b instanceof Problem && a.message === b.message;
    if (a instanceof DMap) {
      if (!(b instanceof DMap) || a.keys.length !== b.keys.length) return false;
      for (let i = 0; i < a.keys.length; i++) {
        const j = b.find(a.keys[i]);
        if (j < 0 || !equal(a.vals[i], b.vals[j])) return false;
      }
      return true;
    }
    return a === b;
  }
  function equalArrays(a, b) {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) if (!equal(a[i], b[i])) return false;
    return true;
  }

  // ---------------------------------------------------------------- operators

  const opNames = { "+": "+", "-": "-", "*": "*", "/": "/", "%": "%", "<": "<", ">": ">", "<=": "<=", ">=": ">=" };
  const toFloat = (v) => (isInt(v) ? Number(v) : isFloat(v) ? v : null);

  function bin(op, a, b, pos) {
    if (op === "==") return equal(a, b);
    if (op === "!=") return !equal(a, b);
    if (op === "+") {
      if (typeof a === "string") return a + format(b);
      if (typeof b === "string") return format(a) + b;
      if (a instanceof List && b instanceof List) return new List(a.elems.concat(b.elems));
    }
    if (typeof a === "string" && typeof b === "string") {
      switch (op) {
        case "<": return a < b;
        case ">": return a > b;
        case "<=": return a <= b;
        case ">=": return a >= b;
      }
    }
    if (isInt(a) && isInt(b)) {
      switch (op) {
        case "+": return BigInt.asIntN(64, a + b);
        case "-": return BigInt.asIntN(64, a - b);
        case "*": return BigInt.asIntN(64, a * b);
        case "/":
          if (b === 0n) raise(pos, "قسمة على صفر");
          return Number(a) / Number(b);
        case "%":
          if (b === 0n) raise(pos, "قسمة على صفر");
          return a % b;
        case "<": return a < b;
        case ">": return a > b;
        case "<=": return a <= b;
        case ">=": return a >= b;
      }
    }
    const fa = toFloat(a), fb = toFloat(b);
    if (fa !== null && fb !== null) {
      switch (op) {
        case "+": return fa + fb;
        case "-": return fa - fb;
        case "*": return fa * fb;
        case "/":
          if (fb === 0) raise(pos, "قسمة على صفر");
          return fa / fb;
        case "%":
          if (fb === 0) raise(pos, "قسمة على صفر");
          return fa - fb * Math.trunc(fa / fb);
        case "<": return fa < fb;
        case ">": return fa > fb;
        case "<=": return fa <= fb;
        case ">=": return fa >= fb;
      }
    }
    fail(pos, `لا يمكن تطبيق «${opNames[op]}» على «${typeName(a)}» و«${typeName(b)}»`);
  }

  function neg(v, pos) {
    if (isInt(v)) return BigInt.asIntN(64, -v);
    if (isFloat(v)) return -v;
    fail(pos, `لا يمكن تطبيق «-» على قيمة من النوع «${typeName(v)}»`);
  }

  function truthy(v, pos, what) {
    if (typeof v !== "boolean") fail(pos, `${what} يجب أن يكون «صح» أو «خطأ» لكنه من النوع «${typeName(v)}»`);
    return v;
  }

  // ---------------------------------------------------------------- types

  // A written type is a descriptor: {k: "n", name, opt} for a name, {k: "l"} a list,
  // {k: "m"} a map, {k: "f"} a function. Program types are looked up by name.
  const types = {}; // name → StructType or EnumType
  const variants = {}; // name → Variant

  const builtinTypeNames = new Set(["صحيح", "عشري", "نص", "منطقي", "مشكلة", "قائمة"]);
  function knownType(name) {
    if (builtinTypeNames.has(name) || variants[name]) return true;
    return !!types[name];
  }

  function typeString(t) {
    let s;
    switch (t.k) {
      case "l": s = "[" + (t.elem ? typeString(t.elem) : "") + "]"; break;
      case "m": s = "[" + (t.key ? typeString(t.key) : "") + ": " + (t.elem ? typeString(t.elem) : "") + "]"; break;
      case "f": s = t.str || "دالة"; break;
      default: s = t.name;
    }
    return t.opt ? s + "؟" : s;
  }

  function isType(v, t) {
    if (v === null || v === undefined) return !!t.opt;
    switch (t.k) {
      case "l": return v instanceof List || v instanceof Range;
      case "m": return v instanceof DMap;
      case "f": return isFunc(v);
    }
    switch (t.name) {
      case "صحيح": return isInt(v);
      case "عشري": return isFloat(v);
      case "نص": return typeof v === "string";
      case "منطقي": return typeof v === "boolean";
      case "مشكلة": return v instanceof Problem;
      case "قائمة": return v instanceof List || v instanceof Range;
      case "أي": case "قيمة": return true;
    }
    if (variants[t.name]) return v instanceof EnumVal && v.variant === variants[t.name];
    const tt = types[t.name];
    if (tt instanceof StructType) return v instanceof Struct && v.type === tt;
    if (tt instanceof EnumType) return v instanceof EnumVal && v.variant.enum === tt;
    return false;
  }

  function convert(v, t, pos, what) {
    if (!t) return v;
    if (v === null || v === undefined) {
      if (t.opt || (t.k === "n" && !knownType(t.name))) return null;
      fail(pos, `${what} لا يقبل «عدم»؛ استخدم النوع «${typeString(t)}؟» إذا كانت القيمة اختيارية`);
    }
    if (t.k === "n") {
      if (t.name === "عشري" && isInt(v)) return Number(v);
      if (!knownType(t.name)) return v; // contracts and generic names
    }
    if (!isType(v, t)) fail(pos, `${what}: توقعت قيمة من النوع «${typeString(t)}» لكن وجدت «${typeName(v)}»`);
    return v;
  }

  function parseNumber(s) {
    s = s.replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x660))
      .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x6f0))
      .replace(/٫/g, ".").replace(/[٬_]/g, "").trim();
    if (s === "" || !/^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/.test(s)) return null;
    return Number(s);
  }

  function cast(v, t, pos) {
    if (v === null || v === undefined) {
      if (t.opt) return null;
      fail(pos, `لا يمكن تحويل «عدم» إلى «${typeString(t)}»`);
    }
    switch (t.name) {
      case "نص": return format(v);
      case "عشري":
        if (isInt(v)) return Number(v);
        if (isFloat(v)) return v;
        if (typeof v === "string") {
          const f = parseNumber(v);
          if (f !== null) return f;
          raise(pos, `لا يمكن تحويل «${v}» إلى عدد عشري`);
        }
        break;
      case "صحيح":
        if (isInt(v)) return v;
        if (isFloat(v)) return Number.isFinite(v) ? BigInt(Math.trunc(v)) : -9223372036854775808n;
        if (typeof v === "string") {
          const f = parseNumber(v);
          if (f !== null && Number.isInteger(f)) return BigInt(f);
          raise(pos, `لا يمكن تحويل «${v}» إلى عدد صحيح`);
        }
        break;
      case "منطقي":
        if (typeof v === "boolean") return v;
        break;
      default:
        if (isType(v, t)) return v;
    }
    fail(pos, `لا يمكن تحويل قيمة من النوع «${typeName(v)}» إلى «${typeString(t)}»`);
  }

  function intValue(v, pos, what) {
    if (!isInt(v)) fail(pos, `${what} يجب أن تكون عدداً صحيحاً لكنها من النوع «${typeName(v)}»`);
    return v;
  }

  // ---------------------------------------------------------------- functions and calls

  // A Ḍād function is a JS function (args, named, pos): args is an array, named
  // an array of [name, value] pairs, pos the call's [line, column].
  function fn(name, impl) { impl.$name = name; return impl; }

  // bind matches arguments to parameters, as the interpreter's bindArgs does.
  // params: [{name, type, def}] where def(...earlierValues) gives a default.
  function bind(params, args, named, pos, what) {
    const vals = new Array(params.length), set = new Array(params.length).fill(false);
    if (args.length > params.length) fail(pos, `${what} تأخذ ${params.length} من المعاملات لكن أُعطيت ${args.length}`);
    for (let i = 0; i < args.length; i++) { vals[i] = args[i]; set[i] = true; }
    for (const [name, val] of named || []) {
      const idx = params.findIndex((p) => p.name === name);
      if (idx < 0) fail(pos, `${what} لا تحتوي معاملاً باسم «${name}»`);
      if (set[idx]) fail(pos, `المعامل «${name}» أُعطي مرتين`);
      vals[idx] = val; set[idx] = true;
    }
    for (let i = 0; i < params.length; i++) {
      const p = params[i];
      if (!set[i]) {
        if (!p.def) fail(pos, `ينقص المعامل «${p.name}» عند استدعاء ${what}`);
        vals[i] = p.def(...vals.slice(0, i));
      }
      vals[i] = convert(vals[i], p.type, pos, `المعامل «${p.name}» في ${what}`);
    }
    return vals;
  }

  let depth = 0;
  function call(f, args, named, pos, name) {
    if (++depth > 2000) { depth = 0; fail(pos, "تجاوز البرنامج الحد الأقصى لعمق الاستدعاءات — هل توجد دالة تستدعي نفسها بلا نهاية؟"); }
    try {
      if (isFunc(f)) return f(args, named, pos);
      if (f instanceof StructType) return new Struct(f, bind(f.fields, args, named, pos, "«" + f.name + "»"));
      if (f instanceof Variant) return new EnumVal(f, bind(f.fields, args, named, pos, "«" + f.name + "»"));
      if (f instanceof EnumVal && args.length === 0 && (!named || named.length === 0)) return f;
      if (f instanceof Component) return renderComponent(f, args, named, pos);
      if (f === null || f === undefined) fail(pos, "لا يمكن استدعاء «عدم»");
      fail(pos, `«${name || format(f)}» ليس دالة، بل قيمة من النوع «${typeName(f)}»`);
    } finally { depth--; }
  }

  // lam makes a lambda: «{ هو * 2 }» or «{ أ، ب => أ + ب }».
  function lam(n, impl) {
    const f = (args, named, pos) => {
      if (named && named.length) fail(pos, "الدوال المجهولة لا تقبل معاملات مسماة");
      if (n > 0 && args.length !== n) fail(pos, `الدالة المجهولة تتوقع ${n} من المعاملات لكن أُعطيت ${args.length}`);
      try { return impl(...args); } catch (e) { if (e instanceof ReturnSig) return e.val; throw e; }
    };
    return f;
  }

  // ---------------------------------------------------------------- members and methods

  const ext = {}; // type key → name → method(self, args, named, pos)
  function typeKey(v) { return v instanceof List || v instanceof Range ? "قائمة" : typeName(v); }

  function bindMethod(m, self) {
    return fn(m.$name, (args, named, pos) => m(self, args, named, pos));
  }

  function tryMember(r, name) {
    if (r instanceof Struct) {
      const i = r.type.fields.findIndex((f) => f.name === name);
      if (i >= 0) return [r.vals[i], true];
      if (r.type.methods[name]) return [bindMethod(r.type.methods[name], r), true];
    } else if (r instanceof EnumVal) {
      const i = r.variant.fields.findIndex((f) => f.name === name);
      if (i >= 0) return [r.vals[i], true];
    } else if (r instanceof EnumType) {
      for (const v of r.variants) if (v.name === name) return [v.fields.length === 0 ? new EnumVal(v, []) : v, true];
    } else if (r instanceof Problem) {
      if (name === "رسالة") return [r.message, true];
    }
    if (r !== null && r !== undefined) {
      const m = (ext[typeKey(r)] || {})[name];
      if (m) {
        const [v, ok] = builtinMember(r, name);
        if (ok) return [v, ok];
        return [bindMethod(m, r), true];
      }
    }
    return builtinMember(r, name);
  }

  function builtinMember(r, name) {
    if (typeof r === "string") {
      if (name === "طول" || name === "عدد") return [BigInt([...r].length), true];
      if (name === "فارغ") return [r === "", true];
    } else if (r instanceof List) {
      const p = listProperty(r.elems, name);
      if (p) return p;
    } else if (r instanceof Range) {
      const p = listProperty(r.values(), name);
      if (p) return p;
    } else if (r instanceof DMap) {
      switch (name) {
        case "طول": case "عدد": return [BigInt(r.keys.length), true];
        case "فارغ": return [r.keys.length === 0, true];
        case "مفاتيح": return [new List(r.keys.slice()), true];
        case "قيم": return [new List(r.vals.slice()), true];
      }
    }
    return [null, false];
  }

  function listProperty(elems, name) {
    switch (name) {
      case "طول": case "عدد": return [BigInt(elems.length), true];
      case "فارغ": return [elems.length === 0, true];
      case "أول": return [elems.length ? elems[0] : null, true];
      case "آخر": return [elems.length ? elems[elems.length - 1] : null, true];
    }
    return null;
  }

  function member(r, name, pos, safe) {
    if ((r === null || r === undefined) && safe) return null;
    const [v, ok] = tryMember(r, name);
    if (ok) return v;
    if (r === null || r === undefined) fail(pos, `لا يمكن قراءة «${name}» من «عدم»؛ استخدم «؟.» أو «؟؟»`);
    fail(pos, `القيمة من النوع «${typeName(r)}» لا تحتوي «${name}»`);
  }

  function method(r, name, args, named, pos, safe) {
    if ((r === null || r === undefined) && safe) return null;
    if (r instanceof Struct) {
      const i = r.type.fields.findIndex((f) => f.name === name);
      if (i >= 0) return call(r.vals[i], args, named, pos, name);
      if (r.type.methods[name]) return call(bindMethod(r.type.methods[name], r), args, named, pos, name);
      if (name === "نسخة") return new Struct(r.type, copyWith(r.type.fields, r.vals, args, named, pos));
      if (r.type.table) {
        const v = recordMethod(r, name, args, named, pos);
        if (v !== NOT_FOUND) return v;
      }
    } else if (r instanceof StructType) {
      if (r.table) {
        const v = tableMethod(r, name, args, named, pos);
        if (v !== NOT_FOUND) return v;
      }
    } else if (r instanceof EnumVal) {
      if (name === "نسخة" && r.vals.length > 0) return new EnumVal(r.variant, copyWith(r.variant.fields, r.vals, args, named, pos));
    }
    if (r !== null && r !== undefined) {
      const m = (ext[typeKey(r)] || {})[name];
      if (m) return call(bindMethod(m, r), args, named, pos, name);
    }
    const v = builtinMethod(r, name, args, named || [], pos);
    if (v !== NOT_FOUND) return v;
    if (r === null || r === undefined) fail(pos, `لا يمكن استدعاء «${name}» على «عدم»؛ استخدم «؟.» أو «؟؟»`);
    return call(member(r, name, pos), args, named, pos, name);
  }

  // Inside methods and extensions, an unqualified name is a member of هذا.
  function selfGet(self, name, pos) {
    const [v, ok] = tryMember(self, name);
    if (ok) return v;
    fail(pos, `«${name}» غير معرّف`);
  }

  function copyWith(fs, vals, args, named, pos) {
    if (args.length > 0) fail(pos, `«نسخة» تقبل معاملات مسماة فقط، مثل: نسخة(${fs[0].name}: …)`);
    const out = vals.slice();
    for (const [name, val] of named || []) {
      const i = fs.findIndex((f) => f.name === name);
      if (i < 0) fail(pos, `لا يوجد حقل باسم «${name}»`);
      out[i] = convert(val, fs[i].type, pos, `الحقل «${name}»`);
    }
    return out;
  }

  // ---------------------------------------------------------------- built-in methods

  const NOT_FOUND = {};

  function argCount(args, named, n, name, pos) {
    if (args.length !== n || named.length > 0)
      fail(pos, `«${name}» تأخذ ${n} من المعاملات لكن أُعطيت ${args.length + named.length}`);
  }
  function fnArg(args, i, name, pos) {
    if (i >= args.length) fail(pos, `«${name}» تحتاج دالة، مثل: ${name} { هو … }`);
    if (!isFunc(args[i])) fail(pos, `«${name}» تحتاج دالة لكن أُعطيت قيمة من النوع «${typeName(args[i])}»`);
    return args[i];
  }
  const apply = (f, pos, ...args) => call(f, args, [], pos);

  function builtinMethod(r, name, args, named, pos) {
    if (r instanceof List) return listMethod(r.elems, r, name, args, named, pos);
    if (r instanceof Range) return listMethod(r.values(), null, name, args, named, pos);
    if (typeof r === "string") return stringMethod(r, name, args, named, pos);
    if (isInt(r) || isFloat(r)) return numberMethod(r, name, args, named, pos);
    if (r instanceof DMap) {
      switch (name) {
        case "يحتوي": argCount(args, named, 1, name, pos); return r.find(args[0]) >= 0;
        case "احذف": {
          argCount(args, named, 1, name, pos);
          const j = r.find(args[0]);
          if (j >= 0) { r.keys.splice(j, 1); r.vals.splice(j, 1); }
          return null;
        }
      }
    }
    return NOT_FOUND;
  }

  function listMethod(elems, list, name, args, named, pos) {
    switch (name) {
      case "طول": case "عدد": {
        if (args.length === 0) return BigInt(elems.length);
        const f = fnArg(args, 0, name, pos);
        let n = 0n;
        for (const x of elems) if (truthy(apply(f, pos, x), pos, "نتيجة «عدد»")) n++;
        return n;
      }
      case "أول": case "آخر": case "فارغ": return listProperty(elems, name)[0];
      case "رشح": {
        const f = fnArg(args, 0, name, pos);
        return new List(elems.filter((x) => truthy(apply(f, pos, x), pos, "نتيجة «رشح»")));
      }
      case "حول": {
        const f = fnArg(args, 0, name, pos);
        return new List(elems.map((x) => apply(f, pos, x)));
      }
      case "لكل": {
        const f = fnArg(args, 0, name, pos);
        for (const x of elems) apply(f, pos, x);
        return null;
      }
      case "أي": case "كل": {
        const f = fnArg(args, 0, name, pos);
        for (const x of elems) {
          const ok = truthy(apply(f, pos, x), pos, "نتيجة «" + name + "»");
          if (name === "أي" && ok) return true;
          if (name === "كل" && !ok) return false;
        }
        return name === "كل";
      }
      case "اختزل": {
        let acc, f, rest = elems;
        if (args.length === 2) { acc = args[0]; f = fnArg(args, 1, name, pos); }
        else if (args.length === 1) {
          f = fnArg(args, 0, name, pos);
          if (elems.length === 0) raise(pos, "لا يمكن «اختزل» قائمة فارغة بلا قيمة بداية");
          acc = elems[0]; rest = elems.slice(1);
        } else fail(pos, "«اختزل» تأخذ قيمة بداية ودالة، مثل: اختزل(0) { أ، ب => أ + ب }");
        for (const x of rest) acc = apply(f, pos, acc, x);
        return acc;
      }
      case "ضم": {
        let sep = "";
        if (args.length > 0) {
          if (typeof args[0] !== "string") fail(pos, "فاصل «ضم» يجب أن يكون نصاً");
          sep = args[0];
        }
        return elems.map(format).join(sep);
      }
      case "يحتوي":
        argCount(args, named, 1, name, pos);
        return elems.some((x) => equal(x, args[0]));
      case "مجموع": {
        let si = 0n, sf = 0, isF = false;
        for (const x of elems) {
          if (isInt(x)) { si = BigInt.asIntN(64, si + x); sf += Number(x); }
          else if (isFloat(x)) { isF = true; sf += x; }
          else fail(pos, "«مجموع» تعمل على قوائم الأعداد فقط");
        }
        return isF ? sf : si;
      }
      case "عكس": return new List(elems.slice().reverse());
      case "رتب": {
        let bad = false;
        const less = (a, b) => {
          if (typeof a === "string" && typeof b === "string") return a < b;
          const fa = toFloat(a), fb = toFloat(b);
          if (fa === null || fb === null) { bad = true; return false; }
          return fa < fb;
        };
        const out = elems.slice().sort((a, b) => (less(a, b) ? -1 : less(b, a) ? 1 : 0));
        if (bad) fail(pos, "«رتب» تعمل على قوائم الأعداد أو النصوص فقط");
        return new List(out);
      }
      case "أضف":
        if (!list) fail(pos, "لا يمكن الإضافة إلى نطاق؛ حوّله إلى قائمة أولاً");
        list.elems.push(...args);
        return null;
    }
    return NOT_FOUND;
  }

  function stringMethod(s, name, args, named, pos) {
    const arg = (i) => {
      if (i >= args.length) fail(pos, `«${name}» تحتاج معاملاً نصياً`);
      if (typeof args[i] !== "string") fail(pos, `معامل «${name}» يجب أن يكون نصاً`);
      return args[i];
    };
    switch (name) {
      case "طول": case "عدد": return BigInt([...s].length);
      case "قص": return s.trim();
      case "يحتوي": return s.includes(arg(0));
      case "يبدأ_ب": case "يبدأ": return s.startsWith(arg(0));
      case "ينتهي_ب": case "ينتهي": return s.endsWith(arg(0));
      case "استبدل": { const a = arg(0), b = arg(1); return a === "" ? [...s].map((c) => b + c).join("") + b : s.split(a).join(b); }
      case "قسم": { const sep = arg(0); return new List(sep === "" ? [...s] : s.split(sep)); }
      case "أحرف": return new List([...s]);
      case "كبير": return s.toUpperCase();
      case "صغير": return s.toLowerCase();
      case "كرر": {
        if (args.length !== 1) fail(pos, "«كرر» تأخذ عدد المرات");
        const n = intValue(args[0], pos, "عدد مرات «كرر»");
        if (n < 0n) raise(pos, "عدد مرات «كرر» لا يمكن أن يكون سالباً");
        return s.repeat(Number(n));
      }
    }
    return NOT_FOUND;
  }

  // Go rounds halves away from zero.
  const goRound = (f) => Math.sign(f) * Math.round(Math.abs(f));

  function numberMethod(v, name, args, named, pos) {
    const f = toFloat(v);
    switch (name) {
      case "مطلق": return isInt(v) ? (v < 0n ? -v : v) : Math.abs(f);
      case "تقريب":
        if (args.length === 1) {
          const d = intValue(args[0], pos, "عدد الخانات");
          const p = Math.pow(10, Number(d));
          return goRound(f * p) / p;
        }
        return BigInt(goRound(f));
      case "أرضية": return BigInt(Math.floor(f));
      case "سقف": return BigInt(Math.ceil(f));
    }
    return NOT_FOUND;
  }

  // ---------------------------------------------------------------- built-in functions

  let out = (line) => console.log(line);
  const b = {};
  const def = (name, impl) => { b[name] = fn(name, impl); };
  def("اطبع", (args, named, pos) => {
    if (named && named.length) fail(pos, "«اطبع» لا تقبل معاملات مسماة");
    out(args.map(format).join(" "));
    return null;
  });
  def("مشكلة", (args, named, pos) => { argCount(args, named || [], 1, "مشكلة", pos); return new Problem(format(args[0])); });
  const num = (name, args, named, pos) => {
    argCount(args, named || [], 1, name, pos);
    const f = toFloat(args[0]);
    if (f === null) fail(pos, `«${name}» تحتاج عدداً لكن أُعطيت «${typeName(args[0])}»`);
    return f;
  };
  def("جذر", (args, named, pos) => {
    const f = num("جذر", args, named, pos);
    if (f < 0) raise(pos, "لا يوجد جذر حقيقي لعدد سالب");
    return Math.sqrt(f);
  });
  def("مطلق", (args, named, pos) => {
    if (args.length !== 1) argCount(args, named || [], 1, "مطلق", pos);
    if (isInt(args[0])) return args[0] < 0n ? -args[0] : args[0];
    return Math.abs(num("مطلق", args, named, pos));
  });
  const extreme = (name, wantLess) => def(name, (args, named, pos) => {
    let xs = args;
    if (xs.length === 1) {
      if (xs[0] instanceof List) xs = xs[0].elems;
      else if (xs[0] instanceof Range) xs = xs[0].values();
    }
    if (xs.length === 0) raise(pos, `«${name}» تحتاج قيمة واحدة على الأقل`);
    let best = xs[0];
    for (const a of xs.slice(1)) if (bin("<", a, best, pos) === wantLess) best = a;
    return best;
  });
  extreme("أدنى", true);
  extreme("أقصى", false);

  // ---------------------------------------------------------------- expressions the compiler uses

  function str(parts) {
    let s = "";
    for (const p of parts) s += typeof p === "string" ? p : format(p.v);
    return s;
  }

  function index(c, i, pos) {
    const at = (n) => {
      const k = intValue(i, pos, "الفهرس");
      if (k < 0n || k >= BigInt(n)) raise(pos, `الفهرس ${k} خارج الحدود (الطول ${n})`);
      return Number(k);
    };
    if (c instanceof List) return c.elems[at(c.elems.length)];
    if (c instanceof Range) { const vs = c.values(); return vs[at(vs.length)]; }
    if (typeof c === "string") { const rs = [...c]; return rs[at(rs.length)]; }
    if (c instanceof DMap) { const j = c.find(i); return j >= 0 ? c.vals[j] : null; }
    if (c === null || c === undefined) fail(pos, "لا يمكن الفهرسة في «عدم»");
    fail(pos, `لا يمكن الفهرسة في قيمة من النوع «${typeName(c)}»`);
  }

  // setIndex runs «ق[ف] = …» or «ق[ف] += …»; update(old) gives the new value.
  function setIndex(c, i, update, pos) {
    if (c instanceof List) {
      const k = intValue(i, pos, "الفهرس");
      if (k < 0n || k >= BigInt(c.elems.length)) raise(pos, `الفهرس ${k} خارج الحدود (الطول ${c.elems.length})`);
      c.elems[Number(k)] = update(c.elems[Number(k)]);
      return;
    }
    if (c instanceof DMap) {
      const j = c.find(i);
      const nv = update(j >= 0 ? c.vals[j] : null);
      if (j >= 0) c.vals[j] = nv; else { c.keys.push(i); c.vals.push(nv); }
      return;
    }
    fail(pos, `لا يمكن الإسناد بالفهرس إلى قيمة من النوع «${typeName(c)}»`);
  }

  // A عشري variable stays عشري when given a whole number.
  const keepFloat = (old, nv) => (isFloat(old) && isInt(nv) ? Number(nv) : nv);

  function map(pairs) {
    const m = new DMap();
    for (const [k, v] of pairs) {
      const j = m.find(k);
      if (j >= 0) m.vals[j] = v; else { m.keys.push(k); m.vals.push(v); }
    }
    return m;
  }

  function range(from, to, inclusive, step, pos) {
    const r = new Range(intValue(from[0], from[1], "بداية النطاق"), intValue(to[0], to[1], "نهاية النطاق"), 1n, inclusive);
    if (step) {
      r.step = intValue(step[0], step[1], "الخطوة");
      if (r.step === 0n) fail(step[1], "الخطوة في «بخطوة» لا يمكن أن تكون صفراً");
    }
    return r;
  }

  // iter gives what «لكل» goes through.
  function iter(v, pos, forPos) {
    if (v instanceof Range) {
      if (v.step === 0n) fail(forPos, "الخطوة في «بخطوة» لا يمكن أن تكون صفراً");
      return v.values();
    }
    if (v instanceof List) return v.elems.slice();
    if (typeof v === "string") return [...v];
    if (v instanceof DMap) return v.keys.slice();
    fail(pos, `لا يمكن المرور على قيمة من النوع «${typeName(v)}» بـ«لكل»`);
  }

  function doThrow(v, pos) { throw throwValue(v, pos); }
  function undef(pos, name) { fail(pos, `«${name}» غير معرّف`); }
  function variant(en, name, fields) {
    const v = new Variant(en, name, fields);
    en.variants.push(v);
    variants[name] = v;
    return fields.length === 0 ? new EnumVal(v, []) : v;
  }
  function extend(key, name, m) { (ext[key] = ext[key] || {})[name] = m; }

  function throwValue(v, pos) {
    if (typeof v === "string") v = new Problem(v);
    return new Thrown(v, pos);
  }

  // catchable returns the value of a مشكلة raised inside «حاول», or rethrows anything else.
  function caught(e) {
    if (e instanceof Thrown) return e.val;
    throw e;
  }

  function noMatch(v, pos) {
    fail(pos, `لا توجد حالة في «طابق» تطابق القيمة «${format(v)}»`);
  }

  function patternFields(v, variant, n, pos) {
    if (n > v.vals.length) fail(pos, `الحالة «${variant.name}» تحتوي ${v.vals.length} حقول لكن النمط يذكر ${n}`);
  }

  // ---------------------------------------------------------------- tables (kept in memory, or in the browser)

  const store = {
    tables: {},
    persist: null, // set in the browser to save to localStorage
    table(name) { return (this.tables[name] = this.tables[name] || { next: 1, rows: {} }); },
  };
  function loadRows(t) {
    const tab = store.table(t.name);
    return Object.keys(tab.rows).map(Number).sort((a, b) => a - b).map((id) => recordFromRow(t, tab.rows[id]));
  }
  function recordFromRow(t, row) {
    return new Struct(t, t.fields.map((f) => {
      if (f.name in row && row[f.name] !== null) return convert(fromData(row[f.name], f.type), f.type, [0, 0], "الحقل «" + f.name + "»");
      if (f.name === "رقم") return BigInt(row["رقم"]);
      return f.def ? f.def() : null;
    }));
  }
  function saveRecord(r, pos) {
    const t = r.type, tab = store.table(t.name);
    const row = {};
    t.fields.slice(0, -1).forEach((f, i) => { row[f.name] = toData(r.vals[i]); });
    const idx = t.fields.length - 1;
    let id = r.vals[idx];
    if (id === 0n) { id = BigInt(tab.next++); }
    else if (!tab.rows[Number(id)]) fail(pos, `لا يوجد سجل برقم ${id} في «${t.name}»؛ ربما حُذف`);
    row["رقم"] = Number(id);
    tab.rows[Number(id)] = row;
    if (store.persist) store.persist();
    const vals = r.vals.slice();
    vals[idx] = id;
    return new Struct(t, vals);
  }
  function tableMethod(t, name, args, named, pos) {
    switch (name) {
      case "أنشئ": return saveRecord(call(t, args, named, pos), pos);
      case "الكل": return new List(loadRows(t));
      case "عدد": return BigInt(loadRows(t).length);
      case "حيث": {
        const f = args[0];
        if (args.length !== 1) fail(pos, `«حيث» تأخذ دالة، مثل: حيث { هو.منجزة }`);
        return new List(loadRows(t).filter((r) => truthy(apply(f, pos, r), pos, "شرط «حيث»")));
      }
      case "بالرقم": {
        if (args.length !== 1) fail(pos, `«بالرقم» تأخذ رقم السجل، مثل: ${t.name}.بالرقم(1)`);
        if (!isInt(args[0])) fail(pos, `«بالرقم» تأخذ عدداً صحيحاً، لكن وجدت «${typeName(args[0])}»`);
        return loadRows(t).find((r) => r.vals[t.fields.length - 1] === args[0]) || null;
      }
    }
    return NOT_FOUND;
  }
  function recordMethod(r, name, args, named, pos) {
    switch (name) {
      case "احفظ": return saveRecord(r, pos);
      case "احذف": {
        const id = r.vals[r.type.fields.length - 1];
        if (id === 0n) fail(pos, "لا يمكن حذف سجل لم يُحفظ بعد");
        const tab = store.table(r.type.name);
        const ok = !!tab.rows[Number(id)];
        delete tab.rows[Number(id)];
        if (store.persist) store.persist();
        return ok;
      }
    }
    return NOT_FOUND;
  }
  function toData(v) {
    if (isInt(v)) return Number(v);
    if (v instanceof List) return v.elems.map(toData);
    if (v instanceof Range) return v.values().map(toData);
    if (v instanceof DMap) { const o = {}; v.keys.forEach((k, i) => { o[typeof k === "string" ? k : format(k)] = toData(v.vals[i]); }); return o; }
    if (v instanceof Struct) { const o = {}; v.type.fields.forEach((f, i) => { o[f.name] = toData(v.vals[i]); }); return o; }
    if (v instanceof EnumVal) {
      if (v.vals.length === 0) return v.variant.name;
      const o = { _: v.variant.name };
      v.variant.fields.forEach((f, i) => { o[f.name] = toData(v.vals[i]); });
      return o;
    }
    if (v instanceof Problem) return { "مشكلة": v.message };
    return v;
  }
  function fromData(v, t) {
    if (typeof v === "number") return Number.isInteger(v) && !(t && t.name === "عشري") ? BigInt(v) : v;
    if (Array.isArray(v)) return new List(v.map((x) => fromData(x, t && t.k === "l" ? t.elem : null)));
    if (v && typeof v === "object") {
      const st = t && t.k === "n" ? types[t.name] : null;
      if (st instanceof StructType) return new Struct(st, st.fields.map((f) => (f.name in v ? convert(fromData(v[f.name], f.type), f.type, [0, 0], "الحقل «" + f.name + "»") : f.def ? f.def() : null)));
      return new DMap(Object.keys(v).sort(), Object.keys(v).sort().map((k) => fromData(v[k], null)));
    }
    return v;
  }

  // ---------------------------------------------------------------- the interface

  // Elements and components render to a tree of nodes, written as HTML; the
  // page's script (ui.js) sends clicks and typing back through window.dadSend.
  const elements = {};
  // Every element also takes «صنف», a class for a program's own stylesheet.
  const boxOptions = ["محاذاة", "مسافة", "حشو", "خلفية", "عرض", "صنف"];
  const elementOptions = {
    "عمود": boxOptions, "صف": boxOptions, "بطاقة": boxOptions,
    "عنوان": ["مستوى", "لون", "محاذاة", "صنف"], "فقرة": ["لون", "محاذاة", "حجم", "صنف"],
    "زر": ["معطل", "نمط", "حجم", "ملء", "أيقونة", "صنف"], "حقل": ["تلميح", "صيغة", "معطل", "أدنى", "أقصى", "صنف"],
    "مربع_اختيار": ["نص", "معطل", "صنف"], "صورة": ["وصف", "عرض", "صنف"], "رابط": ["نمط", "حجم", "أيقونة", "صنف"],
    "فاصل": ["صنف"], "قائمة_منسدلة": ["تلميح", "معطل", "صنف"], "اختيار": ["معطل", "صنف"],
    "منطقة_نص": ["تلميح", "أسطر", "معطل", "صنف"], "جدول_عرض": ["أعمدة", "فارغ", "صنف"],
    "قائمة_عناصر": ["مرقمة", "فارغ", "صنف"], "ألسنة": ["صنف"], "لسان": ["أيقونة", "صنف"],
    "نافذة": ["عنوان", "صنف"], "تنبيه": ["نمط", "صنف"], "أيقونة": ["وصف", "حجم", "لون", "صنف"],
  };
  const optionValues = {
    "نمط": ["أساسي", "ثانوي", "خفيف", "خطر"], "حجم": ["صغير", "عادي", "كبير"],
    "محاذاة": ["بداية", "وسط", "نهاية"], "صيغة": ["نص", "رقم", "كلمة_سر", "بريد", "تاريخ", "وقت"],
  };
  const alertStyles = ["معلومة", "نجاح", "تحذير", "خطأ"];
  const optionWords = (element, opt) => (element === "تنبيه" && opt === "نمط" ? alertStyles : optionValues[opt]);
  // icons:begin — generated by tools/icons/gen.py from Lucide (ISC licence)
  const icons = {
    "أسفل": "<path d=\"m6 9 6 6 6-6\"/>",
    "أضف": "<path d=\"M5 12h14\"/><path d=\"M12 5v14\"/>",
    "أعلى": "<path d=\"m18 15-6-6-6 6\"/>",
    "إرسال": "<path d=\"M14.536 21.686a.5.5 0 0 0 .937-.024l6.5-19a.496.496 0 0 0-.635-.635l-19 6.5a.5.5 0 0 0-.024.937l7.93 3.18a2 2 0 0 1 1.112 1.11z\"/><path d=\"m21.854 2.147-10.94 10.939\"/>",
    "إعدادات": "<path d=\"M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915\"/><circle cx=\"12\" cy=\"12\" r=\"3\"/>",
    "إغلاق": "<path d=\"M18 6 6 18\"/><path d=\"m6 6 12 12\"/>",
    "التالي": "<path d=\"m15 18-6-6 6-6\"/>",
    "السابق": "<path d=\"m9 18 6-6-6-6\"/>",
    "بحث": "<path d=\"m21 21-4.34-4.34\"/><circle cx=\"11\" cy=\"11\" r=\"8\"/>",
    "بريد": "<path d=\"m22 7-8.991 5.727a2 2 0 0 1-2.009 0L2 7\"/><rect x=\"2\" y=\"4\" width=\"20\" height=\"16\" rx=\"2\"/>",
    "تحديث": "<path d=\"M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8\"/><path d=\"M21 3v5h-5\"/><path d=\"M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16\"/><path d=\"M8 16H3v5\"/>",
    "تحذير": "<path d=\"m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3\"/><path d=\"M12 9v4\"/><path d=\"M12 17h.01\"/>",
    "تعديل": "<path d=\"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z\"/><path d=\"m15 5 4 4\"/>",
    "تقويم": "<path d=\"M8 2v3\"/><path d=\"M16 2v3\"/><rect x=\"3\" y=\"3\" width=\"18\" height=\"18\" rx=\"2\"/><path d=\"M3 9h18\"/>",
    "تم": "<path d=\"M20 6 9 17l-5-5\"/>",
    "تنزيل": "<path d=\"M12 15V3\"/><path d=\"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4\"/><path d=\"m7 10 5 5 5-5\"/>",
    "جرس": "<path d=\"M10.268 21a2 2 0 0 0 3.464 0\"/><path d=\"M3.262 15.326A1 1 0 0 0 4 17h16a1 1 0 0 0 .74-1.673C19.41 13.956 18 12.499 18 8A6 6 0 0 0 6 8c0 4.499-1.411 5.956-2.738 7.326\"/>",
    "حذف": "<path d=\"M10 11v6\"/><path d=\"M14 11v6\"/><path d=\"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6\"/><path d=\"M3 6h18\"/><path d=\"M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2\"/>",
    "حفظ": "<path d=\"M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z\"/><path d=\"M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7\"/><path d=\"M7 3v4a1 1 0 0 0 1 1h7\"/>",
    "خروج": "<path d=\"m16 17 5-5-5-5\"/><path d=\"M21 12H9\"/><path d=\"M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4\"/>",
    "خطر": "<path d=\"M12 16h.01\"/><path d=\"M12 8v4\"/><path d=\"M15.312 2a2 2 0 0 1 1.414.586l4.688 4.688A2 2 0 0 1 22 8.688v6.624a2 2 0 0 1-.586 1.414l-4.688 4.688a2 2 0 0 1-1.414.586H8.688a2 2 0 0 1-1.414-.586l-4.688-4.688A2 2 0 0 1 2 15.312V8.688a2 2 0 0 1 .586-1.414l4.688-4.688A2 2 0 0 1 8.688 2z\"/>",
    "دخول": "<path d=\"m10 17 5-5-5-5\"/><path d=\"M15 12H3\"/><path d=\"M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4\"/>",
    "رئيسية": "<path d=\"M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8\"/><path d=\"M3 10a2 2 0 0 1 .709-1.528l7-6a2 2 0 0 1 2.582 0l7 6A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z\"/>",
    "رابط": "<path d=\"M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71\"/><path d=\"M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71\"/>",
    "رسم_بياني": "<path d=\"M3 3v16a2 2 0 0 0 2 2h16\"/><path d=\"M18 17V9\"/><path d=\"M13 17V5\"/><path d=\"M8 17v-3\"/>",
    "رفع": "<path d=\"M12 3v12\"/><path d=\"m17 8-5-5-5 5\"/><path d=\"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4\"/>",
    "ساعة": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"M12 6v6l4 2\"/>",
    "سلة": "<path d=\"m2.05 2.05 1.099-.028a1 1 0 0 1 1.008.815l2.69 14.347A1 1 0 0 0 7.83 18H18\"/><path d=\"M4.563 5h16.435a1 1 0 0 1 .981 1.204l-1.026 6.226A2 2 0 0 1 18.962 14H6.25\"/><circle cx=\"18\" cy=\"20\" r=\"2\"/><circle cx=\"8\" cy=\"20\" r=\"2\"/>",
    "شمس": "<circle cx=\"12\" cy=\"12\" r=\"4\"/><path d=\"M12 2v2\"/><path d=\"M12 20v2\"/><path d=\"m4.93 4.93 1.41 1.41\"/><path d=\"m17.66 17.66 1.41 1.41\"/><path d=\"M2 12h2\"/><path d=\"M20 12h2\"/><path d=\"m6.34 17.66-1.41 1.41\"/><path d=\"m19.07 4.93-1.41 1.41\"/>",
    "صورة": "<rect width=\"18\" height=\"18\" x=\"3\" y=\"3\" rx=\"2\" ry=\"2\"/><circle cx=\"9\" cy=\"9\" r=\"2\"/><path d=\"m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21\"/>",
    "طباعة": "<path d=\"M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2\"/><path d=\"M6 9V3a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v6\"/><rect x=\"6\" y=\"14\" width=\"12\" height=\"8\" rx=\"1\"/>",
    "عالم": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20\"/><path d=\"M2 12h20\"/>",
    "عين": "<path d=\"M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0\"/><circle cx=\"12\" cy=\"12\" r=\"3\"/>",
    "قائمة": "<path d=\"M4 5h16\"/><path d=\"M4 12h16\"/><path d=\"M4 19h16\"/>",
    "قفل": "<rect width=\"18\" height=\"11\" x=\"3\" y=\"11\" rx=\"2\" ry=\"2\"/><path d=\"M7 11V7a5 5 0 0 1 10 0v4\"/>",
    "قلب": "<path d=\"M2 9.5a5.5 5.5 0 0 1 9.591-3.676.56.56 0 0 0 .818 0A5.49 5.49 0 0 1 22 9.5c0 2.29-1.5 4-3 5.5l-5.492 5.313a2 2 0 0 1-3 .019L5 15c-1.5-1.5-3-3.2-3-5.5\"/>",
    "قمر": "<path d=\"M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401\"/>",
    "كتاب": "<path d=\"M12 5v16\"/><path d=\"M20.001 19A2 2 0 0022 17V5a2 2 0 00-1.999-2L16 3.002A5 5 0 0012 5a5 5 0 00-4-2H4a2 2 0 00-2 2v12a2 2 0 001.999 2H8a5 5 0 014 2 5 5 0 014-2z\"/>",
    "مجلد": "<path d=\"M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z\"/>",
    "مستخدم": "<path d=\"M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2\"/><circle cx=\"12\" cy=\"7\" r=\"4\"/>",
    "مستخدمون": "<path d=\"M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2\"/><path d=\"M16 3.128a4 4 0 0 1 0 7.744\"/><path d=\"M22 21v-2a4 4 0 0 0-3-3.87\"/><circle cx=\"9\" cy=\"7\" r=\"4\"/>",
    "مشاركة": "<circle cx=\"18\" cy=\"5\" r=\"3\"/><circle cx=\"6\" cy=\"12\" r=\"3\"/><circle cx=\"18\" cy=\"19\" r=\"3\"/><line x1=\"8.59\" x2=\"15.42\" y1=\"13.51\" y2=\"17.49\"/><line x1=\"15.41\" x2=\"8.59\" y1=\"6.51\" y2=\"10.49\"/>",
    "معلومة": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"M12 16v-4\"/><path d=\"M12 8h.01\"/>",
    "ملف": "<path d=\"M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z\"/><path d=\"M14 2v5a1 1 0 0 0 1 1h5\"/><path d=\"M10 9H8\"/><path d=\"M16 13H8\"/><path d=\"M16 17H8\"/>",
    "موقع": "<path d=\"M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0\"/><circle cx=\"12\" cy=\"10\" r=\"3\"/>",
    "ناقص": "<path d=\"M5 12h14\"/>",
    "نجاح": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"m16 9-5.5 5.5L8 12\"/>",
    "نجمة": "<path d=\"M11.525 2.295a.53.53 0 0 1 .95 0l2.31 4.679a2.123 2.123 0 0 0 1.595 1.16l5.166.756a.53.53 0 0 1 .294.904l-3.736 3.638a2.123 2.123 0 0 0-.611 1.878l.882 5.14a.53.53 0 0 1-.771.56l-4.618-2.428a2.122 2.122 0 0 0-1.973 0L6.396 21.01a.53.53 0 0 1-.77-.56l.881-5.139a2.122 2.122 0 0 0-.611-1.879L2.16 9.795a.53.53 0 0 1 .294-.906l5.165-.755a2.122 2.122 0 0 0 1.597-1.16z\"/>",
    "نسخ": "<rect width=\"14\" height=\"14\" x=\"8\" y=\"8\" rx=\"2\" ry=\"2\"/><path d=\"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2\"/>",
    "هاتف": "<path d=\"M13.832 16.568a1 1 0 0 0 1.213-.303l.355-.465A2 2 0 0 1 17 15h3a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2A18 18 0 0 1 2 4a2 2 0 0 1 2-2h3a2 2 0 0 1 2 2v3a2 2 0 0 1-.8 1.6l-.468.351a1 1 0 0 0-.292 1.233 14 14 0 0 0 6.392 6.384\"/>",
  };
  // icons:end
  let ui = null; // the render in progress

  function renderComponent(c, args, named, pos) {
    if (!ui) fail(pos, `المكون «${c.name}» يُعرض في صفحة ويب؛ شغّل البرنامج بـ«ضاد تطوير» وافتح الصفحة`);
    const parent = ui.frames[ui.frames.length - 1];
    const n = parent.counts[c.name] || 0;
    parent.counts[c.name] = n + 1;
    const key = parent.key + "/" + c.name + "#" + n;
    ui.seen[key] = true;
    const vals = bind(c.params, args, named, pos, "المكون «" + c.name + "»");
    ui.frames.push({ key, state: 0, counts: {} });
    try {
      try { c.body(...vals); } catch (e) { if (!(e instanceof ReturnSig)) throw e; }
    } finally { ui.frames.pop(); }
    return null;
  }

  // state runs «حالة اسم = قيمة»: init() gives the first value; the cell keeps it.
  function state(init, type, pos, name) {
    if (!ui || ui.frames.length < 2) fail(pos, "«حالة» تُستخدم داخل مكون فقط؛ خارجه استخدم «متغير»");
    const f = ui.frames[ui.frames.length - 1];
    const cells = (ui.sess.states[f.key] = ui.sess.states[f.key] || []);
    if (f.state >= cells.length) {
      let v = init();
      if (type) v = convert(v, type, pos, "الحالة «" + name + "»");
      cells.push({ val: v });
    }
    return cells[f.state++];
  }

  function emit(node) { const top = ui.stack[ui.stack.length - 1]; top.kids.push(node); }
  const node = (tag, attrs, text, kids, raw) => ({ tag, attrs: attrs || [], text: text || "", kids: kids || [], raw: raw || "" });
  function handler(h) { const id = "h" + ++ui.nextID; ui.handlers[id] = h; return id; }
  const textOf = (v) => (v instanceof StateRef ? textOf(v.cell.val) : typeof v === "string" ? v : format(v));

  function options(name, named, pos) {
    const allowed = elementOptions[name], o = {};
    for (const [k, v] of named || []) {
      if (!allowed.includes(k)) {
        if (allowed.length === 0) fail(pos, `«${name}» لا يأخذ معاملات مسماة`);
        fail(pos, `«${name}» لا يأخذ «${k}»؛ يأخذ: ${allowed.join("، ")}`);
      }
      o[k] = v;
    }
    return o;
  }
  function lambdaAfter(args, n, name, pos) {
    if (args.length <= n) return null;
    if (args.length > n + 1) fail(pos, `«${name}» أُعطي معاملات أكثر مما يأخذ`);
    if (!isFunc(args[n])) fail(pos, `«${name}» يتوقع هنا دالة بين { }، لكن وجدت «${typeName(args[n])}»`);
    return args[n];
  }
  function textArg(args, name, pos) {
    if (args.length === 0) fail(pos, `«${name}» يحتاج نصاً، مثل: ${name}("…")`);
    return textOf(args[0]);
  }
  const optText = (v, pos, name, opt) => { if (typeof v !== "string") fail(pos, `«${opt}» في «${name}» يأخذ نصاً`); return v; };
  const flag = (o, k, pos) => { if (!(k in o)) return false; if (typeof o[k] !== "boolean") fail(pos, `«${k}» يأخذ صح أو خطأ`); return o[k]; };

  function element(name, bindFirst, impl) {
    elements[name] = fn(name, (args, named, pos) => {
      if (!ui) fail(pos, `«${name}» عنصر واجهة، ويُستخدم داخل مكون`);
      return impl(args, named, pos);
    });
    elements[name].$bindFirst = bindFirst;
  }
  // ---------- the look of an element (elements.go)

  const colorNames = {
    "أساسي": "var(--lapis)", "باهت": "var(--muted)",
    "أحمر": "#c0392b", "أخضر": "#1e8e3e", "أزرق": "#2563eb", "أصفر": "#d4a106", "برتقالي": "#e67e22",
    "بنفسجي": "#7d3c98", "وردي": "#d63384", "فيروزي": "#0e8f9a", "ذهبي": "#c9a04a", "رمادي": "#6b7280",
    "كحلي": "#1b2b6b", "بني": "#8d5524", "أسود": "#111827", "أبيض": "#ffffff",
  };
  const hexColor = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;
  function color(v, pos, name, opt) {
    if (typeof v === "string" && colorNames[v]) return colorNames[v];
    if (typeof v === "string" && hexColor.test(v)) return v.toLowerCase();
    fail(pos, `«${opt}» في «${name}» يأخذ اسم لون مثل «أزرق» أو «فيروزي»، أو قيمة مثل «#1b2b6b»`);
  }
  const quoteAll = (words) => words.map((w) => "«" + w + "»").join(" أو ");
  function choice(v, pos, name, opt) {
    const words = optionWords(name, opt);
    if (typeof v === "string" && words.includes(v)) return v;
    fail(pos, `«${opt}» في «${name}» يكون ${quoteAll(words)}`);
  }
  const classWord = /^[\p{L}\p{N}_-]+$/u;
  const variantClass = { "ثانوي": "dad-secondary", "خفيف": "dad-quiet", "خطر": "dad-danger" };
  const sizeClass = { "صغير": "dad-small", "كبير": "dad-large" };
  const textAlign = { "بداية": "start", "وسط": "center", "نهاية": "end" };
  const isBox = (name) => name === "عمود" || name === "صف" || name === "بطاقة";

  function look(n, o, pos, name) {
    const classes = [], styles = [];
    if ("نمط" in o && name !== "تنبيه") {
      const v = choice(o["نمط"], pos, name, "نمط");
      if (name === "رابط") classes.push("dad-button");
      if (variantClass[v]) classes.push(variantClass[v]);
    }
    if ("حجم" in o) { const c = sizeClass[choice(o["حجم"], pos, name, "حجم")]; if (c) classes.push(c); }
    if (flag(o, "ملء", pos)) classes.push("dad-fill");
    if ("صنف" in o) {
      for (const w of optText(o["صنف"], pos, name, "صنف").split(/\s+/).filter(Boolean)) {
        if (!classWord.test(w)) fail(pos, "«صنف» يأخذ أسماء من حروف وأرقام و«_» و«-»، بينها مسافات، مثل: \"بطاقة-مميزة\"");
        classes.push(w);
      }
    }
    const box = isBox(name);
    if ("مسافة" in o && box) styles.push("gap:" + intValue(o["مسافة"], pos, "مسافة") + "px");
    if ("حشو" in o) styles.push("padding:" + intValue(o["حشو"], pos, "حشو") + "px");
    if ("عرض" in o && box) styles.push("max-width:" + intValue(o["عرض"], pos, "عرض") + "px");
    if ("لون" in o) styles.push("color:" + color(o["لون"], pos, name, "لون"));
    if ("خلفية" in o) styles.push("background:color-mix(in srgb, " + color(o["خلفية"], pos, name, "خلفية") + " 16%, var(--paper))");
    if ("محاذاة" in o && !box) styles.push("text-align:" + textAlign[choice(o["محاذاة"], pos, name, "محاذاة")]);
    if (classes.length) {
      const joined = classes.join(" ");
      let found = false;
      for (const at of n.attrs) if (at[0] === "class") { at[1] += " " + joined; found = true; }
      if (!found) n.attrs.push(["class", joined]);
    }
    if (styles.length) n.attrs.push(["style", styles.join(";")]);
  }

  function icon(v, pos, name, desc) {
    if (typeof v !== "string" || !Object.prototype.hasOwnProperty.call(icons, v)) {
      fail(pos, `لا توجد أيقونة باسم «${typeof v === "string" ? v : format(v)}» في «${name}»؛ انظر قائمة الأيقونات في دليل الواجهة`);
    }
    const n = node("svg", [["class", "dad-icon"], ["viewBox", "0 0 24 24"]], "", [], icons[v]);
    if (desc === "") n.attrs.push(["aria-hidden", "true"]);
    else n.attrs.push(["role", "img"], ["aria-label", desc]);
    return n;
  }
  function withIcon(n, o, pos, name, text) {
    if (!("أيقونة" in o)) { n.text = text; return; }
    n.kids.push(icon(o["أيقونة"], pos, name, ""), node("span", [], text));
  }
  function limit(v, pos, opt) {
    if (isInt(v) || isFloat(v) || typeof v === "string") return textOf(v);
    fail(pos, `«${opt}» يأخذ عدداً أو تاريخاً مثل "2026-01-31"`);
  }
  function label(v) {
    if (typeof v === "string") return v;
    if (v === null || v === undefined) return "";
    if (v instanceof EnumVal && v.vals.length === 0) return v.variant.name;
    if (v instanceof Struct) { for (const f of v.vals) if (typeof f === "string") return f; }
    return format(v);
  }
  function cellText(v) {
    if (typeof v === "boolean") return v ? "✓" : "";
    if (v === null || v === undefined) return "";
    return label(v);
  }
  function choiceIndex(v) {
    if (typeof v === "number") return Number.isInteger(v) ? v : -1;
    if (typeof v === "string" && /^-?\d+$/.test(v)) return parseInt(v, 10);
    return -1;
  }
  function anonState(pos, name, init) {
    if (ui.frames.length < 2) fail(pos, `«${name}» يُستخدم داخل مكون أو صفحة`);
    const f = ui.frames[ui.frames.length - 1];
    const cells = (ui.sess.states[f.key] = ui.sess.states[f.key] || []);
    if (f.state >= cells.length) cells.push({ val: init });
    return cells[f.state++];
  }
  function listArg(v, pos, name) {
    const x = v instanceof StateRef ? v.cell.val : v;
    if (!(x instanceof List)) fail(pos, `«${name}» يأخذ قائمة، لكن وجدت «${typeName(x)}»`);
    return x.elems;
  }
  function columnsOf(r) {
    if (r instanceof Struct) {
      const fs = r.type.fields.map((f) => f.name);
      return r.type.table ? fs.slice(0, -1) : fs;
    }
    if (r instanceof DMap) return r.keys.map(textOf);
    return [""];
  }
  function fieldOf(r, col, pos) {
    if (r instanceof Struct) {
      const i = r.type.fields.findIndex((f) => f.name === col);
      if (i < 0) fail(pos, `«جدول_عرض»: لا يوجد حقل باسم «${col}» في «${r.type.name}»`);
      return r.vals[i];
    }
    if (r instanceof DMap) { const i = r.keys.findIndex((k) => textOf(k) === col); return i < 0 ? null : r.vals[i]; }
    if (col === "") return r;
    fail(pos, `«جدول_عرض»: لا يوجد حقل باسم «${col}» في قيمة من النوع «${typeName(r)}»`);
  }

  // ---------- the elements

  const container = (name, cls) => element(name, false, (args, named, pos) => {
    const o = options(name, named, pos);
    const n = node("div", [["class", cls]]);
    if ("محاذاة" in o) n.attrs.push(["data-align", choice(o["محاذاة"], pos, name, "محاذاة")]);
    look(n, o, pos, name);
    emit(n);
    const f = lambdaAfter(args, 0, name, pos);
    if (f) {
      ui.stack.push(n);
      try { call(f, [], [], pos, name); } finally { ui.stack.pop(); }
    }
    return null;
  });
  container("عمود", "dad-col");
  container("صف", "dad-row");
  container("بطاقة", "dad-card");
  element("عنوان", false, (args, named, pos) => {
    const o = options("عنوان", named, pos);
    let level = 1n;
    if ("مستوى" in o) { level = intValue(o["مستوى"], pos, "مستوى"); level = level < 1n ? 1n : level > 6n ? 6n : level; }
    const n = node("h" + level, [], textArg(args, "عنوان", pos));
    look(n, o, pos, "عنوان");
    emit(n);
    return null;
  });
  element("فقرة", false, (args, named, pos) => {
    const o = options("فقرة", named, pos);
    const n = node("p", [], textArg(args, "فقرة", pos));
    look(n, o, pos, "فقرة");
    emit(n);
    return null;
  });
  element("زر", false, (args, named, pos) => {
    const o = options("زر", named, pos);
    const n = node("button");
    withIcon(n, o, pos, "زر", textArg(args, "زر", pos));
    n.attrs.push(["type", "button"]);
    look(n, o, pos, "زر");
    if (flag(o, "معطل", pos)) n.attrs.push(["disabled", ""]);
    const f = lambdaAfter(args, 1, "زر", pos);
    if (f) n.attrs.push(["data-click", handler({ kind: "click", fn: f })]);
    emit(n);
    return null;
  });
  element("حقل", true, (args, named, pos) => {
    const o = options("حقل", named, pos);
    if (args.length === 0) fail(pos, "«حقل» يأخذ متغير حالة، مثل: حقل(اسم، تلميح: \"اسمك\")");
    const n = node("input");
    const v0 = args[0] instanceof StateRef ? args[0].cell.val : args[0];
    let kind = isInt(v0) || isFloat(v0) ? "number" : "text";
    if ("صيغة" in o) kind = { "نص": "text", "رقم": "number", "كلمة_سر": "password", "بريد": "email", "تاريخ": "date", "وقت": "time" }[choice(o["صيغة"], pos, "حقل", "صيغة")];
    n.attrs.push(["type", kind]);
    const h = { kind: "input", fn: lambdaAfter(args, 1, "حقل", pos), cell: args[0] instanceof StateRef ? args[0].cell : null };
    n.attrs.push(["value", textOf(args[0])]);
    if ("تلميح" in o) n.attrs.push(["placeholder", optText(o["تلميح"], pos, "حقل", "تلميح")]);
    for (const [k, attr] of [["أدنى", "min"], ["أقصى", "max"]]) if (k in o) n.attrs.push([attr, limit(o[k], pos, k)]);
    look(n, o, pos, "حقل");
    if (flag(o, "معطل", pos)) n.attrs.push(["disabled", ""]);
    if (h.cell || h.fn) n.attrs.push(["data-input", handler(h)]);
    emit(n);
    return null;
  });
  element("مربع_اختيار", true, (args, named, pos) => {
    const o = options("مربع_اختيار", named, pos);
    if (args.length === 0) fail(pos, "«مربع_اختيار» يأخذ قيمة منطقية، مثل: مربع_اختيار(م.منجزة) { … }");
    const h = { kind: "toggle", fn: lambdaAfter(args, 1, "مربع_اختيار", pos), cell: args[0] instanceof StateRef ? args[0].cell : null };
    const on = h.cell ? h.cell.val : args[0];
    if (typeof on !== "boolean") fail(pos, `«مربع_اختيار» يأخذ قيمة منطقية (صح أو خطأ)، لكن وجدت «${typeName(on)}»`);
    const box = node("input", [["type", "checkbox"]]);
    if (on) box.attrs.push(["checked", ""]);
    if (flag(o, "معطل", pos)) box.attrs.push(["disabled", ""]);
    if (h.cell || h.fn) box.attrs.push(["data-toggle", handler(h)]);
    if ("نص" in o) {
      const l = node("label", [["class", "dad-check"]], "", [box, node("span", [], optText(o["نص"], pos, "مربع_اختيار", "نص"))]);
      look(l, o, pos, "مربع_اختيار");
      emit(l);
    } else {
      look(box, o, pos, "مربع_اختيار");
      emit(box);
    }
    return null;
  });
  element("صورة", false, (args, named, pos) => {
    const o = options("صورة", named, pos);
    const n = node("img", [["src", textArg(args, "صورة", pos)], ["alt", "وصف" in o ? optText(o["وصف"], pos, "صورة", "وصف") : ""]]);
    if ("عرض" in o) n.attrs.push(["width", String(intValue(o["عرض"], pos, "عرض"))]);
    look(n, o, pos, "صورة");
    emit(n);
    return null;
  });
  element("رابط", false, (args, named, pos) => {
    const o = options("رابط", named, pos);
    if (args.length !== 2) fail(pos, "«رابط» يأخذ النص والعنوان، مثل: رابط(\"الموقع\"، \"https://…\")");
    const href = optText(args[1], pos, "رابط", "العنوان");
    const n = node("a", [["href", href]]);
    withIcon(n, o, pos, "رابط", textOf(args[0]));
    // A link to one of the program's pages opens it in place; the link to the
    // page being shown is marked, for navigation bars.
    if (isPageLink(href) && matchPage(href)) {
      if (routing === "hash") n.attrs[0][1] = "#" + href;
      n.attrs.push(["data-nav", ""]);
      if (cleanPath(href) === cleanPath(ui.path)) n.attrs.push(["aria-current", "page"]);
    }
    look(n, o, pos, "رابط");
    emit(n);
    return null;
  });
  element("فاصل", false, (args, named, pos) => {
    const o = options("فاصل", named, pos);
    const n = node("hr");
    look(n, o, pos, "فاصل");
    emit(n);
    return null;
  });

  const choose = (name, radios) => element(name, true, (args, named, pos) => {
    const o = options(name, named, pos);
    if (args.length < 2) fail(pos, `«${name}» يأخذ القيمة المختارة وقائمة الخيارات، مثل: ${name}(اللون، ["أحمر"، "أخضر"])`);
    const opts = listArg(args[1], pos, name);
    const h = { kind: "choose", fn: lambdaAfter(args, 2, name, pos), options: opts, cell: args[0] instanceof StateRef ? args[0].cell : null };
    const current = args[0] instanceof StateRef ? args[0].cell.val : args[0];
    const disabled = flag(o, "معطل", pos);
    const id = h.cell || h.fn ? handler(h) : "";
    let selected = -1;
    for (let i = 0; i < opts.length; i++) if (equal(opts[i], current)) { selected = i; break; }
    if (!radios) {
      const n = node("select");
      if (id) n.attrs.push(["data-choose", id]);
      if (disabled) n.attrs.push(["disabled", ""]);
      look(n, o, pos, name);
      if ("تلميح" in o || selected < 0) {
        const op = node("option", [], "تلميح" in o ? optText(o["تلميح"], pos, name, "تلميح") : "");
        op.attrs.push(["value", ""], ["disabled", ""]);
        if (selected < 0) op.attrs.push(["selected", ""]);
        n.kids.push(op);
      }
      opts.forEach((v, i) => {
        const op = node("option", [["value", String(i)]], label(v));
        if (i === selected) op.attrs.push(["selected", ""]);
        n.kids.push(op);
      });
      emit(n);
      return null;
    }
    const group = node("div", [["class", "dad-choices"], ["role", "radiogroup"]]);
    look(group, o, pos, name);
    opts.forEach((v, i) => {
      const box = node("input", [["type", "radio"], ["name", "r" + (ui.nextID + 1) + "-" + id], ["value", String(i)]]);
      if (i === selected) box.attrs.push(["checked", ""]);
      if (disabled) box.attrs.push(["disabled", ""]);
      if (id) box.attrs.push(["data-choose", id]);
      group.kids.push(node("label", [["class", "dad-check"]], "", [box, node("span", [], label(v))]));
    });
    emit(group);
    return null;
  });
  choose("قائمة_منسدلة", false);
  choose("اختيار", true);

  element("منطقة_نص", true, (args, named, pos) => {
    const o = options("منطقة_نص", named, pos);
    if (args.length === 0) fail(pos, "«منطقة_نص» يأخذ متغير حالة، مثل: منطقة_نص(الملاحظات)");
    const h = { kind: "input", fn: lambdaAfter(args, 1, "منطقة_نص", pos), cell: args[0] instanceof StateRef ? args[0].cell : null };
    let text = textOf(args[0]);
    if (text.startsWith("\n")) text = "\n" + text; // a browser drops the first line break inside <textarea>
    const n = node("textarea", [], text);
    let rows = 4n;
    if ("أسطر" in o) { rows = intValue(o["أسطر"], pos, "أسطر"); rows = rows < 1n ? 1n : rows > 40n ? 40n : rows; }
    n.attrs.push(["rows", String(rows)]);
    if ("تلميح" in o) n.attrs.push(["placeholder", optText(o["تلميح"], pos, "منطقة_نص", "تلميح")]);
    look(n, o, pos, "منطقة_نص");
    if (flag(o, "معطل", pos)) n.attrs.push(["disabled", ""]);
    if (h.cell || h.fn) n.attrs.push(["data-input", handler(h)]);
    emit(n);
    return null;
  });

  function emptyMessage(o, pos, name) {
    emit(node("p", [["class", "dad-empty"]], "فارغ" in o ? optText(o["فارغ"], pos, name, "فارغ") : "لا شيء لعرضه بعد."));
  }
  element("جدول_عرض", false, (args, named, pos) => {
    const o = options("جدول_عرض", named, pos);
    if (args.length === 0) fail(pos, "«جدول_عرض» يأخذ قائمة، مثل: جدول_عرض(مهمة.الكل())");
    const rows = listArg(args[0], pos, "جدول_عرض");
    const f = lambdaAfter(args, 1, "جدول_عرض", pos);
    if (rows.length === 0) { emptyMessage(o, pos, "جدول_عرض"); return null; }
    let cols;
    if ("أعمدة" in o) {
      cols = listArg(o["أعمدة"], pos, "أعمدة").map((c) => {
        if (typeof c !== "string") fail(pos, "«أعمدة» قائمة بأسماء الحقول، مثل: [\"العنوان\"، \"منجزة\"]");
        return c;
      });
    } else cols = columnsOf(rows[0]);
    const wrap = node("div", [["class", "dad-table"]]);
    look(wrap, o, pos, "جدول_عرض");
    const head = node("tr", [], "", cols.map((c) => node("th", [], c.replaceAll("_", " "))));
    if (f) head.kids.push(node("th"));
    const body = node("tbody");
    wrap.kids.push(node("table", [], "", [node("thead", [], "", [head]), body]));
    emit(wrap);
    for (const r of rows) {
      const tr = node("tr");
      body.kids.push(tr);
      for (const c of cols) tr.kids.push(node("td", [], cellText(fieldOf(r, c, pos))));
      if (f) {
        const td = node("td", [["class", "dad-actions"]]);
        tr.kids.push(td);
        ui.stack.push(td);
        try { call(f, [r], [], pos, "جدول_عرض"); } finally { ui.stack.pop(); }
      }
    }
    return null;
  });
  element("قائمة_عناصر", false, (args, named, pos) => {
    const o = options("قائمة_عناصر", named, pos);
    if (args.length === 0) fail(pos, "«قائمة_عناصر» يأخذ قائمة، مثل: قائمة_عناصر([\"أ\"، \"ب\"])");
    const items = listArg(args[0], pos, "قائمة_عناصر");
    const f = lambdaAfter(args, 1, "قائمة_عناصر", pos);
    if (items.length === 0) { emptyMessage(o, pos, "قائمة_عناصر"); return null; }
    const n = node(flag(o, "مرقمة", pos) ? "ol" : "ul", [["class", "dad-list"]]);
    look(n, o, pos, "قائمة_عناصر");
    emit(n);
    for (const it of items) {
      const li = node("li");
      n.kids.push(li);
      if (!f) { li.text = label(it); continue; }
      ui.stack.push(li);
      try { call(f, [it], [], pos, "قائمة_عناصر"); } finally { ui.stack.pop(); }
    }
    return null;
  });

  element("ألسنة", true, (args, named, pos) => {
    const o = options("ألسنة", named, pos);
    let f = null, cellRef = null;
    if (args.length === 1) f = lambdaAfter(args, 0, "ألسنة", pos);
    else if (args.length === 2) {
      if (!(args[0] instanceof StateRef)) fail(pos, "«ألسنة» يأخذ متغير حالة للسان المفتوح، مثل: ألسنة(اللسان) { … }");
      cellRef = args[0].cell;
      f = lambdaAfter(args, 1, "ألسنة", pos);
    }
    if (!f) fail(pos, "«ألسنة» يأخذ ألسنتها بين { }، مثل: ألسنة { لسان(\"الأول\") { … } }");
    if (!cellRef) cellRef = anonState(pos, "ألسنة", 0n);
    // Collect the tabs: their blocks run only for the tab that is open.
    const tabs = [];
    const saved = ui.tabs, savedStack = ui.stack, scratch = node("div");
    ui.tabs = tabs;
    ui.stack = [...ui.stack, scratch];
    try { call(f, [], [], pos, "ألسنة"); } finally { ui.tabs = saved; ui.stack = savedStack; }
    if (scratch.kids.length) fail(pos, "داخل «ألسنة» تُكتب الألسنة فقط، مثل: لسان(\"الأول\") { … }");
    let open = 0;
    const cur = cellRef.val;
    if (isInt(cur)) open = Number(cur);
    else if (typeof cur === "string") tabs.forEach((t, i) => { if (t.title === cur) open = i; });
    if (open < 0 || open >= tabs.length) open = 0;
    const n = node("div", [["class", "dad-tabs"]]);
    look(n, o, pos, "ألسنة");
    const list = node("div", [["class", "dad-tablist"], ["role", "tablist"]]);
    const panel = node("div", [["class", "dad-tabpanel"], ["role", "tabpanel"]]);
    n.kids.push(list, panel);
    emit(n);
    tabs.forEach((t, i) => {
      const b = node("button");
      withIcon(b, t.o, t.pos, "لسان", t.title);
      b.attrs.push(["type", "button"], ["role", "tab"], ["aria-selected", String(i === open)]);
      b.attrs.push(["data-click", handler({ kind: "set", cell: cellRef, value: typeof cellRef.val === "string" ? t.title : BigInt(i) })]);
      look(b, t.o, t.pos, "لسان");
      list.kids.push(b);
    });
    if (tabs.length) {
      ui.stack.push(panel);
      try { call(tabs[open].f, [], [], tabs[open].pos, "لسان"); } finally { ui.stack.pop(); }
    }
    return null;
  });
  element("لسان", false, (args, named, pos) => {
    const o = options("لسان", named, pos);
    if (!ui.tabs) fail(pos, "«لسان» يُستخدم داخل «ألسنة»");
    const title = textArg(args, "لسان", pos);
    const f = lambdaAfter(args, 1, "لسان", pos);
    if (!f) fail(pos, "«لسان» يأخذ عنوانه وما يعرضه بين { }، مثل: لسان(\"الأول\") { … }");
    if ("أيقونة" in o) icon(o["أيقونة"], pos, "لسان", "");
    ui.tabs.push({ title, f, o, pos });
    return null;
  });

  element("نافذة", true, (args, named, pos) => {
    const o = options("نافذة", named, pos);
    if (args.length === 0) fail(pos, "«نافذة» يأخذ حالة منطقية تفتحها وتغلقها، مثل: نافذة(مفتوحة) { … }");
    const cellRef = args[0] instanceof StateRef ? args[0].cell : null;
    const isOpen = cellRef ? cellRef.val : args[0];
    if (typeof isOpen !== "boolean") fail(pos, `«نافذة» تأخذ قيمة منطقية (صح أو خطأ)، لكن وجدت «${typeName(isOpen)}»`);
    const f = lambdaAfter(args, 1, "نافذة", pos);
    if (!isOpen) return null;
    const title = "عنوان" in o ? optText(o["عنوان"], pos, "نافذة", "عنوان") : "";
    const box = node("div", [["class", "dad-dialog"], ["role", "dialog"], ["aria-modal", "true"], ["tabindex", "-1"]]);
    if (title) box.attrs.push(["aria-label", title]);
    look(box, o, pos, "نافذة");
    const head = node("div", [["class", "dad-dialog-head"]]);
    if (title) head.kids.push(node("h2", [], title));
    if (cellRef) {
      const x = node("button", [["type", "button"], ["class", "dad-close"], ["aria-label", "إغلاق"]], "", [icon("إغلاق", pos, "نافذة", "")]);
      x.attrs.push(["data-click", handler({ kind: "set", cell: cellRef, value: false })]);
      head.kids.push(x);
    }
    const body = node("div", [["class", "dad-col"]]);
    box.kids.push(head, body);
    emit(node("div", [["class", "dad-backdrop"]], "", [box]));
    if (f) {
      ui.stack.push(body);
      try { call(f, [], [], pos, "نافذة"); } finally { ui.stack.pop(); }
    }
    return null;
  });

  element("تنبيه", false, (args, named, pos) => {
    const o = options("تنبيه", named, pos);
    const kind = "نمط" in o ? choice(o["نمط"], pos, "تنبيه", "نمط") : "معلومة";
    const cls = { "معلومة": "info", "نجاح": "success", "تحذير": "warning", "خطأ": "error" }[kind];
    const ic = { "معلومة": "معلومة", "نجاح": "نجاح", "تحذير": "تحذير", "خطأ": "خطر" }[kind];
    const n = node("div", [["class", "dad-alert dad-alert-" + cls], ["role", kind === "خطأ" || kind === "تحذير" ? "alert" : "status"]]);
    look(n, o, pos, "تنبيه");
    n.kids.push(icon(ic, pos, "تنبيه", ""), node("span", [], textArg(args, "تنبيه", pos)));
    emit(n);
    return null;
  });

  element("أيقونة", false, (args, named, pos) => {
    const o = options("أيقونة", named, pos);
    if (args.length !== 1) fail(pos, "«أيقونة» تأخذ اسم الأيقونة، مثل: أيقونة(\"بحث\")");
    const n = icon(args[0], pos, "أيقونة", "وصف" in o ? optText(o["وصف"], pos, "أيقونة", "وصف") : "");
    look(n, o, pos, "أيقونة");
    emit(n);
    return null;
  });

  // ---------- the theme: «سمة(اللون: "فيروزي")» at the top of a program

  let theme = null;
  const themeOptions = ["اللون", "الخط", "الزوايا", "العرض", "الحجم", "الوضع"];
  const themeValues = { "الخط": ["كوفي", "نسخ", "النظام"], "الوضع": ["تلقائي", "فاتح", "داكن"] };
  function onColor(hex) {
    let h = hex.slice(1);
    if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
    const r = parseInt(h.slice(0, 2), 16), g = parseInt(h.slice(2, 4), 16), b = parseInt(h.slice(4, 6), 16);
    return 299 * r + 587 * g + 114 * b > 150000 ? "#111827" : "#ffffff";
  }
  def("سمة", (args, named, pos) => {
    if (ui || eventSess) fail(pos, "«سمة» تُكتب في أعلى البرنامج، لا داخل مكون أو حدث");
    if (args.length) fail(pos, "«سمة» تأخذ معاملات مسماة، مثل: سمة(اللون: \"فيروزي\")");
    const t = { vars: [], mode: "" };
    const px = (v, opt, lo, hi) => {
      const n = intValue(v, pos, opt);
      if (n < BigInt(lo) || n > BigInt(hi)) fail(pos, `«${opt}» في «سمة» بين ${lo} و${hi}`);
      return n + "px";
    };
    const word = (v, opt) => {
      if (typeof v === "string" && themeValues[opt].includes(v)) return v;
      fail(pos, `«${opt}» في «سمة» يكون ${quoteAll(themeValues[opt])}`);
    };
    for (const [k, v] of named || []) {
      if (k === "اللون") {
        const c = color(v, pos, "سمة", "اللون");
        if (!c.startsWith("#")) fail(pos, "«اللون» في «سمة» يأخذ لوناً بعينه، مثل «فيروزي» أو «#0e8f9a»");
        t.vars.push("--theme-primary:" + c, "--theme-on-primary:" + onColor(c));
      } else if (k === "الخط") t.vars.push("--theme-font:var(--font-" + { "كوفي": "kufi", "نسخ": "naskh", "النظام": "system" }[word(v, "الخط")] + ")");
      else if (k === "الزوايا") t.vars.push("--theme-radius:" + px(v, "الزوايا", 0, 40));
      else if (k === "العرض") t.vars.push("--theme-width:" + px(v, "العرض", 320, 2000));
      else if (k === "الحجم") t.vars.push("--theme-size:" + px(v, "الحجم", 12, 28));
      else if (k === "الوضع") t.mode = { "تلقائي": "", "فاتح": "light", "داكن": "dark" }[word(v, "الوضع")];
      else fail(pos, `«سمة» لا تأخذ «${k}»؛ تأخذ: ${themeOptions.join("، ")}`);
    }
    theme = t;
    return null;
  });
  function themed(root) {
    if (!theme) return;
    if (theme.mode) root.attrs.push(["data-mode", theme.mode]);
    if (theme.vars.length) root.kids.unshift(node("style", [], "", [], ":root{" + theme.vars.join(";") + "}"));
  }

  const voidTags = new Set(["input", "img", "hr", "br"]);
  const esc = (s) => s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&#34;", "'": "&#39;" }[c]));
  function html(n) {
    let s = "<" + n.tag;
    for (const [k, v] of n.attrs) s += " " + k + (v !== "" || k === "value" || k === "alt" ? '="' + esc(v) + '"' : "");
    s += ">";
    if (voidTags.has(n.tag)) return s;
    return s + esc(n.text) + n.raw + n.kids.map(html).join("") + "</" + n.tag + ">";
  }

  function newSession() {
    return { version: 0, path: "", title: "", notFound: false, moved: false, away: "", states: {}, handlers: {}, page: null, params: {} };
  }

  function render(sess, root) {
    appRoot = root;
    return draw(sess, () => call(root, [], [], [0, 0], root.name));
  }

  function draw(sess, layout) {
    const top = node("div", [["class", "dad-app"]]);
    ui = { sess, path: sess.path, stack: [top], frames: [{ key: "", counts: {} }], handlers: {}, nextID: 0, seen: {} };
    try { layout(); } finally { var done = ui; ui = null; }
    for (const k of Object.keys(sess.states)) if (!done.seen[k]) delete sess.states[k];
    sess.handlers = done.handlers;
    sess.version++;
    themed(top);
    sess.title = firstHeading(top);
    return html(top);
  }

  // ---------------------------------------------------------------- pages

  // «صفحة "/المهام" = …» registers a page; a program without pages shows its
  // main component at "/". Addresses match as in the interpreter.
  const pages = [];
  let appRoot = null;
  let routing = "history"; // "hash" in a static site: links are «#/المهام»
  function page(path, fn) { pages.push({ path, fn, comp: null }); }
  const allPages = () => (pages.length ? pages : appRoot ? [{ path: "/", fn: null, comp: appRoot }] : []);

  function cleanPath(path) {
    path = String(path || "");
    const i = path.search(/[?#]/);
    if (i >= 0) path = path.slice(0, i);
    try { path = decodeURIComponent(path); } catch (e) {}
    if (path.length > 1) path = path.replace(/\/+$/, "");
    return path || "/";
  }
  const isPageLink = (href) => href.startsWith("/") && !href.startsWith("//");
  const trimSlashes = (s) => s.replace(/^\/+|\/+$/g, "");
  function matchPath(pattern, path) {
    const ps = trimSlashes(pattern).split("/"), xs = trimSlashes(path).split("/");
    if (ps.length !== xs.length) return null;
    const params = {};
    for (let i = 0; i < ps.length; i++) {
      const m = /^\{([^{}/]+)\}$/.exec(ps[i]);
      if (m) { params[m[1].trim()] = xs[i]; continue; }
      if (ps[i] !== xs[i]) return null;
    }
    return params;
  }
  function matchPage(path) {
    path = cleanPath(path);
    for (const p of allPages()) {
      const params = matchPath(p.path, path);
      if (params) return { page: p, params };
    }
    return null;
  }
  // pathValue is a value from an address: a whole number when it is one.
  function pathValue(v) {
    if (/^[+-]?\d+$/.test(v)) {
      const n = BigInt(v);
      if (BigInt.asIntN(64, n) === n) return n;
    }
    return v;
  }

  // open points a session at an address; it reports whether a page is there.
  function open(sess, path, root) {
    if (root !== undefined) appRoot = root;
    sess.path = cleanPath(path);
    const m = matchPage(sess.path);
    sess.page = m ? m.page : null;
    sess.params = m ? m.params : {};
    sess.notFound = !m;
    return !!m;
  }

  function renderPage(sess) {
    const p = sess.page;
    if (!p) {
      const out = draw(sess, () => notFound(sess.path));
      sess.title = "الصفحة غير موجودة";
      return out;
    }
    if (p.comp) return draw(sess, () => call(p.comp, [], [], [0, 0], p.comp.name));
    return draw(sess, () => {
      // laid out like a component of its own, with its state kept by address
      const key = "@" + sess.path;
      ui.frames.push({ key, state: 0, counts: {} });
      ui.seen[key] = true;
      try {
        try { p.fn(sess.params); } catch (e) { if (!(e instanceof ReturnSig)) throw e; }
      } finally { ui.frames.pop(); }
    });
  }

  function notFound(path) {
    const col = node("div", [["class", "dad-col dad-notfound"]], "", [
      node("h1", [], "الصفحة غير موجودة"),
      node("p", [], "لا توجد صفحة بالعنوان «" + path + "»."),
    ]);
    if (matchPage("/")) col.kids.push(node("a", [["href", routing === "hash" ? "#/" : "/"], ["data-nav", ""]], "إلى الصفحة الرئيسية"));
    emit(col);
  }

  function firstHeading(n) {
    if (/^h[1-6]$/.test(n.tag)) return n.text;
    for (const k of n.kids) { const t = firstHeading(k); if (t) return t; }
    return "";
  }

  // «انتقل("/المهام")»: after the event, the page moves to that address.
  let eventSess = null;
  def("انتقل", (args, named, pos) => {
    if (args.length !== 1 || (named && named.length)) fail(pos, "«انتقل» يأخذ عنوان الصفحة، مثل: انتقل(\"/المهام\")");
    const to = args[0];
    if (typeof to !== "string") fail(pos, `«انتقل» يأخذ عنواناً نصياً، مثل: انتقل("/المهام")، لكن وجدت «${typeName(to)}»`);
    const sess = eventSess;
    if (!sess) {
      if (ui) fail(pos, "«انتقل» تُستخدم عند حدث، مثل الضغط على زر، لا أثناء عرض الصفحة");
      fail(pos, "«انتقل» تُستخدم في صفحة ويب، عند حدث مثل الضغط على زر");
    }
    if (to.startsWith("https://") || to.startsWith("http://")) sess.away = to;
    else if (to.startsWith("/")) { open(sess, to); sess.moved = true; }
    else fail(pos, "عنوان الصفحة في «انتقل» يبدأ بـ «/»، مثل: انتقل(\"/المهام\")");
    return null;
  });

  function fieldValue(old, s) {
    if (isInt(old)) { const t = s.trim(); if (/^[+-]?\d+$/.test(t)) return BigInt(t); }
    if (isFloat(old)) { const f = parseNumber(s); if (f !== null) return f; }
    return s;
  }

  function event(sess, version, id, value) {
    if (version !== sess.version) return;
    const h = sess.handlers[id];
    if (!h) return;
    sess.moved = false;
    sess.away = "";
    eventSess = sess;
    try { handle(h, value); } finally { eventSess = null; }
  }

  function handle(h, value) {
    if (h.kind === "click") { if (h.fn) call(h.fn, [], [], [0, 0], "زر"); }
    else if (h.kind === "input") {
      let v = String(value);
      if (h.cell) { v = fieldValue(h.cell.val, v); h.cell.val = v; }
      if (h.fn) call(h.fn, [v], [], [0, 0], "حقل");
    } else if (h.kind === "toggle") {
      if (h.cell) h.cell.val = !!value;
      if (h.fn) call(h.fn, [!!value], [], [0, 0], "مربع_اختيار");
    } else if (h.kind === "choose") {
      const i = choiceIndex(value);
      if (i < 0 || i >= h.options.length) return;
      const v = h.options[i];
      if (h.cell) h.cell.val = v;
      if (h.fn) call(h.fn, [v], [], [0, 0], "اختيار");
    } else if (h.kind === "set") {
      h.cell.val = h.value;
    }
  }

  // mount shows the program in #dad-root and handles its events in the page.
  // A static site keeps the address of its pages after «#», like «#/المهام», so
  // it works from any host, and from the disk.
  function mount(root) {
    if (typeof localStorage !== "undefined") {
      const key = "dad:" + (document.title || "app");
      try { const saved = JSON.parse(localStorage.getItem(key)); if (saved && saved.tables) store.tables = saved.tables; } catch (e) {}
      store.persist = () => { try { localStorage.setItem(key, JSON.stringify({ tables: store.tables })); } catch (e) {} };
    }
    routing = "hash";
    const el = document.getElementById("dad-root");
    el.setAttribute("data-routing", "hash");
    const name = document.title;
    const sess = newSession();
    const here = () => { try { return decodeURIComponent(location.hash.replace(/^#/, "")) || "/"; } catch (e) { return "/"; } };
    const show = (fn) => {
      try { return { html: fn(), version: sess.version, title: sess.title || name }; }
      catch (e) { return { error: e instanceof Thrown ? "مشكلة لم تُعالَج: " + problemMessage(e.val) : e.message }; }
    };
    open(sess, here(), root || null);
    const first = show(() => renderPage(sess));
    el.innerHTML = first.html || "";
    el.setAttribute("data-version", String(sess.version));
    if (first.title) document.title = first.title;
    if (first.error) { const box = document.getElementById("dad-error"); box.textContent = first.error; box.hidden = false; }
    root$ = root;
    window.dadSend = (e) => {
      if (e.nav !== undefined) {
        open(sess, e.nav);
        const r = show(() => renderPage(sess));
        r.path = sess.path;
        return r;
      }
      let away = "";
      const r = show(() => {
        event(sess, e.v, e.id, e.value);
        away = sess.away;
        return away ? "" : renderPage(sess);
      });
      if (away) return { location: away };
      if (sess.moved) r.path = sess.path;
      return r;
    };
  }
  let root$ = null;

  // ---------------------------------------------------------------- running a program

  const problemMessage = (v) => (v instanceof Problem ? v.message : format(v));

  // main runs a compiled program and reports an error the way «dad run» does.
  function main(program, file) {
    try {
      return program();
    } catch (e) {
      let msg;
      if (e instanceof RuntimeError) msg = e.message;
      else if (e instanceof Thrown) msg = `خطأ أثناء التشغيل [السطر ${e.pos[0]}، العمود ${e.pos[1]}]: مشكلة لم تُعالَج: ${problemMessage(e.val)}`;
      else if (e instanceof ReturnSig) msg = "خطأ أثناء التشغيل [السطر 0، العمود 0]: «أرجع» مستخدمة خارج دالة";
      else if (e === BREAK) msg = "خطأ أثناء التشغيل [السطر 0، العمود 0]: «توقف» مستخدمة خارج حلقة";
      else if (e === CONTINUE) msg = "خطأ أثناء التشغيل [السطر 0، العمود 0]: «تابع» مستخدمة خارج حلقة";
      else if (e instanceof RangeError) msg = "تجاوز البرنامج الحد الأقصى لعمق الاستدعاءات — هل توجد دالة تستدعي نفسها بلا نهاية؟";
      else throw e;
      if (typeof process !== "undefined" && process.stderr) {
        process.stderr.write((file ? file + ": " : "") + msg + "\n");
        process.exitCode = 1;
      } else console.error(msg);
      return null;
    }
  }

  const $ = {
    List, Range, DMap, StructType, Struct, EnumType, Variant, EnumVal, Problem, Component, StateRef,
    RuntimeError, Thrown, ReturnSig, BREAK, CONTINUE,
    fail, raise, format, equal, bin, neg, truthy, convert, cast, isType, intValue, typeName,
    fn, bind, call, lam, member, method, selfGet, ext, types, variants, b, elements,
    E: Object.freeze([]), doThrow, undef, variant, extend,
    str, index, setIndex, keepFloat, map, range, iter, throwValue, caught, noMatch, patternFields,
    state, render, newSession, event, mount, main, store,
    page, pages, pathValue, open, renderPage,
    setOutput(f) { out = f; },
  };
  if (typeof module !== "undefined" && module.exports) module.exports = $;
  else root.$dad = $;
})(typeof globalThis !== "undefined" ? globalThis : this);
