---
icon: lucide/download
---

# تثبيت ضاد

!!! note "جديد على البرمجة؟"
    يبدأ [دليل المبتدئين](tutorial/index.md) من هنا، وينتهي بموقع على الإنترنت. ويمكنك
    أيضاً أن [تجرّب الكود في المتصفح](playground.md) دون أن تثبّت شيئاً.

## التثبيت

=== "Homebrew (macOS وLinux)"

    ```sh
    brew install mtwalli/dad/dad
    ```

    ولتحديثه فيما بعد: `brew upgrade dad`.

=== "winget (Windows)"

    ```powershell
    winget install mtwalli.dad
    ```

    ثم افتح طرفية جديدة. ولتحديثه فيما بعد: `winget upgrade mtwalli.dad`.

=== "Debian وUbuntu"

    نزّل `dad_<الإصدار>_amd64.deb` (أو `_arm64.deb`) من
    [صفحة الإصدارات](https://github.com/mtwalli/dad-lang/releases/latest)، ثم:

    ```sh
    sudo apt install ./dad_*_amd64.deb
    ```

=== "Fedora وRHEL"

    نزّل `dad-<الإصدار>-1.x86_64.rpm` (أو `.aarch64.rpm`) من
    [صفحة الإصدارات](https://github.com/mtwalli/dad-lang/releases/latest)، ثم:

    ```sh
    sudo dnf install ./dad-*.x86_64.rpm
    ```

=== "سكربت التثبيت (macOS وLinux)"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.sh | sh
    ```

    ينزّل السكربت أحدث إصدار لجهازك، ويتحقق من بصمته، ويضع `dad` في `~/.dad/bin` ومعه
    الاسم العربي `ضاد`. وإن لم يكن ذلك المجلد في `PATH` بعد، طبع لك السطر الذي تضيفه.

=== "سكربت التثبيت (Windows)"

    في PowerShell:

    ```powershell
    irm https://raw.githubusercontent.com/mtwalli/dad-lang/main/install.ps1 | iex
    ```

    يثبّت السكربت `dad.exe` في `%LOCALAPPDATA%\dad\bin` ويضيفه إلى `PATH`. افتح طرفية
    جديدة بعده.

=== "بلغة Go"

    إن كان عندك Go 1.26 أو أحدث:

    ```sh
    go install github.com/mtwalli/dad-lang/cmd/dad@latest
    ```

=== "تنزيل يدوي"

    نزّل الأرشيف المناسب لجهازك من
    [صفحة الإصدارات](https://github.com/mtwalli/dad-lang/releases/latest)، وفكّه، وضع `dad`
    (أو `dad.exe`) في أي مجلد ضمن `PATH`. وفي `checksums.txt` بصمة SHA-256 لكل أرشيف.

ولتثبيت إصدار بعينه، اضبط `DAD_VERSION` قبل تشغيل السكربت، مثل `DAD_VERSION=v0.1.0`.

## تأكد من التثبيت

```sh
dad version
```

سيظهر رقم الإصدار، مثل `ضاد 0.2.0`. وعلى macOS وLinux يعطيك Homebrew والحزم وسكربت
التثبيت اسم الأمر بالعربية أيضاً: `ضاد`.

## أول ملف

احفظ هذا باسم `hello.dad`:

```dad
// أول برنامج بلغة ضاد
ثابت اسم = "العالم"
اطبع("مرحبا يا {اسم}!")
```

وشغّله من المجلد الذي حفظته فيه:

```sh
dad run hello.dad
```

``` { .text .dad-out }
مرحبا يا العالم!
```

وبالعربية: `ضاد شغل hello.dad`. ويعمل `dad hello.dad` أيضاً. وهكذا يعمل أي ملف `.dad`،
ومنها [الأمثلة](examples.md) في المستودع:

```sh
git clone https://github.com/mtwalli/dad-lang
cd dad-lang
dad run examples/03-shapes.dad
```

### حين يخطئ شيء

الأخطاء التي تظهر أثناء التشغيل توقف البرنامج، وتدلّك على السطر بالعربية:

```dad
ثابت أ = 10
اطبع(أ / 0)
```

``` { .text .dad-out }
hello.dad: خطأ أثناء التشغيل [السطر 2، العمود 8]: مشكلة لم تُعالَج: قسمة على صفر
```

ولتجد الأخطاء دون أن تشغّل شيئاً، استعمل `check` (بالعربية: `ضاد فحص`):

```sh
dad check hello.dad
```

``` { .text .dad-out }
✓ hello.dad: لا أخطاء (11 رمزاً)
```

أضف الآن سطراً فيه خطأ. يكتب ضاد الأعداد بالأرقام 0-9 فقط، فالعدد المكتوب بالأرقام
العربية المشرقية خطأ:

```dad
ثابت عمر = ١٢
```

افحص مرة أخرى. يخبرك ضاد بالمشكلة، وبسطرها وعمودها:

``` { .text .dad-out }
hello.dad: خطأ [السطر 4، العمود 12]: اكتب الأعداد بالأرقام 0-9: «١٢» تُكتب «12»
```

ولترى كيف يقرأ ضاد الكود، اطبع رموزه (بالعربية: `ضاد رموز`):

```sh
dad tokens hello.dad
```

وللتلوين، والأخطاء وأنت تكتب، وزر للتشغيل، ثبّت [إضافة ضاد لـVS Code](https://marketplace.visualstudio.com/items?itemName=mtwalli.dad-lang)
(أو ابحث عن **Ḍād** في قائمة الإضافات). وفي [دليل المحرر](editor.md) التفاصيل.

## الأوامر

| الأمر | بالعربية | ما يفعله |
|---|---|---|
| `dad run <ملف>` | `ضاد شغل <ملف>` | يشغّل البرنامج (ويعمل `dad <ملف>` أيضاً) |
| `dad fmt <ملف أو مجلد>` | `ضاد نسق <ملف>` | ينسّق الملفات ويحفظها (`--check` يعرض ما يحتاج تنسيقاً). انظر [التنسيق](editor.md#formatting) |
| `dad lsp` | `ضاد خادم_اللغة` | خادم اللغة للمحررات: التعريفات والمراجع وإعادة التسمية والإكمال. انظر [دليل المحرر](editor.md#the-language-server) |
| `dad dev <ملف>` | `ضاد تطوير <ملف>` | يشغّل البرنامج خادماً على `localhost:8080`، ويعيد التحميل عند الحفظ (`--port` لمنفذ آخر). انظر [الخادم والبيانات](server.md) |
| `dad serve <ملف>` | `ضاد انشر <ملف>` | يشغّل البرنامج خادماً للنشر: بلا صفحة مطوّر ولا إعادة تحميل، على كل الواجهات، ومع `$PORT` (`--port` و`--host` و`--data`). انظر [النشر على الإنترنت](deploy.md) |
| `dad build <ملف>` | `ضاد ابن <ملف>` | يحوّل البرنامج إلى JavaScript: موقعاً ثابتاً إن كانت له واجهة، وإلا `app.js` لـNode (`--out` لاختيار المجلد). انظر [البناء للويب](build.md) |
| `dad check <ملف>` | `ضاد فحص <ملف>` | يفحص الملف دون تشغيله: الصياغة والأسماء والمعاملات والأنواع وحالات `طابق` (`--json` للأدوات) |
| `dad preview <ملف>` | `ضاد معاينة <ملف>` | صفحة واجهة البرنامج، بصيغة JSON، لمعاينتها في المحرر. انظر [معاينة الواجهة](editor.md#preview-the-interface) |
| `dad tokens <ملف>` | `ضاد رموز <ملف>` | يطبع كل رمز مع سطره وعموده |
| `dad version` | `ضاد إصدار` | يطبع الإصدار المثبّت |
| `dad help` | `ضاد مساعدة` | يعرض الأوامر |

## إزالة ضاد

=== "Homebrew"

    ```sh
    brew uninstall dad
    ```

=== "winget"

    ```powershell
    winget uninstall mtwalli.dad
    ```

=== "Debian وUbuntu وFedora"

    ```sh
    sudo apt remove dad      # or: sudo dnf remove dad
    ```

=== "سكربت التثبيت (macOS وLinux)"

    ```sh
    rm -rf ~/.dad
    ```

    ثم احذف سطر `PATH` من ملف إعدادات الطرفية إن كنت أضفته.

=== "سكربت التثبيت (Windows)"

    احذف المجلد `%LOCALAPPDATA%\dad`، ثم احذفه من `PATH` الخاص بك من
    *الإعدادات ← النظام ← حول ← إعدادات النظام المتقدمة ← متغيرات البيئة*.

=== "بلغة Go"

    ```sh
    rm "$(go env GOPATH)/bin/dad"
    ```
