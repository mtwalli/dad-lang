// Package parser builds a syntax tree from Ḍād (ضاد) source.
//
// It is a hand-written recursive-descent parser with precedence climbing for
// expressions, so that every error can be reported precisely and in Arabic.
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/token"
)

type parseError struct {
	pos token.Pos
	msg string
}

type parser struct {
	toks     []token.Token
	i        int
	noLambda bool // true while parsing a condition: "إذا x {" must not read "x {…}" as a call
}

// ParseFile parses a whole source file. Lexer warnings are returned as
// diagnostics; on any error the file is nil and the diagnostics explain why.
func ParseFile(src string) (file *ast.File, diags []lexer.Diagnostic) {
	toks, lexDiags := lexer.Tokenize(src)
	diags = lexDiags
	if lexer.HasErrors(lexDiags) {
		return nil, diags
	}
	p := &parser{toks: toks}
	defer func() {
		if r := recover(); r != nil {
			pe, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			file = nil
			diags = append(diags, lexer.Diagnostic{Pos: pe.pos, Msg: pe.msg})
		}
	}()
	file = &ast.File{Private: map[string]bool{}}
	p.skipNL()
	for !p.at(token.EOF) {
		private := p.at(token.PRIVATE)
		s := p.parseStmt()
		if private {
			if name := ast.DeclName(s); name != "" {
				file.Private[name] = true
			} else {
				p.failf(s.P(), "«خاص» تسبق تعريفاً: دالة أو نوع أو ثابت…")
			}
		}
		file.Stmts = append(file.Stmts, s)
		p.endStmt()
		p.skipNL()
	}
	return file, diags
}

// ---------------------------------------------------------------- helpers

func (p *parser) cur() token.Token { return p.toks[p.i] }

func (p *parser) peek(n int) token.Token {
	if p.i+n < len(p.toks) {
		return p.toks[p.i+n]
	}
	return p.toks[len(p.toks)-1]
}

func (p *parser) at(k token.Kind) bool { return p.cur().Kind == k }

func (p *parser) next() token.Token {
	t := p.toks[p.i]
	if p.i < len(p.toks)-1 {
		p.i++
	}
	return t
}

func (p *parser) accept(k token.Kind) bool {
	if p.at(k) {
		p.next()
		return true
	}
	return false
}

func (p *parser) failf(pos token.Pos, format string, args ...any) {
	panic(parseError{pos: pos, msg: fmt.Sprintf(format, args...)})
}

func describe(t token.Token) string {
	switch t.Kind {
	case token.NEWLINE:
		return "نهاية السطر"
	case token.EOF:
		return "نهاية الملف"
	case token.STRING:
		return "نص"
	}
	if t.Lit != "" {
		return t.Lit
	}
	return t.Kind.String()
}

// attachedHint explains a name like «وب» or «كعشري»: in Arabic writing و and ك join the
// next word, but in Ḍād they are separate keywords and need a space.
func attachedHint(t token.Token) string {
	if t.Kind != token.IDENT {
		return ""
	}
	r := []rune(t.Value)
	if len(r) < 2 || (r[0] != 'و' && r[0] != 'ك') {
		return ""
	}
	return fmt.Sprintf(" — هل تقصد «%c %s»؟", r[0], string(r[1:]))
}

func (p *parser) expect(k token.Kind, what string) token.Token {
	if !p.at(k) {
		p.failf(p.cur().Pos, "توقعت «%s» لكن وجدت «%s»%s", what, describe(p.cur()), attachedHint(p.cur()))
	}
	return p.next()
}

func (p *parser) expectIdent() string {
	if !p.at(token.IDENT) {
		p.failf(p.cur().Pos, "توقعت اسماً لكن وجدت «%s»", describe(p.cur()))
	}
	return p.next().Value
}

func (p *parser) skipNL() {
	for p.at(token.NEWLINE) || p.at(token.SEMI) {
		p.next()
	}
}

func (p *parser) endStmt() {
	switch p.cur().Kind {
	case token.NEWLINE, token.SEMI:
		p.next()
	case token.RBRACE, token.EOF:
	default:
		p.failf(p.cur().Pos, "توقعت نهاية السطر لكن وجدت «%s»%s", describe(p.cur()), attachedHint(p.cur()))
	}
}

func (p *parser) stmtEnds() bool {
	switch p.cur().Kind {
	case token.NEWLINE, token.SEMI, token.RBRACE, token.EOF:
		return true
	}
	return false
}

// ---------------------------------------------------------------- statements

func (p *parser) parseStmt() ast.Stmt {
	t := p.cur()
	switch t.Kind {
	case token.CONST, token.VAR:
		return p.parseLet()
	case token.SERVER:
		p.next()
		if !p.at(token.FUNC) {
			p.failf(p.cur().Pos, "توقعت «دالة» بعد «خادم»")
		}
		f := p.parseFunc()
		f.Server = true
		return f
	case token.FUNC:
		return p.parseFunc()
	case token.PRIVATE:
		p.next()
		return p.parseStmt()
	case token.TYPE:
		return p.parseTypeDecl()
	case token.ENUM:
		return p.parseEnum()
	case token.CONTRACT:
		p.next()
		name := p.expectIdent()
		p.skipBraces()
		return &ast.ContractDecl{At: ast.At{Pos: t.Pos}, Name: name}
	case token.EXTENSION:
		p.next()
		typ := p.parseType()
		return &ast.ExtensionDecl{At: ast.At{Pos: t.Pos}, Type: typ, Methods: p.parseMembers()}
	case token.IMPORT:
		return p.parseImport()
	case token.TABLE:
		return p.parseTable()
	case token.ROUTE:
		return p.parseRoute()
	case token.COMPONENT:
		return p.parseComponent()
	case token.STATE:
		return p.parseState()
	case token.FOR:
		return p.parseFor()
	case token.WHILE:
		p.next()
		cond := p.parseExprNoLambda()
		return &ast.While{At: ast.At{Pos: t.Pos}, Cond: cond, Body: p.parseBlock()}
	case token.RETURN:
		p.next()
		r := &ast.Return{At: ast.At{Pos: t.Pos}}
		if !p.stmtEnds() {
			r.X = p.parseExpr()
		}
		return r
	case token.BREAK:
		p.next()
		return &ast.Break{At: ast.At{Pos: t.Pos}}
	case token.CONTINUE:
		p.next()
		return &ast.Continue{At: ast.At{Pos: t.Pos}}
	}

	// «صفحة» is a word with a meaning only here, before an address: a program
	// may still use it as a name.
	if t.Kind == token.IDENT && t.Value == "صفحة" && p.peek(1).Kind == token.STRING {
		return p.parsePage()
	}

	x := p.parseExpr()
	switch p.cur().Kind {
	case token.ASSIGN, token.PLUS_EQ, token.MINUS_EQ, token.STAR_EQ, token.SLASH_EQ:
		op := p.next()
		switch x.(type) {
		case *ast.Ident, *ast.Index, *ast.Member:
		default:
			p.failf(op.Pos, "لا يمكن الإسناد إلى هذا التعبير")
		}
		return &ast.Assign{At: ast.At{Pos: op.Pos}, Target: x, Op: op.Kind, Value: p.parseExpr()}
	}
	return &ast.ExprStmt{At: ast.At{Pos: t.Pos}, X: x}
}

func (p *parser) parseLet() ast.Stmt {
	t := p.next()
	l := &ast.Let{At: ast.At{Pos: t.Pos}, Mutable: t.Kind == token.VAR}
	l.Name = p.expectIdent()
	if p.accept(token.COLON) {
		l.Type = p.parseType()
	}
	p.expect(token.ASSIGN, "=")
	l.Value = p.parseExpr()
	return l
}

// parseComponent reads «مكون اسم(معاملات) { … }»; the parentheses are optional.
func (p *parser) parseComponent() ast.Stmt {
	t := p.next()
	c := &ast.ComponentDecl{At: ast.At{Pos: t.Pos}, Name: p.expectIdent()}
	if p.at(token.LPAREN) {
		c.Params = p.parseParams()
	}
	if !p.at(token.LBRACE) {
		p.failf(p.cur().Pos, "توقعت «{» لجسم المكون «%s»", c.Name)
	}
	c.Body = p.parseBlock()
	return c
}

// parseState reads «حالة اسم = قيمة» (or «حالة اسم: نوع = قيمة»).
func (p *parser) parseState() ast.Stmt {
	t := p.next()
	s := &ast.StateDecl{At: ast.At{Pos: t.Pos}, Name: p.expectIdent()}
	if p.accept(token.COLON) {
		s.Type = p.parseType()
	}
	if !p.at(token.ASSIGN) {
		p.failf(p.cur().Pos, "الحالة «%s» تحتاج قيمة أولى، مثل: حالة %s = 0", s.Name, s.Name)
	}
	p.next()
	s.Value = p.parseExpr()
	return s
}

func (p *parser) parseParams() []*ast.Param {
	p.expect(token.LPAREN, "(")
	var params []*ast.Param
	for !p.at(token.RPAREN) {
		pos := p.cur().Pos
		prm := &ast.Param{At: ast.At{Pos: pos}, Name: p.expectIdent()}
		p.expect(token.COLON, ":")
		prm.Type = p.parseType()
		if p.accept(token.ASSIGN) {
			prm.Default = p.withLambdas(p.parseExpr)
		}
		params = append(params, prm)
		if !p.accept(token.COMMA) {
			break
		}
	}
	p.expect(token.RPAREN, ")")
	return params
}

func (p *parser) parseFunc() *ast.FuncDecl {
	t := p.expect(token.FUNC, "دالة")
	f := &ast.FuncDecl{At: ast.At{Pos: t.Pos}}
	f.Name = p.expectIdent()
	f.Params = p.parseParams()
	if p.accept(token.COLON) {
		f.Result = p.parseType()
	}
	switch {
	case p.accept(token.ASSIGN):
		f.Body = p.parseExpr()
	case p.at(token.LBRACE):
		f.Body = p.parseBlock()
	default:
		p.failf(p.cur().Pos, "توقعت «{» أو «=» لجسم الدالة «%s»", f.Name)
	}
	return f
}

func (p *parser) parseMembers() []*ast.FuncDecl {
	p.expect(token.LBRACE, "{")
	p.skipNL()
	var methods []*ast.FuncDecl
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		p.accept(token.PRIVATE)
		if !p.at(token.FUNC) {
			p.failf(p.cur().Pos, "يُسمح هنا بالدوال فقط، لكن وجدت «%s»", describe(p.cur()))
		}
		methods = append(methods, p.parseFunc())
		p.endStmt()
		p.skipNL()
	}
	p.expect(token.RBRACE, "}")
	return methods
}

func (p *parser) parseTypeDecl() ast.Stmt {
	t := p.next()
	d := &ast.TypeDecl{At: ast.At{Pos: t.Pos}, Name: p.expectIdent()}
	if p.at(token.LPAREN) {
		d.Fields = p.parseParams()
	}
	if p.accept(token.COLON) {
		d.Contracts = append(d.Contracts, p.parseType())
		for p.accept(token.COMMA) {
			d.Contracts = append(d.Contracts, p.parseType())
		}
	}
	if p.at(token.LBRACE) {
		d.Methods = p.parseMembers()
	}
	return d
}

func (p *parser) parseEnum() ast.Stmt {
	t := p.next()
	d := &ast.EnumDecl{At: ast.At{Pos: t.Pos}, Name: p.expectIdent()}
	p.expect(token.LBRACE, "{")
	p.skipNL()
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		v := &ast.Variant{At: ast.At{Pos: p.cur().Pos}, Name: p.expectIdent()}
		if p.at(token.LPAREN) {
			v.Fields = p.parseParams()
		}
		d.Variants = append(d.Variants, v)
		if !p.accept(token.COMMA) {
			p.endStmt()
		}
		p.skipNL()
	}
	p.expect(token.RBRACE, "}")
	return d
}

func (p *parser) parseImport() ast.Stmt {
	t := p.next()
	im := &ast.Import{At: ast.At{Pos: t.Pos}}
	for {
		im.NamesPos = append(im.NamesPos, p.cur().Pos)
		im.Names = append(im.Names, p.expectIdent())
		if !p.accept(token.COMMA) {
			break
		}
	}
	p.expect(token.FROM, "من")
	im.From = p.expect(token.STRING, "نص").Value
	return im
}

// parseTable reads «جدول اسم { حقل: نوع  حقل = قيمة  حقل: نوع = قيمة }».
func (p *parser) parseTable() ast.Stmt {
	t := p.next()
	d := &ast.TableDecl{At: ast.At{Pos: t.Pos}, Name: p.expectIdent()}
	p.expect(token.LBRACE, "{")
	p.skipNL()
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		f := &ast.Param{At: ast.At{Pos: p.cur().Pos}, Name: p.expectIdent()}
		if f.Name == "رقم" {
			p.failf(f.Pos, "«رقم» يُضاف تلقائياً لكل جدول؛ اختر اسماً آخر للحقل")
		}
		if p.accept(token.COLON) {
			f.Type = p.parseType()
		}
		if p.accept(token.ASSIGN) {
			f.Default = p.withLambdas(p.parseExpr)
		}
		if f.Type == nil && f.Default == nil {
			p.failf(f.Pos, "الحقل «%s» يحتاج نوعاً أو قيمة افتراضية، مثل: «%s: نص» أو «%s = 0»", f.Name, f.Name, f.Name)
		}
		d.Fields = append(d.Fields, f)
		if !p.accept(token.COMMA) {
			p.endStmt()
		}
		p.skipNL()
	}
	p.expect(token.RBRACE, "}")
	return d
}

// routeMethods maps the Arabic method words to HTTP methods.
var routeMethods = map[string]string{
	"جلب": "GET", "أرسل": "POST", "عدل": "PUT", "احذف": "DELETE",
	"GET": "GET", "POST": "POST", "PUT": "PUT", "DELETE": "DELETE",
}

// parseRoute reads «مسار جلب "/api/مهام" = تعبير» or «مسار أرسل "/api/مهام" { … }».
func (p *parser) parseRoute() ast.Stmt {
	t := p.next()
	d := &ast.RouteDecl{At: ast.At{Pos: t.Pos}}
	mt := p.cur()
	method, ok := routeMethods[mt.Value]
	if mt.Kind != token.IDENT || !ok {
		p.failf(mt.Pos, "توقعت طريقة المسار: «جلب» أو «أرسل» أو «عدّل» أو «احذف»، لكن وجدت «%s»", describe(mt))
	}
	p.next()
	d.Method = method
	pt := p.expect(token.STRING, "مسار مثل \"/api/مهام\"")
	d.Path = pt.Value
	if !strings.HasPrefix(d.Path, "/") {
		p.failf(pt.Pos, "المسار يبدأ بـ «/»، مثل: \"/api/مهام\"")
	}
	switch {
	case p.accept(token.ASSIGN):
		d.Body = p.parseExpr()
	case p.at(token.LBRACE):
		d.Body = p.parseBlock()
	default:
		p.failf(p.cur().Pos, "توقعت «=» أو «{» بعد المسار")
	}
	return d
}

// parsePage reads «صفحة "/المهام" = قائمة_المهام()» or «صفحة "/" { … }».
func (p *parser) parsePage() ast.Stmt {
	t := p.next()
	d := &ast.PageDecl{At: ast.At{Pos: t.Pos}}
	pt := p.next()
	d.Path, d.PathPos = pt.Value, pt.Pos
	if !strings.HasPrefix(d.Path, "/") {
		p.failf(pt.Pos, "عنوان الصفحة يبدأ بـ «/»، مثل: \"/المهام\"")
	}
	switch {
	case p.accept(token.ASSIGN):
		d.Body = p.parseExpr()
	case p.at(token.LBRACE):
		d.Body = p.parseBlock()
	default:
		p.failf(p.cur().Pos, "توقعت «=» أو «{» بعد عنوان الصفحة")
	}
	return d
}

// parseUnsupported reads a full-stack declaration that cannot run yet and
// skips its body.
func (p *parser) parseUnsupported() ast.Stmt {
	t := p.next()
	u := &ast.Unsupported{At: ast.At{Pos: t.Pos}, Keyword: t.Value}
	if p.at(token.IDENT) {
		u.Name = p.cur().Value
	}
	depth := 0
	for !p.at(token.EOF) {
		switch p.cur().Kind {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			if depth == 0 {
				return u
			}
			depth--
		case token.NEWLINE, token.SEMI:
			if depth == 0 {
				return u
			}
		}
		p.next()
	}
	return u
}

func (p *parser) skipBraces() {
	p.expect(token.LBRACE, "{")
	depth := 1
	for depth > 0 && !p.at(token.EOF) {
		switch p.next().Kind {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
		}
	}
}

func (p *parser) parseFor() ast.Stmt {
	t := p.next()
	f := &ast.For{At: ast.At{Pos: t.Pos}, Var: p.expectIdent()}
	switch {
	case p.accept(token.IN):
		f.Iter = p.parseExprNoLambda()
	case p.at(token.FROM):
		f.Iter = p.parseRangeTail(p.next().Pos)
	default:
		p.failf(p.cur().Pos, "توقعت «في» أو «من» بعد «لكل %s»", f.Var)
	}
	f.Body = p.parseBlock()
	return f
}

// parseRangeTail parses "a إلى b [بخطوة s]" or "a حتى b [بخطوة s]" after من.
func (p *parser) parseRangeTail(pos token.Pos) *ast.RangeExpr {
	r := &ast.RangeExpr{At: ast.At{Pos: pos}}
	r.From = p.parseExprNoLambda()
	switch {
	case p.accept(token.TO):
		r.Inclusive = true
	case p.accept(token.UNTIL):
	default:
		p.failf(p.cur().Pos, "توقعت «إلى» أو «حتى» لكن وجدت «%s»", describe(p.cur()))
	}
	r.To = p.parseExprNoLambda()
	if p.accept(token.STEP) {
		r.Step = p.parseExprNoLambda()
	}
	return r
}

func (p *parser) parseBlock() *ast.Block {
	t := p.expect(token.LBRACE, "{")
	saved := p.noLambda
	p.noLambda = false
	b := &ast.Block{At: ast.At{Pos: t.Pos}}
	p.skipNL()
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		b.Stmts = append(b.Stmts, p.parseStmt())
		p.endStmt()
		p.skipNL()
	}
	p.expect(token.RBRACE, "}")
	p.noLambda = saved
	return b
}

// ---------------------------------------------------------------- types

func (p *parser) parseType() *ast.TypeRef {
	pos := p.cur().Pos
	tr := &ast.TypeRef{At: ast.At{Pos: pos}}
	switch {
	case p.accept(token.LBRACK):
		elem := p.parseType()
		if p.accept(token.COLON) {
			tr.Kind, tr.Key, tr.Elem = ast.MapType, elem, p.parseType()
		} else {
			tr.Kind, tr.Elem = ast.ListType, elem
		}
		p.expect(token.RBRACK, "]")
	case p.accept(token.LPAREN):
		tr.Kind = ast.FuncType
		for !p.at(token.RPAREN) {
			tr.Args = append(tr.Args, p.parseType())
			if !p.accept(token.COMMA) {
				break
			}
		}
		p.expect(token.RPAREN, ")")
		p.expect(token.ARROW, "=>")
		tr.Elem = p.parseType()
	default:
		tr.Kind = ast.NamedType
		tr.Name = p.expectIdent()
		if p.accept(token.LT) {
			for !p.at(token.GT) {
				tr.Args = append(tr.Args, p.parseType())
				if !p.accept(token.COMMA) {
					break
				}
			}
			p.expect(token.GT, ">")
		}
	}
	if p.accept(token.QUESTION) {
		tr.Optional = true
	}
	return tr
}

// ---------------------------------------------------------------- expressions

func (p *parser) parseExpr() ast.Expr { return p.parseBinary(1) }

func (p *parser) parseExprNoLambda() ast.Expr {
	saved := p.noLambda
	p.noLambda = true
	e := p.parseExpr()
	p.noLambda = saved
	return e
}

func (p *parser) withLambdas(f func() ast.Expr) ast.Expr {
	saved := p.noLambda
	p.noLambda = false
	e := f()
	p.noLambda = saved
	return e
}

func precedence(k token.Kind) int {
	switch k {
	case token.COALESCE:
		return 1
	case token.OR, token.OR_SYM:
		return 2
	case token.AND, token.AND_SYM:
		return 3
	case token.EQ, token.NEQ:
		return 4
	case token.LT, token.GT, token.LE, token.GE:
		return 5
	case token.RANGE:
		return 6
	case token.PLUS, token.MINUS:
		return 7
	case token.STAR, token.SLASH, token.PERCENT:
		return 8
	}
	return 0
}

func (p *parser) parseBinary(minPrec int) ast.Expr {
	left := p.parseCast()
	for {
		t := p.cur()
		if t.Kind == token.IT {
			// «هو» means "it" (in lambdas); the type test is «من نوع».
			p.failf(t.Pos, "لفحص النوع اكتب «من نوع»، مثل: «ش من نوع دائرة»")
		}
		if t.Kind == token.FROM && p.peek(1).Kind == token.TYPE { // "x من نوع T": type test
			if 5 < minPrec {
				return left
			}
			p.next()
			p.next()
			left = &ast.TypeTest{At: ast.At{Pos: t.Pos}, X: left, Type: p.parseType()}
			continue
		}
		pr := precedence(t.Kind)
		if pr == 0 || pr < minPrec {
			return left
		}
		p.next()
		right := p.parseBinary(pr + 1)
		if t.Kind == token.RANGE {
			left = &ast.RangeExpr{At: ast.At{Pos: t.Pos}, From: left, To: right, Inclusive: true}
			continue
		}
		left = &ast.Binary{At: ast.At{Pos: t.Pos}, Op: t.Kind, L: left, R: right}
	}
}

func (p *parser) parseCast() ast.Expr {
	x := p.parseUnary()
	for p.at(token.AS) {
		t := p.next()
		x = &ast.Cast{At: ast.At{Pos: t.Pos}, X: x, Type: p.parseType()}
	}
	return x
}

func (p *parser) parseUnary() ast.Expr {
	switch p.cur().Kind {
	case token.MINUS, token.NOT, token.BANG:
		t := p.next()
		return &ast.Unary{At: ast.At{Pos: t.Pos}, Op: t.Kind, X: p.parseUnary()}
	}
	return p.parsePostfix()
}

func memberName(t token.Token) (string, bool) {
	if t.Kind == token.IDENT || t.Kind.IsKeyword() {
		return t.Value, true
	}
	return "", false
}

func (p *parser) parsePostfix() ast.Expr {
	x := p.parsePrimary()
	for {
		t := p.cur()
		switch t.Kind {
		case token.LPAREN:
			call := &ast.Call{At: ast.At{Pos: t.Pos}, Fn: x, Args: p.parseArgs()}
			if !p.noLambda && p.at(token.LBRACE) {
				call.Lambda = p.parseLambda()
			}
			x = call
		case token.DOT, token.SAFE_DOT:
			p.next()
			name, ok := memberName(p.cur())
			if !ok {
				p.failf(p.cur().Pos, "توقعت اسماً بعد «.» لكن وجدت «%s»", describe(p.cur()))
			}
			p.next()
			x = &ast.Member{At: ast.At{Pos: t.Pos}, X: x, Name: name, Safe: t.Kind == token.SAFE_DOT}
		case token.LBRACK:
			p.next()
			i := p.withLambdas(p.parseExpr)
			p.expect(token.RBRACK, "]")
			x = &ast.Index{At: ast.At{Pos: t.Pos}, X: x, I: i}
		case token.LBRACE:
			switch x.(type) {
			case *ast.Ident, *ast.Member:
				if !p.noLambda {
					x = &ast.Call{At: ast.At{Pos: t.Pos}, Fn: x, Lambda: p.parseLambda()}
					continue
				}
			}
			return x
		default:
			return x
		}
	}
}

func (p *parser) parseArgs() []ast.Arg {
	p.expect(token.LPAREN, "(")
	saved := p.noLambda
	p.noLambda = false
	var args []ast.Arg
	for !p.at(token.RPAREN) && !p.at(token.EOF) {
		var a ast.Arg
		if p.at(token.IDENT) && p.peek(1).Kind == token.COLON {
			a.Name = p.next().Value
			p.next()
		}
		a.Value = p.parseExpr()
		args = append(args, a)
		if !p.accept(token.COMMA) {
			break
		}
	}
	p.expect(token.RPAREN, ")")
	p.noLambda = saved
	return args
}

func (p *parser) parseLambda() *ast.Lambda {
	t := p.cur()
	lam := &ast.Lambda{At: ast.At{Pos: t.Pos}}
	// Explicit parameters: "{ أ، ب => …"
	j := p.i + 1
	var params []string
	for p.toks[j].Kind == token.IDENT {
		params = append(params, p.toks[j].Value)
		j++
		if p.toks[j].Kind == token.COMMA {
			j++
			continue
		}
		break
	}
	if len(params) > 0 && p.toks[j].Kind == token.ARROW {
		lam.Params = params
		p.next() // {
		for p.i < j {
			p.next()
		}
		p.next() // =>
		// Re-enter block parsing with the brace already consumed.
		b := &ast.Block{At: ast.At{Pos: t.Pos}}
		saved := p.noLambda
		p.noLambda = false
		p.skipNL()
		for !p.at(token.RBRACE) && !p.at(token.EOF) {
			b.Stmts = append(b.Stmts, p.parseStmt())
			p.endStmt()
			p.skipNL()
		}
		p.expect(token.RBRACE, "}")
		p.noLambda = saved
		lam.Body = b
		return lam
	}
	lam.Body = p.parseBlock()
	return lam
}

func (p *parser) parsePrimary() ast.Expr {
	t := p.cur()
	at := ast.At{Pos: t.Pos}
	switch t.Kind {
	case token.INT:
		p.next()
		v, err := strconv.ParseInt(t.Value, 10, 64)
		if err != nil {
			p.failf(t.Pos, "العدد «%s» كبير جداً", t.Lit)
		}
		return &ast.IntLit{At: at, Value: v}
	case token.FLOAT:
		p.next()
		v, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			p.failf(t.Pos, "عدد عشري غير صالح «%s»", t.Lit)
		}
		return &ast.FloatLit{At: at, Value: v}
	case token.STRING:
		p.next()
		return p.parseString(t)
	case token.TRUE, token.FALSE:
		p.next()
		return &ast.BoolLit{At: at, Value: t.Kind == token.TRUE}
	case token.NULL:
		p.next()
		return &ast.NullLit{At: at}
	case token.THIS:
		p.next()
		return &ast.ThisExpr{At: at}
	case token.IT:
		p.next()
		return &ast.ItExpr{At: at}
	case token.IDENT:
		p.next()
		return &ast.Ident{At: at, Name: t.Value}
	case token.LPAREN:
		p.next()
		e := p.withLambdas(p.parseExpr)
		p.expect(token.RPAREN, ")")
		return e
	case token.LBRACK:
		return p.parseListOrMap()
	case token.LBRACE:
		return p.parseLambda()
	case token.IF:
		return p.parseIf()
	case token.MATCH:
		return p.parseMatch()
	case token.TRY:
		return p.parseTry()
	case token.THROW:
		p.next()
		return &ast.Throw{At: at, X: p.parseExpr()}
	case token.FROM:
		p.next()
		return p.parseRangeTail(t.Pos)
	}
	p.failf(t.Pos, "تعبير غير متوقع «%s»", describe(t))
	return nil
}

func (p *parser) parseListOrMap() ast.Expr {
	t := p.next() // [
	saved := p.noLambda
	p.noLambda = false
	defer func() { p.noLambda = saved }()
	if p.at(token.COLON) && p.peek(1).Kind == token.RBRACK {
		p.next()
		p.next()
		return &ast.MapLit{At: ast.At{Pos: t.Pos}}
	}
	if p.accept(token.RBRACK) {
		return &ast.ListLit{At: ast.At{Pos: t.Pos}}
	}
	first := p.parseExpr()
	if p.accept(token.COLON) {
		m := &ast.MapLit{At: ast.At{Pos: t.Pos}}
		m.Keys = append(m.Keys, first)
		m.Values = append(m.Values, p.parseExpr())
		for p.accept(token.COMMA) && !p.at(token.RBRACK) {
			m.Keys = append(m.Keys, p.parseExpr())
			p.expect(token.COLON, ":")
			m.Values = append(m.Values, p.parseExpr())
		}
		p.expect(token.RBRACK, "]")
		return m
	}
	l := &ast.ListLit{At: ast.At{Pos: t.Pos}, Elems: []ast.Expr{first}}
	for p.accept(token.COMMA) && !p.at(token.RBRACK) {
		l.Elems = append(l.Elems, p.parseExpr())
	}
	p.expect(token.RBRACK, "]")
	return l
}

// acceptAfterNewlines consumes newlines followed by k; otherwise it leaves
// the position unchanged.
func (p *parser) acceptAfterNewlines(k token.Kind) bool {
	j := p.i
	for p.toks[j].Kind == token.NEWLINE {
		j++
	}
	if p.toks[j].Kind == k {
		p.i = j
		p.next()
		return true
	}
	return false
}

func (p *parser) parseIf() ast.Expr {
	t := p.next()
	n := &ast.If{At: ast.At{Pos: t.Pos}}
	n.Cond = p.parseExprNoLambda()
	n.Then = p.parseBlock()
	if p.acceptAfterNewlines(token.ELSE) {
		if p.at(token.IF) {
			n.Else = p.parseIf()
		} else {
			n.Else = p.parseBlock()
		}
	}
	return n
}

func (p *parser) parseMatch() ast.Expr {
	t := p.next()
	m := &ast.Match{At: ast.At{Pos: t.Pos}}
	m.Subject = p.parseExprNoLambda()
	p.expect(token.LBRACE, "{")
	saved := p.noLambda
	p.noLambda = false
	p.skipNL()
	for !p.at(token.RBRACE) && !p.at(token.EOF) {
		c := &ast.Case{At: ast.At{Pos: p.cur().Pos}, Pattern: p.parsePattern()}
		p.expect(token.ARROW, "=>")
		if p.at(token.LBRACE) {
			c.Body = p.parseBlock()
		} else {
			c.Body = p.parseExpr()
		}
		m.Cases = append(m.Cases, c)
		p.endStmt()
		p.skipNL()
	}
	p.expect(token.RBRACE, "}")
	p.noLambda = saved
	return m
}

func (p *parser) parsePattern() *ast.Pattern {
	t := p.cur()
	pat := &ast.Pattern{At: ast.At{Pos: t.Pos}}
	switch t.Kind {
	case token.IDENT:
		p.next()
		if t.Value == "_" {
			pat.Kind = ast.WildcardPattern
			return pat
		}
		pat.Kind, pat.Name = ast.NamePattern, t.Value
		if p.accept(token.LPAREN) {
			pat.HasArgs = true
			for !p.at(token.RPAREN) {
				pat.Binds = append(pat.Binds, p.expectIdent())
				if !p.accept(token.COMMA) {
					break
				}
			}
			p.expect(token.RPAREN, ")")
		}
		return pat
	case token.INT, token.FLOAT, token.STRING, token.TRUE, token.FALSE, token.NULL, token.MINUS:
		pat.Kind = ast.LiteralPattern
		pat.Lit = p.parseUnary()
		return pat
	}
	p.failf(t.Pos, "نمط غير صالح «%s» في «طابق»", describe(t))
	return nil
}

func (p *parser) parseTry() ast.Expr {
	t := p.next()
	tr := &ast.Try{At: ast.At{Pos: t.Pos}}
	tr.Body = p.parseBlock()
	if !p.acceptAfterNewlines(token.CATCH) {
		p.failf(p.cur().Pos, "توقعت «التقط» بعد كتلة «حاول»")
	}
	tr.Var = p.expectIdent()
	tr.Catch = p.parseBlock()
	return tr
}

// ---------------------------------------------------------------- strings

// parseString splits a string token into literal text and {interpolations}.
func (p *parser) parseString(t token.Token) *ast.StrLit {
	s := &ast.StrLit{At: ast.At{Pos: t.Pos}}
	r := []rune(t.Value)
	var text []rune
	flush := func() {
		if len(text) > 0 {
			s.Parts = append(s.Parts, ast.StrPart{Text: string(text)})
			text = nil
		}
	}
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case c == '\\' && i+1 < len(r):
			i++
			switch r[i] {
			case 'n':
				text = append(text, '\n')
			case 't':
				text = append(text, '\t')
			case 'r':
				text = append(text, '\r')
			case '0':
				text = append(text, 0)
			default:
				text = append(text, r[i])
			}
		case c == '{':
			end := matchingBrace(r, i)
			if end < 0 {
				p.failf(t.Pos, "تعبير «{…}» داخل النص لم يُغلق")
			}
			flush()
			s.Parts = append(s.Parts, ast.StrPart{Expr: parseEmbedded(string(r[i+1:end]), t.Pos)})
			i = end
		default:
			text = append(text, c)
		}
	}
	flush()
	return s
}

func matchingBrace(r []rune, open int) int {
	depth := 0
	for i := open; i < len(r); i++ {
		switch r[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		case '"', '«':
			closer := '"'
			if r[i] == '«' {
				closer = '»'
			}
			for i++; i < len(r) && r[i] != closer; i++ {
				if r[i] == '\\' {
					i++
				}
			}
		}
	}
	return -1
}

func parseEmbedded(src string, pos token.Pos) ast.Expr {
	toks, diags := lexer.Tokenize(src)
	if lexer.HasErrors(diags) {
		panic(parseError{pos: pos, msg: "تعبير غير صالح داخل النص: " + firstError(diags)})
	}
	for i := range toks {
		toks[i].Pos = pos
	}
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(parseError); ok {
				panic(parseError{pos: pos, msg: "تعبير غير صالح داخل النص: " + pe.msg})
			}
			panic(r)
		}
	}()
	sp := &parser{toks: toks}
	e := sp.parseExpr()
	sp.skipNL()
	if !sp.at(token.EOF) {
		sp.failf(pos, "تعبير غير صالح داخل النص عند «%s»", describe(sp.cur()))
	}
	return e
}

func firstError(diags []lexer.Diagnostic) string {
	for _, d := range diags {
		if !d.Warning {
			return d.Msg
		}
	}
	return ""
}
