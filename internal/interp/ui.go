package interp

import (
	"errors"
	"html"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

// The interface layer: components (مكون) with state (حالة) that lay out
// elements (عمود، صف، زر، حقل…). The interpreter renders a component to a tree
// of Nodes and HTML; a page in the browser sends clicks and typing back as
// events, the handlers run here, and the page is rendered again — the way
// Phoenix LiveView and Blazor Server work. A later version compiles components
// to JavaScript instead; programs stay the same.

// Component is a declared مكون.
type Component struct {
	Name   string
	Params []*ast.Param
	Body   *ast.Block
	Env    *Env
	Pos    token.Pos
}

// StateRef is passed to an element in place of a state variable's value, so
// the element can change it: «حقل(جديدة)» edits «جديدة».
type StateRef struct {
	Name string
	b    *binding
}

// Node is an element of a rendered page.
type Node struct {
	Tag   string
	Attrs [][2]string
	Text  string
	Raw   string // markup written as it is, after Text: the drawing inside an icon's <svg>
	Kids  []*Node
}

func (n *Node) attr(k, v string) { n.Attrs = append(n.Attrs, [2]string{k, v}) }

// UISession is one open page: the state of its components and the handlers of
// its last render.
type UISession struct {
	Version  int    // increases with every render
	Path     string // the address of the page shown, such as "/مهمة/3"
	Title    string // the page's first heading, for the browser tab
	NotFound bool   // no page has this address
	Moved    bool   // an event moved to another page («انتقل»); the browser should show Path
	Away     string // an event asked to leave for another site («انتقل("https://…")»)
	states   map[string][]*binding
	handlers map[string]uiHandler
	page     *Page
	params   map[string]string
}

// NewUISession starts the state of a new page.
func NewUISession() *UISession {
	return &UISession{states: map[string][]*binding{}, handlers: map[string]uiHandler{}}
}

type uiHandler struct {
	kind    string // "click", "input", "toggle", "choose" (one of options) or "set" (cell = value)
	fn      Value  // the lambda to run, if any
	cell    *binding
	options []Value // choose: what each choice stands for
	value   Value   // set: the value to put in cell
}

// renderCtx is the work in progress of one render.
type renderCtx struct {
	sess     *UISession
	path     string  // the address shown: a link to it is the current page
	stack    []*Node // the open containers; elements go into the last one
	frames   []*frame
	handlers map[string]uiHandler
	nextID   int
	seen     map[string]bool // component instances in this render
	tabs     *[]tabSpec      // inside «ألسنة»: the tabs found so far
}

// frame is one component instance being rendered.
type frame struct {
	key    string
	state  int            // the next «حالة» of this instance
	counts map[string]int // how many of each child component so far
}

// ErrStale means the page sent an event for a render that has been replaced.
var ErrStale = errors.New("stale page")

// Components lists the declared components, in source order.
func (in *Interp) Components() []*Component { return in.components }

// RootComponent is the component a page shows: one named الرئيسية or التطبيق,
// otherwise the last declared component that takes no arguments.
func (in *Interp) RootComponent() *Component {
	var last *Component
	for _, c := range in.components[in.mainComps:] {
		if c.Name == "الرئيسية" || c.Name == "التطبيق" || c.Name == "تطبيق" {
			return c
		}
		if !needsArgs(c.Params) {
			last = c
		}
	}
	return last
}

func needsArgs(params []*ast.Param) bool {
	for _, p := range params {
		if p.Default == nil && (p.Type == nil || !p.Type.Optional) {
			return true
		}
	}
	return false
}

// Render renders a component for a page and returns its HTML.
func (in *Interp) Render(sess *UISession, c *Component) (string, error) {
	return in.render(sess, func() { in.callValue(c, &callInfo{pos: c.Pos, name: c.Name}) })
}

// render lays out a page with draw and returns its HTML.
func (in *Interp) render(sess *UISession, draw func()) (string, error) {
	root := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-app"}}}
	ctx := &renderCtx{sess: sess, path: sess.Path, stack: []*Node{root}, handlers: map[string]uiHandler{}, seen: map[string]bool{}}
	ctx.frames = []*frame{{key: "", counts: map[string]int{}}}
	err := in.safely(func() {
		in.ui = ctx
		defer func() { in.ui = nil }()
		draw()
	})
	if err != nil {
		return "", err
	}
	in.themed(root)
	sess.Title = firstHeading(root)
	for k := range sess.states {
		if !ctx.seen[k] {
			delete(sess.states, k) // a component that is no longer shown forgets its state
		}
	}
	sess.handlers = ctx.handlers
	sess.Version++
	var b strings.Builder
	writeHTML(&b, root)
	return b.String(), nil
}

// Event runs what the page asked for: a click, typing in a field, or a check
// box. version is the render the page was showing.
func (in *Interp) Event(sess *UISession, version int, id string, value any) error {
	if version != sess.Version {
		return ErrStale
	}
	h, ok := sess.handlers[id]
	if !ok {
		return ErrStale
	}
	sess.Moved, sess.Away = false, ""
	in.event = sess
	defer func() { in.event = nil }()
	return in.safely(func() {
		switch h.kind {
		case "click":
			if h.fn != nil {
				in.callValue(h.fn, &callInfo{name: "زر"})
			}
		case "input":
			s, _ := value.(string)
			var v Value = s
			if h.cell != nil {
				v = fieldValue(h.cell.val, s)
				h.cell.val = v
			}
			if h.fn != nil {
				in.callValue(h.fn, &callInfo{name: "حقل", args: []Value{v}})
			}
		case "toggle":
			checked, _ := value.(bool)
			if h.cell != nil {
				h.cell.val = checked
			}
			if h.fn != nil {
				in.callValue(h.fn, &callInfo{name: "مربع_اختيار", args: []Value{checked}})
			}
		case "choose":
			i, ok := choiceIndex(value)
			if !ok || i < 0 || i >= len(h.options) {
				return
			}
			v := h.options[i]
			if h.cell != nil {
				h.cell.val = v
			}
			if h.fn != nil {
				in.callValue(h.fn, &callInfo{name: "اختيار", args: []Value{v}})
			}
		case "set":
			h.cell.val = h.value
		}
	})
}

// fieldValue keeps a number field a number while it holds a number.
func fieldValue(old Value, s string) Value {
	switch old.(type) {
	case int64:
		if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			return n
		}
	case float64:
		if f, ok := parseNumber(s); ok {
			return f
		}
	}
	return s
}

// ---------------------------------------------------------------- components and state

func (in *Interp) renderComponent(c *Component, ci *callInfo) Value {
	ctx := in.ui
	if ctx == nil {
		in.fail(ci.pos, "المكون «%s» يُعرض في صفحة ويب؛ شغّل البرنامج بـ«ضاد تطوير» وافتح الصفحة", c.Name)
	}
	parent := ctx.frames[len(ctx.frames)-1]
	n := parent.counts[c.Name]
	parent.counts[c.Name]++
	key := parent.key + "/" + c.Name + "#" + strconv.Itoa(n)
	ctx.seen[key] = true
	env := newEnv(c.Env)
	in.bindArgs(c.Params, ci, env, "المكون «"+c.Name+"»")
	ctx.frames = append(ctx.frames, &frame{key: key, counts: map[string]int{}})
	defer func() { ctx.frames = ctx.frames[:len(ctx.frames)-1] }()
	in.runBody(&Func{Name: c.Name, Body: c.Body, Env: env}, env)
	return nil
}

// declareState runs «حالة اسم = قيمة»: the first render sets the value, later
// renders keep it.
func (in *Interp) declareState(s *ast.StateDecl, env *Env) {
	ctx := in.ui
	if ctx == nil || len(ctx.frames) < 2 {
		in.fail(s.Pos, "«حالة» تُستخدم داخل مكون فقط؛ خارجه استخدم «متغير»")
	}
	f := ctx.frames[len(ctx.frames)-1]
	cells := ctx.sess.states[f.key]
	var b *binding
	if f.state < len(cells) {
		b = cells[f.state]
	} else {
		v := in.eval(s.Value, env)
		if s.Type != nil {
			v = in.convert(v, s.Type, s.Pos, "الحالة «"+s.Name+"»")
		}
		b = &binding{val: v, mutable: true, state: true}
		ctx.sess.states[f.key] = append(cells, b)
	}
	f.state++
	if _, exists := env.vars[s.Name]; exists {
		in.fail(s.Pos, "الاسم «%s» معرّف مسبقاً في هذا النطاق", s.Name)
	}
	env.vars[s.Name] = b
}

// ---------------------------------------------------------------- elements

// Elements are the names of the interface elements, for editors and docs.
var Elements = []string{"عمود", "صف", "بطاقة", "عنوان", "فقرة", "زر", "حقل", "مربع_اختيار", "صورة", "رابط", "فاصل",
	"قائمة_منسدلة", "اختيار", "منطقة_نص", "جدول_عرض", "قائمة_عناصر", "ألسنة", "لسان", "نافذة", "تنبيه", "أيقونة"}

// ElementOptions are the named arguments each element takes. Every element also
// takes «صنف», a class for a program's own stylesheet.
var ElementOptions = map[string][]string{
	"عمود": boxOptions, "صف": boxOptions, "بطاقة": boxOptions,
	"عنوان":        {"مستوى", "لون", "محاذاة", "صنف"},
	"فقرة":         {"لون", "محاذاة", "حجم", "صنف"},
	"زر":           {"معطل", "نمط", "حجم", "ملء", "أيقونة", "صنف"},
	"حقل":          {"تلميح", "صيغة", "معطل", "أدنى", "أقصى", "صنف"},
	"مربع_اختيار":  {"نص", "معطل", "صنف"},
	"صورة":         {"وصف", "عرض", "صنف"},
	"رابط":         {"نمط", "حجم", "أيقونة", "صنف"},
	"فاصل":         {"صنف"},
	"قائمة_منسدلة": {"تلميح", "معطل", "صنف"},
	"اختيار":       {"معطل", "صنف"},
	"منطقة_نص":     {"تلميح", "أسطر", "معطل", "صنف"},
	"جدول_عرض":     {"أعمدة", "فارغ", "صنف"},
	"قائمة_عناصر":  {"مرقمة", "فارغ", "صنف"},
	"ألسنة":        {"صنف"},
	"لسان":         {"أيقونة", "صنف"},
	"نافذة":        {"عنوان", "صنف"},
	"تنبيه":        {"نمط", "صنف"},
	"أيقونة":       {"وصف", "حجم", "لون", "صنف"},
}

var boxOptions = []string{"محاذاة", "مسافة", "حشو", "خلفية", "عرض", "صنف"}

// OptionValues are the words some options take, for the checker and the errors.
var OptionValues = map[string][]string{
	"نمط":    {"أساسي", "ثانوي", "خفيف", "خطر"},
	"حجم":    {"صغير", "عادي", "كبير"},
	"محاذاة": {"بداية", "وسط", "نهاية"},
	"صيغة":   {"نص", "رقم", "كلمة_سر", "بريد", "تاريخ", "وقت"},
}

// AlertStyles are the words «نمط» takes in «تنبيه».
var AlertStyles = []string{"معلومة", "نجاح", "تحذير", "خطأ"}

// OptionWords are the words an option of an element takes, if it takes words.
func OptionWords(element, opt string) []string {
	if element == "تنبيه" && opt == "نمط" {
		return AlertStyles
	}
	return OptionValues[opt]
}

func (in *Interp) installElements(env *Env) {
	def := func(name string, bind bool, fn func(in *Interp, ci *callInfo) Value) {
		env.define(name, &Builtin{Name: name, Fn: func(in *Interp, ci *callInfo) Value {
			if in.ui == nil {
				in.fail(ci.pos, "«%s» عنصر واجهة، ويُستخدم داخل مكون", name)
			}
			return fn(in, ci)
		}, BindFirst: bind}, false)
	}
	container := func(name, class string) {
		def(name, false, func(in *Interp, ci *callInfo) Value {
			opts := in.uiOptions(ci, name)
			n := &Node{Tag: "div", Attrs: [][2]string{{"class", class}}}
			if a, ok := opts["محاذاة"]; ok {
				n.attr("data-align", in.choice(a, ci.pos, name, "محاذاة"))
			}
			in.look(n, opts, ci, name)
			in.withChildren(n, ci, name)
			return nil
		})
	}
	container("عمود", "dad-col")
	container("صف", "dad-row")
	container("بطاقة", "dad-card")

	def("عنوان", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "عنوان")
		level := int64(1)
		if l, ok := opts["مستوى"]; ok {
			level = min(max(in.intValue(l, ci.pos, "مستوى"), 1), 6)
		}
		n := &Node{Tag: "h" + strconv.FormatInt(level, 10), Text: in.textArg(ci, "عنوان")}
		in.look(n, opts, ci, "عنوان")
		in.emit(n)
		return nil
	})
	def("فقرة", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "فقرة")
		n := &Node{Tag: "p", Text: in.textArg(ci, "فقرة")}
		in.look(n, opts, ci, "فقرة")
		in.emit(n)
		return nil
	})
	def("زر", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "زر")
		n := &Node{Tag: "button"}
		in.withIcon(n, opts, ci, "زر", in.textArg(ci, "زر"))
		n.attr("type", "button")
		in.look(n, opts, ci, "زر")
		if in.flag(opts, "معطل", ci.pos) {
			n.attr("disabled", "")
		}
		if fn := in.lambdaAfter(ci, 1, "زر"); fn != nil {
			n.attr("data-click", in.handler(uiHandler{kind: "click", fn: fn}))
		}
		in.emit(n)
		return nil
	})
	def("حقل", true, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "حقل")
		if len(ci.args) == 0 {
			in.fail(ci.pos, "«حقل» يأخذ متغير حالة، مثل: حقل(اسم، تلميح: \"اسمك\")")
		}
		n := &Node{Tag: "input"}
		kind := "text"
		switch textOrRef(ci.args[0]).(type) {
		case int64, float64:
			kind = "number" // a number state gets a number field
		}
		if t, ok := opts["صيغة"]; ok {
			kind = map[string]string{"نص": "text", "رقم": "number", "كلمة_سر": "password", "بريد": "email", "تاريخ": "date", "وقت": "time"}[in.choice(t, ci.pos, "حقل", "صيغة")]
		}
		n.attr("type", kind)
		h := uiHandler{kind: "input", fn: in.lambdaAfter(ci, 1, "حقل")}
		switch v := ci.args[0].(type) {
		case *StateRef:
			h.cell = v.b
			n.attr("value", textOf(v.b.val))
		default:
			n.attr("value", textOf(v))
		}
		if p, ok := opts["تلميح"]; ok {
			n.attr("placeholder", in.optText(p, ci.pos, "حقل", "تلميح"))
		}
		for _, o := range [][2]string{{"أدنى", "min"}, {"أقصى", "max"}} {
			if v, ok := opts[o[0]]; ok {
				n.attr(o[1], in.limit(v, ci.pos, o[0]))
			}
		}
		in.look(n, opts, ci, "حقل")
		if in.flag(opts, "معطل", ci.pos) {
			n.attr("disabled", "")
		}
		if h.cell != nil || h.fn != nil {
			n.attr("data-input", in.handler(h))
		}
		in.emit(n)
		return nil
	})
	def("مربع_اختيار", true, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "مربع_اختيار")
		if len(ci.args) == 0 {
			in.fail(ci.pos, "«مربع_اختيار» يأخذ قيمة منطقية، مثل: مربع_اختيار(م.منجزة) { … }")
		}
		h := uiHandler{kind: "toggle", fn: in.lambdaAfter(ci, 1, "مربع_اختيار")}
		var checked Value
		if r, ok := ci.args[0].(*StateRef); ok {
			h.cell, checked = r.b, r.b.val
		} else {
			checked = ci.args[0]
		}
		on, ok := checked.(bool)
		if !ok {
			in.fail(ci.pos, "«مربع_اختيار» يأخذ قيمة منطقية (صح أو خطأ)، لكن وجدت «%s»", typeName(checked))
		}
		box := &Node{Tag: "input"}
		box.attr("type", "checkbox")
		if on {
			box.attr("checked", "")
		}
		if in.flag(opts, "معطل", ci.pos) {
			box.attr("disabled", "")
		}
		if h.cell != nil || h.fn != nil {
			box.attr("data-toggle", in.handler(h))
		}
		if t, ok := opts["نص"]; ok {
			label := &Node{Tag: "label", Attrs: [][2]string{{"class", "dad-check"}}, Kids: []*Node{box, {Tag: "span", Text: in.optText(t, ci.pos, "مربع_اختيار", "نص")}}}
			in.look(label, opts, ci, "مربع_اختيار")
			in.emit(label)
		} else {
			in.look(box, opts, ci, "مربع_اختيار")
			in.emit(box)
		}
		return nil
	})
	def("صورة", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "صورة")
		n := &Node{Tag: "img"}
		n.attr("src", in.textArg(ci, "صورة"))
		alt := ""
		if d, ok := opts["وصف"]; ok {
			alt = in.optText(d, ci.pos, "صورة", "وصف")
		}
		n.attr("alt", alt)
		if w, ok := opts["عرض"]; ok {
			n.attr("width", strconv.FormatInt(in.intValue(w, ci.pos, "عرض"), 10))
		}
		in.look(n, opts, ci, "صورة")
		in.emit(n)
		return nil
	})
	def("رابط", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "رابط")
		if len(ci.args) != 2 {
			in.fail(ci.pos, "«رابط» يأخذ النص والعنوان، مثل: رابط(\"الموقع\"، \"https://…\")")
		}
		n := &Node{Tag: "a"}
		href := in.optText(ci.args[1], ci.pos, "رابط", "العنوان")
		n.attr("href", href)
		in.withIcon(n, opts, ci, "رابط", textOf(ci.args[0]))
		// A link to one of the program's pages opens it in place; the link to the
		// page being shown is marked, for navigation bars.
		if isPageLink(href) && in.hasPage(href) {
			n.attr("data-nav", "")
			if samePath(href, in.ui.path) {
				n.attr("aria-current", "page")
			}
		}
		in.look(n, opts, ci, "رابط")
		in.emit(n)
		return nil
	})
	def("فاصل", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "فاصل")
		n := &Node{Tag: "hr"}
		in.look(n, opts, ci, "فاصل")
		in.emit(n)
		return nil
	})
	in.installMoreElements(def)

	// «انتقل("/المهام")» moves to another page after an event.
	env.define("انتقل", &Builtin{Name: "انتقل", Fn: (*Interp).navigate}, false)
	// «سمة(اللون: "فيروزي")» sets the look of the whole app.
	env.define("سمة", &Builtin{Name: "سمة", Fn: (*Interp).setTheme}, false)
}

func (in *Interp) emit(n *Node) {
	top := in.ui.stack[len(in.ui.stack)-1]
	top.Kids = append(top.Kids, n)
}

// withChildren runs the trailing lambda of a container with n as the parent.
func (in *Interp) withChildren(n *Node, ci *callInfo, name string) {
	in.emit(n)
	fn := in.lambdaAfter(ci, 0, name)
	if fn == nil {
		return
	}
	in.ui.stack = append(in.ui.stack, n)
	defer func() { in.ui.stack = in.ui.stack[:len(in.ui.stack)-1] }()
	in.callValue(fn, &callInfo{pos: ci.pos, name: name})
}

// lambdaAfter returns the trailing lambda when there are more than n positional
// arguments, and checks there is nothing else.
func (in *Interp) lambdaAfter(ci *callInfo, n int, name string) Value {
	if len(ci.args) <= n {
		return nil
	}
	if len(ci.args) > n+1 {
		in.fail(ci.pos, "«%s» أُعطي معاملات أكثر مما يأخذ", name)
	}
	switch ci.args[n].(type) {
	case *Func, *Builtin:
		return ci.args[n]
	}
	in.fail(ci.pos, "«%s» يتوقع هنا دالة بين { }، لكن وجدت «%s»", name, typeName(ci.args[n]))
	return nil
}

// uiOptions checks an element's named arguments.
func (in *Interp) uiOptions(ci *callInfo, name string) map[string]Value {
	allowed := ElementOptions[name]
	out := map[string]Value{}
	for _, na := range ci.named {
		ok := false
		for _, a := range allowed {
			ok = ok || a == na.name
		}
		if !ok {
			if len(allowed) == 0 {
				in.fail(ci.pos, "«%s» لا يأخذ معاملات مسماة", name)
			}
			in.fail(ci.pos, "«%s» لا يأخذ «%s»؛ يأخذ: %s", name, na.name, strings.Join(allowed, "، "))
		}
		out[na.name] = na.val
	}
	return out
}

func (in *Interp) textArg(ci *callInfo, name string) string {
	if len(ci.args) == 0 {
		in.fail(ci.pos, "«%s» يحتاج نصاً، مثل: %s(\"…\")", name, name)
	}
	return textOf(ci.args[0])
}

func (in *Interp) optText(v Value, pos token.Pos, name, opt string) string {
	if s, ok := v.(string); ok {
		return s
	}
	in.fail(pos, "«%s» في «%s» يأخذ نصاً", opt, name)
	return ""
}

func (in *Interp) flag(opts map[string]Value, name string, pos token.Pos) bool {
	v, ok := opts[name]
	if !ok {
		return false
	}
	b, isBool := v.(bool)
	if !isBool {
		in.fail(pos, "«%s» يأخذ صح أو خطأ", name)
	}
	return b
}

// textOrRef is the value behind a StateRef.
func textOrRef(v Value) Value {
	if r, ok := v.(*StateRef); ok {
		return r.b.val
	}
	return v
}

func textOf(v Value) string {
	if r, ok := v.(*StateRef); ok {
		v = r.b.val
	}
	if s, ok := v.(string); ok {
		return s
	}
	return Format(v)
}

func (in *Interp) handler(h uiHandler) string {
	in.ui.nextID++
	id := "h" + strconv.Itoa(in.ui.nextID)
	in.ui.handlers[id] = h
	return id
}

// ---------------------------------------------------------------- HTML

var voidTags = map[string]bool{"input": true, "img": true, "hr": true, "br": true}

func writeHTML(b *strings.Builder, n *Node) {
	b.WriteString("<" + n.Tag)
	for _, a := range n.Attrs {
		b.WriteString(" " + a[0])
		if a[1] != "" || a[0] == "value" || a[0] == "alt" {
			b.WriteString(`="` + html.EscapeString(a[1]) + `"`)
		}
	}
	b.WriteString(">")
	if voidTags[n.Tag] {
		return
	}
	b.WriteString(html.EscapeString(n.Text))
	b.WriteString(n.Raw)
	for _, k := range n.Kids {
		writeHTML(b, k)
	}
	b.WriteString("</" + n.Tag + ">")
}
