package interp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/ast"
	"github.com/mtwalli/dad-lang/internal/token"
)

// A جدول is a value type with a hidden id field, «رقم», plus methods that read and
// write its rows through the interpreter's Store:
//
//	مهمة.أنشئ(عنوان: "…")   create and save a record
//	مهمة.الكل()              every record, oldest first
//	مهمة.حيث { هو.منجزة }   the records that match
//	مهمة.بالرقم(3)           one record, or عدم
//	مهمة.عدد()               how many records there are
//	م.احفظ()                 save a new or changed record, returns it
//	م.احذف()                 delete it
//
// Records are ordinary values, so «م.نسخة(منجزة: صح)» works and keeps the id.

const idField = "رقم"

// Route is a public web route declared with مسار.
type Route struct {
	Method string
	Path   string
	Pos    token.Pos
	body   ast.Expr
	env    *Env
}

func (in *Interp) declareTable(d *ast.TableDecl, env *Env) {
	var fields []*ast.Param
	for _, f := range d.Fields {
		if f.Type == nil {
			// «منجزة = خطأ»: the column's type comes from the default.
			c := *f
			c.Type = literalType(f.Default)
			f = &c
		}
		fields = append(fields, f)
	}
	fields = append(fields, &ast.Param{
		At:      d.At,
		Name:    idField,
		Type:    &ast.TypeRef{At: d.At, Kind: ast.NamedType, Name: "صحيح"},
		Default: &ast.IntLit{At: d.At, Value: 0},
	})
	st := &StructType{Name: d.Name, Fields: fields, Methods: map[string]*ast.FuncDecl{}, Env: env, Table: true}
	in.tables = append(in.tables, st)
	in.defineName(env, d.Name, st, d.Pos)
	if m, ok := in.Store.(Migrator); ok {
		if err := m.Prepare(d.Name, in.schema(st)); err != nil {
			in.fail(d.Pos, "%v", err)
		}
	}
}

// literalType is the type of a literal default, or nil when it isn't one.
func literalType(e ast.Expr) *ast.TypeRef {
	name := ""
	switch x := e.(type) {
	case *ast.BoolLit:
		name = "منطقي"
	case *ast.IntLit:
		name = "صحيح"
	case *ast.FloatLit:
		name = "عشري"
	case *ast.StrLit:
		name = "نص"
	case *ast.Unary:
		if x.Op == token.MINUS {
			return literalType(x.X)
		}
	}
	if name == "" {
		return nil
	}
	return &ast.TypeRef{At: ast.At{Pos: e.P()}, Kind: ast.NamedType, Name: name}
}

// schema describes a table's fields (without the id) for the store.
func (in *Interp) schema(t *StructType) []Field {
	var out []Field
	for _, f := range t.Fields[:len(t.Fields)-1] {
		fd := Field{Name: f.Name, Kind: fieldKind(f.Type), Optional: f.Type == nil || f.Type.Optional}
		if f.Default != nil && isConstant(f.Default) {
			fd.Default, fd.HasDefault = in.ToJSON(in.eval(f.Default, t.Env)), true
		}
		out = append(out, fd)
	}
	return out
}

// isConstant reports whether a default is made only of literals, so the store
// can use its value when it adds a column.
func isConstant(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.NullLit:
		return true
	case *ast.StrLit:
		for _, p := range x.Parts {
			if p.Expr != nil {
				return false
			}
		}
		return true
	case *ast.Unary:
		return x.Op == token.MINUS && isConstant(x.X)
	case *ast.ListLit:
		for _, el := range x.Elems {
			if !isConstant(el) {
				return false
			}
		}
		return true
	case *ast.MapLit:
		for i := range x.Keys {
			if !isConstant(x.Keys[i]) || !isConstant(x.Values[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// tableMethod runs a method called on a table, such as «مهمة.الكل()».
func (in *Interp) tableMethod(t *StructType, name string, ci *callInfo) (Value, bool) {
	switch name {
	case "أنشئ":
		rec := in.callValue(t, ci).(*Struct)
		return in.saveRecord(rec, ci.pos), true
	case "الكل":
		return &List{Elems: in.loadRows(t, ci.pos)}, true
	case "عدد":
		return int64(len(in.loadRows(t, ci.pos))), true
	case "حيث":
		pred := in.lambdaArg(ci, "حيث")
		var out []Value
		for _, r := range in.loadRows(t, ci.pos) {
			if in.truthy(in.callValue(pred, &callInfo{pos: ci.pos, args: []Value{r}}), ci.pos, "شرط «حيث»") {
				out = append(out, r)
			}
		}
		return &List{Elems: out}, true
	case "بالرقم":
		if len(ci.args) != 1 {
			in.fail(ci.pos, "«بالرقم» تأخذ رقم السجل، مثل: %s.بالرقم(1)", t.Name)
		}
		id, ok := ci.args[0].(int64)
		if !ok {
			in.fail(ci.pos, "«بالرقم» تأخذ عدداً صحيحاً، لكن وجدت «%s»", typeName(ci.args[0]))
		}
		for _, r := range in.loadRows(t, ci.pos) {
			if r.(*Struct).Vals[len(t.Fields)-1] == id {
				return r, true
			}
		}
		return nil, true
	}
	return nil, false
}

// recordMethod runs احفظ or احذف on a record.
func (in *Interp) recordMethod(r *Struct, name string, ci *callInfo) (Value, bool) {
	switch name {
	case "احفظ":
		return in.saveRecord(r, ci.pos), true
	case "احذف":
		id := r.Vals[len(r.Type.Fields)-1].(int64)
		if id == 0 {
			in.fail(ci.pos, "لا يمكن حذف سجل لم يُحفظ بعد")
		}
		ok, err := in.Store.Delete(r.Type.Name, id)
		if err != nil {
			in.fail(ci.pos, "تعذر حفظ البيانات: %v", err)
		}
		return ok, true
	}
	return nil, false
}

func (in *Interp) lambdaArg(ci *callInfo, method string) Value {
	if len(ci.args) != 1 {
		in.fail(ci.pos, "«%s» تأخذ دالة، مثل: %s { هو.منجزة }", method, method)
	}
	return ci.args[0]
}

// saveRecord inserts a new record (رقم = 0) or updates an existing one and returns it.
func (in *Interp) saveRecord(r *Struct, pos token.Pos) Value {
	t := r.Type
	row := Row{}
	for i, f := range t.Fields[:len(t.Fields)-1] {
		row[f.Name] = in.ToJSON(r.Vals[i])
	}
	idx := len(t.Fields) - 1
	id := r.Vals[idx].(int64)
	if id == 0 {
		newID, err := in.Store.Insert(t.Name, row)
		if err != nil {
			in.fail(pos, "تعذر حفظ البيانات: %v", err)
		}
		id = newID
	} else {
		ok, err := in.Store.Update(t.Name, id, row)
		if err != nil {
			in.fail(pos, "تعذر حفظ البيانات: %v", err)
		}
		if !ok {
			in.fail(pos, "لا يوجد سجل برقم %d في «%s»؛ ربما حُذف", id, t.Name)
		}
	}
	vals := append([]Value{}, r.Vals...)
	vals[idx] = id
	return &Struct{Type: t, Vals: vals}
}

func (in *Interp) loadRows(t *StructType, pos token.Pos) []Value {
	rows, err := in.Store.Rows(t.Name)
	if err != nil {
		in.fail(pos, "تعذرت قراءة البيانات: %v", err)
	}
	return in.recordsFromRows(t, rows, pos)
}

func (in *Interp) recordsFromRows(t *StructType, rows []Row, pos token.Pos) []Value {
	out := make([]Value, 0, len(rows))
	for _, row := range rows {
		out = append(out, in.recordFromRow(t, row, pos))
	}
	return out
}

// recordFromRow builds a record; fields missing from old rows get their defaults.
func (in *Interp) recordFromRow(t *StructType, row Row, pos token.Pos) *Struct {
	vals := make([]Value, len(t.Fields))
	for i, f := range t.Fields {
		raw, ok := row[f.Name]
		if raw == nil && f.Default != nil && (f.Type == nil || !f.Type.Optional) {
			ok = false // a column added or brought back after the row was saved
		}
		switch {
		case ok:
			vals[i] = in.convert(in.FromJSON(raw, f.Type), f.Type, pos, "الحقل «"+f.Name+"»")
		case f.Default != nil:
			vals[i] = in.eval(f.Default, newEnv(t.Env))
		default:
			vals[i] = nil
		}
	}
	return &Struct{Type: t, Vals: vals}
}

// ---------------------------------------------------------------- JSON

// ToJSON turns a value into plain data for the data file and the web.
func (in *Interp) ToJSON(v Value) any {
	switch x := v.(type) {
	case nil, bool, int64, float64, string:
		return x
	case *List:
		out := make([]any, len(x.Elems))
		for i, e := range x.Elems {
			out[i] = in.ToJSON(e)
		}
		return out
	case *Range:
		return in.ToJSON(&List{Elems: x.values()})
	case *Map:
		out := map[string]any{}
		for i, k := range x.Keys {
			key, ok := k.(string)
			if !ok {
				key = Format(k)
			}
			out[key] = in.ToJSON(x.Vals[i])
		}
		return out
	case *Struct:
		out := map[string]any{}
		for i, f := range x.Type.Fields {
			out[f.Name] = in.ToJSON(x.Vals[i])
		}
		return out
	case *EnumVal:
		if len(x.Vals) == 0 {
			return x.Variant.Name
		}
		out := map[string]any{"_": x.Variant.Name}
		for i, f := range x.Variant.Fields {
			out[f.Name] = in.ToJSON(x.Vals[i])
		}
		return out
	case *Problem:
		return map[string]any{"مشكلة": x.Message}
	}
	return Format(v)
}

// FromJSON turns plain data into a value, guided by the expected type when known.
func (in *Interp) FromJSON(v any, t *ast.TypeRef) Value {
	switch x := v.(type) {
	case nil:
		return nil
	case float64:
		if x == float64(int64(x)) && (t == nil || t.Name != "عشري") {
			return int64(x)
		}
		return x
	case int, int64, bool, string:
		if i, ok := x.(int); ok {
			return int64(i)
		}
		if s, ok := x.(string); ok && t != nil && t.Kind == ast.NamedType {
			switch t.Name {
			case "صحيح":
				if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
					return n
				}
			case "عشري":
				if f, ok := parseNumber(s); ok {
					return f
				}
			}
		}
		return x
	case []any:
		var elem *ast.TypeRef
		if t != nil && t.Kind == ast.ListType {
			elem = t.Elem
		}
		out := make([]Value, len(x))
		for i, e := range x {
			out[i] = in.FromJSON(e, elem)
		}
		return &List{Elems: out}
	case map[string]any:
		if t != nil && t.Kind == ast.NamedType {
			if st, ok := in.lookupType(t.Name).(*StructType); ok {
				vals := make([]Value, len(st.Fields))
				for i, f := range st.Fields {
					if raw, ok := x[f.Name]; ok {
						vals[i] = in.convert(in.FromJSON(raw, f.Type), f.Type, token.Pos{}, "الحقل «"+f.Name+"»")
					} else if f.Default != nil {
						vals[i] = in.eval(f.Default, newEnv(st.Env))
					}
				}
				return &Struct{Type: st, Vals: vals}
			}
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m := &Map{}
		for _, k := range keys {
			m.Keys = append(m.Keys, k)
			m.Vals = append(m.Vals, in.FromJSON(x[k], nil))
		}
		return m
	}
	return fmt.Sprint(v)
}

func (in *Interp) lookupType(name string) Value {
	if b := in.globals.lookup(name); b != nil {
		return b.val
	}
	return nil
}
