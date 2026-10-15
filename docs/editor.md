# Editor

Ḍād files are plain UTF-8 text, so any editor can open them. For the best experience use
[Visual Studio Code](https://code.visualstudio.com/) with the Ḍād extension. It runs on
macOS, Windows and Linux.

## The VS Code extension

- **Highlighting** for keywords, strings with `{…}` interpolation, `«…»` strings, Arabic and
  Latin numbers, comments, and Ḍād code blocks in Markdown.
- **Errors as you type.** The same Arabic messages as `dad check`, with the exact word underlined
  and listed in the Problems panel.
- **Run** the current file with the ▶ button in the editor title. The output appears in the
  terminal panel.
- **Keyword help.** Hover a keyword to see what it means in English.
- **Completion and snippets** for keywords, built-ins, functions, types, enums, loops and
  matching. Snippets answer to Arabic and English prefixes, so `دالة` and `func` both work.

### Install it

The extension needs the `dad` program, version 0.2.0 or newer. [Install Ḍād](install.md) first.

1. Download [`dad-lang.vsix`](https://github.com/mtwalli/dad-lang/releases/latest/download/dad-lang.vsix)
   from the latest release.
2. In VS Code, open the Extensions view (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>X</kbd>, or
   <kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>X</kbd> on a Mac).
3. Open the **…** menu at the top of the view and choose **Install from VSIX…**, then pick the
   downloaded file.

Or from a terminal:

```sh
code --install-extension dad-lang.vsix
```

To update, install the newer `.vsix` the same way.

### Use it

Open any `.dad` file, or a folder that contains some.

| To | Do this |
|---|---|
| Run the file | Click ▶ in the editor title, or press <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> (<kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> on a Mac) |
| Run a file from the sidebar | Right-click it and choose **Run Ḍād File** |
| See all problems | Open the Problems panel (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>M</kbd>) |
| Insert a snippet | Type `دالة`, `لكل`, `طابق`, `func`, `for`, `match`… and press <kbd>Tab</kbd> |
| Check which `dad` is used | Run **Ḍād: Show Installed Version** from the Command Palette |

### Settings

| Setting | Default | What it does |
|---|---|---|
| `dad.path` | empty | Path to `dad`. Empty means find it on your `PATH` or in `~/.dad/bin`. |
| `dad.checkOnType` | `true` | Check for errors as you type. Off means on open and save only. |
| `dad.saveBeforeRun` | `true` | Save the file before running it. |

If VS Code says it can't find `dad`, run `which dad` (macOS and Linux) or `where dad`
(Windows) in a terminal and put the result in `dad.path`.

## Right-to-left text in code editors

No mainstream code editor has a true right-to-left mode. VS Code lays out each line with the
Unicode bidirectional algorithm: Arabic words and sentences read correctly, but every line
starts at the left edge. Ḍād is designed to read well this way:

- Brackets and comparison signs are mirrored characters, so `أ < ب` reads correctly from the
  right.
- Hidden direction characters, which could make code look different from what it does, are
  highlighted by the editor and rejected by Ḍād outside strings.
- Arabic punctuation (`،` `؛` `؟`) works everywhere its Latin form does.

## Other editors

Any editor works with the `dad` command in a terminal:

```sh
dad check file.dad    # errors and warnings
dad run file.dad      # run it
```

Editors that read TextMate grammars, such as the JetBrains IDEs through their TextMate Bundles
support, can use
[`dad.tmLanguage.json`](https://github.com/mtwalli/dad-lang/blob/main/editors/vscode/syntaxes/dad.tmLanguage.json)
for highlighting. Other tools can use `dad check --json <file>`, or `-` to read
the source from standard input, to get problems in a machine-readable form.
A language server for all editors is planned (see the [roadmap](ROADMAP.md)).
