# Ḍād (ضاد) for Visual Studio Code

Support for [Ḍād](https://mtwalli.github.io/dad-lang/), a programming language with Arabic keywords.

- **Highlighting** for keywords, strings with `{…}` interpolation, `«…»` strings, numbers, and Ḍād code blocks in Markdown.
- **Right to left**: `.dad` files open in a right-to-left editor, with lines starting at the right
  and line numbers on the right. Saving, undo, git and problems work as usual, and so do go to
  definition (<kbd>F12</kbd> or <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+click), references, rename and
  symbols, also from the right-click menu with **Format Document**. **Open as text** switches
  to VS Code's text editor for completion, hovers and find.
- **Ḍād Mono**, a coding font made for Ḍād: Kufi Arabic that sits with Latin letters and
  digits. The extension uses it for `.dad` files and offers to install it the first time.
- **Errors as you type**, in Arabic, with the exact word underlined.
- **Formatting** with `dad fmt` on save and with **Format Document**, and indentation that
  follows the code as you type in the right-to-left editor.
- **Run** the current file, with the output shown right to left in the **Ḍād** panel until the next run. Use the ▶ button in the editor title, or
  <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> (<kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> on a Mac).
- **Preview the interface** beside the code before running it (**Preview Interface** in the
  title bar, or <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+<kbd>K</kbd> <kbd>V</kbd>): it follows the code as you
  type, buttons and pages work, errors are listed with a link to the code, and **📱** shows it
  at phone width.
- **Keyword help**: hover a keyword to see what it means in English.
- **Go to definition, references, rename and the outline**, from Ḍād's language server, across
  the files of a program: imported names and the fields of imported types lead to their file,
  the file in `استورد … من "./ملف"` is a link, and **Go to Symbol in Workspace** finds every
  type and function. Completion after `.` lists the value's real fields and methods.
- **Snippets** for functions, types, enums, loops and matching. They answer to Arabic and English
  prefixes, so `func` and `دالة` both work.

## Requirements

The extension uses the `dad` program to check and run files.
[Install Ḍād](https://mtwalli.github.io/dad-lang/en/install/) version 0.1.4 or newer. The extension
finds `dad` on your `PATH` or in the default install folder. If yours is somewhere else, set
**Ḍād: Path** (`dad.path`) in the settings.

## Settings

| Setting | Default | What it does |
|---|---|---|
| `dad.path` | empty | Path to `dad`. Empty means find it automatically. |
| `dad.checkOnType` | `true` | Check for errors as you type. Off means on open and save only. |
| `dad.saveBeforeRun` | `true` | Save the file before running it. |

## Right-to-left text

VS Code's own text editor has no right-to-left mode, so the extension opens `.dad` files in
its own right-to-left editor. It edits VS Code's copy of the file, so saving, undo, git and
the Problems panel work as usual, and so does moving around the code: go to definition,
references, rename, symbols and **Go Back**, from the keyboard or the right-click menu.
Completion, keyword hovers, find and AI suggestions need
VS Code's text editor: use **Open as text** (title bar or the bar at the bottom) to switch, and
**Ḍād: Reopen Right to Left** to come back.

In the text editor, Arabic words read correctly but lines start at the left. Brackets and
comparison signs are mirrored characters, so `أ < ب` still reads right, and hidden direction
characters are highlighted.
