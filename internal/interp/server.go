package interp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/token"
)

// This file is the interpreter's side of "dad dev": it lists what a program offers
// on the web (routes, server functions, tables) and runs requests against it.
// The interpreter is not safe for concurrent use; callers serialize requests.

// ServerFunc describes a function marked خادم, which the browser may call.
type ServerFunc struct {
	Name   string        `json:"name"`
	Params []ServerParam `json:"params"`
}

// ServerParam is one parameter of a server function.
type ServerParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
}

// TableInfo describes a جدول and its rows.
type TableInfo struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Rows   []any    `json:"rows"`
}

// Routes lists the program's مسار declarations, in source order.
func (in *Interp) Routes() []*Route { return in.routes }

// ServerFunctions lists the functions marked خادم.
func (in *Interp) ServerFunctions() []ServerFunc {
	var out []ServerFunc
	for _, name := range in.serverFns {
		f := in.serverFn[name]
		sf := ServerFunc{Name: name, Params: []ServerParam{}}
		for _, p := range f.Params {
			sf.Params = append(sf.Params, ServerParam{Name: p.Name, Type: typeString(p.Type), Optional: p.Default != nil || (p.Type != nil && p.Type.Optional)})
		}
		out = append(out, sf)
	}
	return out
}

// Tables lists every جدول with its current rows.
func (in *Interp) Tables() ([]TableInfo, error) {
	var out []TableInfo
	err := in.safely(func() {
		for _, t := range in.tables {
			info := TableInfo{Name: t.Name, Rows: []any{}}
			for _, f := range t.Fields {
				info.Fields = append(info.Fields, f.Name)
			}
			for _, r := range in.loadRows(t, t.Fields[0].Pos) {
				info.Rows = append(info.Rows, in.ToJSON(r))
			}
			out = append(out, info)
		}
	})
	return out, err
}

// CallServer runs a server function with JSON arguments, given by parameter name,
// and returns its result as JSON data.
func (in *Interp) CallServer(name string, args map[string]any) (result any, err error) {
	isServer := false
	for _, n := range in.serverFns {
		isServer = isServer || n == name
	}
	if !isServer {
		return nil, fmt.Errorf("لا توجد دالة خادم باسم «%s»", name)
	}
	f := in.serverFn[name]
	err = in.safely(func() {
		ci := &callInfo{pos: f.Params0Pos(), name: name}
		for _, p := range f.Params {
			raw, ok := args[p.Name]
			if !ok {
				continue // bindArgs reports a missing argument or uses the default
			}
			ci.named = append(ci.named, namedArg{name: p.Name, val: in.FromJSON(raw, p.Type)})
		}
		for k := range args {
			if paramIndex(f.Params, k) < 0 {
				in.fail(ci.pos, "الدالة «%s» ليس لها معامل باسم «%s»", name, k)
			}
		}
		result = in.ToJSON(in.callValue(f, ci))
	})
	return result, err
}

// MatchRoute finds the route for a request and the values of its {params}.
func (in *Interp) MatchRoute(method, path string) (*Route, map[string]string) {
	for _, r := range in.routes {
		if r.Method != method {
			continue
		}
		if params, ok := matchPath(r.Path, path); ok {
			return r, params
		}
	}
	return nil, nil
}

// ServeRoute runs a route's body. Path values are bound by name (numbers become
// صحيح), and a JSON request body is available as «البيانات».
func (in *Interp) ServeRoute(r *Route, params map[string]string, body any) (result any, err error) {
	err = in.safely(func() {
		env := newEnv(r.env)
		for k, v := range params {
			var val Value = v
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				val = n
			}
			env.define(k, val, false)
		}
		env.define("البيانات", in.FromJSON(body, nil), false)
		v, _ := in.runBody(&Func{Name: r.Path, Body: r.body, Env: r.env, IsLam: true}, env)
		result = in.ToJSON(v)
	})
	return result, err
}

var paramRe = regexp.MustCompile(`\{([^{}/]+)\}`)

// matchPath matches "/api/مهام/{رقم}" against "/api/مهام/3".
func matchPath(pattern, path string) (map[string]string, bool) {
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	xs := strings.Split(strings.Trim(path, "/"), "/")
	if len(ps) != len(xs) {
		return nil, false
	}
	params := map[string]string{}
	for i, p := range ps {
		if m := paramRe.FindStringSubmatch(p); m != nil && m[0] == p {
			params[strings.TrimSpace(m[1])] = xs[i]
			continue
		}
		if p != xs[i] {
			return nil, false
		}
	}
	return params, true
}

// safely runs fn and turns a runtime error or an unhandled مشكلة into an error.
func (in *Interp) safely(fn func()) (err error) {
	in.steps = 0
	defer func() {
		if r := recover(); r != nil {
			err = in.recovered(r)
		}
	}()
	fn()
	return nil
}

// Params0Pos is the position used for errors about a call that comes from the web.
func (f *Func) Params0Pos() (p token.Pos) {
	if len(f.Params) > 0 {
		return f.Params[0].Pos
	}
	return p
}
