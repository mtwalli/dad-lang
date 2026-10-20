// Package module loads a Ḍād (ضاد) program made of several files.
//
// One file is one module. «استورد نقطة، شكل من "./أشكال"» imports names from
// أشكال.dad, found next to the importing file. Everything a file declares at
// the top is exported unless it is marked «خاص». A program is its main file
// and every file it imports, directly or not; imports may not go round in a
// circle.
package module

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/parser"
)

// Module is one file of a program.
type Module struct {
	Path  string // as the program refers to it: the main file's path, joined with imports
	Src   string
	File  *ast.File // nil when the file doesn't parse
	Diags []lexer.Diagnostic
	// Deps are the modules each «استورد» loads; an import whose file is
	// missing or makes a circle is not in it (its problem is in Diags).
	Deps map[*ast.Import]*Module
}

// Name is the module's file name without .dad, for messages: «أشكال».
func (m *Module) Name() string {
	return strings.TrimSuffix(filepath.Base(m.Path), ".dad")
}

// Program is a main file with everything it imports.
type Program struct {
	Main *Module
	// Modules lists every module once, each after the modules it imports;
	// Main is last.
	Modules []*Module
}

// Diag is a problem in one file of a program.
type Diag struct {
	Path string
	lexer.Diagnostic
}

func (d Diag) String() string {
	if d.Pos.Line == 0 {
		return d.Path + ": " + d.Msg // the file itself, such as one that can't be read
	}
	return d.Path + ": " + d.Diagnostic.String()
}

// HasErrors reports whether any diagnostic is an error rather than a warning.
func HasErrors(ds []Diag) bool {
	for _, d := range ds {
		if !d.Warning {
			return true
		}
	}
	return false
}

// Diags lists the problems of every module, file by file in program order.
func (p *Program) Diags() []Diag {
	var out []Diag
	for _, m := range p.Modules {
		ds := append([]lexer.Diagnostic(nil), m.Diags...)
		sort.SliceStable(ds, func(i, j int) bool {
			a, b := ds[i].Pos, ds[j].Pos
			return a.Line < b.Line || a.Line == b.Line && a.Col < b.Col
		})
		for _, d := range ds {
			out = append(out, Diag{Path: m.Path, Diagnostic: d})
		}
	}
	return out
}

// Paths lists the files of the program.
func (p *Program) Paths() []string {
	var out []string
	for _, m := range p.Modules {
		out = append(out, m.Path)
	}
	return out
}

// Reader returns the contents of a file.
type Reader func(path string) (string, error)

// Resolve finds the file an import names, relative to the importing file.
// «"./أشكال"» and «"أشكال"» both mean أشكال.dad next to it.
func Resolve(from, name string) string {
	name = filepath.FromSlash(name)
	if filepath.Ext(name) != ".dad" {
		name += ".dad"
	}
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	return filepath.Join(filepath.Dir(from), name)
}

// Load reads the program whose main file is mainPath.
func Load(mainPath string, read Reader) *Program {
	l := &loader{read: read, byPath: map[string]*Module{}, state: map[*Module]int{}}
	main := l.module(filepath.Clean(mainPath))
	p := &Program{Main: main}
	l.visit(main, nil, p)
	return p
}

// Single makes a program of one source text that imports nothing, for the
// playground and for code with no file.
func Single(name, src string) *Program {
	m := &Module{Path: name, Src: src, Deps: map[*ast.Import]*Module{}}
	m.File, m.Diags = parser.ParseFile(src)
	if m.File != nil {
		for _, s := range m.File.Stmts {
			if im, ok := s.(*ast.Import); ok {
				m.Diags = append(m.Diags, lexer.Diagnostic{Pos: im.Pos,
					Msg: "الاستيراد يحتاج ملفاً محفوظاً؛ هنا ضع الكود في ملف واحد"})
			}
		}
	}
	return &Program{Main: m, Modules: []*Module{m}}
}

type loader struct {
	read   Reader
	byPath map[string]*Module
	state  map[*Module]int // 1: being visited, 2: done
}

// module reads and parses a file once.
func (l *loader) module(p string) *Module {
	key := path.Clean(filepath.ToSlash(p))
	if m, ok := l.byPath[key]; ok {
		return m
	}
	m := &Module{Path: p, Deps: map[*ast.Import]*Module{}}
	l.byPath[key] = m
	src, err := l.read(p)
	if err != nil {
		m.Diags = append(m.Diags, lexer.Diagnostic{Msg: fmt.Sprintf("تعذرت قراءة الملف: %v", err)})
		return m
	}
	m.Src = src
	m.File, m.Diags = parser.ParseFile(src)
	return m
}

// visit loads a module's imports, depth first, and lists it after them.
func (l *loader) visit(m *Module, stack []*Module, p *Program) {
	l.state[m] = 1
	stack = append(stack, m)
	if m.File != nil {
		for _, s := range m.File.Stmts {
			im, ok := s.(*ast.Import)
			if !ok {
				continue
			}
			target := Resolve(m.Path, im.From)
			if _, err := l.read(target); err != nil {
				m.Diags = append(m.Diags, lexer.Diagnostic{Pos: im.Pos,
					Msg: fmt.Sprintf("لا يوجد ملف «%s» (بحثت عن «%s»)", im.From, filepath.ToSlash(target))})
				continue
			}
			dep := l.module(target)
			switch l.state[dep] {
			case 1:
				// the circle: dep imports … which imports m, which imports dep
				var circle []*Module
				for i := range stack {
					if stack[i] == dep {
						circle = append(append(circle, stack[i:]...), dep)
						break
					}
				}
				var steps []string
				for i := 0; i+1 < len(circle); i++ {
					steps = append(steps, "«"+circle[i].Name()+"» يستورد «"+circle[i+1].Name()+"»")
				}
				m.Diags = append(m.Diags, lexer.Diagnostic{Pos: im.Pos,
					Msg: "استيراد دائري: " + strings.Join(steps, "، و") + "؛ انقل ما تحتاجه الملفات معاً إلى ملف آخر"})
				continue
			case 0:
				l.visit(dep, stack, p)
			}
			m.Deps[im] = dep
		}
	}
	l.state[m] = 2
	p.Modules = append(p.Modules, m)
}
