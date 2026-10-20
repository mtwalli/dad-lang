# النشر على الإنترنت

يشغّل `dad serve` البرنامج خادم ويب حقيقياً. وكل من يزوره يرى الجداول نفسها، المحفوظة في
ملف SQLite واحد على الخادم.

```sh
dad serve todo.dad
dad serve todo.dad --port 3000 --data /var/lib/todo/todo.db
```

وبالعربية: `ضاد انشر todo.dad`. وهو خادم [`dad dev`](server.md#dad-dev) نفسه، دون ما لا
معنى له إلا على جهازك:

| | `dad dev` | `dad serve` |
|---|---|---|
| يستقبل الاتصال من | جهازك فقط (`127.0.0.1`) | كل واجهات الشبكة (`0.0.0.0`)؛ و`--host` لتغييرها |
| المنفذ | `--port`، و8080 افتراضياً | `--port`، أو `$PORT` كما تضبطه منصات الاستضافة، و8080 افتراضياً |
| صفحة المطوّر `/_dad` | نعم: الدوال والمسارات وسجلات كل جدول | **لا**: يجيب `/_dad/…` بـ404، إلا ما تحتاجه الصفحات |
| إعادة التحميل عند تغيّر الملفات | نعم | لا: أعد تشغيله لتحديث البرنامج |
| برنامج فيه أخطاء | تبقى آخر نسخة سليمة تعمل | لا يبدأ (رمز الخروج 1) |
| السجلات | ناتج البرنامج | ناتج البرنامج، وسطر لكل طلب في stderr |
| الإيقاف | Ctrl+C | Ctrl+C أو `SIGTERM`: ينهي الطلبات الجارية ويغلق قاعدة البيانات |

ويقدّم الموقع:

- [واجهتك](ui.md) عند `/`، إن كانت للبرنامج واجهة؛
- [مساراتك](server.md#routes) (`مسار`)؛
- دوال الخادم عند `POST /_dad/call/<الاسم>`؛
- و`/_dad/health`، الذي يجيب `{"ok": true}` لفحوص التشغيل.

## أين تُحفظ البيانات { #where-the-data-is-kept }

تُحفظ البيانات افتراضياً في `.dad/<الاسم>.db` بجانب البرنامج، كما في `dad dev`. أما على
الخادم فاحفظها في مكان يبقى بعد كل نشر جديد، ووجّه `--data` إليه:

```sh
dad serve app.dad --data /data/app.db
```

وتغيير `جدول` يغيّر قاعدة البيانات في المرة التالية التي يبدأ فيها البرنامج، كما في
`dad dev`: انظر [تغيير جدول](server.md#changing-a-table). فبيانات الحقل المحذوف تبقى مخفية،
وتغيير النوع الذي لا تقبله بعض السجلات يمنع البرنامج من البدء، وتسرد رسالة الخطأ تلك
السجلات.

وللنسخ الاحتياطي والخادم يعمل، استعمل النسخ الاحتياطي الخاص بـSQLite، لا النسخ العادي:

```sh
sqlite3 /data/app.db ".backup /backups/app-$(date +%F).db"
```

## HTTPS { #https }

يتكلم `dad serve` بـHTTP العادي. وللموقع العام ضعه خلف خادم يتولى الشهادات. ومع
[Caddy](https://caddyserver.com) يكفي ملف `Caddyfile` هذا:

```text
example.com {
    reverse_proxy localhost:8080
}
```

ثم شغّل `dad serve app.dad --host 127.0.0.1`، فلا يصل إليه إلا Caddy.

## خادم Linux مع systemd { #a-linux-server-with-systemd }

```ini title="/etc/systemd/system/todo.service"
[Unit]
Description=To-do app in Ḍād
After=network.target

[Service]
WorkingDirectory=/srv/todo
ExecStart=/usr/local/bin/dad serve todo.dad --host 127.0.0.1 --data /var/lib/todo/todo.db
Restart=on-failure
DynamicUser=yes
StateDirectory=todo

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemctl enable --now todo
journalctl -u todo -f        # the request log
```

## Docker، ومنصات مثل Fly.io وRender { #docker-and-platforms-such-as-flyio-or-render }

`dad` برنامج واحد بلا اعتماديات، وSQLite مدمجة فيه، فتكفيه صورة صغيرة:

```dockerfile title="Dockerfile"
FROM golang:1.26 AS dad
RUN CGO_ENABLED=0 go install github.com/mtwalli/dad-lang/cmd/dad@latest

FROM gcr.io/distroless/static
COPY --from=dad /go/bin/dad /dad
COPY . /app
WORKDIR /app
VOLUME /data
CMD ["/dad", "serve", "todo.dad", "--data", "/data/todo.db"]
```

تضبط المنصة `$PORT`، ويستعمله `dad serve`. وأعطِ التطبيق مساحة تخزين دائمة مثبّتة عند
`/data`، لتبقى قاعدة البيانات بعد كل نشر. وشغّل نسخة واحدة من التطبيق: انظر القسم التالي.

## ما يجب أن تعرفه قبل أن تعتمد عليه { #what-to-know-before-you-rely-on-it }

- **نسخة واحدة.** البيانات في ملف SQLite واحد، والصفحات تحفظ حالتها في ذاكرة الخادم. فشغّل
  `dad serve` واحداً، لا عدة نسخ خلف موزّع أحمال.
- **طلب واحد في كل مرة.** تُعالَج الطلبات بالدور. وهذا كافٍ لفريق صغير أو فصل دراسي، لا
  لزيارات كثيفة.
- **إعادة التشغيل تعيد تحميل الصفحات المفتوحة.** يضيع ما كتبه الزائر ولم يحفظه، أما الجداول
  فلا تضيع.
- **لا تسجيل دخول بعد.** كل زائر يستطيع أن يستدعي كل دوال الخادم والمسارات. فلا تنشر
  برنامجاً فيه بيانات لا يجوز إلا لبعض الناس أن يروها أو يغيّروها. والمستخدمون والصلاحيات
  في [خارطة الطريق](ROADMAP.md).
