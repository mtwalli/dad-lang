---
icon: lucide/shapes
---

# Examples

These sample programs live in the [`examples/`](https://github.com/mtwalli/dad-lang/tree/main/examples) folder of the repository.
Run any of them with `dad run`, or try them in the [playground](playground.md). Programs that
print something are followed by their output. Examples 5 and 6 have an interface: run them
with `dad dev` and open <http://localhost:8080>.

## 1. Hello — functions, default and named arguments

```dad
--8<-- "examples/01-hello.dad"
```

Output:

``` { .text .dad-out }
--8<-- "examples/output/01-hello.txt"
```

## 2. Basics — expressions, ranges, lambdas

```dad
--8<-- "examples/02-basics.dad"
```

Output:

``` { .text .dad-out }
--8<-- "examples/output/02-basics.txt"
```

## 3. Shapes — value types, enums, exhaustive matching

```dad
--8<-- "examples/03-shapes.dad"
```

Output:

``` { .text .dad-out }
--8<-- "examples/output/03-shapes.txt"
```

## 4. Safety — optionals, errors, contracts, extensions

```dad
--8<-- "examples/04-safety.dad"
```

Output:

``` { .text .dad-out }
--8<-- "examples/output/04-safety.txt"
```

## 5. To-do app — data, server and interface in one file

```dad
--8<-- "examples/05-todo.dad"
```

## 6. Counters — an interface with no server

```dad
--8<-- "examples/06-counter.dad"
```

## 7. A program in three files

`نماذج.dad` declares the products, `أسعار.dad` the discounts, and `الرئيسي.dad` imports
both. Run it with `dad run examples/07-modules/الرئيسي.dad`. See
[Programs in several files](modules.md).

```dad title="نماذج.dad"
--8<-- "examples/07-modules/نماذج.dad"
```

```dad title="أسعار.dad"
--8<-- "examples/07-modules/أسعار.dad"
```

```dad title="الرئيسي.dad"
--8<-- "examples/07-modules/الرئيسي.dad"
```

Output:

``` { .text .dad-out }
--8<-- "examples/output/07-modules.txt"
```

## 8. A program with pages

Three pages with their own addresses, a navigation bar, and `انتقل` after adding and saving.
Run it with `dad dev examples/08-pages.dad`. See [Pages](ui.md#pages).

```dad
--8<-- "examples/08-pages.dad"
```

## 9. A library — the elements, and a theme

Tabs, a form with a drop-down list, radio buttons, a date and a text area, a table of
records with a button on each row, a dialog to confirm, an alert, icons, a theme with
`سمة`, and a stylesheet of its own. Run it with `dad dev examples/09-library.dad`. See
[Interface](ui.md) and [Customizing the look](styling.md).

```dad
--8<-- "examples/09-library.dad"
```

```css title="09-library.css"
--8<-- "examples/09-library.css"
```

## 10. Signing up — a form that checks its fields

A `نموذج` whose fields have rules: required fields, an email, a length, a password typed
twice, a country to choose, terms to accept, and an email that must not be taken (a server
function). The button runs only when every field is right. Run it with
`dad dev examples/10-signup.dad`. See [Forms](ui.md#forms).

```dad
--8<-- "examples/10-signup.dad"
```
