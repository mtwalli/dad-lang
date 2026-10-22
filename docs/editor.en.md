---
icon: lucide/code-xml
---

# Editor

Ḍād files are plain UTF-8 text, so any editor can open them. For the best experience use
[Visual Studio Code](https://code.visualstudio.com/) with the Ḍād extension. It runs on
macOS, Windows and Linux.

## The VS Code extension

- **Right to left.** `.dad` files open in an editor whose lines start at the right ([details](#right-to-left)).
- **Highlighting** for keywords, strings with `{…}` interpolation, `«…»` strings, numbers, comments, and Ḍād code blocks in Markdown.
- **Errors as you type.** The same Arabic messages as `dad check`, with the exact word underlined: mistyped names (with «هل تقصد …؟»), wrong arguments and types, missing fields and `طابق` cases
  and listed in the Problems panel.
- **Run** the current file with the ▶ button in the editor title. The output appears right to
  left in the **Ḍād** panel, next to the terminal, and stays there until the next run. The
  panel has **Run again** and **Stop** buttons.
- **Preview the interface** beside the code, live, before running it ([details](#preview-the-interface)).
- **Keyword help.** Hover a keyword to see what it means in English.
- **Go to definition** (<kbd>F12</kbd>, or <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+click), **find all
  references** (<kbd>Shift</kbd>+<kbd>F12</kbd>) and **rename** (<kbd>F2</kbd>) for variables,
  functions, types, fields, parameters and cases. Renaming a field also changes `نقطة(س: …)`
  and `نسخة(س: …)` wherever the field is named.
- **Across files.** In a program of several files, go to definition leads to the file that
  declares a name, including the fields and methods of an imported type; the file name in
  `استورد … من "./نماذج"` is a link to that file; and references and rename cover every file
  that imports the name. **Go to Symbol in Workspace** (<kbd>⌘</kbd>/<kbd>Ctrl</kbd>+<kbd>T</kbd>)
  finds the types, functions, components and fields of all the workspace's `.dad` files.
- **Hover a name** to see how it is declared, such as `دالة مربع(عدد: صحيح): صحيح`.
- **The outline** (<kbd>⌘</kbd>/<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>O</kbd>, and the Outline
  view) lists the file's types, functions, tables and components with their fields and methods.
- **Completion and snippets** for keywords, built-ins and the names in scope. After a `.`,
  completion lists what the value really has: the fields and methods of `نقطة`, the methods of
  text or lists, the queries of a table. Snippets cover functions, types, enums, loops and
  matching. Snippets answer to Arabic and English prefixes, so `دالة` and `func` both work.

### Install it

The extension needs the `dad` program, version 0.1.4 or newer. [Install Ḍād](install.md) first.

1. In VS Code, open the Extensions view (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>X</kbd>, or
   <kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>X</kbd> on a Mac).
2. Search for **Ḍād** (or `mtwalli.dad-lang`) and click **Install**.

[Open it on the Marketplace](https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang){ .md-button }

Or from a terminal:

```sh
code --install-extension mtwalli.dad-lang
```

VS Code updates the extension automatically.

??? note "Installing without the Marketplace"
    Each [release](https://github.com/mtwalli/dad-lang/releases/latest) also has the extension
    as [`dad-lang.vsix`](https://github.com/mtwalli/dad-lang/releases/latest/download/dad-lang.vsix).
    Download it, then in the Extensions view open the **…** menu, choose
    **Install from VSIX…** and pick the file.

### Use it

Open any `.dad` file, or a folder that contains some.

| To | Do this |
|---|---|
| Run the file | Click ▶ in the editor title, or press <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> (<kbd>⌘</kbd>+<kbd>Shift</kbd>+<kbd>Enter</kbd> on a Mac) |
| Run a file from the sidebar | Right-click it and choose **Run Ḍād File** |
| See all problems | Open the Problems panel (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>M</kbd>) |
| Format the file | Save it (on by default), or press <kbd>Shift</kbd>+<kbd>Alt</kbd>+<kbd>F</kbd> (<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>I</kbd> on Linux). See [Formatting](#formatting) |
| Insert a snippet | Type `دالة`, `لكل`, `طابق`, `func`, `for`, `match`… and press <kbd>Tab</kbd> |
| Check which `dad` is used | Run **Ḍād: Show Installed Version** from the Command Palette |

### Settings

| Setting | Default | What it does |
|---|---|---|
| `dad.path` | empty | Path to `dad`. Empty means find it on your `PATH` or in `~/.dad/bin`. |
| `dad.checkOnType` | `true` | Check for errors as you type. Off means on open and save only. |
| `dad.saveBeforeRun` | `true` | Save the file before running it. |
| `dad.formatOnSave` | `true` | Format the file with `dad fmt` when you save it, in both editors. |

If VS Code says it can't find `dad`, run `which dad` (macOS and Linux) or `where dad`
(Windows) in a terminal and put the result in `dad.path`.

## Preview the interface

To see a program's interface while you write it, without running it, click
**Preview Interface** in the editor's title bar (or **معاينة** at the bottom of the
right-to-left editor, or press <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+<kbd>K</kbd> <kbd>V</kbd>). The
interface opens beside the code:

- **It works.** Buttons, fields, tabs, dialogs and pages respond, server functions run in the
  preview, and tables keep their rows there. The program is compiled as by
  [`dad build`](build.md), so routes (`مسار`) are left out.
- **It follows the code** as you type, and when you save a file it imports or its stylesheet
  (`app.css`). It stays on the page you were looking at.
- **With errors**, it keeps showing the last version that worked, with the errors listed at
  the bottom. Click one to go to it in the code.
- **📱** shows the interface as wide as a phone, so the layout for phones applies; **↺**
  clears the tables and starts again.

A program without `مكون الرئيسية()` or `صفحة` has no interface to show: run it to see what it
prints. The preview needs a `dad` newer than 0.2.2.

## Ḍād Mono, the coding font

Ḍād has its own coding font, **Ḍād Mono** (installed as "Dad Mono"). Its Arabic is
geometric Kufi, sized and weighted to sit on the same line as the Latin letters, digits
and symbols. Every code block on this site uses it.

<p class="dad-font-sample" dir="rtl" lang="ar">إذا عدد > 10 { اطبع("كبير") }</p>

**In VS Code** the extension sets Ḍād Mono for `.dad` files. The first time you open one, it
offers to install the font for you: choose **Install font**, then reload VS Code. You can
also run **Ḍād: Install the Ḍād Mono Font** from the Command Palette at any time.
Other files keep your usual font.

**Elsewhere** (terminal, other editors), download and install it yourself:

[Ḍād Mono Regular](assets/fonts/DadMono-Regular.ttf){ .md-button }
[Ḍād Mono Bold](assets/fonts/DadMono-Bold.ttf){ .md-button }

=== "macOS"

    Double-click each file and choose **Install Font**.

=== "Windows"

    Right-click each file and choose **Install** (or **Install for all users**).

=== "Linux"

    ```sh
    mkdir -p ~/.local/share/fonts
    cp DadMono-*.ttf ~/.local/share/fonts/
    fc-cache -f
    ```

Then pick **Dad Mono** as the font in the app. To use it for all files in VS Code, set
`editor.fontFamily` to `'Dad Mono', monospace`.

The Arabic comes from Noto Kufi Arabic and the Latin from DejaVu Sans Mono. Ḍād Mono is free
under the SIL Open Font License ([license](assets/fonts/LICENSE.txt)). To rebuild it, run
`python3 tools/font/build.py`.

## Formatting

`dad fmt` lays out Ḍād code one standard way, following the
[Kotlin coding conventions](https://kotlinlang.org/docs/coding-conventions.html), so every
file reads the same and nobody argues about spaces:

- Four spaces of indentation for each level, worked out from the code.
- A block that spans lines opens with `{` at the end of its first line and closes with `}`
  on a line of its own. `} وإلا {` and `} التقط خ {` stay on one line. A short block can stay
  on one line: `إذا ب == 0 { ارفع مشكلة("…") }`.
- One statement per line: `؛` between statements becomes a new line.
- One space around operators and after commas, none inside `( )` and `[ ]`, one inside
  `{ }` on a single line. Lambda parameters stay on the first line: `{ أ، ب =>`.
- Lines longer than 100 columns are split at their outermost `(` or `[`, one argument per line.
- A blank line before and after a declaration that spans lines (a function, type, table…),
  at most one blank line in a row, no spaces at line ends, one newline at the end of the file.
- Comments stay with their code. Multi-line strings are not touched.

```dad
// Before
دالة مربع(س:صحيح){
أرجع س*س
      }
اطبع( [1،2].حول {هو*2} )؛ اطبع(مربع(3))

// After
دالة مربع(س: صحيح) {
    أرجع س * س
}

اطبع([1، 2].حول { هو * 2 })
اطبع(مربع(3))
```

Only the layout changes, never the code: `dad fmt` checks that the result has exactly the
same words and symbols (a `؛` and a new line count as the same) and that it still reads
without errors. A file with errors is left as it is until you fix them.

In VS Code, files are formatted when you save them (turn this off with `dad.formatOnSave`) and
with **Format Document**. From a terminal:

```sh
dad fmt app.dad          # or: ضاد نسق app.dad
dad fmt .                # every .dad file in this folder and below
dad fmt --check .        # list files that aren't formatted, without changing them (for CI)
```

## Right to left

VS Code's text editor always starts lines at the left, and no setting changes that. So the
extension opens `.dad` files in its own **right-to-left editor**: lines start at the right,
line numbers sit on the right, and the cursor, selection and Arabic keyboard input work the
way they do in any Arabic text. It shows the same colors, problems (with wavy underlines and
the message for the current line at the bottom) and Ḍād Mono.

It edits VS Code's own copy of the file, so saving, undo, the unsaved-changes dot, Git and the
Problems panel all work as usual. <kbd>Ctrl</kbd>+<kbd>Enter</kbd>
(<kbd>⌘</kbd>+<kbd>Enter</kbd>) or **▶ شغّل** runs the file.

Indentation follows the code as you type:

| Key | What it does |
|---|---|
| <kbd>Enter</kbd> | Keeps the line's indentation, one level (four spaces) more after `{`, `(` or `[`. Between `{` and `}` it puts the `}` on its own line |
| `}` `)` `]` | At the start of a line, lines up with the line that opened the bracket |
| <kbd>Tab</kbd> / <kbd>Shift</kbd>+<kbd>Tab</kbd> | Indent or outdent the line, or every selected line |
| <kbd>Ctrl</kbd>+<kbd>]</kbd> / <kbd>Ctrl</kbd>+<kbd>[</kbd> | The same, with <kbd>⌘</kbd> on a Mac |
| <kbd>Backspace</kbd> | In the indentation, goes back one level |
| <kbd>Shift</kbd>+<kbd>Alt</kbd>+<kbd>F</kbd> | Format the file |

Moving around the code works there too, across the files of a program:

| Key | What it does |
|---|---|
| <kbd>F12</kbd>, or <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+click | Go to the definition. Holding <kbd>⌘</kbd>/<kbd>Ctrl</kbd> underlines the name under the mouse. On the file name of an `استورد`, opens that file |
| <kbd>Shift</kbd>+<kbd>F12</kbd> | Find all references: choose one from the list |
| <kbd>F2</kbd> | Rename the name, in every file that names it |
| <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>O</kbd> | Go to a type, function or field of the file |
| <kbd>⌘</kbd>/<kbd>Ctrl</kbd>+<kbd>T</kbd> | Go to a type, function or field of any file in the workspace |
| <kbd>Ctrl</kbd>+<kbd>-</kbd> on a Mac, <kbd>Alt</kbd>+<kbd>←</kbd> on Windows, <kbd>Ctrl</kbd>+<kbd>Alt</kbd>+<kbd>-</kbd> on Linux | Go back to where you were before |

**Right-click** in the editor for the same commands, with **Format Document** and **Run**.

The rest of VS Code's own features need its text editor, and are only in the text view:
completion, hovers, find and replace, multiple cursors and AI suggestions. To switch, use the
button in the editor's title bar or at the bottom of the editor:

| To | Do this |
|---|---|
| Open the file in VS Code's text editor | **Open as text**, or **Ḍād: Reopen as Text (Left to Right)** |
| Go back to right to left | **Ḍād: Reopen Right to Left** |
| Always open `.dad` files as text | Right-click a `.dad` file, **Open With…**, choose **Text Editor**, then **Configure default editor for '*.dad'…** |

In the text editor each line is laid out with the Unicode bidirectional algorithm: Arabic
words read correctly, but lines start at the left. Ḍād is designed to read well there too:

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
dad dev file.dad      # serve its routes and server functions
```

### The language server

`dad lsp` is a language server: any editor that speaks the
[Language Server Protocol](https://microsoft.github.io/language-server-protocol/) gets
errors as you type, go to definition, find references, rename (across the files of a
program), hover, completion (with members after `.`), the outline, links to imported files,
the workspace's symbols and formatting. Point the editor at the command `dad lsp`
for `.dad` files:

=== "Neovim"

    ```lua
    vim.filetype.add({ extension = { dad = "dad" } })
    vim.api.nvim_create_autocmd("FileType", {
      pattern = "dad",
      callback = function()
        vim.lsp.start({ name = "dad", cmd = { "dad", "lsp" }, root_dir = vim.fs.root(0, { ".git" }) })
      end,
    })
    ```

=== "Helix"

    ```toml
    # ~/.config/helix/languages.toml
    [language-server.dad]
    command = "dad"
    args = ["lsp"]

    [[language]]
    name = "dad"
    scope = "source.dad"
    file-types = ["dad"]
    language-servers = ["dad"]
    indent = { tab-width = 4, unit = "    " }
    ```

=== "Zed and others"

    Add a language server whose command is `dad` with the argument `lsp`, for files ending
    in `.dad`.

Editors that read TextMate grammars, such as the JetBrains IDEs through their TextMate Bundles
support, can use
[`dad.tmLanguage.json`](https://github.com/mtwalli/dad-lang/blob/main/editors/vscode/syntaxes/dad.tmLanguage.json)
for highlighting. Tools can also run `dad check --json <file>` (or `-` to read standard
input) to get problems in a machine-readable form.
