# Server and data

A Ḍād file can hold its data, its server code and its web routes next to each other.
`dad dev` serves them on your computer and reloads when you save. The [interface](ui.md)
(`مكون`) calls server functions and tables directly; other programs reach them through the
dev page, `curl`, or any web page.

## Tables — `جدول`

```dad
جدول مهمة {
  عنوان: نص
  منجزة = خطأ
  أولوية = 1
}
```

Each field has a type, a default, or both. Every table also gets a hidden field `رقم`
(its id), set when the record is first saved. You can't declare a field called `رقم` yourself.

| Code | What it does |
|---|---|
| `مهمة.أنشئ(عنوان: "…")` | Create a record, save it and return it (with its `رقم`) |
| `مهمة.الكل()` | Every record, oldest first |
| `مهمة.حيث { هو.منجزة }` | The records that match |
| `مهمة.بالرقم(3)` | One record, or `عدم` |
| `مهمة.عدد()` | How many records there are |
| `م.احفظ()` | Save a new or changed record and return it |
| `م.احذف()` | Delete a saved record |

Records are values, like any other type: change one with `نسخة` and save the copy.

```dad
ثابت م = مهمة.أنشئ(عنوان: "اشترِ خبزاً")
م.نسخة(منجزة: صح).احفظ()
اطبع(مهمة.حيث { هو.منجزة }.حول { هو.عنوان })   // [«اشترِ خبزاً»]
```

### Where the data goes

`dad run` and `dad dev` keep a program's tables in a SQLite database, `.dad/<name>.db` next to
the program, so `todo.dad` saves to `.dad/todo.db`. The data is still there the next time you
run the program. Delete the `.dad` folder to start fresh, and add `.dad/` to your `.gitignore`
if you don't want to commit it. Any SQLite tool can open the file; each `جدول` is a table with
a column per field. The playground keeps tables in memory only.

If you used an earlier version, its `.dad/<name>.json` file is copied into the database the
first time you run the program, keeping the ids, and then renamed to `.json.imported`.

### Changing a table

When you change a `جدول` and run the program again, the database follows:

| Change | What happens |
|---|---|
| Add a field | Saved records get its default. A field with no default can only be added while the table is empty, unless its type is optional (`نص؟`) |
| Remove a field | Its column is kept but hidden. Add the field back and its data comes back |
| Change a field's type | Every value is converted, as `ك` would. If some can't be (say `"أحمد"` to `صحيح`), nothing changes and the error lists those records |

Renaming a field looks like removing one and adding another, so the old values stay in the
hidden column.

### Fast queries

Simple `حيث` conditions run inside the database instead of reading every record. A condition
counts as simple when it compares fields with values or variables, joined by `و`, `أو` and `ليس`:

```dad
مهمة.حيث { هو.منجزة }
مهمة.حيث { ليس هو.منجزة و هو.أولوية >= حد }
```

Anything else, such as `{ هو.عنوان.يحتوي("خبز") }` or `{ هو.أولوية * 2 > 3 }`, still works the
same; it just reads every record. `بالرقم` and `عدد` always ask the database directly.

## Server functions — `خادم`

```dad
خادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)
خادم دالة بدل(م: مهمة) = م.نسخة(منجزة: ليس م.منجزة).احفظ()
```

A function marked `خادم` can be called from the browser. A [component](ui.md) calls it like
any other function. Other pages and programs reach it at `/_dad/call/<name>`: send a POST
with the arguments as a JSON object, by parameter name.

```sh
curl -X POST localhost:8080/_dad/call/أضف -d '{"عنوان": "اشترِ خبزاً"}'
# {"result": {"أولوية": 1, "رقم": 1, "عنوان": "اشترِ خبزاً", "منجزة": false}}
```

Arguments are checked against the parameter types, and a record type such as `مهمة` accepts a
JSON object. Errors come back with status 400 and an Arabic message under `"خطأ"`.

## Routes — `مسار`

Routes are public web addresses for other programs.

```dad
مسار جلب "/api/مهام" = مهمة.الكل()
مسار جلب "/api/مهام/{رقم}" = مهمة.بالرقم(رقم)
مسار أرسل "/api/مهام" {
  أرجع مهمة.أنشئ(عنوان: البيانات["عنوان"])
}
```

| Method | HTTP |
|---|---|
| `جلب` | GET |
| `أرسل` | POST |
| `عدل` | PUT |
| `احذف` | DELETE |

- A `{name}` part of the path becomes a variable in the body. A value made of digits
  becomes a `صحيح`, otherwise it is `نص`.
- The JSON body of the request is `البيانات`.
- The result is sent back as JSON. A `جلب` route that returns `عدم` answers 404.

## `dad dev`

```sh
dad dev todo.dad              # or: ضاد تطوير todo.dad
dad dev todo.dad --port 3000
```

Open <http://localhost:8080>. If the program has an [interface](ui.md), that's what you
see; the developer page is at <http://localhost:8080/_dad>. The developer page lists your server functions, with a form to call
each one, your tables and their rows, and your routes. It updates by itself.

- Save the file and the program reloads. The data is kept.
- If the file has an error, the last working version keeps running and the page shows
  the error until you fix it.
- Top-level code (such as `اطبع`) runs on every reload, and its output appears in the terminal.
- The server only listens on your own computer (`127.0.0.1`).
