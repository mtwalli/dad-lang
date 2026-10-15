// Package token defines the lexical tokens of the Ḍād (ضاد) language.
package token

import "fmt"

// Kind identifies the type of a token.
type Kind int

const (
	ILLEGAL Kind = iota
	EOF
	NEWLINE

	// Literals
	IDENT
	INT
	FLOAT
	STRING

	// Operators and punctuation
	PLUS     // +
	MINUS    // -
	STAR     // *
	SLASH    // /
	PERCENT  // %
	ASSIGN   // =
	PLUS_EQ  // +=
	MINUS_EQ // -=
	STAR_EQ  // *=
	SLASH_EQ // /=
	EQ       // ==
	NEQ      // !=
	LT       // <
	GT       // >
	LE       // <=
	GE       // >=
	ARROW    // =>
	AND_SYM  // &&
	OR_SYM   // ||
	BANG     // !
	DOT      // .
	RANGE    // ..
	COLON    // :
	COMMA    // , or ،
	SEMI     // ; or ؛
	QUESTION // ? or ؟
	SAFE_DOT // ?. or ؟.
	COALESCE // ?? or ؟؟
	LPAREN   // (
	RPAREN   // )
	LBRACK   // [
	RBRACK   // ]
	LBRACE   // {
	RBRACE   // }

	keywordStart
	// Declarations
	CONST     // ثابت
	VAR       // متغير
	FUNC      // دالة
	TYPE      // نوع
	ENUM      // تعداد
	CONTRACT  // عقد
	EXTENSION // امتداد
	IMPORT    // استورد
	PRIVATE   // خاص
	// Control flow
	IF       // إذا
	ELSE     // وإلا
	MATCH    // طابق
	FOR      // لكل
	IN       // في
	FROM     // من
	TO       // إلى
	UNTIL    // حتى
	STEP     // بخطوة
	WHILE    // طالما
	RETURN   // أرجع
	BREAK    // توقف
	CONTINUE // تابع
	TRY      // حاول
	CATCH    // التقط
	THROW    // ارفع
	// Values and operators
	TRUE  // صح
	FALSE // خطأ
	NULL  // عدم
	AND   // و
	OR    // أو
	NOT   // ليس
	IT    // هو
	AS    // ك
	THIS  // هذا
	// Full-stack
	SERVER    // خادم
	COMPONENT // مكون
	STATE     // حالة
	TABLE     // جدول
	ROUTE     // مسار
	keywordEnd
)

var names = [...]string{
	ILLEGAL: "ILLEGAL", EOF: "EOF", NEWLINE: "NEWLINE",
	IDENT: "IDENT", INT: "INT", FLOAT: "FLOAT", STRING: "STRING",
	PLUS: "+", MINUS: "-", STAR: "*", SLASH: "/", PERCENT: "%",
	ASSIGN: "=", PLUS_EQ: "+=", MINUS_EQ: "-=", STAR_EQ: "*=", SLASH_EQ: "/=",
	EQ: "==", NEQ: "!=", LT: "<", GT: ">", LE: "<=", GE: ">=", ARROW: "=>",
	AND_SYM: "&&", OR_SYM: "||", BANG: "!", DOT: ".", RANGE: "..", COLON: ":",
	COMMA: ",", SEMI: ";", QUESTION: "?", SAFE_DOT: "?.", COALESCE: "??",
	LPAREN: "(", RPAREN: ")", LBRACK: "[", RBRACK: "]", LBRACE: "{", RBRACE: "}",
}

// Keywords maps the normalized Arabic spelling of each reserved word to its kind.
var Keywords = map[string]Kind{
	"ثابت": CONST, "متغير": VAR, "دالة": FUNC, "نوع": TYPE, "تعداد": ENUM,
	"عقد": CONTRACT, "امتداد": EXTENSION, "استورد": IMPORT, "خاص": PRIVATE,
	"إذا": IF, "وإلا": ELSE, "طابق": MATCH, "لكل": FOR, "في": IN, "من": FROM,
	"إلى": TO, "حتى": UNTIL, "بخطوة": STEP, "طالما": WHILE, "أرجع": RETURN,
	"توقف": BREAK, "تابع": CONTINUE, "حاول": TRY, "التقط": CATCH, "ارفع": THROW,
	"صح": TRUE, "خطأ": FALSE, "عدم": NULL, "و": AND, "أو": OR, "ليس": NOT,
	"هو": IT, "ك": AS, "هذا": THIS,
	"خادم": SERVER, "مكون": COMPONENT, "حالة": STATE, "جدول": TABLE, "مسار": ROUTE,
}

var keywordNames = func() map[Kind]string {
	m := make(map[Kind]string, len(Keywords))
	for s, k := range Keywords {
		m[k] = s
	}
	return m
}()

func (k Kind) String() string {
	if k.IsKeyword() {
		return keywordNames[k]
	}
	if int(k) < len(names) && names[k] != "" {
		return names[k]
	}
	return fmt.Sprintf("Kind(%d)", int(k))
}

// IsKeyword reports whether k is a reserved word.
func (k Kind) IsKeyword() bool { return k > keywordStart && k < keywordEnd }

// Pos is a 1-based line and column (columns count Unicode code points).
type Pos struct {
	Line, Col int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Token is a single lexical token.
type Token struct {
	Kind  Kind
	Lit   string // exact source text
	Value string // normalized value: identifier/keyword without diacritics, number in ASCII digits, decoded string body
	Pos   Pos
}

func (t Token) String() string {
	switch t.Kind {
	case IDENT, INT, FLOAT, STRING:
		return fmt.Sprintf("%s %s(%q)", t.Pos, t.Kind.names(), t.Value)
	}
	if t.Kind.IsKeyword() {
		return fmt.Sprintf("%s KEYWORD(%s)", t.Pos, t.Kind)
	}
	return fmt.Sprintf("%s %s", t.Pos, t.Kind)
}

func (k Kind) names() string { return names[k] }
