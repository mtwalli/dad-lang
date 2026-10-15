# Install Ḍād

!!! note "What works today"
    Ḍād is in its first phase. The `dad` tool can **check** a Ḍād file and show its
    **tokens**, with errors and warnings in Arabic. Running programs comes in Phase 1
    (see the [roadmap](ROADMAP.md)). You can also [try it in your browser](playground.md)
    without installing anything.

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

You should see the version number in Arabic and Latin digits, for example `ضاد ٠.١.٠ (0.1.0)`.

## Your first file

Save this as `hello.dad`:

```dad
// أول برنامج بلغة ضاد
ثابت اسم = "العالم"
اطبع("مرحبا يا {اسم}!")
```

Check it:

```sh
dad check hello.dad      # or: ضاد فحص hello.dad
```

``` { .text .dad-out }
✓ hello.dad: لا أخطاء (١١ رمزاً)
```

Now add a line with a mistake, a number that mixes Arabic and Latin digits:

```dad
ثابت عمر = ١2
```

Check again. Ḍād reports the problem in Arabic, with its line and column:

``` { .text .dad-out }
hello.dad: خطأ [السطر ٤، العمود ١٢]: لا يجوز خلط الأرقام العربية (٠-٩) واللاتينية (0-9) في عدد واحد
```

To see how Ḍād reads your code, print its tokens:

```sh
dad tokens hello.dad     # or: ضاد رموز hello.dad
```

## Commands

| Command | Arabic | What it does |
|---|---|---|
| `dad check <file>` | `ضاد فحص <ملف>` | Report errors and warnings in a file |
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
