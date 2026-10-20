# 6. Types and enums

You know some ready-made types: `صحيح`, `نص` and `منطقي`. But a library program deals with
**books**, and a school program with **students**. In this lesson you make your own types.

## `نوع`: a thing with properties

```dad
نوع كتاب(العنوان: نص، المؤلف: نص، الصفحات: صحيح)

ثابت الأول = كتاب("الأيام"، "طه حسين"، 160)
ثابت الثاني = كتاب(العنوان: "رجال في الشمس"، المؤلف: "غسان كنفاني"، الصفحات: 112)

اطبع(الأول.العنوان، "—"، الأول.المؤلف)
اطبع(الثاني)
```

``` { .text .dad-out }
الأيام — طه حسين
كتاب(العنوان: «رجال في الشمس»، المؤلف: «غسان كنفاني»، الصفحات: 112)
```

`نوع كتاب(…)` («type book») describes a book: it has a title, an author and a number of
pages, its **fields**. Then you make books the way you call a function, and read each field
with a dot: `الأول.العنوان` («the first one's title»).

### Functions inside the type

Put the functions that belong to the type between `{ }` after it. Inside them, the fields
are read by name. Here is a rectangle with its area, its perimeter, and whether it is a
square:

```dad
نوع مستطيل(الطول: عشري، العرض: عشري) {
    دالة المساحة() = الطول * العرض
    دالة المحيط() = 2 * (الطول + العرض)
    دالة مربع() = الطول == العرض
}

ثابت م = مستطيل(4، 2.5)
اطبع("المساحة"، م.المساحة()، "والمحيط"، م.المحيط()، "مربع؟"، م.مربع())
```

``` { .text .dad-out }
المساحة 10.0 والمحيط 13.0 مربع؟ خطأ
```

### Fields don't change; copies do

The fields of a type are constant, so a thing can't change behind your back. When you want
something slightly different, make a **copy** of it with `نسخة`:

```dad
نوع طالب(الاسم: نص، الصف: صحيح)

ثابت أحمد = طالب("أحمد"، 5)
ثابت أحمد_العام_القادم = أحمد.نسخة(الصف: 6)
اطبع(أحمد)
اطبع(أحمد_العام_القادم)
```

``` { .text .dad-out }
طالب(الاسم: «أحمد»، الصف: 5)
طالب(الاسم: «أحمد»، الصف: 6)
```

## `تعداد`: one of a few choices

Some values can only be one of a few options: a traffic light is red (`أحمر`), yellow
(`أصفر`) or green (`أخضر`).

```dad
تعداد إشارة { أحمر، أصفر، أخضر }

دالة ماذا_أفعل(إ: إشارة): نص = طابق إ {
    أحمر => "قف"
    أصفر => "استعد"
    أخضر => "انطلق"
}

اطبع(ماذا_أفعل(أحمر))
اطبع(ماذا_أفعل(أخضر))
```

``` { .text .dad-out }
قف
انطلق
```

`تعداد` means «enum». `طابق` («match») compares the value with each case, and picks what
follows `=>`. Ḍād makes sure you haven't forgotten a case. Delete the `أصفر` line and see:

```dad
// خطأ مقصود
تعداد إشارة { أحمر، أصفر، أخضر }

دالة ماذا_أفعل(إ: إشارة): نص = طابق إ {
    أحمر => "قف"
    أخضر => "انطلق"
}
```

``` { .text .dad-out }
خطأ [السطر 4، العمود 32]: «طابق» لا تغطي كل حالات «إشارة»: تنقص «أصفر». أضفها، أو أضف «_ => …» لما تبقى
```

The message says `طابق` doesn't cover every case of «إشارة»: «أصفر» is missing. Add it, or
add `_ => …` for the rest.

### Cases that carry values

Each case of an enum can carry fields of its own. Here, shapes: a circle, a rectangle and a
triangle:

```dad
تعداد شكل {
    دائرة(نصف_القطر: عشري)
    مستطيل(الطول: عشري، العرض: عشري)
    مثلث(القاعدة: عشري، الارتفاع: عشري)
}

دالة المساحة(ش: شكل): عشري = طابق ش {
    دائرة(نق) => 3.14159 * نق * نق
    مستطيل(ط، ع) => ط * ع
    مثلث(ق، ر) => ق * ر / 2
}

ثابت الأشكال = [دائرة(1)، مستطيل(3، 4)، مثلث(6، 2)]
لكل ش في الأشكال {
    اطبع(ش، "مساحته"، المساحة(ش).تقريب(2))
}
```

``` { .text .dad-out }
دائرة(نصف_القطر: 1.0) مساحته 3.14
مستطيل(الطول: 3.0، العرض: 4.0) مساحته 12.0
مثلث(القاعدة: 6.0، الارتفاع: 2.0) مساحته 6.0
```

In `دائرة(نق) => …`, the name `نق` takes the radius, and you use it after `=>`.

## Exercise

1. Make a type `منتج(الاسم: نص، السعر: عشري، الكمية: صحيح)` (a product with a name, a price
   and a quantity) with a function `القيمة()` that returns price × quantity. Make a list of
   three products and print their total value.
2. Make an enum `يوم` with the days of the week, and a function `عطلة(ي: يوم): منطقي` that
   returns `صح` for Friday and Saturday (`الجمعة` and `السبت`).

??? note "Solution"

    ```dad
    نوع منتج(الاسم: نص، السعر: عشري، الكمية: صحيح) {
        دالة القيمة() = السعر * الكمية
    }

    ثابت المخزن = [منتج("قلم"، 2.5، 40)، منتج("دفتر"، 6، 25)، منتج("حقيبة"، 45، 3)]
    اطبع("قيمة المخزن:"، المخزن.حول { هو.القيمة() }.مجموع())

    تعداد يوم { السبت، الأحد، الاثنين، الثلاثاء، الأربعاء، الخميس، الجمعة }

    دالة عطلة(ي: يوم): منطقي = طابق ي {
        الجمعة => صح
        السبت => صح
        _ => خطأ
    }

    اطبع(عطلة(الجمعة)، عطلة(الاثنين))
    ```

    ``` { .text .dad-out }
    قيمة المخزن: 385.0
    صح خطأ
    ```

    In `طابق`, `_` means «any other case».

**Next:** [When things go wrong](07-errors.md)
