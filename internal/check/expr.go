package check

import (
	"sort"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/token"
)

// expr checks an expression and returns its type, or nil when unknown.
func (c *checker) expr(e ast.Expr, s *scope) *Type {
	switch e := e.(type) {
	case *ast.IntLit:
		return tInt
	case *ast.FloatLit:
		return tFloat
	case *ast.BoolLit:
		return tBool
	case *ast.NullLit:
		return tNull
	case *ast.StrLit:
		for _, p := range e.Parts {
			if p.Expr != nil {
				c.expr(p.Expr, s)
			}
		}
		return tString
	case *ast.Ident:
		return c.ident(e, s)
	case *ast.ThisExpr:
		if !c.fr.hasSelf {
			c.errorf(e.Pos, "«هذا» متاحة فقط داخل دوال الأنواع والامتدادات")
		}
		return c.fr.self
	case *ast.ItExpr:
		if !c.fr.it {
			c.errorf(e.Pos, "«هو» متاحة فقط داخل دالة مجهولة تأخذ معاملاً واحداً")
		}
		return nil
	case *ast.ListLit:
		var elem *Type
		for i, x := range e.Elems {
			t := c.expr(x, s)
			if i == 0 {
				elem = t
			} else if elem != nil {
				elem = sameType(elem, t)
			}
		}
		return listOf(elem)
	case *ast.MapLit:
		var kt, vt *Type
		for i := range e.Keys {
			k, v := c.expr(e.Keys[i], s), c.expr(e.Values[i], s)
			if i == 0 {
				kt, vt = k, v
			} else {
				kt, vt = sameType(kt, k), sameType(vt, v)
			}
		}
		return &Type{Kind: Map, Key: kt, Elem: vt}
	case *ast.Unary:
		t := c.expr(e.X, s)
		switch e.Op {
		case token.NOT, token.BANG:
			if known(t) && !t.Optional && t.Kind != Bool {
				c.errorf(e.Pos, "«ليس» تعمل على «صح» أو «خطأ» فقط، لكن القيمة من النوع «%s»", t)
			}
			return tBool
		case token.MINUS:
			if known(t) && !t.Optional && !numeric(t) {
				c.errorf(e.Pos, "لا يمكن تطبيق «-» على «%s»", t)
				return nil
			}
			return t
		}
		return nil
	case *ast.Binary:
		return c.binary(e, s)
	case *ast.Call:
		return c.call(e, s)
	case *ast.Member:
		rt := c.expr(e.X, s)
		if !known(rt) {
			return nil
		}
		if rt.Kind == Null {
			if !e.Safe {
				c.errorf(e.Pos, "لا يمكن قراءة «%s» من «عدم»؛ استخدم «؟.» أو «؟؟»", e.Name)
			}
			return nil
		}
		t, found := c.member(rt, e.Name, false)
		if !found {
			c.noMember(rt, e.Name, e.Pos, false)
			return nil
		}
		c.ref(c.tokenAfter(e.Pos), c.memberDef(rt, e.Name))
		if e.Safe || rt.Optional {
			return optional(t)
		}
		return t
	case *ast.Index:
		ct := c.expr(e.X, s)
		it := c.expr(e.I, s)
		if !known(ct) || ct.Optional {
			return nil
		}
		switch ct.Kind {
		case List, Range, String:
			if known(it) && !it.Optional && it.Kind != Int {
				c.errorf(e.I.P(), "الفهرس يجب أن يكون عدداً صحيحاً، لكنه من النوع «%s»", it)
			}
			switch ct.Kind {
			case List:
				return ct.Elem
			case Range:
				return tInt
			}
			return tString
		case Map:
			return optional(ct.Elem)
		case Null:
			c.errorf(e.Pos, "لا يمكن الفهرسة في «عدم»")
		default:
			c.errorf(e.Pos, "لا يمكن الفهرسة في قيمة من النوع «%s»", ct)
		}
		return nil
	case *ast.Lambda:
		c.lambda(e, s, nil)
		return tFunc
	case *ast.If:
		c.condition(e.Cond, s, "شرط «إذا»")
		tt := c.block(e.Then, s.child())
		if e.Else == nil {
			return nil
		}
		var et *Type
		switch els := e.Else.(type) {
		case *ast.Block:
			et = c.block(els, s.child())
		default:
			et = c.expr(els, s)
		}
		return sameType(tt, et)
	case *ast.Match:
		return c.match(e, s)
	case *ast.Try:
		bt := c.block(e.Body, s.child())
		cs := s.child()
		cd := c.def(e.Var, c.nameAfter(e.Catch.Pos, e.Var), "variable", "التقط "+e.Var+": مشكلة", nil, nil)
		if c.recording() {
			cd.Pos = c.nameBeforeOr(e.Catch.Pos, e.Var)
		}
		cs.names[e.Var] = &symbol{kind: sVar, def: cd}
		ct := c.block(e.Catch, cs.child())
		return sameType(bt, ct)
	case *ast.RangeExpr:
		for _, x := range []ast.Expr{e.From, e.To, e.Step} {
			if x == nil {
				continue
			}
			if t := c.expr(x, s); known(t) && !t.Optional && t.Kind != Int {
				c.errorf(x.P(), "حدود النطاق تكون أعداداً صحيحة، لكن هذه من النوع «%s»", t)
			}
		}
		return tRange
	case *ast.TypeTest:
		c.expr(e.X, s)
		c.checkRef(e.Type)
		return tBool
	case *ast.Cast:
		c.expr(e.X, s)
		t := c.checkRef(e.Type)
		return t
	case *ast.Throw:
		c.expr(e.X, s)
		return nil
	case *ast.Block:
		return c.block(e, s.child())
	}
	return nil
}

func (c *checker) ident(e *ast.Ident, s *scope) *Type {
	sym := s.lookup(e.Name)
	if sym == nil {
		if c.selfHas(e.Name) {
			if known(c.fr.self) {
				c.ref(e.Pos, c.memberDef(c.fr.self, e.Name))
				t, _ := c.member(c.fr.self, e.Name, false)
				return t
			}
			return nil
		}
		c.errorf(e.Pos, "«%s» غير معرّف%s", e.Name, didYouMean(e.Name, s.visible()))
		return nil
	}
	c.ref(e.Pos, sym.def)
	switch sym.kind {
	case sFunc, sBuiltin, sElement:
		return tFunc
	}
	return sym.typ
}

// lambda checks a lambda's body. params, when known, are the types its
// parameters receive.
func (c *checker) lambda(l *ast.Lambda, s *scope, params []*Type) {
	ls := s.child()
	for i, p := range l.Params {
		var t *Type
		if i < len(params) {
			t = params[i]
		}
		ld := c.def(p, c.nameAfter(l.Pos, p), "parameter", p+": "+t.String(), t, nil)
		ls.names[p] = &symbol{kind: sVar, typ: t, def: ld}
	}
	fr := &frame{lambda: true, component: c.fr.component, self: c.fr.self, hasSelf: c.fr.hasSelf, it: len(l.Params) == 0}
	body := func() { c.withFrame(fr, func() { c.block(l.Body, ls.child()) }) }
	if c.fr.name == "" && !c.fr.lambda && !c.fr.component && c.quiet == 0 {
		// At the top level a lambda may use names defined further down; check it at the end.
		c.deferred = append(c.deferred, body)
		return
	}
	body()
}

// ---------------------------------------------------------------- operators

func (c *checker) binary(e *ast.Binary, s *scope) *Type {
	switch e.Op {
	case token.AND, token.AND_SYM, token.OR, token.OR_SYM:
		what := "طرف «و»"
		if e.Op == token.OR || e.Op == token.OR_SYM {
			what = "طرف «أو»"
		}
		c.condition(e.L, s, what)
		c.condition(e.R, s, what)
		return tBool
	case token.COALESCE:
		l := c.expr(e.L, s)
		r := c.expr(e.R, s)
		if known(l) {
			nl := *l
			nl.Optional = false
			if known(r) && r.Kind != Null {
				return sameType(&nl, r)
			}
		}
		return nil
	}
	l := c.expr(e.L, s)
	r := c.expr(e.R, s)
	return c.binaryType(e.Op, l, r, e.Pos)
}

var opNames = map[token.Kind]string{
	token.PLUS: "+", token.MINUS: "-", token.STAR: "*", token.SLASH: "/", token.PERCENT: "%",
	token.LT: "<", token.GT: ">", token.LE: "<=", token.GE: ">=",
}

// binaryType gives the type of «l op r» and reports operands it can't take,
// following the interpreter's rules.
func (c *checker) binaryType(op token.Kind, l, r *Type, pos token.Pos) *Type {
	switch op {
	case token.EQ, token.NEQ:
		return tBool
	}
	isCmp := op == token.LT || op == token.GT || op == token.LE || op == token.GE
	result := func() *Type {
		if isCmp {
			return tBool
		}
		return nil
	}
	if !known(l) || !known(r) || (l.Optional && l.Kind != Null) || (r.Optional && r.Kind != Null) {
		if op == token.PLUS && (known(l) && l.Kind == String && !l.Optional || known(r) && r.Kind == String && !r.Optional) {
			return tString
		}
		return result()
	}
	bad := func() *Type {
		c.errorf(pos, "لا يمكن تطبيق «%s» على «%s» و«%s»", opNames[op], l, r)
		return result()
	}
	if op == token.PLUS {
		if l.Kind == String || r.Kind == String {
			return tString
		}
		if l.Kind == List && r.Kind == List {
			return listOf(sameType(l.Elem, r.Elem))
		}
	}
	if isCmp && l.Kind == String && r.Kind == String {
		return tBool
	}
	if !numeric(l) || !numeric(r) {
		return bad()
	}
	if isCmp {
		return tBool
	}
	if op == token.SLASH || l.Kind == Float || r.Kind == Float {
		return tFloat
	}
	return tInt
}

// ---------------------------------------------------------------- calls

func (c *checker) call(e *ast.Call, s *scope) *Type {
	// Argument types, checked once; a trailing lambda is the last positional argument.
	argTypes := make([]*Type, len(e.Args))
	var lambdaParams []*Type
	switch fn := e.Fn.(type) {
	case *ast.Ident:
		sym := s.lookup(fn.Name)
		if sym == nil {
			for i, a := range e.Args {
				argTypes[i] = c.expr(a.Value, s)
			}
			c.trailing(e, s, nil)
			if c.selfHas(fn.Name) {
				if known(c.fr.self) {
					c.ref(fn.Pos, c.memberDef(c.fr.self, fn.Name))
					t, _ := c.member(c.fr.self, fn.Name, true)
					return t
				}
				return nil
			}
			c.errorf(fn.Pos, "«%s» غير معرّف%s", fn.Name, didYouMean(fn.Name, s.visible()))
			return nil
		}
		c.ref(fn.Pos, sym.def)
		// «حقل(اسم)»: the element gets the state variable itself; it's still a name to check.
		for i, a := range e.Args {
			argTypes[i] = c.expr(a.Value, s)
		}
		switch sym.kind {
		case sFunc:
			lambdaParams = nil
			c.trailing(e, s, lambdaParams)
			c.args(sym.what, sym.decl.Params, e, argTypes)
			return c.resultOf(sym.decl)
		case sType, sVariant:
			c.trailing(e, s, nil)
			c.args(sym.what, sym.params, e, argTypes)
			return sym.result
		case sComponent:
			c.trailing(e, s, nil)
			if c.atTopLevel() {
				c.errorf(fn.Pos, "%s يُعرض في صفحة: استدعه داخل مكون آخر، أو شغّل البرنامج بـ«ضاد تطوير»", sym.what)
			}
			c.args(sym.what, sym.params, e, argTypes)
			return nil
		case sElement:
			c.trailing(e, s, nil)
			if c.atTopLevel() {
				c.errorf(fn.Pos, "«%s» عنصر واجهة، ويُستخدم داخل مكون", fn.Name)
			}
			c.elementOptions(fn.Name, e)
			if fn.Name == "أيقونة" && len(e.Args) > 0 && e.Args[0].Name == "" {
				if text, ok := literalText(e.Args[0].Value); ok {
					c.iconName("أيقونة", text, e.Args[0].Value)
				}
			}
			return nil
		case sBuiltin:
			c.trailing(e, s, nil)
			if fn.Name == "سمة" {
				c.themeOptions(e)
			}
			return sym.result
		case sEnum:
			c.trailing(e, s, nil)
			c.errorf(fn.Pos, "«%s» تعداد؛ أنشئ قيمة من إحدى حالاته", fn.Name)
			return nil
		}
		c.trailing(e, s, nil)
		if known(sym.typ) && !sym.typ.Optional && sym.typ.Kind != Func && sym.typ.Kind != Unknown {
			c.errorf(fn.Pos, "«%s» ليس دالة، بل قيمة من النوع «%s»", fn.Name, sym.typ)
		}
		return nil
	case *ast.Member:
		rt := c.expr(fn.X, s)
		for i, a := range e.Args {
			argTypes[i] = c.expr(a.Value, s)
		}
		c.trailing(e, s, c.lambdaParamsFor(rt, fn.Name))
		if !known(rt) {
			return nil
		}
		if rt.Kind == Null {
			if !fn.Safe {
				c.errorf(fn.Pos, "لا يمكن استدعاء «%s» على «عدم»؛ استخدم «؟.» أو «؟؟»", fn.Name)
			}
			return nil
		}
		t, found := c.member(rt, fn.Name, true)
		if !found {
			c.noMember(rt, fn.Name, fn.Pos, true)
			return nil
		}
		c.ref(c.tokenAfter(fn.Pos), c.memberDef(rt, fn.Name))
		c.methodArgs(rt, fn.Name, e, argTypes)
		if fn.Safe || rt.Optional {
			return optional(t)
		}
		return t
	default:
		c.expr(e.Fn, s)
		for i, a := range e.Args {
			argTypes[i] = c.expr(a.Value, s)
		}
		c.trailing(e, s, nil)
		return nil
	}
}

// trailing checks a call's trailing lambda.
func (c *checker) trailing(e *ast.Call, s *scope, params []*Type) {
	if e.Lambda != nil {
		c.lambda(e.Lambda, s, params)
	}
}

// lambdaParamsFor gives the lambda of «قائمة.رشح { … }» its element type.
func (c *checker) lambdaParamsFor(rt *Type, method string) []*Type {
	if !known(rt) || rt.Optional {
		return nil
	}
	switch rt.Kind {
	case List:
		switch method {
		case "رشح", "حول", "لكل", "أي", "كل", "رتب":
			return []*Type{rt.Elem}
		}
	case Range:
		switch method {
		case "رشح", "حول", "لكل", "أي", "كل", "رتب":
			return []*Type{tInt}
		}
	case TypeValue:
		if method == "حيث" {
			return []*Type{c.types[rt.Name]}
		}
	}
	return nil
}

// args checks a call's arguments against parameters, as bindArgs does at run time.
func (c *checker) args(what string, params []*ast.Param, e *ast.Call, types []*Type) {
	set := make([]bool, len(params))
	positional := 0
	sawNamed := false
	for i, a := range e.Args {
		if a.Name != "" {
			sawNamed = true
			continue
		}
		if sawNamed {
			c.errorf(a.Value.P(), "المعاملات المسماة يجب أن تأتي بعد المعاملات العادية")
		}
		if positional < len(params) {
			set[positional] = true
			c.argType(what, params[positional], types[i], a.Value.P())
		}
		positional++
	}
	if e.Lambda != nil {
		if positional < len(params) {
			set[positional] = true
		}
		positional++
	}
	if positional > len(params) {
		c.errorf(e.Pos, "%s تأخذ %d من المعاملات لكن أُعطيت %d", what, len(params), positional)
		return
	}
	for i, a := range e.Args {
		if a.Name == "" {
			continue
		}
		idx := -1
		for j, p := range params {
			if p.Name == a.Name {
				idx = j
			}
		}
		if idx < 0 {
			var names []string
			for _, p := range params {
				names = append(names, p.Name)
			}
			c.errorf(a.Value.P(), "%s لا تحتوي معاملاً باسم «%s»%s", what, a.Name, didYouMean(a.Name, names))
			continue
		}
		if set[idx] {
			c.errorf(a.Value.P(), "المعامل «%s» أُعطي مرتين", a.Name)
			continue
		}
		set[idx] = true
		if pos, ok := c.nameBefore(a.Value.P(), a.Name); ok {
			c.ref(pos, c.paramDefs[params[idx]])
		}
		c.argType(what, params[idx], types[i], a.Value.P())
	}
	for i, p := range params {
		if !set[i] && p.Default == nil {
			c.errorf(e.Pos, "ينقص المعامل «%s» عند استدعاء %s", p.Name, what)
		}
	}
}

func (c *checker) argType(what string, p *ast.Param, got *Type, pos token.Pos) {
	want, _ := c.fromRef(p.Type)
	if p.Type == nil {
		want = literalType(p.Default) // a table field like «منجزة = خطأ»
	}
	if want == nil {
		return
	}
	if !assignable(got, want) {
		c.errorf(pos, "المعامل «%s» في %s: توقعت قيمة من النوع «%s» لكن وجدت «%s»", p.Name, what, want, got)
	}
}

// resultOf is a function's result type: the declared one, or the type of an
// «= expr» body.
func (c *checker) resultOf(d *ast.FuncDecl) *Type {
	if d.Result != nil {
		t, _ := c.fromRef(d.Result)
		return t
	}
	if _, isBlock := d.Body.(*ast.Block); isBlock {
		return nil
	}
	if t, ok := c.results[d]; ok {
		return t
	}
	if c.inferring[d] {
		return nil // recursion
	}
	c.inferring[d] = true
	c.quiet++
	s := c.global.child()
	for _, p := range d.Params {
		t, _ := c.fromRef(p.Type)
		s.names[p.Name] = &symbol{kind: sVar, typ: t}
	}
	var t *Type
	c.withFrame(&frame{name: d.Name}, func() { t = c.expr(d.Body, s) })
	c.quiet--
	delete(c.inferring, d)
	c.results[d] = t
	return t
}

// elementOptions checks the named arguments of an interface element.
func (c *checker) elementOptions(name string, e *ast.Call) {
	allowed := interp.ElementOptions[name]
	for _, a := range e.Args {
		if a.Name == "" {
			continue
		}
		ok := false
		for _, x := range allowed {
			ok = ok || x == a.Name
		}
		if ok {
			c.optionValue(name, a.Name, a.Value)
			continue
		}
		if len(allowed) == 0 {
			c.errorf(a.Value.P(), "«%s» لا يأخذ معاملات مسماة", name)
		} else {
			c.errorf(a.Value.P(), "«%s» لا يأخذ «%s»؛ يأخذ: %s%s", name, a.Name, strings.Join(allowed, "، "), didYouMean(a.Name, allowed))
		}
	}
}

// literalText is the text of a string literal without {…}, if e is one.
func literalText(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.StrLit)
	if !ok {
		return "", false
	}
	text := ""
	for _, p := range lit.Parts {
		if p.Expr != nil {
			return "", false
		}
		text += p.Text
	}
	return text, true
}

// optionValue checks the words, colours and icons written in an element's options.
func (c *checker) optionValue(element, opt string, v ast.Expr) {
	text, ok := literalText(v)
	if !ok {
		return
	}
	switch {
	case opt == "لون" || opt == "خلفية":
		if !interp.IsColor(text) {
			c.errorf(v.P(), "«%s» في «%s» يأخذ اسم لون مثل «أزرق» أو «فيروزي»، أو قيمة مثل «#1b2b6b»%s", opt, element, didYouMean(text, interp.ColorNames()))
		}
	case opt == "أيقونة":
		c.iconName(element, text, v)
	case interp.OptionWords(element, opt) != nil:
		words := interp.OptionWords(element, opt)
		if !has(words, text) {
			c.errorf(v.P(), "«%s» في «%s» يكون %s%s", opt, element, quoteWords(words), didYouMean(text, words))
		}
	}
}

func (c *checker) iconName(element, name string, at ast.Expr) {
	if _, ok := interp.Icons[name]; ok {
		return
	}
	names := make([]string, 0, len(interp.Icons))
	for n := range interp.Icons {
		names = append(names, n)
	}
	c.errorf(at.P(), "لا توجد أيقونة باسم «%s» في «%s»%s", name, element, didYouMean(name, names))
}

// themeOptions checks «سمة(اللون: "فيروزي"، الخط: "نسخ")».
func (c *checker) themeOptions(e *ast.Call) {
	for _, a := range e.Args {
		if a.Name == "" {
			c.errorf(a.Value.P(), "«سمة» تأخذ معاملات مسماة، مثل: سمة(اللون: \"فيروزي\")")
			continue
		}
		if !has(interp.ThemeOptions, a.Name) {
			c.errorf(a.Value.P(), "«سمة» لا تأخذ «%s»؛ تأخذ: %s%s", a.Name, strings.Join(interp.ThemeOptions, "، "), didYouMean(a.Name, interp.ThemeOptions))
			continue
		}
		text, ok := literalText(a.Value)
		if !ok {
			continue
		}
		if words := interp.ThemeValues[a.Name]; words != nil && !has(words, text) {
			c.errorf(a.Value.P(), "«%s» في «سمة» يكون %s%s", a.Name, quoteWords(words), didYouMean(text, words))
		}
		if a.Name == "اللون" && (!interp.IsColor(text) || text == "أساسي" || text == "باهت") {
			c.errorf(a.Value.P(), "«اللون» في «سمة» يأخذ لوناً بعينه، مثل «فيروزي» أو «#0e8f9a»%s", didYouMean(text, interp.ColorNames()))
		}
	}
}

func quoteWords(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = "«" + w + "»"
	}
	return strings.Join(q, " أو ")
}

// ---------------------------------------------------------------- members

var (
	listMethods   = []string{"طول", "عدد", "أول", "آخر", "فارغ", "رشح", "حول", "لكل", "أي", "كل", "اختزل", "ضم", "يحتوي", "مجموع", "عكس", "رتب", "أضف"}
	stringMethods = []string{"طول", "عدد", "قص", "يحتوي", "يبدأ_ب", "يبدأ", "ينتهي_ب", "ينتهي", "استبدل", "قسم", "أحرف", "كبير", "صغير", "كرر"}
	numberMethods = []string{"مطلق", "تقريب", "أرضية", "سقف"}
	tableMethods  = []string{"أنشئ", "الكل", "حيث", "بالرقم", "عدد"}
)

func has(list []string, name string) bool {
	for _, x := range list {
		if x == name {
			return true
		}
	}
	return false
}

// member looks up «x.name» on a value of type t. call says whether it is
// being called. found is false only when the type certainly has no such member.
func (c *checker) member(t *Type, name string, call bool) (result *Type, found bool) {
	if ext := c.ext[typeKey(t)][name]; ext != nil && call {
		return c.resultOf(ext), true
	}
	switch t.Kind {
	case Struct:
		info := c.structs[t.Name]
		if info == nil {
			return nil, true
		}
		for _, f := range info.fields {
			if f.Name == name {
				return info.ftypes[name], true
			}
		}
		if m, ok := info.methods[name]; ok {
			if call {
				return c.resultOf(m), true
			}
			return tFunc, true
		}
		if call && name == "نسخة" {
			return &Type{Kind: Struct, Name: t.Name}, true
		}
		if call && info.table {
			switch name {
			case "احفظ":
				return &Type{Kind: Struct, Name: t.Name}, true
			case "احذف":
				return tBool, true
			}
		}
		return nil, false
	case TypeValue:
		info := c.structs[t.Name]
		if info != nil && info.table && call {
			rec := &Type{Kind: Struct, Name: t.Name}
			switch name {
			case "أنشئ":
				return rec, true
			case "الكل", "حيث":
				return listOf(rec), true
			case "بالرقم":
				return optional(rec), true
			case "عدد":
				return tInt, true
			}
		}
		return nil, false
	case EnumType:
		info := c.enums[t.Name]
		for _, v := range info.variants {
			if v.Name == name {
				if len(v.Fields) > 0 && !call {
					return tFunc, true
				}
				return c.types[t.Name], true
			}
		}
		return nil, false
	case Enum:
		if call && name == "نسخة" {
			return c.types[t.Name], true
		}
		for _, v := range c.enums[t.Name].variants {
			for _, f := range v.Fields {
				if f.Name == name {
					ft, _ := c.fromRef(f.Type)
					return ft, true
				}
			}
		}
		return nil, false
	case String:
		if call {
			if !has(stringMethods, name) {
				return nil, false
			}
		} else if name != "طول" && name != "عدد" && name != "فارغ" {
			return nil, false
		}
		switch name {
		case "طول", "عدد":
			return tInt, true
		case "فارغ", "يحتوي", "يبدأ_ب", "يبدأ", "ينتهي_ب", "ينتهي":
			return tBool, true
		case "قسم", "أحرف":
			return listOf(tString), true
		}
		return tString, true
	case List, Range:
		elem := t.Elem
		if t.Kind == Range {
			elem = tInt
		}
		if call {
			if !has(listMethods, name) {
				return nil, false
			}
		} else if !has([]string{"طول", "عدد", "فارغ", "أول", "آخر"}, name) {
			return nil, false
		}
		switch name {
		case "طول", "عدد":
			return tInt, true
		case "فارغ", "يحتوي", "أي", "كل":
			return tBool, true
		case "أول", "آخر":
			return optional(elem), true
		case "رشح", "عكس", "رتب":
			return listOf(elem), true
		case "حول":
			return listOf(nil), true
		case "ضم":
			return tString, true
		case "لكل", "أضف":
			return tNull, true
		case "مجموع":
			if numeric(elem) {
				return elem, true
			}
		}
		return nil, true
	case Map:
		if call {
			switch name {
			case "يحتوي":
				return tBool, true
			case "احذف":
				return tNull, true
			}
			return nil, false
		}
		switch name {
		case "طول", "عدد":
			return tInt, true
		case "فارغ":
			return tBool, true
		case "مفاتيح":
			return listOf(t.Key), true
		case "قيم":
			return listOf(t.Elem), true
		}
		return nil, false
	case Int, Float:
		if call && has(numberMethods, name) {
			switch name {
			case "أرضية", "سقف":
				return tInt, true
			case "مطلق":
				return &Type{Kind: t.Kind}, true
			}
			return nil, true
		}
		return nil, false
	case Bool:
		return nil, false
	case Problem:
		if !call && name == "رسالة" {
			return tString, true
		}
		return nil, false
	}
	return nil, true
}

// noMember reports a member that a type doesn't have, with the closest name.
func (c *checker) noMember(t *Type, name string, pos token.Pos, call bool) {
	var names []string
	switch t.Kind {
	case Struct:
		if info := c.structs[t.Name]; info != nil {
			for _, f := range info.fields {
				names = append(names, f.Name)
			}
			for m := range info.methods {
				names = append(names, m)
			}
			if info.table {
				names = append(names, "احفظ", "احذف")
			}
		}
		names = append(names, "نسخة")
	case TypeValue:
		if info := c.structs[t.Name]; info != nil && info.table {
			names = tableMethods
			if !call {
				c.errorf(pos, "استدعِ «%s» كدالة: %s.%s(…)", name, t.Name, name)
				return
			}
		}
	case EnumType:
		for _, v := range c.enums[t.Name].variants {
			names = append(names, v.Name)
		}
	case String:
		names = stringMethods
	case List, Range:
		names = listMethods
	case Map:
		names = []string{"طول", "عدد", "فارغ", "مفاتيح", "قيم", "يحتوي", "احذف"}
	case Int, Float:
		names = numberMethods
	case Problem:
		names = []string{"رسالة"}
	}
	for m := range c.ext[typeKey(t)] {
		names = append(names, m)
	}
	sort.Strings(names)
	c.errorf(pos, "القيمة من النوع «%s» لا تحتوي «%s»%s", strings.TrimSuffix(t.String(), "؟"), name, didYouMean(name, names))
}

// methodArgs checks the arguments of calls to a type's own methods and نسخة.
func (c *checker) methodArgs(t *Type, name string, e *ast.Call, types []*Type) {
	if ext := c.ext[typeKey(t)][name]; ext != nil {
		c.args("«"+name+"»", ext.Params, e, types)
		return
	}
	switch t.Kind {
	case Struct:
		info := c.structs[t.Name]
		if info == nil {
			return
		}
		if m, ok := info.methods[name]; ok {
			c.args("«"+name+"»", m.Params, e, types)
			return
		}
		if name == "نسخة" {
			c.copyArgs(info.fields, e, types, info.ftypes)
		}
	case TypeValue:
		info := c.structs[t.Name]
		if info != nil && info.table && name == "أنشئ" {
			c.args("«"+t.Name+"»", info.fields, e, types)
		}
	}
}

// copyArgs checks «نسخة(حقل: قيمة)»: named arguments only, each a field.
func (c *checker) copyArgs(fields []*ast.Param, e *ast.Call, types []*Type, ftypes map[string]*Type) {
	var names []string
	for _, f := range fields {
		names = append(names, f.Name)
	}
	for i, a := range e.Args {
		if a.Name == "" {
			c.errorf(a.Value.P(), "«نسخة» تقبل معاملات مسماة فقط، مثل: نسخة(%s: …)", fields[0].Name)
			continue
		}
		if !has(names, a.Name) {
			c.errorf(a.Value.P(), "لا يوجد حقل باسم «%s»%s", a.Name, didYouMean(a.Name, names))
			continue
		}
		if pos, ok := c.nameBefore(a.Value.P(), a.Name); ok {
			for _, f := range fields {
				if f.Name == a.Name {
					c.ref(pos, c.paramDefs[f])
				}
			}
		}
		if want := ftypes[a.Name]; !assignable(types[i], want) {
			c.errorf(a.Value.P(), "الحقل «%s» من النوع «%s»، لكن القيمة من النوع «%s»", a.Name, want, types[i])
		}
	}
}

// ---------------------------------------------------------------- طابق

func (c *checker) match(m *ast.Match, s *scope) *Type {
	st := c.expr(m.Subject, s)
	var enum *enumInfo
	if known(st) && st.Kind == Enum {
		enum = c.enums[st.Name]
	}
	covered := map[string]bool{}
	catchAll := false
	var result *Type
	for i, cs := range m.Cases {
		scope := s.child()
		p := cs.Pattern
		switch p.Kind {
		case ast.WildcardPattern:
			catchAll = true
		case ast.LiteralPattern:
			c.expr(p.Lit, s)
		case ast.NamePattern:
			en, isVariant := c.variantOf[p.Name]
			switch {
			case isVariant:
				covered[p.Name] = true
				if sym := c.global.lookup(p.Name); sym != nil {
					c.ref(p.Pos, sym.def)
				}
				if enum != nil && en != enum.name {
					c.errorf(p.Pos, "«%s» ليست من حالات «%s»", p.Name, enum.name)
				}
				var fields []*ast.Param
				for _, v := range c.enums[en].variants {
					if v.Name == p.Name {
						fields = v.Fields
					}
				}
				if len(p.Binds) > len(fields) {
					c.errorf(p.Pos, "الحالة «%s» تحتوي %d حقول لكن النمط يذكر %d", p.Name, len(fields), len(p.Binds))
				}
				for j, b := range p.Binds {
					if b == "_" {
						continue
					}
					var bt *Type
					if j < len(fields) {
						bt, _ = c.fromRef(fields[j].Type)
					}
					bd := c.def(b, c.nameAfter(p.Pos, b), "variable", b+": "+bt.String(), bt, nil)
					scope.names[b] = &symbol{kind: sVar, typ: bt, def: bd}
				}
			case p.HasArgs:
				c.errorf(p.Pos, "«%s» ليست حالة معروفة في أي تعداد%s", p.Name, didYouMean(p.Name, keys(c.variantOf)))
			default:
				catchAll = true
				bd := c.def(p.Name, p.Pos, "variable", p.Name+": "+st.String(), st, nil)
				scope.names[p.Name] = &symbol{kind: sVar, typ: st, def: bd}
			}
		}
		var bt *Type
		if b, ok := cs.Body.(*ast.Block); ok {
			bt = c.block(b, scope.child())
		} else {
			bt = c.expr(cs.Body, scope)
		}
		if i == 0 {
			result = bt
		} else if result != nil {
			result = sameType(result, bt)
		}
	}
	if enum != nil && !catchAll {
		var missing []string
		for _, v := range enum.variants {
			if !covered[v.Name] {
				missing = append(missing, "«"+v.Name+"»")
			}
		}
		if len(missing) > 0 {
			c.errorf(m.Pos, "«طابق» لا تغطي كل حالات «%s»: تنقص %s. أضفها، أو أضف «_ => …» لما تبقى",
				enum.name, strings.Join(missing, "، "))
		}
	}
	return result
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// literalType is the type of a literal default value, or nil.
func literalType(e ast.Expr) *Type {
	switch x := e.(type) {
	case *ast.IntLit:
		return tInt
	case *ast.FloatLit:
		return tFloat
	case *ast.BoolLit:
		return tBool
	case *ast.StrLit:
		return tString
	case *ast.Unary:
		if x.Op == token.MINUS {
			return literalType(x.X)
		}
	}
	return nil
}
