---
icon: lucide/palette
---

# Customizing the look

Every app starts with the built-in look: right to left, Kufi type, a lapis colour, and light
and dark themes that follow the device. You can change it in three ways, from the smallest
to the largest:

1. **Options on an element** change one button or one card.
2. **A theme**, `سمة`, changes the colour, the type and the corners of the whole app.
3. **Your own stylesheet** does anything CSS can do, on the elements you give a `صنف`.

All three work the same in `dad dev`, `dad serve`, `dad build` and the
[playground](playground.md), except the stylesheet, which needs a file.

## Options on elements

| Option | Elements | What it takes |
|---|---|---|
| `نمط` | `زر`, `رابط` | `"أساسي"` (filled, the default for buttons), `"ثانوي"` (outlined), `"خفيف"` (no frame), `"خطر"` (red). Any style makes a link look like a button |
| `حجم` | `زر`, `رابط`, `فقرة`, `أيقونة` | `"صغير"`, `"عادي"`, `"كبير"` |
| `ملء` | `زر` | `صح`: as wide as the element around it |
| `لون` | `عنوان`, `فقرة`, `أيقونة` | a colour, for the text |
| `خلفية` | `عمود`, `صف`, `بطاقة` | a colour, as a light tint behind the elements |
| `حشو` | `عمود`, `صف`, `بطاقة` | the space inside, in pixels |
| `عرض` | `عمود`, `صف`, `بطاقة` | the largest width, in pixels; the element is centred |
| `محاذاة` | `عنوان`, `فقرة` | `"بداية"`, `"وسط"`, `"نهاية"` |
| `صنف` | every element | class names for [your own stylesheet](#your-own-stylesheet) |

```dad
بطاقة(خلفية: "ذهبي"، حشو: 24، عرض: 520) {
    عنوان("اشترك في النشرة"، محاذاة: "وسط")
    فقرة("يصلك كل جديد، مرة في الشهر."، لون: "باهت"، محاذاة: "وسط")
    زر("اشترك"، حجم: "كبير"، ملء: صح، أيقونة: "بريد") { اشترك() }
    زر("لاحقاً"، نمط: "خفيف") { أخف() }
}
```

### Colours

A colour is a name, or a value such as `"#1b2b6b"`. `"أساسي"` is the app's own colour (see
the theme below) and `"باهت"` the colour of quiet text. These are the other names:

<div class="dad-swatches" dir="rtl" lang="ar"><span><i style="background:#c0392b"></i><code>أحمر</code></span><span><i style="background:#1e8e3e"></i><code>أخضر</code></span><span><i style="background:#2563eb"></i><code>أزرق</code></span><span><i style="background:#d4a106"></i><code>أصفر</code></span><span><i style="background:#e67e22"></i><code>برتقالي</code></span><span><i style="background:#7d3c98"></i><code>بنفسجي</code></span><span><i style="background:#d63384"></i><code>وردي</code></span><span><i style="background:#0e8f9a"></i><code>فيروزي</code></span><span><i style="background:#c9a04a"></i><code>ذهبي</code></span><span><i style="background:#6b7280"></i><code>رمادي</code></span><span><i style="background:#1b2b6b"></i><code>كحلي</code></span><span><i style="background:#8d5524"></i><code>بني</code></span><span><i style="background:#111827"></i><code>أسود</code></span><span><i style="background:#ffffff"></i><code>أبيض</code></span></div>

A background is a light tint of its colour over the page, so text on it stays readable in
both the light and the dark theme.

## A theme for the whole app — `سمة`

Write `سمة` once, at the top of the program:

```dad
سمة(اللون: "فيروزي"، الخط: "نسخ"، الزوايا: 14، العرض: 960)
```

| Option | What it changes | Values |
|---|---|---|
| `اللون` | The main colour: buttons, links, focus, the stroke under page titles. Text on buttons is made light or dark to stay readable | a colour name or `"#…"` |
| `الخط` | The typeface | `"كوفي"` (Noto Kufi Arabic, built in; the default), `"نسخ"` (a Naskh face on the device, such as Noto Naskh Arabic or Geeza Pro), `"النظام"` (the device's own) |
| `الزوايا` | How round corners are, in pixels | 0 to 40; 8 by default |
| `العرض` | The widest the page gets, in pixels | 320 to 2000; about 670 by default |
| `الحجم` | The size of text, in pixels | 12 to 28; 17 by default |
| `الوضع` | Light or dark | `"تلقائي"` (follow the device; the default), `"فاتح"`, `"داكن"` |

The theme applies to every page. If `سمة` is written twice, the last one counts.

## Your own stylesheet { #your-own-stylesheet }

Put a CSS file named like your program next to it: `app.css` for `app.dad`. `dad dev` and
`dad serve` load it after the built-in stylesheet, and `dad dev` reloads the page when you
save it. `dad build` copies it into the site as `style.css`.

Give elements its classes with `صنف`. Class names can be Arabic:

```dad
بطاقة(صنف: "عرض-خاص") {
    عنوان("خصم الأسبوع")
}
```

```css
.عرض-خاص {
  border: 2px dashed var(--lapis);
  transform: rotate(-1deg);
}
```

The built-in stylesheet's colours are CSS variables you can use: `--lapis` (the app's colour),
`--ink` (text), `--muted`, `--paper` (the page), `--surface`, `--line` and `--radius`. The
elements have classes too, such as `.dad-card` and `.dad-table`.

This is the stylesheet of `examples/09-library.dad`, which hides the stroke under its title:

```css
--8<-- "examples/09-library.css"
```

## Which one wins

An element's options apply to that element only, over the theme. Your stylesheet comes last,
so it can change anything, options included.
