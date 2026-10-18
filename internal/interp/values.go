package interp

import (
	"math"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
)

// Value is any Ḍād runtime value:
//
//	nil (عدم), bool, int64, float64, string, *List, *Map, *Range,
//	*Func, *Builtin, *StructType, *Struct, *EnumType, *Variant, *EnumVal, *Problem.
type Value any

type List struct{ Elems []Value }

type Map struct {
	Keys []Value
	Vals []Value
}

type Range struct {
	From, To, Step int64
	Inclusive      bool
}

func (r *Range) values() []Value {
	var out []Value
	if r.Step == 0 {
		return out
	}
	for i := r.From; rangeContinues(r, i); i += r.Step {
		out = append(out, i)
	}
	return out
}

// Func is a user function, method or lambda closed over its environment.
type Func struct {
	Name    string
	Params  []*ast.Param // nil for lambdas
	Lambda  []string     // lambda parameter names
	IsLam   bool
	Body    ast.Expr
	Env     *Env
	Recv    Value // bound receiver for methods (هذا)
	HasRecv bool
	Result  *ast.TypeRef
}

type Builtin struct {
	Name string
	Fn   func(in *Interp, c *callInfo) Value
	// BindFirst passes a state variable given as the first argument by
	// reference (*StateRef), so an element like حقل can change it.
	BindFirst bool
}

type StructType struct {
	Name    string
	Fields  []*ast.Param
	Methods map[string]*ast.FuncDecl
	Env     *Env
	Table   bool // declared with جدول: records are stored, and the last field is رقم
}

type Struct struct {
	Type *StructType
	Vals []Value
}

type EnumType struct {
	Name     string
	Variants []*Variant
}

type Variant struct {
	Enum   *EnumType
	Name   string
	Fields []*ast.Param
}

type EnumVal struct {
	Variant *Variant
	Vals    []Value
}

// Problem is the error value created by مشكلة("…") and raised with ارفع.
type Problem struct{ Message string }

// ---------------------------------------------------------------- type names

func typeName(v Value) string {
	switch v := v.(type) {
	case nil:
		return "عدم"
	case bool:
		return "منطقي"
	case int64:
		return "صحيح"
	case float64:
		return "عشري"
	case string:
		return "نص"
	case *List, *Range:
		return "قائمة"
	case *Map:
		return "قاموس"
	case *Func, *Builtin:
		return "دالة"
	case *Component:
		return "مكون"
	case *StructType:
		return "نوع"
	case *Struct:
		return v.Type.Name
	case *EnumType:
		return "تعداد"
	case *Variant:
		return v.Enum.Name
	case *EnumVal:
		return v.Variant.Enum.Name
	case *Problem:
		return "مشكلة"
	}
	return "؟"
}

// ---------------------------------------------------------------- formatting

func formatInt(n int64) string { return strconv.FormatInt(n, 10) }

func formatFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "ليس عدداً"
	case math.IsInf(f, 1):
		return "∞"
	case math.IsInf(f, -1):
		return "-∞"
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// Format renders a value the way اطبع prints it: digits 0-9, صح/خطأ, عدم.
func Format(v Value) string {
	switch v := v.(type) {
	case nil:
		return "عدم"
	case bool:
		if v {
			return "صح"
		}
		return "خطأ"
	case int64:
		return formatInt(v)
	case float64:
		return formatFloat(v)
	case string:
		return v
	case *List:
		return "[" + joinValues(v.Elems, "، ", true) + "]"
	case *Range:
		return "[" + joinValues(v.values(), "، ", true) + "]"
	case *Map:
		parts := make([]string, len(v.Keys))
		for i := range v.Keys {
			parts[i] = formatNested(v.Keys[i]) + ": " + formatNested(v.Vals[i])
		}
		if len(parts) == 0 {
			return "[:]"
		}
		return "[" + strings.Join(parts, "، ") + "]"
	case *Func:
		if v.Name != "" {
			return "<دالة " + v.Name + ">"
		}
		return "<دالة>"
	case *Builtin:
		return "<دالة " + v.Name + ">"
	case *Component:
		return "<مكون " + v.Name + ">"
	case *StateRef:
		return Format(v.b.val)
	case *StructType:
		return "<نوع " + v.Name + ">"
	case *EnumType:
		return "<تعداد " + v.Name + ">"
	case *Variant:
		return v.Name
	case *Struct:
		return v.Type.Name + "(" + formatFields(v.Type.Fields, v.Vals) + ")"
	case *EnumVal:
		if len(v.Vals) == 0 {
			return v.Variant.Name
		}
		return v.Variant.Name + "(" + formatFields(v.Variant.Fields, v.Vals) + ")"
	case *Problem:
		return "مشكلة: " + v.Message
	}
	return "؟"
}

func formatNested(v Value) string {
	if s, ok := v.(string); ok {
		return "«" + s + "»"
	}
	return Format(v)
}

func joinValues(vs []Value, sep string, nested bool) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		if nested {
			parts[i] = formatNested(v)
		} else {
			parts[i] = Format(v)
		}
	}
	return strings.Join(parts, sep)
}

func formatFields(fields []*ast.Param, vals []Value) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fields[i].Name + ": " + formatNested(v)
	}
	return strings.Join(parts, "، ")
}

// ---------------------------------------------------------------- equality

func equal(a, b Value) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case int64:
		switch y := b.(type) {
		case int64:
			return x == y
		case float64:
			return float64(x) == y
		}
		return false
	case float64:
		switch y := b.(type) {
		case int64:
			return x == float64(y)
		case float64:
			return x == y
		}
		return false
	case bool, string:
		return a == b
	case *List:
		y, ok := b.(*List)
		return ok && equalSlices(x.Elems, y.Elems)
	case *Range:
		if y, ok := b.(*Range); ok {
			return equalSlices(x.values(), y.values())
		}
		if y, ok := b.(*List); ok {
			return equalSlices(x.values(), y.Elems)
		}
		return false
	case *Struct:
		y, ok := b.(*Struct)
		return ok && x.Type == y.Type && equalSlices(x.Vals, y.Vals)
	case *EnumVal:
		y, ok := b.(*EnumVal)
		return ok && x.Variant == y.Variant && equalSlices(x.Vals, y.Vals)
	case *Problem:
		y, ok := b.(*Problem)
		return ok && x.Message == y.Message
	case *Map:
		y, ok := b.(*Map)
		if !ok || len(x.Keys) != len(y.Keys) {
			return false
		}
		for i := range x.Keys {
			j := y.find(x.Keys[i])
			if j < 0 || !equal(x.Vals[i], y.Vals[j]) {
				return false
			}
		}
		return true
	}
	return a == b
}

func equalSlices(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func (m *Map) find(k Value) int {
	for i, key := range m.Keys {
		if equal(key, k) {
			return i
		}
	}
	return -1
}
