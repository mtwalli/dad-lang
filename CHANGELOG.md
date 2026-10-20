# Changelog

Each release on GitHub uses its section from this file as its notes.

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
