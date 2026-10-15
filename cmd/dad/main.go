// Command dad (ضاد) is the Ḍād language toolchain.
package main

import (
	"fmt"
	"os"

	"github.com/dad-lang/dad/internal/lexer"
)

const version = "0.0.1-dev"

const usage = `ضاد — لغة برمجة عربية

الاستخدام:
  ضاد رموز <ملف.dad>     اعرض الرموز (tokens) الناتجة عن الملف
  ضاد إصدار              اعرض رقم الإصدار
  ضاد مساعدة             اعرض هذه الرسالة

Usage (ASCII aliases):
  dad tokens <file.dad>
  dad version
  dad help
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "رموز", "tokens":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "الاستخدام: ضاد رموز <ملف.dad>")
			os.Exit(2)
		}
		os.Exit(cmdTokens(os.Args[2]))
	case "إصدار", "version", "--version":
		fmt.Println("ضاد", lexer.ToArabicDigits(version), "("+version+")")
	case "مساعدة", "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "أمر غير معروف «%s»\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func cmdTokens(path string) int {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", path, err)
		return 1
	}
	toks, diags := lexer.Tokenize(string(src))
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
