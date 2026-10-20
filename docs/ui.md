# Interface

A Ḍād program can have an interface: pages made of components (`مكون`) that keep state
(`حالة`) and lay out elements such as `عمود`, `زر` and `حقل`. Pages are right to left and
in Arabic type from the start. Run the program with `dad dev` and open
<http://localhost:8080>, or try it in the [playground](playground.md).

```dad
مكون الرئيسية {
    حالة ع = 0
    عمود {
        عنوان("العدد: {ع}")
        زر("زد") { ع += 1 }
    }
}
```

Click **زد** and the heading changes. Nothing else is needed: no HTML, no JavaScript, no
build step. When the page is ready to share, `dad build` turns it into a static site that
needs no server (see [Building for the web](build.md)).

## Components — `مكون`

A component is a function that lays out part of a page. It can take parameters, like any
function, and use other components:

```dad
مكون عداد(اسم: نص، بداية: صحيح = 0) {
    حالة ع = بداية
    بطاقة {
        عنوان("{اسم}: {ع}"، مستوى: 2)
        صف {
            زر("زد") { ع += 1 }
            زر("أنقص"، معطل: ع == 0) { ع -= 1 }
        }
    }
}

مكون الرئيسية {
    عمود {
        عداد("التفاح")
        عداد("البرتقال"، بداية: 5)
    }
}
```

The page shows the component named `الرئيسية` (or `التطبيق`). If there is none, it shows the
last component that needs no arguments.

Inside a component, ordinary code works as everywhere else: `إذا` shows elements only when
something is true, and `لكل` repeats them.

```dad
إذا اسم != "" { فقرة("أهلاً يا {اسم}") }
لكل م في مهام { فقرة(م.عنوان) }
```

## State — `حالة`

`حالة ع = 0` is a variable that keeps its value while the page is open. The value after `=`
is only used the first time. Change it anywhere, in a button's block for example, and the
page shows the new value.

Each copy of a component has its own state: the two counters above count separately.
State lasts as long as the page is open; to keep data for longer, save it in a
[table](server.md#tables).

## Elements

| Element | What it shows | Example |
|---|---|---|
| `عمود` | Its children, top to bottom | `عمود { … }` |
| `صف` | Its children side by side, from the right | `صف { … }` |
| `بطاقة` | A boxed group | `بطاقة { … }` |
| `عنوان` | A heading; `مستوى: 2` for a smaller one | `عنوان("مهامي")` |
| `فقرة` | A paragraph | `فقرة("مرحباً")` |
| `زر` | A button; its block runs when it is clicked | `زر("أضف") { … }` |
| `حقل` | A text field that edits a state | `حقل(اسم، تلميح: "اسمك")` |
| `مربع_اختيار` | A check box | `مربع_اختيار(م.منجزة) { … }` |
| `صورة` | An image | `صورة("شعار.png"، وصف: "الشعار")` |
| `رابط` | A link | `رابط("الموقع"، "https://…")` |
| `فاصل` | A line between sections | `فاصل()` |

Options are named arguments:

| Element | Options |
|---|---|
| `عمود`, `صف` | `محاذاة: "وسط"` (or `"بداية"`, `"نهاية"`), `مسافة: 24` (pixels) |
| `عنوان` | `مستوى: 1` to `6` |
| `زر` | `معطل: صح` |
| `حقل` | `تلميح: "…"`, `صيغة: "رقم"` (or `"نص"`, `"كلمة_سر"`, `"بريد"`), `معطل: صح` |
| `مربع_اختيار` | `نص: "…"` (a label next to it), `معطل: صح` |
| `صورة` | `وصف: "…"` (for screen readers), `عرض: 120` |

### Fields

Give `حقل` a state and it edits it: what you type goes into the state, and changing the
state changes the field.

```dad
حالة اسم = ""
حقل(اسم، تلميح: "ما اسمك؟")
فقرة("أهلاً يا {اسم}")
```

A field for a number state (`حالة خطوة = 1`) is a number field, and its state stays a
number. A block after the field runs on each change, with the new text as `هو`.
Pressing <kbd>Enter</kbd> in a field presses the button next to it.

### Check boxes

`مربع_اختيار` takes صح or خطأ. Given a state, it changes it. Given another value, such as a
record's field, its block runs with the new value as `هو`:

```dad
حالة مفعل = خطأ
مربع_اختيار(مفعل، نص: "الإشعارات")

مربع_اختيار(م.منجزة) { بدل(م) }
```

## With a server and data

Components can call [server functions and tables](server.md) directly. This is the whole
to-do app in `examples/05-todo.dad`: a table, three server functions, an API route and the
page.

```dad
--8<-- "examples/05-todo.dad"
```

## How it runs

For now, components run on the server, inside `dad dev`. The page in the browser sends each
click and change to the program, the program runs the block and lays out the page again,
and the page updates in place, keeping focus and what you are typing. Phoenix LiveView and
Blazor Server work this way. A later version compiles components to JavaScript so they run
in the browser; programs stay the same.

The look comes from a built-in stylesheet: right to left, Noto Kufi Arabic, light and dark
themes that follow the system setting.

When you save the file, `dad dev` reloads the program and the page reloads with it. Data in
tables is kept; state starts again. Errors in a block appear at the bottom of the page.
