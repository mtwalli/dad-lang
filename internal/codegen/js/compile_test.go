package js_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/codegen/js"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/module"
)

// Every program must print the same thing run by the interpreter and
// compiled to JavaScript and run by Node, down to the error messages.

func node(t *testing.T) string {
	path, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	return path
}

// interpret runs a program with the interpreter: its output, then its error.
func interpret(t *testing.T, src string) string {
	file, diags := check.Source(src)
	if file == nil || lexer.HasErrors(diags) {
		t.Fatalf("the program has errors: %v", diags)
	}
	var out bytes.Buffer
	in := interp.New(&out)
	if err := in.Run(file); err != nil {
		out.WriteString("ERROR: " + err.Error() + "\n")
	}
	return out.String()
}

// compiled runs the compiled program with Node.
func compiled(t *testing.T, nodePath, src string) string {
	file, _ := check.Source(src)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "dad-runtime.js"), js.Runtime)
	write(t, filepath.Join(dir, "app.js"), js.Compile(file).Node("app.dad"))
	cmd := exec.Command(nodePath, "app.js")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.Run()
	got := stdout.String()
	for _, l := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
		if msg, ok := strings.CutPrefix(l, "app.dad: "); ok {
			got += "ERROR: " + msg + "\n"
		} else if l != "" {
			got += "STDERR: " + l + "\n"
		}
	}
	return got
}

func write(t *testing.T, path, s string) {
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func same(t *testing.T, name, src string) {
	t.Helper()
	nodePath := node(t)
	want := interpret(t, src)
	got := compiled(t, nodePath, src)
	if os.Getenv("DAD_SHOW") != "" {
		t.Logf("%s:\n%s", name, got)
	}
	if got != want {
		t.Errorf("%s: the compiled program printed something else\n--- interpreter\n%s\n--- compiled\n%s", name, want, got)
		file, _ := check.Source(src)
		os.WriteFile(filepath.Join(os.TempDir(), "dad-failed-"+filepath.Base(name)+".js"), []byte(js.Compile(file).Node("app.dad")), 0o644)
	}
}

func TestExamples(t *testing.T) {
	files, _ := filepath.Glob("../../../examples/*.dad")
	files = append(files, "../../interp/testdata/features.dad")
	if len(files) < 2 {
		t.Fatal("no examples found")
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(filepath.Base(f), func(t *testing.T) { same(t, f, string(src)) })
	}
}

// Snippets for the rules the examples don't reach, errors above all.
var snippets = map[string]string{
	"numbers": `اطبع(7 / 2، 7 % 3، -7 % 3، 2.5 * 2، 1.0 / 3، 0.1 + 0.2، 1000000000000000000000.0 * 10، 0.0000001، -0.0)
اطبع((2.5).تقريب()، (-2.5).تقريب()، (3.14159).تقريب(2)، (2.7).أرضية()، (2.1).سقف()، (-4).مطلق())
اطبع("12" ك صحيح، "٣٫٥" ك عشري، 3.9 ك صحيح، 5 ك نص + "!"، 9223372036854775807 + 1)
اطبع(جذر(16)، مطلق(-3.5)، أدنى(3، 1، 2)، أقصى([4، 9، 2])، أدنى(1..5))`,
	"strings": `ثابت ن = "  مرحبا بالعالم  "
اطبع(ن.قص()، ن.طول، "أ،ب،ج".قسم("،")، "abc".كبير()، "ها".كرر(3)، "سلام".أحرف())
اطبع("abc".استبدل("b"، "-")، "ضاد".يبدأ_ب("ض")، "ضاد".ينتهي("د")، "ضاد"[1]، "ب" < "ت")
اطبع("قيمة: {1 + 2} و{[1، 2]}"، "" + عدم + صح)`,
	"lists": `متغير ق = [3، 1، 2]
ق.أضف(5)
ق[0] = 10
ق[1] += 1
اطبع(ق، ق.رتب()، ق.عكس()، ق.مجموع()، ق.رشح { هو > 2 }، ق.حول { هو * 2 }، ق.ضم("-"))
اطبع(ق.أول، ق.آخر، ق.عدد { هو > 2 }، ق.أي { هو == 5 }، ق.كل { هو > 0 }، ق.اختزل(0) { أ، ب => أ + ب }، ق.اختزل { أ، ب => أ * ب })
اطبع(من 1 إلى 5، من 0 حتى 10 بخطوة 3، 5..1، (1..3) == [1، 2، 3]، [1، 2] + [3]، [].فارغ، [1، [2، "ب"]])
متغير م = ["أ": 1، "ب": 2]
م["ج"] = 3
م["أ"] += 10
م.احذف("ب")
اطبع(م، م.مفاتيح، م.قيم، م["س"]، م.يحتوي("ج")، [:]، م.طول)`,
	"types": `نوع نقطة(س: عشري، ص: عشري = 0) {
    دالة بعد() = جذر(س * س + ص * ص)
    دالة أضف(أخرى: نقطة): نقطة = نقطة(س + أخرى.س، ص + أخرى.ص)
    دالة وصف() = "({س}، {ص}) على بعد {بعد()}"
}
تعداد شكل { دائرة(نق: عشري)، مربع(ض: عشري)، لا_شيء }
دالة مساحة(ش: شكل): عشري = طابق ش {
    دائرة(نق) => 3.0 * نق * نق
    مربع(ض) => ض * ض
    لا_شيء => 0
}
امتداد صحيح { دالة ضعف() = هذا * 2 }
امتداد نقطة { دالة مقلوبة() = نقطة(ص، س) }
ثابت ن = نقطة(3، 4)
اطبع(ن، ن.بعد()، ن.أضف(نقطة(1، 1))، ن.وصف()، ن.نسخة(ص: 0)، ن.مقلوبة()، (21).ضعف())
اطبع(مساحة(دائرة(2))، مساحة(مربع(ض: 3))، مساحة(لا_شيء)، دائرة(1) == دائرة(1)، شكل.لا_شيء، دائرة(1.5))
اطبع(ن من نوع نقطة، 3 من نوع نص، عدم من نوع صحيح؟، دائرة(1) من نوع شكل، دائرة(1) من نوع مربع، [1] من نوع [صحيح])`,
	"control": `دالة تصنيف(ن: صحيح): نص {
    إذا ن < 0 { أرجع "سالب" }
    لكل ع في 1..3 {
        إذا ع == ن { أرجع "صغير" }
    }
    أرجع "كبير"
}
اطبع(تصنيف(-1)، تصنيف(2)، تصنيف(9))
متغير مجموع = 0
لكل س في من 1 إلى 10 {
    إذا س % 2 == 0 { تابع }
    إذا س > 7 { توقف }
    مجموع += س
}
اطبع(مجموع)
ثابت قيمة = إذا مجموع > 10 { "كبير" } وإلا إذا مجموع > 5 { "وسط" } وإلا { "صغير" }
ثابت كمية = {
    ثابت أ = 2
    أ * 21
}
اطبع(قيمة، كمية)
دالة أول_زوجي(ق: [صحيح]): صحيح؟ {
    لكل س في ق {
        ثابت ر = طابق س % 2 {
            0 => { أرجع س }
            _ => س
        }
    }
    أرجع عدم
}
اطبع(أول_زوجي([1، 3، 4، 5])، أول_زوجي([1]))
متغير ع = 0
طالما صح {
    ع += 1
    ثابت ف = إذا ع > 3 { توقف } وإلا { ع }
}
اطبع(ع)
متغير ش = 1.5
ش = 2
اطبع(ش)`,
	"closures": `دالة صانع(بداية: صحيح) {
    متغير ع = بداية
    أرجع { ع += 1؛ ع }
}
ثابت عد = صانع(10)
عد()
اطبع(عد()، عد)
ثابت مضاعف = { س => س * 2 }
اطبع(مضاعف(4)، [1، 2].حول(مضاعف))
دالة طبق(د: (صحيح) => صحيح، س: صحيح = 1): صحيح = د(س)
اطبع(طبق(مضاعف)، طبق(مضاعف، س: 5)، طبق({ هو + 100 }))
ثابت س = 1
دالة ظل() {
    اطبع(س)
    ثابت س = 2
    اطبع(س)
}
ظل()`,
	"errors": `دالة قسمة(أ: صحيح، ب: صحيح): عشري {
    إذا ب == 0 { ارفع مشكلة("قسمة على صفر يا صديقي") }
    أرجع أ / ب
}
ثابت ن = حاول { قسمة(1، 0) } التقط خ { اطبع("التقطنا:"، خ.رسالة)؛ -1.0 }
اطبع(ن)
حاول { ارفع "نص" } التقط خ { اطبع(خ، خ من نوع مشكلة) }
حاول { [1][5] } التقط خ { اطبع(خ.رسالة) }
حاول { 1 / 0 } التقط خ { اطبع(خ.رسالة) }
حاول { "x" ك صحيح } التقط خ { اطبع(خ.رسالة) }
اطبع(عدم؟.طول، عدم ?? "بديل")
قسمة(1، 0)`,
	"runtime error": `ثابت ق = [1، 2]
اطبع("قبل")
دالة اقرأ(س: أي) = س.غير_موجود
اطبع(اقرأ(ق))`,
	"missing return": `دالة ف(س: صحيح): صحيح {
    إذا س > 0 { أرجع 1 }
}
اطبع(ف(1))
ف(0)`,
	"wrong type": `دالة ف(س: صحيح) = س
دالة كما(س: أي) = س
ف(كما("نص"))`,
	"recursion": `دالة ف(ن: صحيح): صحيح = ف(ن + 1)
ف(0)`,
	"tables": `جدول مهمة { عنوان: نص، منجزة = خطأ }
مهمة.أنشئ(عنوان: "أ")
ثابت ب = مهمة.أنشئ(عنوان: "ب")
ب.نسخة(منجزة: صح).احفظ()
اطبع(مهمة.الكل()، مهمة.عدد()، مهمة.حيث { هو.منجزة }، مهمة.بالرقم(1)، مهمة.بالرقم(9))
اطبع(ب.احذف()، مهمة.عدد())`,
}

func TestSnippets(t *testing.T) {
	for name, src := range snippets {
		t.Run(name, func(t *testing.T) { same(t, name, src) })
	}
}

// Pages must render the same HTML, and change the same way after the same
// clicks and typing.
func TestInterfaces(t *testing.T) {
	nodePath := node(t)
	for _, f := range []string{"../../../examples/05-todo.dad", "../../../examples/06-counter.dad"} {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			file, _ := check.Source(string(src))
			in := interp.New(io.Discard)
			if err := in.Run(file); err != nil {
				t.Fatal(err)
			}
			root := in.RootComponent()
			sess := interp.NewUISession()
			page, err := in.Render(sess, root)
			if err != nil {
				t.Fatal(err)
			}
			pages := []string{page}
			type event struct {
				ID    string `json:"id"`
				Value any    `json:"value"`
			}
			var events []event
			handler := regexp.MustCompile(`<[a-z]+ [^>]*data-(click|input|toggle)="(h\d+)"[^>]*>`)
			for step := 0; step < 14; step++ {
				hs := handler.FindAllStringSubmatch(page, -1)
				if len(hs) == 0 {
					break
				}
				h := hs[(step*5)%len(hs)]
				var v any
				switch h[1] {
				case "input":
					v = []string{"سارة", "شراء الخبز"}[step%2]
					if strings.Contains(h[0], `type="number"`) {
						v = "7"
					}
				case "toggle":
					v = step%2 == 0
				}
				events = append(events, event{h[2], v})
				if err := in.Event(sess, sess.Version, h[2], v); err != nil {
					t.Fatal(err)
				}
				if page, err = in.Render(sess, root); err != nil {
					t.Fatal(err)
				}
				pages = append(pages, page)
			}

			evJSON, _ := json.Marshal(events)
			dir := t.TempDir()
			write(t, filepath.Join(dir, "dad-runtime.js"), js.Runtime)
			write(t, filepath.Join(dir, "ui.js"), `"use strict";
const $ = require("./dad-runtime.js");
const root = $.main(() => {
`+js.Compile(file).Body+`}, "app.dad");
const sess = $.newSession();
const pages = [$.render(sess, root)];
for (const e of `+string(evJSON)+`) {
  $.event(sess, sess.version, e.id, e.value);
  pages.push($.render(sess, root));
}
console.log(JSON.stringify(pages));
`)
			cmd := exec.Command(nodePath, "ui.js")
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			var got []string
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if len(got) != len(pages) {
				t.Fatalf("%d pages, want %d", len(got), len(pages))
			}
			for i := range pages {
				if got[i] != pages[i] {
					t.Errorf("after %d events:\n--- interpreter\n%s\n--- compiled\n%s", i, pages[i], got[i])
					break
				}
			}
			distinct := map[string]bool{}
			for _, p := range pages {
				distinct[regexp.MustCompile(`h\d+`).ReplaceAllString(p, "")] = true
			}
			if len(events) < 5 || len(distinct) < 3 {
				t.Errorf("only %d events and %d different pages were tried", len(events), len(distinct))
			}
		})
	}
}

// Programs of several files: each file compiles to a module of its own.
func TestPrograms(t *testing.T) {
	nodePath := node(t)
	disk := func(p string) (string, error) {
		b, err := os.ReadFile(p)
		return string(b), err
	}
	memory := func(files map[string]string) module.Reader {
		return func(p string) (string, error) {
			if s, ok := files[filepath.ToSlash(p)]; ok {
				return s, nil
			}
			return "", os.ErrNotExist
		}
	}
	programs := map[string]struct {
		main string
		read module.Reader
	}{
		"example": {"../../../examples/07-modules/الرئيسي.dad", disk},
		"names": {"main.dad", memory(map[string]string{
			"main.dad": `استورد عد، زد، شكل، العدد، نقطة من "./عداد"
استورد وصف من "./أشكال"
دالة مساعد() = "مساعد الرئيسي"
زد()
زد()
اطبع(عد()، العدد، مساعد()، وصف(دائرة(2))، وصف(مربع)، شكل.مربع)
ثابت ن = نقطة(1، 2)
اطبع(ن، ن.مجموع()، ن من نوع نقطة، (5).مضاعف())
ارفع مشكلة("من الرئيسي")
`,
			"عداد.dad": `متغير العدد = 0
خاص دالة مساعد() = 1
دالة زد() { العدد += مساعد() }
دالة عد() = العدد
تعداد شكل { دائرة(نق: صحيح)، مربع }
نوع نقطة(س: صحيح، ص: صحيح) {
    دالة مجموع() = س + ص
}
امتداد صحيح { دالة مضاعف() = هذا * 2 }
`,
			"أشكال.dad": `استورد شكل من "./عداد"
دالة وصف(ش: شكل): نص = طابق ش {
    دائرة(نق) => "دائرة {نق}"
    مربع => "مربع"
}
`,
		})},
	}
	for name, p := range programs {
		t.Run(name, func(t *testing.T) {
			prog, diags := check.Load(p.main, p.read)
			if len(diags) > 0 {
				t.Fatalf("%v", diags)
			}
			var want bytes.Buffer
			if err := interp.New(&want).RunProgram(prog); err != nil {
				want.WriteString("ERROR: " + err.Error() + "\n")
			}
			dir := t.TempDir()
			write(t, filepath.Join(dir, "dad-runtime.js"), js.Runtime)
			write(t, filepath.Join(dir, "app.js"), js.CompileProgram(prog).Node("app.dad"))
			cmd := exec.Command(nodePath, "app.js")
			cmd.Dir = dir
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			cmd.Run()
			got := stdout.String()
			if msg, ok := strings.CutPrefix(strings.TrimSpace(stderr.String()), "app.dad: "); ok {
				got += "ERROR: " + msg + "\n"
			}
			if got != want.String() {
				t.Errorf("--- interpreter\n%s\n--- compiled\n%s\n%s", want.String(), got, stderr.String())
			}
		})
	}
}
