// Package interp runs Ḍād (ضاد) programs by walking their syntax tree.
//
// It covers the core language: bindings, functions and lambdas, value types
// (نوع), enums (تعداد) with exhaustive matching, optionals, errors, ranges,
// lists, maps and extensions. It also runs the server half of the full-stack
// layer: tables (جدول) through a Store, server functions (خادم) and routes
// (مسار), which "dad dev" serves through the API in server.go. Components
// (مكون، حالة) arrive with the UI layer in Phase 2.
package interp

import (
	"fmt"
	"io"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/token"
)

// ---------------------------------------------------------------- environment

type binding struct {
	val     Value
	mutable bool
	state   bool // a component's «حالة»
}

// Env is a lexical scope.
type Env struct {
	vars   map[string]*binding
	parent *Env
}

func newEnv(parent *Env) *Env { return &Env{vars: map[string]*binding{}, parent: parent} }

func (e *Env) lookup(name string) *binding {
	for s := e; s != nil; s = s.parent {
		if b, ok := s.vars[name]; ok {
			return b
		}
	}
	return nil
}

func (e *Env) define(name string, v Value, mutable bool) bool {
	if _, exists := e.vars[name]; exists {
		return false
	}
	e.vars[name] = &binding{val: v, mutable: mutable}
	return true
}

// ---------------------------------------------------------------- errors and signals

// RuntimeError stops the program. Its message is in Arabic.
type RuntimeError struct {
	Pos  token.Pos
	Msg  string
	File string // the file the position is in, in a program of several files
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("خطأ أثناء التشغيل [السطر %d، العمود %d]: %s",
		e.Pos.Line, e.Pos.Col, e.Msg)
}

type thrown struct {
	val  Value
	pos  token.Pos
	file string
}
type returnSig struct{ val Value }
type breakSig struct{}
type continueSig struct{}

// ---------------------------------------------------------------- interpreter

// Interp holds the state of one program run.
type Interp struct {
	// Out receives everything the program prints.
	Out io.Writer
	// MaxSteps stops runaway programs (0 means no limit).
	MaxSteps int64
	// Notices lists declarations that were skipped because they belong to
	// later phases (مكون، حالة).
	Notices []string
	// Store keeps the rows of every جدول. New uses an in-memory store; dad run
	// and dad dev use a JSON file next to the program.
	Store Store

	steps      int64
	depth      int
	globals    *Env             // the main file's top-level names
	builtins   *Env             // اطبع and the other built-in functions, around every file
	types      map[string]Value // every نوع, جدول and تعداد by name, from all files
	modEnvs    map[*module.Module]*Env
	mainComps  int // components before this index come from imported files
	serverFn   map[string]*Func
	file       string          // the file running now, for error messages
	envFile    map[*Env]string // each file's top-level scope → its path
	ext        map[string]map[string]*Func
	variants   map[string]*Variant
	tables     []*StructType
	routes     []*Route
	serverFns  []string
	components []*Component
	ui         *renderCtx // set while a page is being rendered
}

const maxDepth = 2000

// New returns an interpreter that prints to out.
func New(out io.Writer) *Interp {
	// Interface elements live in a scope around the program's own, so a program
	// can still use their names (عنوان، صف…) for its own things.
	prelude := newEnv(nil)
	builtins := newEnv(prelude)
	in := &Interp{
		Out:      out,
		builtins: builtins,
		globals:  newEnv(builtins),
		types:    map[string]Value{},
		modEnvs:  map[*module.Module]*Env{},
		serverFn: map[string]*Func{},
		envFile:  map[*Env]string{},
		ext:      map[string]map[string]*Func{},
		variants: map[string]*Variant{},
		Store:    NewMemoryStore(),
	}
	in.installBuiltins()
	in.installElements(prelude)
	return in
}

func (in *Interp) fail(pos token.Pos, format string, args ...any) {
	panic(&RuntimeError{Pos: pos, Msg: fmt.Sprintf(format, args...), File: in.file})
}

// raise throws a catchable مشكلة.
func (in *Interp) raise(pos token.Pos, format string, args ...any) {
	panic(thrown{val: &Problem{Message: fmt.Sprintf(format, args...)}, pos: pos, file: in.file})
}

func (in *Interp) step(pos token.Pos) {
	in.steps++
	if in.MaxSteps > 0 && in.steps > in.MaxSteps {
		in.fail(pos, "توقف البرنامج لأنه تجاوز الحد المسموح من الخطوات — هل توجد حلقة لا تنتهي؟")
	}
}

// Run executes a parsed file.
func (in *Interp) Run(file *ast.File) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = in.recovered(r)
		}
	}()

	in.runFile(file, in.globals, nil)
	return nil
}

// RunProgram runs a program of several files: each file once, after the
// files it imports, in a scope of its own.
func (in *Interp) RunProgram(p *module.Program) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = in.recovered(r)
		}
	}()
	for _, m := range p.Modules {
		env := newEnv(in.builtins)
		if m == p.Main {
			env = in.globals
			in.mainComps = len(in.components)
		}
		in.modEnvs[m] = env
		in.envFile[env] = m.Path
		in.file = m.Path
		in.runFile(m.File, env, m)
	}
	return nil
}

// fileOf is the file whose code a closure's scope belongs to.
func (in *Interp) fileOf(env *Env) string {
	if len(in.envFile) == 0 {
		return ""
	}
	for e := env; e != nil; e = e.parent {
		if p, ok := in.envFile[e]; ok {
			return p
		}
	}
	return ""
}

// runFile declares a file's names, then runs its statements.
func (in *Interp) runFile(file *ast.File, env *Env, m *module.Module) {
	for _, s := range file.Stmts {
		if im, ok := s.(*ast.Import); ok {
			in.importNames(im, env, m)
		}
	}
	hoisted := map[ast.Stmt]bool{}
	for _, s := range file.Stmts {
		if _, ok := s.(*ast.Import); ok || in.declare(s, env) {
			hoisted[s] = true
		}
	}
	for _, s := range file.Stmts {
		if !hoisted[s] {
			in.exec(s, env)
		}
	}
}

// importNames binds imported names to the values the other file gave them.
// Importing a تعداد brings its cases.
func (in *Interp) importNames(im *ast.Import, env *Env, m *module.Module) {
	var from *Env
	if m != nil {
		from = in.modEnvs[m.Deps[im]]
	}
	if from == nil {
		in.fail(im.Pos, "الاستيراد يحتاج ملفاً محفوظاً؛ هنا ضع الكود في ملف واحد")
	}
	for _, name := range im.Names {
		b, ok := from.vars[name]
		if !ok {
			in.fail(im.Pos, "«%s» غير موجود في «%s»", name, m.Deps[im].Name())
		}
		in.defineName(env, name, b.val, im.Pos)
		if et, ok := b.val.(*EnumType); ok {
			for _, v := range et.Variants {
				if vb, ok := from.vars[v.Name]; ok && env.vars[v.Name] == nil {
					env.define(v.Name, vb.val, false)
				}
			}
		}
	}
}

// recovered turns a panic from the interpreter into an error; other panics go on.
func (in *Interp) recovered(r any) error {
	switch x := r.(type) {
	case *RuntimeError:
		return x
	case thrown:
		return &RuntimeError{Pos: x.pos, Msg: "مشكلة لم تُعالَج: " + problemMessage(x.val), File: x.file}
	case returnSig:
		return &RuntimeError{Msg: "«أرجع» مستخدمة خارج دالة"}
	case breakSig:
		return &RuntimeError{Msg: "«توقف» مستخدمة خارج حلقة"}
	case continueSig:
		return &RuntimeError{Msg: "«تابع» مستخدمة خارج حلقة"}
	}
	panic(r)
}

func problemMessage(v Value) string {
	if p, ok := v.(*Problem); ok {
		return p.Message
	}
	return Format(v)
}

// declare handles declarations; it reports whether s was one.
func (in *Interp) declare(s ast.Stmt, env *Env) bool {
	switch d := s.(type) {
	case *ast.FuncDecl:
		f := &Func{Name: d.Name, Params: d.Params, Body: d.Body, Env: env, Result: d.Result}
		in.defineName(env, d.Name, f, d.Pos)
		if d.Server {
			in.serverFns = append(in.serverFns, d.Name)
			in.serverFn[d.Name] = f
		}
	case *ast.TableDecl:
		in.declareTable(d, env)
	case *ast.ComponentDecl:
		c := &Component{Name: d.Name, Params: d.Params, Body: d.Body, Env: env, Pos: d.Pos}
		in.defineName(env, d.Name, c, d.Pos)
		in.components = append(in.components, c)
	case *ast.RouteDecl:
		in.routes = append(in.routes, &Route{Method: d.Method, Path: d.Path, Pos: d.Pos, body: d.Body, env: env})
	case *ast.TypeDecl:
		st := &StructType{Name: d.Name, Fields: d.Fields, Methods: map[string]*ast.FuncDecl{}, Env: env}
		for _, m := range d.Methods {
			st.Methods[m.Name] = m
		}
		in.defineName(env, d.Name, st, d.Pos)
		in.types[d.Name] = st
	case *ast.EnumDecl:
		et := &EnumType{Name: d.Name}
		in.defineName(env, d.Name, et, d.Pos)
		in.types[d.Name] = et
		for _, v := range d.Variants {
			variant := &Variant{Enum: et, Name: v.Name, Fields: v.Fields, Env: env}
			et.Variants = append(et.Variants, variant)
			in.variants[v.Name] = variant
			if len(v.Fields) == 0 {
				in.defineName(env, v.Name, &EnumVal{Variant: variant}, v.Pos)
			} else {
				in.defineName(env, v.Name, variant, v.Pos)
			}
		}
	case *ast.ExtensionDecl:
		key := typeKeyOfRef(d.Type)
		if in.ext[key] == nil {
			in.ext[key] = map[string]*Func{}
		}
		for _, m := range d.Methods {
			in.ext[key][m.Name] = &Func{Name: m.Name, Params: m.Params, Body: m.Body, Env: env, Result: m.Result}
		}
	case *ast.ContractDecl:
		// Contracts are checked by the type checker (Phase 1); nothing to run.
	case *ast.Unsupported:
		name := d.Keyword
		if d.Name != "" {
			name += " " + d.Name
		}
		in.Notices = append(in.Notices, "«"+name+"»")
	default:
		return false
	}
	return true
}

func (in *Interp) defineName(env *Env, name string, v Value, pos token.Pos) {
	if !env.define(name, v, false) {
		in.fail(pos, "الاسم «%s» معرّف مسبقاً في هذا النطاق", name)
	}
}

func typeKeyOfRef(t *ast.TypeRef) string {
	switch t.Kind {
	case ast.ListType:
		return "قائمة"
	case ast.MapType:
		return "قاموس"
	case ast.FuncType:
		return "دالة"
	}
	return t.Name
}

func typeKey(v Value) string {
	switch v.(type) {
	case *List, *Range:
		return "قائمة"
	}
	return typeName(v)
}

// ---------------------------------------------------------------- statements

func (in *Interp) exec(s ast.Stmt, env *Env) Value {
	switch s := s.(type) {
	case *ast.ExprStmt:
		return in.eval(s.X, env)
	case *ast.StateDecl:
		in.declareState(s, env)
	case *ast.Let:
		v := in.eval(s.Value, env)
		if s.Type != nil {
			v = in.convert(v, s.Type, s.Pos, "«"+s.Name+"»")
		}
		if !env.define(s.Name, v, s.Mutable) {
			in.fail(s.Pos, "الاسم «%s» معرّف مسبقاً في هذا النطاق", s.Name)
		}
	case *ast.Assign:
		in.assign(s, env)
	case *ast.Return:
		var v Value
		if s.X != nil {
			v = in.eval(s.X, env)
		}
		panic(returnSig{val: v})
	case *ast.Break:
		panic(breakSig{})
	case *ast.Continue:
		panic(continueSig{})
	case *ast.For:
		in.execFor(s, env)
	case *ast.While:
		for {
			in.step(s.Pos)
			if !in.truthy(in.eval(s.Cond, env), s.Cond.P(), "شرط «طالما»") {
				break
			}
			if in.loopBody(s.Body, newEnv(env)) {
				break
			}
		}
	case *ast.Import:
		in.fail(s.Pos, "«استورد» تُكتب في أعلى الملف، لا داخل كتلة")
	default:
		if !in.declare(s, env) {
			in.fail(s.P(), "عبارة غير مدعومة")
		}
	}
	return nil
}

func (in *Interp) assign(s *ast.Assign, env *Env) {
	rhs := func(old Value) Value {
		v := in.eval(s.Value, env)
		switch s.Op {
		case token.PLUS_EQ:
			return in.binop(token.PLUS, old, v, s.Pos)
		case token.MINUS_EQ:
			return in.binop(token.MINUS, old, v, s.Pos)
		case token.STAR_EQ:
			return in.binop(token.STAR, old, v, s.Pos)
		case token.SLASH_EQ:
			return in.binop(token.SLASH, old, v, s.Pos)
		}
		return v
	}
	switch t := s.Target.(type) {
	case *ast.Ident:
		b := env.lookup(t.Name)
		if b == nil {
			in.fail(t.Pos, "«%s» غير معرّف", t.Name)
		}
		if !b.mutable {
			in.fail(t.Pos, "لا يمكن تغيير «%s» لأنه ثابت — عرّفه بـ«متغير» إذا أردت تغييره", t.Name)
		}
		nv := rhs(b.val)
		if _, wasFloat := b.val.(float64); wasFloat {
			if n, isInt := nv.(int64); isInt {
				nv = float64(n) // a عشري variable stays عشري
			}
		}
		b.val = nv
	case *ast.Index:
		coll := in.eval(t.X, env)
		idx := in.eval(t.I, env)
		switch c := coll.(type) {
		case *List:
			i := in.index(idx, len(c.Elems), t.Pos)
			c.Elems[i] = rhs(c.Elems[i])
		case *Map:
			j := c.find(idx)
			var old Value
			if j >= 0 {
				old = c.Vals[j]
			}
			nv := rhs(old)
			if j >= 0 {
				c.Vals[j] = nv
			} else {
				c.Keys = append(c.Keys, idx)
				c.Vals = append(c.Vals, nv)
			}
		default:
			in.fail(t.Pos, "لا يمكن الإسناد بالفهرس إلى قيمة من النوع «%s»", typeName(coll))
		}
	case *ast.Member:
		in.fail(t.Pos, "حقول الأنواع لا تتغير؛ استخدم «نسخة(%s: …)» لإنشاء نسخة معدّلة", t.Name)
	}
}

func (in *Interp) execFor(s *ast.For, env *Env) {
	iter := in.eval(s.Iter, env)
	run := func(item Value) bool {
		in.step(s.Pos)
		child := newEnv(env)
		child.define(s.Var, item, false)
		return in.loopBody(s.Body, child)
	}
	switch it := iter.(type) {
	case *Range:
		if it.Step == 0 {
			in.fail(s.Pos, "الخطوة في «بخطوة» لا يمكن أن تكون صفراً")
		}
		for i := it.From; rangeContinues(it, i); i += it.Step {
			if run(i) {
				return
			}
		}
	case *List:
		items := append([]Value(nil), it.Elems...)
		for _, item := range items {
			if run(item) {
				return
			}
		}
	case string:
		for _, r := range it {
			if run(string(r)) {
				return
			}
		}
	case *Map:
		keys := append([]Value(nil), it.Keys...)
		for _, k := range keys {
			if run(k) {
				return
			}
		}
	default:
		in.fail(s.Iter.P(), "لا يمكن المرور على قيمة من النوع «%s» بـ«لكل»", typeName(iter))
	}
}

func rangeContinues(r *Range, i int64) bool {
	if r.Step > 0 {
		if r.Inclusive {
			return i <= r.To
		}
		return i < r.To
	}
	if r.Inclusive {
		return i >= r.To
	}
	return i > r.To
}

// loopBody runs one iteration and reports whether the loop should stop.
func (in *Interp) loopBody(body *ast.Block, env *Env) (stop bool) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case breakSig:
				stop = true
			case continueSig:
				stop = false
			default:
				panic(r)
			}
		}
	}()
	in.runStmts(body.Stmts, env)
	return false
}

// runStmts executes statements in env and returns the value of the last
// expression statement (blocks are expressions in Ḍād).
func (in *Interp) runStmts(stmts []ast.Stmt, env *Env) Value {
	var last Value
	for _, s := range stmts {
		last = nil
		last = in.exec(s, env)
		if _, isExpr := s.(*ast.ExprStmt); !isExpr {
			last = nil
		}
	}
	return last
}

func (in *Interp) evalBlock(b *ast.Block, env *Env) Value {
	return in.runStmts(b.Stmts, newEnv(env))
}

// ---------------------------------------------------------------- expressions

func (in *Interp) eval(e ast.Expr, env *Env) Value {
	in.step(e.P())
	switch e := e.(type) {
	case *ast.IntLit:
		return e.Value
	case *ast.FloatLit:
		return e.Value
	case *ast.BoolLit:
		return e.Value
	case *ast.NullLit:
		return nil
	case *ast.StrLit:
		var b strings.Builder
		for _, p := range e.Parts {
			if p.Expr != nil {
				b.WriteString(Format(in.eval(p.Expr, env)))
			} else {
				b.WriteString(p.Text)
			}
		}
		return b.String()
	case *ast.Ident:
		return in.lookup(e.Name, e.Pos, env)
	case *ast.ThisExpr:
		if b := env.lookup("هذا"); b != nil {
			return b.val
		}
		in.fail(e.Pos, "«هذا» متاحة فقط داخل دوال الأنواع والامتدادات")
	case *ast.ItExpr:
		if b := env.lookup("هو"); b != nil {
			return b.val
		}
		in.fail(e.Pos, "«هو» متاحة فقط داخل دالة مجهولة تأخذ معاملاً واحداً")
	case *ast.ListLit:
		l := &List{Elems: make([]Value, len(e.Elems))}
		for i, x := range e.Elems {
			l.Elems[i] = in.eval(x, env)
		}
		return l
	case *ast.MapLit:
		m := &Map{}
		for i := range e.Keys {
			k := in.eval(e.Keys[i], env)
			v := in.eval(e.Values[i], env)
			if j := m.find(k); j >= 0 {
				m.Vals[j] = v
			} else {
				m.Keys = append(m.Keys, k)
				m.Vals = append(m.Vals, v)
			}
		}
		return m
	case *ast.Unary:
		return in.unary(e, env)
	case *ast.Binary:
		return in.binary(e, env)
	case *ast.Call:
		return in.call(e, env)
	case *ast.Member:
		recv := in.eval(e.X, env)
		if recv == nil && e.Safe {
			return nil
		}
		return in.member(recv, e.Name, e.Pos)
	case *ast.Index:
		return in.indexExpr(e, env)
	case *ast.Lambda:
		return &Func{IsLam: true, Lambda: e.Params, Body: e.Body, Env: env}
	case *ast.If:
		if in.truthy(in.eval(e.Cond, env), e.Cond.P(), "شرط «إذا»") {
			return in.evalBlock(e.Then, env)
		}
		switch els := e.Else.(type) {
		case *ast.Block:
			return in.evalBlock(els, env)
		case *ast.If:
			return in.eval(els, env)
		}
		return nil
	case *ast.Match:
		return in.match(e, env)
	case *ast.Try:
		return in.try(e, env)
	case *ast.RangeExpr:
		r := &Range{Step: 1, Inclusive: e.Inclusive}
		r.From = in.intValue(in.eval(e.From, env), e.From.P(), "بداية النطاق")
		r.To = in.intValue(in.eval(e.To, env), e.To.P(), "نهاية النطاق")
		if e.Step != nil {
			r.Step = in.intValue(in.eval(e.Step, env), e.Step.P(), "الخطوة")
			if r.Step == 0 {
				in.fail(e.Step.P(), "الخطوة في «بخطوة» لا يمكن أن تكون صفراً")
			}
		}
		return r
	case *ast.TypeTest:
		return in.isType(in.eval(e.X, env), e.Type)
	case *ast.Cast:
		return in.cast(in.eval(e.X, env), e.Type, e.Pos)
	case *ast.Throw:
		v := in.eval(e.X, env)
		if s, ok := v.(string); ok {
			v = &Problem{Message: s}
		}
		panic(thrown{val: v, pos: e.Pos, file: in.file})
	case *ast.Block:
		return in.evalBlock(e, env)
	}
	in.fail(e.P(), "تعبير غير مدعوم")
	return nil
}

func (in *Interp) lookup(name string, pos token.Pos, env *Env) Value {
	if b := env.lookup(name); b != nil {
		return b.val
	}
	// Inside methods and extensions, members of هذا are in scope.
	if self := env.lookup("هذا"); self != nil {
		if v, ok := in.tryMember(self.val, name); ok {
			return v
		}
	}
	in.fail(pos, "«%s» غير معرّف", name)
	return nil
}

func (in *Interp) truthy(v Value, pos token.Pos, what string) bool {
	b, ok := v.(bool)
	if !ok {
		in.fail(pos, "%s يجب أن يكون «صح» أو «خطأ» لكنه من النوع «%s»", what, typeName(v))
	}
	return b
}

func (in *Interp) intValue(v Value, pos token.Pos, what string) int64 {
	n, ok := v.(int64)
	if !ok {
		in.fail(pos, "%s يجب أن تكون عدداً صحيحاً لكنها من النوع «%s»", what, typeName(v))
	}
	return n
}

func (in *Interp) index(idx Value, n int, pos token.Pos) int {
	i := in.intValue(idx, pos, "الفهرس")
	if i < 0 || i >= int64(n) {
		in.raise(pos, "الفهرس %s خارج الحدود (الطول %s)", formatInt(i), formatInt(int64(n)))
	}
	return int(i)
}

func (in *Interp) indexExpr(e *ast.Index, env *Env) Value {
	coll := in.eval(e.X, env)
	idx := in.eval(e.I, env)
	switch c := coll.(type) {
	case *List:
		return c.Elems[in.index(idx, len(c.Elems), e.Pos)]
	case *Range:
		vals := c.values()
		return vals[in.index(idx, len(vals), e.Pos)]
	case string:
		r := []rune(c)
		return string(r[in.index(idx, len(r), e.Pos)])
	case *Map:
		if j := c.find(idx); j >= 0 {
			return c.Vals[j]
		}
		return nil
	case nil:
		in.fail(e.Pos, "لا يمكن الفهرسة في «عدم»")
	}
	in.fail(e.Pos, "لا يمكن الفهرسة في قيمة من النوع «%s»", typeName(coll))
	return nil
}

func (in *Interp) unary(e *ast.Unary, env *Env) Value {
	v := in.eval(e.X, env)
	switch e.Op {
	case token.MINUS:
		switch n := v.(type) {
		case int64:
			return -n
		case float64:
			return -n
		}
		in.fail(e.Pos, "لا يمكن تطبيق «-» على قيمة من النوع «%s»", typeName(v))
	default: // ليس / !
		return !in.truthy(v, e.Pos, "ما بعد «ليس»")
	}
	return nil
}

func (in *Interp) binary(e *ast.Binary, env *Env) Value {
	switch e.Op {
	case token.AND, token.AND_SYM:
		if !in.truthy(in.eval(e.L, env), e.L.P(), "طرف «و»") {
			return false
		}
		return in.truthy(in.eval(e.R, env), e.R.P(), "طرف «و»")
	case token.OR, token.OR_SYM:
		if in.truthy(in.eval(e.L, env), e.L.P(), "طرف «أو»") {
			return true
		}
		return in.truthy(in.eval(e.R, env), e.R.P(), "طرف «أو»")
	case token.COALESCE:
		if l := in.eval(e.L, env); l != nil {
			return l
		}
		return in.eval(e.R, env)
	}
	return in.binop(e.Op, in.eval(e.L, env), in.eval(e.R, env), e.Pos)
}

var opNames = map[token.Kind]string{
	token.PLUS: "+", token.MINUS: "-", token.STAR: "*", token.SLASH: "/", token.PERCENT: "%",
	token.LT: "<", token.GT: ">", token.LE: "<=", token.GE: ">=",
}

func (in *Interp) binop(op token.Kind, a, b Value, pos token.Pos) Value {
	switch op {
	case token.EQ:
		return equal(a, b)
	case token.NEQ:
		return !equal(a, b)
	}
	if op == token.PLUS {
		if s, ok := a.(string); ok {
			return s + Format(b)
		}
		if s, ok := b.(string); ok {
			return Format(a) + s
		}
		if la, ok := a.(*List); ok {
			if lb, ok := b.(*List); ok {
				return &List{Elems: append(append([]Value(nil), la.Elems...), lb.Elems...)}
			}
		}
	}
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			switch op {
			case token.LT:
				return sa < sb
			case token.GT:
				return sa > sb
			case token.LE:
				return sa <= sb
			case token.GE:
				return sa >= sb
			}
		}
	}
	ia, aInt := a.(int64)
	ib, bInt := b.(int64)
	if aInt && bInt {
		switch op {
		case token.PLUS:
			return ia + ib
		case token.MINUS:
			return ia - ib
		case token.STAR:
			return ia * ib
		case token.SLASH:
			// Division always gives a decimal, as in school maths: 7 / 2 = 3.5.
			// For a whole result: (7 / 2) ك صحيح.
			if ib == 0 {
				in.raise(pos, "قسمة على صفر")
			}
			return float64(ia) / float64(ib)
		case token.PERCENT:
			if ib == 0 {
				in.raise(pos, "قسمة على صفر")
			}
			return ia % ib
		case token.LT:
			return ia < ib
		case token.GT:
			return ia > ib
		case token.LE:
			return ia <= ib
		case token.GE:
			return ia >= ib
		}
	}
	fa, aNum := toFloat(a)
	fb, bNum := toFloat(b)
	if aNum && bNum {
		switch op {
		case token.PLUS:
			return fa + fb
		case token.MINUS:
			return fa - fb
		case token.STAR:
			return fa * fb
		case token.SLASH:
			if fb == 0 {
				in.raise(pos, "قسمة على صفر")
			}
			return fa / fb
		case token.PERCENT:
			if fb == 0 {
				in.raise(pos, "قسمة على صفر")
			}
			return fa - fb*float64(int64(fa/fb))
		case token.LT:
			return fa < fb
		case token.GT:
			return fa > fb
		case token.LE:
			return fa <= fb
		case token.GE:
			return fa >= fb
		}
	}
	in.fail(pos, "لا يمكن تطبيق «%s» على «%s» و«%s»", opNames[op], typeName(a), typeName(b))
	return nil
}

func toFloat(v Value) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// ---------------------------------------------------------------- match and try

func (in *Interp) match(m *ast.Match, env *Env) Value {
	subject := in.eval(m.Subject, env)
	for _, c := range m.Cases {
		scope := newEnv(env)
		if !in.matches(c.Pattern, subject, scope) {
			continue
		}
		if b, ok := c.Body.(*ast.Block); ok {
			return in.runStmts(b.Stmts, scope)
		}
		return in.eval(c.Body, scope)
	}
	in.fail(m.Pos, "لا توجد حالة في «طابق» تطابق القيمة «%s»", Format(subject))
	return nil
}

func (in *Interp) matches(p *ast.Pattern, subject Value, scope *Env) bool {
	switch p.Kind {
	case ast.WildcardPattern:
		return true
	case ast.LiteralPattern:
		return equal(in.eval(p.Lit, scope), subject)
	}
	if variant, ok := in.variants[p.Name]; ok {
		ev, ok := subject.(*EnumVal)
		if !ok || ev.Variant != variant {
			return false
		}
		if len(p.Binds) > len(ev.Vals) {
			in.fail(p.Pos, "الحالة «%s» تحتوي %s حقول لكن النمط يذكر %s", p.Name,
				formatInt(int64(len(ev.Vals))), formatInt(int64(len(p.Binds))))
		}
		for i, name := range p.Binds {
			if name != "_" {
				scope.define(name, ev.Vals[i], false)
			}
		}
		return true
	}
	if p.HasArgs {
		in.fail(p.Pos, "«%s» ليست حالة معروفة في أي تعداد", p.Name)
	}
	scope.define(p.Name, subject, false)
	return true
}

func (in *Interp) try(t *ast.Try, env *Env) (result Value) {
	defer func() {
		if r := recover(); r != nil {
			th, ok := r.(thrown)
			if !ok {
				panic(r)
			}
			scope := newEnv(env)
			scope.define(t.Var, th.val, false)
			result = in.runStmts(t.Catch.Stmts, scope)
		}
	}()
	return in.evalBlock(t.Body, env)
}

// ---------------------------------------------------------------- types

func (in *Interp) isType(v Value, t *ast.TypeRef) bool {
	if v == nil {
		return t.Optional
	}
	switch t.Kind {
	case ast.ListType:
		switch v.(type) {
		case *List, *Range:
			return true
		}
		return false
	case ast.MapType:
		_, ok := v.(*Map)
		return ok
	case ast.FuncType:
		switch v.(type) {
		case *Func, *Builtin:
			return true
		}
		return false
	}
	switch t.Name {
	case "صحيح":
		_, ok := v.(int64)
		return ok
	case "عشري":
		_, ok := v.(float64)
		return ok
	case "نص":
		_, ok := v.(string)
		return ok
	case "منطقي":
		_, ok := v.(bool)
		return ok
	case "مشكلة":
		_, ok := v.(*Problem)
		return ok
	case "قائمة":
		switch v.(type) {
		case *List, *Range:
			return true
		}
		return false
	case "أي", "قيمة":
		return true
	}
	if variant, ok := in.variants[t.Name]; ok {
		ev, ok := v.(*EnumVal)
		return ok && ev.Variant == variant
	}
	if tv := in.types[t.Name]; tv != nil {
		switch tt := tv.(type) {
		case *StructType:
			s, ok := v.(*Struct)
			return ok && s.Type == tt
		case *EnumType:
			ev, ok := v.(*EnumVal)
			return ok && ev.Variant.Enum == tt
		}
	}
	return false
}

func typeString(t *ast.TypeRef) string {
	var s string
	switch t.Kind {
	case ast.ListType:
		s = "[" + typeString(t.Elem) + "]"
	case ast.MapType:
		s = "[" + typeString(t.Key) + ": " + typeString(t.Elem) + "]"
	case ast.FuncType:
		parts := make([]string, len(t.Args))
		for i, a := range t.Args {
			parts[i] = typeString(a)
		}
		s = "(" + strings.Join(parts, "، ") + ") => " + typeString(t.Elem)
	default:
		s = t.Name
	}
	if t.Optional {
		s += "؟"
	}
	return s
}

// knownType reports whether name is a type the interpreter can check.
func (in *Interp) knownType(name string) bool {
	switch name {
	case "صحيح", "عشري", "نص", "منطقي", "مشكلة", "قائمة":
		return true
	}
	if _, ok := in.variants[name]; ok {
		return true
	}
	if tv := in.types[name]; tv != nil {
		switch tv.(type) {
		case *StructType, *EnumType:
			return true
		}
	}
	return false
}

// convert checks v against a declared type and widens صحيح to عشري.
func (in *Interp) convert(v Value, t *ast.TypeRef, pos token.Pos, what string) Value {
	if t == nil {
		return v
	}
	if v == nil {
		if t.Optional || (t.Kind == ast.NamedType && !in.knownType(t.Name)) {
			return nil
		}
		in.fail(pos, "%s لا يقبل «عدم»؛ استخدم النوع «%s؟» إذا كانت القيمة اختيارية", what, typeString(t))
	}
	if t.Kind == ast.NamedType {
		if t.Name == "عشري" {
			if n, ok := v.(int64); ok {
				return float64(n)
			}
		}
		if !in.knownType(t.Name) {
			return v // contracts, generic names: checked by the future type checker
		}
	}
	if !in.isType(v, t) {
		in.fail(pos, "%s: توقعت قيمة من النوع «%s» لكن وجدت «%s»", what, typeString(t), typeName(v))
	}
	return v
}

func (in *Interp) cast(v Value, t *ast.TypeRef, pos token.Pos) Value {
	if v == nil {
		if t.Optional {
			return nil
		}
		in.fail(pos, "لا يمكن تحويل «عدم» إلى «%s»", typeString(t))
	}
	switch t.Name {
	case "نص":
		return Format(v)
	case "عشري":
		switch n := v.(type) {
		case int64:
			return float64(n)
		case float64:
			return n
		case string:
			if f, ok := parseNumber(n); ok {
				return f
			}
			in.raise(pos, "لا يمكن تحويل «%s» إلى عدد عشري", n)
		}
	case "صحيح":
		switch n := v.(type) {
		case int64:
			return n
		case float64:
			return int64(n)
		case string:
			if f, ok := parseNumber(n); ok && f == float64(int64(f)) {
				return int64(f)
			}
			in.raise(pos, "لا يمكن تحويل «%s» إلى عدد صحيح", n)
		}
	case "منطقي":
		if b, ok := v.(bool); ok {
			return b
		}
	default:
		if in.isType(v, t) {
			return v
		}
	}
	in.fail(pos, "لا يمكن تحويل قيمة من النوع «%s» إلى «%s»", typeName(v), typeString(t))
	return nil
}

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(strings.Map(func(r rune) rune {
		switch {
		// Text typed on an Arabic or Persian keyboard still converts to a number.
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r == '٫':
			return '.'
		case r == '٬' || r == '_':
			return -1
		}
		return r
	}, s))
	var f float64
	_, err := fmt.Sscanf(s, "%g", &f)
	if err != nil || s == "" {
		return 0, false
	}
	// Reject trailing garbage such as "12abc".
	if fmt.Sprintf("%g", f) != s && strings.TrimLeft(s, "+-0123456789.eE") != "" {
		return 0, false
	}
	return f, true
}

// BuiltinNames lists the names every program starts with: functions such as
// اطبع and the interface elements.
func BuiltinNames() []string {
	in := New(io.Discard)
	var out []string
	for e := in.builtins; e != nil; e = e.parent {
		for name := range e.vars {
			out = append(out, name)
		}
	}
	return out
}
