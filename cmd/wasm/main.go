//go:build js && wasm

// Command wasm exposes the Ḍād toolchain to the browser playground on the docs site.
//
// Build: GOOS=js GOARCH=wasm go build -o dad.wasm ./cmd/wasm
// It registers two global functions:
//
//	dadTokenize(source) → {"tokens":[…], "diagnostics":[…]}   (lexer + parser checks)
//	dadRun(source)      → {"output":"…", "error":"…", "notices":[…], "diagnostics":[…]}
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/mtwalli/dad-lang/internal/format"
	"github.com/mtwalli/dad-lang/internal/interp"
	"github.com/mtwalli/dad-lang/internal/lexer"
	"github.com/mtwalli/dad-lang/internal/parser"
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
		// Also report syntax errors from the parser (it repeats lexer warnings).
		_, diags = parser.ParseFile(src)
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
		file, diags := parser.ParseFile(args[0].String())
		res.Diagnostics = toJSDiagnostics(diags)
		if file == nil {
			res.Error = "لم يُشغَّل البرنامج بسبب الأخطاء أعلاه."
		} else {
			var out bytes.Buffer
			in := interp.New(&out)
			in.MaxSteps = 5_000_000
			if err := in.Run(file); err != nil {
				res.Error = err.Error()
			}
			res.Output = out.String()
			if len(in.Notices) > 0 {
				res.Notices = append(res.Notices, "لم يُشغَّل "+strings.Join(in.Notices, "، ")+
					" — الواجهات تأتي في المرحلة 2.")
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
	js.Global().Set("dadRun", js.FuncOf(run))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("dad-ready"))
	select {}
}
