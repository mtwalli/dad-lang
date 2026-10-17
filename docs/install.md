# Install Ḍād

!!! note "What works today"
    `dad run` runs Ḍād programs: values, functions, lambdas, types, enums, matching,
    optionals and errors. Tables, components and routes (`جدول`, `مكون`, `مسار`) come in
    Phases 2 and 3 (see the [roadmap](ROADMAP.md)). You can also
    [run code in your browser](playground.md) without installing anything.

## Install

=== "macOS and Linux"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
    ```

    The script downloads the latest release for your system, verifies its checksum and
    puts `dad` in `~/.dad/bin`, together with the Arabic alias `ضاد`. If that folder isn't
    on your `PATH` yet, the script prints the one line to add.

=== "Windows"

    In PowerShell:

    ```powershell
    irm https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.ps1 | iex
    ```

    The script installs `dad.exe` in `%LOCALAPPDATA%\dad\bin` and adds it to your `PATH`.
    Open a new terminal afterwards.

=== "With Go"

    If you have Go 1.24 or newer:

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

You should see the version number, for example `ضاد 0.1.6`.

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
| `dad dev <file>` | `ضاد تطوير <ملف>` | Serve routes and server functions on `localhost:8080`, reloading on save (`--port` to change it). See [Server and data](server.md) |
| `dad check <file>` | `ضاد فحص <ملف>` | Report errors and warnings in a file (`--json` for tools) |
| `dad tokens <file>` | `ضاد رموز <ملف>` | Print every token with its line and column |
| `dad version` | `ضاد إصدار` | Print the installed version |
| `dad help` | `ضاد مساعدة` | List the commands |

## Uninstall

=== "macOS and Linux"

    ```sh
    rm -rf ~/.dad
    ```

    Then remove the `PATH` line from your shell profile if you added one.

=== "Windows"

    Delete `%LOCALAPPDATA%\dad`, then remove that folder from your user `PATH` in
    *Settings → System → About → Advanced system settings → Environment Variables*.

=== "With Go"

    ```sh
    rm "$(go env GOPATH)/bin/dad"
    ```
