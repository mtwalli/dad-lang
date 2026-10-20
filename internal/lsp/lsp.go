// Package lsp is Ḍād's language server: «dad lsp» speaks the Language Server
// Protocol over standard input and output, so any editor that supports LSP
// gets errors, go to definition, find references, rename, hover, completion,
// an outline and formatting for .dad files.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/format"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/token"
)

// ---------------------------------------------------------------- protocol types

type message struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Position and Range are the protocol's: lines from 0, columns in UTF-16 units.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type docPos struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position Position `json:"position"`
}

// ---------------------------------------------------------------- documents

type document struct {
	text  string
	lines []string
	info  *check.Info // the last version that parsed
}

func newDocument(text string) *document {
	d := &document{text: text, lines: strings.Split(text, "\n")}
	return d
}

// toRune converts a protocol position to a Ḍād one (line and column in letters, from 1).
func (d *document) toRune(p Position) (int, int) {
	if p.Line < 0 || p.Line >= len(d.lines) {
		return p.Line + 1, 1
	}
	units := 0
	col := 1
	for _, r := range d.lines[p.Line] {
		if units >= p.Character {
			break
		}
		units += len(utf16.Encode([]rune{r}))
		col++
	}
	return p.Line + 1, col
}

// fromRune converts a Ḍād position to a protocol one.
func (d *document) fromRune(line, col int) Position {
	if line < 1 || line > len(d.lines) {
		return Position{Line: max(line-1, 0)}
	}
	units := 0
	for i, r := range []rune(d.lines[line-1]) {
		if i >= col-1 {
			break
		}
		units += len(utf16.Encode([]rune{r}))
	}
	return Position{Line: line - 1, Character: units}
}

func (d *document) rangeOf(p token.Pos, length int) Range {
	return Range{Start: d.fromRune(p.Line, p.Col), End: d.fromRune(p.Line, p.Col+max(length, 1))}
}

// ---------------------------------------------------------------- server

// Server holds the open documents of one editor session.
type Server struct {
	mu       sync.Mutex
	out      io.Writer
	outMu    sync.Mutex
	docs     map[string]*document
	keywords bool // offer keywords and built-ins in completion (VS Code's extension does its own)
	shutdown bool
}

// Serve runs the language server until the editor says exit or input ends.
func Serve(in io.Reader, out io.Writer) error {
	s := &Server{out: out, docs: map[string]*document{}, keywords: true}
	r := bufio.NewReader(in)
	for {
		msg, err := read(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if msg.Method == "exit" {
			return nil
		}
		s.handle(msg)
	}
}

func read(r *bufio.Reader) (*message, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "Content-Length") {
			length, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("lsp: message without Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	var m message
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Server) send(m message) {
	m.JSONRPC = "2.0"
	body, _ := json.Marshal(m)
	s.outMu.Lock()
	defer s.outMu.Unlock()
	fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n%s", len(body), body)
}

func (s *Server) reply(id *json.RawMessage, result any) {
	if result == nil {
		result = json.RawMessage("null")
	}
	s.send(message{ID: id, Result: result})
}

func (s *Server) notify(method string, params any) {
	raw, _ := json.Marshal(params)
	s.send(message{Method: method, Params: raw})
}

func (s *Server) handle(m *message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if r := recover(); r != nil && m.ID != nil {
			s.send(message{ID: m.ID, Error: &rpcError{Code: -32603, Message: fmt.Sprint(r)}})
		}
	}()
	switch m.Method {
	case "initialize":
		var p struct {
			InitializationOptions struct {
				Keywords *bool `json:"keywords"`
			} `json:"initializationOptions"`
		}
		json.Unmarshal(m.Params, &p)
		if p.InitializationOptions.Keywords != nil {
			s.keywords = *p.InitializationOptions.Keywords
		}
		s.reply(m.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync":           map[string]any{"openClose": true, "change": 1},
				"hoverProvider":              true,
				"definitionProvider":         true,
				"referencesProvider":         true,
				"documentHighlightProvider":  true,
				"renameProvider":             map[string]any{"prepareProvider": true},
				"completionProvider":         map[string]any{"triggerCharacters": []string{".", "؟"}},
				"documentSymbolProvider":     true,
				"documentFormattingProvider": true,
			},
			"serverInfo": map[string]any{"name": "dad lsp"},
		})
	case "shutdown":
		s.shutdown = true
		s.reply(m.ID, nil)
	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		json.Unmarshal(m.Params, &p)
		s.update(p.TextDocument.URI, p.TextDocument.Text)
	case "textDocument/didChange":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		json.Unmarshal(m.Params, &p)
		if n := len(p.ContentChanges); n > 0 {
			s.update(p.TextDocument.URI, p.ContentChanges[n-1].Text)
		}
	case "textDocument/didClose":
		var p docPos
		json.Unmarshal(m.Params, &p)
		delete(s.docs, p.TextDocument.URI)
		s.notify("textDocument/publishDiagnostics", map[string]any{"uri": p.TextDocument.URI, "diagnostics": []any{}})
	case "textDocument/hover":
		s.reply(m.ID, s.hover(m.Params))
	case "textDocument/definition":
		s.reply(m.ID, s.definition(m.Params))
	case "textDocument/references":
		s.reply(m.ID, s.references(m.Params))
	case "textDocument/documentHighlight":
		s.reply(m.ID, s.highlights(m.Params))
	case "textDocument/prepareRename":
		s.reply(m.ID, s.prepareRename(m.Params))
	case "textDocument/rename":
		result, err := s.rename(m.Params)
		if err != "" {
			s.send(message{ID: m.ID, Error: &rpcError{Code: -32602, Message: err}})
			return
		}
		s.reply(m.ID, result)
	case "textDocument/completion":
		s.reply(m.ID, s.completion(m.Params))
	case "textDocument/documentSymbol":
		s.reply(m.ID, s.symbols(m.Params))
	case "textDocument/formatting":
		s.reply(m.ID, s.formatting(m.Params))
	default:
		if m.ID != nil && !strings.HasPrefix(m.Method, "$/") {
			s.send(message{ID: m.ID, Error: &rpcError{Code: -32601, Message: "method not found: " + m.Method}})
		}
	}
}

// analyze checks a document. A saved file is checked with the files it
// imports, read from the editor when they are open there, or from the disk.
func (s *Server) analyze(uri, text string) (*check.Info, []lexer.Diagnostic) {
	path, ok := uriPath(uri)
	if !ok {
		return check.Analyze(text)
	}
	read := func(p string) (string, error) {
		if filepath.Clean(p) == filepath.Clean(path) {
			return text, nil
		}
		for u, d := range s.docs {
			if up, ok := uriPath(u); ok && filepath.Clean(up) == filepath.Clean(p) {
				return d.text, nil
			}
		}
		b, err := os.ReadFile(p)
		return string(b), err
	}
	return check.AnalyzeProgram(module.Load(path, read))
}

// pathURI turns a path into a file:// URI.
func pathURI(p string) string {
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}
	p = filepath.ToSlash(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // C:/… on Windows
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// uriPath turns a file:// URI into a path.
func uriPath(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	p := u.Path
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:] // file:///C:/… on Windows
	}
	return filepath.FromSlash(p), true
}

// update analyses a new version of a document and publishes its problems.
func (s *Server) update(uri, text string) {
	d := newDocument(strings.ReplaceAll(text, "\r\n", "\n"))
	info, diags := s.analyze(uri, d.text)
	if info != nil {
		d.info = info
	} else if old := s.docs[uri]; old != nil {
		d.info = old.info // keep answering from the last version that parsed
	}
	s.docs[uri] = d
	toks, _ := lexer.Tokenize(d.text)
	lengths := map[token.Pos]int{}
	for _, t := range toks {
		lengths[t.Pos] = len([]rune(t.Lit))
	}
	out := []any{}
	for _, dg := range diags {
		sev := 1
		if dg.Warning {
			sev = 2
		}
		out = append(out, map[string]any{
			"range":    d.rangeOf(dg.Pos, lengths[dg.Pos]),
			"severity": sev,
			"source":   "ضاد",
			"message":  dg.Msg,
		})
	}
	s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": out})
}

// at finds the document and the definition under the cursor.
func (s *Server) at(params json.RawMessage) (*document, string, *check.Def, int, int) {
	var p docPos
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	if d == nil || d.info == nil {
		return d, p.TextDocument.URI, nil, 0, 0
	}
	line, col := d.toRune(p.Position)
	def, ok := d.info.DefAt(line, col)
	if !ok && col > 1 {
		def, ok = d.info.DefAt(line, col-1) // the cursor just after a name
	}
	if !ok {
		return d, p.TextDocument.URI, nil, line, col
	}
	return d, p.TextDocument.URI, def, line, col
}

var kindNames = map[string]string{
	"function": "دالة", "method": "دالة", "type": "نوع", "table": "جدول", "enum": "تعداد", "case": "حالة من تعداد",
	"component": "مكون", "field": "حقل", "parameter": "معامل", "constant": "ثابت", "variable": "متغير", "state": "حالة",
}

func (s *Server) hover(params json.RawMessage) any {
	d, _, def, _, _ := s.at(params)
	if def == nil {
		return nil
	}
	text := "```dad\n" + def.Detail + "\n```"
	what := kindNames[def.Kind]
	if def.Parent != nil {
		what += " في «" + def.Parent.Name + "»"
	}
	text += "\n\n" + what
	return map[string]any{
		"contents": map[string]any{"kind": "markdown", "value": text},
		"range":    d.rangeOf(def.Pos, 0),
	}
}

func (s *Server) definition(params json.RawMessage) any {
	d, uri, def, _, _ := s.at(params)
	if def == nil {
		return nil
	}
	if o := def.Origin; o != nil {
		// an imported name: go to the file that declares it
		return Location{URI: pathURI(o.Path), Range: newDocument(strings.ReplaceAll(o.Src, "\r\n", "\n")).rangeOf(o.Pos, o.Len)}
	}
	return Location{URI: uri, Range: d.rangeOf(def.Pos, d.info.NameLen(def.Pos))}
}

func (s *Server) references(params json.RawMessage) any {
	d, uri, def, _, _ := s.at(params)
	if def == nil {
		return []Location{}
	}
	out := []Location{}
	for _, p := range d.info.Uses(def) {
		out = append(out, Location{URI: uri, Range: d.rangeOf(p, d.info.NameLen(p))})
	}
	return out
}

func (s *Server) highlights(params json.RawMessage) any {
	d, _, def, _, _ := s.at(params)
	if def == nil {
		return []any{}
	}
	out := []any{}
	for i, p := range d.info.Uses(def) {
		kind := 2 // read
		if i == 0 {
			kind = 3 // the declaration
		}
		out = append(out, map[string]any{"range": d.rangeOf(p, d.info.NameLen(p)), "kind": kind})
	}
	return out
}

func (s *Server) prepareRename(params json.RawMessage) any {
	d, _, def, line, col := s.at(params)
	if def == nil {
		return nil
	}
	t, ok := d.info.TokenAt(line, col)
	if !ok {
		t, _ = d.info.TokenAt(line, col-1)
	}
	return map[string]any{"range": d.rangeOf(t.Pos, len([]rune(t.Lit))), "placeholder": def.Name}
}

func (s *Server) rename(params json.RawMessage) (any, string) {
	var p struct {
		NewName string `json:"newName"`
	}
	json.Unmarshal(params, &p)
	d, uri, def, _, _ := s.at(params)
	if def == nil {
		return nil, "لا يوجد اسم هنا يمكن تغييره"
	}
	toks, diags := lexer.Tokenize(p.NewName)
	var words []token.Token
	for _, t := range toks {
		if t.Kind != token.NEWLINE && t.Kind != token.EOF {
			words = append(words, t)
		}
	}
	if len(diags) > 0 || len(words) != 1 || words[0].Kind != token.IDENT {
		return nil, "«" + p.NewName + "» ليس اسماً صالحاً في ضاد"
	}
	var edits []TextEdit
	for _, pos := range d.info.Uses(def) {
		edits = append(edits, TextEdit{Range: d.rangeOf(pos, d.info.NameLen(pos)), NewText: p.NewName})
	}
	return map[string]any{"changes": map[string][]TextEdit{uri: edits}}, ""
}

// ---------------------------------------------------------------- completion

var memberChain = regexp.MustCompile(`([\p{L}\p{Mn}_0-9]+(?:(?:[.]|[؟?][.])[\p{L}\p{Mn}_0-9]+)*)(?:[.]|[؟?][.])([\p{L}\p{Mn}_0-9]*)$`)

var completionKinds = map[string]int{
	"function": 3, "method": 2, "type": 22, "table": 22, "enum": 13, "case": 20, "component": 7,
	"field": 5, "parameter": 6, "constant": 21, "variable": 6, "state": 6, "property": 10,
}

func (s *Server) completion(params json.RawMessage) any {
	var p docPos
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	items := []map[string]any{}
	if d == nil {
		return items
	}
	line, col := d.toRune(p.Position)
	prefix := ""
	if line-1 < len(d.lines) {
		runes := []rune(d.lines[line-1])
		prefix = string(runes[:min(col-1, len(runes))])
	}
	if m := memberChain.FindStringSubmatch(prefix); m != nil {
		// «ن.» is not finished code: analyse the file without the dot and what follows it.
		info := d.info
		if lines := append([]string{}, d.lines...); line-1 < len(lines) {
			runes := []rune(lines[line-1])
			cut := col - 1 - len([]rune(m[0])) + len([]rune(m[1]))
			lines[line-1] = string(runes[:cut]) + string(runes[min(col-1, len(runes)):])
			if fixed, _ := s.analyze(p.TextDocument.URI, strings.Join(lines, "\n")); fixed != nil {
				info = fixed
			}
		}
		if info == nil {
			return items
		}
		chain := regexp.MustCompile(`[.]|[؟?][.]`).Split(m[1], -1)
		for _, mem := range info.Members(info.TypeOf(chain, line, col)) {
			items = append(items, map[string]any{"label": mem.Name, "kind": completionKinds[mem.Kind], "detail": mem.Detail})
		}
		return items
	}
	if d.info != nil {
		for _, def := range d.info.Visible(line, col) {
			items = append(items, map[string]any{"label": def.Name, "kind": completionKinds[def.Kind], "detail": def.Detail})
		}
	}
	if s.keywords {
		for kw := range token.Keywords {
			items = append(items, map[string]any{"label": kw, "kind": 14})
		}
		for _, b := range interp.BuiltinNames() {
			items = append(items, map[string]any{"label": b, "kind": 3})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i]["label"].(string) < items[j]["label"].(string) })
	return items
}

// ---------------------------------------------------------------- outline and formatting

var symbolKinds = map[string]int{
	"function": 12, "method": 6, "type": 23, "table": 23, "enum": 10, "case": 22, "component": 5,
	"field": 8, "constant": 14, "variable": 13, "state": 13,
}

func (s *Server) symbols(params json.RawMessage) any {
	var p docPos
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	out := []any{}
	if d == nil || d.info == nil {
		return out
	}
	var symbol func(def *check.Def) map[string]any
	symbol = func(def *check.Def) map[string]any {
		r := d.rangeOf(def.Pos, d.info.NameLen(def.Pos))
		sym := map[string]any{"name": def.Name, "detail": def.Detail, "kind": symbolKinds[def.Kind], "range": r, "selectionRange": r}
		var kids []any
		for _, c := range def.Children {
			kids = append(kids, symbol(c))
		}
		if kids != nil {
			sym["children"] = kids
		}
		return sym
	}
	for _, def := range d.info.Defs {
		if def.Global && def.Parent == nil && def.Kind != "parameter" {
			out = append(out, symbol(def))
		}
	}
	return out
}

func (s *Server) formatting(params json.RawMessage) any {
	var p docPos
	json.Unmarshal(params, &p)
	d := s.docs[p.TextDocument.URI]
	if d == nil {
		return []TextEdit{}
	}
	out, err := format.Source(d.text)
	if err != nil || out == d.text {
		return []TextEdit{}
	}
	end := d.fromRune(len(d.lines), len([]rune(d.lines[len(d.lines)-1]))+1)
	return []TextEdit{{Range: Range{End: end}, NewText: out}}
}
