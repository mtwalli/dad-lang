# 10. Going online

Your app runs on your computer. In this last lesson you organise it in several files, then
put it on the internet, in one of two ways.

## A program in several files

When a program grows, split it into files, each with its own subject. Each file gives the
others what they need, and imports what it needs itself with `استورد` («import»):

```dad title="حساب.dad"
// أدوات الحساب، يستعملها أي ملف
خاص ثابت الضريبة = 0.15

دالة مع_الضريبة(السعر: عشري): عشري = (السعر * (1 + الضريبة)).تقريب(2)

دالة المجموع(الأسعار: [عشري]): عشري = الأسعار.حول { مع_الضريبة(هو) }.مجموع().تقريب(2)
```

```dad title="الفاتورة.dad"
استورد مع_الضريبة، المجموع من "./حساب"

ثابت السلة = [10.0، 4.5، 20.0]
لكل س في السلة {
    اطبع(س، "←"، مع_الضريبة(س))
}
اطبع("المجموع:"، المجموع(السلة))
```

```sh
dad run الفاتورة.dad
```

``` { .text .dad-out }
10.0 ← 11.5
4.5 ← 5.18
20.0 ← 23.0
المجموع: 39.68
```

- `حساب.dad` («calculation») has the tax tools, and `الفاتورة.dad` («the invoice») uses them.
  `استورد مع_الضريبة، المجموع من "./حساب"` brings these two functions in from `حساب.dad`,
  next to it.
- Everything at the top of a file is given to the files that import it, except what starts
  with `خاص` («private»). No other file can import `الضريبة` (the tax rate), so the file is
  free to change it.
- You only run the **main** file; Ḍād finds the others by itself.

The page [Programs in several files](../modules.md) has the details.

## Way one: a static site

`dad build` turns your program into a ready-made site:

```sh
dad build مهامي.dad
```

(Or in Arabic: `ضاد ابن مهامي.dad`.) A folder `dist/مهامي` appears, with `index.html` and
what it needs. Open it in a browser directly, or upload it to any static site host, such as
GitHub Pages or Netlify, even a free one.

On this site the whole program runs in the visitor's browser, and **needs no server**. That
has a price: tables are saved in each visitor's own browser, so nobody sees anyone else's
tasks. That suits a personal tool, such as your own task list or the tip calculator.

## Way two: a server for everyone

When everyone should see the same data, as in a shop or a guest book, run the program as a
server:

```sh
dad serve مهامي.dad
```

(Or in Arabic: `ضاد انشر مهامي.dad`.) This is the server you have used with `dad dev`, set up
for going online: it doesn't reload, shows nobody the developer page, and accepts
connections from any device. Run it on a computer connected to the internet, or on a
platform such as Fly.io or Render: the [deployment guide](../deploy.md) shows how.

!!! warning "Before you go online"

    Ḍād has no sign-in or permissions yet: every visitor can call every server function. So
    don't put data online this way that not everyone may see or change.

| | `dad build` | `dad serve` |
|---|---|---|
| Needs | static file hosting | a computer that is always on |
| The data | in each visitor's browser | in one file on the server, seen by everyone |
| Suits | personal tools, games, calculators | shared apps |

## Where next?

You have finished the tutorial. Congratulations! You know how to write a Ḍād program, from
the first line to a site on the internet.

- Read the [examples](../examples.md) to see bigger programs.
- Go back to the [interface](../ui.md) and [server and data](../server.md) pages for the
  details.
- The [language specification](../SPEC.md) has everything, for those who want precision.
- Found a mistake, or have an idea? Write it on the
  [project's GitHub page](https://github.com/mtwalli/dad-lang/issues).

## One last exercise

Move the table and the server functions of the to-do app from lesson 9 into a file
`بيانات.dad` («data»), and leave only the component in `مهامي.dad`, importing what it needs.

??? note "Solution"

    ```dad title="بيانات.dad"
    جدول مهمة {
        العنوان: نص
        منجزة = خطأ
    }

    خادم دالة كل_المهام() = مهمة.الكل()
    خادم دالة أضف(العنوان: نص) = مهمة.أنشئ(العنوان: العنوان)
    خادم دالة بدّل(م: مهمة) = م.نسخة(منجزة: ليس م.منجزة).احفظ()
    ```

    ```dad title="مهامي.dad"
    استورد كل_المهام، أضف، بدّل من "./بيانات"

    مكون الرئيسية {
        حالة المهام = كل_المهام()
        حالة الجديدة = ""

        عمود {
            عنوان("مهامي")
            صف {
                حقل(الجديدة، تلميح: "مهمة جديدة…")
                زر("أضف") {
                    أضف(الجديدة)
                    الجديدة = ""
                    المهام = كل_المهام()
                }
            }
            لكل م في المهام {
                مربع_اختيار(م.منجزة، نص: م.العنوان) {
                    بدّل(م)
                    المهام = كل_المهام()
                }
            }
        }
    }
    ```

    Run it with `dad dev مهامي.dad`: Ḍād finds `بيانات.dad` by itself, and reloads when you
    save either file.
