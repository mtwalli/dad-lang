package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/codegen/js"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/web"
)

// cmdBuild compiles a program to JavaScript: «dad build app.dad [--out dir]».
//
// A program with an interface becomes a static site (index.html, the compiled
// program, the runtime and the page assets) that works from any web host with
// no server: tables are kept in the browser's storage. Any other program
// becomes app.js, run with Node.
func cmdBuild(args []string) int {
	var path, out string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--out" || a == "-o":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "«--out» يحتاج اسم مجلد")
				return 2
			}
			i++
			out = args[i]
		case strings.HasPrefix(a, "--out="):
			out = strings.TrimPrefix(a, "--out=")
		case path == "":
			path = a
		default:
			path = ""
			i = len(args)
		}
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "الاستخدام: ضاد ابن <ملف.dad> [--out مجلد]")
		return 2
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", path, err)
		return 1
	}
	file, diags := check.Source(string(src))
	printDiags(path, diags)
	if file == nil || lexer.HasErrors(diags) {
		return 1
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if out == "" {
		out = filepath.Join(filepath.Dir(path), "dist", name)
	}
	prog := js.Compile(file)
	base := filepath.Base(path)

	files := map[string][]byte{"dad-runtime.js": []byte(js.Runtime)}
	if prog.Root != "" {
		files["app.js"] = []byte(prog.Browser(base))
		files["index.html"] = []byte(web.Page(web.PageOptions{
			Title:   name,
			Assets:  "assets/",
			Scripts: []string{"dad-runtime.js", "app.js"},
		}))
		for _, a := range web.Assets() {
			b, err := web.File(a)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			files["assets/"+a] = b
		}
	} else {
		files["app.js"] = []byte(prog.Node(base))
	}
	for rel, b := range files {
		dst := filepath.Join(out, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	if prog.Root != "" {
		fmt.Printf("✓ بُني «%s» في %s — افتح index.html أو انشر المجلد على أي استضافة ثابتة\n", base, out)
		if prog.ServerFuncs > 0 {
			fmt.Fprintln(os.Stderr, "ملاحظة: دوال الخادم تعمل في الصفحة نفسها، والجداول تُحفظ في متصفح كل زائر.")
		}
	} else {
		fmt.Printf("✓ بُني «%s» في %s — شغّله بـ: node %s\n", base, out, filepath.Join(out, "app.js"))
	}
	if prog.Routes > 0 {
		fmt.Fprintln(os.Stderr, "ملاحظة: المسارات («مسار») تحتاج خادماً، فلم تُضمَّن؛ لها شغّل «ضاد تطوير».")
	}
	return 0
}
