// Package format lays out Ḍād (ضاد) source code in one standard way, following
// the Kotlin coding conventions (what ktlint and IntelliJ do for Kotlin):
//
//   - four spaces of indentation per open bracket, worked out from the code;
//   - in a block that spans lines, «{» ends its line and «}» starts its own line,
//     and «} وإلا {» / «} التقط» stay together;
//   - one statement per line: «؛» between statements becomes a line break;
//   - one space around operators and after commas, none inside ( ) and [ ],
//     one inside { } on a single line;
//   - lines longer than 100 columns are wrapped at the outermost ( or [, one
//     argument per line;
//   - a blank line before and after a declaration that spans lines, at most one
//     blank line in a row, no spaces at line ends, and a final newline;
//   - comments stay with the code they're on; multi-line strings are untouched.
//
// Formatting never changes what a program means: the result must have the same
// tokens (apart from «؛» and line breaks) and must parse without errors.
package format

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/parser"
	"github.com/mtwalli/dad-lang/internal/token"
)

const (
	indentUnit = "    "
	maxWidth   = 100
)

// ErrSyntax is returned (wrapped) when the source has errors and can't be formatted.
var ErrSyntax = errors.New("لا يمكن تنسيق ملف فيه أخطاء")

// Source formats a whole file.
func Source(src string) (string, error) {
	src = strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n")
	src = strings.TrimPrefix(src, "\uFEFF")
	if _, diags := parser.ParseFile(src); lexer.HasErrors(diags) {
		for _, d := range diags {
			if !d.Warning {
				return src, fmt.Errorf("%w — %s", ErrSyntax, d)
			}
		}
	}
	toks, _ := lexer.Tokenize(src)
	items, err := scan(src, toks)
	if err != nil {
		return src, err
	}
	layout(items)
	out := print(items)
	for range 20 { // wrap long lines, outermost brackets first
		if !wrapLongLine(items) {
			break
		}
		out = print(items)
	}
	if err := check(toks, out); err != nil {
		return src, err
	}
	return out, nil
}

// ---------------------------------------------------------------- items

// item is a token or a comment, with the line breaks that come before it.
type item struct {
	tok     token.Token // Kind is ILLEGAL for a comment
	text    string      // the exact source text
	comment bool
	breaks  int  // 0: same line as the previous item, 1: next line, 2: after a blank line
	drop    bool // a «؛» that becomes a line break

	match   int  // for a bracket, the index of the other one; -1 otherwise
	header  int  // for «{», the item its first line ends with (the «{», or «=>» after lambda parameters)
	isMatch bool // «{» of a طابق

	// filled in by print
	line, col int
}

func (it *item) is(k token.Kind) bool { return !it.comment && !it.drop && it.tok.Kind == k }

// scan turns the source into items: the lexer's tokens plus the comments
// between them, and how many line breaks precede each one.
func scan(src string, toks []token.Token) ([]*item, error) {
	lines := strings.Split(src, "\n")
	lineStart := make([]int, len(lines))
	off := 0
	for i, l := range lines {
		lineStart[i] = off
		off += len(l) + 1
	}
	var items []*item
	prevEnd := 0
	pending := 0 // line breaks after the last comment of a gap, for the next token
	addGap := func(gap string) {
		// comments in the text between two tokens
		breaks := 0
		for i := 0; i < len(gap); {
			switch {
			case gap[i] == '\n':
				breaks++
				i++
			case strings.HasPrefix(gap[i:], "//"):
				j := strings.IndexByte(gap[i:], '\n')
				if j < 0 {
					j = len(gap) - i
				}
				items = append(items, &item{text: strings.TrimRight(gap[i:i+j], " \t"), comment: true, breaks: min(breaks, 2), match: -1})
				breaks = 0
				i += j
			case strings.HasPrefix(gap[i:], "/*"):
				j := strings.Index(gap[i+2:], "*/")
				end := len(gap)
				if j >= 0 {
					end = i + 2 + j + 2
				}
				items = append(items, &item{text: gap[i:end], comment: true, breaks: min(breaks, 2), match: -1})
				breaks = 0
				i = end
			default:
				i++
			}
		}
		pending = breaks
	}
	for _, t := range toks {
		if t.Kind == token.NEWLINE {
			continue
		}
		var start int
		if t.Kind == token.EOF {
			start = len(src)
		} else {
			line := t.Pos.Line - 1
			if line < 0 || line >= len(lines) {
				return nil, fmt.Errorf("خطأ داخلي في المنسّق: موضع غير متوقع %s", t.Pos)
			}
			start = lineStart[line] + runeOffset(lines[line], t.Pos.Col-1)
			if !strings.HasPrefix(src[start:], t.Lit) {
				return nil, fmt.Errorf("خطأ داخلي في المنسّق عند %s", t.Pos)
			}
		}
		addGap(src[prevEnd:start])
		if t.Kind == token.EOF {
			break
		}
		items = append(items, &item{tok: t, text: t.Lit, breaks: min(pending, 2), match: -1})
		prevEnd = start + len(t.Lit)
	}
	if len(items) > 0 {
		items[0].breaks = 0
	}
	return items, nil
}

func runeOffset(s string, n int) int {
	off := 0
	for i := 0; i < n && off < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[off:])
		off += size
	}
	return off
}

// ---------------------------------------------------------------- layout rules

func isOpen(k token.Kind) bool  { return k == token.LPAREN || k == token.LBRACK || k == token.LBRACE }
func isClose(k token.Kind) bool { return k == token.RPAREN || k == token.RBRACK || k == token.RBRACE }

func (it *item) opens() bool  { return !it.comment && isOpen(it.tok.Kind) }
func (it *item) closes() bool { return !it.comment && isClose(it.tok.Kind) }

var declKeywords = map[token.Kind]bool{
	token.FUNC: true, token.TYPE: true, token.ENUM: true, token.CONTRACT: true, token.EXTENSION: true,
	token.TABLE: true, token.ROUTE: true, token.COMPONENT: true, token.SERVER: true, token.PRIVATE: true,
}

// isPage reports a «صفحة "/…"» declaration: «صفحة» is a keyword only there.
func isPage(items []*item, start, end int) bool {
	t := items[start].tok
	return t.Kind == token.IDENT && t.Value == "صفحة" && start < end && items[start+1].tok.Kind == token.STRING
}

func layout(items []*item) {
	// Match brackets, and find the «{» of each طابق.
	var stack []int
	matchAt := -1 // depth at which a طابق is waiting for its «{»
	for i, it := range items {
		switch {
		case it.is(token.MATCH):
			matchAt = len(stack)
		case it.opens():
			if it.tok.Kind == token.LBRACE && matchAt == len(stack) {
				it.isMatch = true
				matchAt = -1
			}
			stack = append(stack, i)
		case it.closes() && len(stack) > 0:
			o := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			items[o].match, it.match = i, o
		}
	}

	// One statement per line: «؛» between statements becomes a line break.
	stack = stack[:0]
	for i, it := range items {
		switch {
		case it.opens():
			stack = append(stack, i)
		case it.closes() && len(stack) > 0:
			stack = stack[:len(stack)-1]
		case it.is(token.SEMI) && (len(stack) == 0 || items[stack[len(stack)-1]].tok.Kind == token.LBRACE):
			it.drop = true
			if n := next(items, i); n >= 0 && !(items[n].comment && items[n].breaks == 0) {
				items[n].breaks = max(items[n].breaks, 1)
			}
		}
	}

	// Blocks that span lines: «{» (or the lambda's «=>») ends its line, «}» starts one.
	for i, it := range items {
		if !it.is(token.LBRACE) || it.match < 0 {
			continue
		}
		it.header = i
		if !it.isMatch {
			if a := lambdaArrow(items, i); a >= 0 {
				it.header = a
			}
		}
		end := it.match
		multi := false
		for j := i + 1; j <= end; j++ {
			if items[j].breaks > 0 {
				multi = true
				break
			}
		}
		if !multi {
			continue
		}
		if next(items, i) == end { // «{ }» written over two lines
			items[end].breaks = 0
			continue
		}
		after := next(items, it.header)
		if after >= 0 && items[after].comment && items[after].breaks == 0 {
			after = next(items, after) // keep a comment on the «{» line
		}
		if after >= 0 && after != end {
			items[after].breaks = max(items[after].breaks, 1)
		}
		items[end].breaks = max(items[end].breaks, 1)
	}

	// «} وإلا {» and «} التقط» stay on one line.
	for i, it := range items {
		if it.is(token.ELSE) || it.is(token.CATCH) {
			if p := prev(items, i); p >= 0 && items[p].is(token.RBRACE) {
				it.breaks = 0
			}
		}
	}

	blankLinesAroundDeclarations(items)
}

func next(items []*item, i int) int {
	for j := i + 1; j < len(items); j++ {
		if !items[j].drop {
			return j
		}
	}
	return -1
}

func prev(items []*item, i int) int {
	for j := i - 1; j >= 0; j-- {
		if !items[j].drop {
			return j
		}
	}
	return -1
}

// lambdaArrow returns the «=>» of «{ أ، ب =>», or -1.
func lambdaArrow(items []*item, open int) int {
	j := next(items, open)
	for j >= 0 && items[j].is(token.IDENT) {
		k := next(items, j)
		if k < 0 {
			return -1
		}
		if items[k].is(token.ARROW) {
			return k
		}
		if !items[k].is(token.COMMA) {
			return -1
		}
		j = next(items, k)
	}
	return -1
}

// blankLinesAroundDeclarations puts a blank line before and after every
// top-level declaration that spans lines, as Kotlin does between functions.
// Comments directly above a declaration stay with it.
func blankLinesAroundDeclarations(items []*item) {
	// the items that start a top-level line
	var starts []int
	depth := 0
	for i, it := range items {
		if it.drop {
			continue
		}
		if depth == 0 && (i == 0 || it.breaks > 0) && !it.closes() {
			starts = append(starts, i)
		}
		if it.opens() {
			depth++
		} else if it.closes() && depth > 0 {
			depth--
		}
	}
	type stmt struct{ lead, start, end int }
	var stmts []stmt
	for n, s := range starts {
		if items[s].comment {
			continue
		}
		end := len(items) - 1
		if n+1 < len(starts) {
			end = starts[n+1] - 1
		}
		lead := s
		for m := n - 1; m >= 0 && items[starts[m]].comment && items[lead].breaks == 1; m-- {
			lead = starts[m]
		}
		stmts = append(stmts, stmt{lead, s, end})
	}
	multiDecl := func(st stmt) bool {
		if !declKeywords[items[st.start].tok.Kind] && !isPage(items, st.start, st.end) {
			return false
		}
		for j := st.start + 1; j <= st.end; j++ {
			if items[j].breaks > 0 && !items[j].drop {
				return true
			}
		}
		return false
	}
	for n := 1; n < len(stmts); n++ {
		if multiDecl(stmts[n]) || multiDecl(stmts[n-1]) {
			items[stmts[n].lead].breaks = 2
		}
	}
}

// ---------------------------------------------------------------- printing

func print(items []*item) string {
	var b strings.Builder
	type open struct{ line int }
	var stack []open
	depth := func() int {
		n, last := 0, -1
		for _, o := range stack {
			if o.line != last {
				n++
				last = o.line
			}
		}
		return n
	}
	line, col := 0, 0
	prevIdx := -1
	write := func(s string) {
		b.WriteString(s)
		if k := strings.LastIndexByte(s, '\n'); k >= 0 {
			line += strings.Count(s, "\n")
			col = displayWidth(s[k+1:])
		} else {
			col += displayWidth(s)
		}
	}
	for i, it := range items {
		if it.drop {
			continue
		}
		if i > 0 && prevIdx >= 0 && it.breaks > 0 {
			write(strings.Repeat("\n", it.breaks))
			// Closing brackets at the start of the line belong to the outer level.
			saved := append([]open{}, stack...)
			for j := i; j < len(items); j = next(items, j) {
				if j < 0 || !items[j].closes() || (j > i && items[j].breaks > 0) {
					break
				}
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
			level := depth()
			stack = saved
			// A line that starts with «.» or an operator continues the one before (inside brackets).
			if !it.comment && (it.tok.Kind == token.DOT || it.tok.Kind == token.SAFE_DOT ||
				binary[it.tok.Kind] && it.tok.Kind != token.MINUS || it.tok.Kind == token.AND || it.tok.Kind == token.OR) {
				level++
			}
			write(strings.Repeat(indentUnit, level))
		} else if prevIdx >= 0 {
			write(spaceBetween(items, prevIdx, i))
		}
		it.line, it.col = line, col
		write(it.text)
		if it.opens() {
			stack = append(stack, open{line})
		} else if it.closes() && len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
		prevIdx = i
	}
	return strings.TrimRight(b.String(), "\n ") + "\n"
}

var binary = map[token.Kind]bool{
	token.PLUS: true, token.MINUS: true, token.STAR: true, token.SLASH: true, token.PERCENT: true,
	token.ASSIGN: true, token.PLUS_EQ: true, token.MINUS_EQ: true, token.STAR_EQ: true, token.SLASH_EQ: true,
	token.EQ: true, token.NEQ: true, token.LE: true, token.GE: true, token.ARROW: true,
	token.AND_SYM: true, token.OR_SYM: true, token.COALESCE: true,
}

// unary reports whether the + - ! at items[i] is a prefix operator.
func unary(items []*item, i int) bool {
	p := prev(items, i)
	for p >= 0 && items[p].comment {
		p = prev(items, p)
	}
	if p < 0 || items[i].breaks > 0 {
		return true
	}
	switch items[p].tok.Kind {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.RPAREN, token.RBRACK, token.RBRACE,
		token.TRUE, token.FALSE, token.NULL, token.IT, token.THIS, token.QUESTION:
		return false
	}
	return true
}

// spaceBetween is what goes between two items on the same line.
func spaceBetween(items []*item, ai, bi int) string {
	a, b := items[ai], items[bi]
	if a.comment || b.comment {
		return " "
	}
	ak, bk := a.tok.Kind, b.tok.Kind
	switch {
	case ak == token.LT || ak == token.GT || bk == token.LT || bk == token.GT:
		// «<» also opens type arguments (نتيجة<صحيح>); keep the writer's choice.
		if b.tok.Pos.Line == a.tok.Pos.Line && b.tok.Pos.Col == a.tok.Pos.Col+utf8.RuneCountInString(a.text) {
			return ""
		}
		return " "
	case bk == token.COMMA || bk == token.SEMI:
		return ""
	case ak == token.COMMA || ak == token.SEMI:
		return " "
	case bk == token.RPAREN || bk == token.RBRACK:
		return ""
	case ak == token.LPAREN || ak == token.LBRACK:
		return ""
	case bk == token.RBRACE:
		if ak == token.LBRACE {
			return ""
		}
		return " "
	case ak == token.LBRACE || bk == token.LBRACE:
		return " "
	case ak == token.DOT || bk == token.DOT || ak == token.SAFE_DOT || bk == token.SAFE_DOT ||
		ak == token.RANGE || bk == token.RANGE:
		return ""
	case bk == token.QUESTION:
		return "" // نص؟
	case bk == token.COLON:
		return ""
	case ak == token.COLON:
		return " "
	case (ak == token.MINUS || ak == token.PLUS || ak == token.BANG) && unary(items, ai):
		return ""
	case binary[ak] || binary[bk]:
		return " "
	case bk == token.LPAREN || bk == token.LBRACK:
		switch ak {
		case token.IDENT, token.RPAREN, token.RBRACK, token.QUESTION, token.IT, token.THIS, token.STRING:
			return "" // a call or an index
		}
		return " "
	}
	return " "
}

// displayWidth counts the columns text takes in a monospace font: Arabic
// vowel marks sit on the letter before them and take no column.
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.Is(unicode.Mn, r) {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------- long lines

// wrapLongLine finds the first line longer than maxWidth and breaks it at its
// outermost ( or [ that has commas: one argument per line, Kotlin style.
// It reports whether it changed anything.
func wrapLongLine(items []*item) bool {
	// the width of each output line
	widths := map[int]int{}
	for _, it := range items {
		if it.drop {
			continue
		}
		end := it.col + displayWidth(it.text)
		if k := strings.LastIndexByte(it.text, '\n'); k >= 0 {
			widths[it.line+strings.Count(it.text, "\n")] = displayWidth(it.text[k+1:])
			end = displayWidth(it.text[:strings.IndexByte(it.text, '\n')]) + it.col
		}
		widths[it.line] = max(widths[it.line], end)
	}
	for i, it := range items {
		if it.drop || widths[it.line] <= maxWidth || !it.opens() || it.tok.Kind == token.LBRACE || it.match < 0 {
			continue
		}
		end := it.match
		if items[end].line != it.line {
			continue // already spans lines
		}
		// the commas directly inside this bracket
		var commas []int
		depth := 0
		for j := i + 1; j < end; j++ {
			switch {
			case items[j].opens():
				depth++
			case items[j].closes():
				depth--
			case depth == 0 && items[j].is(token.COMMA):
				commas = append(commas, j)
			}
		}
		if len(commas) == 0 {
			continue
		}
		if n := next(items, i); n >= 0 && n != end {
			items[n].breaks = 1
		}
		for _, c := range commas {
			if n := next(items, c); n >= 0 && n != end {
				items[n].breaks = 1
			}
		}
		items[end].breaks = 1
		return true
	}
	return false
}

// ---------------------------------------------------------------- safety

// check makes sure formatting changed only layout: the same tokens (ignoring
// line breaks and «؛», which can stand for each other), and no syntax errors.
func check(before []token.Token, out string) error {
	after, _ := lexer.Tokenize(out)
	strip := func(ts []token.Token) []token.Token {
		var r []token.Token
		for _, t := range ts {
			if t.Kind != token.NEWLINE && t.Kind != token.SEMI {
				r = append(r, t)
			}
		}
		return r
	}
	x, y := strip(before), strip(after)
	if len(x) != len(y) {
		return errors.New("خطأ داخلي في المنسّق: تغيّر عدد الرموز")
	}
	for i := range x {
		if x[i].Kind != y[i].Kind || x[i].Lit != y[i].Lit {
			return fmt.Errorf("خطأ داخلي في المنسّق عند %s", y[i].Pos)
		}
	}
	if _, diags := parser.ParseFile(out); lexer.HasErrors(diags) {
		return fmt.Errorf("خطأ داخلي في المنسّق: النتيجة لا تُقرأ — %v", diags)
	}
	return nil
}
