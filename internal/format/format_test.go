package format_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/format"
)

func TestFormat(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"four spaces per level, from the brackets",
			"دالة مربع(س: صحيح) {\nأرجع س*س\n      }\n",
			"دالة مربع(س: صحيح) {\n    أرجع س * س\n}\n"},
		{"nested blocks, and } وإلا { on one line",
			"إذا أ {\nإذا ب {\nاطبع(1)\n}\n}\nوإلا {\n  اطبع(2)\n}\n",
			"إذا أ {\n    إذا ب {\n        اطبع(1)\n    }\n} وإلا {\n    اطبع(2)\n}\n"},
		{"a block that spans lines opens and closes on its own lines",
			"إذا صح { اطبع(1)\nاطبع(2) }\n",
			"إذا صح {\n    اطبع(1)\n    اطبع(2)\n}\n"},
		{"one statement per line instead of ؛",
			"ثابت ن = حاول { اقسم(10، 0) } التقط خ { اطبع(خ)؛ 0 }\nاطبع(1)؛ اطبع(2)\n",
			"ثابت ن = حاول { اقسم(10، 0) } التقط خ {\n    اطبع(خ)\n    0\n}\nاطبع(1)\nاطبع(2)\n"},
		{"match cases go one per line",
			"اطبع(طابق ج { شمال => \"↑\" ; جنوب => \"↓\" })\n",
			"اطبع(طابق ج {\n    شمال => \"↑\"\n    جنوب => \"↓\"\n})\n"},
		{"pages that span lines get blank lines around them, like other declarations",
			"صفحة \"/\" = الرئيسية()\nصفحة \"/عن\" { حالة ع = 0؛ عنوان(\"عن\") }\nصفحة \"/ب\" = الرئيسية()\n",
			"صفحة \"/\" = الرئيسية()\n\nصفحة \"/عن\" {\n    حالة ع = 0\n    عنوان(\"عن\")\n}\n\nصفحة \"/ب\" = الرئيسية()\n"},
		{"lambda parameters stay on the first line",
			"ثابت ج = [1].اختزل(0) { أ، ب =>\nأ + ب }\n",
			"ثابت ج = [1].اختزل(0) { أ، ب =>\n    أ + ب\n}\n"},
		{"a short block stays on one line",
			"ثابت ك2 = [1، 2].حول {هو*2}\nإذا صح { اطبع(1) } وإلا { اطبع(2) }\nثابت ف = {\n}\n",
			"ثابت ك2 = [1، 2].حول { هو * 2 }\nإذا صح { اطبع(1) } وإلا { اطبع(2) }\nثابت ف = {}\n"},
		{"spaces around operators and after commas",
			"ثابت س=1+2*3\nاطبع(س ,س،  س)\nمتغير ع=0\nع+=1\n",
			"ثابت س = 1 + 2 * 3\nاطبع(س, س، س)\nمتغير ع = 0\nع += 1\n"},
		{"no space inside parentheses",
			"اطبع( [ 1،2 ] )\n",
			"اطبع([1، 2])\n"},
		{"unary minus and not",
			"ثابت س = - 1\nثابت ص = (- س) * 2\nاطبع(! صح، ليس خطأ)\n",
			"ثابت س = -1\nثابت ص = (-س) * 2\nاطبع(!صح، ليس خطأ)\n"},
		{"members, optionals, ranges and named arguments",
			"ثابت ب = م ؟. بريد ؟؟ \"لا\"\nدالة ف(أ : نص ؟) = أ\nلكل ع في 1 .. 3 { اطبع(ع) }\nاطبع(نقطة(س : 1، ص:2) . بعد())\n",
			"ثابت ب = م؟.بريد ؟؟ \"لا\"\nدالة ف(أ: نص؟) = أ\nلكل ع في 1..3 { اطبع(ع) }\nاطبع(نقطة(س: 1، ص: 2).بعد())\n"},
		{"a blank line around declarations that span lines",
			"ثابت أ = 1\nدالة ف() {\nأرجع 1\n}\nدالة ق() = 2\nدالة ل() = 3\nنوع ن(س: صحيح) {\nدالة م() = س\n}\nاطبع(ف())\n",
			"ثابت أ = 1\n\nدالة ف() {\n    أرجع 1\n}\n\nدالة ق() = 2\nدالة ل() = 3\n\nنوع ن(س: صحيح) {\n    دالة م() = س\n}\n\nاطبع(ف())\n"},
		{"comments stay with their code",
			"// أول\nثابت س = 1   // واحد\nثابت صص = 22 // اثنان\n  /* كتلة\n   تبقى كما هي */\nاطبع(س)\n// قبل الدالة\nدالة ف() {\nاطبع(1)\n// لاحقاً\n}\n",
			"// أول\nثابت س = 1 // واحد\nثابت صص = 22 // اثنان\n/* كتلة\n   تبقى كما هي */\nاطبع(س)\n\n// قبل الدالة\nدالة ف() {\n    اطبع(1)\n    // لاحقاً\n}\n"},
		{"a comment after { stays on that line",
			"إذا صح { // دائماً\nاطبع(1) }\n",
			"إذا صح { // دائماً\n    اطبع(1)\n}\n"},
		{"multi-line strings are untouched",
			"ثابت ن = \"\"\"\n  سطر   أول\n سطر\"\"\"\nاطبع( ن )\n",
			"ثابت ن = \"\"\"\n  سطر   أول\n سطر\"\"\"\nاطبع(ن)\n"},
		{"blank lines collapse, trailing spaces go, one final newline",
			"\n\nثابت س = 1   \n\n\n\nاطبع(س)\n\n\n",
			"ثابت س = 1\n\nاطبع(س)\n"},
		{"several brackets opened on one line indent once",
			"اطبع([\n1،\n2\n])\n",
			"اطبع([\n    1،\n    2\n])\n"},
		{"long lines are wrapped, one argument per line",
			"اطبع(\"الاسم الكامل للمستخدم\"، \"عنوان البريد الإلكتروني\"، \"رقم الهاتف المحمول\"، \"المدينة والحي السكني\"، \"البلد\")\n",
			"اطبع(\n    \"الاسم الكامل للمستخدم\"،\n    \"عنوان البريد الإلكتروني\"،\n    \"رقم الهاتف المحمول\"،\n    \"المدينة والحي السكني\"،\n    \"البلد\"\n)\n"},
		{"windows line endings and tabs",
			"إذا صح {\r\n\tاطبع(1)\r\n}\r\n",
			"إذا صح {\n    اطبع(1)\n}\n"},
		{"angle brackets keep the writer's spacing",
			"ثابت ن: قائمة<صحيح> = []\nاطبع(1<2، 3 > 2)\n",
			"ثابت ن: قائمة<صحيح> = []\nاطبع(1<2، 3 > 2)\n"},
		{"tables, routes and server functions",
			"جدول مهمة{\nعنوان:نص\nمنجزة=خطأ\n}\nخادم دالة أضف(عنوان:نص)=مهمة.أنشئ(عنوان:عنوان)\nمسار جلب \"/api/مهام/{رقم}\"=مهمة.بالرقم(رقم)\n",
			"جدول مهمة {\n    عنوان: نص\n    منجزة = خطأ\n}\n\nخادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)\nمسار جلب \"/api/مهام/{رقم}\" = مهمة.بالرقم(رقم)\n"},
	}
	for _, c := range cases {
		got, err := format.Source(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s\n--- got\n%s--- want\n%s", c.name, got, c.want)
		}
		if again, _ := format.Source(got); again != got {
			t.Errorf("%s: formatting twice changes the result\n%s", c.name, again)
		}
	}
}

func TestRefusesFilesWithErrors(t *testing.T) {
	src := "دالة (\n"
	got, err := format.Source(src)
	if !errors.Is(err, format.ErrSyntax) || got != src || !strings.Contains(err.Error(), "أخطاء") {
		t.Errorf("got %q, %v", got, err)
	}
}

// Every example and test program is already formatted.
func TestExamplesAreFormatted(t *testing.T) {
	files, _ := filepath.Glob("../../examples/*.dad")
	files = append(files, "../interp/testdata/features.dad")
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		in := strings.ReplaceAll(string(src), "\r\n", "\n")
		got, err := format.Source(in)
		if err != nil {
			t.Errorf("%s: %v", f, err)
		} else if got != in {
			t.Errorf("%s is not formatted; run: dad fmt %s", f, f)
		}
	}
}
