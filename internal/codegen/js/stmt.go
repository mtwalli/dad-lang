package js

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

func (c *compiler) stmt(s ast.Stmt, t tail) {
	switch s := s.(type) {
	case *ast.ExprStmt:
		c.exprStmt(s.X, t)
	case *ast.Let:
		v := c.expr(s.Value)
		if s.Type != nil {
			v = "$.convert(" + v + ", " + typeJS(s.Type) + ", " + pos(s.Pos) + ", " + quote("«"+s.Name+"»") + ")"
		}
		kw, kind := "const ", symConst
		if s.Mutable {
			kw, kind = "let ", symVar
		}
		// the value is compiled before the name exists: «ثابت س = س + 1» reads an outer س
		c.line(kw + c.declare(s.Name, kind) + " = " + v + ";")
	case *ast.StateDecl:
		v := "() => " + c.expr(s.Value)
		js := c.declare(s.Name, symState)
		c.line("const " + js + " = $.state(" + v + ", " + typeJS(s.Type) + ", " + pos(s.Pos) + ", " + quote(s.Name) + ");")
	case *ast.Assign:
		c.assign(s)
	case *ast.Return:
		v := "null"
		if s.X != nil {
			v = c.expr(s.X)
		}
		c.ret(v)
	case *ast.Break:
		c.jump("break", "$.BREAK")
	case *ast.Continue:
		c.jump("continue", "$.CONTINUE")
	case *ast.For:
		c.forStmt(s)
	case *ast.While:
		c.loop("while ("+c.cond(s.Cond, "شرط «طالما»")+")", func() { c.block(s.Body, tailNone) })
	case *ast.Import:
		c.line(failExpr(s.Pos, "الاستيراد من ملفات أخرى غير مدعوم بعد؛ ضع الكود في ملف واحد حالياً") + ";")
	case *ast.FuncDecl:
		c.line("const " + c.declare(s.Name, symConst) + " = " + c.function(s, nil, false) + ";")
	case *ast.TypeDecl:
		c.typeDecl(s)
	case *ast.TableDecl:
		c.tableDecl(s)
	case *ast.EnumDecl:
		c.enumDecl(s)
	case *ast.ExtensionDecl:
		c.extensionDecl(s)
	case *ast.ComponentDecl:
		c.componentDecl(s)
	case *ast.PageDecl:
		c.pageDecl(s)
	case *ast.RouteDecl, *ast.ContractDecl, *ast.Unsupported:
		// routes need a server; contracts are checked by the checker
	default:
		c.line(failExpr(s.P(), "عبارة غير مدعومة") + ";")
	}
}

// ret compiles «أرجع». Directly in a function it is a JavaScript return; from
// inside a block used as a value it is thrown to the function.
func (c *compiler) ret(v string) {
	crossed := false
	for i := len(c.frames) - 1; i >= 0; i-- {
		f := c.frames[i]
		switch f.kind {
		case frameLoop:
			continue
		case frameIIFE:
			crossed = true
			continue
		case frameFunc, frameLambda:
			if !crossed {
				if f.ret != nil {
					v = f.ret(v)
				}
				c.line("return " + v + ";")
				return
			}
			f.catch = true
		}
		break
	}
	c.line("throw new $.ReturnSig(" + v + ");")
}

// jump compiles «توقف» and «تابع». They cross functions to the loop around
// them, as in the interpreter.
func (c *compiler) jump(kw, signal string) {
	crossed := false
	for i := len(c.frames) - 1; i >= 0; i-- {
		f := c.frames[i]
		if f.kind == frameLoop {
			if !crossed {
				c.line(kw + ";")
				return
			}
			f.catch = true
			break
		}
		crossed = true
	}
	c.line("throw " + signal + ";")
}

// loop writes a loop; when its body throws «توقف» or «تابع» (from a block used
// as a value), the body is wrapped to catch them.
func (c *compiler) loop(head string, body func()) {
	fr := &frame{kind: frameLoop}
	c.frames = append(c.frames, fr)
	text := c.sub(c.indent+1, body)
	c.frames = c.frames[:len(c.frames)-1]
	c.line(head + " {")
	if !fr.catch {
		c.out.WriteString(text)
	} else {
		c.indent++
		c.line("try {")
		c.out.WriteString(reindent(text, 1))
		c.line("} catch ($$e) {")
		c.line("  if ($$e === $.BREAK) break;")
		c.line("  if ($$e === $.CONTINUE) continue;")
		c.line("  throw $$e;")
		c.line("}")
		c.indent--
	}
	c.line("}")
}

func reindent(s string, levels int) string {
	pad := strings.Repeat("  ", levels)
	lines := strings.SplitAfter(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "")
}

func (c *compiler) forStmt(s *ast.For) {
	iter := "$.iter(" + c.expr(s.Iter) + ", " + pos(s.Iter.P()) + ", " + pos(s.Pos) + ")"
	c.push()
	v := c.declare(s.Var, symConst)
	c.loop("for (const "+v+" of "+iter+")", func() { c.block(s.Body, tailNone) })
	c.pop()
}

var assignOps = map[token.Kind]string{token.PLUS_EQ: "+", token.MINUS_EQ: "-", token.STAR_EQ: "*", token.SLASH_EQ: "/"}

func (c *compiler) assign(s *ast.Assign) {
	rhs := func(old string) string {
		v := c.expr(s.Value)
		if op, ok := assignOps[s.Op]; ok {
			return "$.bin(" + quote(op) + ", " + old + ", " + v + ", " + pos(s.Pos) + ")"
		}
		return v
	}
	switch t := s.Target.(type) {
	case *ast.Ident:
		sym := c.scope.lookup(t.Name)
		switch {
		case sym == nil:
			c.line(failExpr(t.Pos, "«"+t.Name+"» غير معرّف") + ";")
		case sym.kind == symConst:
			c.line(failExpr(t.Pos, "لا يمكن تغيير «"+t.Name+"» لأنه ثابت — عرّفه بـ«متغير» إذا أردت تغييره") + ";")
		default:
			ref := sym.js
			if sym.kind == symState {
				ref += ".val"
			}
			c.line(ref + " = $.keepFloat(" + ref + ", " + rhs(ref) + ");")
		}
	case *ast.Index:
		x, i := c.expr(t.X), c.expr(t.I)
		o := c.temp()
		c.line("$.setIndex(" + x + ", " + i + ", (" + o + ") => " + rhs(o) + ", " + pos(t.Pos) + ");")
	case *ast.Member:
		c.line(failExpr(t.Pos, "حقول الأنواع لا تتغير؛ استخدم «نسخة("+t.Name+": …)» لإنشاء نسخة معدّلة") + ";")
	default:
		c.line(failExpr(s.Pos, "لا يمكن الإسناد إلى هذا") + ";")
	}
}

// ---------------------------------------------------------------- «طابق» and «حاول»

func (c *compiler) matchStmt(m *ast.Match, t tail) {
	subject := c.temp()
	c.line("{")
	c.indent++
	c.line("const " + subject + " = " + c.expr(m.Subject) + ";")
	kw := "if"
	for _, cs := range m.Cases {
		c.push()
		test, binds := c.pattern(cs.Pattern, subject)
		c.line(kw + " (" + test + ") {")
		c.indent++
		for _, b := range binds {
			c.line(b)
		}
		if b, ok := cs.Body.(*ast.Block); ok {
			c.stmts(b.Stmts, t)
		} else {
			c.exprStmt(cs.Body, t)
		}
		c.indent--
		c.pop()
		kw = "} else if"
	}
	if len(m.Cases) > 0 {
		c.line("} else {")
		c.line("  $.noMatch(" + subject + ", " + pos(m.Pos) + ");")
		c.line("}")
	} else {
		c.line("$.noMatch(" + subject + ", " + pos(m.Pos) + ");")
	}
	c.indent--
	c.line("}")
}

// pattern returns the test for a case and the statements that bind its names.
func (c *compiler) pattern(p *ast.Pattern, subject string) (string, []string) {
	switch p.Kind {
	case ast.WildcardPattern:
		return "true", nil
	case ast.LiteralPattern:
		return "$.equal(" + c.expr(p.Lit) + ", " + subject + ")", nil
	}
	if c.variants[p.Name] {
		v := "$.variants[" + quote(p.Name) + "]"
		test := subject + " instanceof $.EnumVal && " + subject + ".variant === " + v
		var binds []string
		if len(p.Binds) > 0 {
			binds = append(binds, "$.patternFields("+subject+", "+v+", "+strconv.Itoa(len(p.Binds))+", "+pos(p.Pos)+");")
		}
		for i, name := range p.Binds {
			if name != "_" {
				binds = append(binds, "const "+c.declare(name, symConst)+" = "+subject+".vals["+strconv.Itoa(i)+"];")
			}
		}
		return test, binds
	}
	if p.HasArgs {
		return failExpr(p.Pos, "«"+p.Name+"» ليست حالة معروفة في أي تعداد"), nil
	}
	return "true", []string{"const " + c.declare(p.Name, symConst) + " = " + subject + ";"}
}

func (c *compiler) tryStmt(x *ast.Try, t tail) {
	c.line("try {")
	c.indent++
	c.block(x.Body, t)
	c.indent--
	e := c.temp()
	c.line("} catch (" + e + ") {")
	c.indent++
	c.push()
	c.line("const " + c.declare(x.Var, symConst) + " = $.caught(" + e + ");")
	c.stmts(x.Catch.Stmts, t)
	c.pop()
	c.indent--
	c.line("}")
}

// ---------------------------------------------------------------- functions

// function compiles a function or method to a $.fn value. A method takes the
// receiver first; fields are the receiver's fields, in scope by name.
func (c *compiler) function(d *ast.FuncDecl, fields []*ast.Param, method bool) string {
	c.push()
	defer c.pop()
	var pre []string
	if method {
		pre = append(pre, "const "+c.declare("هذا", symConst)+" = self;")
		for i, f := range fields {
			pre = append(pre, "const "+c.declare(f.Name, symConst)+" = self.vals["+strconv.Itoa(i)+"];")
		}
	}
	if len(d.Params) > 0 {
		ps, names := c.params(d.Params)
		pre = append(pre, "const ["+strings.Join(names, ", ")+"] = $.bind("+ps+", a, n, p, "+quote("«"+d.Name+"»")+");")
	} else {
		pre = append(pre, "$.bind($.E, a, n, p, "+quote("«"+d.Name+"»")+");")
	}
	ret := func(v string) string { return v }
	if d.Result != nil {
		what := quote("قيمة الإرجاع من «" + d.Name + "»")
		ret = func(v string) string { return "$.convert(" + v + ", " + typeJS(d.Result) + ", p, " + what + ")" }
	}
	fr := &frame{kind: frameFunc, ret: ret}
	c.frames = append(c.frames, fr)
	body := c.sub(c.indent+1, func() {
		if b, ok := d.Body.(*ast.Block); ok {
			c.stmts(b.Stmts, tailNone)
			if d.Result != nil && !d.Result.Optional {
				c.line("$.fail(p, " + quote("الدالة «"+d.Name+"» يجب أن ترجع قيمة من النوع «"+typeString(d.Result)+"» — هل نسيت «أرجع»؟") + ");")
			} else {
				c.line("return null;")
			}
		} else if d.Result == nil {
			c.exprStmt(d.Body, tailReturn)
		} else {
			c.line("return " + ret(c.expr(d.Body)) + ";")
		}
	})
	c.frames = c.frames[:len(c.frames)-1]

	pad := strings.Repeat("  ", c.indent)
	var b strings.Builder
	args := "(a, n, p)"
	if method {
		args = "(self, a, n, p)"
	}
	b.WriteString("$.fn(" + quote(d.Name) + ", " + args + " => {\n")
	for _, l := range pre {
		b.WriteString(pad + "  " + l + "\n")
	}
	if fr.catch {
		b.WriteString(pad + "  try {\n" + reindent(body, 1))
		b.WriteString(pad + "  } catch ($$e) {\n")
		b.WriteString(pad + "    if ($$e instanceof $.ReturnSig) return " + ret("$$e.val") + ";\n")
		b.WriteString(pad + "    throw $$e;\n" + pad + "  }\n")
	} else {
		b.WriteString(body)
	}
	b.WriteString(pad + "})")
	return b.String()
}

// fieldList writes the fields of a type for the runtime; defaults are
// functions of the fields before them.
func (c *compiler) fieldList(fields []*ast.Param) string {
	c.push()
	defer c.pop()
	s, _ := c.params(fields)
	return s
}

func (c *compiler) typeDecl(d *ast.TypeDecl) {
	js := c.declare(d.Name, symConst)
	c.line("const " + js + " = new $.StructType(" + quote(d.Name) + ", " + c.fieldList(d.Fields) + ", false);")
	c.line("$.types[" + quote(d.Name) + "] = " + js + ";")
	for _, m := range d.Methods {
		c.line(js + ".methods[" + quote(m.Name) + "] = " + c.function(m, d.Fields, true) + ";")
	}
}

// tableFields are a table's columns and its hidden id «رقم».
func tableFields(d *ast.TableDecl) []*ast.Param {
	var out []*ast.Param
	for _, f := range d.Fields {
		if f.Type == nil {
			cp := *f
			cp.Type = literalType(f.Default)
			f = &cp
		}
		out = append(out, f)
	}
	return append(out, &ast.Param{Name: "رقم", Type: &ast.TypeRef{Kind: ast.NamedType, Name: "صحيح"}, Default: &ast.IntLit{Value: 0}})
}

func (c *compiler) tableDecl(d *ast.TableDecl) {
	js := c.declare(d.Name, symConst)
	c.line("const " + js + " = new $.StructType(" + quote(d.Name) + ", " + c.fieldList(tableFields(d)) + ", true);")
	c.line("$.types[" + quote(d.Name) + "] = " + js + ";")
}

func (c *compiler) enumDecl(d *ast.EnumDecl) {
	js := c.declare(d.Name, symConst)
	c.line("const " + js + " = new $.EnumType(" + quote(d.Name) + ");")
	c.line("$.types[" + quote(d.Name) + "] = " + js + ";")
	for _, v := range d.Variants {
		c.line("const " + c.declare(v.Name, symConst) + " = $.variant(" + js + ", " + quote(v.Name) + ", " + c.fieldList(v.Fields) + ");")
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

func (c *compiler) extensionDecl(d *ast.ExtensionDecl) {
	key := typeKeyOfRef(d.Type)
	var fields []*ast.Param
	if td := c.types[key]; td != nil {
		fields = td.Fields
	} else if tb := c.tables[key]; tb != nil {
		fields = tableFields(tb)
	}
	for _, m := range d.Methods {
		c.line("$.extend(" + quote(key) + ", " + quote(m.Name) + ", " + c.function(m, fields, true) + ");")
	}
}

// pageDecl compiles «صفحة "/مهمة/{رقم}" = …» to a page the runtime shows at that
// address: a function of the address's values that lays the page out.
func (c *compiler) pageDecl(d *ast.PageDecl) {
	c.push()
	type value struct{ name, js string }
	var values []value
	for _, m := range pathName.FindAllStringSubmatch(d.Path, -1) {
		n := strings.TrimSpace(m[1])
		values = append(values, value{n, c.declare(n, symConst)})
	}
	fr := &frame{kind: frameLambda}
	c.frames = append(c.frames, fr)
	body := c.sub(c.indent+1, func() {
		for _, v := range values {
			c.line("const " + v.js + " = $.pathValue($$v[" + quote(v.name) + "]);")
		}
		if blk, ok := d.Body.(*ast.Block); ok {
			c.stmts(blk.Stmts, tailNone)
		} else {
			c.stmt(&ast.ExprStmt{At: ast.At{Pos: d.Body.P()}, X: d.Body}, tailNone)
		}
	})
	c.frames = c.frames[:len(c.frames)-1]
	c.pop()
	c.line("$.page(" + quote(d.Path) + ", ($$v) => {")
	c.out.WriteString(body)
	c.line("});")
}

var pathName = regexp.MustCompile(`\{([^{}/]+)\}`)

func (c *compiler) componentDecl(d *ast.ComponentDecl) {
	js := c.declare(d.Name, symConst)
	c.push()
	ps, names := c.params(d.Params)
	fr := &frame{kind: frameLambda}
	c.frames = append(c.frames, fr)
	body := c.sub(c.indent+1, func() { c.stmts(d.Body.Stmts, tailNone) })
	c.frames = c.frames[:len(c.frames)-1]
	c.pop()
	c.line("const " + js + " = new $.Component(" + quote(d.Name) + ", " + ps + ", (" + strings.Join(names, ", ") + ") => {")
	c.out.WriteString(body)
	c.line("});")
}
