---
icon: lucide/shapes
---

# الأمثلة

هذه البرامج في مجلد [`examples/`](https://github.com/mtwalli/dad-lang/tree/main/examples) في
المستودع. شغّل أياً منها بـ`dad run`، أو جرّبه في [الملعب](playground.md). وتحت البرامج التي
تطبع شيئاً ناتجُها. وللمثالين 5 و6 واجهة: شغّلهما بـ`dad dev` وافتح <http://localhost:8080>.

## 1. مرحبا: الدوال، والقيم الافتراضية، والمعاملات المسماة { #1-hello }

```dad
--8<-- "examples/01-hello.dad"
```

الناتج:

``` { .text .dad-out }
--8<-- "examples/output/01-hello.txt"
```

## 2. الأساسيات: التعابير، والمجالات، والدوال المجهولة { #2-basics }

```dad
--8<-- "examples/02-basics.dad"
```

الناتج:

``` { .text .dad-out }
--8<-- "examples/output/02-basics.txt"
```

## 3. الأشكال: الأنواع، والتعدادات، والمطابقة الشاملة { #3-shapes }

```dad
--8<-- "examples/03-shapes.dad"
```

الناتج:

``` { .text .dad-out }
--8<-- "examples/output/03-shapes.txt"
```

## 4. الأمان: القيم الاختيارية، والأخطاء، والعقود، والامتدادات { #4-safety }

```dad
--8<-- "examples/04-safety.dad"
```

الناتج:

``` { .text .dad-out }
--8<-- "examples/output/04-safety.txt"
```

## 5. تطبيق مهام: البيانات والخادم والواجهة في ملف واحد { #5-to-do-app }

```dad
--8<-- "examples/05-todo.dad"
```

## 6. عدّادات: واجهة بلا خادم { #6-counters }

```dad
--8<-- "examples/06-counter.dad"
```

## 7. برنامج في ثلاثة ملفات { #7-a-program-in-three-files }

يعرّف `نماذج.dad` المنتجات، و`أسعار.dad` الخصومات، ويستورد `الرئيسي.dad` كليهما. شغّله
بـ`dad run examples/07-modules/الرئيسي.dad`. وانظر [برنامج في عدة ملفات](modules.md).

```dad title="نماذج.dad"
--8<-- "examples/07-modules/نماذج.dad"
```

```dad title="أسعار.dad"
--8<-- "examples/07-modules/أسعار.dad"
```

```dad title="الرئيسي.dad"
--8<-- "examples/07-modules/الرئيسي.dad"
```

الناتج:

``` { .text .dad-out }
--8<-- "examples/output/07-modules.txt"
```

## 8. برنامج من عدة صفحات { #8-a-program-with-pages }

ثلاث صفحات لكل منها عنوانها، وشريط تنقّل، و`انتقل` بعد الإضافة والحفظ. شغّله
بـ`dad dev examples/08-pages.dad`. وانظر [الصفحات](ui.md#pages).

```dad
--8<-- "examples/08-pages.dad"
```

## 9. مكتبة: العناصر، وسمة { #9-a-library }

ألسنة، ونموذج فيه قائمة منسدلة وأزرار اختيار وتاريخ ومنطقة نص، وجدول سجلات بزر في كل صف،
ونافذة للتأكيد، وتنبيه، وأيقونات، وسمة بـ`سمة`، وملف تنسيق خاص. شغّله
بـ`dad dev examples/09-library.dad`. وانظر [الواجهة](ui.md) و[تخصيص المظهر](styling.md).

```dad
--8<-- "examples/09-library.dad"
```

```css title="09-library.css"
--8<-- "examples/09-library.css"
```

## 10. إنشاء حساب: نموذج يفحص حقوله { #10-signing-up }

`نموذج` لحقوله قواعد: حقول مطلوبة، وبريد، وطول، وكلمة سر تُكتب مرتين، وبلد يُختار، وشروط
يُوافق عليها، وبريد لا يكون مستعملاً (بدالة خادم). ولا يعمل الزر إلا حين تصح الحقول كلها.
شغّله بـ`dad dev examples/10-signup.dad`. وانظر [النماذج](ui.md#forms).

```dad
--8<-- "examples/10-signup.dad"
```
