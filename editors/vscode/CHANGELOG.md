# Changelog

## Next release

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
