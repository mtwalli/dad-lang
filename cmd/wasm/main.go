//go:build js && wasm

// Command wasm exposes the Ḍād toolchain to the browser playground on the docs site.
//
// Build: GOOS=js GOARCH=wasm go build -o dad.wasm ./cmd/wasm
// It registers two global functions:
//
//	dadTokenize(source) → {"tokens":[…], "diagnostics":[…]}   (lexer + parser checks)
//	dadRun(source)      → {"output":"…", "error":"…", "notices":[…], "diagnostics":[…], "ui":{…}}
//	dadUIEvent(event)   → {"html":"…", "version":n} or {"error":"…"}   (a click or typing in the interface)
//	dadFormat(source)   → {"text":"…"} or {"error":"…"}
//
// When the program has a main component, "ui" holds its first render; the page
// sends events back with dadUIEvent. Tables are kept in memory.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/mtwalli/dad-lang/internal/check"
	"github.com/mtwalli/dad-lang/internal/format"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/token"
)

type jsToken struct {
	Kind     string `json:"kind"`
	Category string `json:"category"`
	Text     string `json:"text"`
	Value    string `json:"value"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
}

type jsDiagnostic struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Warning bool   `json:"warning"`
}

type result struct {
	Tokens      []jsToken      `json:"tokens"`
	Diagnostics []jsDiagnostic `json:"diagnostics"`
}

func category(k token.Kind) string {
	switch {
	case k.IsKeyword():
		return "keyword"
	case k == token.IDENT:
		return "name"
	case k == token.INT || k == token.FLOAT:
		return "number"
	case k == token.STRING:
		return "string"
	case k == token.NEWLINE:
		return "newline"
	case k == token.ILLEGAL:
		return "illegal"
	default:
		return "symbol"
	}
}

func tokenize(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return `{"tokens":[],"diagnostics":[]}`
	}
	src := args[0].String()
	toks, diags := lexer.Tokenize(src)
	if !lexer.HasErrors(diags) {
		// Also report the parser's and the checker's problems (they repeat lexer warnings).
		_, diags = check.Source(src)
	}
	res := result{Tokens: []jsToken{}, Diagnostics: []jsDiagnostic{}}
	for _, t := range toks {
		if t.Kind == token.EOF {
			continue
		}
		res.Tokens = append(res.Tokens, jsToken{
			Kind: t.Kind.String(), Category: category(t.Kind),
			Text: t.Lit, Value: t.Value, Line: t.Pos.Line, Col: t.Pos.Col,
		})
	}
	res.Diagnostics = toJSDiagnostics(diags)
	out, _ := json.Marshal(res)
	return string(out)
}

func toJSDiagnostics(diags []lexer.Diagnostic) []jsDiagnostic {
	out := []jsDiagnostic{}
	for _, d := range diags {
		out = append(out, jsDiagnostic{Line: d.Pos.Line, Col: d.Pos.Col, Message: d.String(), Warning: d.Warning})
	}
	return out
}

type runResult struct {
	Output      string         `json:"output"`
	Error       string         `json:"error"`
	Notices     []string       `json:"notices"`
	Diagnostics []jsDiagnostic `json:"diagnostics"`
	UI          *uiResult      `json:"ui,omitempty"`
}

type uiResult struct {
	Title   string `json:"title"`
	HTML    string `json:"html"`
	Version int    `json:"version"`
	Error   string `json:"error,omitempty"`
}

// The program whose interface is showing, kept between events.
var (
	uiInterp *interp.Interp
	uiSess   *interp.UISession
	uiOut    *bytes.Buffer
)

// uiEvent runs a click or typing from the interface and renders it again.
func uiEvent(_ js.Value, args []js.Value) (ret any) {
	reply := map[string]any{}
	defer func() {
		if r := recover(); r != nil {
			reply = map[string]any{"error": "خطأ داخلي في المفسّر — نرجو الإبلاغ عنه على GitHub."}
		}
		out, _ := json.Marshal(reply)
		ret = string(out)
	}()
	if uiInterp == nil || len(args) == 0 {
		reply["error"] = "شغّل البرنامج أولاً"
		return
	}
	var ev struct {
		Version int    `json:"v"`
		ID      string `json:"id"`
		Value   any    `json:"value"`
		Nav     string `json:"nav"` // a link to one of the program's pages
	}
	json.Unmarshal([]byte(args[0].String()), &ev)
	uiInterp.MaxSteps = 5_000_000
	if ev.Nav != "" {
		uiInterp.Open(uiSess, ev.Nav)
		reply["path"] = uiSess.Path
	} else {
		if err := uiInterp.Event(uiSess, ev.Version, ev.ID, ev.Value); err != nil && err != interp.ErrStale {
			reply["error"] = err.Error()
		}
		if uiSess.Away != "" {
			reply = map[string]any{"location": uiSess.Away}
			return
		}
		if uiSess.Moved {
			reply["path"] = uiSess.Path
		}
	}
	html, err := uiInterp.RenderPage(uiSess)
	if err != nil {
		reply["error"] = err.Error()
	} else {
		reply["html"], reply["version"] = html, uiSess.Version
	}
	reply["title"] = uiTitle(uiSess)
	if uiOut.Len() > 0 {
		reply["output"] = uiOut.String() // what اطبع printed while handling the event
		uiOut.Reset()
	}
	return
}

// uiTitle is the title of the page showing: its first heading.
func uiTitle(sess *interp.UISession) string {
	if sess.Title != "" {
		return sess.Title
	}
	return "ضاد"
}

// run executes a program with a step limit so a runaway loop cannot freeze the page.
func run(_ js.Value, args []js.Value) (ret any) {
	res := runResult{Notices: []string{}}
	defer func() {
		if r := recover(); r != nil {
			// An interpreter bug must not take the playground down with it.
			res.Error = "خطأ داخلي في المفسّر — نرجو الإبلاغ عنه على GitHub."
			out, _ := json.Marshal(res)
			ret = string(out)
		}
	}()
	if len(args) > 0 {
		file, diags := check.Source(args[0].String())
		res.Diagnostics = toJSDiagnostics(diags)
		if file == nil || lexer.HasErrors(diags) {
			res.Error = "لم يُشغَّل البرنامج بسبب الأخطاء أعلاه."
		} else {
			var out bytes.Buffer
			in := interp.New(&out)
			in.MaxSteps = 5_000_000
			if err := in.Run(file); err != nil {
				res.Error = err.Error()
			}
			res.Output = out.String()
			uiInterp, uiSess = nil, nil
			if len(in.Pages()) > 0 && res.Error == "" {
				out.Reset()
				sess := interp.NewUISession()
				in.Open(sess, "/")
				html, err := in.RenderPage(sess)
				ui := &uiResult{Title: uiTitle(sess), HTML: html, Version: sess.Version}
				if err != nil {
					ui.Error = err.Error()
				}
				res.UI = ui
				res.Output += out.String()
				out.Reset()
				uiInterp, uiSess, uiOut = in, sess, &out
			}
			if len(in.Notices) > 0 {
				res.Notices = append(res.Notices, "لم يُشغَّل "+strings.Join(in.Notices, "، ")+
					".")
			}
		}
	}
	out, _ := json.Marshal(res)
	return string(out)
}

// formatSource lays out a program the standard way (dad fmt).
func formatSource(_ js.Value, args []js.Value) any {
	res := map[string]string{}
	if len(args) > 0 {
		out, err := format.Source(args[0].String())
		if err != nil {
			res["error"] = err.Error()
		} else {
			res["text"] = out
		}
	}
	b, _ := json.Marshal(res)
	return string(b)
}

func main() {
	js.Global().Set("dadTokenize", js.FuncOf(tokenize))
	js.Global().Set("dadFormat", js.FuncOf(formatSource))
	js.Global().Set("dadUIEvent", js.FuncOf(uiEvent))
	js.Global().Set("dadRun", js.FuncOf(run))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("dad-ready"))
	select {}
}
