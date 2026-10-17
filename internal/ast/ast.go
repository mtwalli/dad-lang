// Package ast defines the syntax tree of Ḍād (ضاد) programs.
package ast

import "github.com/mtwalli/dad-lang/internal/token"

// Node is any syntax tree node.
type Node interface{ P() token.Pos }

// Expr is an expression node.
type Expr interface {
	Node
	exprNode()
}

// Stmt is a statement or declaration node.
type Stmt interface {
	Node
	stmtNode()
}

// At records a node's source position.
type At struct{ Pos token.Pos }

func (a At) P() token.Pos { return a.Pos }

// ---------------------------------------------------------------- types

// TypeKind distinguishes the shapes a type reference can take.
type TypeKind int

const (
	NamedType TypeKind = iota // صحيح, نقطة, …
	ListType                  // [T]
	MapType                   // [K: V]
	FuncType                  // (A, B) => R
)

// TypeRef is a type written in source code.
type TypeRef struct {
	At
	Kind     TypeKind
	Name     string     // NamedType
	Args     []*TypeRef // generic arguments, or function parameter types
	Elem     *TypeRef   // ListType element, MapType value, FuncType result
	Key      *TypeRef   // MapType key
	Optional bool       // T؟
}

// ---------------------------------------------------------------- expressions

type (
	IntLit struct {
		At
		Value int64
	}
	FloatLit struct {
		At
		Value float64
	}
	BoolLit struct {
		At
		Value bool
	}
	NullLit  struct{ At }
	ThisExpr struct{ At } // هذا
	ItExpr   struct{ At } // هو (implicit lambda parameter)
	Ident    struct {
		At
		Name string
	}

	// StrLit is a string with optional {interpolations}.
	StrLit struct {
		At
		Parts []StrPart
	}

	ListLit struct {
		At
		Elems []Expr
	}

	MapLit struct {
		At
		Keys, Values []Expr
	}

	Unary struct {
		At
		Op token.Kind
		X  Expr
	}

	Binary struct {
		At
		Op   token.Kind
		L, R Expr
	}

	// Call is f(args) with an optional trailing lambda.
	Call struct {
		At
		Fn     Expr
		Args   []Arg
		Lambda *Lambda
	}

	// Member is x.name or x؟.name.
	Member struct {
		At
		X    Expr
		Name string
		Safe bool
	}

	Index struct {
		At
		X, I Expr
	}

	Lambda struct {
		At
		Params []string
		Body   *Block
	}

	If struct {
		At
		Cond Expr
		Then *Block
		Else Expr // *If, *Block or nil
	}

	Match struct {
		At
		Subject Expr
		Cases   []*Case
	}

	Try struct {
		At
		Body  *Block
		Var   string
		Catch *Block
	}

	// RangeExpr is من a إلى b, من a حتى b, or a..b.
	RangeExpr struct {
		At
		From, To  Expr
		Inclusive bool
		Step      Expr
	}

	TypeTest struct {
		At
		X    Expr
		Type *TypeRef
	}

	Cast struct {
		At
		X    Expr
		Type *TypeRef
	}

	Throw struct {
		At
		X Expr
	}

	Block struct {
		At
		Stmts []Stmt
	}
)

// StrPart is either literal text or an interpolated expression.
type StrPart struct {
	Text string
	Expr Expr
}

// Arg is a call argument, optionally named.
type Arg struct {
	Name  string
	Value Expr
}

// PatternKind classifies match patterns.
type PatternKind int

const (
	WildcardPattern PatternKind = iota // _
	LiteralPattern                     // 1، "نص"، صح
	NamePattern                        // دائرة(ق) or a binding name
)

type Pattern struct {
	At
	Kind    PatternKind
	Name    string
	Binds   []string
	HasArgs bool
	Lit     Expr
}

type Case struct {
	At
	Pattern *Pattern
	Body    Expr // *Block or an expression
}

func (*IntLit) exprNode()    {}
func (*FloatLit) exprNode()  {}
func (*BoolLit) exprNode()   {}
func (*NullLit) exprNode()   {}
func (*ThisExpr) exprNode()  {}
func (*ItExpr) exprNode()    {}
func (*Ident) exprNode()     {}
func (*StrLit) exprNode()    {}
func (*ListLit) exprNode()   {}
func (*MapLit) exprNode()    {}
func (*Unary) exprNode()     {}
func (*Binary) exprNode()    {}
func (*Call) exprNode()      {}
func (*Member) exprNode()    {}
func (*Index) exprNode()     {}
func (*Lambda) exprNode()    {}
func (*If) exprNode()        {}
func (*Match) exprNode()     {}
func (*Try) exprNode()       {}
func (*RangeExpr) exprNode() {}
func (*TypeTest) exprNode()  {}
func (*Cast) exprNode()      {}
func (*Throw) exprNode()     {}
func (*Block) exprNode()     {}

// ---------------------------------------------------------------- statements

type (
	Let struct {
		At
		Mutable bool
		Name    string
		Type    *TypeRef
		Value   Expr
	}

	Assign struct {
		At
		Target Expr
		Op     token.Kind // ASSIGN, PLUS_EQ, …
		Value  Expr
	}

	ExprStmt struct {
		At
		X Expr
	}

	Return struct {
		At
		X Expr
	}

	Break    struct{ At }
	Continue struct{ At }

	For struct {
		At
		Var  string
		Iter Expr
		Body *Block
	}

	While struct {
		At
		Cond Expr
		Body *Block
	}

	FuncDecl struct {
		At
		Name   string
		Params []*Param
		Result *TypeRef
		Body   Expr // *Block, or an expression for "= expr" bodies
		Server bool
	}

	TypeDecl struct {
		At
		Name      string
		Fields    []*Param
		Contracts []*TypeRef
		Methods   []*FuncDecl
	}

	EnumDecl struct {
		At
		Name     string
		Variants []*Variant
	}

	ContractDecl struct {
		At
		Name string
	}

	ExtensionDecl struct {
		At
		Type    *TypeRef
		Methods []*FuncDecl
	}

	Import struct {
		At
		Names []string
		From  string
	}

	// TableDecl is a database table: «جدول مهمة { عنوان: نص، منجزة = خطأ }».
	// A field has a type, a default value, or both.
	TableDecl struct {
		At
		Name   string
		Fields []*Param
	}

	// RouteDecl is a public web route: «مسار جلب "/api/مهام/{رقم}" = …».
	// Method is the HTTP method (GET, POST, PUT, DELETE); {name} in the path binds a value.
	RouteDecl struct {
		At
		Method string
		Path   string
		Body   Expr // *Block, or an expression for "= expr" bodies
	}

	// Unsupported is a full-stack declaration (مكون، حالة) that
	// parses but cannot run yet.
	Unsupported struct {
		At
		Keyword string
		Name    string
	}
)

type Param struct {
	At
	Name    string
	Type    *TypeRef
	Default Expr
}

type Variant struct {
	At
	Name   string
	Fields []*Param
}

func (*Let) stmtNode()           {}
func (*Assign) stmtNode()        {}
func (*ExprStmt) stmtNode()      {}
func (*Return) stmtNode()        {}
func (*Break) stmtNode()         {}
func (*Continue) stmtNode()      {}
func (*For) stmtNode()           {}
func (*While) stmtNode()         {}
func (*FuncDecl) stmtNode()      {}
func (*TypeDecl) stmtNode()      {}
func (*EnumDecl) stmtNode()      {}
func (*ContractDecl) stmtNode()  {}
func (*ExtensionDecl) stmtNode() {}
func (*Import) stmtNode()        {}
func (*Unsupported) stmtNode()   {}
func (*TableDecl) stmtNode()     {}
func (*RouteDecl) stmtNode()     {}

// File is a parsed source file.
type File struct {
	Stmts []Stmt
}
