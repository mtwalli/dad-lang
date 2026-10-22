// Package web holds the browser side of Ḍād pages: the page shell, the default
// stylesheet, the Kufi font and the script that sends events to the program.
package web

import (
	"embed"
	"html"
	"io/fs"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// closingTag finds «</script» and «</style» in code written into a page.
var closingTag = regexp.MustCompile(`(?i)</(script|style)`)

//go:embed assets
var assets embed.FS

// Assets are served under AssetsPath: ui.js, ui.css and fonts/.
const AssetsPath = "/_dad/assets/"

// Handler serves the assets.
func Handler() http.Handler {
	sub, _ := fs.Sub(assets, "assets")
	files := http.StripPrefix(AssetsPath, http.FileServer(http.FS(sub)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}

// File returns one asset, such as "ui.js".
func File(name string) ([]byte, error) { return assets.ReadFile("assets/" + name) }

// PageOptions describe a page.
type PageOptions struct {
	Title   string
	Body    string // the rendered component
	Session string
	Version int
	Error   string   // shown at the bottom, if any
	Live    int      // dad dev: the program version shown; the page reloads when it changes (0: never)
	Assets  string   // where ui.js and ui.css are; AssetsPath when empty
	Scripts []string // scripts loaded before ui.js, such as a compiled program
	Routing string   // how pages keep their address: "history" (default), "hash" or "none"; see ui.js
	Styles  []string // the program's own stylesheets, after ui.css

	// For a page shown inside an editor (dad preview): a Content-Security-Policy,
	// and code and styles written into the page, its scripts marked with Nonce.
	CSP           string
	Nonce         string
	InlineScripts []string // after Scripts, before ui.js
	InlineStyles  []string // after Styles
}

// Assets lists every asset by name: ui.js, ui.css and fonts/….
func Assets() []string {
	var out []string
	fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, strings.TrimPrefix(path, "assets/"))
		}
		return nil
	})
	return out
}

// Page is a whole HTML page showing a rendered component.
func Page(o PageOptions) string {
	base := o.Assets
	if base == "" {
		base = AssetsPath
	}
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
`)
	if o.CSP != "" {
		b.WriteString(`<meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(o.CSP) + `">
`)
	}
	b.WriteString(`<title>`)
	b.WriteString(html.EscapeString(o.Title))
	b.WriteString(`</title>
<link rel="stylesheet" href="` + html.EscapeString(base) + `ui.css">
`)
	for _, href := range o.Styles {
		b.WriteString(`<link rel="stylesheet" href="` + html.EscapeString(href) + `">
`)
	}
	for _, css := range o.InlineStyles {
		b.WriteString("<style>\n" + closingTag.ReplaceAllString(css, `<\/$1`) + "\n</style>\n")
	}
	b.WriteString(`</head>
<body>
<main id="dad-root" data-session="`)
	b.WriteString(html.EscapeString(o.Session))
	b.WriteString(`" data-version="` + strconv.Itoa(o.Version) + `"`)
	if o.Live > 0 {
		b.WriteString(` data-live="` + strconv.Itoa(o.Live) + `"`)
	}
	if o.Routing != "" {
		b.WriteString(` data-routing="` + html.EscapeString(o.Routing) + `"`)
	}
	b.WriteString(">")
	b.WriteString(o.Body)
	b.WriteString("</main>\n<div id=\"dad-error\" role=\"alert\"")
	if o.Error == "" {
		b.WriteString(" hidden")
	}
	b.WriteString(">")
	b.WriteString(html.EscapeString(o.Error))
	b.WriteString("</div>\n")
	nonce := ""
	if o.Nonce != "" {
		nonce = ` nonce="` + html.EscapeString(o.Nonce) + `"`
	}
	for _, src := range o.Scripts {
		b.WriteString("<script" + nonce + " src=\"" + html.EscapeString(src) + "\"></script>\n")
	}
	for _, code := range o.InlineScripts {
		// «</script» inside the code would end the element early
		b.WriteString("<script" + nonce + ">\n" + closingTag.ReplaceAllString(code, `<\/$1`) + "\n</script>\n")
	}
	b.WriteString("<script" + nonce + " src=\"" + html.EscapeString(base) + "ui.js\"></script>\n</body>\n</html>\n")
	return b.String()
}
