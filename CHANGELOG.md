# Changelog

Each release on GitHub uses its section from this file as its notes.

## Unreleased

### New

- **Several pages.** `صفحة "/مهمة/{رقم}" = تفاصيل(رقم)` gives part of the interface an address
  of its own; `{رقم}` in the address becomes a value. Links to pages open them in place, the
  link to the page on screen is marked for navigation bars, and `انتقل("/…")` moves to a page
  after a click. The back button and bookmarks work, the browser tab shows the page's first
  heading, and an address with no page shows a page that says so (404 from `dad serve`).
  It works the same in `dad dev`, `dad serve`, the playground and `dad build`, where pages
  have addresses such as `index.html#/عن`. `صفحة` is a keyword only before an address, so
  existing programs keep working. See `examples/08-pages.dad`.
- **Ten more elements:** `قائمة_منسدلة` (drop-down list) and `اختيار` (radio buttons), which
  choose a state's value from any list of values; `منطقة_نص` (text area); `جدول_عرض` (records
  as a table, with a block for each row); `قائمة_عناصر` (bulleted or numbered list);
  `ألسنة` and `لسان` (tabs); `نافذة` (a dialog, closed with ×, Esc or a click outside);
  `تنبيه` (info, success, warning or error); and `أيقونة`, 50 icons with Arabic names from
  Lucide, which buttons, links and tabs take too. `حقل` gains dates and times, and `أدنى` and
  `أقصى`.
- **Customizing the look,** in three layers: options on elements (`نمط`، `حجم`، `ملء`، `لون`،
  `خلفية`، `حشو`، `عرض`، `محاذاة`), a theme for the whole app with `سمة(اللون: …، الخط: …،
  الزوايا: …، الوضع: "داكن")`, and the program's own stylesheet, `app.css` next to `app.dad`,
  whose classes elements take with `صنف`. See `examples/09-library.dad`.
- The checker finds two pages, or a page and a `مسار جلب` route, at the same address, and
  mistakes in an address's `{names}`. It also knows the words options take, the colour names
  and the icon names, and suggests the closest one.
- `dad dev`'s developer page lists the pages; editors show them in the outline.
- **Moving around a program of several files** in editors: go to definition reaches the
  fields and methods of imported types, references and rename cover every file that names
  the name, the file in `استورد … من "./ملف"` is a link, and `dad lsp` answers
  `workspace/symbol`. In VS Code, the right-to-left editor gets go to definition
  (F12, Cmd/Ctrl+click), references, rename, symbols and Go Back, and a right-click menu
  with those and **Format Document**.

### Fixed

- VS Code: a slow check no longer turns off checking imports for the rest of the session,
  which showed an import error on every `استورد` in saved files. The message for code that
  is not in a saved file now says to save it.

## 0.2.0

The first public release of Ḍād (ضاد): a programming language with Arabic keywords for
whole web apps, with the interface, server and data in one language.

### Install

- **macOS and Linux:** `brew install mtwalli/dad/dad`, or
  `curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh`
- **Windows:** `winget install mtwalli.dad`, or
  `irm https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.ps1 | iex`
- **Debian and Ubuntu:** `sudo apt install ./dad_0.2.0_amd64.deb`
- **Fedora and RHEL:** `sudo dnf install ./dad-0.2.0-1.x86_64.rpm`
- **VS Code:** the «Ḍād» extension, or `dad-lang.vsix` below

### New since 0.1.15

- **Interfaces.** Components (`مكون`) with state (`حالة`), and the elements `عمود`،
  `صف`، `بطاقة`، `عنوان`، `فقرة`، `زر`، `حقل`، `مربع_اختيار`، `صورة`، `رابط`، `فاصل`.
  Pages are right to left, set in a Kufi typeface, with light and dark themes. They are live
  in `dad dev` and in the playground's Interface tab.
- **`dad build`** compiles a program to JavaScript. A program with an interface becomes a
  static site that needs no server; any other program becomes `app.js` for Node. The tests
  check that compiled programs print exactly what the interpreter prints.
- **`dad serve`** puts a program online for everyone, with shared SQLite data. It has no
  developer page and no reloading, takes `$PORT`, logs requests, and stops cleanly on
  `SIGTERM`. See *Putting a program online* in the docs.
- **Programs in several files:** `استورد … من "./ملف"` and `خاص`. Programs are checked across
  files, and go to definition opens the file that declares a name.
- **Packages:** Homebrew, winget, `.deb` and `.rpm`.
- **A tutorial in Arabic** on the docs site, from the first line to a site online.
- `أي` and `قيمة` are accepted as types that take any value.

### Already in 0.1.x

- `dad run`, with Arabic error messages that give the line and column.
- Values, functions, lambdas, types (`نوع`), enums (`تعداد`) with exhaustive `طابق`,
  optional values, and errors (`حاول`/`التقط`/`ارفع`).
- Tables (`جدول`) in SQLite, server functions (`خادم`), routes (`مسار`), and `dad dev` with
  live reload and a developer page.
- `dad check`: undefined names with suggestions, wrong arguments and types, missing fields
  and `طابق` cases.
- `dad fmt` in the Kotlin style.
- `dad lsp`, for definitions, references, rename, hover, completion and the outline in any
  editor.
- The VS Code extension, with a right-to-left editor for `.dad` files.
- The web playground.
