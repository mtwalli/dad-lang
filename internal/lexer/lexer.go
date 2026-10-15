// Package lexer turns Ḍād (ضاد) source text into tokens.
//
// It implements SPEC §2: Arabic and Latin digits and punctuation, diacritic- and
// tatweel-insensitive names, rejection of bidi/zero-width control characters
// outside strings, and Kotlin-style newline handling (newlines end statements,
// except inside parentheses and square brackets).
package lexer

import (
	"fmt"
	"strings"

	"github.com/mtwalli/dad-lang/internal/token"
)

// Diagnostic is an error or warning found while lexing. Messages are in Arabic.
type Diagnostic struct {
	Pos     token.Pos
	Msg     string
	Warning bool
}

func (d Diagnostic) String() string {
	kind := "خطأ"
	if d.Warning {
		kind = "تنبيه"
	}
	return fmt.Sprintf("%s [السطر %s، العمود %s]: %s", kind,
		ToArabicDigits(fmt.Sprint(d.Pos.Line)), ToArabicDigits(fmt.Sprint(d.Pos.Col)), d.Msg)
}

// Tokenize lexes src completely. The token list always ends with EOF.
func Tokenize(src string) ([]token.Token, []Diagnostic) {
	runes := []rune(src)
	if len(runes) > 0 && runes[0] == 0xFEFF {
		runes = runes[1:]
	}
	l := &lexer{src: composeArabic(runes), line: 1, col: 1}
	l.run()
	return l.toks, l.diags
}

// HasErrors reports whether diags contains at least one non-warning.
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if !d.Warning {
			return true
		}
	}
	return false
}

type lexer struct {
	src       []rune
	i         int
	line, col int
	toks      []token.Token
	diags     []Diagnostic
	nest      []rune // open brackets: '(' '[' '{'
}

func (l *lexer) peek(n int) rune {
	if l.i+n < len(l.src) {
		return l.src[l.i+n]
	}
	return 0
}

func (l *lexer) advance() rune {
	r := l.src[l.i]
	l.i++
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *lexer) pos() token.Pos { return token.Pos{Line: l.line, Col: l.col} }

func (l *lexer) errorf(p token.Pos, format string, args ...any) {
	l.diags = append(l.diags, Diagnostic{Pos: p, Msg: fmt.Sprintf(format, args...)})
}

func (l *lexer) warnf(p token.Pos, format string, args ...any) {
	l.diags = append(l.diags, Diagnostic{Pos: p, Msg: fmt.Sprintf(format, args...), Warning: true})
}

func (l *lexer) emit(k token.Kind, lit, value string, p token.Pos) {
	l.toks = append(l.toks, token.Token{Kind: k, Lit: lit, Value: value, Pos: p})
}

func (l *lexer) insideParens() bool {
	n := len(l.nest)
	return n > 0 && (l.nest[n-1] == '(' || l.nest[n-1] == '[')
}

func (l *lexer) run() {
	for l.i < len(l.src) {
		p := l.pos()
		r := l.peek(0)
		switch {
		case r == ' ' || r == '\t' || r == '\r' || r == 0x00A0:
			l.advance()
		case r == '\n':
			l.advance()
			if !l.insideParens() && len(l.toks) > 0 && l.toks[len(l.toks)-1].Kind != token.NEWLINE {
				l.emit(token.NEWLINE, "\n", "", p)
			}
		case r == '/' && l.peek(1) == '/':
			l.lineComment()
		case r == '/' && l.peek(1) == '*':
			l.blockComment()
		case isForbidden(r):
			l.advance()
			l.errorf(p, "محرف تحكم خفي (U+%04X) غير مسموح خارج النصوص — قد يجعل الشيفرة تظهر بغير ما تعمل", r)
		case isNameStart(r):
			l.name()
		case isASCIIDigit(r) || isArabicDigit(r) || isPersianDigit(r):
			l.number()
		case r == '"' || r == '«':
			l.str()
		default:
			l.operator()
		}
	}
	if len(l.nest) > 0 {
		l.errorf(l.pos(), "القوس «%c» لم يُغلق", l.nest[len(l.nest)-1])
	}
	if n := len(l.toks); n > 0 && l.toks[n-1].Kind != token.NEWLINE {
		l.emit(token.NEWLINE, "", "", l.pos())
	}
	l.emit(token.EOF, "", "", l.pos())
}

func (l *lexer) lineComment() {
	for l.i < len(l.src) && l.peek(0) != '\n' {
		p := l.pos()
		if r := l.advance(); isForbidden(r) {
			l.errorf(p, "محرف تحكم خفي (U+%04X) غير مسموح داخل التعليقات", r)
		}
	}
}

func (l *lexer) blockComment() {
	start := l.pos()
	l.advance()
	l.advance()
	for l.i < len(l.src) {
		if l.peek(0) == '*' && l.peek(1) == '/' {
			l.advance()
			l.advance()
			return
		}
		p := l.pos()
		if r := l.advance(); isForbidden(r) {
			l.errorf(p, "محرف تحكم خفي (U+%04X) غير مسموح داخل التعليقات", r)
		}
	}
	l.errorf(start, "تعليق لم يُغلق: ينقصه «*/»")
}

func (l *lexer) name() {
	p := l.pos()
	startIdx := l.i
	for l.i < len(l.src) && isNamePart(l.peek(0)) {
		l.advance()
	}
	lit := string(l.src[startIdx:l.i])
	val := normalizeName(lit)
	if k, ok := token.Keywords[val]; ok {
		l.emit(k, lit, val, p)
		return
	}
	if kw, ok := foldedKeywords[foldLetters(val)]; ok {
		l.warnf(p, "«%s» ليست كلمة محجوزة — هل تقصد «%s»؟", val, kw)
	}
	l.emit(token.IDENT, lit, val, p)
}

var foldedKeywords = func() map[string]string {
	m := map[string]string{}
	for kw := range token.Keywords {
		if f := foldLetters(kw); f != kw {
			m[f] = kw
		}
	}
	return m
}()

func (l *lexer) number() {
	p := l.pos()
	startIdx := l.i
	first := l.peek(0)
	if isPersianDigit(first) {
		for l.i < len(l.src) && isPersianDigit(l.peek(0)) {
			l.advance()
		}
		l.errorf(p, "الأرقام الفارسية غير مدعومة؛ استخدم ٠١٢٣٤٥٦٧٨٩ أو 0123456789")
		l.emit(token.ILLEGAL, string(l.src[startIdx:l.i]), "", p)
		return
	}
	arabic := isArabicDigit(first)
	sameScript := func(r rune) bool {
		if arabic {
			return isArabicDigit(r)
		}
		return isASCIIDigit(r)
	}
	isDigit := func(r rune) bool { return isASCIIDigit(r) || isArabicDigit(r) }

	var b strings.Builder
	kind := token.INT
	mixed := false
	for l.i < len(l.src) {
		r := l.peek(0)
		switch {
		case isDigit(r):
			if !sameScript(r) {
				mixed = true
			}
			if isArabicDigit(r) {
				b.WriteRune('0' + (r - '٠'))
			} else {
				b.WriteRune(r)
			}
			l.advance()
		case (r == '_' || r == '٬') && isDigit(l.peek(1)):
			l.advance()
		case (r == '٫' || (r == '.' && l.peek(1) != '.')) && isDigit(l.peek(1)) && kind == token.INT:
			kind = token.FLOAT
			b.WriteByte('.')
			l.advance()
		default:
			goto done
		}
	}
done:
	if mixed {
		l.errorf(p, "لا يجوز خلط الأرقام العربية (٠-٩) واللاتينية (0-9) في عدد واحد")
	}
	l.emit(kind, string(l.src[startIdx:l.i]), b.String(), p)
}

// str lexes "…", """…""" and «…». The token value is the raw body between the
// delimiters (escapes and {interpolations} are kept as written; the parser
// splits interpolations). The lexer validates escapes and balances braces,
// including strings nested inside interpolations.
func (l *lexer) str() {
	p := l.pos()
	startIdx := l.i
	open := l.advance()
	closeRune, triple := '"', false
	if open == '«' {
		closeRune = '»'
	} else if l.peek(0) == '"' && l.peek(1) == '"' {
		triple = true
		l.advance()
		l.advance()
	}
	bodyStart := l.i
	for l.i < len(l.src) {
		r := l.peek(0)
		switch {
		case r == '\\':
			l.escape()
		case r == '{':
			if !l.interpolation() {
				return
			}
		case r == '\n' && !triple:
			l.errorf(p, "نص لم يُغلق قبل نهاية السطر")
			l.emit(token.STRING, string(l.src[startIdx:l.i]), string(l.src[bodyStart:l.i]), p)
			return
		case triple && r == '"' && l.peek(1) == '"' && l.peek(2) == '"':
			body := string(l.src[bodyStart:l.i])
			l.advance()
			l.advance()
			l.advance()
			l.emit(token.STRING, string(l.src[startIdx:l.i]), body, p)
			return
		case !triple && r == closeRune:
			body := string(l.src[bodyStart:l.i])
			l.advance()
			l.emit(token.STRING, string(l.src[startIdx:l.i]), body, p)
			return
		default:
			l.advance()
		}
	}
	l.errorf(p, "نص لم يُغلق قبل نهاية الملف")
	l.emit(token.STRING, string(l.src[startIdx:l.i]), string(l.src[bodyStart:l.i]), p)
}

func (l *lexer) escape() {
	p := l.pos()
	l.advance() // backslash
	if l.i >= len(l.src) {
		return
	}
	switch r := l.advance(); r {
	case 'n', 't', 'r', '\\', '"', '{', '}', '«', '»', '0':
	default:
		l.errorf(p, "تسلسل هروب غير معروف «\\%c»", r)
	}
}

// interpolation skips a {…} section inside a string, tracking nested braces and
// nested string literals. It returns false if the source ended first.
func (l *lexer) interpolation() bool {
	p := l.pos()
	l.advance() // {
	depth := 1
	for l.i < len(l.src) {
		r := l.peek(0)
		switch {
		case r == '{':
			depth++
			l.advance()
		case r == '}':
			depth--
			l.advance()
			if depth == 0 {
				return true
			}
		case r == '"' || r == '«':
			// Nested string: lex it into a throwaway lexer state.
			saved := len(l.toks)
			l.str()
			l.toks = l.toks[:saved]
		case r == '\n':
			l.errorf(p, "تعبير «{…}» داخل النص لم يُغلق")
			return true
		default:
			l.advance()
		}
	}
	l.errorf(p, "تعبير «{…}» داخل النص لم يُغلق")
	return false
}

func (l *lexer) operator() {
	p := l.pos()
	r := l.advance()
	n := l.peek(0)
	isQ := func(c rune) bool { return c == '?' || c == '؟' }
	two := func(k token.Kind) {
		lit := string([]rune{r, l.advance()})
		l.emit(k, lit, "", p)
	}
	one := func(k token.Kind) { l.emit(k, string(r), "", p) }

	switch r {
	case '+':
		if n == '=' {
			two(token.PLUS_EQ)
		} else {
			one(token.PLUS)
		}
	case '-':
		if n == '=' {
			two(token.MINUS_EQ)
		} else {
			one(token.MINUS)
		}
	case '*':
		if n == '=' {
			two(token.STAR_EQ)
		} else {
			one(token.STAR)
		}
	case '/':
		if n == '=' {
			two(token.SLASH_EQ)
		} else {
			one(token.SLASH)
		}
	case '%':
		one(token.PERCENT)
	case '=':
		switch n {
		case '=':
			two(token.EQ)
		case '>':
			two(token.ARROW)
		default:
			one(token.ASSIGN)
		}
	case '!':
		if n == '=' {
			two(token.NEQ)
		} else {
			one(token.BANG)
		}
	case '<':
		if n == '=' {
			two(token.LE)
		} else {
			one(token.LT)
		}
	case '>':
		if n == '=' {
			two(token.GE)
		} else {
			one(token.GT)
		}
	case '&':
		if n == '&' {
			two(token.AND_SYM)
		} else {
			l.errorf(p, "هل تقصد «&&» أو «و»؟")
			one(token.ILLEGAL)
		}
	case '|':
		if n == '|' {
			two(token.OR_SYM)
		} else {
			l.errorf(p, "هل تقصد «||» أو «أو»؟")
			one(token.ILLEGAL)
		}
	case '.':
		if n == '.' {
			two(token.RANGE)
		} else {
			one(token.DOT)
		}
	case ':':
		one(token.COLON)
	case ',', '،':
		one(token.COMMA)
	case ';', '؛':
		one(token.SEMI)
	case '?', '؟':
		switch {
		case n == '.':
			two(token.SAFE_DOT)
		case isQ(n):
			two(token.COALESCE)
		default:
			one(token.QUESTION)
		}
	case '(', '[', '{':
		l.nest = append(l.nest, r)
		one(map[rune]token.Kind{'(': token.LPAREN, '[': token.LBRACK, '{': token.LBRACE}[r])
	case ')', ']', '}':
		want := map[rune]rune{')': '(', ']': '[', '}': '{'}[r]
		if k := len(l.nest); k == 0 || l.nest[k-1] != want {
			l.errorf(p, "القوس «%c» لا يقابله قوس مفتوح", r)
		} else {
			l.nest = l.nest[:k-1]
		}
		one(map[rune]token.Kind{')': token.RPAREN, ']': token.RBRACK, '}': token.RBRACE}[r])
	default:
		l.errorf(p, "محرف غير متوقع «%c» (U+%04X)", r, r)
		one(token.ILLEGAL)
	}
}
