package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtwalli/dad-lang/internal/format"
)

const fmtUsage = `الاستخدام: ضاد نسق [--check] <ملف.dad أو مجلد>…
  بلا خيارات: ينسّق الملفات ويحفظها
  --check:    يذكر الملفات غير المنسّقة دون تغييرها (للفحص في CI)
  «-»:        يقرأ من الدخل القياسي ويكتب النتيجة إلى الخرج القياسي
`

// cmdFmt formats files in place, like gofmt -w.
func cmdFmt(args []string) int {
	check := false
	var paths []string
	for _, a := range args {
		switch a {
		case "--check", "-check", "-l":
			check = true
		case "-h", "--help":
			fmt.Print(fmtUsage)
			return 0
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		fmt.Fprint(os.Stderr, fmtUsage)
		return 2
	}
	if len(paths) == 1 && paths[0] == "-" {
		src, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		out, err := format.Source(string(src))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Print(out)
		return 0
	}

	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "تعذرت قراءة «%s»: %v\n", p, err)
			return 1
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && path != p && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(path, ".dad") {
				files = append(files, path)
			}
			return nil
		})
	}

	code := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "تعذرت قراءة «%s»: %v\n", path, err)
			code = 1
			continue
		}
		out, err := format.Source(string(src))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			code = 1
			continue
		}
		if out == string(src) {
			continue
		}
		if check {
			fmt.Println(path)
			code = 1
			continue
		}
		if err := writeKeepingMode(path, out); err != nil {
			fmt.Fprintf(os.Stderr, "تعذر حفظ «%s»: %v\n", path, err)
			code = 1
		}
	}
	return code
}

func writeKeepingMode(path, data string) error {
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp := path + ".dadfmt"
	if err := os.WriteFile(tmp, []byte(data), mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
