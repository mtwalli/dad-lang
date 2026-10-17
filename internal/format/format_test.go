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
		{"indentation comes from the brackets",
			"دالة مربع(س: صحيح) {\nأرجع س*س\n      }\n",
			"دالة مربع(س: صحيح) {\n  أرجع س * س\n}\n"},
		{"nested blocks and else",
			"إذا أ {\nإذا ب {\nاطبع(1)\n}\n} وإلا {\n    اطبع(2)\n}\n",
			"إذا أ {\n  إذا ب {\n    اطبع(1)\n  }\n} وإلا {\n  اطبع(2)\n}\n"},
		{"spaces around operators and after commas",
			"ثابت س=1+2*3\nاطبع(س ,س،  س)\nمتغير ع=0\nع+=1\n",
			"ثابت س = 1 + 2 * 3\nاطبع(س, س، س)\nمتغير ع = 0\nع += 1\n"},
		{"no space inside parentheses, one inside braces",
			"اطبع( [ 1،2 ] )\nثابت ك2 = [1، 2].حول {هو*2}\nثابت ف = {}\n",
			"اطبع([1، 2])\nثابت ك2 = [1، 2].حول { هو * 2 }\nثابت ف = {}\n"},
		{"unary minus and not",
			"ثابت س = - 1\nثابت ص = (- س) * 2\nاطبع(! صح، ليس خطأ)\n",
			"ثابت س = -1\nثابت ص = (-س) * 2\nاطبع(!صح، ليس خطأ)\n"},
		{"members, optionals, ranges and named arguments",
			"ثابت ب = م ؟. بريد ؟؟ \"لا\"\nدالة ف(أ : نص ؟) = أ\nلكل ع في 1 .. 3 { اطبع(ع) }\nاطبع(نقطة(س : 1، ص:2) . بعد())\n",
			"ثابت ب = م؟.بريد ؟؟ \"لا\"\nدالة ف(أ: نص؟) = أ\nلكل ع في 1..3 { اطبع(ع) }\nاطبع(نقطة(س: 1، ص: 2).بعد())\n"},
		{"lambdas with parameters and arrows",
			"ثابت ج = [1].اختزل(0){أ،ب=>أ+ب}\n",
			"ثابت ج = [1].اختزل(0) { أ، ب => أ + ب }\n"},
		{"comments are kept and lined up",
			"// أول\nثابت س = 1   // واحد\nثابت صص = 22 // اثنان\n  /* كتلة\n   تبقى كما هي */\nاطبع(س)\n",
			"// أول\nثابت س = 1   // واحد\nثابت صص = 22 // اثنان\n/* كتلة\n   تبقى كما هي */\nاطبع(س)\n"},
		{"a comment before a closing brace is inside the block",
			"دالة ف() {\nاطبع(1)\n// لاحقاً\n}\n",
			"دالة ف() {\n  اطبع(1)\n  // لاحقاً\n}\n"},
		{"multi-line strings are untouched",
			"ثابت ن = \"\"\"\n  سطر   أول\n سطر\"\"\"\nاطبع( ن )\n",
			"ثابت ن = \"\"\"\n  سطر   أول\n سطر\"\"\"\nاطبع(ن)\n"},
		{"blank lines collapse, trailing spaces go, one final newline",
			"\n\nثابت س = 1   \n\n\n\nاطبع(س)\n\n\n",
			"ثابت س = 1\n\nاطبع(س)\n"},
		{"several brackets opened on one line indent once",
			"اطبع([\n1،\n2\n])\n",
			"اطبع([\n  1،\n  2\n])\n"},
		{"windows line endings and tabs",
			"إذا صح {\r\n\tاطبع(1)\r\n}\r\n",
			"إذا صح {\n  اطبع(1)\n}\n"},
		{"angle brackets keep the writer's spacing",
			"ثابت ن: قائمة<صحيح> = []\nاطبع(1<2، 3 > 2)\n",
			"ثابت ن: قائمة<صحيح> = []\nاطبع(1<2، 3 > 2)\n"},
		{"tables, routes and server functions",
			"جدول مهمة{\nعنوان:نص\nمنجزة=خطأ\n}\nخادم دالة أضف(عنوان:نص)=مهمة.أنشئ(عنوان:عنوان)\nمسار جلب \"/api/مهام/{رقم}\"=مهمة.بالرقم(رقم)\n",
			"جدول مهمة {\n  عنوان: نص\n  منجزة = خطأ\n}\nخادم دالة أضف(عنوان: نص) = مهمة.أنشئ(عنوان: عنوان)\nمسار جلب \"/api/مهام/{رقم}\" = مهمة.بالرقم(رقم)\n"},
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
