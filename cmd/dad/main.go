// Command dad (ضاد) is the Ḍād language toolchain.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/lsp"
	"github.com/mtwalli/dad-lang/internal/module"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

const usage = `ضاد — لغة برمجة عربية

الاستخدام:
  ضاد شغل <ملف.dad>      شغّل البرنامج
  ضاد فحص <ملف.dad>      افحص الملف واعرض الأخطاء والتنبيهات
                         (--json لإخراج يقرؤه المحرر، و«-» للقراءة من الدخل القياسي)
  ضاد نسق <ملف.dad>      نسّق الملف (المسافات والإزاحة) واحفظه
                         (--check لعرض الملفات غير المنسّقة، و«-» للدخل القياسي)
  ضاد تطوير <ملف.dad>    شغّل الخادم على http://localhost:8080 مع إعادة التحميل
                         (--port لاختيار منفذ آخر)
  ضاد انشر <ملف.dad>     شغّل البرنامج خادماً للنشر: بلا أدوات المطوّر ولا إعادة تحميل،
                         على كل الواجهات (--port أو $PORT، --host، --data لمسار قاعدة البيانات)
  ضاد ابن <ملف.dad>      حوّل البرنامج إلى JavaScript: موقع ثابت لبرامج الواجهات،
                         أو app.js يعمل بـ node (--out لاختيار المجلد)
  ضاد معاينة <ملف.dad>   صفحة الواجهة للمحرر، ليراها قبل التشغيل (JSON)
  ضاد رموز <ملف.dad>     اعرض الرموز (tokens) الناتجة عن الملف
  ضاد إصدار              اعرض رقم الإصدار
  ضاد مساعدة             اعرض هذه الرسالة

Usage (ASCII aliases):
  dad run <file.dad>       run a program (also: dad <file.dad>)
  dad check <file.dad>     check a file and report errors and warnings
                           (--json for editors; "-" reads the source from stdin)
  dad fmt <file.dad|dir>   format files in place (--check lists unformatted files;
                           "-" formats stdin to stdout)
  dad lsp                  the language server, for editors (LSP over stdin/stdout)
  dad dev <file.dad>       serve routes and server functions on localhost:8080,
                           reloading when the file changes (--port to change it)
  dad serve <file.dad>     run the program as a production server: no developer tools,
                           no reloading (--port or $PORT, --host, --data for the database)
  dad build <file.dad>     compile to JavaScript: a static site for programs with
                           an interface, otherwise app.js for node (--out dir)
  dad preview <file.dad>   the interface's page for an editor's preview, as JSON
                           (--path, --base, --nonce, --csp; --assets DIR writes its assets)
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
	cmd = strings.ReplaceAll(cmd, "ِ", "") // «ابنِ» is «ابن»
	switch cmd {
	case "شغل", "run":
		os.Exit(withPath("شغل", cmdRun))
	case "فحص", "check":
		args := os.Args[2:]
		if len(args) == 2 && args[0] == "--json" {
			os.Exit(cmdCheckJSON("", args[1]))
		}
		// «--json --path file.dad -»: the editor's unsaved text of file.dad, whose
		// imports are read from the disk
		if len(args) == 4 && args[0] == "--json" && args[1] == "--path" {
			os.Exit(cmdCheckJSON(args[2], args[3]))
		}
		os.Exit(withPath("فحص", cmdCheck))
	case "خادم_اللغة", "lsp":
		if err := lsp.Serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "نسق", "fmt", "format":
		os.Exit(cmdFmt(os.Args[2:]))
	case "تطوير", "dev":
		os.Exit(cmdDev(os.Args[2:]))
	case "انشر", "serve":
		os.Exit(cmdServe(os.Args[2:]))
	case "ابن", "build":
		os.Exit(cmdBuild(os.Args[2:]))
	case "معاينة", "preview":
		os.Exit(cmdPreview(os.Args[2:]))
	case "رموز", "tokens":
		os.Exit(withFile("رموز", cmdTokens))
	case "إصدار", "version", "--version":
		fmt.Println("ضاد", version)
	case "مساعدة", "help", "-h", "--help":
		fmt.Print(usage)
	default:
		if strings.HasSuffix(cmd, ".dad") && len(os.Args) == 2 {
			os.Exit(cmdRun(cmd))
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

// withPath passes the single file argument to run.
func withPath(name string, run func(path string) int) int {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "الاستخدام: ضاد %s <ملف.dad>\n", name)
		return 2
	}
	return run(os.Args[2])
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

// loadProgram reads and checks a program: its main file and every file it
// imports ("-" reads one file from standard input). It prints the problems
// and reports whether the program can run.
func loadProgram(path string) (*module.Program, bool) {
	var p *module.Program
	if path == "-" {
		src, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "تعذرت قراءة الدخل القياسي: %v\n", err)
			return nil, false
		}
		p = module.Single("-", string(src))
		check.Program(p)
	} else {
		p, _ = check.Load(path, readFile)
	}
	diags := p.Diags()
	for _, d := range diags {
		if d.Pos.Line == 0 && d.Path == path && p.Main.File == nil {
			fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»\n", path)
			continue
		}
		fmt.Fprintln(os.Stderr, d)
	}
	return p, !module.HasErrors(diags)
}

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

func cmdRun(path string) int {
	prog, ok := loadProgram(path)
	if !ok {
		return 1 // like a compiler: a program with mistakes doesn't start
	}
	in := interp.New(os.Stdout)
	store, err := openStore(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	in.Store = store
	err = in.RunProgram(prog)
	storeNotes(store)
	if len(in.Notices) > 0 {
		fmt.Fprintf(os.Stderr, "ملاحظة: لم يُشغَّل %s.\n", strings.Join(in.Notices, "، "))
	}
	switch {
	case len(in.Pages()) > 0:
		fmt.Fprintf(os.Stderr, "ملاحظة: لهذا البرنامج واجهة؛ لعرضها شغّل «ضاد تطوير %s» وافتح الصفحة.\n", filepath.Base(path))
	case len(in.Routes()) > 0 || len(in.ServerFunctions()) > 0:
		fmt.Fprintf(os.Stderr, "ملاحظة: للمسارات ودوال الخادم شغّل «ضاد تطوير %s».\n", filepath.Base(path))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", errorFile(path, err), err)
		return 1
	}
	return 0
}

// errorFile is the file a runtime error happened in.
func errorFile(path string, err error) string {
	if re, ok := err.(*interp.RuntimeError); ok && re.File != "" {
		return re.File
	}
	return path
}

func cmdCheck(path string) int {
	prog, ok := loadProgram(path)
	if !ok {
		return 1
	}
	toks := 0
	for _, m := range prog.Modules {
		t, _ := lexer.Tokenize(m.Src)
		toks += len(t)
	}
	files := ""
	if n := len(prog.Modules); n > 1 {
		files = fmt.Sprintf("، %d ملفات", n)
	}
	fmt.Printf("✓ %s: لا أخطاء (%d رمزاً%s)\n", path, toks, files)
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

// cmdCheckJSON checks a file for an editor. With a path, the file's imports
// are read from the disk and src (or standard input, for "-") is its text.
func cmdCheckJSON(path, src string) int {
	text, err := readFile(src)
	if src == "-" {
		var b []byte
		b, err = io.ReadAll(os.Stdin)
		text = string(b)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", src, err)
		return 1
	}
	var diags []lexer.Diagnostic
	if path == "" && src == "-" {
		_, diags = check.Source(text)
	} else {
		if path == "" {
			path = src
		}
		clean := filepath.Clean(path)
		_, diags = check.AnalyzeProgram(module.Load(path, func(p string) (string, error) {
			if filepath.Clean(p) == clean {
				return text, nil
			}
			return readFile(p)
		}))
	}
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
