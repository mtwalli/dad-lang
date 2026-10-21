package lsp_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mtwalli/dad-lang/internal/lsp"
)

// client is a minimal editor talking to the server through pipes. Like a
// real editor, it reads everything the server sends as it arrives.
type client struct {
	t       *testing.T
	in      io.WriteCloser
	out     *bufio.Reader
	nextID  int
	mu      sync.Mutex
	notes   []map[string]any // notifications received
	replies map[float64]chan map[string]any
}

func start(t *testing.T) *client {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	go func() {
		lsp.Serve(inR, outW)
		outW.Close()
	}()
	c := &client{t: t, in: inW, out: bufio.NewReader(outR), replies: map[float64]chan map[string]any{}}
	go func() {
		for {
			m, ok := c.read()
			if !ok {
				return
			}
			c.mu.Lock()
			if id, isReply := m["id"].(float64); isReply {
				if ch := c.replies[id]; ch != nil {
					ch <- m
				}
			} else {
				c.notes = append(c.notes, m)
			}
			c.mu.Unlock()
		}
	}()
	t.Cleanup(func() { c.notify("exit", nil); inW.Close() })
	c.request("initialize", map[string]any{"initializationOptions": map[string]any{"keywords": false}})
	return c
}

func (c *client) write(m map[string]any) {
	m["jsonrpc"] = "2.0"
	b, _ := json.Marshal(m)
	fmt.Fprintf(c.in, "Content-Length: %d\r\n\r\n%s", len(b), b)
}

func (c *client) read() (map[string]any, bool) {
	length := 0
	for {
		line, err := c.out.ReadString('\n')
		if err != nil {
			return nil, false
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			length, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	b := make([]byte, length)
	io.ReadFull(c.out, b)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m, true
}

func (c *client) notify(method string, params any) {
	c.write(map[string]any{"method": method, "params": params})
}

func (c *client) request(method string, params any) any {
	c.nextID++
	id := c.nextID
	done := make(chan map[string]any, 1)
	c.mu.Lock()
	c.replies[float64(id)] = done
	c.mu.Unlock()
	c.write(map[string]any{"id": id, "method": method, "params": params})
	select {
	case m := <-done:
		if m["error"] != nil {
			return m["error"]
		}
		return m["result"]
	case <-time.After(5 * time.Second):
		c.t.Fatalf("%s: no answer", method)
		return nil
	}
}

const uri = "file:///tmp/app.dad"

const src = `نوع نقطة(س: عشري، ص: عشري) {
    دالة بعد() = جذر(س * س + ص * ص)
}

دالة مربع(عدد: صحيح): صحيح = عدد * عدد

ثابت ن = نقطة(س: 3، ص: 4)
اطبع(ن.بعد()، مربع(2)، مربع(عدد: 3))
اطبع(نن)
`

func at(line, char int) map[string]any {
	return map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": line, "character": char}}
}

func TestLanguageServer(t *testing.T) {
	c := start(t)
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "dad", "version": 1, "text": src}})

	// Problems arrive as a notification.
	c.request("textDocument/hover", at(0, 0)) // any request, to collect the notification first
	var diags []any
	c.mu.Lock()
	notes := append([]map[string]any{}, c.notes...)
	c.mu.Unlock()
	for _, n := range notes {
		if n["method"] == "textDocument/publishDiagnostics" {
			diags = n["params"].(map[string]any)["diagnostics"].([]any)
		}
	}
	if len(diags) != 1 || !strings.Contains(fmt.Sprint(diags[0]), "هل تقصد «ن»") {
		t.Errorf("diagnostics: %v", diags)
	}

	// Hover on a call to مربع shows its declaration.
	h := c.request("textDocument/hover", at(7, 15)).(map[string]any)
	if v := h["contents"].(map[string]any)["value"].(string); !strings.Contains(v, "دالة مربع(عدد: صحيح): صحيح") {
		t.Errorf("hover: %v", v)
	}

	// Go to definition: from «ن.بعد()» to the method.
	def := c.request("textDocument/definition", at(7, 7)).(map[string]any)
	if r := def["range"].(map[string]any)["start"].(map[string]any); r["line"] != 1.0 || r["character"] != 9.0 {
		t.Errorf("definition of بعد: %v", def)
	}

	// References to مربع: the declaration and two calls.
	if refs := c.request("textDocument/references", at(4, 6)).([]any); len(refs) != 3 {
		t.Errorf("references: %v", refs)
	}

	// Rename the parameter عدد: the declaration, its two uses and the named argument.
	r := c.request("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri},
		"position": map[string]any{"line": 4, "character": 10}, "newName": "قيمة"}).(map[string]any)
	if edits := r["changes"].(map[string]any)[uri].([]any); len(edits) != 4 {
		t.Errorf("rename: %v", edits)
	}
	// A keyword is not a valid new name.
	if e, ok := c.request("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uri},
		"position": map[string]any{"line": 4, "character": 10}, "newName": "إذا"}).(map[string]any); !ok || e["message"] == nil {
		t.Errorf("renaming to a keyword should fail")
	}

	// Completion after «ن.» lists the fields and methods of نقطة.
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 2},
		"contentChanges": []any{map[string]any{"text": src + "اطبع(ن.)\n"}}})
	items := c.request("textDocument/completion", at(9, 7)).([]any)
	var labels []string
	for _, it := range items {
		labels = append(labels, it.(map[string]any)["label"].(string))
	}
	if strings.Join(labels, " ") != "بعد س ص نسخة" {
		t.Errorf("member completion: %v", labels)
	}
	// Plain completion lists names in scope.
	items = c.request("textDocument/completion", at(8, 5)).([]any)
	labels = nil
	for _, it := range items {
		labels = append(labels, it.(map[string]any)["label"].(string))
	}
	if !strings.Contains(" "+strings.Join(labels, " ")+" ", " مربع ") || !strings.Contains(strings.Join(labels, " "), "نقطة") {
		t.Errorf("completion: %v", labels)
	}

	// The outline: the type with its fields and method, the function and the constant.
	syms := c.request("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}}).([]any)
	var outline []string
	for _, s := range syms {
		m := s.(map[string]any)
		outline = append(outline, m["name"].(string))
		if m["name"] == "نقطة" && len(m["children"].([]any)) != 3 {
			t.Errorf("نقطة should have 3 children: %v", m["children"])
		}
	}
	if strings.Join(outline, " ") != "نقطة مربع ن" {
		t.Errorf("outline: %v", outline)
	}

	// Formatting returns the whole file laid out.
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 3},
		"contentChanges": []any{map[string]any{"text": "اطبع( 1+2 )\n"}}})
	edits := c.request("textDocument/formatting", map[string]any{"textDocument": map[string]any{"uri": uri}}).([]any)
	if len(edits) != 1 || edits[0].(map[string]any)["newText"] != "اطبع(1 + 2)\n" {
		t.Errorf("formatting: %v", edits)
	}

	// Columns are in UTF-16 units: a letter outside the BMP counts as two.
	c.notify("textDocument/didChange", map[string]any{"textDocument": map[string]any{"uri": uri, "version": 4},
		"contentChanges": []any{map[string]any{"text": "ثابت ق = \"😀\"؛ ثابت م = ق\n"}}})
	def = c.request("textDocument/definition", at(0, 25)).(map[string]any)
	if s := def["range"].(map[string]any)["start"].(map[string]any); s["character"] != 5.0 {
		t.Errorf("UTF-16 columns: %v", def)
	}
}

// A file is checked with the files it imports; definitions lead into them.
func TestImports(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "أشكال.dad")
	os.WriteFile(lib, []byte("// أشكال\nدالة مساحة(ض: عشري): عشري = ض * ض\nخاص ثابت سر = 1\n"), 0o644)
	main := filepath.Join(dir, "main.dad")
	mainURI := (&url.URL{Scheme: "file", Path: filepath.ToSlash(main)}).String()
	if !strings.HasPrefix(filepath.ToSlash(main), "/") {
		mainURI = (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(main)}).String()
	}
	c := start(t)
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "dad", "version": 1,
		"text": "استورد مساحة، سر من \"./أشكال\"\nاطبع(مساحة(2))\n"}})
	pos := func(line, char int) map[string]any {
		return map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": line, "character": char}}
	}
	c.request("textDocument/hover", pos(1, 6))
	var diags []any
	c.mu.Lock()
	for _, n := range c.notes {
		if n["method"] == "textDocument/publishDiagnostics" {
			diags = n["params"].(map[string]any)["diagnostics"].([]any)
		}
	}
	c.mu.Unlock()
	if len(diags) != 1 || !strings.Contains(fmt.Sprint(diags[0]), "«سر» خاص بالملف «أشكال»") {
		t.Errorf("diagnostics: %v", diags)
	}
	h := c.request("textDocument/hover", pos(1, 6)).(map[string]any)
	if v := h["contents"].(map[string]any)["value"].(string); !strings.Contains(v, "دالة مساحة(ض: عشري): عشري — مستورد من «أشكال»") {
		t.Errorf("hover: %v", v)
	}
	def := c.request("textDocument/definition", pos(1, 6)).(map[string]any)
	if u, _ := url.Parse(def["uri"].(string)); filepath.Base(filepath.FromSlash(u.Path)) != "أشكال.dad" {
		t.Errorf("definition: %v", def)
	}
	if r := def["range"].(map[string]any)["start"].(map[string]any); r["line"] != 1.0 || r["character"] != 5.0 {
		t.Errorf("definition range: %v", def)
	}
}

// Pages show in the outline by their address, and an address is not offered as a name.
func TestPagesInTheOutline(t *testing.T) {
	c := start(t)
	text := "صفحة \"/\" = الرئيسية()\nصفحة \"/مهمة/{رقم}\" {\n    عنوان(\"{رقم}\")\n}\nمكون الرئيسية() {\n    عنوان(\"أ\")\n}\n"
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "dad", "version": 1, "text": text}})
	syms := c.request("textDocument/documentSymbol", map[string]any{"textDocument": map[string]any{"uri": uri}}).([]any)
	var outline []string
	for _, s := range syms {
		m := s.(map[string]any)
		outline = append(outline, m["name"].(string))
		if m["name"] == "/مهمة/{رقم}" {
			r := m["range"].(map[string]any)
			if r["start"].(map[string]any)["character"] != 5.0 || r["end"].(map[string]any)["character"] != 18.0 {
				t.Errorf("range of the page: %v", r)
			}
		}
	}
	if strings.Join(outline, " ") != "/ /مهمة/{رقم} الرئيسية" {
		t.Errorf("outline: %v", outline)
	}
	items := c.request("textDocument/completion", at(5, 4)).([]any)
	for _, it := range items {
		if l := it.(map[string]any)["label"].(string); strings.HasPrefix(l, "/") {
			t.Errorf("completion offers the address %q", l)
		}
	}
}

// A program of three files: names, fields and files lead across them.
func TestAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"نماذج.dad": "نوع منتج(اسم: نص، سعر: عشري)\n\nدالة بالضريبة(سعر: عشري): عشري = سعر * 1.15\n",
		"أسعار.dad": "استورد منتج، بالضريبة من \"./نماذج\"\n\nدالة السعر(م: منتج): عشري = بالضريبة(م.سعر)\n",
		"main.dad":  "استورد منتج من \"./نماذج\"\nاستورد السعر من \"./أسعار\"\n\nثابت م = منتج(\"خبز\"، 2.5)\nاطبع(م.اسم، السعر(م))\n",
	}
	uris := map[string]string{}
	for name, text := range files {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(text), 0o644)
		u := filepath.ToSlash(p)
		if !strings.HasPrefix(u, "/") {
			u = "/" + u
		}
		uris[name] = (&url.URL{Scheme: "file", Path: u}).String()
	}
	base := func(u string) string {
		parsed, _ := url.Parse(u)
		return filepath.Base(filepath.FromSlash(parsed.Path))
	}
	c := start(t)
	open := func(name string) {
		c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": uris[name], "languageId": "dad", "version": 1, "text": files[name]}})
	}
	pos := func(name string, line, char int) map[string]any {
		return map[string]any{"textDocument": map[string]any{"uri": uris[name]}, "position": map[string]any{"line": line, "character": char}}
	}
	start := func(loc map[string]any) string {
		r := loc["range"].(map[string]any)["start"].(map[string]any)
		return fmt.Sprintf("%s:%v:%v", base(loc["uri"].(string)), r["line"], r["character"])
	}
	open("main.dad")

	// a field of an imported type
	if def, ok := c.request("textDocument/definition", pos("main.dad", 4, 8)).(map[string]any); !ok || start(def) != "نماذج.dad:0:9" {
		t.Errorf("definition of م.اسم: %v", def)
	}
	h := c.request("textDocument/hover", pos("main.dad", 4, 8)).(map[string]any)
	if r := h["range"].(map[string]any)["start"].(map[string]any); r["line"] != 4.0 || r["character"] != 7.0 {
		t.Errorf("hover range: %v", h["range"])
	}
	// the file an import names
	if def, ok := c.request("textDocument/definition", pos("main.dad", 1, 20)).(map[string]any); !ok || start(def) != "أسعار.dad:0:0" {
		t.Errorf("definition of an import's file: %v", def)
	}
	links := c.request("textDocument/documentLink", map[string]any{"textDocument": map[string]any{"uri": uris["main.dad"]}}).([]any)
	var targets []string
	for _, l := range links {
		targets = append(targets, base(l.(map[string]any)["target"].(string)))
	}
	if strings.Join(targets, " ") != "نماذج.dad أسعار.dad" {
		t.Errorf("links: %v", targets)
	}

	// every file that names منتج, from the file that declares it
	open("نماذج.dad")
	var found []string
	for _, l := range c.request("textDocument/references", pos("نماذج.dad", 0, 5)).([]any) {
		found = append(found, start(l.(map[string]any)))
	}
	want := "نماذج.dad:0:4 أسعار.dad:0:7 أسعار.dad:2:14 main.dad:0:7 main.dad:3:9"
	if strings.Join(found, " ") != want {
		t.Errorf("references:\n got %v\nwant %v", strings.Join(found, " "), want)
	}

	// renaming from a file that imports the name changes the file that declares it
	open("أسعار.dad")
	res := c.request("textDocument/rename", map[string]any{"textDocument": map[string]any{"uri": uris["أسعار.dad"]},
		"position": map[string]any{"line": 2, "character": 30}, "newName": "مع_الضريبة"}).(map[string]any)
	var edited []string
	for u, edits := range res["changes"].(map[string]any) {
		edited = append(edited, fmt.Sprintf("%s:%d", base(u), len(edits.([]any))))
	}
	sort.Strings(edited)
	if strings.Join(edited, " ") != "أسعار.dad:2 نماذج.dad:1" {
		t.Errorf("rename: %v", edited)
	}

	// Go to Symbol in Workspace
	var syms []string
	for _, s := range c.request("workspace/symbol", map[string]any{"query": "ضريبه"}).([]any) {
		m := s.(map[string]any)
		syms = append(syms, m["name"].(string)+" @"+m["containerName"].(string))
	}
	if strings.Join(syms, " ") != "بالضريبة @نماذج" {
		t.Errorf("workspace symbols: %v", syms)
	}
	var fields []string
	for _, s := range c.request("workspace/symbol", map[string]any{"query": "سعر"}).([]any) {
		m := s.(map[string]any)
		fields = append(fields, m["name"].(string)+" @"+m["containerName"].(string))
	}
	if strings.Join(fields, " ") != "السعر @أسعار سعر @منتج" {
		t.Errorf("workspace symbols for سعر: %v", fields)
	}
}
