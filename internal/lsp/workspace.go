package lsp

// Programs of several files: references and rename across the files of the
// workspace, links from «استورد … من "./ملف"» to the file, and the
// workspace's symbols (types, functions, components …) for «Go to Symbol».

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/module"
	"github.com/mtwalli/dad-lang/internal/token"
)

// maxFiles bounds the files read for references and workspace symbols, so a
// folder such as a home directory opened by mistake stays fast.
const maxFiles = 2000

// setRoots records the workspace folders the editor sent in «initialize».
func (s *Server) setRoots(params json.RawMessage) {
	var p struct {
		RootURI          string `json:"rootUri"`
		WorkspaceFolders []struct {
			URI string `json:"uri"`
		} `json:"workspaceFolders"`
	}
	json.Unmarshal(params, &p)
	for _, f := range p.WorkspaceFolders {
		if path, ok := uriPath(f.URI); ok {
			s.roots = append(s.roots, path)
		}
	}
	if len(s.roots) == 0 {
		if path, ok := uriPath(p.RootURI); ok {
			s.roots = append(s.roots, path)
		}
	}
}

// read returns a file's text: the editor's copy when it is open there.
func (s *Server) read(p string) (string, error) {
	clean := filepath.Clean(p)
	for u, d := range s.docs {
		if up, ok := uriPath(u); ok && filepath.Clean(up) == clean {
			return d.text, nil
		}
	}
	b, err := os.ReadFile(p)
	return strings.ReplaceAll(string(b), "\r\n", "\n"), err
}

// uriFor is the URI the editor uses for a file when it is open, so that edits
// reach its open copy; otherwise a file:// URI.
func (s *Server) uriFor(p string) string {
	clean := filepath.Clean(p)
	for u := range s.docs {
		if up, ok := uriPath(u); ok && filepath.Clean(up) == clean {
			return u
		}
	}
	return pathURI(p)
}

// files lists the .dad files of the workspace and the open ones. Without
// workspace folders, the folders of the open files are searched.
func (s *Server) files() []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = filepath.Clean(p)
		if !seen[p] && len(out) < maxFiles {
			seen[p] = true
			out = append(out, p)
		}
	}
	roots := s.roots
	for u := range s.docs {
		if p, ok := uriPath(u); ok {
			add(p)
			if len(s.roots) == 0 {
				roots = append(roots, filepath.Dir(p))
			}
		}
	}
	for _, root := range roots {
		filepath.WalkDir(root, func(p string, e os.DirEntry, err error) error {
			if err != nil || len(out) >= maxFiles {
				return filepath.SkipDir
			}
			if e.IsDir() {
				name := e.Name()
				if p != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "site" || name == "dist" || name == "build") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(p) == ".dad" {
				add(p)
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

// analyzeFile checks a file of the workspace together with the files it imports.
func (s *Server) analyzeFile(p string) (*check.Info, *document) {
	text, err := s.read(p)
	if err != nil {
		return nil, nil
	}
	info, _ := check.AnalyzeProgram(module.Load(p, s.read))
	return info, newDocument(text)
}

// declaration identifies a declaration across files: the file and the
// position of its name there.
type declaration struct {
	path string
	pos  token.Pos
}

// declOf is where a definition seen in the file at path is declared.
func declOf(path string, d *check.Def) declaration {
	if d.Origin != nil {
		return declaration{filepath.Clean(d.Origin.Path), d.Origin.Pos}
	}
	return declaration{filepath.Clean(path), d.Pos}
}

// occurrences lists where a file names a declaration: its own declaration,
// the name in «استورد», and every use.
func occurrences(info *check.Info, path string, target declaration) []token.Pos {
	seen := map[token.Pos]bool{}
	for _, d := range info.Defs {
		if declOf(path, d) == target {
			seen[d.Pos] = true
		}
	}
	for p, d := range info.Refs {
		if declOf(path, d) == target {
			seen[p] = true
		}
	}
	out := make([]token.Pos, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Line < out[j].Line || out[i].Line == out[j].Line && out[i].Col < out[j].Col
	})
	return out
}

// everywhere finds a declaration in every file of the workspace: for each
// file's URI, the ranges where it is named. The current file is read from
// its open copy; a file that is not saved has no other files.
func (s *Server) everywhere(d *document, uri string, def *check.Def) map[string][]Range {
	out := map[string][]Range{}
	path, ok := uriPath(uri)
	if !ok {
		for _, p := range d.info.Uses(def) {
			out[uri] = append(out[uri], d.rangeOf(p, d.info.NameLen(p)))
		}
		return out
	}
	target := declOf(path, def)
	if def.Kind == "parameter" || (!def.Global && def.Parent == nil && def.Origin == nil) {
		// a local name: only this file can name it
		for _, p := range d.info.Uses(def) {
			out[uri] = append(out[uri], d.rangeOf(p, d.info.NameLen(p)))
		}
		return out
	}
	for _, p := range occurrences(d.info, path, target) {
		out[uri] = append(out[uri], d.rangeOf(p, d.info.NameLen(p)))
	}
	for _, f := range s.files() {
		if f == filepath.Clean(path) {
			continue
		}
		text, err := s.read(f)
		if err != nil || !strings.Contains(text, def.Name) {
			continue // cheap: a file that never writes the name can't name it
		}
		info, doc := s.analyzeFile(f)
		if info == nil {
			continue
		}
		u := s.uriFor(f)
		for _, p := range occurrences(info, f, target) {
			out[u] = append(out[u], doc.rangeOf(p, info.NameLen(p)))
		}
	}
	return out
}

// importPath finds the file an «استورد … من "…"» names, when the token at a
// position is that string.
func importPath(info *check.Info, path string, line, col int) (string, token.Token, bool) {
	for i, t := range info.Toks {
		if t.Kind != token.STRING || t.Pos.Line != line || col < t.Pos.Col || col > t.Pos.Col+len([]rune(t.Lit)) {
			continue
		}
		if i == 0 || info.Toks[i-1].Kind != token.FROM {
			return "", t, false
		}
		for j := i - 2; j >= 0; j-- {
			switch info.Toks[j].Kind {
			case token.IMPORT:
				return module.Resolve(path, t.Value), t, true
			case token.IDENT, token.COMMA, token.NEWLINE:
				continue
			}
			return "", t, false
		}
		return "", t, false
	}
	return "", token.Token{}, false
}

// links makes each imported file's name a link to the file.
func (s *Server) links(params json.RawMessage) any {
	var p docPos
	json.Unmarshal(params, &p)
	out := []any{}
	d := s.docs[p.TextDocument.URI]
	path, ok := uriPath(p.TextDocument.URI)
	if d == nil || d.info == nil || !ok {
		return out
	}
	for i, t := range d.info.Toks {
		if t.Kind != token.STRING || i == 0 || d.info.Toks[i-1].Kind != token.FROM {
			continue
		}
		target, _, ok := importPath(d.info, path, t.Pos.Line, t.Pos.Col)
		if !ok {
			continue
		}
		if _, err := s.read(target); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"range":   d.rangeOf(t.Pos, len([]rune(t.Lit))),
			"target":  s.uriFor(target),
			"tooltip": filepath.Base(target),
		})
	}
	return out
}

// workspaceSymbols lists the top-level declarations of every file, and the
// fields, methods and cases of its types, whose names match the query.
func (s *Server) workspaceSymbols(params json.RawMessage) any {
	var p struct {
		Query string `json:"query"`
	}
	json.Unmarshal(params, &p)
	out := []any{}
	for _, f := range s.files() {
		text, err := s.read(f)
		if err != nil {
			continue
		}
		info, _ := check.Analyze(text)
		if info == nil {
			continue
		}
		doc := newDocument(text)
		u := s.uriFor(f)
		for _, def := range info.Defs {
			if def.Origin != nil || def.Kind == "parameter" || (!def.Global && def.Parent == nil) || !fuzzy(p.Query, def.Name) {
				continue
			}
			sym := map[string]any{
				"name":     def.Name,
				"kind":     symbolKinds[def.Kind],
				"location": Location{URI: u, Range: doc.rangeOf(def.Pos, info.NameLen(def.Pos))},
			}
			if def.Parent != nil {
				sym["containerName"] = def.Parent.Name
			} else {
				sym["containerName"] = strings.TrimSuffix(filepath.Base(f), ".dad")
			}
			out = append(out, sym)
		}
	}
	return out
}

// fuzzy reports whether the letters of query appear in name in order,
// ignoring case and the differences between ا أ إ آ, ه ة and ى ي.
func fuzzy(query, name string) bool {
	q := []rune(fold(query))
	if len(q) == 0 {
		return true
	}
	i := 0
	for _, r := range fold(name) {
		if r == q[i] {
			i++
			if i == len(q) {
				return true
			}
		}
	}
	return false
}

func fold(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 'أ', 'إ', 'آ':
			return 'ا'
		case 'ة':
			return 'ه'
		case 'ى':
			return 'ي'
		case '_', ' ':
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}
