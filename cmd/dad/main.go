// Command dad (ضاد) is the Ḍād language toolchain.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/parser"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

const usage = `ضاد — لغة برمجة عربية

الاستخدام:
  ضاد شغل <ملف.dad>      شغّل البرنامج
  ضاد فحص <ملف.dad>      افحص الملف واعرض الأخطاء والتنبيهات
                         (--json لإخراج يقرؤه المحرر، و«-» للقراءة من الدخل القياسي)
  ضاد تطوير <ملف.dad>    شغّل الخادم على http://localhost:8080 مع إعادة التحميل
                         (--port لاختيار منفذ آخر)
  ضاد رموز <ملف.dad>     اعرض الرموز (tokens) الناتجة عن الملف
  ضاد إصدار              اعرض رقم الإصدار
  ضاد مساعدة             اعرض هذه الرسالة

Usage (ASCII aliases):
  dad run <file.dad>       run a program (also: dad <file.dad>)
  dad check <file.dad>     check a file and report errors and warnings
                           (--json for editors; "-" reads the source from stdin)
  dad dev <file.dad>       serve routes and server functions on localhost:8080,
                           reloading when the file changes (--port to change it)
  dad tokens <file.dad>    print the tokens in a file
  dad version
  dad help
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	// "شغّل" with a shadda is the same command as "شغل".
	cmd = strings.ReplaceAll(cmd, "ّ", "")
	switch cmd {
	case "شغل", "run":
		os.Exit(withFile("شغل", cmdRun))
	case "فحص", "check":
		args := os.Args[2:]
		if len(args) == 2 && args[0] == "--json" {
			os.Exit(runWith(args[1], cmdCheckJSON))
		}
		os.Exit(withFile("فحص", cmdCheck))
	case "تطوير", "dev":
		os.Exit(cmdDev(os.Args[2:]))
	case "رموز", "tokens":
		os.Exit(withFile("رموز", cmdTokens))
	case "إصدار", "version", "--version":
		fmt.Println("ضاد", version)
	case "مساعدة", "help", "-h", "--help":
		fmt.Print(usage)
	default:
		if strings.HasSuffix(cmd, ".dad") && len(os.Args) == 2 {
			os.Exit(runFile(cmd))
		}
		fmt.Fprintf(os.Stderr, "أمر غير معروف «%s»\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

// withFile reads the single file argument and passes its contents to run.
func withFile(name string, run func(path, src string) int) int {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "الاستخدام: ضاد %s <ملف.dad>\n", name)
		return 2
	}
	return runWith(os.Args[2], run)
}

func runFile(path string) int { return runWith(path, cmdRun) }

func runWith(path string, run func(path, src string) int) int {
	var src []byte
	var err error
	if path == "-" {
		src, err = io.ReadAll(os.Stdin)
	} else {
		src, err = os.ReadFile(path)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", path, err)
		return 1
	}
	return run(path, string(src))
}

func printDiags(path string, diags []lexer.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, d)
	}
}

func cmdRun(path, src string) int {
	file, diags := parser.ParseFile(src)
	printDiags(path, diags)
	if file == nil {
		return 1
	}
	in := interp.New(os.Stdout)
	store, err := openStore(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	in.Store = store
	err = in.Run(file)
	if len(in.Notices) > 0 {
		fmt.Fprintf(os.Stderr, "ملاحظة: لم يُشغَّل %s — الواجهات تأتي في المرحلة 2.\n",
			strings.Join(in.Notices, "، "))
	}
	if len(in.Routes()) > 0 || len(in.ServerFunctions()) > 0 {
		fmt.Fprintf(os.Stderr, "ملاحظة: للمسارات ودوال الخادم شغّل «ضاد تطوير %s».\n", filepath.Base(path))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return 1
	}
	return 0
}

func cmdCheck(path, src string) int {
	file, diags := parser.ParseFile(src)
	printDiags(path, diags)
	if file == nil {
		return 1
	}
	toks, _ := lexer.Tokenize(src)
	fmt.Printf("✓ %s: لا أخطاء (%d رمزاً)\n", path, len(toks))
	return 0
}

func cmdTokens(path, src string) int {
	toks, diags := lexer.Tokenize(src)
	for _, t := range toks {
		fmt.Println(t)
	}
	printDiags(path, diags)
	if lexer.HasErrors(diags) {
		return 1
	}
	return 0
}

// jsonDiag is one problem in the output of "dad check --json", read by editor extensions.
// Line and column are 1-based and count characters (Unicode code points).
type jsonDiag struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Warning bool   `json:"warning"`
}

func cmdCheckJSON(_ string, src string) int {
	_, diags := parser.ParseFile(src)
	out := struct {
		Diagnostics []jsonDiag `json:"diagnostics"`
	}{Diagnostics: []jsonDiag{}}
	for _, d := range diags {
		out.Diagnostics = append(out.Diagnostics, jsonDiag{d.Pos.Line, d.Pos.Col, d.Msg, d.Warning})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return 1
	}
	return 0 // problems are reported in the JSON, not through the exit code
}
