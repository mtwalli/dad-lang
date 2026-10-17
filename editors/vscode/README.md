# Ḍād (ضاد) for Visual Studio Code

Support for [Ḍād](https://mtwalli.github.io/dad-lang/), a programming language with Arabic keywords.

- **Highlighting** for keywords, strings with `{…}` interpolation, `«…»` strings, numbers, and Ḍād code blocks in Markdown.
- **Ḍād Mono**, a coding font made for Ḍād: Kufi Arabic that sits with Latin letters and
  digits. The extension uses it for `.dad` files and offers to install it the first time.
- **Errors as you type**, in Arabic, with the exact word underlined.
- **Run** the current file with the ▶ button in the editor title, or
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

VS Code has no right-to-left editor mode. Each line is laid out with the Unicode
bidirectional algorithm, so Arabic words read correctly, but lines start at the left edge.
Ḍād is designed for this: brackets and comparison signs are mirrored characters, so `أ < ب`
still reads right. Hidden direction characters, which could make code look different from
what it does, are highlighted, and Ḍād rejects them outside strings.
