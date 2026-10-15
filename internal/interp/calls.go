package interp

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

type namedArg struct {
	name string
	val  Value
}

// callInfo carries evaluated arguments to any callable.
type callInfo struct {
	pos   token.Pos
	name  string
	args  []Value
	named []namedArg
}

func (in *Interp) call(c *ast.Call, env *Env) Value {
	ci := &callInfo{pos: c.Pos}
	for _, a := range c.Args {
		v := in.eval(a.Value, env)
		if a.Name != "" {
			ci.named = append(ci.named, namedArg{name: a.Name, val: v})
			continue
		}
		if len(ci.named) > 0 {
			in.fail(a.Value.P(), "المعاملات المسماة يجب أن تأتي بعد المعاملات العادية")
		}
		ci.args = append(ci.args, v)
	}
	if c.Lambda != nil {
		ci.args = append(ci.args, in.eval(c.Lambda, env))
	}
	if m, ok := c.Fn.(*ast.Member); ok {
		recv := in.eval(m.X, env)
		if recv == nil && m.Safe {
			return nil
		}
		ci.name = m.Name
		return in.callMethod(recv, m.Name, ci)
	}
	if id, ok := c.Fn.(*ast.Ident); ok {
		ci.name = id.Name
		// Inside methods and extensions, "نسخة(…)" or "تحية()" call methods of هذا.
		if env.lookup(id.Name) == nil {
			if self := env.lookup("هذا"); self != nil {
				return in.callMethod(self.val, id.Name, ci)
			}
		}
	}
	return in.callValue(in.eval(c.Fn, env), ci)
}

func (in *Interp) callValue(fn Value, ci *callInfo) Value {
	in.depth++
	defer func() { in.depth-- }()
	if in.depth > maxDepth {
		in.fail(ci.pos, "تجاوز البرنامج الحد الأقصى لعمق الاستدعاءات — هل توجد دالة تستدعي نفسها بلا نهاية؟")
	}
	switch f := fn.(type) {
	case *Func:
		return in.callFunc(f, ci)
	case *Builtin:
		return f.Fn(in, ci)
	case *StructType:
		return &Struct{Type: f, Vals: in.bindArgs(f.Fields, ci, newEnv(f.Env), "«"+f.Name+"»")}
	case *Variant:
		return &EnumVal{Variant: f, Vals: in.bindArgs(f.Fields, ci, newEnv(in.globals), "«"+f.Name+"»")}
	case *EnumVal:
		if len(ci.args) == 0 && len(ci.named) == 0 {
			return f
		}
	case nil:
		in.fail(ci.pos, "لا يمكن استدعاء «عدم»")
	}
	name := ci.name
	if name == "" {
		name = Format(fn)
	}
	in.fail(ci.pos, "«%s» ليس دالة، بل قيمة من النوع «%s»", name, typeName(fn))
	return nil
}

func (in *Interp) callFunc(f *Func, ci *callInfo) Value {
	scope := f.Env
	if f.HasRecv {
		scope = newEnv(f.Env)
		scope.define("هذا", f.Recv, false)
		if s, ok := f.Recv.(*Struct); ok {
			for i, fld := range s.Type.Fields {
				scope.define(fld.Name, s.Vals[i], false)
			}
		}
	}
	env := newEnv(scope)
	if f.IsLam {
		if len(ci.named) > 0 {
			in.fail(ci.pos, "الدوال المجهولة لا تقبل معاملات مسماة")
		}
		if len(f.Lambda) == 0 {
			if len(ci.args) > 0 {
				env.define("هو", ci.args[0], false)
			}
		} else {
			if len(ci.args) != len(f.Lambda) {
				in.fail(ci.pos, "الدالة المجهولة تتوقع %s من المعاملات لكن أُعطيت %s",
					formatInt(int64(len(f.Lambda))), formatInt(int64(len(ci.args))))
			}
			for i, n := range f.Lambda {
				env.define(n, ci.args[i], false)
			}
		}
		v, _ := in.runBody(f, env)
		return v
	}
	in.bindArgs(f.Params, ci, env, "«"+f.Name+"»")
	v, returned := in.runBody(f, env)
	if _, isBlock := f.Body.(*ast.Block); isBlock && !returned && f.Result != nil && !f.Result.Optional {
		in.fail(ci.pos, "الدالة «%s» يجب أن ترجع قيمة من النوع «%s» — هل نسيت «أرجع»؟", f.Name, typeString(f.Result))
	}
	if f.Result != nil {
		v = in.convert(v, f.Result, ci.pos, "قيمة الإرجاع من «"+f.Name+"»")
	}
	return v
}

// runBody runs a function body; returned reports whether «أرجع» was used.
func (in *Interp) runBody(f *Func, env *Env) (result Value, returned bool) {
	defer func() {
		if r := recover(); r != nil {
			if rs, ok := r.(returnSig); ok {
				result, returned = rs.val, true
				return
			}
			panic(r)
		}
	}()
	if b, ok := f.Body.(*ast.Block); ok {
		v := in.runStmts(b.Stmts, env)
		if f.IsLam {
			return v, false
		}
		return nil, false // block-bodied functions return with «أرجع»
	}
	return in.eval(f.Body, env), false
}

// bindArgs matches positional and named arguments to parameters, fills in
// defaults, checks types and defines the parameters in env.
func (in *Interp) bindArgs(params []*ast.Param, ci *callInfo, env *Env, what string) []Value {
	vals := make([]Value, len(params))
	set := make([]bool, len(params))
	if len(ci.args) > len(params) {
		in.fail(ci.pos, "%s تأخذ %s من المعاملات لكن أُعطيت %s", what,
			formatInt(int64(len(params))), formatInt(int64(len(ci.args))))
	}
	for i, a := range ci.args {
		vals[i], set[i] = a, true
	}
	for _, na := range ci.named {
		idx := paramIndex(params, na.name)
		if idx < 0 {
			in.fail(ci.pos, "%s لا تحتوي معاملاً باسم «%s»", what, na.name)
		}
		if set[idx] {
			in.fail(ci.pos, "المعامل «%s» أُعطي مرتين", na.name)
		}
		vals[idx], set[idx] = na.val, true
	}
	for i, p := range params {
		if !set[i] {
			if p.Default == nil {
				in.fail(ci.pos, "ينقص المعامل «%s» عند استدعاء %s", p.Name, what)
			}
			vals[i] = in.eval(p.Default, env)
		}
		vals[i] = in.convert(vals[i], p.Type, ci.pos, "المعامل «"+p.Name+"» في "+what)
		env.define(p.Name, vals[i], false)
	}
	return vals
}

func paramIndex(params []*ast.Param, name string) int {
	for i, p := range params {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (in *Interp) bindMethod(m *ast.FuncDecl, recv Value, env *Env) *Func {
	return &Func{Name: m.Name, Params: m.Params, Body: m.Body, Env: env, Result: m.Result, Recv: recv, HasRecv: true}
}

func (in *Interp) extFor(recv Value, name string) *Func {
	if recv == nil {
		return nil
	}
	m := in.ext[typeKey(recv)][name]
	if m == nil {
		return nil
	}
	bound := *m
	bound.Recv, bound.HasRecv = recv, true
	return &bound
}

// ---------------------------------------------------------------- members

func (in *Interp) callMethod(recv Value, name string, ci *callInfo) Value {
	switch r := recv.(type) {
	case *Struct:
		if i := paramIndex(r.Type.Fields, name); i >= 0 {
			return in.callValue(r.Vals[i], ci)
		}
		if m, ok := r.Type.Methods[name]; ok {
			return in.callValue(in.bindMethod(m, r, r.Type.Env), ci)
		}
		if name == "نسخة" {
			return &Struct{Type: r.Type, Vals: in.copyWith(r.Type.Fields, r.Vals, ci)}
		}
	case *EnumVal:
		if name == "نسخة" && len(r.Vals) > 0 {
			return &EnumVal{Variant: r.Variant, Vals: in.copyWith(r.Variant.Fields, r.Vals, ci)}
		}
	}
	if f := in.extFor(recv, name); f != nil {
		return in.callValue(f, ci)
	}
	if v, ok := in.builtinMethod(recv, name, ci); ok {
		return v
	}
	if recv == nil {
		in.fail(ci.pos, "لا يمكن استدعاء «%s» على «عدم»؛ استخدم «؟.» أو «؟؟»", name)
	}
	return in.callValue(in.member(recv, name, ci.pos), ci)
}

func (in *Interp) member(recv Value, name string, pos token.Pos) Value {
	if v, ok := in.tryMember(recv, name); ok {
		return v
	}
	if recv == nil {
		in.fail(pos, "لا يمكن قراءة «%s» من «عدم»؛ استخدم «؟.» أو «؟؟»", name)
	}
	in.fail(pos, "القيمة من النوع «%s» لا تحتوي «%s»", typeName(recv), name)
	return nil
}

func (in *Interp) tryMember(recv Value, name string) (Value, bool) {
	switch r := recv.(type) {
	case *Struct:
		if i := paramIndex(r.Type.Fields, name); i >= 0 {
			return r.Vals[i], true
		}
		if m, ok := r.Type.Methods[name]; ok {
			return in.bindMethod(m, r, r.Type.Env), true
		}
	case *EnumVal:
		if i := paramIndex(r.Variant.Fields, name); i >= 0 {
			return r.Vals[i], true
		}
	case *EnumType:
		for _, v := range r.Variants {
			if v.Name == name {
				if len(v.Fields) == 0 {
					return &EnumVal{Variant: v}, true
				}
				return v, true
			}
		}
	case *Problem:
		if name == "رسالة" {
			return r.Message, true
		}
	case string:
		switch name {
		case "طول", "عدد":
			return int64(len([]rune(r))), true
		case "فارغ":
			return r == "", true
		}
	case *List:
		if v, ok := listProperty(r.Elems, name); ok {
			return v, true
		}
	case *Range:
		if v, ok := listProperty(r.values(), name); ok {
			return v, true
		}
	case *Map:
		switch name {
		case "طول", "عدد":
			return int64(len(r.Keys)), true
		case "فارغ":
			return len(r.Keys) == 0, true
		case "مفاتيح":
			return &List{Elems: append([]Value(nil), r.Keys...)}, true
		case "قيم":
			return &List{Elems: append([]Value(nil), r.Vals...)}, true
		}
	}
	if f := in.extFor(recv, name); f != nil {
		return f, true
	}
	return nil, false
}

func listProperty(elems []Value, name string) (Value, bool) {
	switch name {
	case "طول", "عدد":
		return int64(len(elems)), true
	case "فارغ":
		return len(elems) == 0, true
	case "أول":
		if len(elems) == 0 {
			return nil, true
		}
		return elems[0], true
	case "آخر":
		if len(elems) == 0 {
			return nil, true
		}
		return elems[len(elems)-1], true
	}
	return nil, false
}

func (in *Interp) copyWith(fields []*ast.Param, vals []Value, ci *callInfo) []Value {
	if len(ci.args) > 0 {
		in.fail(ci.pos, "«نسخة» تقبل معاملات مسماة فقط، مثل: نسخة(%s: …)", fields[0].Name)
	}
	out := append([]Value(nil), vals...)
	for _, na := range ci.named {
		i := paramIndex(fields, na.name)
		if i < 0 {
			in.fail(ci.pos, "لا يوجد حقل باسم «%s»", na.name)
		}
		out[i] = in.convert(na.val, fields[i].Type, ci.pos, "الحقل «"+na.name+"»")
	}
	return out
}

// ---------------------------------------------------------------- built-in methods

func (in *Interp) fnArg(ci *callInfo, i int, method string) Value {
	if i >= len(ci.args) {
		in.fail(ci.pos, "«%s» تحتاج دالة، مثل: %s { هو … }", method, method)
	}
	switch ci.args[i].(type) {
	case *Func, *Builtin:
		return ci.args[i]
	}
	in.fail(ci.pos, "«%s» تحتاج دالة لكن أُعطيت قيمة من النوع «%s»", method, typeName(ci.args[i]))
	return nil
}

func (in *Interp) apply(f Value, pos token.Pos, args ...Value) Value {
	return in.callValue(f, &callInfo{pos: pos, args: args})
}

func (in *Interp) argCount(ci *callInfo, n int, method string) {
	if len(ci.args) != n || len(ci.named) > 0 {
		in.fail(ci.pos, "«%s» تأخذ %s من المعاملات لكن أُعطيت %s", method,
			formatInt(int64(n)), formatInt(int64(len(ci.args)+len(ci.named))))
	}
}

func (in *Interp) builtinMethod(recv Value, name string, ci *callInfo) (Value, bool) {
	switch r := recv.(type) {
	case *List:
		return in.listMethod(r.Elems, r, name, ci)
	case *Range:
		return in.listMethod(r.values(), nil, name, ci)
	case string:
		return in.stringMethod(r, name, ci)
	case int64, float64:
		return in.numberMethod(r, name, ci)
	case *Map:
		switch name {
		case "يحتوي":
			in.argCount(ci, 1, name)
			return r.find(ci.args[0]) >= 0, true
		case "احذف":
			in.argCount(ci, 1, name)
			if j := r.find(ci.args[0]); j >= 0 {
				r.Keys = append(r.Keys[:j], r.Keys[j+1:]...)
				r.Vals = append(r.Vals[:j], r.Vals[j+1:]...)
			}
			return nil, true
		}
	}
	return nil, false
}

func (in *Interp) listMethod(elems []Value, list *List, name string, ci *callInfo) (Value, bool) {
	switch name {
	case "طول", "عدد":
		if len(ci.args) == 0 {
			return int64(len(elems)), true
		}
		f := in.fnArg(ci, 0, name)
		var n int64
		for _, x := range elems {
			if in.truthy(in.apply(f, ci.pos, x), ci.pos, "نتيجة «عدد»") {
				n++
			}
		}
		return n, true
	case "أول", "آخر", "فارغ":
		v, _ := listProperty(elems, name)
		return v, true
	case "رشح":
		f := in.fnArg(ci, 0, name)
		out := &List{}
		for _, x := range elems {
			if in.truthy(in.apply(f, ci.pos, x), ci.pos, "نتيجة «رشح»") {
				out.Elems = append(out.Elems, x)
			}
		}
		return out, true
	case "حول":
		f := in.fnArg(ci, 0, name)
		out := &List{Elems: make([]Value, len(elems))}
		for i, x := range elems {
			out.Elems[i] = in.apply(f, ci.pos, x)
		}
		return out, true
	case "لكل":
		f := in.fnArg(ci, 0, name)
		for _, x := range elems {
			in.apply(f, ci.pos, x)
		}
		return nil, true
	case "أي", "كل":
		f := in.fnArg(ci, 0, name)
		for _, x := range elems {
			ok := in.truthy(in.apply(f, ci.pos, x), ci.pos, "نتيجة «"+name+"»")
			if name == "أي" && ok {
				return true, true
			}
			if name == "كل" && !ok {
				return false, true
			}
		}
		return name == "كل", true
	case "اختزل":
		var acc Value
		var f Value
		rest := elems
		switch len(ci.args) {
		case 2:
			acc, f = ci.args[0], in.fnArg(ci, 1, name)
		case 1:
			f = in.fnArg(ci, 0, name)
			if len(elems) == 0 {
				in.raise(ci.pos, "لا يمكن «اختزل» قائمة فارغة بلا قيمة بداية")
			}
			acc, rest = elems[0], elems[1:]
		default:
			in.fail(ci.pos, "«اختزل» تأخذ قيمة بداية ودالة، مثل: اختزل(٠) { أ، ب => أ + ب }")
		}
		for _, x := range rest {
			acc = in.apply(f, ci.pos, acc, x)
		}
		return acc, true
	case "ضم":
		sep := ""
		if len(ci.args) > 0 {
			s, ok := ci.args[0].(string)
			if !ok {
				in.fail(ci.pos, "فاصل «ضم» يجب أن يكون نصاً")
			}
			sep = s
		}
		return joinValues(elems, sep, false), true
	case "يحتوي":
		in.argCount(ci, 1, name)
		for _, x := range elems {
			if equal(x, ci.args[0]) {
				return true, true
			}
		}
		return false, true
	case "مجموع":
		var si int64
		var sf float64
		isFloat := false
		for _, x := range elems {
			switch n := x.(type) {
			case int64:
				si += n
				sf += float64(n)
			case float64:
				isFloat = true
				sf += n
			default:
				in.fail(ci.pos, "«مجموع» تعمل على قوائم الأعداد فقط")
			}
		}
		if isFloat {
			return sf, true
		}
		return si, true
	case "عكس":
		out := &List{Elems: make([]Value, len(elems))}
		for i, x := range elems {
			out.Elems[len(elems)-1-i] = x
		}
		return out, true
	case "رتب":
		out := &List{Elems: append([]Value(nil), elems...)}
		var bad bool
		sort.SliceStable(out.Elems, func(i, j int) bool {
			a, b := out.Elems[i], out.Elems[j]
			if sa, ok := a.(string); ok {
				if sb, ok := b.(string); ok {
					return sa < sb
				}
			}
			fa, okA := toFloat(a)
			fb, okB := toFloat(b)
			if !okA || !okB {
				bad = true
				return false
			}
			return fa < fb
		})
		if bad {
			in.fail(ci.pos, "«رتب» تعمل على قوائم الأعداد أو النصوص فقط")
		}
		return out, true
	case "أضف":
		if list == nil {
			in.fail(ci.pos, "لا يمكن الإضافة إلى نطاق؛ حوّله إلى قائمة أولاً")
		}
		list.Elems = append(list.Elems, ci.args...)
		return nil, true
	}
	return nil, false
}

func (in *Interp) stringMethod(s, name string, ci *callInfo) (Value, bool) {
	arg := func(i int) string {
		if i >= len(ci.args) {
			in.fail(ci.pos, "«%s» تحتاج معاملاً نصياً", name)
		}
		a, ok := ci.args[i].(string)
		if !ok {
			in.fail(ci.pos, "معامل «%s» يجب أن يكون نصاً", name)
		}
		return a
	}
	switch name {
	case "طول", "عدد":
		return int64(len([]rune(s))), true
	case "قص":
		return strings.TrimSpace(s), true
	case "يحتوي":
		return strings.Contains(s, arg(0)), true
	case "يبدأ_ب", "يبدأ":
		return strings.HasPrefix(s, arg(0)), true
	case "ينتهي_ب", "ينتهي":
		return strings.HasSuffix(s, arg(0)), true
	case "استبدل":
		return strings.ReplaceAll(s, arg(0), arg(1)), true
	case "قسم":
		parts := strings.Split(s, arg(0))
		out := &List{Elems: make([]Value, len(parts))}
		for i, p := range parts {
			out.Elems[i] = p
		}
		return out, true
	case "أحرف":
		out := &List{}
		for _, r := range s {
			out.Elems = append(out.Elems, string(r))
		}
		return out, true
	case "كبير":
		return strings.ToUpper(s), true
	case "صغير":
		return strings.ToLower(s), true
	case "كرر":
		if len(ci.args) != 1 {
			in.fail(ci.pos, "«كرر» تأخذ عدد المرات")
		}
		n := in.intValue(ci.args[0], ci.pos, "عدد مرات «كرر»")
		if n < 0 {
			in.raise(ci.pos, "عدد مرات «كرر» لا يمكن أن يكون سالباً")
		}
		return strings.Repeat(s, int(n)), true
	}
	return nil, false
}

func (in *Interp) numberMethod(v Value, name string, ci *callInfo) (Value, bool) {
	f, _ := toFloat(v)
	switch name {
	case "مطلق":
		if n, ok := v.(int64); ok {
			if n < 0 {
				return -n, true
			}
			return n, true
		}
		return math.Abs(f), true
	case "تقريب":
		if len(ci.args) == 1 {
			d := in.intValue(ci.args[0], ci.pos, "عدد الخانات")
			p := math.Pow(10, float64(d))
			return math.Round(f*p) / p, true
		}
		return int64(math.Round(f)), true
	case "أرضية":
		return int64(math.Floor(f)), true
	case "سقف":
		return int64(math.Ceil(f)), true
	}
	return nil, false
}

// ---------------------------------------------------------------- global built-ins

func (in *Interp) installBuiltins() {
	def := func(name string, fn func(in *Interp, ci *callInfo) Value) {
		in.globals.define(name, &Builtin{Name: name, Fn: fn}, false)
	}
	def("اطبع", func(in *Interp, ci *callInfo) Value {
		if len(ci.named) > 0 {
			in.fail(ci.pos, "«اطبع» لا تقبل معاملات مسماة")
		}
		parts := make([]string, len(ci.args))
		for i, a := range ci.args {
			parts[i] = Format(a)
		}
		fmt.Fprintln(in.Out, strings.Join(parts, " "))
		return nil
	})
	def("مشكلة", func(in *Interp, ci *callInfo) Value {
		in.argCount(ci, 1, "مشكلة")
		return &Problem{Message: Format(ci.args[0])}
	})
	num := func(name string, ci *callInfo) float64 {
		in.argCount(ci, 1, name)
		f, ok := toFloat(ci.args[0])
		if !ok {
			in.fail(ci.pos, "«%s» تحتاج عدداً لكن أُعطيت «%s»", name, typeName(ci.args[0]))
		}
		return f
	}
	def("جذر", func(in *Interp, ci *callInfo) Value {
		f := num("جذر", ci)
		if f < 0 {
			in.raise(ci.pos, "لا يوجد جذر حقيقي لعدد سالب")
		}
		return math.Sqrt(f)
	})
	def("مطلق", func(in *Interp, ci *callInfo) Value {
		if len(ci.args) != 1 {
			in.argCount(ci, 1, "مطلق")
		}
		if n, ok := ci.args[0].(int64); ok {
			if n < 0 {
				return -n
			}
			return n
		}
		return math.Abs(num("مطلق", ci))
	})
	extreme := func(name string, less bool) {
		def(name, func(in *Interp, ci *callInfo) Value {
			args := ci.args
			if len(args) == 1 {
				switch l := args[0].(type) {
				case *List:
					args = l.Elems
				case *Range:
					args = l.values()
				}
			}
			if len(args) == 0 {
				in.raise(ci.pos, "«%s» تحتاج قيمة واحدة على الأقل", name)
			}
			best := args[0]
			for _, a := range args[1:] {
				if in.binop(token.LT, a, best, ci.pos) == less {
					best = a
				}
			}
			return best
		})
	}
	extreme("أدنى", true)
	extreme("أقصى", false)
}
