# Changelog

## Next release

- Pages: `صفحة` is highlighted before an address, `انتقل` like the other built-ins, and
  hovers explain both. A `صفحة` snippet, and pages appear in the outline by their address.
- The new elements (`قائمة_منسدلة`، `اختيار`، `منطقة_نص`، `جدول_عرض`، `قائمة_عناصر`، `ألسنة`،
  `لسان`، `نافذة`، `تنبيه`، `أيقونة`) and `سمة` are highlighted and explained on hover.
- Fixed: a check that took too long (the first run of a new `dad` on macOS, say) made every
  later check ignore imports, so each `استورد` showed «الاستيراد يحتاج ملفاً محفوظاً» until
  VS Code restarted.

- Programs in several files: a saved file is checked together with the files it imports
  (`استورد`), and saving a file checks the files that import it again. Go to definition on
  an imported name opens the file that declares it. With an older `dad`, files are checked
  alone, as before.

- Go to definition, find all references, rename, the outline, and hovers that show how a
  name is declared, from Ḍād's language server (`dad lsp`). After a `.`, completion lists the
  real fields and methods of the value. Needs a `dad` with `dad lsp`; with an older one these
  features stay off.

- Errors as you type now include the checker's: names that aren't defined (with a
  suggestion), wrong arguments and types, fields that don't exist and `طابق` that misses
  cases. They come from `dad check`, so update `dad` to get them.

- Hovers, completion and highlighting for the interface: `مكون`, `حالة` and the elements
  `عمود`، `صف`، `بطاقة`، `عنوان`، `فقرة`، `زر`، `حقل`، `مربع_اختيار`، `صورة`، `رابط`، `فاصل`.

- Formatting: **Format Document** and formatting on save (`dad.formatOnSave`, on by default)
  lay out the file with `dad fmt`, in the text editor and the right-to-left editor. The
  style follows the Kotlin conventions: four spaces, `{` at the end of the line, `}` on its
  own line, one statement per line. Update
  `dad` to the release that adds `dad fmt`; an older one shows a note instead.

- `.dad` files use four spaces per level.

- The right-to-left editor indents as you type: Enter adds a level after `{`, `(` or `[` and
  puts `}` on its own line, a closing bracket lines up with its opening line, Tab and
  Shift+Tab indent or outdent the selected lines, Backspace removes a level, and
  Shift+Alt+F formats the file.

- Run works when `dad.path` is a relative path such as `./bin/dad`.

- Hovers for `جدول`, `خادم` and `مسار` describe what they do now that they work with `dad dev`.

- Run shows the output in a new **Ḍād** panel next to the terminal: right to left, in Ḍād Mono,
  with errors in red, and it stays until the next run. **Run again** and **Stop** are in the
  panel. Before, the output disappeared when the program finished.

- `.dad` files open in a right-to-left editor: lines start at the right, line numbers are on
  the right, with highlighting, problems and Ḍād Mono. **Ḍād: Reopen as Text** and
  **Ḍād: Reopen Right to Left** switch between it and VS Code's text editor.

- Ḍād Mono, the coding font for Ḍād, is bundled and set for `.dad` files. The extension offers
  to install it once, and **Ḍād: Install the Ḍād Mono Font** installs it any time.

- Numbers are written with the digits 0-9 only. Arabic-Indic and Persian digits outside text
  are highlighted as mistakes, and `dad check` shows the number written correctly.

## 0.2.0

First release.

- Highlighting for keywords, strings with `{…}` interpolation, `«…»` strings, Arabic and Latin
  numbers, comments, declarations, and Ḍād code blocks in Markdown.
- Errors and warnings as you type, from `dad check`, in Arabic.
- Run the current file with the ▶ button, <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd>
  (<kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> on a Mac), or the file's right-click menu.
- Hover a keyword to see its English meaning. Keyword and built-in completion, and snippets
  that also answer to English prefixes (`func`, `for`, `match`…).
