---
icon: lucide/layout-panel-top
---

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

A program without [pages](#pages) shows the component named `الرئيسية` (or `التطبيق`). If
there is none, it shows the last component that needs no arguments.

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

## Pages — `صفحة` { #pages }

A program with more than one screen gives each one an address with `صفحة` («page»):

```dad
صفحة "/" = قائمة_المهام()
صفحة "/عن" = عن_التطبيق()
صفحة "/مهمة/{رقم}" = صفحة_المهمة(رقم)
```

After the address comes what the page shows, usually a component. It can also be a block,
written like a component's body and with its own `حالة`:

```dad
صفحة "/عن" {
    الشريط()
    عنوان("عن التطبيق")
    فقرة("مثال على برنامج من عدة صفحات.")
}
```

- **Values in the address.** `{رقم}` takes its value from the address: `/مهمة/3` shows
  `صفحة_المهمة(3)`. Digits become a whole number (`صحيح`), anything else is text.
- **Links.** A link to one of the pages, such as `رابط("عن التطبيق"، "/عن")`, opens it in
  place, without reloading. The back and forward buttons work, and an address can be
  bookmarked or shared.
- **The current page.** The link to the page on screen is marked (bold and underlined in the
  built-in style), so a row of links makes a navigation bar. Put it in a component and call
  it at the top of each page:

    ```dad
    مكون الشريط() {
        صف {
            رابط("المهام"، "/")
            رابط("عن التطبيق"، "/عن")
        }
        فاصل()
    }
    ```

- **Moving from code.** `انتقل("/المهام")` («go to») moves to a page from a button's block,
  after saving for example. `انتقل("https://…")` leaves for another site.

    ```dad
    زر("أضف") {
        ثابت م = أضف(الجديدة)
        انتقل("/مهمة/{م.رقم}")
    }
    ```

- **The title.** The browser tab shows the page's first `عنوان`.
- **No page there.** An address with no page shows a page that says so, with a link to the
  home page. `dad serve` answers it with status 404.
- **State.** Each page starts with fresh state when it opens. What has to last between pages
  belongs in a [table](server.md#tables).

Two pages can't share an address, and neither can a page and a `مسار جلب` route: the checker
says so. `صفحة` has this meaning only before an address, so it can still be the name of a
variable. A program without `صفحة` shows its main component at `/`, as before.

Here is a whole program with pages, `examples/08-pages.dad`:

```dad
--8<-- "examples/08-pages.dad"
```

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
| `رابط` | A link; to one of the [pages](#pages), it opens in place | `رابط("الموقع"، "https://…")` |
| `فاصل` | A line between sections | `فاصل()` |
| `قائمة_منسدلة` | A drop-down list that chooses a state's value | `قائمة_منسدلة(اللون، ["أحمر"، "أخضر"])` |
| `اختيار` | The same, as radio buttons | `اختيار(اللون، ["أحمر"، "أخضر"])` |
| `منطقة_نص` | A field for longer text | `منطقة_نص(الملاحظات)` |
| `جدول_عرض` | Records as a table | `جدول_عرض(مهمة.الكل())` |
| `قائمة_عناصر` | A bulleted or numbered list | `قائمة_عناصر(["أ"، "ب"])` |
| `ألسنة`, `لسان` | Tabs | `ألسنة { لسان("الأول") { … } }` |
| `نافذة` | A dialog over the page | `نافذة(مفتوحة) { … }` |
| `تنبيه` | A message that stands out | `تنبيه("تم الحفظ"، نمط: "نجاح")` |
| `أيقونة` | An icon | `أيقونة("بحث")` |

Options are named arguments:

| Element | Options |
|---|---|
| `عمود`, `صف`, `بطاقة` | `محاذاة: "وسط"` (or `"بداية"`, `"نهاية"`), `مسافة: 24` (pixels) |
| `عنوان` | `مستوى: 1` to `6` |
| `زر` | `معطل: صح`, `أيقونة: "حفظ"` |
| `حقل` | `تلميح: "…"`, `صيغة: "رقم"` (or `"نص"`, `"كلمة_سر"`, `"بريد"`, `"تاريخ"`, `"وقت"`), `أدنى` and `أقصى` (for numbers and dates), `معطل: صح` |
| `مربع_اختيار` | `نص: "…"` (a label next to it), `معطل: صح` |
| `قائمة_منسدلة` | `تلميح: "…"` (shown before a choice is made), `معطل: صح` |
| `اختيار` | `معطل: صح` |
| `منطقة_نص` | `تلميح: "…"`, `أسطر: 6`, `معطل: صح` |
| `جدول_عرض` | `أعمدة: ["…"]`, `فارغ: "…"` (shown when there are no rows) |
| `قائمة_عناصر` | `مرقمة: صح`, `فارغ: "…"` |
| `لسان` | `أيقونة: "…"` |
| `نافذة` | `عنوان: "…"` |
| `تنبيه` | `نمط: "نجاح"` (or `"معلومة"`, `"تحذير"`, `"خطأ"`) |
| `أيقونة` | `وصف: "…"` (for screen readers) |
| `صورة` | `وصف: "…"` (for screen readers), `عرض: 120` |
| `رابط` | `أيقونة: "…"` |

Other options change how an element looks: `نمط`, `حجم`, `لون`, `خلفية` and more, and `صنف`
for your own stylesheet. They are explained in [Customizing the look](styling.md).

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

### Choices — `قائمة_منسدلة` and `اختيار`

Both choose a state's value from a list. `قائمة_منسدلة` is a drop-down list, and `اختيار`
shows the choices as radio buttons, which suits two to five of them:

```dad
حالة اللون = "أخضر"
قائمة_منسدلة(اللون، ["أحمر"، "أخضر"، "أزرق"])
اختيار(اللون، ["أحمر"، "أخضر"، "أزرق"])
فقرة("اخترت {اللون}")
```

The choices can be any values, not only text: numbers, the cases of a `تعداد`, or records.
A case shows its name and a record its first text field, and the state gets the value
itself:

```dad
تعداد أولوية {
    منخفضة
    عادية
    عاجلة
}

حالة الأولوية = عادية
اختيار(الأولوية، [منخفضة، عادية، عاجلة])
```

As with `مربع_اختيار`, a block after the list runs with the new choice as `هو`.

### Longer text, dates and times

`منطقة_نص` is a field for text over several lines, such as notes. `حقل` with
`صيغة: "تاريخ"` or `"وقت"` shows the browser's date or time picker; the state holds the
value as text, such as `"2026-10-21"` or `"09:30"`.

```dad
حالة الملاحظات = ""
حالة الموعد = ""
منطقة_نص(الملاحظات، أسطر: 5، تلميح: "ملاحظات…")
حقل(الموعد، صيغة: "تاريخ"، أدنى: "2026-01-01")
```

## Tables and lists

`جدول_عرض` shows a list of records as a table, with a column for each field. `أعمدة` chooses
the fields and their order; an underscore in a name shows as a space. A block after it adds
a last column with elements for each row, such as buttons:

```dad
جدول_عرض(مهمة.الكل()، أعمدة: ["العنوان"، "منجزة"]، فارغ: "لا مهام بعد") { م =>
    زر("احذف"، نمط: "خفيف"، أيقونة: "حذف") { احذف(م) }
}
```

صح shows as ✓, and خطأ and عدم as nothing. A wide table scrolls sideways on a phone.

`قائمة_عناصر` shows a bulleted list, or a numbered one with `مرقمة: صح`. Without a block, each
item shows as text; with one, the block lays out each item:

```dad
قائمة_عناصر(["الحليب"، "الخبز"، "التمر"])

قائمة_عناصر(مهمة.الكل()، مرقمة: صح) { م =>
    رابط(م.العنوان، "/مهمة/{م.رقم}")
}
```

Both show `فارغ` (or «لا شيء لعرضه بعد.») when the list is empty.

## Tabs — `ألسنة`

Inside `ألسنة`, each `لسان` has a title and the elements it shows. Only the open tab is laid
out. The open tab is kept like a `حالة`; give a state as the first argument to choose it from
code, either by number (from 0) or by title:

```dad
حالة اللسان = "الإعدادات"
ألسنة(اللسان) {
    لسان("الحساب"، أيقونة: "مستخدم") {
        فقرة("بياناتك")
    }
    لسان("الإعدادات"، أيقونة: "إعدادات") {
        فقرة("الإعدادات")
    }
}
```

The arrow keys move between tabs.

## Dialogs — `نافذة`

`نافذة` shows its elements over the page while its state is صح. The × button, <kbd>Esc</kbd>
and a click outside it set the state back to خطأ:

```dad
حالة مفتوحة = خطأ
زر("احذف الكل"، نمط: "خطر") { مفتوحة = صح }
نافذة(مفتوحة، عنوان: "هل أنت متأكد؟") {
    فقرة("ستُحذف كل المهام.")
    صف {
        زر("احذف"، نمط: "خطر") {
            امسح_الكل()
            مفتوحة = خطأ
        }
        زر("إلغاء"، نمط: "ثانوي") { مفتوحة = خطأ }
    }
}
```

## Alerts and icons

`تنبيه` is a message that stands out, with an icon for its kind: `"معلومة"` (the default),
`"نجاح"`, `"تحذير"` or `"خطأ"`. Screen readers announce warnings and errors at once.

```dad
تنبيه("حُفظت التغييرات."، نمط: "نجاح")
تنبيه("ستنتهي الجلسة بعد دقيقة."، نمط: "تحذير")
```

`أيقونة` shows an icon by its Arabic name. Buttons, links and tabs take one too, before
their text, with the option `أيقونة`. An icon on its own is decoration; give it `وصف` when it
carries meaning, so screen readers can say it.

```dad
أيقونة("بحث"، وصف: "بحث")
زر("احفظ"، أيقونة: "حفظ") { احفظ() }
رابط("الإعدادات"، "/الإعدادات"، أيقونة: "إعدادات")
```

These are the icons, from [Lucide](https://lucide.dev):

--8<-- "tools/icons/gallery.html"

## With a server and data

Components can call [server functions and tables](server.md) directly. This is the whole
to-do app in `examples/05-todo.dad`: a table, three server functions, an API route and the
page.

```dad
--8<-- "examples/05-todo.dad"
```

## How it runs

With `dad dev` and `dad serve`, components run on the server. The page in the browser sends
each click and change to the program, the program runs the block and lays out the page
again, and the page updates in place, keeping focus and what you are typing. Phoenix
LiveView and Blazor Server work this way. Links to pages work the same way: the program lays
out the new page, and the browser shows its address. With [`dad build`](build.md), the same
components are compiled to JavaScript and run in the browser, with no server, and pages have
addresses such as `index.html#/عن`. The program stays the same either way.

The look comes from a built-in stylesheet: right to left, Noto Kufi Arabic, light and dark
themes that follow the system setting. You can change it, for one element or for the whole
app: see [Customizing the look](styling.md).

When you save the file, `dad dev` reloads the program and the page reloads with it. Data in
tables is kept; state starts again. Errors in a block appear at the bottom of the page.
