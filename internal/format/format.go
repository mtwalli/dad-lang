// Package format lays out Ḍād (ضاد) source code the standard way, like gofmt:
//
//   - two spaces of indentation per open bracket, decided by the code, not by
//     the spaces that were there;
//   - one space around operators and after commas, none inside ( ) and [ ],
//     one inside { } on a single line;
//   - comments are kept where they are; multi-line strings are left untouched;
//   - trailing spaces go, runs of blank lines become one, and the file ends with
//     a single newline.
//
// Only spaces change: the formatter checks that the tokens of the result are
// exactly the tokens of the input, and returns an error otherwise.
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

const indentUnit = "  "

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
	f, err := newFile(src, toks)
	if err != nil {
		return src, err
	}
	out := f.format()
	if err := sameTokens(toks, out); err != nil {
		return src, err
	}
	return out, nil
}

// tok is a token with its byte offsets in the source.
type tok struct {
	token.Token
	start, end int
	line       int // 0-based line of its first character
	endLine    int // 0-based line of its last character
}

type file struct {
	src       string
	lines     []string
	lineStart []int // byte offset of each line
	toks      []tok
	byLine    [][]int // indexes into toks of the tokens that start on each line
	verbatim  []bool  // lines kept exactly as written (inside multi-line strings or comments)
	inComment []bool  // lines that start inside a block comment
}

func newFile(src string, raw []token.Token) (*file, error) {
	f := &file{src: src, lines: strings.Split(src, "\n")}
	off := 0
	for _, l := range f.lines {
		f.lineStart = append(f.lineStart, off)
		off += len(l) + 1
	}
	f.byLine = make([][]int, len(f.lines))
	f.verbatim = make([]bool, len(f.lines))
	f.inComment = make([]bool, len(f.lines))
	for _, t := range raw {
		if t.Kind == token.NEWLINE || t.Kind == token.EOF {
			continue
		}
		line := t.Pos.Line - 1
		if line < 0 || line >= len(f.lines) {
			return nil, fmt.Errorf("خطأ داخلي في المنسّق: موضع غير متوقع %s", t.Pos)
		}
		start := f.lineStart[line] + runeOffset(f.lines[line], t.Pos.Col-1)
		if !strings.HasPrefix(src[start:], t.Lit) {
			return nil, fmt.Errorf("خطأ داخلي في المنسّق عند %s", t.Pos)
		}
		end := start + len(t.Lit)
		tk := tok{Token: t, start: start, end: end, line: line, endLine: line + strings.Count(t.Lit, "\n")}
		f.byLine[line] = append(f.byLine[line], len(f.toks))
		f.toks = append(f.toks, tk)
		for l := tk.line + 1; l <= tk.endLine; l++ {
			f.verbatim[l] = true // continuation of a multi-line string
		}
	}
	f.markComments()
	return f, nil
}

func runeOffset(s string, n int) int {
	off := 0
	for i := 0; i < n && off < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[off:])
		off += size
	}
	return off
}

// markComments finds block comments that span lines, by scanning the text that
// isn't part of a token.
func (f *file) markComments() {
	inTok := func(off int) (int, bool) {
		// tokens are sorted; a linear walk is fine for source files
		for _, t := range f.toks {
			if off >= t.start && off < t.end {
				return t.end, true
			}
			if t.start > off {
				break
			}
		}
		return 0, false
	}
	line := 0
	for i := 0; i < len(f.src); {
		if end, ok := inTok(i); ok {
			for _, c := range f.src[i:end] {
				if c == '\n' {
					line++
				}
			}
			i = end
			continue
		}
		switch {
		case strings.HasPrefix(f.src[i:], "//"):
			j := strings.IndexByte(f.src[i:], '\n')
			if j < 0 {
				return
			}
			i += j
		case strings.HasPrefix(f.src[i:], "/*"):
			j := strings.Index(f.src[i+2:], "*/")
			end := len(f.src)
			if j >= 0 {
				end = i + 2 + j + 2
			}
			for _, c := range f.src[i:end] {
				if c == '\n' {
					line++
					f.inComment[line] = true
				}
			}
			i = end
		default:
			if f.src[i] == '\n' {
				line++
			}
			i++
		}
	}
}

// ---------------------------------------------------------------- layout

func isOpen(k token.Kind) bool  { return k == token.LPAREN || k == token.LBRACK || k == token.LBRACE }
func isClose(k token.Kind) bool { return k == token.RPAREN || k == token.RBRACK || k == token.RBRACE }

var binary = map[token.Kind]bool{
	token.PLUS: true, token.MINUS: true, token.STAR: true, token.SLASH: true, token.PERCENT: true,
	token.ASSIGN: true, token.PLUS_EQ: true, token.MINUS_EQ: true, token.STAR_EQ: true, token.SLASH_EQ: true,
	token.EQ: true, token.NEQ: true, token.LE: true, token.GE: true, token.ARROW: true,
	token.AND_SYM: true, token.OR_SYM: true, token.COALESCE: true,
}

// continues reports whether a line ending with k continues on the next line.
func continues(k token.Kind) bool {
	return binary[k] || k == token.AND || k == token.OR || k == token.DOT
}

// entry is one output line: code and its trailing comment, kept apart so
// comments on neighbouring lines can be lined up.
type entry struct {
	code, comment string
}

func (f *file) format() string {
	var out []entry
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
	prevCodeEnd := token.ILLEGAL // last token kind of the previous line with code
	blank := 0

	for i, line := range f.lines {
		idx := f.byLine[i]
		if f.verbatim[i] || f.inComment[i] {
			out = append(out, entry{code: line})
			blank = 0
			// tokens on the closing line of a multi-line string still count for brackets
			for _, ti := range idx {
				stack = f.track(stack, ti, i)
			}
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blank++
			if blank == 1 && len(out) > 0 {
				out = append(out, entry{})
			}
			continue
		}
		blank = 0
		if len(idx) == 0 { // a comment on its own line
			out = append(out, entry{code: strings.Repeat(indentUnit, depth()) + trimmed})
			continue
		}

		// Closing brackets at the start of the line belong to the outer level.
		saved := append([]open{}, stack...)
		for _, ti := range idx {
			if !isClose(f.toks[ti].Kind) {
				break
			}
			stack = f.track(stack, ti, i)
		}
		level := depth()
		first := f.toks[idx[0]].Kind
		if first == token.DOT || first == token.SAFE_DOT || binary[first] && first != token.MINUS ||
			first == token.AND || first == token.OR || (continues(prevCodeEnd) && !isClose(first)) {
			level++
		}
		stack = saved
		for _, ti := range idx {
			stack = f.track(stack, ti, i)
		}

		code, comment := f.lineText(i, idx)
		out = append(out, entry{code: strings.Repeat(indentUnit, level) + code, comment: comment})
		last := f.toks[idx[len(idx)-1]]
		prevCodeEnd = last.Kind
		if last.endLine != last.line { // ends in a multi-line string
			prevCodeEnd = token.ILLEGAL
		}
	}
	for len(out) > 0 && out[len(out)-1] == (entry{}) {
		out = out[:len(out)-1]
	}
	alignComments(out)
	lines := make([]string, len(out))
	for i, e := range out {
		lines[i] = e.code
		if e.comment != "" {
			lines[i] += " " + e.comment
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// alignComments lines up trailing comments on consecutive lines, as gofmt does.
func alignComments(out []entry) {
	for i := 0; i < len(out); {
		if out[i].comment == "" {
			i++
			continue
		}
		j, width := i, 0
		for ; j < len(out) && out[j].comment != ""; j++ {
			width = max(width, displayWidth(out[j].code))
		}
		if j-i > 1 {
			for k := i; k < j; k++ {
				out[k].code += strings.Repeat(" ", width-displayWidth(out[k].code))
			}
		}
		i = j
	}
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

// open is a bracket that hasn't been closed yet, with the line it was opened on.
type open struct{ line int }

func (f *file) track(stack []open, ti, line int) []open {
	k := f.toks[ti].Kind
	switch {
	case isOpen(k):
		return append(stack, open{line})
	case isClose(k) && len(stack) > 0:
		return stack[:len(stack)-1]
	}
	return stack
}

// lineText lays out the code of one line and returns its trailing comment.
func (f *file) lineText(line int, idx []int) (string, string) {
	if lead := f.src[f.lineStart[line]:f.toks[idx[0]].start]; strings.TrimSpace(lead) != "" {
		return strings.TrimSpace(f.lines[line]), "" // starts with a /* comment */: keep it as written
	}
	var b strings.Builder
	for n, ti := range idx {
		t := f.toks[ti]
		if n > 0 {
			p := f.toks[idx[n-1]]
			gap := f.src[p.end:t.start]
			if strings.TrimSpace(gap) != "" {
				// A /* comment */ between tokens: keep the rest of the line as written.
				b.WriteString(f.src[p.end:f.lineEnd(line, idx)])
				return b.String(), f.trailing(line, idx)
			}
			b.WriteString(f.space(ti, gap))
		}
		if t.endLine != line {
			// A multi-line string: its first line here, the rest is kept as written.
			b.WriteString(t.Lit[:strings.IndexByte(t.Lit, '\n')])
			break
		}
		b.WriteString(t.Lit)
	}
	return b.String(), f.trailing(line, idx)
}

func (f *file) lineEnd(line int, idx []int) int {
	return f.toks[idx[len(idx)-1]].end
}

func (f *file) trailing(line int, idx []int) string {
	last := f.toks[idx[len(idx)-1]]
	if last.endLine != line {
		return "" // the line ends inside a multi-line string, which is kept as written
	}
	rest := strings.TrimSpace(f.src[last.end : f.lineStart[line]+len(f.lines[line])])
	return rest
}

// unary reports whether the + - ! at toks[i] is a prefix operator.
func (f *file) unary(i int) bool {
	if i == 0 {
		return true
	}
	p := f.toks[i-1]
	if p.endLine < f.toks[i].line && !continues(p.Kind) {
		return true // first on its line, after a finished statement
	}
	switch p.Kind {
	case token.IDENT, token.INT, token.FLOAT, token.STRING, token.RPAREN, token.RBRACK, token.RBRACE,
		token.TRUE, token.FALSE, token.NULL, token.IT, token.THIS, token.QUESTION:
		return false
	}
	return true
}

// space is what goes between toks[i-1] and toks[i]; gap is what was there.
func (f *file) space(i int, gap string) string {
	a, b := f.toks[i-1], f.toks[i]
	orig := ""
	if gap != "" {
		orig = " "
	}
	switch {
	case a.Kind == token.LT || a.Kind == token.GT || b.Kind == token.LT || b.Kind == token.GT:
		return orig // «<» also opens type arguments (نتيجة<صحيح>); keep the writer's choice
	case b.Kind == token.COMMA || b.Kind == token.SEMI:
		return ""
	case a.Kind == token.COMMA || a.Kind == token.SEMI:
		return " "
	case b.Kind == token.RPAREN || b.Kind == token.RBRACK:
		return ""
	case a.Kind == token.LPAREN || a.Kind == token.LBRACK:
		return ""
	case b.Kind == token.RBRACE:
		if a.Kind == token.LBRACE {
			return ""
		}
		return " "
	case a.Kind == token.LBRACE || b.Kind == token.LBRACE:
		return " "
	case a.Kind == token.DOT || b.Kind == token.DOT || a.Kind == token.SAFE_DOT || b.Kind == token.SAFE_DOT ||
		a.Kind == token.RANGE || b.Kind == token.RANGE:
		return ""
	case b.Kind == token.QUESTION:
		return "" // نص؟
	case b.Kind == token.COLON:
		return ""
	case a.Kind == token.COLON:
		return " "
	case (a.Kind == token.MINUS || a.Kind == token.PLUS || a.Kind == token.BANG) && f.unary(i-1):
		return ""
	case binary[a.Kind] || binary[b.Kind]:
		return " "
	case b.Kind == token.LPAREN || b.Kind == token.LBRACK:
		switch a.Kind {
		case token.IDENT, token.RPAREN, token.RBRACK, token.QUESTION, token.IT, token.THIS, token.STRING:
			return "" // a call or an index
		}
		return " "
	}
	return " "
}

// sameTokens checks that formatting changed nothing but spaces.
func sameTokens(before []token.Token, out string) error {
	after, _ := lexer.Tokenize(out)
	strip := func(ts []token.Token) []token.Token {
		var r []token.Token
		for _, t := range ts {
			if t.Kind != token.NEWLINE {
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
	return nil
}
