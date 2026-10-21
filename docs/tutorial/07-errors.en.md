# 7. When things go wrong

Programs deal with an imperfect world: a user who didn't give an email address, a file that
isn't there, a number that isn't valid. This lesson is about handling that calmly.

## `عدم`: when there is no value

`عدم` means «nothing». In Ḍād, a name only accepts it if you say so explicitly, with `؟`
after the type:

```dad
نوع مستخدم(الاسم: نص، البريد: نص؟ = عدم)

ثابت هدى = مستخدم("هدى"، البريد: "huda@example.com")
ثابت سالم = مستخدم("سالم")

اطبع(هدى.البريد، سالم.البريد)
```

``` { .text .dad-out }
huda@example.com عدم
```

`نص؟` means «text, or nothing». So the type alone tells you a value might be missing
(here, a user's email, `البريد`), and you aren't caught by surprise.

### `؟.` and `؟؟`

```dad
نوع مستخدم(الاسم: نص، البريد: نص؟ = عدم)

دالة عرض_البريد(م: مستخدم): نص = م.البريد ؟؟ "(بلا بريد)"
دالة طول_البريد(م: مستخدم؟): صحيح = م؟.البريد؟.طول ؟؟ 0

اطبع(عرض_البريد(مستخدم("هدى"، البريد: "huda@example.com")))
اطبع(عرض_البريد(مستخدم("سالم")))
اطبع(طول_البريد(مستخدم("هدى"، البريد: "huda@example.com"))، طول_البريد(عدم))
```

``` { .text .dad-out }
huda@example.com
(بلا بريد)
16 0
```

- `أ ؟؟ ب`: take `أ`, and if it's nothing, take `ب`.
- `م؟.البريد`: if `م` is nothing, the result is nothing; otherwise, its email. That way the
  program doesn't stop when you ask «nothing» for something.

## Errors while running

Some mistakes only show up while the program runs, such as asking a list of three for its
tenth item. The program then stops and tells you where and why:

```dad
ثابت الألوان = ["أحمر"، "أخضر"، "أزرق"]
اطبع(الألوان[1])
اطبع(الألوان[10])
اطبع("لن يُطبع هذا")
```

``` { .text .dad-out }
أخضر
خطأ أثناء التشغيل [السطر 3، العمود 13]: مشكلة لم تُعالَج: الفهرس 10 خارج الحدود (الطول 3)
```

The message gives the line and column, and says the index 10 is out of bounds (the length
is 3). The last line («this won't be printed») never runs.

### `حاول` and `التقط`

When you expect that something might fail, put it in `حاول` («try»), and say what happens if
it fails in `التقط` («catch»). Here, reading an age from text:

```dad
دالة اقرأ_العمر(النص: نص): صحيح {
    ثابت العمر = حاول { النص ك صحيح } التقط خ { -1 }
    أرجع العمر
}

اطبع(اقرأ_العمر("25"))
اطبع(اقرأ_العمر("خمسة وعشرون"))
```

``` { .text .dad-out }
25
-1
```

`خ` is the **problem** that happened, and `خ.رسالة` describes it.

### `ارفع`: raising a problem yourself

`ارفع` («raise») stops the work with a problem of your own. Here, a withdrawal (`اسحب`) that
asks for more than the balance (`الرصيد`):

```dad
دالة اسحب(الرصيد: عشري، المبلغ: عشري): عشري {
    إذا المبلغ > الرصيد {
        ارفع مشكلة("الرصيد لا يكفي: معك {الرصيد} وتريد {المبلغ}")
    }
    أرجع الرصيد - المبلغ
}

اطبع(اسحب(100، 30))
حاول {
    اطبع(اسحب(100، 250))
} التقط خ {
    اطبع("لم تتم العملية:"، خ.رسالة)
}
```

``` { .text .dad-out }
70.0
لم تتم العملية: الرصيد لا يكفي: معك 100.0 وتريد 250.0
```

## Find mistakes before running

`dad check` checks a program without running it, and shows everything it finds. It is what
VS Code does as you type:

```sh
dad check برنامج.dad
```

```dad
// خطأ مقصود
نوع نقطة(س: عشري، ص: عشري)

ثابت ن = نقطة(1، 2)
اطبع(ن.ع)
اطبع(نقطه(3، 4))
```

``` { .text .dad-out }
خطأ [السطر 5، العمود 7]: القيمة من النوع «نقطة» لا تحتوي «ع»؛ هل تقصد «س»؟
خطأ [السطر 6، العمود 6]: «نقطه» غير معرّف؛ هل تقصد «نقطة»؟
```

Notice «هل تقصد …؟» («did you mean …?»): Ḍād suggests the name closest to what you wrote, `س`
for the field and `نقطة` for the type.

## Exercise

Write a function `قسمة_آمنة(أ: عشري، ب: عشري): عشري؟` («safe division») that returns `عدم` if
`ب` is zero, and the result of the division otherwise. Then print `قسمة_آمنة(10، 4)` and
`قسمة_آمنة(1، 0) ؟؟ "غير معرّفة"` («undefined»).

??? note "Solution"

    ```dad
    دالة قسمة_آمنة(أ: عشري، ب: عشري): عشري؟ {
        إذا ب == 0 { أرجع عدم }
        أرجع أ / ب
    }

    اطبع(قسمة_آمنة(10، 4))
    اطبع(قسمة_آمنة(1، 0) ؟؟ "غير معرّفة")
    ```

    ``` { .text .dad-out }
    2.5
    غير معرّفة
    ```

**Next:** [Your first page](08-page.md)
