# Learn Ḍād

This tutorial is for people starting to program from scratch, and for programmers who
know another language and want to see Ḍād. In the first lesson you write one line that
prints a greeting. In the last one you put a complete to-do app online, with its interface
and its data. The code is in Arabic all the way.

The tutorial is also [in Arabic](https://mtwalli.github.io/dad-lang/ar/tutorial/): switch
with the language button at the top of the page.

## The lessons

1. [Getting started](01-start.md): install Ḍād, then write and run your first program.
2. [Values and names](02-values.md): numbers and text, `ثابت` and `متغير`.
3. [Decisions and loops](03-control.md): `إذا`, `لكل` and `طالما`.
4. [Functions](04-functions.md): give a piece of work a name and use it again.
5. [Lists and maps](05-lists.md): collections of values, filtered and transformed.
6. [Types and enums](06-types.md): describe the things in your world.
7. [When things go wrong](07-errors.md): missing values, errors, and Ḍād's messages.
8. [Your first page](08-page.md): an interface with buttons and fields.
9. [Saving data](09-data.md): tables, server functions, and a to-do app.
10. [Going online](10-online.md): a program in several files, a static site, and a server
    for everyone.

## How to read it

Each lesson has short examples. Copy each one into a file and run it, then change
something and run it again: that is how programmers learn. Many examples are followed by
what the program prints, so you can compare:

```dad
اطبع("السلام عليكم")
```

``` { .text .dad-out }
السلام عليكم
```

Each lesson ends with an exercise or two. Try them before you open the solution.

The names in the examples are Arabic words. Each lesson explains the ones it introduces,
and the [specification](../SPEC.md#3-keywords-reserved) lists every keyword with its
English equivalent.

Don't want to install anything yet? Try the examples in the [playground](../playground.md),
right in your browser, and install Ḍād when you reach lesson 8.
