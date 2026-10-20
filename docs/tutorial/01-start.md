# 1. Getting started

## Install Ḍād

Ḍād is a single program called `dad`, which you can also type in Arabic as `ضاد`. Install
it the way that suits your computer:

=== "macOS"

    Open Terminal and type:

    ```sh
    brew install mtwalli/dad/dad
    ```

    No Homebrew? Use the install script instead:

    ```sh
    curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
    ```

=== "Windows"

    Open PowerShell and type:

    ```powershell
    winget install mtwalli.dad
    ```

    Then open a new window.

=== "Linux"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
    ```

    Or download a `.deb` or `.rpm` package from the
    [releases page](https://github.com/mtwalli/dad-lang/releases/latest).

Then check that it works:

```sh
dad version
```

You'll see something like `ضاد 0.2.0`. The [install page](../install.md) has other ways to
install.

### An editor to write in

You can write Ḍād in any text editor, but **VS Code** with the «Ḍād» extension makes it much
easier: the extension shows code right to left, colours the words, and tells you about
mistakes as you type. Search for «Ḍād» in the Extensions view and install it. The
[editor guide](../editor.md) has the details.

## Your first program

Make a folder for your lessons, and in it a file called `مرحبا.dad`. Write this in it:

```dad
// أول برنامج لي بلغة ضاد
اطبع("مرحبا بالعالم!")
```

Save the file and run it from the same folder:

```sh
dad run مرحبا.dad
```

``` { .text .dad-out }
مرحبا بالعالم!
```

(Or in Arabic: `ضاد شغل مرحبا.dad`.)

Congratulations, that is your first program! Let's read it line by line:

- The line that starts with `//` is a **comment** («my first program in Ḍād»). Programmers
  write comments for themselves and others, and Ḍād ignores them.
- `اطبع` («print») is a **function**: a piece of work with a name. It writes what you give
  it on the screen.
- What goes between the brackets is what you give the function. Here it is **text**
  («hello, world!»), written between quotation marks `"…"`.

`اطبع` takes more than one thing. Separate them with commas, and it prints them with a space
between:

```dad
اطبع("واحد"، "اثنان"، 3)
اطبع(2 + 2)
```

``` { .text .dad-out }
واحد اثنان 3
4
```

The Arabic comma `،` works just like `,`. Numbers are written with the digits `0-9`: if you
type `٣`, Ḍād tells you to write `3`.

## When you make a mistake

Try deleting the last quotation mark:

```text
اطبع("مرحبا بالعالم!)
```

The program won't run, and Ḍād tells you the line, the column and the mistake, in Arabic.
Mistakes are a normal part of programming. What matters is reading the message: it usually
points you to the fix.

## Exercise

Write a program that prints your name on one line, and your age on the next.

??? note "Solution"

    ```dad
    اطبع("اسمي سارة")
    اطبع("عمري"، 12، "سنة")
    ```

**Next:** [Values and names](02-values.md)
