// Package check finds mistakes in Ḍād (ضاد) programs before they run: names
// that are not defined, calls with the wrong arguments, values of the wrong
// type, fields that don't exist, and «طابق» that misses cases of a تعداد.
//
// It reports only what it is sure of. When it can't tell a value's type, it
// says nothing, so a program that runs correctly is never rejected. Messages
// are in Arabic, like the rest of the toolchain.
package check

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/parser"
	"github.com/mtwalli/dad-lang/internal/token"
)

// File checks a parsed file and returns its problems, in source order.
func File(f *ast.File) []lexer.Diagnostic {
	c := newChecker()
	c.file(f)
	sort.SliceStable(c.diags, func(i, j int) bool {
		a, b := c.diags[i].Pos, c.diags[j].Pos
		return a.Line < b.Line || a.Line == b.Line && a.Col < b.Col
	})
	return c.diags
}

// ---------------------------------------------------------------- symbols and scopes

type symKind int

const (
	sVar symKind = iota
	sFunc
	sType
	sEnum
	sVariant
	sComponent
	sBuiltin
	sElement
)

type symbol struct {
	kind    symKind
	typ     *Type
	mutable bool
	decl    *ast.FuncDecl // sFunc
	params  []*ast.Param  // sFunc, sType, sVariant, sComponent: what a call takes
	result  *Type         // what a call gives
	what    string        // how messages name it: «مربع»، المكون «عداد»
	def     *Def          // where it is declared, when recording for editors
}

type scope struct {
	names  map[string]*symbol
	parent *scope
}

func (s *scope) lookup(name string) *symbol {
	for x := s; x != nil; x = x.parent {
		if sym, ok := x.names[name]; ok {
			return sym
		}
	}
	return nil
}

func (s *scope) child() *scope { return &scope{names: map[string]*symbol{}, parent: s} }

func (s *scope) visible() []string {
	var out []string
	for x := s; x != nil; x = x.parent {
		for n := range x.names {
			out = append(out, n)
		}
	}
	return out
}

// frame is the function, lambda or component whose body is being checked.
type frame struct {
	result    *Type // declared result of a function
	name      string
	lambda    bool
	component bool  // inside a مكون (lambdas in it count too)
	self      *Type // هذا, in methods and extensions
	hasSelf   bool
	it        bool // هو is available
	loops     int
}

type structInfo struct {
	name    string
	fields  []*ast.Param
	ftypes  map[string]*Type
	methods map[string]*ast.FuncDecl
	table   bool
}

type enumInfo struct {
	name     string
	variants []*ast.Variant
}

type checker struct {
	diags     []lexer.Diagnostic
	quiet     int // >0: inferring a type, don't report
	global    *scope
	types     map[string]*Type // user types by name: نوع, جدول (Struct) and تعداد (Enum)
	structs   map[string]*structInfo
	enums     map[string]*enumInfo
	variantOf map[string]string // variant name → enum name
	contracts map[string]bool
	ext       map[string]map[string]*ast.FuncDecl // extension methods by type key
	inferring map[*ast.FuncDecl]bool
	results   map[*ast.FuncDecl]*Type
	deferred  []func()
	fr        *frame

	// programs of several files
	mod      *module.Module             // the file being checked; nil for a lone source text
	scopes   map[*module.Module]*scope  // each checked file's top-level names
	imported map[*scope]map[string]bool // names a file imported rather than declared
	owners   map[string]*module.Module  // file that declares each type, table, enum, case and server function
	addrs    map[string]addrUse         // the addresses of pages and «جلب» routes, to find two at one address

	// for editors (Analyze): definitions of functions, members and parameters
	info       *Info
	funcDefs   map[*ast.FuncDecl]*Def
	memberDefs map[string]map[string]*Def // type name → field or method → definition
	paramDefs  map[*ast.Param]*Def
}

func newChecker() *checker {
	c := &checker{
		global:     &scope{names: map[string]*symbol{}},
		types:      map[string]*Type{},
		structs:    map[string]*structInfo{},
		enums:      map[string]*enumInfo{},
		variantOf:  map[string]string{},
		contracts:  map[string]bool{},
		ext:        map[string]map[string]*ast.FuncDecl{},
		inferring:  map[*ast.FuncDecl]bool{},
		results:    map[*ast.FuncDecl]*Type{},
		fr:         &frame{},
		funcDefs:   map[*ast.FuncDecl]*Def{},
		memberDefs: map[string]map[string]*Def{},
		paramDefs:  map[*ast.Param]*Def{},
		scopes:     map[*module.Module]*scope{},
		imported:   map[*scope]map[string]bool{},
		owners:     map[string]*module.Module{},
		addrs:      map[string]addrUse{},
	}
	// Built-ins live in a scope around the program's own, like in the interpreter.
	prelude := &scope{names: map[string]*symbol{}}
	for _, name := range interp.BuiltinNames() {
		sym := &symbol{kind: sBuiltin, what: "«" + name + "»"}
		switch name {
		case "اطبع":
			sym.result = tNull
		case "مشكلة":
			sym.result = tProb
		case "جذر":
			sym.result = tFloat
		}
		prelude.names[name] = sym
	}
	for _, name := range interp.Elements {
		prelude.names[name] = &symbol{kind: sElement, result: tNull, what: "«" + name + "»"}
	}
	c.global.parent = prelude
	return c
}

func (c *checker) errorf(pos token.Pos, format string, args ...any) {
	if c.quiet > 0 {
		return
	}
	c.diags = append(c.diags, lexer.Diagnostic{Pos: pos, Msg: fmt.Sprintf(format, args...)})
}

func (c *checker) define(s *scope, name string, sym *symbol, pos token.Pos) {
	if _, exists := s.names[name]; exists {
		c.errorf(pos, "الاسم «%s» معرّف مسبقاً في هذا النطاق", name)
		return
	}
	s.names[name] = sym
}

// ---------------------------------------------------------------- the file

func (c *checker) file(f *ast.File) {
	// Declarations are visible everywhere, as in the interpreter.
	for _, s := range f.Stmts {
		c.collect(s)
	}
	for _, s := range f.Stmts {
		c.collectMembers(s)
	}
	c.fr = &frame{}
	for _, s := range f.Stmts {
		c.topLevel(s)
	}
	// Bodies run later, when every top-level name exists.
	for len(c.deferred) > 0 {
		d := c.deferred[0]
		c.deferred = c.deferred[1:]
		d()
	}
}

func (c *checker) collect(s ast.Stmt) {
	g := c.global
	switch d := s.(type) {
	case *ast.FuncDecl:
		if d.Server {
			c.claim(d.Name, d.Pos, "دالة الخادم")
		}
		fd := c.funcDef(d, "function", nil)
		c.define(g, d.Name, &symbol{kind: sFunc, decl: d, params: d.Params, typ: tFunc, what: "«" + d.Name + "»", def: fd}, d.Pos)
	case *ast.TypeDecl:
		c.claim(d.Name, d.Pos, "النوع")
		c.types[d.Name] = &Type{Kind: Struct, Name: d.Name}
		c.structs[d.Name] = &structInfo{name: d.Name, fields: d.Fields, methods: map[string]*ast.FuncDecl{}}
		td := c.def(d.Name, c.nameAfter(d.Pos, d.Name), "type", "نوع "+d.Name+paramsSig(d.Fields), &Type{Kind: TypeValue, Name: d.Name}, nil)
		c.fieldDefs(td, d.Name, d.Fields)
		for _, m := range d.Methods {
			c.memberDefs[d.Name][m.Name] = c.funcDef(m, "method", td)
		}
		c.define(g, d.Name, &symbol{kind: sType, typ: &Type{Kind: TypeValue, Name: d.Name}, params: d.Fields,
			result: c.types[d.Name], what: "«" + d.Name + "»", def: td}, d.Pos)
	case *ast.TableDecl:
		fields := append([]*ast.Param{}, d.Fields...)
		fields = append(fields, &ast.Param{At: d.At, Name: "رقم", Type: &ast.TypeRef{Kind: ast.NamedType, Name: "صحيح"}, Default: &ast.IntLit{}})
		c.claim(d.Name, d.Pos, "الجدول")
		c.types[d.Name] = &Type{Kind: Struct, Name: d.Name}
		c.structs[d.Name] = &structInfo{name: d.Name, fields: fields, methods: map[string]*ast.FuncDecl{}, table: true}
		td := c.def(d.Name, c.nameAfter(d.Pos, d.Name), "table", "جدول "+d.Name+paramsSig(d.Fields), &Type{Kind: TypeValue, Name: d.Name}, nil)
		c.fieldDefs(td, d.Name, d.Fields)
		c.define(g, d.Name, &symbol{kind: sType, typ: &Type{Kind: TypeValue, Name: d.Name}, params: fields,
			result: c.types[d.Name], what: "«" + d.Name + "»", def: td}, d.Pos)
	case *ast.EnumDecl:
		c.claim(d.Name, d.Pos, "التعداد")
		c.types[d.Name] = &Type{Kind: Enum, Name: d.Name}
		c.enums[d.Name] = &enumInfo{name: d.Name, variants: d.Variants}
		ed := c.def(d.Name, c.nameAfter(d.Pos, d.Name), "enum", "تعداد "+d.Name, &Type{Kind: EnumType, Name: d.Name}, nil)
		c.define(g, d.Name, &symbol{kind: sEnum, typ: &Type{Kind: EnumType, Name: d.Name}, def: ed}, d.Pos)
		for _, v := range d.Variants {
			c.claim(v.Name, v.Pos, "الحالة")
			c.variantOf[v.Name] = d.Name
			vd := c.def(v.Name, c.nameAfter(v.Pos, v.Name), "case", v.Name+paramsSig(v.Fields)+" — "+d.Name, c.types[d.Name], ed)
			if vd != nil {
				vd.Global = true // a case is used by its own name everywhere
			}
			for _, f := range v.Fields {
				if fd := c.def(f.Name, c.nameAfter(f.Pos, f.Name), "field", f.Name+": "+refString(f.Type), nil, vd); fd != nil {
					c.paramDefs[f] = fd
				}
			}
			sym := &symbol{kind: sVariant, params: v.Fields, result: c.types[d.Name], what: "«" + v.Name + "»", def: vd}
			if len(v.Fields) == 0 {
				sym.typ = c.types[d.Name]
			} else {
				sym.typ = tFunc
			}
			c.define(g, v.Name, sym, v.Pos)
		}
	case *ast.ContractDecl:
		c.contracts[d.Name] = true
	case *ast.ComponentDecl:
		cd := c.def(d.Name, c.nameAfter(d.Pos, d.Name), "component", "مكون "+d.Name+paramsSig(d.Params), &Type{Kind: Component}, nil)
		c.paramDefsFor(d.Params, cd)
		c.define(g, d.Name, &symbol{kind: sComponent, typ: &Type{Kind: Component}, params: d.Params, result: tNull,
			what: "المكون «" + d.Name + "»", def: cd}, d.Pos)
	case *ast.Import:
		c.importDecl(d)
	}
}

// funcDef records a function or method and its parameters.
func (c *checker) funcDef(d *ast.FuncDecl, kind string, parent *Def) *Def {
	fd := c.def(d.Name, c.nameAfter(d.Pos, d.Name), kind, funcSig(d), tFunc, parent)
	if fd != nil {
		c.funcDefs[d] = fd
		c.paramDefsFor(d.Params, fd)
	}
	return fd
}

// paramDefsFor records parameters, so named arguments can point at them.
func (c *checker) paramDefsFor(params []*ast.Param, owner *Def) {
	if owner == nil {
		return
	}
	for _, p := range params {
		t, _ := c.fromRef(p.Type)
		pd := &Def{Name: p.Name, Pos: p.Pos, Kind: "parameter", Detail: p.Name + ": " + refString(p.Type), Type: t}
		c.info.Defs = append(c.info.Defs, pd)
		c.paramDefs[p] = pd
	}
}

// fieldDefs records the fields of a type or table as its children.
func (c *checker) fieldDefs(owner *Def, typeName string, fields []*ast.Param) {
	if c.memberDefs[typeName] == nil {
		c.memberDefs[typeName] = map[string]*Def{}
	}
	if owner == nil {
		return
	}
	for _, f := range fields {
		detail := f.Name
		if f.Type != nil {
			detail += ": " + refString(f.Type)
		}
		fd := c.def(f.Name, f.Pos, "field", detail, nil, owner)
		c.memberDefs[typeName][f.Name] = fd
		c.paramDefs[f] = fd
	}
}

// collectMembers records fields, methods and extensions, once all type names are known.
func (c *checker) collectMembers(s ast.Stmt) {
	switch d := s.(type) {
	case *ast.TypeDecl:
		info := c.structs[d.Name]
		info.ftypes = c.fieldTypes(d.Fields, false)
		c.setFieldTypes(d.Name, info.ftypes)
		for _, m := range d.Methods {
			info.methods[m.Name] = m
		}
		for _, ct := range d.Contracts {
			c.checkRef(ct)
		}
	case *ast.TableDecl:
		info := c.structs[d.Name]
		info.ftypes = c.fieldTypes(info.fields, true)
		c.setFieldTypes(d.Name, info.ftypes)
	case *ast.EnumDecl:
		for _, v := range d.Variants {
			c.fieldTypes(v.Fields, false)
		}
	case *ast.ExtensionDecl:
		t := c.checkRef(d.Type)
		key := extKey(d.Type, t)
		if c.ext[key] == nil {
			c.ext[key] = map[string]*ast.FuncDecl{}
		}
		for _, m := range d.Methods {
			c.ext[key][m.Name] = m
			c.funcDef(m, "method", nil)
		}
	}
}

func (c *checker) setFieldTypes(typeName string, types map[string]*Type) {
	for name, d := range c.memberDefs[typeName] {
		if d != nil && d.Kind == "field" {
			d.Type = types[name]
			if t := types[name]; t != nil && !strings.Contains(d.Detail, ":") {
				d.Detail += ": " + t.String()
			}
		}
	}
}

// fieldTypes checks the types of fields and returns them by name. Table fields
// without a type take their default's type.
func (c *checker) fieldTypes(fields []*ast.Param, table bool) map[string]*Type {
	out := map[string]*Type{}
	for _, f := range fields {
		var t *Type
		if f.Type != nil {
			t = c.checkRef(f.Type)
		} else if table && f.Default != nil {
			c.quiet++
			t = c.expr(f.Default, c.global)
			c.quiet--
		}
		out[f.Name] = t
	}
	return out
}

func extKey(r *ast.TypeRef, t *Type) string {
	switch r.Kind {
	case ast.ListType:
		return "قائمة"
	case ast.MapType:
		return "قاموس"
	case ast.FuncType:
		return "دالة"
	}
	return r.Name
}

func typeKey(t *Type) string {
	if t == nil {
		return ""
	}
	switch t.Kind {
	case List, Range:
		return "قائمة"
	case Map:
		return "قاموس"
	}
	return strings.TrimSuffix(t.String(), "؟")
}

// topLevel checks a top-level statement. Declarations' bodies are deferred.
func (c *checker) topLevel(s ast.Stmt) {
	g := c.global
	switch d := s.(type) {
	case *ast.FuncDecl:
		c.funcDecl(d, g, nil, false)
	case *ast.TypeDecl:
		self := c.types[d.Name]
		for _, m := range d.Methods {
			c.funcDecl(m, g, self, true)
		}
		c.paramDefaults(d.Fields, g)
	case *ast.TableDecl:
		c.paramDefaults(d.Fields, g)
	case *ast.EnumDecl:
	case *ast.ContractDecl:
	case *ast.ExtensionDecl:
		self, _ := c.fromRef(d.Type)
		for _, m := range d.Methods {
			c.funcDecl(m, g, self, true)
		}
	case *ast.ComponentDecl:
		c.componentDecl(d)
	case *ast.RouteDecl:
		c.routeDecl(d)
	case *ast.PageDecl:
		c.pageDecl(d)
	case *ast.Import:
	default:
		c.stmt(s, g)
	}
}

func (c *checker) paramDefaults(params []*ast.Param, s *scope) {
	for _, p := range params {
		if p.Default == nil {
			continue
		}
		pt, _ := c.fromRef(p.Type)
		dt := c.expr(p.Default, s)
		if p.Type != nil && !assignable(dt, pt) {
			c.errorf(p.Default.P(), "القيمة الافتراضية لـ«%s» من النوع «%s»، لكن نوعه «%s»", p.Name, dt, pt)
		}
	}
}

// funcDecl checks a function's signature now and its body later.
func (c *checker) funcDecl(d *ast.FuncDecl, outer *scope, self *Type, method bool) {
	for _, p := range d.Params {
		c.checkRef(p.Type)
	}
	result := c.checkRef(d.Result)
	c.deferred = append(c.deferred, func() {
		s := outer.child()
		fr := &frame{result: result, name: d.Name, self: self, hasSelf: method}
		if method && self != nil && self.Kind == Struct {
			if info := c.structs[self.Name]; info != nil {
				for _, f := range info.fields {
					s.names[f.Name] = &symbol{kind: sVar, typ: info.ftypes[f.Name], def: c.memberDefs[self.Name][f.Name]}
				}
			}
		}
		s = s.child()
		c.params(d.Params, s, outer)
		c.withFrame(fr, func() {
			t := c.body(d.Body, s)
			if _, isBlock := d.Body.(*ast.Block); isBlock {
				if result != nil && !result.Optional && !returnsOrThrows(d.Body) {
					c.errorf(d.Pos, "الدالة «%s» يجب أن ترجع قيمة من النوع «%s» — هل نسيت «أرجع»؟", d.Name, result)
				}
			} else if d.Result != nil && !assignable(t, result) {
				c.errorf(d.Body.P(), "الدالة «%s» ترجع «%s»، لكنها تعطي «%s»", d.Name, result, t)
			}
		})
	})
}

// params defines a function's parameters and checks their defaults.
func (c *checker) params(params []*ast.Param, s, outer *scope) {
	for _, p := range params {
		t, _ := c.fromRef(p.Type)
		if p.Default != nil {
			dt := c.expr(p.Default, s)
			if p.Type != nil && !assignable(dt, t) {
				c.errorf(p.Default.P(), "القيمة الافتراضية لـ«%s» من النوع «%s»، لكن نوعه «%s»", p.Name, dt, t)
			}
		}
		pd := c.paramDefs[p]
		if pd == nil {
			pd = c.def(p.Name, p.Pos, "parameter", p.Name+": "+refString(p.Type), t, nil)
		}
		c.define(s, p.Name, &symbol{kind: sVar, typ: t, def: pd}, p.Pos)
	}
}

func (c *checker) withFrame(fr *frame, fn func()) {
	saved := c.fr
	c.fr = fr
	defer func() { c.fr = saved }()
	fn()
}

func (c *checker) componentDecl(d *ast.ComponentDecl) {
	for _, p := range d.Params {
		c.checkRef(p.Type)
	}
	c.deferred = append(c.deferred, func() {
		s := c.global.child()
		c.params(d.Params, s, c.global)
		c.withFrame(&frame{name: d.Name, component: true}, func() { c.block(d.Body, s.child()) })
	})
}

var pathParam = regexp.MustCompile(`\{([^{}/]+)\}`)

// addrUse is a page or a «جلب» route at an address.
type addrUse struct {
	what string // "الصفحة" or "المسار"
	mod  *module.Module
	line int
}

// address checks a path's {names} and records it, so that two pages, or a page and a
// «جلب» route, can't answer at the same address. «/مهمة/{رقم}» and «/مهمة/{ر}/» are
// the same address.
func (c *checker) address(path string, pos token.Pos, what string) []string {
	var names []string
	seen := map[string]bool{}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	for i, seg := range segs {
		m := pathParam.FindStringSubmatch(seg)
		if m == nil {
			if strings.ContainsAny(seg, "{}") {
				c.errorf(pos, "في العنوان «%s» جزء غير صحيح: «%s»؛ المتغير يُكتب وحده بين جزأين، مثل: \"/مهمة/{رقم}\"", path, seg)
			}
			continue
		}
		name := strings.TrimSpace(m[1])
		if m[0] != seg || !isName(name) {
			c.errorf(pos, "في العنوان «%s» جزء غير صحيح: «%s»؛ المتغير يُكتب وحده بين جزأين، مثل: \"/مهمة/{رقم}\"", path, seg)
			continue
		}
		if seen[name] {
			c.errorf(pos, "المتغير «%s» مكرر في العنوان «%s»", name, path)
		}
		seen[name] = true
		names = append(names, name)
		segs[i] = "{}"
	}
	key := "/" + strings.Join(segs, "/")
	if prev, ok := c.addrs[key]; ok {
		where := "السطر " + strconv.Itoa(prev.line)
		if prev.mod != nil && prev.mod != c.mod {
			where = "«" + prev.mod.Name() + "»، " + where
		}
		c.errorf(pos, "%s «%s» بالعنوان نفسه لـ%s في %s", what, path, prev.what, where)
	} else {
		c.addrs[key] = addrUse{what: what, mod: c.mod, line: pos.Line}
	}
	return names
}

// isName reports whether s can be a variable name.
func isName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(unicode.IsLetter(r) || r == '_' || (i > 0 && unicode.IsDigit(r)) || unicode.Is(unicode.Mn, r)) {
			return false
		}
	}
	return true
}

// pageDecl checks «صفحة "/…" = …»: the body lays out the page like a component's,
// with the address's {names} as values.
func (c *checker) pageDecl(d *ast.PageDecl) {
	names := c.address(d.Path, d.PathPos, "الصفحة")
	c.def(d.Path, d.PathPos, "page", "صفحة "+d.Path, nil, nil) // for the outline of editors
	c.deferred = append(c.deferred, func() {
		s := c.global.child()
		for _, n := range names {
			s.names[n] = &symbol{kind: sVar, what: "«" + n + "»"}
		}
		c.withFrame(&frame{name: "صفحة " + d.Path, component: true}, func() { c.body(d.Body, s) })
	})
}

func (c *checker) routeDecl(d *ast.RouteDecl) {
	if d.Method == "GET" {
		c.address(d.Path, d.Pos, "المسار")
	}
	c.deferred = append(c.deferred, func() {
		s := c.global.child()
		for _, m := range pathParam.FindAllStringSubmatch(d.Path, -1) {
			s.names[strings.TrimSpace(m[1])] = &symbol{kind: sVar}
		}
		s.names["البيانات"] = &symbol{kind: sVar}
		c.withFrame(&frame{lambda: true, name: d.Path}, func() { c.body(d.Body, s) })
	})
}

// body checks a function body (a block or an expression) and returns its type.
func (c *checker) body(b ast.Expr, s *scope) *Type {
	if blk, ok := b.(*ast.Block); ok {
		return c.block(blk, s.child())
	}
	return c.expr(b, s)
}

// returnsOrThrows reports whether a block contains «أرجع» or «ارفع» anywhere.
func returnsOrThrows(b ast.Expr) bool {
	found := false
	var walk func(n any)
	walk = func(n any) {
		if found || n == nil {
			return
		}
		switch x := n.(type) {
		case *ast.Return, *ast.Throw:
			found = true
		case *ast.Block:
			for _, s := range x.Stmts {
				walk(s)
			}
		case *ast.ExprStmt:
			walk(x.X)
		case *ast.If:
			walk(x.Then)
			walk(x.Else)
		case *ast.While:
			walk(x.Body)
		case *ast.For:
			walk(x.Body)
		case *ast.Match:
			for _, cs := range x.Cases {
				walk(cs.Body)
			}
		case *ast.Try:
			walk(x.Body)
			walk(x.Catch)
		case *ast.Let:
			walk(x.Value)
		}
	}
	walk(b)
	return found
}

// ---------------------------------------------------------------- statements

// block checks statements and returns the type of the last one when it is an
// expression (the block's value).
func (c *checker) block(b *ast.Block, s *scope) *Type {
	var last *Type
	for i, st := range b.Stmts {
		t := c.stmt(st, s)
		if _, isExpr := st.(*ast.ExprStmt); isExpr && i == len(b.Stmts)-1 {
			last = t
		}
	}
	return last
}

// atTopLevel reports whether code runs directly at the top of the file, not in
// a function, lambda or component.
func (c *checker) atTopLevel() bool {
	return c.fr.name == "" && !c.fr.lambda && !c.fr.component
}

func (c *checker) stmt(st ast.Stmt, s *scope) *Type {
	switch d := st.(type) {
	case *ast.ExprStmt:
		return c.expr(d.X, s)
	case *ast.Let:
		declared := c.checkRef(d.Type)
		vt := c.expr(d.Value, s)
		t := vt
		if d.Type != nil {
			if !assignable(vt, declared) {
				c.errorf(d.Value.P(), "«%s» من النوع «%s»، لكن قيمته من النوع «%s»", d.Name, declared, vt)
			}
			t = declared
		}
		kind, word := "constant", "ثابت "
		if d.Mutable {
			kind, word = "variable", "متغير "
		}
		ld := c.def(d.Name, c.nameAfter(d.Pos, d.Name), kind, word+d.Name+": "+t.String(), t, nil)
		c.define(s, d.Name, &symbol{kind: sVar, typ: t, mutable: d.Mutable, def: ld}, d.Pos)
	case *ast.StateDecl:
		if c.atTopLevel() {
			c.errorf(d.Pos, "«حالة» تُستخدم داخل مكون فقط؛ خارجه استخدم «متغير»")
		}
		declared := c.checkRef(d.Type)
		vt := c.expr(d.Value, s)
		t := vt
		if d.Type != nil {
			if !assignable(vt, declared) {
				c.errorf(d.Value.P(), "الحالة «%s» من النوع «%s»، لكن قيمتها الأولى من النوع «%s»", d.Name, declared, vt)
			}
			t = declared
		}
		sd := c.def(d.Name, c.nameAfter(d.Pos, d.Name), "state", "حالة "+d.Name+": "+t.String(), t, nil)
		c.define(s, d.Name, &symbol{kind: sVar, typ: t, mutable: true, def: sd}, d.Pos)
	case *ast.Assign:
		c.assign(d, s)
	case *ast.Return:
		if c.fr.name == "" && !c.fr.lambda && !c.fr.component {
			c.errorf(d.Pos, "«أرجع» مستخدمة خارج دالة")
		}
		var t *Type
		if d.X != nil {
			t = c.expr(d.X, s)
		}
		if !c.fr.lambda && c.fr.result != nil && d.X != nil && !assignable(t, c.fr.result) {
			c.errorf(d.X.P(), "الدالة «%s» ترجع «%s»، لكن هنا «%s»", c.fr.name, c.fr.result, t)
		}
		if !c.fr.lambda && c.fr.result != nil && !c.fr.result.Optional && d.X == nil {
			c.errorf(d.Pos, "الدالة «%s» يجب أن ترجع قيمة من النوع «%s»", c.fr.name, c.fr.result)
		}
	case *ast.Break:
		if c.fr.loops == 0 && !c.fr.lambda {
			c.errorf(d.Pos, "«توقف» مستخدمة خارج حلقة")
		}
	case *ast.Continue:
		if c.fr.loops == 0 && !c.fr.lambda {
			c.errorf(d.Pos, "«تابع» مستخدمة خارج حلقة")
		}
	case *ast.For:
		it := c.expr(d.Iter, s)
		var elem *Type
		if known(it) && !it.Optional {
			switch it.Kind {
			case List:
				elem = it.Elem
			case Range:
				elem = tInt
			case String:
				elem = tString
			case Map:
				elem = it.Key
			case Unknown:
			default:
				c.errorf(d.Iter.P(), "لا يمكن المرور على قيمة من النوع «%s» بـ«لكل»", it)
			}
		}
		body := s.child()
		vd := c.def(d.Var, c.nameAfter(d.Pos, d.Var), "variable", "لكل "+d.Var+": "+elem.String(), elem, nil)
		body.names[d.Var] = &symbol{kind: sVar, typ: elem, def: vd}
		c.fr.loops++
		c.block(d.Body, body.child())
		c.fr.loops--
	case *ast.While:
		c.condition(d.Cond, s, "شرط «طالما»")
		c.fr.loops++
		c.block(d.Body, s.child())
		c.fr.loops--
	case *ast.FuncDecl:
		// a local function
		fd := c.funcDef(d, "function", nil)
		c.define(s, d.Name, &symbol{kind: sFunc, decl: d, params: d.Params, typ: tFunc, what: "«" + d.Name + "»", def: fd}, d.Pos)
		c.funcDeclNow(d, s)
	case *ast.Import:
		c.errorf(d.Pos, "«استورد» تُكتب في أعلى الملف، لا داخل كتلة")
	case *ast.PageDecl:
		c.errorf(d.Pos, "«صفحة» تُعرَّف في أعلى الملف، لا داخل دالة أو مكون")
	default:
		// other declarations inside a block are handled like top-level ones
		c.collect(st)
		c.collectMembers(st)
		c.topLevel(st)
	}
	return nil
}

// funcDeclNow checks a local function's body with the current frame's context.
func (c *checker) funcDeclNow(d *ast.FuncDecl, outer *scope) {
	for _, p := range d.Params {
		c.checkRef(p.Type)
	}
	result := c.checkRef(d.Result)
	s := outer.child()
	c.params(d.Params, s, outer)
	fr := &frame{result: result, name: d.Name, component: c.fr.component, self: c.fr.self, hasSelf: c.fr.hasSelf}
	c.withFrame(fr, func() { c.body(d.Body, s) })
}

func (c *checker) assign(d *ast.Assign, s *scope) {
	vt := c.expr(d.Value, s)
	switch t := d.Target.(type) {
	case *ast.Ident:
		sym := s.lookup(t.Name)
		if sym == nil {
			if c.selfHas(t.Name) {
				return
			}
			c.errorf(t.Pos, "«%s» غير معرّف%s", t.Name, didYouMean(t.Name, s.visible()))
			return
		}
		c.ref(t.Pos, sym.def)
		if !sym.mutable {
			c.errorf(t.Pos, "لا يمكن تغيير «%s» لأنه ثابت — عرّفه بـ«متغير» إذا أردت تغييره", t.Name)
			return
		}
		nt := vt
		if d.Op != token.ASSIGN {
			nt = c.binaryType(compoundOps[d.Op], sym.typ, vt, d.Pos)
		}
		if known(sym.typ) && !assignable(nt, sym.typ) && !(sym.typ.Kind == Float && known(nt) && nt.Kind == Int) {
			c.errorf(d.Value.P(), "«%s» من النوع «%s»، ولا يمكن أن يأخذ قيمة من النوع «%s»", t.Name, sym.typ, nt)
		}
	case *ast.Index:
		ct := c.expr(t.X, s)
		c.expr(t.I, s)
		if known(ct) && !ct.Optional && ct.Kind != List && ct.Kind != Map {
			c.errorf(t.Pos, "لا يمكن الإسناد بالفهرس إلى قيمة من النوع «%s»", ct)
		}
	case *ast.Member:
		c.expr(t.X, s)
		c.errorf(t.Pos, "حقول الأنواع لا تتغير؛ استخدم «نسخة(%s: …)» لإنشاء نسخة معدّلة", t.Name)
	}
}

var compoundOps = map[token.Kind]token.Kind{
	token.PLUS_EQ: token.PLUS, token.MINUS_EQ: token.MINUS, token.STAR_EQ: token.STAR, token.SLASH_EQ: token.SLASH,
}

// condition checks that a condition is صح or خطأ, when its type is known.
func (c *checker) condition(e ast.Expr, s *scope, what string) {
	t := c.expr(e, s)
	if known(t) && !t.Optional && t.Kind != Bool {
		c.errorf(e.P(), "%s يجب أن يكون «صح» أو «خطأ» لكنه من النوع «%s»", what, t)
	}
}

// selfHas reports whether an unqualified name is a member of هذا (in methods
// and extensions). When هذا's type is unknown, any name may be.
func (c *checker) selfHas(name string) bool {
	if !c.fr.hasSelf {
		return false
	}
	if !known(c.fr.self) {
		return true
	}
	_, found := c.member(c.fr.self, name, true)
	return found
}

// Source parses and checks source code. The file is nil when it doesn't parse;
// diagnostics hold the parser's problems and the checker's, in source order.
func Source(src string) (*ast.File, []lexer.Diagnostic) {
	f, diags := parser.ParseFile(src)
	if f == nil {
		return nil, diags
	}
	all := append(diags, File(f)...)
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i].Pos, all[j].Pos
		return a.Line < b.Line || a.Line == b.Line && a.Col < b.Col
	})
	return f, all
}

// ---------------------------------------------------------------- programs of several files

// Program checks every file of a program, each after the files it imports,
// and adds the problems to each module's Diags.
func Program(p *module.Program) {
	c := newChecker()
	for _, m := range p.Modules {
		c.module(m, nil)
	}
}

// Load reads a program, checks it, and returns it with its problems.
func Load(path string, read module.Reader) (*module.Program, []module.Diag) {
	p := module.Load(path, read)
	Program(p)
	return p, p.Diags()
}

// module checks one file of a program; info records it for editors.
func (c *checker) module(m *module.Module, info *Info) {
	if m.File == nil {
		return
	}
	c.global = &scope{names: map[string]*symbol{}, parent: c.global.root()}
	c.mod, c.info, c.diags = m, info, nil
	if info != nil {
		info.c = c
	}
	c.file(m.File)
	c.scopes[m] = c.global
	m.Diags = append(m.Diags, c.diags...)
	c.diags = nil
}

// root is the scope of built-ins around every file.
func (s *scope) root() *scope {
	for s.parent != nil {
		s = s.parent
	}
	return s
}

// claim records the file that declares a name every file shares: types,
// tables, enums and their cases are known by name across the program, and
// server functions are called by name from the page.
func (c *checker) claim(name string, pos token.Pos, what string) {
	if c.mod == nil {
		return
	}
	if other := c.owners[name]; other != nil && other != c.mod {
		c.errorf(pos, "%s «%s» معرّف أيضاً في «%s»؛ أسماء الأنواع والجداول والتعدادات وحالاتها ودوال الخادم لا تتكرر بين ملفات البرنامج", what, name, other.Name())
		return
	}
	c.owners[name] = c.mod
}

// importDecl brings names from another file: «استورد نقطة، شكل من "./أشكال"».
// Importing a تعداد brings its cases too.
func (c *checker) importDecl(d *ast.Import) {
	g := c.global
	if c.imported[g] == nil {
		c.imported[g] = map[string]bool{}
	}
	var dep *module.Module
	if c.mod != nil {
		dep = c.mod.Deps[d]
	} else {
		c.errorf(d.Pos, "الاستيراد يحتاج ملفاً محفوظاً؛ هنا ضع الكود في ملف واحد")
	}
	from := c.scopes[dep]
	for i, n := range d.Names {
		pos := d.NamesPos[i]
		if from == nil {
			// the file is missing, makes a circle or doesn't parse: reported already
			g.names[n] = &symbol{kind: sVar}
			continue
		}
		sym := from.names[n]
		switch {
		case sym == nil || c.imported[from][n]:
			var exported []string
			for name := range from.names {
				if !c.imported[from][name] && !dep.File.Private[name] {
					exported = append(exported, name)
				}
			}
			sort.Strings(exported)
			c.errorf(pos, "«%s» غير موجود في «%s»%s", n, dep.Name(), didYouMean(n, exported))
			g.names[n] = &symbol{kind: sVar}
		case dep.File.Private[n] || dep.File.Private[c.variantOf[n]]:
			c.errorf(pos, "«%s» خاص بالملف «%s»، فلا يمكن استيراده", n, dep.Name())
			g.names[n] = &symbol{kind: sVar}
		default:
			c.importName(n, sym, pos, dep)
			if sym.kind == sEnum {
				for _, v := range c.enums[n].variants {
					if vs := from.names[v.Name]; vs != nil && g.names[v.Name] == nil {
						c.importName(v.Name, vs, pos, dep)
					}
				}
			}
		}
	}
}

func (c *checker) importName(name string, sym *symbol, pos token.Pos, dep *module.Module) {
	cp := *sym
	cp.mutable = false
	detail := "مستورد من «" + dep.Name() + "»"
	kind := "constant"
	switch sym.kind {
	case sFunc:
		detail, kind = funcSig(sym.decl)+" — "+detail, "function"
	case sType:
		detail, kind = "نوع "+name+paramsSig(sym.params)+" — "+detail, "type"
	case sEnum:
		detail, kind = "تعداد "+name+" — "+detail, "enum"
	case sVariant:
		detail, kind = name+paramsSig(sym.params)+" — "+detail, "case"
	case sComponent:
		detail, kind = "مكون "+name+paramsSig(sym.params)+" — "+detail, "component"
	default:
		if sym.typ != nil {
			detail = name + ": " + sym.typ.String() + " — " + detail
		}
	}
	cp.def = c.def(name, pos, kind, detail, sym.typ, nil)
	if cp.def != nil {
		cp.def.Origin = origin(dep, name)
	}
	c.define(c.global, name, &cp, pos)
	c.imported[c.global][name] = true
}

// visibleType reports whether a type declared in another file was imported.
func (c *checker) visibleType(name string) bool {
	owner := c.owners[name]
	if c.mod == nil || owner == nil || owner == c.mod {
		return true
	}
	return c.global.lookup(name) != nil
}

// origin finds where a file declares a top-level name, or a case of its enums.
func origin(m *module.Module, name string) *Origin {
	var at token.Pos
	for _, s := range m.File.Stmts {
		if ast.DeclName(s) == name {
			at = s.P()
		}
		if e, ok := s.(*ast.EnumDecl); ok {
			for _, v := range e.Variants {
				if v.Name == name {
					at = v.Pos
				}
			}
		}
	}
	toks, _ := lexer.Tokenize(m.Src)
	for _, t := range toks {
		if t.Kind == token.IDENT && t.Value == name && !before(t.Pos, at) {
			return &Origin{Path: m.Path, Src: m.Src, Pos: t.Pos, Len: len([]rune(t.Lit))}
		}
	}
	return nil
}
