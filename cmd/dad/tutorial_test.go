package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/format"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/module"
)

// The tutorial's examples must work, and print what the pages say they print.
//
// Every ```dad block is checked. A block followed by an output block
// (``` { .text .dad-out }) is run, and its output must match. A block whose first
// line is «// خطأ مقصود» must have mistakes, and its output block shows them.
// Blocks with title="name.dad" are files of one program, written together.
//
//	go test ./cmd/dad -run Tutorial -update   rewrites the output blocks

var update = flag.Bool("update", false, "rewrite the output blocks of the tutorial")

var fence = regexp.MustCompile("^( *)```(.*)$")

type block struct {
	info   string
	indent string
	start  int // index of the first line inside the block
	end    int // index of the closing fence
	text   string
}

func blocks(lines []string) []block {
	var out []block
	for i := 0; i < len(lines); i++ {
		m := fence.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		b := block{info: strings.TrimSpace(m[2]), indent: m[1], start: i + 1}
		var body []string
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimRight(lines[j], " ") == m[1]+"```" {
				b.end = j
				break
			}
			body = append(body, strings.TrimPrefix(lines[j], m[1]))
		}
		if b.end == 0 {
			break
		}
		b.text = strings.Join(body, "\n") + "\n"
		out = append(out, b)
		i = b.end
	}
	return out
}

var titleRe = regexp.MustCompile(`title="([^"]+)"`)

func TestTutorial(t *testing.T) {
	pages, _ := filepath.Glob("../../docs/tutorial/*.md")
	if len(pages) < 11 {
		t.Fatalf("found %d pages", len(pages))
	}
	for _, page := range pages {
		t.Run(filepath.Base(page), func(t *testing.T) { tutorialPage(t, page) })
	}
}

func tutorialPage(t *testing.T, page string) {
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	bs := blocks(lines)
	dir := t.TempDir()
	// the files of the page's program
	for _, b := range bs {
		if m := titleRe.FindStringSubmatch(b.info); m != nil && strings.HasPrefix(b.info, "dad") {
			os.WriteFile(filepath.Join(dir, m[1]), []byte(b.text), 0o644)
		}
	}
	for _, b := range bs {
		if strings.Contains(b.info, "dad-out") && !*update && (strings.Contains(b.text, "@run") || strings.Contains(b.text, "@check")) {
			t.Errorf("line %d: an output block was not filled in: run with -update", b.start)
		}
	}
	replace := map[int]string{} // output block index → new text
	for i, b := range bs {
		if !strings.HasPrefix(b.info, "dad") {
			continue
		}
		path := filepath.Join(dir, "مثال.dad")
		if m := titleRe.FindStringSubmatch(b.info); m != nil {
			path = filepath.Join(dir, m[1])
		} else {
			os.WriteFile(path, []byte(b.text), 0o644)
		}
		wrong := strings.HasPrefix(b.text, "// خطأ مقصود")
		if formatted, err := format.Source(b.text); err == nil && formatted != b.text {
			t.Errorf("line %d: the example is not laid out as «dad fmt» would:\n%s", b.start, formatted)
		}
		prog, diags := check.Load(path, func(p string) (string, error) {
			b, err := os.ReadFile(p)
			return string(b), err
		})
		var got strings.Builder
		if wrong {
			if !module.HasErrors(diags) {
				t.Errorf("line %d: an example marked «خطأ مقصود» has no mistake", b.start)
			}
			for _, d := range diags {
				got.WriteString(d.Diagnostic.String() + "\n")
			}
		} else if module.HasErrors(diags) {
			t.Errorf("line %d: %v\n%s", b.start, diags, b.text)
			continue
		}
		// is there an output block right after it, or after the command that runs it?
		o := i + 1
		if o+1 < len(bs) && bs[o].info == "sh" && strings.HasPrefix(bs[o].text, "dad run ") {
			b, o = bs[o], o+1
		}
		if o >= len(bs) || !strings.Contains(bs[o].info, "dad-out") || !onlyBlank(lines[b.end+1:bs[o].start-1]) {
			continue
		}
		if !wrong {
			var out bytes.Buffer
			in := interp.New(&out)
			in.MaxSteps = 10_000_000
			if err := in.RunProgram(prog); err != nil {
				out.WriteString(err.Error() + "\n")
			}
			got.WriteString(out.String())
		}
		want := bs[o].text
		if *update {
			replace[o] = got.String()
		} else if strings.TrimSpace(want) != strings.TrimSpace(got.String()) {
			t.Errorf("line %d: the page shows\n%s\nbut the example prints\n%s", bs[o].start, want, got.String())
		}
	}
	if *update && len(replace) > 0 {
		var out []string
		next := 0
		for i, b := range bs {
			r, ok := replace[i]
			if !ok {
				continue
			}
			out = append(out, lines[next:b.start]...)
			for _, l := range strings.Split(strings.TrimRight(r, "\n"), "\n") {
				if l == "" {
					out = append(out, "")
				} else {
					out = append(out, b.indent+l)
				}
			}
			next = b.end
		}
		out = append(out, lines[next:]...)
		os.WriteFile(page, []byte(strings.Join(out, "\n")), 0o644)
	}
}

func onlyBlank(lines []string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return false
		}
	}
	return true
}

// Every Ḍād example on the site, in English and in Arabic, is laid out as «dad fmt»
// would lay it out. Fragments that don't parse, snippets of example files, and the
// «before» half of the formatting example are left alone.
func TestDocsCodeLayout(t *testing.T) {
	pages, _ := filepath.Glob("../../docs/*.md")
	more, _ := filepath.Glob("../../docs/tutorial/*.md")
	for _, page := range append(pages, more...) {
		data, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range blocks(strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")) {
			if !strings.HasPrefix(b.info, "dad") || strings.Contains(b.text, "--8<--") ||
				strings.Contains(b.text, "// Before") || strings.Contains(b.text, "// قبل") {
				continue
			}
			if formatted, err := format.Source(b.text); err == nil && formatted != b.text {
				t.Errorf("%s, line %d: not laid out as «dad fmt» would:\n%s", filepath.Base(page), b.start, formatted)
			}
		}
	}
}
