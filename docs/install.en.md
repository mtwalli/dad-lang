---
icon: lucide/download
---

# Install Ḍād

!!! note "New to programming?"
    The [tutorial](tutorial/index.md) starts from here and ends with a site online. It is
    also [in Arabic](https://mtwalli.github.io/dad-lang/tutorial/). You can also [run code in your browser](playground.md) without installing
    anything.

## Install

=== "Homebrew (macOS, Linux)"

    ```sh
    brew install mtwalli/dad/dad
    ```

    `brew upgrade dad` updates it later.

=== "winget (Windows)"

    ```powershell
    winget install mtwalli.dad
    ```

    Open a new terminal afterwards. `winget upgrade mtwalli.dad` updates it later.

=== "Debian, Ubuntu"

    Download `dad_<version>_amd64.deb` (or `_arm64.deb`) from the
    [releases page](https://github.com/mtwalli/dad-lang/releases/latest), then:

    ```sh
    sudo apt install ./dad_*_amd64.deb
    ```

=== "Fedora, RHEL"

    Download `dad-<version>-1.x86_64.rpm` (or `.aarch64.rpm`) from the
    [releases page](https://github.com/mtwalli/dad-lang/releases/latest), then:

    ```sh
    sudo dnf install ./dad-*.x86_64.rpm
    ```

=== "Install script (macOS, Linux)"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
    ```

    The script downloads the latest release for your system, verifies its checksum and
    puts `dad` in `~/.dad/bin`, together with the Arabic alias `ضاد`. If that folder isn't
    on your `PATH` yet, the script prints the one line to add.

=== "Install script (Windows)"

    In PowerShell:

    ```powershell
    irm https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.ps1 | iex
    ```

    The script installs `dad.exe` in `%LOCALAPPDATA%\dad\bin` and adds it to your `PATH`.
    Open a new terminal afterwards.

=== "With Go"

    If you have Go 1.26 or newer:

    ```sh
    go install github.com/mtwalli/dad-lang/cmd/dad@latest
    ```

=== "Download manually"

    Download the archive for your system from the
    [releases page](https://github.com/mtwalli/dad-lang/releases/latest), unpack it and
    put `dad` (or `dad.exe`) anywhere on your `PATH`. `checksums.txt` lists the SHA-256 of
    every archive.

To install a specific version, set `DAD_VERSION`, for example
`DAD_VERSION=v0.1.0`, before running the script.

## Check the installation

```sh
dad version
```

You should see the version number, for example `ضاد 0.2.0`. On macOS and Linux, Homebrew,
the packages and the install script also give you the command's Arabic name, `ضاد`.

## Your first file

Save this as `hello.dad`:

```dad
// أول برنامج بلغة ضاد
ثابت اسم = "العالم"
اطبع("مرحبا يا {اسم}!")
```

Run it from the folder where you saved it:

```sh
dad run hello.dad        # or: ضاد شغل hello.dad
```

``` { .text .dad-out }
مرحبا يا العالم!
```

`dad hello.dad` works too. Any `.dad` file runs the same way, for example the
[examples](examples.md) in the repository:

```sh
git clone https://github.com/mtwalli/dad-lang
cd dad-lang
dad run examples/03-shapes.dad
```

### When something goes wrong

Errors found while running stop the program and point to the line, in Arabic:

```dad
ثابت أ = 10
اطبع(أ / 0)
```

``` { .text .dad-out }
hello.dad: خطأ أثناء التشغيل [السطر 2، العمود 8]: مشكلة لم تُعالَج: قسمة على صفر
```

To find mistakes without running anything, use `check`:

```sh
dad check hello.dad      # or: ضاد فحص hello.dad
```

``` { .text .dad-out }
✓ hello.dad: لا أخطاء (11 رمزاً)
```

Now add a line with a mistake. Ḍād writes numbers with the digits 0-9 only, so a number
typed with Arabic-Indic digits is an error:

```dad
ثابت عمر = ١٢
```

Check again. Ḍād reports the problem in Arabic, with its line and column:

``` { .text .dad-out }
hello.dad: خطأ [السطر 4، العمود 12]: اكتب الأعداد بالأرقام 0-9: «١٢» تُكتب «12»
```

To see how Ḍād reads your code, print its tokens:

```sh
dad tokens hello.dad     # or: ضاد رموز hello.dad
```

For highlighting, errors as you type and a Run button, install the [Ḍād extension for VS Code](https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang) (or search for **Ḍād** in the Extensions view). The [editor guide](editor.md) has the details.

## Commands

| Command | Arabic | What it does |
|---|---|---|
| `dad run <file>` | `ضاد شغل <ملف>` | Run a program (`dad <file>` also works) |
| `dad fmt <file or folder>` | `ضاد نسق <ملف>` | Format files in place (`--check` lists the ones that need it). See [Formatting](editor.md#formatting) |
| `dad lsp` | `ضاد خادم_اللغة` | The language server, for editors: definitions, references, rename, completion. See [the editor guide](editor.md#the-language-server) |
| `dad dev <file>` | `ضاد تطوير <ملف>` | Serve routes and server functions on `localhost:8080`, reloading on save (`--port` to change it). See [Server and data](server.md) |
| `dad serve <file>` | `ضاد انشر <ملف>` | Run the program as a production server: no developer page, no reloading, every network interface, `$PORT` (`--port`, `--host`, `--data`). See [Putting a program online](deploy.md) |
| `dad build <file>` | `ضاد ابن <ملف>` | Compile to JavaScript: a static site for a program with an interface, otherwise `app.js` for Node (`--out` to choose the folder). See [Building for the web](build.md) |
| `dad check <file>` | `ضاد فحص <ملف>` | Check a file without running it: syntax, names, arguments, types and `طابق` cases (`--json` for tools) |
| `dad preview <file>` | `ضاد معاينة <ملف>` | The page of the program's interface, as JSON, for an editor's preview. See [Preview the interface](editor.md#preview-the-interface) |
| `dad tokens <file>` | `ضاد رموز <ملف>` | Print every token with its line and column |
| `dad version` | `ضاد إصدار` | Print the installed version |
| `dad help` | `ضاد مساعدة` | List the commands |

## Uninstall

=== "Homebrew"

    ```sh
    brew uninstall dad
    ```

=== "winget"

    ```powershell
    winget uninstall mtwalli.dad
    ```

=== "Debian, Ubuntu, Fedora"

    ```sh
    sudo apt remove dad      # or: sudo dnf remove dad
    ```

=== "Install script (macOS, Linux)"

    ```sh
    rm -rf ~/.dad
    ```

    Then remove the `PATH` line from your shell profile if you added one.

=== "Install script (Windows)"

    Delete `%LOCALAPPDATA%\dad`, then remove that folder from your user `PATH` in
    *Settings → System → About → Advanced system settings → Environment Variables*.

=== "With Go"

    ```sh
    rm "$(go env GOPATH)/bin/dad"
    ```
