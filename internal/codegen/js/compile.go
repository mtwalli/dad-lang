// Package js compiles Ḍād (ضاد) programs to JavaScript, for «dad build».
//
// The output runs on the runtime in runtime.js, which follows the interpreter
// rule for rule: the same values, the same methods and the same Arabic error
// messages. A program prints the same thing whether the interpreter runs it or
// Node and the browser run the compiled code; the tests check that on every
// example.
package js

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

// Runtime is the JavaScript runtime the compiled code needs, as dad-runtime.js.
//
//go:embed runtime.js
var Runtime string

// Program is a compiled program.
type Program struct {
	// Body is the program as the body of a JavaScript function that takes the
	// runtime as «$». It returns the main component, or null.
	Body string
	// Root is the name of the component a page shows, or "" without an interface.
	Root string
	// Routes and ServerFuncs count what a static build cannot serve:
	// routes (مسار) are left out; server functions (خادم) run in the page.
	Routes, ServerFuncs int
}

// Node is the program as a script for Node: it loads ./dad-runtime.js and runs.
// file names the program in error messages.
func (p *Program) Node(file string) string {
	return "\"use strict\";\n// Compiled from " + file + " by «dad build».\nconst $ = require(\"./dad-runtime.js\");\n" +
		"$.main(() => {\n" + p.Body + "}, " + quote(file) + ");\n"
}

// Browser is the program as a script for a page that loaded dad-runtime.js
// first: it runs the program and shows its main component in #dad-root.
func (p *Program) Browser(file string) string {
	return "\"use strict\";\n// Compiled from " + file + " by «dad build».\n(function ($) {\n" +
		"  const root = $.main(() => {\n" + p.Body + "}, " + quote(file) + ");\n" +
		"  if (root) $.mount(root);\n})($dad);\n"
}

// Compile compiles a parsed, checked file.
func Compile(f *ast.File) *Program {
	c := &compiler{variants: map[string]bool{}, types: map[string]*ast.TypeDecl{}, tables: map[string]*ast.TableDecl{}, used: map[string]int{}}
	c.collect(f.Stmts)
	c.scope = &scope{names: map[string]*sym{}}
	c.frames = []*frame{{kind: frameTop}}
	p := &Program{}

	// Every top-level name is visible everywhere in the file: functions run
	// later than they are declared and may use constants declared after them.
	for _, s := range f.Stmts {
		c.predeclare(s, true)
	}
	var decls, stmts []ast.Stmt
	for _, s := range f.Stmts {
		if isDecl(s) {
			decls = append(decls, s)
		} else {
			stmts = append(stmts, s)
		}
	}
	c.indent = 1
	for _, s := range decls {
		switch d := s.(type) {
		case *ast.RouteDecl:
			p.Routes++
		case *ast.FuncDecl:
			if d.Server {
				p.ServerFuncs++
			}
		}
		c.stmt(s, tailNone)
	}
	c.stmts(stmts, tailNone)
	root := rootComponent(f)
	if root != "" {
		p.Root = root
		c.line("return " + c.scope.lookup(root).js + ";")
	} else {
		c.line("return null;")
	}
	p.Body = c.out.String()
	return p
}

// rootComponent is the component a page shows: one named الرئيسية, التطبيق or
// تطبيق, otherwise the last one that needs no arguments (as in the interpreter).
func rootComponent(f *ast.File) string {
	last := ""
	for _, s := range f.Stmts {
		d, ok := s.(*ast.ComponentDecl)
		if !ok {
			continue
		}
		if d.Name == "الرئيسية" || d.Name == "التطبيق" || d.Name == "تطبيق" {
			return d.Name
		}
		needs := false
		for _, p := range d.Params {
			if p.Default == nil && (p.Type == nil || !p.Type.Optional) {
				needs = true
			}
		}
		if !needs {
			last = d.Name
		}
	}
	return last
}

func isDecl(s ast.Stmt) bool {
	switch s.(type) {
	case *ast.FuncDecl, *ast.TableDecl, *ast.ComponentDecl, *ast.RouteDecl, *ast.TypeDecl,
		*ast.EnumDecl, *ast.ExtensionDecl, *ast.ContractDecl, *ast.Unsupported:
		return true
	}
	return false
}

// ---------------------------------------------------------------- names and scopes

type symKind int

const (
	symConst symKind = iota
	symVar
	symState // a component's «حالة»: a cell, read as .val
)

type sym struct {
	js   string
	kind symKind
}

type scope struct {
	names  map[string]*sym
	parent *scope
}

func (s *scope) lookup(name string) *sym {
	for x := s; x != nil; x = x.parent {
		if v, ok := x.names[name]; ok {
			return v
		}
	}
	return nil
}

type frameKind int

const (
	frameTop frameKind = iota
	frameFunc
	frameLambda // a lambda or component: the runtime catches «أرجع» thrown inside it
	frameIIFE   // a block used as a value, compiled as a function call
	frameLoop
)

type frame struct {
	kind frameKind
	// what a «أرجع» directly in a function becomes
	ret func(v string) string
	// set when «أرجع» (function) or «توقف»/«تابع» (loop) had to be thrown
	catch bool
}

type compiler struct {
	out      strings.Builder
	indent   int
	scope    *scope
	frames   []*frame
	tmp      int
	used     map[string]int // JS names given out, to keep shadowing names apart
	variants map[string]bool
	types    map[string]*ast.TypeDecl
	tables   map[string]*ast.TableDecl
}

// collect finds every enum case, type and table, wherever declared: the
// interpreter keeps them in tables of their own.
func (c *compiler) collect(stmts []ast.Stmt) {
	for _, s := range stmts {
		switch d := s.(type) {
		case *ast.EnumDecl:
			for _, v := range d.Variants {
				c.variants[v.Name] = true
			}
		case *ast.TypeDecl:
			c.types[d.Name] = d
		case *ast.TableDecl:
			c.tables[d.Name] = d
		}
	}
}

func (c *compiler) push() { c.scope = &scope{names: map[string]*sym{}, parent: c.scope} }
func (c *compiler) pop()  { c.scope = c.scope.parent }
func (c *compiler) temp() string {
	c.tmp++
	return "$$" + strconv.Itoa(c.tmp)
}

// declare adds a name to the current scope and returns its JavaScript name.
// A name that shadows another visible one gets a name of its own, so code
// before the declaration still reaches the outer one.
func (c *compiler) declare(name string, kind symKind) string {
	if s, ok := c.scope.names[name]; ok {
		s.kind = kind
		return s.js
	}
	js := "$" + name
	if c.used[name] > 0 {
		js += "$" + strconv.Itoa(c.used[name]+1)
	}
	c.used[name]++
	c.scope.names[name] = &sym{js: js, kind: kind}
	return js
}

// predeclare makes a block's declared names visible from its start.
func (c *compiler) predeclare(s ast.Stmt, top bool) {
	switch d := s.(type) {
	case *ast.FuncDecl:
		c.declare(d.Name, symConst)
	case *ast.TableDecl:
		c.declare(d.Name, symConst)
	case *ast.ComponentDecl:
		c.declare(d.Name, symConst)
	case *ast.TypeDecl:
		c.declare(d.Name, symConst)
	case *ast.EnumDecl:
		c.declare(d.Name, symConst)
		for _, v := range d.Variants {
			c.declare(v.Name, symConst)
		}
	case *ast.Let:
		if top {
			k := symConst
			if d.Mutable {
				k = symVar
			}
			c.declare(d.Name, k)
		}
	}
}

// ---------------------------------------------------------------- output

func (c *compiler) line(s string) {
	c.out.WriteString(strings.Repeat("  ", c.indent))
	c.out.WriteString(s)
	c.out.WriteByte('\n')
}

// sub compiles into a separate buffer and returns what was written.
func (c *compiler) sub(indent int, f func()) string {
	saved, savedIndent := c.out, c.indent
	c.out = strings.Builder{}
	c.indent = indent
	f()
	s := c.out.String()
	c.out, c.indent = saved, savedIndent
	return s
}

func quote(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}

func pos(p token.Pos) string { return fmt.Sprintf("[%d,%d]", p.Line, p.Col) }

func failExpr(p token.Pos, msg string) string { return "$.fail(" + pos(p) + ", " + quote(msg) + ")" }

// ---------------------------------------------------------------- types

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

// typeJS writes a type as the descriptor the runtime checks values against.
func typeJS(t *ast.TypeRef) string {
	if t == nil {
		return "null"
	}
	var s string
	switch t.Kind {
	case ast.ListType:
		s = `{k:"l",elem:` + typeJS(t.Elem)
	case ast.MapType:
		s = `{k:"m",key:` + typeJS(t.Key) + `,elem:` + typeJS(t.Elem)
	case ast.FuncType:
		s = `{k:"f",str:` + quote(typeString(&ast.TypeRef{Kind: ast.FuncType, Args: t.Args, Elem: t.Elem}))
	default:
		s = `{k:"n",name:` + quote(t.Name)
	}
	if t.Optional {
		s += ",opt:true"
	}
	return s + "}"
}

// literalType is the type of a table column given only a default.
func literalType(e ast.Expr) *ast.TypeRef {
	name := ""
	switch x := e.(type) {
	case *ast.BoolLit:
		name = "منطقي"
	case *ast.IntLit:
		name = "صحيح"
	case *ast.FloatLit:
		name = "عشري"
	case *ast.StrLit:
		name = "نص"
	case *ast.Unary:
		if x.Op == token.MINUS {
			return literalType(x.X)
		}
	}
	if name == "" {
		return nil
	}
	return &ast.TypeRef{Kind: ast.NamedType, Name: name}
}

// params writes parameters for $.bind: each default is a function of the
// parameters before it. It declares the parameters in the current scope and
// returns their JavaScript names.
func (c *compiler) params(ps []*ast.Param) (string, []string) {
	var parts, names []string
	for _, p := range ps {
		def := "null"
		if p.Default != nil {
			def = "(" + strings.Join(names, ", ") + ") => " + c.expr(p.Default)
		}
		parts = append(parts, "{name:"+quote(p.Name)+",type:"+typeJS(p.Type)+",def:"+def+"}")
		names = append(names, c.declare(p.Name, symConst))
	}
	return "[" + strings.Join(parts, ", ") + "]", names
}

// ---------------------------------------------------------------- statements

type tail int

const (
	tailNone   tail = iota
	tailReturn      // the block's value is returned (lambdas, blocks used as values)
)

// stmts compiles statements; with tailReturn the value of the last
// expression statement is returned, as a Ḍād block gives it.
func (c *compiler) stmts(list []ast.Stmt, t tail) {
	for _, s := range list {
		c.predeclare(s, false)
	}
	for i, s := range list {
		last := i == len(list)-1
		if last && t == tailReturn {
			if es, ok := s.(*ast.ExprStmt); ok {
				c.exprStmt(es.X, tailReturn)
				return
			}
		}
		c.stmt(s, tailNone)
	}
	if t == tailReturn {
		c.line("return null;")
	}
}

// block compiles a block in a scope of its own.
func (c *compiler) block(b *ast.Block, t tail) {
	c.push()
	c.stmts(b.Stmts, t)
	c.pop()
}

// exprStmt compiles an expression used as a statement. Blocks, «إذا», «طابق»
// and «حاول» become JavaScript statements rather than values.
func (c *compiler) exprStmt(e ast.Expr, t tail) {
	emit := func(v string) {
		if t == tailReturn {
			c.line("return " + v + ";")
		} else {
			c.line(v + ";")
		}
	}
	switch x := e.(type) {
	case *ast.Block:
		c.line("{")
		c.indent++
		c.block(x, t)
		c.indent--
		c.line("}")
	case *ast.If:
		c.ifStmt(x, t, "if")
		if t == tailReturn {
			c.line("return null;")
		}
	case *ast.Match:
		c.matchStmt(x, t)
	case *ast.Try:
		c.tryStmt(x, t)
	case *ast.Throw:
		c.line("throw $.throwValue(" + c.expr(x.X) + ", " + pos(x.Pos) + ");")
	default:
		emit(c.expr(e))
	}
}

func (c *compiler) ifStmt(x *ast.If, t tail, kw string) {
	c.line(kw + " (" + c.cond(x.Cond, "شرط «إذا»") + ") {")
	c.indent++
	c.block(x.Then, t)
	c.indent--
	switch els := x.Else.(type) {
	case *ast.If:
		c.ifStmt(els, t, "} else if")
		return
	case *ast.Block:
		c.line("} else {")
		c.indent++
		c.block(els, t)
		c.indent--
	}
	c.line("}")
}
