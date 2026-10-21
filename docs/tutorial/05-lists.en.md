# 5. Lists and maps

## The list

A list is an ordered collection of values, written between square brackets:

```dad
ثابت الفواكه = ["تفاح"، "موز"، "برتقال"]

اطبع(الفواكه)
اطبع("عددها:"، الفواكه.عدد)
اطبع("الأولى:"، الفواكه[0]، "والأخيرة:"، الفواكه.آخر)
```

``` { .text .dad-out }
[«تفاح»، «موز»، «برتقال»]
عددها: 3
الأولى: تفاح والأخيرة: برتقال
```

- Each item has a **number**, starting at 0: `الفواكه[0]` is «تفاح» (apple), and
  `الفواكه[1]` is «موز» (banana).
- `عدد` is the number of items, and `أول` and `آخر` are the first and the last.

### Going through a list

```dad
ثابت الدرجات = [95، 72، 40، 88]

لكل د في الدرجات {
    اطبع("درجة:"، د)
}
```

``` { .text .dad-out }
درجة: 95
درجة: 72
درجة: 40
درجة: 88
```

### Adding an item

A list held in a `ثابت` keeps its name, but you can still add to it with `أضف` («add»):

```dad
ثابت المشتريات = ["خبز"]
المشتريات.أضف("حليب")
المشتريات.أضف("تمر")
اطبع(المشتريات، المشتريات.يحتوي("تمر"))
```

``` { .text .dad-out }
[«خبز»، «حليب»، «تمر»] صح
```

`يحتوي` («contains») asks whether an item is in the list.

## Asking the list

This is where the anonymous functions from the last lesson shine. A list knows how to filter
and transform itself:

```dad
ثابت الدرجات = [95، 72، 40، 88، 55]

ثابت الناجحون = الدرجات.رشح { هو >= 50 }
اطبع("الناجحون:"، الناجحون)

ثابت مع_المكافأة = الدرجات.حول { هو + 5 }
اطبع("مع المكافأة:"، مع_المكافأة)

اطبع("المجموع:"، الدرجات.مجموع()، "المتوسط:"، الدرجات.مجموع() / الدرجات.عدد)
اطبع("مرتبة:"، الدرجات.رتب())
اطبع("أعلاها:"، أقصى(الدرجات)، "أدناها:"، أدنى(الدرجات))
اطبع("هل رسب أحد؟"، الدرجات.أي { هو < 50 })
```

``` { .text .dad-out }
الناجحون: [95، 72، 88، 55]
مع المكافأة: [100، 77، 45، 93، 60]
المجموع: 350 المتوسط: 70.0
مرتبة: [40، 55، 72، 88، 95]
أعلاها: 95 أدناها: 40
هل رسب أحد؟ صح
```

| You write | You get |
|---|---|
| `ق.رشح { … }` (filter) | a list of the items the condition is true for |
| `ق.حول { … }` (map) | a list with each item transformed |
| `ق.عدد { … }` (count) | how many items the condition is true for |
| `ق.أي { … }` and `ق.كل { … }` (any, all) | is the condition true for at least one item? For every item? |
| `ق.مجموع()`, `ق.رتب()` and `ق.عكس()` (sum, sort, reverse) | the total, a sorted list and a reversed list |
| `ق.ضم("، ")` (join) | a text with the items and the separator between them |

`أقصى` and `أدنى` give the largest and the smallest. You can chain these questions:

```dad
ثابت الكلمات = ["ضاد"، "لغة"، "عربية"، "للبرمجة"]
اطبع(الكلمات.رشح { هو.طول > 3 }.حول { "«" + هو + "»" }.ضم(" و"))
```

``` { .text .dad-out }
«عربية» و«للبرمجة»
```

## The map

A map links a **key** to a **value**, the way a phone book links a name to a number. Here,
countries to their capitals:

```dad
متغير العواصم = ["مصر": "القاهرة"، "المغرب": "الرباط"]
العواصم["الأردن"] = "عمّان"

اطبع(العواصم["المغرب"])
اطبع(العواصم.عدد، العواصم.مفاتيح)
لكل البلد في العواصم {
    اطبع("عاصمة {البلد} هي {العواصم[البلد]}")
}
اطبع(العواصم["فرنسا"] ؟؟ "لا أعرف")
```

``` { .text .dad-out }
الرباط
3 [«مصر»، «المغرب»، «الأردن»]
عاصمة مصر هي القاهرة
عاصمة المغرب هي الرباط
عاصمة الأردن هي عمّان
لا أعرف
```

`[:]` is an empty map. If you ask a map for a key it doesn't have, you get `عدم`, «nothing».
`؟؟` means «if that's nothing, take this instead». You'll learn more about `عدم` in lesson 7.

## Exercise

1. You have a list of prices, `[20، 3، 40، 8]`. Print the prices above 5 after adding 15% tax
   to each, rounded to two decimal places.
2. Use a map to count how many times each letter appears in the word «مممتاز».

??? note "Solution"

    ```dad
    ثابت الأسعار = [20، 3، 40، 8]
    اطبع(الأسعار.رشح { هو > 5 }.حول { (هو * 1.15).تقريب(2) })

    ثابت التكرار = [:] // قاموس فارغ
    لكل حرف في "مممتاز" {
        التكرار[حرف] = (التكرار[حرف] ؟؟ 0) + 1
    }
    اطبع(التكرار)
    ```

    ``` { .text .dad-out }
    [23.0، 46.0، 9.2]
    [«م»: 3، «ت»: 1، «ا»: 1، «ز»: 1]
    ```

**Next:** [Types and enums](06-types.md)
