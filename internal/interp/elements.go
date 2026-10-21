package interp

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/token"
)

// The newer interface elements, the options every element shares for its look,
// icons and the app's theme. runtime.js (dad build) follows this file rule for
// rule: the tests render the same pages both ways and compare the HTML.

// ---------------------------------------------------------------- the look of an element

// Colours by name. «أساسي» is the app's own colour (see «سمة»).
var colorNames = map[string]string{
	"أساسي": "var(--lapis)", "باهت": "var(--muted)",
	"أحمر": "#c0392b", "أخضر": "#1e8e3e", "أزرق": "#2563eb", "أصفر": "#d4a106", "برتقالي": "#e67e22",
	"بنفسجي": "#7d3c98", "وردي": "#d63384", "فيروزي": "#0e8f9a", "ذهبي": "#c9a04a", "رمادي": "#6b7280",
	"كحلي": "#1b2b6b", "بني": "#8d5524", "أسود": "#111827", "أبيض": "#ffffff",
}

// ColorNames lists the colour names, for the checker and the docs.
func ColorNames() []string {
	out := make([]string, 0, len(colorNames))
	for k := range colorNames {
		out = append(out, k)
	}
	return out
}

var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// IsColor reports whether a text names a colour: a name above or «#1b2b6b».
func IsColor(s string) bool { return colorNames[s] != "" || hexColor.MatchString(s) }

func (in *Interp) color(v Value, pos token.Pos, name, opt string) string {
	s, _ := v.(string)
	if c := colorNames[s]; c != "" {
		return c
	}
	if hexColor.MatchString(s) {
		return strings.ToLower(s)
	}
	in.fail(pos, "«%s» في «%s» يأخذ اسم لون مثل «أزرق» أو «فيروزي»، أو قيمة مثل «#1b2b6b»", opt, name)
	return ""
}

// choice reads an option that takes one of a few words (OptionValues).
func (in *Interp) choice(v Value, pos token.Pos, name, opt string) string {
	s, _ := v.(string)
	words := OptionWords(name, opt)
	for _, w := range words {
		if s == w {
			return s
		}
	}
	in.fail(pos, "«%s» في «%s» يكون %s", opt, name, quoteAll(words))
	return ""
}

func quoteAll(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = "«" + w + "»"
	}
	return strings.Join(q, " أو ")
}

var classWord = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

var variantClass = map[string]string{"ثانوي": "dad-secondary", "خفيف": "dad-quiet", "خطر": "dad-danger"}
var sizeClass = map[string]string{"صغير": "dad-small", "كبير": "dad-large"}
var textAlign = map[string]string{"بداية": "start", "وسط": "center", "نهاية": "end"}

// look adds the options that change how an element looks, always in the same
// order: classes (style, size, fill, then the program's own «صنف»), then styles.
func (in *Interp) look(n *Node, opts map[string]Value, ci *callInfo, name string) {
	var classes, styles []string
	if v, ok := opts["نمط"]; ok && name != "تنبيه" {
		s := in.choice(v, ci.pos, name, "نمط")
		if name == "رابط" {
			classes = append(classes, "dad-button")
		}
		if c := variantClass[s]; c != "" {
			classes = append(classes, c)
		}
	}
	if v, ok := opts["حجم"]; ok {
		if c := sizeClass[in.choice(v, ci.pos, name, "حجم")]; c != "" {
			classes = append(classes, c)
		}
	}
	if in.flag(opts, "ملء", ci.pos) {
		classes = append(classes, "dad-fill")
	}
	if v, ok := opts["صنف"]; ok {
		s := in.optText(v, ci.pos, name, "صنف")
		for _, w := range strings.Fields(s) {
			if !classWord.MatchString(w) {
				in.fail(ci.pos, "«صنف» يأخذ أسماء من حروف وأرقام و«_» و«-»، بينها مسافات، مثل: \"بطاقة-مميزة\"")
			}
			classes = append(classes, w)
		}
	}
	box := name == "عمود" || name == "صف" || name == "بطاقة"
	if v, ok := opts["مسافة"]; ok && box {
		styles = append(styles, "gap:"+strconv.FormatInt(in.intValue(v, ci.pos, "مسافة"), 10)+"px")
	}
	if v, ok := opts["حشو"]; ok {
		styles = append(styles, "padding:"+strconv.FormatInt(in.intValue(v, ci.pos, "حشو"), 10)+"px")
	}
	if v, ok := opts["عرض"]; ok && box {
		styles = append(styles, "max-width:"+strconv.FormatInt(in.intValue(v, ci.pos, "عرض"), 10)+"px")
	}
	if v, ok := opts["لون"]; ok {
		styles = append(styles, "color:"+in.color(v, ci.pos, name, "لون"))
	}
	if v, ok := opts["خلفية"]; ok {
		// a tint of the colour over the page, so text stays readable in both themes
		styles = append(styles, "background:color-mix(in srgb, "+in.color(v, ci.pos, name, "خلفية")+" 16%, var(--paper))")
	}
	if v, ok := opts["محاذاة"]; ok && !box {
		styles = append(styles, "text-align:"+textAlign[in.choice(v, ci.pos, name, "محاذاة")])
	}
	if len(classes) > 0 {
		joined := strings.Join(classes, " ")
		found := false
		for i, a := range n.Attrs {
			if a[0] == "class" {
				n.Attrs[i][1] += " " + joined
				found = true
			}
		}
		if !found {
			n.attr("class", joined)
		}
	}
	if len(styles) > 0 {
		n.attr("style", strings.Join(styles, ";"))
	}
}

// withIcon gives a button or link its text, after an icon when «أيقونة» is given.
func (in *Interp) withIcon(n *Node, opts map[string]Value, ci *callInfo, name, text string) {
	v, ok := opts["أيقونة"]
	if !ok {
		n.Text = text
		return
	}
	n.Kids = append(n.Kids, in.icon(v, ci.pos, name, ""), &Node{Tag: "span", Text: text})
}

// icon draws one of Icons as an inline <svg>. Without a description it is
// decoration, hidden from screen readers.
func (in *Interp) icon(v Value, pos token.Pos, name, desc string) *Node {
	s, _ := v.(string)
	drawing, ok := Icons[s]
	if !ok {
		in.fail(pos, "لا توجد أيقونة باسم «%s» في «%s»؛ انظر قائمة الأيقونات في دليل الواجهة", s, name)
	}
	n := &Node{Tag: "svg", Raw: drawing}
	n.attr("class", "dad-icon")
	n.attr("viewBox", "0 0 24 24")
	if desc == "" {
		n.attr("aria-hidden", "true")
	} else {
		n.attr("role", "img")
		n.attr("aria-label", desc)
	}
	return n
}

// limit is the «أدنى» or «أقصى» of a field: a number, or a date or time as text.
func (in *Interp) limit(v Value, pos token.Pos, opt string) string {
	switch x := v.(type) {
	case int64, float64, string:
		return textOf(x)
	}
	in.fail(pos, "«%s» يأخذ عدداً أو تاريخاً مثل \"2026-01-31\"", opt)
	return ""
}

// label is how a value is shown as a choice or in a list: text as it is, a case
// of a تعداد by its name, a record by its first text field.
func label(v Value) string {
	switch x := v.(type) {
	case string:
		return x
	case *EnumVal:
		if len(x.Vals) == 0 {
			return x.Variant.Name
		}
	case *Struct:
		for _, f := range x.Vals {
			if s, ok := f.(string); ok {
				return s
			}
		}
	case nil:
		return ""
	}
	return Format(v)
}

// cell is how a value is shown in a table: ✓ for صح, nothing for خطأ and عدم.
func cell(v Value) string {
	switch x := v.(type) {
	case bool:
		if x {
			return "✓"
		}
		return ""
	case nil:
		return ""
	}
	return label(v)
}

func choiceIndex(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), x == float64(int(x))
	case string:
		i, err := strconv.Atoi(x)
		return i, err == nil
	case int:
		return x, true
	}
	return 0, false
}

// anonState is a state cell an element keeps for itself, such as the open tab,
// in the order elements are laid out, like «حالة».
func (in *Interp) anonState(pos token.Pos, name string, init Value) *binding {
	ctx := in.ui
	if len(ctx.frames) < 2 {
		in.fail(pos, "«%s» يُستخدم داخل مكون أو صفحة", name)
	}
	f := ctx.frames[len(ctx.frames)-1]
	cells := ctx.sess.states[f.key]
	if f.state < len(cells) {
		b := cells[f.state]
		f.state++
		return b
	}
	b := &binding{val: init, mutable: true, state: true}
	ctx.sess.states[f.key] = append(cells, b)
	f.state++
	return b
}

func (in *Interp) listArg(v Value, pos token.Pos, name string) []Value {
	l, ok := textOrRef(v).(*List)
	if !ok {
		in.fail(pos, "«%s» يأخذ قائمة، لكن وجدت «%s»", name, typeName(textOrRef(v)))
	}
	return l.Elems
}

// ---------------------------------------------------------------- the newer elements

type tabSpec struct {
	title string
	icon  Value
	fn    Value
	opts  map[string]Value
	ci    *callInfo
}

func (in *Interp) installMoreElements(def func(string, bool, func(*Interp, *callInfo) Value)) {
	// قائمة_منسدلة(اللون، ["أحمر"، "أخضر"]) and اختيار(…): one of a list of values.
	choose := func(name string, radios bool) {
		def(name, true, func(in *Interp, ci *callInfo) Value {
			opts := in.uiOptions(ci, name)
			if len(ci.args) < 2 {
				in.fail(ci.pos, "«%s» يأخذ القيمة المختارة وقائمة الخيارات، مثل: %s(اللون، [\"أحمر\"، \"أخضر\"])", name, name)
			}
			options := in.listArg(ci.args[1], ci.pos, name)
			h := uiHandler{kind: "choose", fn: in.lambdaAfter(ci, 2, name), options: options}
			current := textOrRef(ci.args[0])
			if r, ok := ci.args[0].(*StateRef); ok {
				h.cell = r.b
			}
			disabled := in.flag(opts, "معطل", ci.pos)
			id := ""
			if h.cell != nil || h.fn != nil {
				id = in.handler(h)
			}
			selected := -1
			for i, o := range options {
				if equal(o, current) {
					selected = i
					break
				}
			}
			if !radios {
				n := &Node{Tag: "select"}
				if id != "" {
					n.attr("data-choose", id)
				}
				if disabled {
					n.attr("disabled", "")
				}
				in.look(n, opts, ci, name)
				if p, ok := opts["تلميح"]; ok || selected < 0 {
					o := &Node{Tag: "option", Text: ""}
					if ok {
						o.Text = in.optText(p, ci.pos, name, "تلميح")
					}
					o.attr("value", "")
					o.attr("disabled", "")
					if selected < 0 {
						o.attr("selected", "")
					}
					n.Kids = append(n.Kids, o)
				}
				for i, v := range options {
					o := &Node{Tag: "option", Text: label(v)}
					o.attr("value", strconv.Itoa(i))
					if i == selected {
						o.attr("selected", "")
					}
					n.Kids = append(n.Kids, o)
				}
				in.emit(n)
				return nil
			}
			group := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-choices"}}}
			group.attr("role", "radiogroup")
			in.look(group, opts, ci, name)
			for i, v := range options {
				box := &Node{Tag: "input"}
				box.attr("type", "radio")
				box.attr("name", "r"+strconv.Itoa(in.ui.nextID+1)+"-"+id)
				box.attr("value", strconv.Itoa(i))
				if i == selected {
					box.attr("checked", "")
				}
				if disabled {
					box.attr("disabled", "")
				}
				if id != "" {
					box.attr("data-choose", id)
				}
				group.Kids = append(group.Kids, &Node{Tag: "label", Attrs: [][2]string{{"class", "dad-check"}}, Kids: []*Node{box, {Tag: "span", Text: label(v)}}})
			}
			in.emit(group)
			return nil
		})
	}
	choose("قائمة_منسدلة", false)
	choose("اختيار", true)

	def("منطقة_نص", true, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "منطقة_نص")
		if len(ci.args) == 0 {
			in.fail(ci.pos, "«منطقة_نص» يأخذ متغير حالة، مثل: منطقة_نص(الملاحظات)")
		}
		h := uiHandler{kind: "input", fn: in.lambdaAfter(ci, 1, "منطقة_نص")}
		if r, ok := ci.args[0].(*StateRef); ok {
			h.cell = r.b
		}
		text := textOf(ci.args[0])
		if strings.HasPrefix(text, "\n") {
			text = "\n" + text // a browser drops the first line break inside <textarea>
		}
		n := &Node{Tag: "textarea", Text: text}
		rows := int64(4)
		if v, ok := opts["أسطر"]; ok {
			rows = min(max(in.intValue(v, ci.pos, "أسطر"), 1), 40)
		}
		n.attr("rows", strconv.FormatInt(rows, 10))
		if p, ok := opts["تلميح"]; ok {
			n.attr("placeholder", in.optText(p, ci.pos, "منطقة_نص", "تلميح"))
		}
		in.look(n, opts, ci, "منطقة_نص")
		if in.flag(opts, "معطل", ci.pos) {
			n.attr("disabled", "")
		}
		if h.cell != nil || h.fn != nil {
			n.attr("data-input", in.handler(h))
		}
		in.emit(n)
		return nil
	})

	empty := func(opts map[string]Value, ci *callInfo, name string) {
		msg := "لا شيء لعرضه بعد."
		if v, ok := opts["فارغ"]; ok {
			msg = in.optText(v, ci.pos, name, "فارغ")
		}
		in.emit(&Node{Tag: "p", Attrs: [][2]string{{"class", "dad-empty"}}, Text: msg})
	}

	// جدول_عرض(المهام، أعمدة: ["العنوان"]) { م => … }: records as a table.
	def("جدول_عرض", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "جدول_عرض")
		if len(ci.args) == 0 {
			in.fail(ci.pos, "«جدول_عرض» يأخذ قائمة، مثل: جدول_عرض(مهمة.الكل())")
		}
		rows := in.listArg(ci.args[0], ci.pos, "جدول_عرض")
		fn := in.lambdaAfter(ci, 1, "جدول_عرض")
		if len(rows) == 0 {
			empty(opts, ci, "جدول_عرض")
			return nil
		}
		var cols []string
		if v, ok := opts["أعمدة"]; ok {
			for _, c := range in.listArg(v, ci.pos, "أعمدة") {
				s, ok := c.(string)
				if !ok {
					in.fail(ci.pos, "«أعمدة» قائمة بأسماء الحقول، مثل: [\"العنوان\"، \"منجزة\"]")
				}
				cols = append(cols, s)
			}
		} else {
			cols = columnsOf(rows[0])
		}
		wrap := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-table"}}}
		in.look(wrap, opts, ci, "جدول_عرض")
		table := &Node{Tag: "table"}
		head := &Node{Tag: "tr"}
		for _, c := range cols {
			head.Kids = append(head.Kids, &Node{Tag: "th", Text: strings.ReplaceAll(c, "_", " ")})
		}
		if fn != nil {
			head.Kids = append(head.Kids, &Node{Tag: "th"})
		}
		body := &Node{Tag: "tbody"}
		table.Kids = []*Node{{Tag: "thead", Kids: []*Node{head}}, body}
		wrap.Kids = []*Node{table}
		in.emit(wrap)
		for _, r := range rows {
			tr := &Node{Tag: "tr"}
			body.Kids = append(body.Kids, tr)
			for _, c := range cols {
				tr.Kids = append(tr.Kids, &Node{Tag: "td", Text: cell(in.fieldOf(r, c, ci.pos))})
			}
			if fn != nil {
				td := &Node{Tag: "td", Attrs: [][2]string{{"class", "dad-actions"}}}
				tr.Kids = append(tr.Kids, td)
				in.ui.stack = append(in.ui.stack, td)
				in.callValue(fn, &callInfo{pos: ci.pos, name: "جدول_عرض", args: []Value{r}})
				in.ui.stack = in.ui.stack[:len(in.ui.stack)-1]
			}
		}
		return nil
	})

	// قائمة_عناصر(القائمة) { ع => … }: a bulleted or numbered list.
	def("قائمة_عناصر", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "قائمة_عناصر")
		if len(ci.args) == 0 {
			in.fail(ci.pos, "«قائمة_عناصر» يأخذ قائمة، مثل: قائمة_عناصر([\"أ\"، \"ب\"])")
		}
		items := in.listArg(ci.args[0], ci.pos, "قائمة_عناصر")
		fn := in.lambdaAfter(ci, 1, "قائمة_عناصر")
		if len(items) == 0 {
			empty(opts, ci, "قائمة_عناصر")
			return nil
		}
		n := &Node{Tag: "ul", Attrs: [][2]string{{"class", "dad-list"}}}
		if in.flag(opts, "مرقمة", ci.pos) {
			n.Tag = "ol"
		}
		in.look(n, opts, ci, "قائمة_عناصر")
		in.emit(n)
		for _, it := range items {
			li := &Node{Tag: "li"}
			n.Kids = append(n.Kids, li)
			if fn == nil {
				li.Text = label(it)
				continue
			}
			in.ui.stack = append(in.ui.stack, li)
			in.callValue(fn, &callInfo{pos: ci.pos, name: "قائمة_عناصر", args: []Value{it}})
			in.ui.stack = in.ui.stack[:len(in.ui.stack)-1]
		}
		return nil
	})

	// ألسنة { لسان("الأول") { … } … }: tabs; the open one is kept like a «حالة»,
	// or in a state given as the first argument (its index or its title).
	def("ألسنة", true, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "ألسنة")
		var fn Value
		var cellRef *binding
		switch len(ci.args) {
		case 1:
			fn = in.lambdaAfter(ci, 0, "ألسنة")
		case 2:
			r, ok := ci.args[0].(*StateRef)
			if !ok {
				in.fail(ci.pos, "«ألسنة» يأخذ متغير حالة للسان المفتوح، مثل: ألسنة(اللسان) { … }")
			}
			cellRef = r.b
			fn = in.lambdaAfter(ci, 1, "ألسنة")
		}
		if fn == nil {
			in.fail(ci.pos, "«ألسنة» يأخذ ألسنتها بين { }، مثل: ألسنة { لسان(\"الأول\") { … } }")
		}
		if cellRef == nil {
			cellRef = in.anonState(ci.pos, "ألسنة", int64(0))
		}
		// Collect the tabs: their blocks run only for the tab that is open.
		var tabs []tabSpec
		saved, savedStack := in.ui.tabs, in.ui.stack
		scratch := &Node{Tag: "div"}
		in.ui.tabs, in.ui.stack = &tabs, append(append([]*Node{}, in.ui.stack...), scratch)
		func() {
			defer func() { in.ui.tabs, in.ui.stack = saved, savedStack }()
			in.callValue(fn, &callInfo{pos: ci.pos, name: "ألسنة"})
		}()
		if len(scratch.Kids) > 0 {
			in.fail(ci.pos, "داخل «ألسنة» تُكتب الألسنة فقط، مثل: لسان(\"الأول\") { … }")
		}
		open := 0
		switch v := cellRef.val.(type) {
		case int64:
			open = int(v)
		case string:
			for i, t := range tabs {
				if t.title == v {
					open = i
				}
			}
		}
		if open < 0 || open >= len(tabs) {
			open = 0
		}
		n := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-tabs"}}}
		in.look(n, opts, ci, "ألسنة")
		list := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-tablist"}, {"role", "tablist"}}}
		panel := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-tabpanel"}, {"role", "tabpanel"}}}
		n.Kids = []*Node{list, panel}
		in.emit(n)
		for i, t := range tabs {
			b := &Node{Tag: "button"}
			in.withIcon(b, t.opts, t.ci, "لسان", t.title)
			b.attr("type", "button")
			b.attr("role", "tab")
			b.attr("aria-selected", strconv.FormatBool(i == open))
			var value Value = int64(i)
			if _, isText := cellRef.val.(string); isText {
				value = t.title
			}
			b.attr("data-click", in.handler(uiHandler{kind: "set", cell: cellRef, value: value}))
			in.look(b, t.opts, t.ci, "لسان")
			list.Kids = append(list.Kids, b)
		}
		if len(tabs) > 0 {
			in.ui.stack = append(in.ui.stack, panel)
			defer func() { in.ui.stack = in.ui.stack[:len(in.ui.stack)-1] }()
			in.callValue(tabs[open].fn, &callInfo{pos: tabs[open].ci.pos, name: "لسان"})
		}
		return nil
	})
	def("لسان", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "لسان")
		if in.ui.tabs == nil {
			in.fail(ci.pos, "«لسان» يُستخدم داخل «ألسنة»")
		}
		title := in.textArg(ci, "لسان")
		fn := in.lambdaAfter(ci, 1, "لسان")
		if fn == nil {
			in.fail(ci.pos, "«لسان» يأخذ عنوانه وما يعرضه بين { }، مثل: لسان(\"الأول\") { … }")
		}
		if _, ok := opts["أيقونة"]; ok {
			in.icon(opts["أيقونة"], ci.pos, "لسان", "") // checked now, drawn with the tab
		}
		*in.ui.tabs = append(*in.ui.tabs, tabSpec{title: title, fn: fn, opts: opts, ci: ci})
		return nil
	})

	// نافذة(مفتوحة، عنوان: "…") { … }: a dialog over the page while مفتوحة is صح.
	def("نافذة", true, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "نافذة")
		if len(ci.args) == 0 {
			in.fail(ci.pos, "«نافذة» يأخذ حالة منطقية تفتحها وتغلقها، مثل: نافذة(مفتوحة) { … }")
		}
		var cellRef *binding
		openV := ci.args[0]
		if r, ok := ci.args[0].(*StateRef); ok {
			cellRef, openV = r.b, r.b.val
		}
		isOpen, ok := openV.(bool)
		if !ok {
			in.fail(ci.pos, "«نافذة» تأخذ قيمة منطقية (صح أو خطأ)، لكن وجدت «%s»", typeName(openV))
		}
		fn := in.lambdaAfter(ci, 1, "نافذة")
		if !isOpen {
			return nil
		}
		title := ""
		if v, ok := opts["عنوان"]; ok {
			title = in.optText(v, ci.pos, "نافذة", "عنوان")
		}
		backdrop := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-backdrop"}}}
		box := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-dialog"}, {"role", "dialog"}, {"aria-modal", "true"}, {"tabindex", "-1"}}}
		if title != "" {
			box.attr("aria-label", title)
		}
		in.look(box, opts, ci, "نافذة")
		head := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-dialog-head"}}}
		if title != "" {
			head.Kids = append(head.Kids, &Node{Tag: "h2", Text: title})
		}
		if cellRef != nil {
			x := &Node{Tag: "button", Attrs: [][2]string{{"type", "button"}, {"class", "dad-close"}, {"aria-label", "إغلاق"}}}
			x.Kids = []*Node{in.icon("إغلاق", ci.pos, "نافذة", "")}
			x.attr("data-click", in.handler(uiHandler{kind: "set", cell: cellRef, value: false}))
			head.Kids = append(head.Kids, x)
		}
		body := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-col"}}}
		box.Kids = []*Node{head, body}
		backdrop.Kids = []*Node{box}
		in.emit(backdrop)
		if fn != nil {
			in.ui.stack = append(in.ui.stack, body)
			defer func() { in.ui.stack = in.ui.stack[:len(in.ui.stack)-1] }()
			in.callValue(fn, &callInfo{pos: ci.pos, name: "نافذة"})
		}
		return nil
	})

	// تنبيه("تم الحفظ"، نوع: "نجاح"): a message that stands out.
	def("تنبيه", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "تنبيه")
		kind := "معلومة"
		if v, ok := opts["نمط"]; ok {
			kind = in.choice(v, ci.pos, "تنبيه", "نمط")
		}
		class := map[string]string{"معلومة": "info", "نجاح": "success", "تحذير": "warning", "خطأ": "error"}[kind]
		icon := map[string]string{"معلومة": "معلومة", "نجاح": "نجاح", "تحذير": "تحذير", "خطأ": "خطر"}[kind]
		role := "status"
		if kind == "خطأ" || kind == "تحذير" {
			role = "alert"
		}
		n := &Node{Tag: "div", Attrs: [][2]string{{"class", "dad-alert dad-alert-" + class}, {"role", role}}}
		in.look(n, opts, ci, "تنبيه")
		n.Kids = []*Node{in.icon(icon, ci.pos, "تنبيه", ""), {Tag: "span", Text: in.textArg(ci, "تنبيه")}}
		in.emit(n)
		return nil
	})

	def("أيقونة", false, func(in *Interp, ci *callInfo) Value {
		opts := in.uiOptions(ci, "أيقونة")
		if len(ci.args) != 1 {
			in.fail(ci.pos, "«أيقونة» تأخذ اسم الأيقونة، مثل: أيقونة(\"بحث\")")
		}
		desc := ""
		if v, ok := opts["وصف"]; ok {
			desc = in.optText(v, ci.pos, "أيقونة", "وصف")
		}
		n := in.icon(ci.args[0], ci.pos, "أيقونة", desc)
		in.look(n, opts, ci, "أيقونة")
		in.emit(n)
		return nil
	})
}

// columnsOf are the columns of a table of rows like r: the fields of a record
// (without the رقم of a جدول), the keys of a map, or the value itself.
func columnsOf(r Value) []string {
	switch x := r.(type) {
	case *Struct:
		var out []string
		for i, f := range x.Type.Fields {
			if x.Type.Table && i == len(x.Type.Fields)-1 {
				break // رقم
			}
			out = append(out, f.Name)
		}
		return out
	case *Map:
		var out []string
		for _, k := range x.Keys {
			out = append(out, textOf(k))
		}
		return out
	}
	return []string{""}
}

// fieldOf is the value in a row under a column.
func (in *Interp) fieldOf(r Value, col string, pos token.Pos) Value {
	switch x := r.(type) {
	case *Struct:
		for i, f := range x.Type.Fields {
			if f.Name == col {
				return x.Vals[i]
			}
		}
		in.fail(pos, "«جدول_عرض»: لا يوجد حقل باسم «%s» في «%s»", col, x.Type.Name)
	case *Map:
		for i, k := range x.Keys {
			if textOf(k) == col {
				return x.Vals[i]
			}
		}
		return nil
	}
	if col == "" {
		return r
	}
	in.fail(pos, "«جدول_عرض»: لا يوجد حقل باسم «%s» في قيمة من النوع «%s»", col, typeName(r))
	return nil
}

// ---------------------------------------------------------------- the theme

// Theme is what «سمة» set: CSS variables for the whole app, and light or dark.
type Theme struct {
	Vars []string // "--theme-primary:#0e8f9a", …
	Mode string   // "", "light" or "dark"
}

// ThemeOptions are the named arguments of «سمة».
var ThemeOptions = []string{"اللون", "الخط", "الزوايا", "العرض", "الحجم", "الوضع"}

// ThemeValues are the words some options of «سمة» take.
var ThemeValues = map[string][]string{"الخط": {"كوفي", "نسخ", "النظام"}, "الوضع": {"تلقائي", "فاتح", "داكن"}}

func (in *Interp) setTheme(ci *callInfo) Value {
	if in.ui != nil || in.event != nil {
		in.fail(ci.pos, "«سمة» تُكتب في أعلى البرنامج، لا داخل مكون أو حدث")
	}
	if len(ci.args) > 0 {
		in.fail(ci.pos, "«سمة» تأخذ معاملات مسماة، مثل: سمة(اللون: \"فيروزي\")")
	}
	t := &Theme{}
	px := func(v Value, opt string, lo, hi int64) string {
		n := in.intValue(v, ci.pos, opt)
		if n < lo || n > hi {
			in.fail(ci.pos, "«%s» في «سمة» بين %d و%d", opt, lo, hi)
		}
		return strconv.FormatInt(n, 10) + "px"
	}
	word := func(v Value, opt string) string {
		s, _ := v.(string)
		for _, w := range ThemeValues[opt] {
			if s == w {
				return s
			}
		}
		in.fail(ci.pos, "«%s» في «سمة» يكون %s", opt, quoteAll(ThemeValues[opt]))
		return ""
	}
	for _, na := range ci.named {
		switch na.name {
		case "اللون":
			c := in.color(na.val, ci.pos, "سمة", "اللون")
			if !strings.HasPrefix(c, "#") {
				in.fail(ci.pos, "«اللون» في «سمة» يأخذ لوناً بعينه، مثل «فيروزي» أو «#0e8f9a»")
			}
			t.Vars = append(t.Vars, "--theme-primary:"+c, "--theme-on-primary:"+onColor(c))
		case "الخط":
			t.Vars = append(t.Vars, "--theme-font:var(--font-"+map[string]string{"كوفي": "kufi", "نسخ": "naskh", "النظام": "system"}[word(na.val, "الخط")]+")")
		case "الزوايا":
			t.Vars = append(t.Vars, "--theme-radius:"+px(na.val, "الزوايا", 0, 40))
		case "العرض":
			t.Vars = append(t.Vars, "--theme-width:"+px(na.val, "العرض", 320, 2000))
		case "الحجم":
			t.Vars = append(t.Vars, "--theme-size:"+px(na.val, "الحجم", 12, 28))
		case "الوضع":
			t.Mode = map[string]string{"تلقائي": "", "فاتح": "light", "داكن": "dark"}[word(na.val, "الوضع")]
		default:
			in.fail(ci.pos, "«سمة» لا تأخذ «%s»؛ تأخذ: %s", na.name, strings.Join(ThemeOptions, "، "))
		}
	}
	in.theme = t
	return nil
}

// onColor is the text colour that reads on a background colour.
func onColor(hex string) string {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	r, _ := strconv.ParseUint(h[0:2], 16, 8)
	g, _ := strconv.ParseUint(h[2:4], 16, 8)
	b, _ := strconv.ParseUint(h[4:6], 16, 8)
	if 299*r+587*g+114*b > 150000 {
		return "#111827"
	}
	return "#ffffff"
}

// themed puts the theme on a rendered page: its variables in a <style> first,
// and data-mode for light or dark.
func (in *Interp) themed(root *Node) {
	t := in.theme
	if t == nil {
		return
	}
	if t.Mode != "" {
		root.attr("data-mode", t.Mode)
	}
	if len(t.Vars) > 0 {
		root.Kids = append([]*Node{{Tag: "style", Raw: ":root{" + strings.Join(t.Vars, ";") + "}"}}, root.Kids...)
	}
}
