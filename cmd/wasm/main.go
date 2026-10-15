//go:build js && wasm

// Command wasm exposes the Ḍād lexer to the browser playground on the docs site.
//
// Build: GOOS=js GOARCH=wasm go build -o dad.wasm ./cmd/wasm
// It registers a global function dadTokenize(source) that returns JSON:
// {"tokens":[{kind,category,text,value,line,col}], "diagnostics":[{line,col,message,warning}]}.
package main

import (
	"encoding/json"
	"syscall/js"

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
	toks, diags := lexer.Tokenize(args[0].String())
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
	for _, d := range diags {
		res.Diagnostics = append(res.Diagnostics, jsDiagnostic{
			Line: d.Pos.Line, Col: d.Pos.Col, Message: d.String(), Warning: d.Warning,
		})
	}
	out, _ := json.Marshal(res)
	return string(out)
}

func main() {
	js.Global().Set("dadTokenize", js.FuncOf(tokenize))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("dad-ready"))
	select {}
}
