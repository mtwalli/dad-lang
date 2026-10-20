# Language specification

> Status: **draft 1**, describing Ḍād 0.2. Parts marked *Planned* are designed but not built
> yet. Source files use the extension `.dad`. The command is `dad`, with the Arabic name `ضاد`.

## 1. Design principles

1. **Less code, no ceremony.** Type inference everywhere, no semicolons, expression-bodied
   functions, trailing lambdas, string interpolation, default and named arguments.
   (Inspired by Kotlin and Swift.)
2. **Precise and safe by default.** Immutable by default (`ثابت`), no `null` surprises
   (optional types `نص؟`), exhaustive `طابق` (match), no implicit numeric conversions,
   no force-unwrap operator.
3. **Everything is an expression.** `إذا`, `طابق` and blocks produce values.
4. **One language for the whole app.** UI, server and data live in the same file and the
   compiler splits them. Calling a `خادم` (server) function from the UI just works — the
   compiler generates the HTTP endpoint and the client call.
5. **No async/await noise.** Server and database calls are asynchronous under the hood;
   the compiler infers this and inserts the waiting for you.
6. **Small core.** 39 reserved keywords (Go has 25, Kotlin ~80, Swift ~90). UI elements and
   HTTP verbs are ordinary library functions / contextual words, not keywords.
7. **Arabic first, not Arabic only.** Arabic keywords and punctuation; Latin punctuation and
   identifiers are also accepted so mixed code never breaks. Numbers use one set of digits,
   0-9, in code and in output (see 2.5).

## 2. Lexical structure

### 2.1 Source text
- UTF-8, normalized to **NFC** before lexing.
- Newlines end statements. `؛` or `;` may separate statements on one line.

### 2.2 Normalization of identifiers and keywords
Applied to identifiers and keywords only — **never** inside strings or comments:

| Rule | Example |
|---|---|
| Diacritics (tashkeel U+064B–U+065F, U+0670) are ignored | `حوِّل` ≡ `حول` |
| Tatweel `ـ` (U+0640) is ignored | `دالـــة` ≡ `دالة` |
| Hamza/alef forms are **not** folded (`أ` ≠ `ا` ≠ `إ`) — errors suggest the near-miss | `اذا` → "هل تقصد `إذا`؟" |
| `ة`/`ه` and `ى`/`ي` are **not** folded (same: suggestion only) | |

### 2.3 Forbidden characters
Bidirectional control characters (U+200E, U+200F, U+202A–U+202E, U+2066–U+2069) and
zero-width characters (U+200B–U+200D, U+FEFF) are a **compile error outside string literals**.
This prevents "Trojan Source" attacks where code displays differently from how it runs.

### 2.4 Identifiers
Start with an Arabic or Latin letter or `_`; continue with letters, digits `0-9` or `_`.
`عدد_المهام`, `userName`, `س2` are all valid.

### 2.5 Numbers
- Digits are `0123456789` only, in code and in output. `.` separates the fraction and `_` may
  group digits: `42`, `3.14`, `1_000_000`.
- Arabic-Indic (`٠-٩`) and Persian (`۰-۹`) digits, the Arabic decimal sign `٫` and the
  thousands sign `٬` are errors in code. The message shows the number written correctly:
  `«١٢» تُكتب «12»`.
- `1,5` is two values. The lexer warns and suggests `1.5`.
- Output uses `0-9` and `.`: `اطبع(7 / 2.0)` prints `3.5`.
- Text is free: `"الطابق ٣"` is fine. Converting text to a number (`"٤٢" ك صحيح`) accepts
  Arabic-Indic and Persian digits too, so input typed on an Arabic keyboard still works.

### 2.6 Strings
- `"..."` or Arabic guillemets `«...»`.
- Interpolation with braces, no prefix needed: `"مرحبا {اسم}، لديك {مهام.عدد} مهام"`.
- Escapes: `\n \t \\ \" \{ \}`. Multi-line: `"""..."""`.

### 2.7 Comments
`// سطر` and `/* كتلة */`.

### 2.8 Punctuation & operators
Arabic and Latin forms are interchangeable (digits are the exception: always `0-9`):

| Arabic | Latin | Meaning |
|---|---|---|
| `،` | `,` | separator |
| `؛` | `;` | statement separator (optional) |
| `؟` | `?` | optional type marker |
| `؟.` | `?.` | safe member access |
| `؟؟` | `??` | default when absent |

Other operators: `+ - * / % == != < > <= >= = += -= *= /= => . .. :`
Logical: `و` / `&&`, `أو` / `||`, `ليس` / `!`.

`/` always gives a decimal, as in school maths: `7 / 2` is `3.5` and `6 / 2` is `3.0`. For a
whole number, convert the result: `(7 / 2) ك صحيح` is `3` (the fraction is dropped). `%` is the
remainder of whole numbers: `7 % 2` is `1`.

`و` and `ك` are separate words in Ḍād, even though Arabic writing joins them to the next word.
Write `أ و ب` and `س ك عشري`. A joined form such as `وب` is read as a name; when that is an
error, the message suggests the fix: `هل تقصد «و ب»؟`.

> RTL note: `<`, `>`, `(`, `)`, `[`, `]`, `{`, `}` are Unicode *mirrored* characters — in an RTL
> line they display flipped, so `أ < ب` still reads correctly right-to-left. Editor support must
> be validated early (Phase 4).

## 3. Keywords (reserved)

| Ḍād | Kotlin / Swift equivalent | |
|---|---|---|
| `ثابت` | `val` / `let` | immutable binding |
| `متغير` | `var` | mutable binding |
| `دالة` | `fun` / `func` | function |
| `نوع` | `data class` / `struct` | value type with auto equality & copy |
| `تعداد` | `sealed class` / `enum` | enum with associated values |
| `عقد` | `interface` / `protocol` | contract |
| `امتداد` | extension functions / `extension` | add methods to an existing type |
| `استورد` | `import` | |
| `خاص` | `private` | (everything else is public) |
| `إذا` / `وإلا` | `if` / `else` | expression |
| `طابق` | `when` / `switch` | exhaustive pattern match |
| `لكل` / `في` | `for` / `in` | |
| `من` / `إلى` / `حتى` / `بخطوة` | `1..10` / `1 until 10` / `step` | ranges: `من 1 إلى 10` |
| `طالما` | `while` | |
| `أرجع` | `return` | |
| `توقف` / `تابع` | `break` / `continue` | |
| `حاول` / `التقط` / `ارفع` | `try` / `catch` / `throw` | |
| `صح` / `خطأ` | `true` / `false` | |
| `عدم` | `null` / `nil` | |
| `و` / `أو` / `ليس` | `&&` / `||` / `!` | |
| `هو` | `it` | the implicit lambda parameter: `{ هو * 2 }` |
| `من نوع` | `is` | type test: `ش من نوع دائرة` (see 4.7) |
| `ك` | `as` | conversion |
| `هذا` | `this` / `self` | |
| `خادم` | — | marks a function that runs on the server |
| `مكون` | `@Composable` / SwiftUI `View` | UI component |
| `حالة` | `@State` / `mutableStateOf` | reactive state inside a component |
| `جدول` | — | persisted data type (database table) |
| `مسار` | — | explicit HTTP route (public APIs) |

**Contextual words** (only special in one position, usable as names elsewhere): HTTP verbs
`جلب` (GET), `أرسل` (POST), `عدل` (PUT), `احذف` (DELETE).

## 4. Language tour

### 4.1 Bindings and types
```dad
ثابت اسم = "سارة" // type inferred: نص
متغير عداد = 0 // صحيح
ثابت سعر: عشري = 9.99
عداد += 1
```
Built-in types: `صحيح` (Int, 64-bit), `عشري` (Double), `نص` (String), `منطقي` (Bool),
lists `[نص]`, maps `[نص: صحيح]`, optionals `نص؟`, functions `(صحيح) => نص`.

### 4.2 Functions
```dad
دالة مربع(س: صحيح) = س * س // expression body, return type inferred

دالة حي(اسم: نص، تحية: نص = "مرحبا"): نص {
    أرجع "{تحية} يا {اسم}"
}

حي("سارة")
حي(اسم: "علي"، تحية: "أهلاً") // named arguments
```

### 4.3 Control flow as expressions
```dad
ثابت وصف = إذا درجة >= 90 { "ممتاز" } وإلا إذا درجة >= 50 { "ناجح" } وإلا { "راسب" }

لكل ع من 1 إلى 10 { اطبع(ع) }
لكل ع من 0 حتى 10 بخطوة 2 { اطبع(ع) } // 0,2,4,6,8
لكل اسم في أسماء { اطبع(اسم) }
```

### 4.4 Lambdas and trailing lambdas
```dad
ثابت كبار = أرقام.رشح { هو > 10 } // `هو` = the implicit single parameter ("it")
ثابت مضاعفة = أرقام.حول { س => س * 2 }
ثابت مجموع = أرقام.اختزل(0) { أ، ب => أ + ب }
```

### 4.5 Value types (`نوع`)
```dad
نوع نقطة(س: عشري، ص: عشري) {
    دالة بعد() = جذر(س * س + ص * ص)
}

ثابت ن = نقطة(س: 3، ص: 4)
ثابت م = ن.نسخة(ص: 0) // copy with change
اطبع(ن == نقطة(3، 4)) // صح — structural equality
```

### 4.6 Enums with data and exhaustive matching
```dad
تعداد شكل {
    دائرة(قطر: عشري)
    مستطيل(عرض: عشري، طول: عشري)
}

دالة مساحة(ش: شكل) = طابق ش {
    دائرة(ق) => 3.14 * ق * ق / 4
    مستطيل(ع، ط) => ع * ط
} // forgetting a case is a compile error
```

### 4.7 Null safety and smart casts
```dad
ثابت بريد: نص؟ = مستخدم؟.بريد
ثابت عرض = بريد ؟؟ "لا يوجد بريد"

إذا بريد != عدم {
    اطبع(بريد.طول) // smart cast: بريد is نص here
}

إذا ش من نوع دائرة { اطبع(ش.قطر) } // type test + smart cast
```
There is no force-unwrap. Use `؟؟` with a default, or `؟؟ ارفع مشكلة("...")`.

### 4.8 Contracts and extensions
```dad
عقد قابل_للعرض {
    دالة اعرض(): نص
}

امتداد نص {
    دالة مكرر(مرات: صحيح) = (من 1 إلى مرات).حول { هذا }.ضم("")
}
```

### 4.9 Errors
```dad
دالة اقسم(أ: صحيح، ب: صحيح): عشري {
    إذا ب == 0 { ارفع مشكلة("قسمة على صفر") }
    أرجع أ / ب
}

ثابت ن = حاول { اقسم(1، 0) } التقط خ { 0 } // try is an expression too
```

### 4.10 Modules
One file = one module. Everything not marked `خاص` is exported.
```dad
استورد نقطة، شكل من "./أشكال"
```
*(Implemented.)* The file is found next to the importing file (`.dad` may be left out).
Each file runs once, after the files it imports, in a scope of its own. Importing a `تعداد`
brings its cases. An imported name is a constant holding the value the other file gave it.
Imports may not form a circle. Types, tables, enums and their cases, and server functions are
known by name across the program, so they may not be declared in two files. See
[Programs in several files](modules.md).

### 4.11 Checking before running
Programs are checked before they run, and a program with errors doesn't start (`dad run`,
`dad dev`, `dad serve`, `dad build`, the playground). `dad check` and the editors show the same messages as you type.
The checker reports only what is certain; where it can't know a value's type, it accepts the
program and the interpreter checks at run time.

It reports:
- names that aren't defined (with the closest defined name: «هل تقصد …؟»), a name used
  before a top-level `ثابت` defines it, and names declared twice in one scope;
- changing a `ثابت`, a parameter or a field (`نسخة` makes a changed copy);
- calls with too many or too few arguments, an unknown named argument, a named argument
  before a positional one, or an argument of the wrong type;
- values of the wrong type for a declared type, a result type or a returned value, and a
  block-bodied function with a result type and no `أرجع`;
- fields and methods that a type doesn't have, on records, tables, text, lists, maps and
  numbers (with the closest name);
- `طابق` over a `تعداد` that misses cases and has no `_` (exhaustiveness), and patterns that
  name unknown cases or too many fields;
- conditions that aren't `منطقي`, operators on the wrong types (`"أ" - 1`), `لكل` over a
  value that can't be looped over, and indexes that aren't `صحيح`;
- `أرجع`, `توقف`, `تابع`, `هو` and `هذا` used where they mean nothing; `حالة` and interface
  elements used at the top level instead of in a component; unknown element options.

Types come from declarations and literals: `ثابت س = 1` is `صحيح`, a function's result is its
declared type or the type of its `= expression` body, a table field without a type takes its
default's type. A `صحيح` is accepted where an `عشري` is expected. Optional values (`نص؟`)
are not yet followed through `إذا س != عدم` checks (smart casts), so they are not reported.

## 5. The full-stack layer

### 5.1 Data — `جدول`
```dad
جدول مهمة {
    عنوان: نص
    منجزة = خطأ
}
```
A table is a value type with a hidden id field `رقم` (`صحيح`, 0 until the record is saved;
declaring a field with that name is an error). Every field needs a type or a default.
Its API: `مهمة.أنشئ(...)`, `مهمة.الكل()`, `مهمة.حيث { هو.منجزة }`, `مهمة.بالرقم(ر)`,
`مهمة.عدد()`, `م.احفظ()`, `م.احذف()`. `م.نسخة(...)` keeps the id, so a changed copy saves
over the original.

A field with a literal default and no type takes the default's type (`منجزة = خطأ` is `منطقي`).

*Implemented:* `dad run`, `dad dev` and `dad serve` store tables in SQLite (`.dad/<program>.db`), one SQL
table per `جدول` and one column per field. Migrations follow the declaration: added fields are
filled with their default, removed fields are hidden but kept, and type changes convert every
value or are refused. `حيث` lambdas that compare fields with values or variables (joined by
`و`، `أو`، `ليس`) are compiled to SQL; other lambdas run on every row with the same result.

### 5.2 Server — `خادم`
```dad
خادم دالة أضف_مهمة(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)
```
Callable directly from UI code. The compiler generates a typed endpoint plus the client call;
the server code never ships to the browser. Server functions are implicitly asynchronous.

*Implemented:* `dad dev` and `dad serve` serve each server function at `POST /_dad/call/<name>`, with the
arguments as a JSON object keyed by parameter name. Arguments are converted to the
parameter types; errors return 400 with `{"خطأ": "…"}`.

### 5.3 Explicit routes — `مسار`
For public APIs that other programs call:
```dad
مسار جلب "/api/مهام" = مهمة.الكل()
مسار جلب "/api/مهام/{رقم}" = مهمة.بالرقم(رقم)

مسار أرسل "/api/مهام" {
    أرجع مهمة.أنشئ(عنوان: البيانات["عنوان"])
}
```
Methods: `جلب` GET, `أرسل` POST, `عدل` PUT, `احذف` DELETE. The path starts with `/`.
`{name}` segments are bound as variables (digits become `صحيح`); the JSON request body is
`البيانات`. The body is an expression or a block; the result is sent as JSON, and a `جلب`
route that returns `عدم` answers 404. See the [server guide](server.md).

### 5.4 UI — `مكون` and `حالة`
Compose/SwiftUI style: components are functions; UI elements (`عمود`, `صف`, `عنوان`,
`فقرة`, `زر`, `حقل`, `صورة`, `رابط`) are library functions with trailing lambdas.
Pages are right-to-left by default.
```dad
مكون عداد() {
    حالة ع = 0
    عمود {
        عنوان("العدد: {ع}")
        زر("زد") { ع += 1 }
    }
}
```
A component is called like a function from another component; it adds its elements where it
is called. Elements with a block (`عمود`, `صف`, `بطاقة`) put the elements laid out in the block
inside them. `حالة` is only allowed in a component: its initial value is evaluated on the
first render, and each instance of a component (by its place in the page) has its own state.
An element given a state variable as its first argument (`حقل(اسم)`, `مربع_اختيار(مفعل)`)
receives the variable itself and changes it. The page shows the component named `الرئيسية`
or `التطبيق`, otherwise the last one that needs no arguments.

*Implemented:* in `dad dev` and `dad serve`, components run in the interpreter on the server;
the page sends events and updates in place, LiveView style. After `dad build`, the same
components are compiled to JavaScript and run in the browser. Elements: `عمود`، `صف`، `بطاقة`،
`عنوان`، `فقرة`، `زر`، `حقل`، `مربع_اختيار`، `صورة`، `رابط`، `فاصل` (see the
[interface guide](ui.md)). *Planned:* a reactive runtime that updates only what changed.

## 6. Open questions (please decide)
1. ~~`هو` doing double duty~~ Decided: `هو` only means "it" in lambdas; the type test is
   `من نوع` (`ش من نوع دائرة`). Writing `ش هو دائرة` is an error that shows the new form.
2. Keyword spellings: `نوع` vs `بنية` for value types; `عقد` vs `واجهة` for contracts.
3. Library method names (`حول`/map, `رشح`/filter, `اختزل`/reduce) — want a naming guide?
4. Should Latin-letter identifiers produce a style warning in Arabic-first projects?
5. ~~Integer division~~ Decided: `/` always gives a decimal (`7 / 2` is `3.5`), as in school maths. `(7 / 2) ك صحيح` gives `3`.
6. ~~Digits~~ Decided: `0-9` only, in code and in output (see 2.5).
7. ~~`و` and `ك` joined to the next word~~ Decided: they stay separate words; a joined form
   such as `وب` is an error that suggests `و ب` (see 2.8).
