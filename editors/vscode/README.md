# Ḍād (ضاد) for Visual Studio Code

Support for [Ḍād](https://mtwalli.github.io/dad-lang/), a programming language with Arabic keywords.

- **Highlighting** for keywords, strings with `{…}` interpolation, `«…»` strings, numbers, and Ḍād code blocks in Markdown.
- **Right to left**: `.dad` files open in a right-to-left editor, with lines starting at the right
  and line numbers on the right. Saving, undo, git and problems work as usual. **Open as text**
  switches to VS Code's text editor for completion, hovers and find.
- **Ḍād Mono**, a coding font made for Ḍād: Kufi Arabic that sits with Latin letters and
  digits. The extension uses it for `.dad` files and offers to install it the first time.
- **Errors as you type**, in Arabic, with the exact word underlined.
- **Run** the current file, with the output shown right to left in the **Ḍād** panel until the next run. Use the ▶ button in the editor title, or
  <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> (<kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> on a Mac).
- **Keyword help**: hover a keyword to see what it means in English.
- **Snippets** for functions, types, enums, loops and matching. They answer to Arabic and English
  prefixes, so `func` and `دالة` both work.

## Requirements

The extension uses the `dad` program to check and run files.
[Install Ḍād](https://mtwalli.github.io/dad-lang/install/) version 0.1.4 or newer. The extension
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
the Problems panel work as usual. Completion, keyword hovers, find and AI suggestions need
VS Code's text editor: use **Open as text** (title bar or the bar at the bottom) to switch, and
**Ḍād: Reopen Right to Left** to come back.

In the text editor, Arabic words read correctly but lines start at the left. Brackets and
comparison signs are mirrored characters, so `أ < ب` still reads right, and hidden direction
characters are highlighted.
