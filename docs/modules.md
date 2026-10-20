# Programs in several files

A program can be split into files. Each file is a **module**: it declares its own names,
and gives other files the ones they import.

```dad title="أشكال.dad"
نوع نقطة(س: عشري، ص: عشري)

تعداد شكل { دائرة(نق: عشري)، مربع(ض: عشري) }

دالة مساحة(ش: شكل): عشري = طابق ش {
    دائرة(نق) => 3.14159 * نق * نق
    مربع(ض) => ض * ض
}

خاص ثابت الدقة = 2
```

```dad title="الرئيسي.dad"
استورد شكل، مساحة من "./أشكال"

اطبع(مساحة(دائرة(1))، مساحة(مربع(3)))
```

```sh
dad run الرئيسي.dad
```

Run, check, build or serve the **main** file. `dad` finds the files it imports, and the
files those import, by itself.

## Importing — `استورد`

```dad
استورد اسم، اسم_آخر من "./ملف"
```

- The file is found **next to the importing file**. `"./أشكال"`, `"أشكال"` and
  `"./أشكال.dad"` all mean `أشكال.dad` in the same folder. `"./نماذج/منتج"` looks in a
  sub-folder, and `"../مشترك"` in the folder above.
- Write imports at the top of the file.
- Importing a `تعداد` brings its cases too. After `استورد شكل`, you can write
  `دائرة(1)` and match on `مربع(ض)`.
- An imported name is a constant in the importing file, and it holds the value the
  other file gave it. To share a changing value, import a function that returns it. For
  example, export `دالة العدد() = عدد` instead of `متغير عدد`.

## Keeping names private — `خاص`

Everything a file declares at its top is exported: functions, types, enums, tables,
components and constants. Mark the ones that are only for the file itself with `خاص`:

```dad
خاص دالة مساعد() = …
خاص ثابت الدقة = 2
```

Two files can each have a private `مساعد` without a clash.

## Names the whole program shares

Some names are known by the whole program, so they can't be repeated in two files:

- types (`نوع`), tables (`جدول`), enums (`تعداد`) and their cases;
- server functions (`خادم دالة`), which the page calls by name.

A file must still import a type before it uses it. To write `ن: نقطة` in a file,
first write `استورد نقطة من "./أشكال"`.

## Problems the checker reports

| You wrote | The checker says |
|---|---|
| a name the file doesn't export | `«منتجج» غير موجود في «نماذج»؛ هل تقصد «منتج»؟` |
| a `خاص` name | `«الضريبة» خاص بالملف «نماذج»، فلا يمكن استيراده` |
| a missing file | `لا يوجد ملف «./مفقود» (بحثت عن «مفقود.dad»)` |
| files that import each other | `استيراد دائري: «أ» يستورد «ب»، و«ب» يستورد «أ»…` |
| a type from another file, not imported | `النوع «منتج» معرّف في «نماذج»؛ استورده أولاً…` |

Errors say which file they are in, whether `dad check` finds them or they happen while
the program runs in `dad run` or `dad dev`:

```text
أسعار.dad: خطأ أثناء التشغيل [السطر 9، العمود 40]: …
```

## In the editor and the tools

- **VS Code** checks a saved file together with the files it imports. Saving one file
  checks the files that import it again.
- **Go to definition** on an imported name opens the file that declares it, and hover
  shows its declaration.
- **`dad dev`** reloads when any file of the program changes.
- **`dad build`** compiles every file into one `app.js`.
- The **playground** has a single file, so `استورد` doesn't work there.
