package interp

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

// Pages: «صفحة "/المهام" = قائمة_المهام()» gives a part of the interface an
// address of its own. A page in the browser opens an address (Open), is laid out
// (RenderPage), and moves to another page when a link to one is clicked or an
// event calls «انتقل».

// Page is a declared صفحة, or the main component of a program without pages.
type Page struct {
	Path string
	Pos  token.Pos
	body ast.Expr
	env  *Env
	comp *Component // the main component, shown at "/" when no صفحة is declared
}

// Pages lists the program's pages, in source order. A program that declares
// none but has a main component shows that component at "/", as before pages
// existed.
func (in *Interp) Pages() []*Page {
	if len(in.pages) > 0 {
		return in.pages
	}
	if c := in.RootComponent(); c != nil {
		return []*Page{{Path: "/", Pos: c.Pos, comp: c}}
	}
	return nil
}

// HasPages reports whether the program declares pages with «صفحة».
func (in *Interp) HasPages() bool { return len(in.pages) > 0 }

// MatchPage finds the page for an address and the values of its {names}.
func (in *Interp) MatchPage(path string) (*Page, map[string]string) {
	path = cleanPath(path)
	for _, p := range in.Pages() {
		if params, ok := matchPath(p.Path, path); ok {
			return p, params
		}
	}
	return nil, nil
}

// Open points a session at an address: the next render shows the page there,
// or the page that says there is none. It reports whether a page was found.
func (in *Interp) Open(sess *UISession, path string) bool {
	sess.Path = cleanPath(path)
	sess.page, sess.params = in.MatchPage(sess.Path)
	sess.NotFound = sess.page == nil
	return !sess.NotFound
}

// RenderPage lays out the page a session shows and returns its HTML.
func (in *Interp) RenderPage(sess *UISession) (string, error) {
	p := sess.page
	if p == nil {
		html, err := in.render(sess, func() { in.notFound(sess.Path) })
		sess.Title = "الصفحة غير موجودة"
		return html, err
	}
	if p.comp != nil {
		return in.render(sess, func() { in.callValue(p.comp, &callInfo{pos: p.comp.Pos, name: p.comp.Name}) })
	}
	return in.render(sess, func() {
		env := newEnv(p.env)
		for k, v := range sess.params {
			var val Value = v
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				val = n
			}
			env.define(k, val, false)
		}
		// The page is laid out like a component of its own, so «حالة» works in its
		// block; its state belongs to its address, and is forgotten when the page
		// moves to another.
		ctx := in.ui
		ctx.frames = append(ctx.frames, &frame{key: "@" + sess.Path, counts: map[string]int{}})
		ctx.seen["@"+sess.Path] = true
		defer func() { ctx.frames = ctx.frames[:len(ctx.frames)-1] }()
		in.runBody(&Func{Name: "صفحة " + p.Path, Body: p.body, Env: p.env, IsLam: true}, env)
	})
}

// notFound lays out the page shown at an address with no page.
func (in *Interp) notFound(path string) {
	col := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-col dad-notfound"}}}
	col.Kids = append(col.Kids,
		&Node{Tag: "h1", Text: "الصفحة غير موجودة"},
		&Node{Tag: "p", Text: "لا توجد صفحة بالعنوان «" + path + "»."})
	if home, _ := in.MatchPage("/"); home != nil {
		a := &Node{Tag: "a", Text: "إلى الصفحة الرئيسية"}
		a.attr("href", "/")
		a.attr("data-nav", "")
		col.Kids = append(col.Kids, a)
	}
	in.emit(col)
}

// navigate runs «انتقل("/المهام")»: after the event, the page moves to that
// address. An address on another site («https://…») leaves for it.
func (in *Interp) navigate(ci *callInfo) Value {
	if len(ci.args) != 1 || len(ci.named) > 0 {
		in.fail(ci.pos, "«انتقل» يأخذ عنوان الصفحة، مثل: انتقل(\"/المهام\")")
	}
	to, ok := ci.args[0].(string)
	if !ok {
		in.fail(ci.pos, "«انتقل» يأخذ عنواناً نصياً، مثل: انتقل(\"/المهام\")، لكن وجدت «%s»", typeName(ci.args[0]))
	}
	sess := in.event
	if sess == nil {
		if in.ui != nil {
			in.fail(ci.pos, "«انتقل» تُستخدم عند حدث، مثل الضغط على زر، لا أثناء عرض الصفحة")
		}
		in.fail(ci.pos, "«انتقل» تُستخدم في صفحة ويب، عند حدث مثل الضغط على زر")
	}
	switch {
	case strings.HasPrefix(to, "https://") || strings.HasPrefix(to, "http://"):
		sess.Away = to
	case strings.HasPrefix(to, "/"):
		in.Open(sess, to)
		sess.Moved = true
	default:
		in.fail(ci.pos, "عنوان الصفحة في «انتقل» يبدأ بـ «/»، مثل: انتقل(\"/المهام\")")
	}
	return nil
}

// cleanPath is an address without its query, fragment or %-escapes, and with no
// slash at the end.
func cleanPath(path string) string {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if p, err := url.PathUnescape(path); err == nil {
		path = p
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		path = "/"
	}
	return path
}

// isPageLink reports an address on this site: "/المهام", not "//cdn…" or "https://…".
func isPageLink(href string) bool {
	return strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//")
}

func (in *Interp) hasPage(href string) bool {
	p, _ := in.MatchPage(href)
	return p != nil
}

func samePath(a, b string) bool { return cleanPath(a) == cleanPath(b) }

// firstHeading is the text of the first عنوان on a page, for its title.
func firstHeading(n *Node) string {
	if len(n.Tag) == 2 && n.Tag[0] == 'h' && n.Tag[1] >= '1' && n.Tag[1] <= '6' {
		return n.Text
	}
	for _, k := range n.Kids {
		if t := firstHeading(k); t != "" {
			return t
		}
	}
	return ""
}
