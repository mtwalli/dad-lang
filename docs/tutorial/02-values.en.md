# 2. Values and names

Every program works with **values**: numbers, text and more. To use a value more than
once, you give it a **name**.

## `ثابت`: a name for a value that doesn't change

```dad
ثابت المدينة = "القاهرة"
ثابت السكان = 22
اطبع(المدينة، "فيها"، السكان، "مليون نسمة تقريباً")
```

``` { .text .dad-out }
القاهرة فيها 22 مليون نسمة تقريباً
```

`ثابت` means «constant». After `ثابت المدينة = "القاهرة"` («the city is Cairo»), the name
`المدينة` means that value in the rest of the program. Names can be written in Arabic or
English, and can contain `_`, as in `عدد_الطلاب` («number of students»).

## `متغير`: a name for a value that changes

```dad
متغير النقاط = 0
النقاط = النقاط + 10
النقاط += 5
اطبع("النقاط:"، النقاط)
```

``` { .text .dad-out }
النقاط: 15
```

`متغير` means «variable». `النقاط += 5` is short for `النقاط = النقاط + 5`, and so are
`-=`, `*=` and `/=`.

Why two different words? Most values in a program never change, and when you say `ثابت`,
Ḍād protects you from changing one by mistake. Try it:

```dad
// خطأ مقصود
ثابت الاسم = "علي"
الاسم = "عمر"
```

``` { .text .dad-out }
خطأ [السطر 3، العمود 1]: لا يمكن تغيير «الاسم» لأنه ثابت — عرّفه بـ«متغير» إذا أردت تغييره
```

The message says that «الاسم» can't change because it is a constant, and suggests declaring
it with «متغير» if you want to change it. Use `ثابت` by default, and switch to `متغير` when you
really need to change the value.

## Kinds of values

Every value has a **type**. These are the main ones:

| Type | What it is | Examples |
|---|---|---|
| `صحيح` | a whole number | `0`, `42`, `-7` |
| `عشري` | a number with a fraction | `3.14`, `2.0` |
| `نص` | text | `"مرحبا"`, `""` |
| `منطقي` | true or false | `صح` (true), `خطأ` (false) |

### Arithmetic

```dad
اطبع(7 + 2، 7 - 2، 7 * 2)
اطبع(7 / 2، 7 % 2)
اطبع((7 / 2) ك صحيح، 2.5 * 2)
```

``` { .text .dad-out }
9 5 14
3.5 1
3 5.0
```

- Division `/` always gives a decimal, as in maths: `7 / 2` is `3.5`.
- `%` is the remainder: `7 % 2` is `1`.
- `ك` («as») converts a value to another type: `(7 / 2) ك صحيح` is `3`.
- A decimal always prints with a decimal point, as in `5.0`, so you can tell its type.

### Text

Put a name between braces `{…}` inside text, and its value appears there:

```dad
ثابت الاسم = "ليلى"
ثابت العمر = 11
اطبع("{الاسم} عمرها {العمر} سنة، وبعد سنة يصير {العمر + 1}")
اطبع(الاسم.طول، الاسم + " وأختها")
```

``` { .text .dad-out }
ليلى عمرها 11 سنة، وبعد سنة يصير 12
4 ليلى وأختها
```

`الاسم.طول` is the number of letters: `طول` means «length». The dot `.` asks a value for
something it has, «the length of the name», and you'll see it a lot.

### Converting between types

The text `"12"` isn't a number, even though it looks like one. Convert it with `ك`:

```dad
ثابت مدخل = "12"
اطبع(مدخل ك صحيح + 1)
اطبع(5 ك نص + "!")
```

``` { .text .dad-out }
13
5!
```

### True and false

```dad
اطبع(3 > 2، 3 == 4، 3 != 4)
اطبع(صح و خطأ، صح أو خطأ، ليس صح)
```

``` { .text .dad-out }
صح خطأ صح
خطأ صح خطأ
```

`==` asks «are they equal?», and `!=` asks «are they different?». A single `=` gives a name
its value: `ثابت س = 1`. `و`, `أو` and `ليس` mean «and», «or» and «not».

## Exercise

1. Write a program with two constants, `الطول` and `العرض` (length and width), for a room
   4.5 metres long and 3 metres wide. Print its area like this:
   `مساحة الغرفة 13.5 متراً مربعاً`.
2. You have 50 riyals, and a book costs 12. Print how many books you can buy, and how much
   is left.

??? note "Solution"

    ```dad
    ثابت الطول = 4.5
    ثابت العرض = 3
    اطبع("مساحة الغرفة {الطول * العرض} متراً مربعاً")

    ثابت المال = 50
    ثابت الثمن = 12
    اطبع("كتب:"، (المال / الثمن) ك صحيح، "والباقي:"، المال % الثمن)
    ```

    ``` { .text .dad-out }
    مساحة الغرفة 13.5 متراً مربعاً
    كتب: 4 والباقي: 2
    ```

**Next:** [Decisions and loops](03-control.md)
