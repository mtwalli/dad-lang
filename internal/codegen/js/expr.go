package js

import (
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

var builtins = map[string]bool{"اطبع": true, "مشكلة": true, "جذر": true, "مطلق": true, "أدنى": true, "أقصى": true}

var elements = map[string]bool{"عمود": true, "صف": true, "بطاقة": true, "عنوان": true, "فقرة": true, "زر": true,
	"حقل": true, "مربع_اختيار": true, "صورة": true, "رابط": true, "فاصل": true}

// bindFirst elements get a state variable itself, so they can change it.
var bindFirst = map[string]bool{"حقل": true, "مربع_اختيار": true}

var binOps = map[token.Kind]string{
	token.PLUS: "+", token.MINUS: "-", token.STAR: "*", token.SLASH: "/", token.PERCENT: "%",
	token.EQ: "==", token.NEQ: "!=", token.LT: "<", token.GT: ">", token.LE: "<=", token.GE: ">=",
}

func (c *compiler) cond(e ast.Expr, what string) string {
	return "$.truthy(" + c.expr(e) + ", " + pos(e.P()) + ", " + quote(what) + ")"
}

// known reports whether a name is defined where the interpreter would find it:
// in the program or among the built-ins.
func (c *compiler) known(name string) bool {
	return c.scope.lookup(name) != nil || builtins[name] || elements[name]
}

func (c *compiler) ident(name string, p token.Pos) string {
	if s := c.scope.lookup(name); s != nil {
		if s.kind == symState {
			return s.js + ".val"
		}
		return s.js
	}
	if builtins[name] {
		return "$.b[" + quote(name) + "]"
	}
	if elements[name] {
		return "$.elements[" + quote(name) + "]"
	}
	// Inside methods and extensions, members of هذا are in scope.
	if self := c.scope.lookup("هذا"); self != nil {
		return "$.selfGet(" + self.js + ", " + quote(name) + ", " + pos(p) + ")"
	}
	return "$.undef(" + pos(p) + ", " + quote(name) + ")"
}

func list(items []string) string {
	if len(items) == 0 {
		return "$.E"
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func (c *compiler) expr(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.IntLit:
		return strconv.FormatInt(x.Value, 10) + "n"
	case *ast.FloatLit:
		s := strconv.FormatFloat(x.Value, 'g', -1, 64)
		if s == "+Inf" {
			return "Infinity"
		}
		return s
	case *ast.BoolLit:
		return strconv.FormatBool(x.Value)
	case *ast.NullLit:
		return "null"
	case *ast.StrLit:
		var parts []string
		hasExpr := false
		for _, p := range x.Parts {
			if p.Expr != nil {
				hasExpr = true
				parts = append(parts, "$.format("+c.expr(p.Expr)+")")
			} else if p.Text != "" {
				parts = append(parts, quote(p.Text))
			}
		}
		if len(parts) == 0 {
			return `""`
		}
		if hasExpr && len(parts) == 1 {
			return parts[0]
		}
		return "(" + strings.Join(parts, " + ") + ")"
	case *ast.Ident:
		return c.ident(x.Name, x.Pos)
	case *ast.ThisExpr:
		if s := c.scope.lookup("هذا"); s != nil {
			return s.js
		}
		return failExpr(x.Pos, "«هذا» متاحة فقط داخل دوال الأنواع والامتدادات")
	case *ast.ItExpr:
		if s := c.scope.lookup("هو"); s != nil {
			return s.js
		}
		return failExpr(x.Pos, "«هو» متاحة فقط داخل دالة مجهولة تأخذ معاملاً واحداً")
	case *ast.ListLit:
		elems := make([]string, len(x.Elems))
		for i, el := range x.Elems {
			elems[i] = c.expr(el)
		}
		return "new $.List([" + strings.Join(elems, ", ") + "])"
	case *ast.MapLit:
		pairs := make([]string, len(x.Keys))
		for i := range x.Keys {
			pairs[i] = "[" + c.expr(x.Keys[i]) + ", " + c.expr(x.Values[i]) + "]"
		}
		return "$.map([" + strings.Join(pairs, ", ") + "])"
	case *ast.Unary:
		if x.Op == token.MINUS {
			if lit, ok := x.X.(*ast.IntLit); ok {
				return "-" + c.expr(lit)
			}
			return "$.neg(" + c.expr(x.X) + ", " + pos(x.Pos) + ")"
		}
		return "!$.truthy(" + c.expr(x.X) + ", " + pos(x.Pos) + ", " + quote("ما بعد «ليس»") + ")"
	case *ast.Binary:
		switch x.Op {
		case token.AND, token.AND_SYM:
			return "(" + c.cond(x.L, "طرف «و»") + " && " + c.cond(x.R, "طرف «و»") + ")"
		case token.OR, token.OR_SYM:
			return "(" + c.cond(x.L, "طرف «أو»") + " || " + c.cond(x.R, "طرف «أو»") + ")"
		case token.COALESCE:
			return "(" + c.expr(x.L) + " ?? " + c.expr(x.R) + ")"
		}
		op, ok := binOps[x.Op]
		if !ok {
			return failExpr(x.Pos, "عامل غير مدعوم")
		}
		return "$.bin(" + quote(op) + ", " + c.expr(x.L) + ", " + c.expr(x.R) + ", " + pos(x.Pos) + ")"
	case *ast.Call:
		return c.call(x)
	case *ast.Member:
		safe := ""
		if x.Safe {
			safe = ", true"
		}
		return "$.member(" + c.expr(x.X) + ", " + quote(x.Name) + ", " + pos(x.Pos) + safe + ")"
	case *ast.Index:
		return "$.index(" + c.expr(x.X) + ", " + c.expr(x.I) + ", " + pos(x.Pos) + ")"
	case *ast.Lambda:
		return c.lambda(x)
	case *ast.If:
		then := c.blockValue(x.Then)
		els := "null"
		switch e := x.Else.(type) {
		case *ast.Block:
			els = c.blockValue(e)
		case *ast.If:
			els = c.expr(e)
		}
		return "(" + c.cond(x.Cond, "شرط «إذا»") + " ? " + then + " : " + els + ")"
	case *ast.Match:
		return c.iife(func() { c.matchStmt(x, tailReturn) })
	case *ast.Try:
		return c.iife(func() { c.tryStmt(x, tailReturn) })
	case *ast.Block:
		return c.blockValue(x)
	case *ast.RangeExpr:
		step := "null"
		if x.Step != nil {
			step = "[" + c.expr(x.Step) + ", " + pos(x.Step.P()) + "]"
		}
		return "$.range([" + c.expr(x.From) + ", " + pos(x.From.P()) + "], [" + c.expr(x.To) + ", " + pos(x.To.P()) + "], " +
			strconv.FormatBool(x.Inclusive) + ", " + step + ")"
	case *ast.TypeTest:
		return "$.isType(" + c.expr(x.X) + ", " + typeJS(x.Type) + ")"
	case *ast.Cast:
		return "$.cast(" + c.expr(x.X) + ", " + typeJS(x.Type) + ", " + pos(x.Pos) + ")"
	case *ast.Throw:
		return "$.doThrow(" + c.expr(x.X) + ", " + pos(x.Pos) + ")"
	}
	return failExpr(e.P(), "تعبير غير مدعوم")
}

// blockValue is the value of a block: its last expression statement.
func (c *compiler) blockValue(b *ast.Block) string {
	switch {
	case len(b.Stmts) == 0:
		return "null"
	case len(b.Stmts) == 1:
		if es, ok := b.Stmts[0].(*ast.ExprStmt); ok {
			return c.expr(es.X)
		}
	}
	return c.iife(func() { c.block(b, tailReturn) })
}

// iife compiles statements as a function called at once, for a block used as a value.
func (c *compiler) iife(body func()) string {
	c.frames = append(c.frames, &frame{kind: frameIIFE})
	text := c.sub(c.indent+1, body)
	c.frames = c.frames[:len(c.frames)-1]
	return "(() => {\n" + text + strings.Repeat("  ", c.indent) + "})()"
}

func (c *compiler) lambda(l *ast.Lambda) string {
	c.push()
	defer c.pop()
	var names []string
	n := len(l.Params)
	if n == 0 {
		names = []string{c.declare("هو", symConst)}
	}
	for _, p := range l.Params {
		names = append(names, c.declare(p, symConst))
	}
	head := "$.lam(" + strconv.Itoa(n) + ", (" + strings.Join(names, ", ") + ") => "
	c.frames = append(c.frames, &frame{kind: frameLambda})
	defer func() { c.frames = c.frames[:len(c.frames)-1] }()
	if len(l.Body.Stmts) == 1 {
		if es, ok := l.Body.Stmts[0].(*ast.ExprStmt); ok && !statementLike(es.X) {
			return head + "(" + c.expr(es.X) + "))"
		}
	}
	if len(l.Body.Stmts) == 0 {
		return head + "null)"
	}
	text := c.sub(c.indent+1, func() { c.stmts(l.Body.Stmts, tailReturn) })
	return head + "{\n" + text + strings.Repeat("  ", c.indent) + "})"
}

// statementLike expressions read better as statements than as nested values.
func statementLike(e ast.Expr) bool {
	switch e.(type) {
	case *ast.If, *ast.Match, *ast.Try, *ast.Block, *ast.Throw:
		return true
	}
	return false
}

func (c *compiler) call(x *ast.Call) string {
	callee, _ := x.Fn.(*ast.Ident)
	bind := callee != nil && c.scope.lookup(callee.Name) == nil && bindFirst[callee.Name]
	var args, named []string
	for i, a := range x.Args {
		if i == 0 && bind && a.Name == "" {
			// «حقل(جديدة)»: the element gets the state variable itself
			if id, ok := a.Value.(*ast.Ident); ok {
				if s := c.scope.lookup(id.Name); s != nil && s.kind == symState {
					args = append(args, "new $.StateRef("+quote(id.Name)+", "+s.js+")")
					continue
				}
			}
		}
		v := c.expr(a.Value)
		if a.Name != "" {
			named = append(named, "["+quote(a.Name)+", "+v+"]")
			continue
		}
		if len(named) > 0 {
			return failExpr(a.Value.P(), "المعاملات المسماة يجب أن تأتي بعد المعاملات العادية")
		}
		args = append(args, v)
	}
	if x.Lambda != nil {
		args = append(args, c.lambda(x.Lambda))
	}
	rest := list(args) + ", " + list(named) + ", " + pos(x.Pos)
	if m, ok := x.Fn.(*ast.Member); ok {
		safe := ""
		if m.Safe {
			safe = ", true"
		}
		return "$.method(" + c.expr(m.X) + ", " + quote(m.Name) + ", " + rest + safe + ")"
	}
	name := ""
	if callee != nil {
		name = callee.Name
		// Inside methods and extensions, «نسخة(…)» or «تحية()» call methods of هذا.
		if !c.known(name) {
			if self := c.scope.lookup("هذا"); self != nil {
				return "$.method(" + self.js + ", " + quote(name) + ", " + rest + ")"
			}
		}
	}
	return "$.call(" + c.expr(x.Fn) + ", " + rest + ", " + quote(name) + ")"
}
