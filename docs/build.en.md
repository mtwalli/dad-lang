---
icon: lucide/package
---

# Building for the web

`dad build` compiles a Ḍād program to JavaScript. What you get depends on the program.

| The program has… | `dad build` makes | Run it with |
|---|---|---|
| an interface (`مكون`, `صفحة`) | a folder with `index.html`: a static site | open `index.html`, or put the folder on any static host |
| no interface | `app.js` | `node app.js` |

```sh
dad build todo.dad                 # → dist/todo/
dad build todo.dad --out site      # → site/
ضاد ابن todo.dad                   # the same, in Arabic
```

A program with mistakes is not built: `dad build` shows the same errors as `dad check`.

## A site with no server

The site is plain files, so it works from GitHub Pages, Netlify, any web server, or straight
from your disk:

```text
dist/todo/
├── index.html        the page
├── app.js            your program, compiled
├── dad-runtime.js    Ḍād's values, methods and elements
└── assets/           the page script, the stylesheet and the Kufi font
```

The program runs in the visitor's browser. Clicking and typing update the page at once,
without asking a server.

- **Tables** (`جدول`) are saved in the browser's storage (`localStorage`). Each visitor has
  their own records. They are kept after closing the page, but other visitors don't see
  them.
- **Pages** (`صفحة`) have addresses after `#`: the page `"/عن"` is `index.html#/عن`. That
  way any static host serves every page from the one `index.html`, and the site works from
  your disk too. Links, `انتقل` and the back button work as with `dad dev`.
- **Server functions** (`خادم دالة`) run in the page, like any function.
- **Routes** (`مسار`) need a server, so they are left out. `dad build` says so. Serve them
  with `dad dev` or `dad serve`.

To share records between visitors, run the program on a server with
[`dad serve`](deploy.md): everyone then sees the same tables, kept in SQLite.

## Programs without an interface

```sh
dad build hello.dad
node dist/hello/app.js
```

`app.js` needs `dad-runtime.js` next to it, and Node 18 or later. Tables are kept in memory
while the program runs.

## The same program, the same results

The interpreter (`dad run`, `dad dev`, the playground) and the compiled JavaScript follow
the same rules. Whole numbers stay whole (`7 % 3` is `1`, and `7 / 2` is `3.5`), decimals
print as `5.0`, and errors keep their Arabic messages and line numbers:

```text
خطأ أثناء التشغيل [السطر 3، العمود 21]: القيمة من النوع «قائمة» لا تحتوي «غير_موجود»
```

The tests run every example both ways and compare the output, the error messages, and the
pages after the same clicks and typing.

In the browser, `اطبع` writes to the developer console.
