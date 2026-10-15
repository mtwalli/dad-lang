// Command dad (ضاد) is the Ḍād language toolchain.
package main

import (
	"fmt"
	"os"

	"github.com/mtwalli/dad-lang/internal/lexer"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

const usage = `ضاد — لغة برمجة عربية

الاستخدام:
  ضاد فحص <ملف.dad>      افحص الملف واعرض الأخطاء والتنبيهات
  ضاد رموز <ملف.dad>     اعرض الرموز (tokens) الناتجة عن الملف
  ضاد إصدار              اعرض رقم الإصدار
  ضاد مساعدة             اعرض هذه الرسالة

Usage (ASCII aliases):
  dad check <file.dad>     check a file and report errors and warnings
  dad tokens <file.dad>    print the tokens in a file
  dad version
  dad help
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "فحص", "check":
		os.Exit(withFile("فحص", cmdCheck))
	case "رموز", "tokens":
		os.Exit(withFile("رموز", cmdTokens))
	case "إصدار", "version", "--version":
		fmt.Println("ضاد", lexer.ToArabicDigits(version), "("+version+")")
	case "مساعدة", "help", "-h", "--help":
		fmt.Print(usage)
	default:
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
	path := os.Args[2]
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", path, err)
		return 1
	}
	return run(path, string(src))
}

func cmdCheck(path, src string) int {
	toks, diags := lexer.Tokenize(src)
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, d)
	}
	if lexer.HasErrors(diags) {
		return 1
	}
	fmt.Printf("✓ %s: لا أخطاء (%s رمزاً)\n", path, lexer.ToArabicDigits(fmt.Sprint(len(toks))))
	return 0
}

func cmdTokens(path, src string) int {
	toks, diags := lexer.Tokenize(src)
	for _, t := range toks {
		fmt.Println(t)
	}
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, d)
	}
	if lexer.HasErrors(diags) {
		return 1
	}
	return 0
}
