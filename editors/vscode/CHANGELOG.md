# Changelog

## Next release

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
