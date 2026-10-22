package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/codegen/js"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/web"
)

// cmdPreview makes the page an editor shows beside the code, so the interface
// can be seen (and clicked) before the program runs. The program is compiled
// to JavaScript as by «dad build» and runs in the page: server functions run
// there too, and tables are kept in the page's storage.
//
//	dad preview --assets DIR
//	    writes ui.js, ui.css, the fonts and dad-runtime.js into DIR, once
//	dad preview [--path FILE] [--base URL] [--nonce N] [--csp POLICY] (FILE | -)
//	    prints {"site", "html", "title", "diagnostics"} as JSON. "-" reads the
//	    text of FILE from standard input (the editor's unsaved copy); the files
//	    it imports and FILE's own stylesheet (FILE.css) are read from the disk.
//	    BASE is where the assets are, as the page sees them.
func cmdPreview(args []string) int {
	var path, src, base, nonce, csp, assets string
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := func() string {
			if i+1 >= len(args) {
				return ""
			}
			i++
			return args[i]
		}
		switch a {
		case "--path":
			path = value()
		case "--base":
			base = value()
		case "--nonce":
			nonce = value()
		case "--csp":
			csp = value()
		case "--assets":
			assets = value()
		default:
			src = a
		}
	}
	if assets != "" {
		if err := writeAssets(assets); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if src == "" {
		fmt.Fprintln(os.Stderr, "الاستخدام: ضاد معاينة [--path ملف.dad] [--base رابط] (ملف.dad | -)")
		return 2
	}
	if path == "" {
		path = src
	}
	var text string
	var err error
	if src == "-" {
		var b []byte
		b, err = io.ReadAll(os.Stdin)
		text = string(b)
	} else {
		text, err = readFile(src)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "تعذرت قراءة الملف «%s»: %v\n", src, err)
		return 1
	}
	out := previewPage(path, text, base, nonce, csp)
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return 1
	}
	return 0
}

type previewDiag struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Warning bool   `json:"warning"`
}

type preview struct {
	// Site is false for a program without an interface: nothing to show.
	Site bool `json:"site"`
	// HTML is the whole page; empty when the program has errors.
	HTML        string        `json:"html"`
	Title       string        `json:"title"`
	Diagnostics []previewDiag `json:"diagnostics"`
}

// previewPage compiles the program whose main file at path has the given text.
func previewPage(path, text, base, nonce, csp string) preview {
	clean := filepath.Clean(path)
	prog := module.Load(path, func(p string) (string, error) {
		if filepath.Clean(p) == clean {
			return text, nil
		}
		return readFile(p)
	})
	check.Program(prog)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	out := preview{Title: name, Diagnostics: []previewDiag{}}
	errors := false
	for _, d := range prog.Diags() {
		out.Diagnostics = append(out.Diagnostics, previewDiag{d.Path, d.Pos.Line, d.Pos.Col, d.Msg, d.Warning})
		errors = errors || !d.Warning
	}
	if errors {
		return out
	}
	compiled := js.CompileProgram(prog)
	out.Site = compiled.Site()
	if !out.Site {
		return out
	}
	var styles []string
	if css, err := os.ReadFile(stylePath(path)); err == nil {
		styles = append(styles, string(css))
	}
	out.HTML = web.Page(web.PageOptions{
		Title:         name,
		Assets:        base,
		Scripts:       []string{base + "dad-runtime.js"},
		InlineScripts: []string{compiled.Browser(filepath.Base(path))},
		InlineStyles:  styles,
		Routing:       "hash",
		CSP:           csp,
		Nonce:         nonce,
	})
	return out
}

// writeAssets writes what a preview page loads: the browser assets and the runtime.
func writeAssets(dir string) error {
	files := map[string][]byte{"dad-runtime.js": []byte(js.Runtime)}
	for _, a := range web.Assets() {
		b, err := web.File(a)
		if err != nil {
			return err
		}
		files[a] = b
	}
	for rel, b := range files {
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
