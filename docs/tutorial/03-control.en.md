# 3. Decisions and loops

So far the program runs every line once, from top to bottom. In this lesson it chooses what
to do, and repeats work.

## `إذا`: choosing

```dad
ثابت الحرارة = 34

إذا الحرارة > 30 {
    اطبع("الجو حار، اشرب ماءً كثيراً")
}
```

``` { .text .dad-out }
الجو حار، اشرب ماءً كثيراً
```

`إذا` means «if». What's between `{` and `}` is a **block**, and it only runs when the
condition is true: here, when the temperature is above 30, the program says «it's hot, drink
plenty of water». Add what happens otherwise with `وإلا` («else»), and ask more than one
question:

```dad
ثابت الدرجة = 72

إذا الدرجة >= 90 {
    اطبع("ممتاز")
} وإلا إذا الدرجة >= 50 {
    اطبع("ناجح")
} وإلا {
    اطبع("حاول مرة أخرى")
}
```

``` { .text .dad-out }
ناجح
```

### `إذا` gives a value

In Ḍād, `إذا` returns a value: the last thing in the block it chose. So you can write:

```dad
ثابت العمر = 15
ثابت الفئة = إذا العمر < 13 { "طفل" } وإلا إذا العمر < 18 { "مراهق" } وإلا { "بالغ" }
اطبع(الفئة)
```

``` { .text .dad-out }
مراهق
```

Combine conditions with `و` («and») and `أو` («or»), and negate them with `ليس` («not»):

```dad
ثابت عطلة = صح
ثابت الجو_جميل = خطأ

إذا عطلة و ليس الجو_جميل {
    اطبع("يوم مناسب للقراءة في البيت")
}
```

``` { .text .dad-out }
يوم مناسب للقراءة في البيت
```

## `لكل`: repeating a known number of times

```dad
لكل ع من 1 إلى 5 {
    اطبع("المرة رقم"، ع)
}
```

``` { .text .dad-out }
المرة رقم 1
المرة رقم 2
المرة رقم 3
المرة رقم 4
المرة رقم 5
```

`لكل … من … إلى` means «for each … from … to». The name `ع` takes the values 1, then 2… up
to 5. There are other ways to count:

| You write | You get |
|---|---|
| `من 1 إلى 5` | 1, 2, 3, 4, 5 |
| `من 0 حتى 5` | 0, 1, 2, 3, 4 (without 5) |
| `من 0 إلى 10 بخطوة 5` | 0, 5, 10 |
| `من 3 إلى 1 بخطوة -1` | 3, 2, 1 |

An example: the times table for 7.

```dad
لكل ع من 1 إلى 10 {
    اطبع("7 × {ع} = {7 * ع}")
}
```

``` { .text .dad-out }
7 × 1 = 7
7 × 2 = 14
7 × 3 = 21
7 × 4 = 28
7 × 5 = 35
7 × 6 = 42
7 × 7 = 49
7 × 8 = 56
7 × 9 = 63
7 × 10 = 70
```

`لكل … في` («for each … in») goes through the letters of a text too:

```dad
لكل حرف في "ضاد" {
    اطبع(حرف)
}
```

``` { .text .dad-out }
ض
ا
د
```

## `طالما`: repeating until something happens

When you don't know the number of times in advance, use `طالما` («while»). The block
repeats while the condition is true:

```dad
// كم سنة حتى يتضاعف المال بفائدة 10٪؟
متغير المال = 1000.0
متغير السنوات = 0

طالما المال < 2000 {
    المال *= 1.1
    السنوات += 1
}
اطبع("بعد"، السنوات، "سنوات يصير المال"، المال.تقريب(2))
```

``` { .text .dad-out }
بعد 8 سنوات يصير المال 2143.59
```

This asks how many years it takes money to double at 10% interest. `تقريب(2)` rounds to two
decimal places.

Careful: if the condition stays true forever, the work repeats with no end. Stop the
program with `Ctrl+C`; in the playground, Ḍād stops it for you and says why.

### `توقف` and `تابع`

`توقف` («stop») leaves the loop at once, and `تابع` («continue») skips to the next round:

```dad
لكل ع من 1 إلى 10 {
    إذا ع % 2 == 0 { تابع } // تخطَّ الأعداد الزوجية
    إذا ع > 7 { توقف }
    اطبع(ع)
}
```

``` { .text .dad-out }
1
3
5
7
```

## Exercise

1. Print the numbers from 1 to 20, but write «فز» instead of each number divisible by 3,
   «بز» instead of each number divisible by 5, and «فز بز» if it is divisible by both.
2. Add up the numbers from 1 to 100 and print the total.

??? note "Solution"

    ```dad
    لكل ع من 1 إلى 20 {
        إذا ع % 15 == 0 {
            اطبع("فز بز")
        } وإلا إذا ع % 3 == 0 {
            اطبع("فز")
        } وإلا إذا ع % 5 == 0 {
            اطبع("بز")
        } وإلا {
            اطبع(ع)
        }
    }

    متغير المجموع = 0
    لكل ع من 1 إلى 100 {
        المجموع += ع
    }
    اطبع("المجموع:"، المجموع)
    ```

**Next:** [Functions](04-functions.md)
