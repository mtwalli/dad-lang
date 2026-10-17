package interp

import (
	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

// compileWhere turns a «حيث» lambda into a Cond when it only compares fields of
// the record with values known before the query runs, such as
//
//	مهمة.حيث { هو.منجزة و هو.أولوية >= حد }
//
// Anything else (method calls on the record, arithmetic on fields, …) returns
// false, and the interpreter runs the lambda on every row instead.
func (in *Interp) compileWhere(fn Value, t *StructType) (*Cond, bool) {
	f, ok := fn.(*Func)
	if !ok || !f.IsLam || len(f.Lambda) > 1 || f.HasRecv {
		return nil, false
	}
	b, ok := f.Body.(*ast.Block)
	if !ok || len(b.Stmts) != 1 {
		return nil, false
	}
	es, ok := b.Stmts[0].(*ast.ExprStmt)
	if !ok {
		return nil, false
	}
	param := ""
	if len(f.Lambda) == 1 {
		param = f.Lambda[0]
	}
	q := &queryCompiler{in: in, t: t, param: param, env: f.Env}
	return q.cond(es.X)
}

type queryCompiler struct {
	in    *Interp
	t     *StructType
	param string // the lambda's parameter, or "" for هو
	env   *Env
}

var compareOps = map[token.Kind]string{
	token.EQ: "eq", token.NEQ: "ne", token.LT: "lt", token.LE: "le", token.GT: "gt", token.GE: "ge",
}

var flipped = map[string]string{"eq": "eq", "ne": "ne", "lt": "gt", "le": "ge", "gt": "lt", "ge": "le"}

func (q *queryCompiler) cond(e ast.Expr) (*Cond, bool) {
	switch x := e.(type) {
	case *ast.Binary:
		switch x.Op {
		case token.AND, token.AND_SYM, token.OR, token.OR_SYM:
			l, ok1 := q.cond(x.L)
			r, ok2 := q.cond(x.R)
			if !ok1 || !ok2 {
				return nil, false
			}
			op := "and"
			if x.Op == token.OR || x.Op == token.OR_SYM {
				op = "or"
			}
			return &Cond{Op: op, L: l, R: r}, true
		}
		op, ok := compareOps[x.Op]
		if !ok {
			return nil, false
		}
		if f, ok := q.field(x.L); ok {
			return q.compare(f, op, x.R)
		}
		if f, ok := q.field(x.R); ok {
			return q.compare(f, flipped[op], x.L)
		}
	case *ast.Unary:
		if x.Op == token.NOT || x.Op == token.BANG {
			if c, ok := q.cond(x.X); ok {
				return &Cond{Op: "not", L: c}, true
			}
		}
	default:
		if f, ok := q.field(e); ok && fieldKind(f.Type) == "bool" && !f.Type.Optional {
			return &Cond{Op: "true", Field: f.Name}, true
		}
	}
	return nil, false
}

// field recognises «هو.حقل» (or «م.حقل» for a named parameter).
func (q *queryCompiler) field(e ast.Expr) (*ast.Param, bool) {
	m, ok := e.(*ast.Member)
	if !ok || m.Safe || !q.isParam(m.X) {
		return nil, false
	}
	for _, f := range q.t.Fields {
		if f.Name == m.Name && f.Type != nil {
			return f, true
		}
	}
	return nil, false
}

func (q *queryCompiler) isParam(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.ItExpr:
		return q.param == ""
	case *ast.Ident:
		return q.param != "" && x.Name == q.param
	}
	return false
}

// compare builds a comparison when the other side is a literal or a plain
// variable, and its value has the field's type.
func (q *queryCompiler) compare(f *ast.Param, op string, other ast.Expr) (*Cond, bool) {
	v, ok := q.value(other)
	if !ok {
		return nil, false
	}
	kind := fieldKind(f.Type)
	switch val := v.(type) {
	case nil:
		if (op != "eq" && op != "ne") || !f.Type.Optional {
			return nil, false
		}
	case int64, float64:
		if kind != "int" && kind != "float" {
			return nil, false
		}
	case string:
		if kind != "text" {
			return nil, false
		}
	case bool:
		if kind != "bool" || (op != "eq" && op != "ne") {
			return nil, false
		}
	default:
		_ = val
		return nil, false
	}
	if f.Type.Optional && op != "eq" && op != "ne" {
		return nil, false // ordering against عدم is an error in Ḍād; keep that behaviour
	}
	return &Cond{Op: op, Field: f.Name, Value: v}, true
}

// value evaluates the side of a comparison that doesn't involve the record.
// Only literals, variables, their fields and negation are allowed, so evaluating
// early can't change what the program does.
func (q *queryCompiler) value(e ast.Expr) (Value, bool) {
	if !q.pure(e) {
		return nil, false
	}
	v := q.in.eval(e, q.env)
	switch v.(type) {
	case nil, int64, float64, string, bool:
		return v, true
	}
	return nil, false
}

func (q *queryCompiler) pure(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.NullLit:
		return true
	case *ast.StrLit:
		for _, p := range x.Parts {
			if p.Expr != nil {
				return false
			}
		}
		return true
	case *ast.Ident:
		return x.Name != q.param && q.env.lookup(x.Name) != nil
	case *ast.Member:
		_, isStruct := q.memberBase(x.X)
		return !x.Safe && isStruct
	case *ast.Unary:
		return x.Op == token.MINUS && q.pure(x.X)
	}
	return false
}

// memberBase allows «إعدادات.حد» where the base is a pure value holding a record.
func (q *queryCompiler) memberBase(e ast.Expr) (Value, bool) {
	if !q.pure(e) {
		return nil, false
	}
	_, ok := q.in.eval(e, q.env).(*Struct)
	return nil, ok
}

// fieldKind is how a field's type is stored.
func fieldKind(t *ast.TypeRef) string {
	if t == nil || t.Kind != ast.NamedType {
		return "json"
	}
	switch t.Name {
	case "صحيح":
		return "int"
	case "عشري":
		return "float"
	case "نص":
		return "text"
	case "منطقي":
		return "bool"
	}
	return "json"
}
