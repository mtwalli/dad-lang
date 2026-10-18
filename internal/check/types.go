package check

import (
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
)

// Kind is the shape of a type the checker knows about.
type Kind int

const (
	Unknown   Kind = iota // anything; never reported
	Int                   // صحيح
	Float                 // عشري
	String                // نص
	Bool                  // منطقي
	Null                  // عدم
	List                  // [T]
	Map                   // [K: V]
	Range                 // من 1 إلى 3
	Func                  // a function or lambda
	Struct                // a value of a نوع or جدول
	Enum                  // a value of a تعداد
	Problem               // مشكلة
	TypeValue             // the name of a نوع or جدول itself: نقطة, مهمة
	EnumType              // the name of a تعداد itself
	Component             // a مكون
)

// Type is what the checker knows about a value. A nil *Type is unknown.
type Type struct {
	Kind     Kind
	Name     string // Struct, Enum, TypeValue, EnumType: the type's name
	Elem     *Type  // List element, Map value
	Key      *Type  // Map key
	Optional bool   // the value may be عدم
}

var (
	tInt    = &Type{Kind: Int}
	tFloat  = &Type{Kind: Float}
	tString = &Type{Kind: String}
	tBool   = &Type{Kind: Bool}
	tNull   = &Type{Kind: Null}
	tRange  = &Type{Kind: Range}
	tFunc   = &Type{Kind: Func}
	tProb   = &Type{Kind: Problem}
)

func listOf(elem *Type) *Type { return &Type{Kind: List, Elem: elem} }

func known(t *Type) bool { return t != nil && t.Kind != Unknown }

func numeric(t *Type) bool { return t != nil && (t.Kind == Int || t.Kind == Float) && !t.Optional }

// optional returns t marked as possibly عدم.
func optional(t *Type) *Type {
	if t == nil {
		return nil
	}
	c := *t
	c.Optional = true
	return &c
}

// String names a type the way Ḍād code writes it.
func (t *Type) String() string {
	if t == nil {
		return "غير معروف"
	}
	var s string
	switch t.Kind {
	case Int:
		s = "صحيح"
	case Float:
		s = "عشري"
	case String:
		s = "نص"
	case Bool:
		s = "منطقي"
	case Null:
		return "عدم"
	case List:
		if known(t.Elem) {
			s = "[" + t.Elem.String() + "]"
		} else {
			s = "قائمة"
		}
	case Map:
		s = "قاموس"
	case Range:
		s = "قائمة"
	case Func:
		s = "دالة"
	case Struct, Enum:
		s = t.Name
	case Problem:
		s = "مشكلة"
	case TypeValue, EnumType:
		s = "النوع " + t.Name
	case Component:
		s = "مكون"
	default:
		return "غير معروف"
	}
	if t.Optional {
		s += "؟"
	}
	return s
}

// assignable reports whether a value of type src can go where dst is expected.
// Unknown types are always assignable: the checker only reports what it is sure of.
func assignable(src, dst *Type) bool {
	if !known(src) || !known(dst) {
		return true
	}
	if src.Kind == Null {
		return dst.Optional
	}
	if src.Optional && !dst.Optional {
		return true // a smart cast after «!= عدم» may have made it safe; not tracked yet
	}
	switch {
	case src.Kind == dst.Kind:
	case src.Kind == Int && dst.Kind == Float:
		return true
	case src.Kind == Range && dst.Kind == List:
		return true
	default:
		return false
	}
	switch src.Kind {
	case Struct, Enum:
		return src.Name == dst.Name
	case List:
		return assignable(src.Elem, dst.Elem)
	case Map:
		return assignable(src.Key, dst.Key) && assignable(src.Elem, dst.Elem)
	}
	return true
}

// sameType is used to give an إذا with two branches a type.
func sameType(a, b *Type) *Type {
	if !known(a) || !known(b) {
		return nil
	}
	if a.Kind == Null {
		return optional(b)
	}
	if b.Kind == Null {
		return optional(a)
	}
	if assignable(a, b) && assignable(b, a) {
		if a.Optional || b.Optional {
			return optional(a)
		}
		return a
	}
	if numeric(a) && numeric(b) {
		return tFloat
	}
	return nil
}

// ---------------------------------------------------------------- written types

var builtinTypes = map[string]*Type{
	"صحيح": tInt, "عشري": tFloat, "نص": tString, "منطقي": tBool, "مشكلة": tProb,
	"قائمة": {Kind: List}, "قاموس": {Kind: Map}, "دالة": tFunc,
}

// anyType names accept every value, as in the interpreter: «س: أي».
var anyType = map[string]bool{"أي": true, "قيمة": true}

// fromRef turns a type written in the source into a Type. ok is false when a
// name in it is not a type.
func (c *checker) fromRef(r *ast.TypeRef) (*Type, bool) {
	if r == nil {
		return nil, true
	}
	var t *Type
	ok := true
	switch r.Kind {
	case ast.ListType:
		var e *Type
		e, ok = c.fromRef(r.Elem)
		t = listOf(e)
	case ast.MapType:
		k, ok1 := c.fromRef(r.Key)
		v, ok2 := c.fromRef(r.Elem)
		t, ok = &Type{Kind: Map, Key: k, Elem: v}, ok1 && ok2
	case ast.FuncType:
		for _, a := range r.Args {
			if _, ok1 := c.fromRef(a); !ok1 {
				ok = false
			}
		}
		if _, ok1 := c.fromRef(r.Elem); !ok1 {
			ok = false
		}
		t = tFunc
	default:
		if b, isBuiltin := builtinTypes[r.Name]; isBuiltin {
			bt := *b
			t = &bt
		} else if d, isDecl := c.types[r.Name]; isDecl {
			t = d
		} else if c.contracts[r.Name] || anyType[r.Name] {
			t = nil // a contract (any value that has its methods), or any value at all
		} else if en, isVariant := c.variantOf[r.Name]; isVariant {
			t = c.types[en] // «ش من نوع دائرة»: a case of a تعداد
		} else {
			return nil, false
		}
		for _, a := range r.Args {
			if _, ok1 := c.fromRef(a); !ok1 {
				ok = false
			}
		}
	}
	if t != nil && r.Optional {
		t = optional(t)
	}
	return t, ok
}

// checkRef reports unknown type names in a written type and returns the type.
func (c *checker) checkRef(r *ast.TypeRef) *Type {
	c.typeRefs(r)
	t, ok := c.fromRef(r)
	if !ok {
		name := firstUnknown(c, r)
		c.errorf(r.Pos, "النوع «%s» غير معروف%s", name, c.suggestType(name))
	}
	return t
}

func firstUnknown(c *checker, r *ast.TypeRef) string {
	if r == nil {
		return ""
	}
	if r.Kind == ast.NamedType {
		_, isVariant := c.variantOf[r.Name]
		if _, ok := builtinTypes[r.Name]; !ok && c.types[r.Name] == nil && !c.contracts[r.Name] && !isVariant && !anyType[r.Name] {
			return r.Name
		}
	}
	for _, x := range append(append([]*ast.TypeRef{}, r.Args...), r.Elem, r.Key) {
		if n := firstUnknown(c, x); n != "" {
			return n
		}
	}
	return ""
}

func (c *checker) suggestType(name string) string {
	var names []string
	for n := range builtinTypes {
		names = append(names, n)
	}
	for n := range c.types {
		names = append(names, n)
	}
	for n := range c.contracts {
		names = append(names, n)
	}
	return didYouMean(name, names)
}

// ---------------------------------------------------------------- suggestions

// didYouMean suggests the closest name, if one is close enough.
func didYouMean(name string, names []string) string {
	best, bestD := "", 3
	limit := 1
	if n := len([]rune(name)); n > 4 {
		limit = 2
	}
	for _, n := range names {
		if n == name || strings.HasPrefix(n, "_") {
			continue
		}
		if d := distance(name, n); d <= limit && (d < bestD || d == bestD && n < best) {
			best, bestD = n, d
		}
	}
	if best == "" {
		return ""
	}
	return "؛ هل تقصد «" + best + "»؟"
}

// distance is the edit distance between two names, counting letters.
func distance(a, b string) int {
	x, y := []rune(a), []rune(b)
	prev := make([]int, len(y)+1)
	cur := make([]int, len(y)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(x); i++ {
		cur[0] = i
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(y)]
}

// typeRefs records the names of user types written in a type.
func (c *checker) typeRefs(r *ast.TypeRef) {
	if r == nil || !c.recording() {
		return
	}
	if r.Kind == ast.NamedType {
		if sym := c.global.lookup(r.Name); sym != nil && (sym.kind == sType || sym.kind == sEnum || sym.kind == sVariant) {
			c.ref(r.Pos, sym.def)
		}
	}
	for _, x := range append(append([]*ast.TypeRef{}, r.Args...), r.Elem, r.Key) {
		c.typeRefs(x)
	}
}
