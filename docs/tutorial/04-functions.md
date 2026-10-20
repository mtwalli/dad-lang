# 4. Functions

A function is a piece of work with a name. You write it once, then use it whenever you
need it, with different values each time. So far you have used a function someone else
wrote, `اطبع`. In this lesson you write your own.

## A first function

```dad
دالة رحّب(اسم: نص) {
    اطبع("أهلاً يا {اسم}، نورت!")
}

رحّب("مريم")
رحّب("يوسف")
```

``` { .text .dad-out }
أهلاً يا مريم، نورت!
أهلاً يا يوسف، نورت!
```

- `دالة` («function») starts the definition, followed by the function's name, here `رحّب`
  («welcome»).
- Between the brackets are the **parameters**: what the function needs to work. Each has a
  name and a type: `اسم: نص` («name: text»).
- What's between `{ }` is the function's **body**. It only runs when you **call** the
  function: `رحّب("مريم")`.

## A function that returns a value

Most functions don't print. They work something out and give it back to the caller:

```dad
دالة مساحة_مستطيل(الطول: عشري، العرض: عشري): عشري {
    أرجع الطول * العرض
}

ثابت م = مساحة_مستطيل(4، 2.5)
اطبع("المساحة:"، م)
اطبع("مساحة غرفتين:"، مساحة_مستطيل(3، 3) * 2)
```

``` { .text .dad-out }
المساحة: 10.0
مساحة غرفتين: 18.0
```

`: عشري` after the brackets is the type of what the function returns, and `أرجع` («return»)
ends the function and gives back the value.

When the body is a single line, write it after `=`:

```dad
دالة مربع(س: صحيح): صحيح = س * س
دالة زوجي(س: صحيح) = س % 2 == 0

اطبع(مربع(9)، زوجي(4)، زوجي(7))
```

``` { .text .dad-out }
81 صح خطأ
```

## Defaults and named arguments

Give a parameter a default value, and mentioning it becomes optional. When calling, you can
also name a parameter, which makes the call clearer. Here is a final price with tax
(`الضريبة`) and discount (`الخصم`):

```dad
دالة سعر_نهائي(السعر: عشري، الضريبة: عشري = 0.15، الخصم: عشري = 0): عشري {
    ثابت بعد_الخصم = السعر * (1 - الخصم)
    أرجع (بعد_الخصم * (1 + الضريبة)).تقريب(2)
}

اطبع(سعر_نهائي(100))
اطبع(سعر_نهائي(100، الخصم: 0.2))
اطبع(سعر_نهائي(100، الضريبة: 0، الخصم: 0.5))
```

``` { .text .dad-out }
115.0
92.0
50.0
```

Ḍād checks every call before running. If you give a value of the wrong type, or misspell
the function's name, it tells you, and suggests the right name:

```dad
// خطأ مقصود
دالة مربع(س: صحيح): صحيح = س * س

اطبع(مربع("خمسة"))
اطبع(مربعات(5))
```

``` { .text .dad-out }
خطأ [السطر 4، العمود 11]: المعامل «س» في «مربع»: توقعت قيمة من النوع «صحيح» لكن وجدت «نص»
خطأ [السطر 5، العمود 6]: «مربعات» غير معرّف؛ هل تقصد «مربع»؟
```

The first message says the parameter «س» expected a `صحيح` but got a `نص`; the second says
«مربعات» isn't defined, and asks «did you mean «مربع»?».

## Functions are values too

You can give a function to another function. An **anonymous** function is written between
`{ }`. If it takes one value, that value is called `هو` («it») inside; otherwise write the
names of its values before `=>`:

```dad
دالة طبّق_مرتين(د: (صحيح) => صحيح، س: صحيح): صحيح = د(د(س))

اطبع(طبّق_مرتين({ هو + 10 }، 1))
اطبع(طبّق_مرتين({ هو * هو }، 3))

ثابت اجمع = { أ، ب => أ + ب }
اطبع(اجمع(2، 3))
```

``` { .text .dad-out }
21
81
5
```

`طبّق_مرتين` («apply twice») takes a function `د` and a number. You'll use anonymous
functions a lot with lists in the next lesson.

## A function that calls itself

A function can call itself, as long as it comes to an end:

```dad
دالة مضروب(ن: صحيح): صحيح = إذا ن <= 1 { 1 } وإلا { ن * مضروب(ن - 1) }

اطبع("مضروب 5 ="، مضروب(5))
```

``` { .text .dad-out }
مضروب 5 = 120
```

## Exercise

1. Write a function `أكبر(أ: صحيح، ب: صحيح): صحيح` that returns the larger of the two
   numbers.
2. Write a function `حرارة_فهرنهايت(مئوية: عشري): عشري` that converts a temperature from
   Celsius to Fahrenheit: F = C × 9 ÷ 5 + 32. Try it on 0, 100 and 37.

??? note "Solution"

    ```dad
    دالة أكبر(أ: صحيح، ب: صحيح): صحيح = إذا أ > ب { أ } وإلا { ب }

    دالة حرارة_فهرنهايت(مئوية: عشري): عشري = مئوية * 9 / 5 + 32

    اطبع(أكبر(3، 8)، أكبر(10، 2))
    اطبع(حرارة_فهرنهايت(0)، حرارة_فهرنهايت(100)، حرارة_فهرنهايت(37))
    ```

    ``` { .text .dad-out }
    8 10
    32.0 212.0 98.6
    ```

**Next:** [Lists and maps](05-lists.md)
