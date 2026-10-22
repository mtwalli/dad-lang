package check_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/parser"
)

func problems(t *testing.T, src string) []string {
	t.Helper()
	file, diags := parser.ParseFile(src)
	if file == nil {
		t.Fatalf("parse failed: %v\n%s", diags, src)
	}
	var out []string
	for _, d := range check.File(file) {
		out = append(out, d.String())
	}
	return out
}

const shapes = `
نوع نقطة(س: عشري، ص: عشري) {
    دالة بعد() = جذر(س * س + ص * ص)
}
تعداد شكل {
    دائرة(قطر: عشري)
    مربع(ضلع: عشري)
    مثلث(قاعدة: عشري، ارتفاع: عشري)
}
دالة مساحة(ش: شكل): عشري = طابق ش {
    دائرة(ق) => 3.14 * ق * ق / 4
    مربع(ض) => ض * ض
    مثلث(ق، ر) => ق * ر / 2
}
`

func TestFindsMistakes(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"undefined name with a suggestion", "ثابت عدد_الطلاب = 3\nاطبع(عدد_الطلب)", "«عدد_الطلب» غير معرّف؛ هل تقصد «عدد_الطلاب»؟"},
		{"undefined function", "اطبعه(1)", "«اطبعه» غير معرّف؛ هل تقصد «اطبع»؟"},
		{"used before it is defined", "اطبع(س)\nثابت س = 1", "«س» غير معرّف"},
		{"changing a constant", "ثابت س = 1\nس = 2", "لا يمكن تغيير «س» لأنه ثابت"},
		{"changing a parameter", "دالة ف(س: صحيح) {\n    س = 2\n}", "لا يمكن تغيير «س»"},
		{"declared twice", "ثابت س = 1\nثابت س = 2", "معرّف مسبقاً"},
		{"wrong type in a declaration", "ثابت س: صحيح = \"عشرة\"", "«س» من النوع «صحيح»، لكن قيمته من النوع «نص»"},
		{"wrong type in an assignment", "متغير س = 1\nس = \"اثنان\"", "«س» من النوع «صحيح»، ولا يمكن أن يأخذ قيمة من النوع «نص»"},
		{"too many arguments", "دالة ف(س: صحيح) = س\nف(1، 2)", "«ف» تأخذ 1 من المعاملات لكن أُعطيت 2"},
		{"missing argument", "دالة ف(س: صحيح، ص: صحيح) = س\nف(1)", "ينقص المعامل «ص» عند استدعاء «ف»"},
		{"unknown named argument", "دالة ف(عنوان: نص) = عنوان\nف(عنون: \"أ\")", "لا تحتوي معاملاً باسم «عنون»؛ هل تقصد «عنوان»؟"},
		{"argument of the wrong type", "دالة ف(س: صحيح) = س\nف(\"ثلاثة\")", "المعامل «س» في «ف»: توقعت قيمة من النوع «صحيح» لكن وجدت «نص»"},
		{"constructor field of the wrong type", shapes + "ثابت ن = نقطة(س: \"أ\"، ص: 1)", "توقعت قيمة من النوع «عشري» لكن وجدت «نص»"},
		{"named before positional", "دالة ف(أ: صحيح، ب: صحيح) = أ\nف(ب: 1، 2)", "المعاملات المسماة يجب أن تأتي بعد"},
		{"field that doesn't exist", shapes + "ثابت ن = نقطة(1، 2)\nاطبع(ن.ع)", "القيمة من النوع «نقطة» لا تحتوي «ع»"},
		{"method that doesn't exist", shapes + "ثابت ن = نقطة(1، 2)\nاطبع(ن.بعدد())", "لا تحتوي «بعدد»؛ هل تقصد «بعد»؟"},
		{"text method typo", "اطبع(\"مرحبا\".كبيرر())", "القيمة من النوع «نص» لا تحتوي «كبيرر»؛ هل تقصد «كبير»؟"},
		{"list method typo", "اطبع([1، 2].رشحح { هو > 1 })", "هل تقصد «رشح»؟"},
		{"copy with an unknown field", shapes + "ثابت ن = نقطة(1، 2)\nاطبع(ن.نسخة(ع: 3))", "لا يوجد حقل باسم «ع»"},
		{"changing a field", shapes + "ثابت ن = نقطة(1، 2)\nن.س = 3", "حقول الأنواع لا تتغير"},
		{"match missing cases", shapes + "دالة اسم(ش: شكل): نص = طابق ش {\n    دائرة(ق) => \"دائرة\"\n}", "«طابق» لا تغطي كل حالات «شكل»: تنقص «مربع»، «مثلث»"},
		{"pattern with too many names", shapes + "دالة ق(ش: شكل): عشري = طابق ش {\n    دائرة(أ، ب) => أ\n    _ => 0\n}", "الحالة «دائرة» تحتوي 1 حقول لكن النمط يذكر 2"},
		{"unknown case in a pattern", shapes + "دالة ق(ش: شكل): عشري = طابق ش {\n    دايرة(أ) => أ\n    _ => 0\n}", "«دايرة» ليست حالة معروفة في أي تعداد؛ هل تقصد «دائرة»؟"},
		{"unknown type", "دالة ف(س: صحيج) = س", "النوع «صحيج» غير معروف؛ هل تقصد «صحيح»؟"},
		{"wrong result type", "دالة ف(): صحيح = \"أ\"", "الدالة «ف» ترجع «صحيح»، لكنها تعطي «نص»"},
		{"wrong returned value", "دالة ف(): صحيح {\n    أرجع \"أ\"\n}", "الدالة «ف» ترجع «صحيح»، لكن هنا «نص»"},
		{"missing return", "دالة ف(): صحيح {\n    اطبع(1)\n}", "هل نسيت «أرجع»؟"},
		{"return outside a function", "أرجع 1", "«أرجع» مستخدمة خارج دالة"},
		{"break outside a loop", "توقف", "«توقف» مستخدمة خارج حلقة"},
		{"condition that isn't true or false", "إذا 1 { اطبع(1) }", "شرط «إذا» يجب أن يكون «صح» أو «خطأ» لكنه من النوع «صحيح»"},
		{"subtracting text", "اطبع(\"أ\" - 1)", "لا يمكن تطبيق «-» على «نص» و«صحيح»"},
		{"comparing text with a number", "اطبع(\"أ\" < 1)", "لا يمكن تطبيق «<» على «نص» و«صحيح»"},
		{"looping over a number", "لكل س في 5 { اطبع(س) }", "لا يمكن المرور على قيمة من النوع «صحيح»"},
		{"it outside a lambda", "اطبع(هو)", "«هو» متاحة فقط داخل دالة مجهولة"},
		{"this outside a method", "اطبع(هذا)", "«هذا» متاحة فقط داخل دوال الأنواع"},
		{"calling a value", "ثابت س = 1\nس()", "«س» ليس دالة، بل قيمة من النوع «صحيح»"},
		{"element outside a component", "زر(\"أ\")", "«زر» عنصر واجهة، ويُستخدم داخل مكون"},
		{"state outside a component", "حالة س = 0", "«حالة» تُستخدم داخل مكون فقط"},
		{"element option typo", "مكون م {\n    حقل(\"أ\"، تلميج: \"ب\")\n}", "«حقل» لا يأخذ «تلميج»؛ يأخذ: تسمية، تلميح، صيغة، معطل، أدنى، أقصى، مطلوب، أدنى_طول، أقصى_طول، تحقق، صنف؛ هل تقصد «تلميح»؟"},
		{"table method typo", "جدول مهمة {\n    عنوان: نص\n}\nاطبع(مهمة.الكلل())", "هل تقصد «الكل»؟"},
		{"index with text", "ثابت ق = [1، 2]\nاطبع(ق[\"أ\"])", "الفهرس يجب أن يكون عدداً صحيحاً"},
		{"range of decimals", "لكل س في من 1.5 إلى 3 { اطبع(س) }", "حدود النطاق تكون أعداداً صحيحة"},
		{"two pages at one address", "صفحة \"/أ\" {\n    عنوان(\"1\")\n}\nصفحة \"/أ/\" {\n    عنوان(\"2\")\n}", "الصفحة «/أ/» بالعنوان نفسه لـالصفحة في السطر 1"},
		{"a page at a route's address", "مسار جلب \"/مهام/{رقم}\" = رقم\nصفحة \"/مهام/{ر}\" {\n    فقرة(\"{ر}\")\n}", "الصفحة «/مهام/{ر}» بالعنوان نفسه لـالمسار في السطر 1"},
		{"a name inside a part of the address", "صفحة \"/مهمة-{رقم}\" {\n    فقرة(\"أ\")\n}", "جزء غير صحيح: «مهمة-{رقم}»"},
		{"a name twice in an address", "صفحة \"/{أ}/{أ}\" {\n    فقرة(\"أ\")\n}", "المتغير «أ» مكرر في العنوان"},
		{"a page inside a function", "دالة ف() {\n    صفحة \"/أ\" {\n        عنوان(\"أ\")\n    }\n}", "«صفحة» تُعرَّف في أعلى الملف"},
		{"a page using a name its address doesn't have", "صفحة \"/مهمة/{رقم}\" {\n    فقرة(\"{رقمم}\")\n}", "«رقمم» غير معرّف؛ هل تقصد «رقم»؟"},
		{"element option typo in a page", "صفحة \"/\" {\n    زر(\"أ\"، معطّلل: صح)\n}", "«زر» لا يأخذ «معطلل»"},
		{"a style that doesn't exist", "مكون م {\n    زر(\"أ\"، نمط: \"ثانويي\")\n}", "«نمط» في «زر» يكون «أساسي» أو «ثانوي» أو «خفيف» أو «خطر»؛ هل تقصد «ثانوي»؟"},
		{"an alert's style", "مكون م {\n    تنبيه(\"أ\"، نمط: \"ثانوي\")\n}", "«نمط» في «تنبيه» يكون «معلومة» أو «نجاح» أو «تحذير» أو «خطأ»"},
		{"a colour that doesn't exist", "مكون م {\n    فقرة(\"أ\"، لون: \"ازرق\")\n}", "يأخذ اسم لون مثل «أزرق» أو «فيروزي»، أو قيمة مثل «#1b2b6b»؛ هل تقصد «أزرق»؟"},
		{"an icon that doesn't exist", "مكون م {\n    أيقونة(\"بحت\")\n}", "لا توجد أيقونة باسم «بحت» في «أيقونة»؛ هل تقصد «بحث»؟"},
		{"an icon option that doesn't exist", "مكون م {\n    زر(\"أ\"، أيقونة: \"حذفف\")\n}", "لا توجد أيقونة باسم «حذفف» في «زر»؛ هل تقصد «حذف»؟"},
		{"a theme option typo", "سمة(اللوان: \"أزرق\")", "«سمة» لا تأخذ «اللوان»؛ تأخذ: اللون، الخط، الزوايا، العرض، الحجم، الوضع؛ هل تقصد «اللون»؟"},
		{"a theme font that doesn't exist", "سمة(الخط: \"رقعة\")", "«الخط» في «سمة» يكون «كوفي» أو «نسخ» أو «النظام»"},
		{"the theme's colour must be a colour", "سمة(اللون: \"أساسي\")", "«اللون» في «سمة» يأخذ لوناً بعينه"},
		{"field type from a table default", "جدول مهمة {\n    منجزة = خطأ\n}\nاطبع(مهمة.أنشئ(منجزة: 1))", "توقعت قيمة من النوع «منطقي» لكن وجدت «صحيح»"},
	} {
		got := problems(t, c.src)
		ok := false
		for _, g := range got {
			ok = ok || strings.Contains(g, c.want)
		}
		if !ok {
			t.Errorf("%s: want %q, got %q", c.name, c.want, got)
		}
	}
}

// Programs that are correct must have no problems, including ones that use
// features the checker can't fully follow.
func TestAcceptsCorrectPrograms(t *testing.T) {
	for _, src := range []string{
		shapes + "اطبع(مساحة(دائرة(2))، نقطة(3، 4).بعد())",
		shapes + "لكل ش في [دائرة(2)، مربع(3)] {\n    إذا ش من نوع دائرة { اطبع(ش.قطر) }\n}",
		shapes + "دالة اسم(ش: شكل): نص = طابق ش {\n    دائرة(_) => \"دائرة\"\n    غيرها => \"أخرى\"\n}",
		// a function may use a constant declared after it
		"دالة ف() = حد * 2\nثابت حد = 3\nاطبع(ف())",
		// methods see their fields and each other without هذا
		"نوع عداد(ع: صحيح) {\n    دالة التالي() = نسخة(ع: ع + 1)\n    دالة مرتين() = التالي().التالي()\n}\nاطبع(عداد(0).مرتين().ع)",
		// extensions can call built-in methods of their type unqualified
		"امتداد نص {\n    دالة نظيف() = قص()\n    دالة صاخب() = هذا.كبير() + \"!\"\n}\nاطبع(\" أ \".نظيف().صاخب())",
		// optional values, safe calls and ??
		"نوع مستخدم(اسم: نص، بريد: نص؟ = عدم)\nثابت م: مستخدم؟ = عدم\nاطبع(م؟.بريد ؟؟ \"لا\"، م؟.اسم)",
		// lambdas, it, and a top-level lambda using a later name
		"ثابت ف = { هو + ب }\nثابت ب = 2\nاطبع([1، 2].حول { هو * 2 }، ف(1))",
		// try, catch, throw
		"ثابت ن = حاول { ارفع مشكلة(\"أ\") } التقط خ { خ.رسالة }\nاطبع(ن)",
		// numbers mix; a decimal variable takes whole numbers
		"متغير م: عشري = 1\nم += 2\nم = 3\nاطبع(م / 2، 7 % 3)",
		// tables, routes and server functions
		"جدول مهمة {\n    عنوان: نص\n    منجزة = خطأ\n}\nخادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)\nمسار جلب \"/مهام/{رقم}\" = مهمة.بالرقم(رقم)\nمسار أرسل \"/مهام\" = أضف(البيانات[\"عنوان\"])\nاطبع(مهمة.حيث { هو.منجزة }.عدد، مهمة.الكل().حول { م => م.عنوان })",
		// components, state and a helper function that lays out elements
		"دالة رأس(نص_العنوان: نص) {\n    عنوان(نص_العنوان)\n}\nمكون الرئيسية {\n    حالة ع = 0\n    عمود {\n        رأس(\"عدّاد\")\n        زر(\"زد\"، معطل: ع > 9) { ع += 1 }\n        حقل(ع)\n    }\n}",
		// pages: a component at an address, a block with state, a value from the
		// address, links and «انتقل»; «صفحة» is still an ordinary name elsewhere
		"مكون تفاصيل(رقم: صحيح) {\n    عنوان(\"المهمة {رقم}\")\n}\nصفحة \"/\" {\n    حالة ع = 0\n    رابط(\"المهمة\"، \"/مهمة/1\")\n    زر(\"زد {ع}\") { ع += 1 }\n    زر(\"افتح\") { انتقل(\"/مهمة/{ع}\") }\n}\nصفحة \"/مهمة/{رقم}\" = تفاصيل(رقم)\nمسار جلب \"/api/مهمة/{رقم}\" = رقم\nثابت صفحة = 3\nاطبع(صفحة)",
		// match on text and numbers with literal patterns
		"دالة وصف(س: صحيح): نص = طابق س {\n    1 => \"واحد\"\n    _ => \"كثير\"\n}\nاطبع(وصف(1))",
		// maps and indexing
		"متغير ق = [\"أ\": 1]\nق[\"ب\"] = 2\nاطبع(ق[\"أ\"]، ق.مفاتيح، ق.يحتوي(\"أ\"))",
		// loops over text, ranges and maps
		"لكل ح في \"نص\" { اطبع(ح) }\nلكل ع من 1 إلى 3 { اطبع(ع) }\nلكل مفتاح في [\"أ\": 1] { اطبع(مفتاح) }",
		// a function that always throws needs no return
		"دالة فشل(): صحيح {\n    ارفع مشكلة(\"لا\")\n}",
		// casts
		"اطبع(\"12\" ك صحيح + 1، (7 / 2) ك صحيح)",
	} {
		if got := problems(t, src); len(got) > 0 {
			t.Errorf("unexpected problems %q in\n%s", got, src)
		}
	}
}

func TestImports(t *testing.T) {
	files := map[string]string{
		"main.dad": `استورد مساحة، سر، غير_موجود، شكل من "./أشكال"
دالة ف(ن: نقطة) = ن
دالة ج(ش: شكل) = طابق ش {
    دائرة(نق) => نق
    مربع => 0
}
اطبع(مساحة(2)، دائرة(1))
مساحة = 3
`,
		"أشكال.dad": `دالة مساحة(ض: صحيح): صحيح = ض * ض
خاص ثابت سر = 1
نوع نقطة(س: صحيح)
تعداد شكل { دائرة(نق: صحيح)، مربع }
`,
		"مكرر.dad": `استورد مساحة من "./أشكال"
نوع نقطة(ص: صحيح)
`,
	}
	read := func(p string) (string, error) {
		if s, ok := files[p]; ok {
			return s, nil
		}
		return "", os.ErrNotExist
	}
	_, diags := check.Load("main.dad", read)
	var got []string
	for _, d := range diags {
		got = append(got, fmt.Sprintf("%s:%d: %s", d.Path, d.Pos.Line, d.Msg))
	}
	want := []string{
		"main.dad:1: «سر» خاص بالملف «أشكال»، فلا يمكن استيراده",
		"main.dad:1: «غير_موجود» غير موجود في «أشكال»",
		"main.dad:2: النوع «نقطة» معرّف في «أشكال»؛ استورده أولاً: استورد نقطة من \"./أشكال\"",
		"main.dad:8: لا يمكن تغيير «مساحة»",
	}
	if len(got) != len(want) {
		t.Fatalf("got:\n%s", strings.Join(got, "\n"))
	}
	for i := range want {
		if !strings.HasPrefix(got[i], want[i]) {
			t.Errorf("got  %s\nwant %s", got[i], want[i])
		}
	}

	_, diags = check.Load("مكرر.dad", read)
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, "النوع «نقطة» معرّف أيضاً في «أشكال»") {
		t.Errorf("duplicate type: %v", diags)
	}
}
