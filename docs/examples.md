# Examples

These sample programs live in the [`examples/`](https://github.com/mtwalli/dad-lang/tree/main/examples) folder of the repository.
Examples 1 to 4 run today with `dad run` or in the [playground](playground.md). Each one is
followed by its output. Example 5 needs tables, a server and components, which come in Phases 2 and 3.

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

## 5. To-do app — data, server and UI in one file

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
