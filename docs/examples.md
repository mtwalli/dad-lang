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
