package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/parser"
	"github.com/mtwalli/dad-lang/internal/token"
)

// What editors need, recorded while checking: where each name is defined,
// where it is used, and what type it has. The language server (dad lsp) uses it.

// Def is a name declared in the program.
type Def struct {
	Name     string
	Pos      token.Pos // the name itself
	Kind     string    // "function", "method", "type", "table", "enum", "case", "component", "field", "parameter", "constant", "variable", "state"
	Detail   string    // how it is declared: «دالة مربع(س: صحيح): صحيح»
	Type     *Type
	Parent   *Def    // the type a field, method or case belongs to
	Children []*Def  // fields, methods and cases of a type
	Global   bool    // declared at the top of the file
	Origin   *Origin // for an imported name: where the other file declares it
}

// Origin is a declaration in another file.
type Origin struct {
	Path string
	Src  string
	Pos  token.Pos // the name, in that file
	Len  int       // its length in runes
}

// Info is what the checker learned about a file.
type Info struct {
	Defs  []*Def
	Refs  map[token.Pos]*Def // name uses, by position
	Toks  []token.Token
	c     *checker
	byPos map[token.Pos]token.Token
}

// Analyze parses and checks source code and records definitions and uses.
// Info is nil when the source doesn't parse.
func Analyze(src string) (*Info, []lexer.Diagnostic) {
	f, diags := parser.ParseFile(src)
	if f == nil {
		return nil, diags
	}
	toks, _ := lexer.Tokenize(src)
	info := &Info{Refs: map[token.Pos]*Def{}, Toks: toks, byPos: map[token.Pos]token.Token{}}
	for _, t := range toks {
		info.byPos[t.Pos] = t
	}
	c := newChecker()
	c.info = info
	info.c = c
	c.file(f)
	all := append(diags, c.diags...)
	sort.SliceStable(all, func(i, j int) bool { return before(all[i].Pos, all[j].Pos) })
	return info, all
}

// AnalyzeProgram checks a program and records its main file for editors. The
// problems returned are the main file's; a problem in a file it imports is
// shown at the «استورد» that brings it in.
func AnalyzeProgram(p *module.Program) (*Info, []lexer.Diagnostic) {
	main := p.Main
	c := newChecker()
	for _, m := range p.Modules {
		if m != main {
			c.module(m, nil)
		}
	}
	if main.File == nil {
		return nil, main.Diags
	}
	toks, _ := lexer.Tokenize(main.Src)
	info := &Info{Refs: map[token.Pos]*Def{}, Toks: toks, byPos: map[token.Pos]token.Token{}}
	for _, t := range toks {
		info.byPos[t.Pos] = t
	}
	c.module(main, info)
	for imp, dep := range main.Deps {
		for _, d := range dep.Diags {
			if !d.Warning {
				main.Diags = append(main.Diags, lexer.Diagnostic{Pos: imp.Pos,
					Msg: fmt.Sprintf("في «%s» خطأ [السطر %d]: %s", dep.Name(), d.Pos.Line, d.Msg)})
				break
			}
		}
	}
	all := append([]lexer.Diagnostic(nil), main.Diags...)
	sort.SliceStable(all, func(i, j int) bool { return before(all[i].Pos, all[j].Pos) })
	return info, all
}

func before(a, b token.Pos) bool { return a.Line < b.Line || a.Line == b.Line && a.Col < b.Col }

// TokenAt returns the token that covers a position (line and column in runes, from 1).
func (in *Info) TokenAt(line, col int) (token.Token, bool) {
	for _, t := range in.Toks {
		if t.Pos.Line == line && col >= t.Pos.Col && col < t.Pos.Col+len([]rune(t.Lit)) {
			return t, true
		}
	}
	return token.Token{}, false
}

// DefAt returns the definition of the name at a position, whether the
// position is the declaration itself or a use.
func (in *Info) DefAt(line, col int) (*Def, bool) {
	t, ok := in.TokenAt(line, col)
	if !ok || t.Kind != token.IDENT {
		return nil, false
	}
	if d, ok := in.Refs[t.Pos]; ok {
		return d, true
	}
	for _, d := range in.Defs {
		if d.Pos == t.Pos {
			return d, true
		}
	}
	return nil, false
}

// Uses lists the positions where a definition's name appears: the declaration
// first, then every use, in order.
func (in *Info) Uses(d *Def) []token.Pos {
	out := []token.Pos{d.Pos}
	var refs []token.Pos
	for p, x := range in.Refs {
		if x == d && p != d.Pos {
			refs = append(refs, p)
		}
	}
	sort.Slice(refs, func(i, j int) bool { return before(refs[i], refs[j]) })
	return append(out, refs...)
}

// NameLen is the length in runes of the name token at a position, as written.
func (in *Info) NameLen(p token.Pos) int {
	if t, ok := in.byPos[p]; ok {
		return len([]rune(t.Lit))
	}
	return 0
}

// Visible lists the names usable at a position: top-level names and the
// local names declared before it in the same top-level declaration.
func (in *Info) Visible(line, col int) []*Def {
	at := token.Pos{Line: line, Col: col}
	// the top-level declaration that contains the position starts at the last
	// global definition before it
	start := token.Pos{}
	for _, d := range in.Defs {
		if d.Global && d.Parent == nil && !before(at, d.Pos) && before(start, d.Pos) {
			start = d.Pos
		}
	}
	var out []*Def
	seen := map[string]bool{}
	for i := len(in.Defs) - 1; i >= 0; i-- {
		d := in.Defs[i]
		if d.Parent != nil && !d.Global || seen[d.Name] || d.Kind == "page" {
			continue // fields and methods are reached with «.»; cases of a تعداد are global names; a page has no name
		}
		if d.Global || (!before(d.Pos, start) && before(d.Pos, at)) {
			seen[d.Name] = true
			out = append(out, d)
		}
	}
	return out
}

// Member is something that can follow «.» on a value.
type Member struct {
	Name   string
	Kind   string // "field", "method", "property"
	Detail string
}

// TypeOf finds the type of a chain of names written before «.», such as
// «م» or «م.عنوان», as seen from a position.
func (in *Info) TypeOf(chain []string, line, col int) *Type {
	if len(chain) == 0 {
		return nil
	}
	var t *Type
	for _, d := range in.Visible(line, col) {
		if d.Name == chain[0] {
			t = d.Type
			if d.Kind == "type" || d.Kind == "table" {
				t = &Type{Kind: TypeValue, Name: d.Name}
			}
			if d.Kind == "enum" {
				t = &Type{Kind: EnumType, Name: d.Name}
			}
		}
	}
	for _, name := range chain[1:] {
		if !known(t) {
			return nil
		}
		t, _ = in.c.member(t, name, false)
	}
	return t
}

// Members lists what can follow «.» on a value of type t.
func (in *Info) Members(t *Type) []Member {
	if !known(t) {
		return nil
	}
	c := in.c
	var out []Member
	add := func(name, kind, detail string) { out = append(out, Member{name, kind, detail}) }
	switch t.Kind {
	case Struct:
		if info := c.structs[t.Name]; info != nil {
			for _, f := range info.fields {
				add(f.Name, "field", f.Name+": "+info.ftypes[f.Name].String())
			}
			for name, m := range info.methods {
				add(name, "method", funcSig(m))
			}
			add("نسخة", "method", "نسخة(حقل: قيمة، …) — a copy with some fields changed")
			if info.table {
				add("احفظ", "method", "احفظ() — save this record")
				add("احذف", "method", "احذف() — delete this record")
			}
		}
	case TypeValue:
		if info := c.structs[t.Name]; info != nil && info.table {
			for _, m := range []struct{ n, d string }{
				{"أنشئ", "أنشئ(…) — create and save a record"}, {"الكل", "الكل() — every record"},
				{"حيث", "حيث { … } — the records that match"}, {"بالرقم", "بالرقم(رقم) — one record, or عدم"},
				{"عدد", "عدد() — how many records"},
			} {
				add(m.n, "method", m.d)
			}
		}
	case EnumType:
		for _, v := range c.enums[t.Name].variants {
			add(v.Name, "case", v.Name)
		}
	case Enum:
		for _, v := range c.enums[t.Name].variants {
			for _, f := range v.Fields {
				add(f.Name, "field", f.Name+" ("+v.Name+")")
			}
		}
	case String:
		for _, n := range []string{"طول", "عدد", "فارغ"} {
			add(n, "property", n)
		}
		for _, n := range stringMethods {
			if n != "طول" && n != "عدد" {
				add(n, "method", n+"()")
			}
		}
	case List, Range:
		for _, n := range []string{"طول", "عدد", "فارغ", "أول", "آخر"} {
			add(n, "property", n)
		}
		for _, n := range listMethods {
			if !has([]string{"طول", "عدد", "فارغ", "أول", "آخر"}, n) {
				add(n, "method", n+"()")
			}
		}
	case Map:
		for _, n := range []string{"طول", "عدد", "فارغ", "مفاتيح", "قيم"} {
			add(n, "property", n)
		}
		add("يحتوي", "method", "يحتوي(مفتاح)")
		add("احذف", "method", "احذف(مفتاح)")
	case Int, Float:
		for _, n := range numberMethods {
			add(n, "method", n+"()")
		}
	case Problem:
		add("رسالة", "property", "رسالة: نص")
	}
	for name, m := range c.ext[typeKey(t)] {
		add(name, "method", funcSig(m))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ---------------------------------------------------------------- recording (used by the checker)

func (c *checker) recording() bool { return c.info != nil && c.quiet == 0 }

// nameAfter finds the token of a name at or after a position (declarations
// record the position of their keyword).
func (c *checker) nameAfter(p token.Pos, name string) token.Pos {
	if c.info == nil {
		return p
	}
	for _, t := range c.info.Toks {
		if t.Kind == token.IDENT && t.Value == name && !before(t.Pos, p) {
			return t.Pos
		}
	}
	return p
}

// nameBefore finds the token of a name before a position, on the same line:
// the «عنوان» of a named argument «عنوان: …».
func (c *checker) nameBefore(p token.Pos, name string) (token.Pos, bool) {
	if c.info == nil {
		return p, false
	}
	found, ok := token.Pos{}, false
	for _, t := range c.info.Toks {
		if !before(t.Pos, p) {
			break
		}
		if t.Kind == token.IDENT && t.Value == name && t.Pos.Line == p.Line {
			found, ok = t.Pos, true
		}
	}
	return found, ok
}

// nameBeforeOr finds the «خ» of «التقط خ {» before the catch block.
func (c *checker) nameBeforeOr(p token.Pos, name string) token.Pos {
	if pos, ok := c.nameBefore(p, name); ok {
		return pos
	}
	return p
}

// tokenAfter is the first token after a position: the name after «.».
func (c *checker) tokenAfter(p token.Pos) token.Pos {
	if c.info == nil {
		return p
	}
	for _, t := range c.info.Toks {
		if before(p, t.Pos) {
			return t.Pos
		}
	}
	return p
}

func (c *checker) def(name string, pos token.Pos, kind, detail string, t *Type, parent *Def) *Def {
	if !c.recording() {
		return nil
	}
	d := &Def{Name: name, Pos: pos, Kind: kind, Detail: detail, Type: t, Parent: parent, Global: c.atTopLevel() && parent == nil}
	if parent != nil {
		parent.Children = append(parent.Children, d)
		d.Global = false
	}
	c.info.Defs = append(c.info.Defs, d)
	return d
}

func (c *checker) ref(pos token.Pos, d *Def) {
	if c.recording() && d != nil {
		c.info.Refs[pos] = d
	}
}

// memberDef finds the definition of a field or method of a user type.
func (c *checker) memberDef(t *Type, name string) *Def {
	if c.info == nil || !known(t) {
		return nil
	}
	if m := c.ext[typeKey(t)][name]; m != nil {
		return c.funcDefs[m]
	}
	switch t.Kind {
	case Struct, TypeValue:
		return c.memberDefs[t.Name][name]
	case EnumType:
		if d := c.global.lookup(name); d != nil {
			return d.def
		}
	}
	return nil
}

// ---------------------------------------------------------------- signatures

func refString(r *ast.TypeRef) string {
	if r == nil {
		return ""
	}
	var s string
	switch r.Kind {
	case ast.ListType:
		s = "[" + refString(r.Elem) + "]"
	case ast.MapType:
		s = "[" + refString(r.Key) + ": " + refString(r.Elem) + "]"
	case ast.FuncType:
		var parts []string
		for _, a := range r.Args {
			parts = append(parts, refString(a))
		}
		s = "(" + strings.Join(parts, "، ") + ") => " + refString(r.Elem)
	default:
		s = r.Name
	}
	if r.Optional {
		s += "؟"
	}
	return s
}

func paramsSig(params []*ast.Param) string {
	var parts []string
	for _, p := range params {
		s := p.Name
		if p.Type != nil {
			s += ": " + refString(p.Type)
		}
		if p.Default != nil {
			s += " = …"
		}
		parts = append(parts, s)
	}
	return "(" + strings.Join(parts, "، ") + ")"
}

func funcSig(d *ast.FuncDecl) string {
	s := "دالة " + d.Name + paramsSig(d.Params)
	if d.Result != nil {
		s += ": " + refString(d.Result)
	}
	if d.Server {
		s = "خادم " + s
	}
	return s
}
