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
  const elementOptions = {
    "عمود": ["محاذاة", "مسافة"], "صف": ["محاذاة", "مسافة"], "بطاقة": ["محاذاة", "مسافة"],
    "عنوان": ["مستوى"], "فقرة": [], "زر": ["معطل"], "حقل": ["تلميح", "صيغة", "معطل"],
    "مربع_اختيار": ["نص", "معطل"], "صورة": ["وصف", "عرض"], "رابط": [], "فاصل": [],
  };
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
  const node = (tag, attrs, text, kids) => ({ tag, attrs: attrs || [], text: text || "", kids: kids || [] });
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
  const container = (name, cls) => element(name, false, (args, named, pos) => {
    const o = options(name, named, pos);
    const n = node("div", [["class", cls]]);
    if ("محاذاة" in o) n.attrs.push(["data-align", optText(o["محاذاة"], pos, name, "محاذاة")]);
    if ("مسافة" in o) n.attrs.push(["style", "gap:" + intValue(o["مسافة"], pos, "مسافة") + "px"]);
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
    emit(node("h" + level, [], textArg(args, "عنوان", pos)));
    return null;
  });
  element("فقرة", false, (args, named, pos) => { options("فقرة", named, pos); emit(node("p", [], textArg(args, "فقرة", pos))); return null; });
  element("زر", false, (args, named, pos) => {
    const o = options("زر", named, pos);
    const n = node("button", [["type", "button"]], textArg(args, "زر", pos));
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
    if ("صيغة" in o) {
      const k = optText(o["صيغة"], pos, "حقل", "صيغة");
      kind = { "نص": "text", "رقم": "number", "كلمة_سر": "password", "كلمة سر": "password", "بريد": "email" }[k];
      if (!kind) fail(pos, "صيغة الحقل تكون «نص» أو «رقم» أو «كلمة_سر» أو «بريد»");
    }
    n.attrs.push(["type", kind]);
    const h = { kind: "input", fn: lambdaAfter(args, 1, "حقل", pos), cell: args[0] instanceof StateRef ? args[0].cell : null };
    n.attrs.push(["value", textOf(args[0])]);
    if ("تلميح" in o) n.attrs.push(["placeholder", optText(o["تلميح"], pos, "حقل", "تلميح")]);
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
    if ("نص" in o) emit(node("label", [["class", "dad-check"]], "", [box, node("span", [], optText(o["نص"], pos, "مربع_اختيار", "نص"))]));
    else emit(box);
    return null;
  });
  element("صورة", false, (args, named, pos) => {
    const o = options("صورة", named, pos);
    const n = node("img", [["src", textArg(args, "صورة", pos)], ["alt", "وصف" in o ? optText(o["وصف"], pos, "صورة", "وصف") : ""]]);
    if ("عرض" in o) n.attrs.push(["width", String(intValue(o["عرض"], pos, "عرض"))]);
    emit(n);
    return null;
  });
  element("رابط", false, (args, named, pos) => {
    options("رابط", named, pos);
    if (args.length !== 2) fail(pos, "«رابط» يأخذ النص والعنوان، مثل: رابط(\"الموقع\"، \"https://…\")");
    emit(node("a", [["href", optText(args[1], pos, "رابط", "العنوان")]], textOf(args[0])));
    return null;
  });
  element("فاصل", false, (args, named, pos) => { options("فاصل", named, pos); emit(node("hr")); return null; });

  const voidTags = new Set(["input", "img", "hr", "br"]);
  const esc = (s) => s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&#34;", "'": "&#39;" }[c]));
  function html(n) {
    let s = "<" + n.tag;
    for (const [k, v] of n.attrs) s += " " + k + (v !== "" || k === "value" || k === "alt" ? '="' + esc(v) + '"' : "");
    s += ">";
    if (voidTags.has(n.tag)) return s;
    return s + esc(n.text) + n.kids.map(html).join("") + "</" + n.tag + ">";
  }

  function newSession() { return { version: 0, states: {}, handlers: {} }; }

  function render(sess, root) {
    const top = node("div", [["class", "dad-app"]]);
    ui = { sess, stack: [top], frames: [{ key: "", counts: {} }], handlers: {}, nextID: 0, seen: {} };
    try { call(root, [], [], [0, 0], root.name); } finally { var done = ui; ui = null; }
    for (const k of Object.keys(sess.states)) if (!done.seen[k]) delete sess.states[k];
    sess.handlers = done.handlers;
    sess.version++;
    return html(top);
  }

  function fieldValue(old, s) {
    if (isInt(old)) { const t = s.trim(); if (/^[+-]?\d+$/.test(t)) return BigInt(t); }
    if (isFloat(old)) { const f = parseNumber(s); if (f !== null) return f; }
    return s;
  }

  function event(sess, version, id, value) {
    if (version !== sess.version) return;
    const h = sess.handlers[id];
    if (!h) return;
    if (h.kind === "click") { if (h.fn) call(h.fn, [], [], [0, 0], "زر"); }
    else if (h.kind === "input") {
      let v = String(value);
      if (h.cell) { v = fieldValue(h.cell.val, v); h.cell.val = v; }
      if (h.fn) call(h.fn, [v], [], [0, 0], "حقل");
    } else if (h.kind === "toggle") {
      if (h.cell) h.cell.val = !!value;
      if (h.fn) call(h.fn, [!!value], [], [0, 0], "مربع_اختيار");
    }
  }

  // mount shows the main component in #dad-root and handles its events in the page.
  function mount(root) {
    if (typeof localStorage !== "undefined") {
      const key = "dad:" + (document.title || "app");
      try { const saved = JSON.parse(localStorage.getItem(key)); if (saved && saved.tables) store.tables = saved.tables; } catch (e) {}
      store.persist = () => { try { localStorage.setItem(key, JSON.stringify({ tables: store.tables })); } catch (e) {} };
    }
    const el = document.getElementById("dad-root");
    const sess = newSession();
    const show = (fn) => {
      try { return { html: fn(), version: sess.version }; }
      catch (e) { return { error: e instanceof Thrown ? "مشكلة لم تُعالَج: " + problemMessage(e.val) : e.message }; }
    };
    const first = show(() => render(sess, root));
    el.innerHTML = first.html || "";
    el.setAttribute("data-version", String(sess.version));
    if (first.error) { const box = document.getElementById("dad-error"); box.textContent = first.error; box.hidden = false; }
    root$ = root;
    window.dadSend = (e) => {
      const r = show(() => { event(sess, e.v, e.id, e.value); return render(sess, root); });
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
    setOutput(f) { out = f; },
  };
  if (typeof module !== "undefined" && module.exports) module.exports = $;
  else root.$dad = $;
})(typeof globalThis !== "undefined" ? globalThis : this);
