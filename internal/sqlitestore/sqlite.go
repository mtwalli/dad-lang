// Package sqlitestore keeps Ḍād tables (جدول) in a SQLite database.
//
// Each جدول becomes a SQL table with a column per field and «رقم» as its
// INTEGER PRIMARY KEY. When a program's tables change, Prepare migrates the
// database: new fields are added (filled with their default), removed fields
// are kept in the file but hidden, and a field whose type changed is converted
// row by row, or the change is refused with the rows that can't be converted.
//
// It uses modernc.org/sqlite, a pure-Go SQLite, so dad still cross-compiles
// without cgo. It is kept out of package interp so the WebAssembly playground
// doesn't pull it in.
package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mtwalli/dad-lang/internal/interp"

	_ "modernc.org/sqlite"
)

const idField = "رقم"

// Store is a SQLite-backed interp.Store. It also implements interp.Migrator and
// interp.Querier.
type Store struct {
	db     *sql.DB
	path   string
	fields map[string][]interp.Field // declared fields per table, set by Prepare

	// Notes are messages for the developer about migrations, such as a field
	// that was removed from the program but whose data is kept.
	Notes []string

	pending  map[string][]interp.Row // rows to import into tables not created yet
	importOf string                  // the JSON file they came from
}

// Open opens or creates the database file at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // dad runs one request at a time; one connection keeps SQLite simple
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		`CREATE TABLE IF NOT EXISTS _dad_fields (
			tbl  TEXT NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			pos  INTEGER NOT NULL,
			PRIMARY KEY (tbl, name)
		)`,
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("تعذر فتح قاعدة البيانات «%s»: %v", path, err)
		}
	}
	return &Store{db: db, path: path, fields: map[string][]interp.Field{}}, nil
}

// Path is the database file's location.
func (s *Store) Path() string { return s.path }

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

func quote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func sqlType(kind string) string {
	switch kind {
	case "int", "bool":
		return "INTEGER"
	case "float":
		return "REAL"
	}
	return "TEXT" // text and json
}

var kindNames = map[string]string{"int": "صحيح", "float": "عشري", "text": "نص", "bool": "منطقي", "json": "قيمة مركبة"}

// ---------------------------------------------------------------- migrations

type stored struct {
	kind string
	pos  int
}

// Prepare creates the table or brings it up to date with fields.
func (s *Store) Prepare(table string, fields []interp.Field) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	have := map[string]stored{}
	rows, err := tx.Query(`SELECT name, kind, pos FROM _dad_fields WHERE tbl = ?`, table)
	if err != nil {
		return err
	}
	for rows.Next() {
		var name, kind string
		var pos int
		if err := rows.Scan(&name, &kind, &pos); err != nil {
			rows.Close()
			return err
		}
		have[name] = stored{kind, pos}
	}
	rows.Close()

	created := false
	if len(have) == 0 && !s.tableExists(tx, table) {
		cols := []string{quote(idField) + " INTEGER PRIMARY KEY AUTOINCREMENT"}
		for _, f := range fields {
			cols = append(cols, quote(f.Name)+" "+sqlType(f.Kind))
		}
		if _, err := tx.Exec("CREATE TABLE " + quote(table) + " (" + strings.Join(cols, ", ") + ")"); err != nil {
			return err
		}
		created = true
	}

	count := func() (int64, error) {
		var n int64
		err := tx.QueryRow("SELECT COUNT(*) FROM " + quote(table)).Scan(&n)
		return n, err
	}
	next := len(have)
	for _, f := range fields {
		old, exists := have[f.Name]
		switch {
		case created:
		case !exists:
			n, err := count()
			if err != nil {
				return err
			}
			if n > 0 && !f.HasDefault && !f.Optional {
				return fmt.Errorf("أضفت الحقل «%s» إلى «%s» وفيه %d سجلات؛ أعطه قيمة افتراضية (%s = …) أو اجعله اختيارياً (%s: %s؟)",
					f.Name, table, n, f.Name, f.Name, kindNames[f.Kind])
			}
			if _, err := tx.Exec("ALTER TABLE " + quote(table) + " ADD COLUMN " + quote(f.Name) + " " + sqlType(f.Kind)); err != nil {
				return err
			}
			if f.HasDefault && n > 0 {
				v, err := encode(f.Kind, f.Default)
				if err != nil {
					return err
				}
				if _, err := tx.Exec("UPDATE "+quote(table)+" SET "+quote(f.Name)+" = ?", v); err != nil {
					return err
				}
			}
		case old.kind != f.Kind:
			if err := convertColumn(tx, table, f, old.kind); err != nil {
				return err
			}
		}
		if !exists || old.kind != f.Kind {
			if _, err := tx.Exec(`INSERT INTO _dad_fields (tbl, name, kind, pos) VALUES (?, ?, ?, ?)
				ON CONFLICT (tbl, name) DO UPDATE SET kind = excluded.kind`, table, f.Name, f.Kind, next); err != nil {
				return err
			}
			next++
		}
	}
	for name := range have {
		if !hasField(fields, name) {
			s.Notes = append(s.Notes, fmt.Sprintf("الحقل «%s» لم يعد في «%s»؛ بياناته باقية في قاعدة البيانات وتعود إذا أعدته", name, table))
		}
	}
	if created {
		if err := s.importRows(tx, table, fields); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.fields[table] = fields
	return nil
}

func hasField(fields []interp.Field, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

func (s *Store) tableExists(tx *sql.Tx, table string) bool {
	var n int
	tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
	return n > 0
}

// convertColumn changes a field's type, converting every value. SQLite can't
// change a column's type in place, so the values are rewritten into a new column.
func convertColumn(tx *sql.Tx, table string, f interp.Field, from string) error {
	rows, err := tx.Query("SELECT " + quote(idField) + ", " + quote(f.Name) + " FROM " + quote(table))
	if err != nil {
		return err
	}
	type change struct {
		id int64
		v  any
	}
	var changes []change
	var bad []string
	for rows.Next() {
		var id int64
		var raw any
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return err
		}
		v, ok := convertValue(decode(from, raw), f.Kind)
		if !ok {
			bad = append(bad, strconv.FormatInt(id, 10))
			continue
		}
		enc, err := encode(f.Kind, v)
		if err != nil {
			bad = append(bad, strconv.FormatInt(id, 10))
			continue
		}
		changes = append(changes, change{id, enc})
	}
	rows.Close()
	if len(bad) > 0 {
		if len(bad) > 10 {
			bad = append(bad[:10], "…")
		}
		return fmt.Errorf("لا يمكن تغيير نوع الحقل «%s» في «%s» من «%s» إلى «%s»: السجلات ذات الأرقام %s لا تتحول. عدّلها أو أعد النوع القديم",
			f.Name, table, kindNames[from], kindNames[f.Kind], strings.Join(bad, "، "))
	}
	tmp := f.Name + "__dad_new"
	stmts := []string{
		"ALTER TABLE " + quote(table) + " ADD COLUMN " + quote(tmp) + " " + sqlType(f.Kind),
	}
	for _, st := range stmts {
		if _, err := tx.Exec(st); err != nil {
			return err
		}
	}
	for _, c := range changes {
		if _, err := tx.Exec("UPDATE "+quote(table)+" SET "+quote(tmp)+" = ? WHERE "+quote(idField)+" = ?", c.v, c.id); err != nil {
			return err
		}
	}
	for _, st := range []string{
		"ALTER TABLE " + quote(table) + " DROP COLUMN " + quote(f.Name),
		"ALTER TABLE " + quote(table) + " RENAME COLUMN " + quote(tmp) + " TO " + quote(f.Name),
	} {
		if _, err := tx.Exec(st); err != nil {
			return err
		}
	}
	return nil
}

// convertValue converts plain data between field kinds, the way «ك» does.
func convertValue(v any, kind string) (any, bool) {
	if v == nil {
		return nil, true
	}
	switch kind {
	case "text":
		switch x := v.(type) {
		case string:
			return x, true
		case int64:
			return strconv.FormatInt(x, 10), true
		case float64:
			return strconv.FormatFloat(x, 'f', -1, 64), true
		case bool:
			if x {
				return "صح", true
			}
			return "خطأ", true
		}
		b, err := json.Marshal(v)
		return string(b), err == nil
	case "int":
		switch x := v.(type) {
		case int64:
			return x, true
		case float64:
			if x == math.Trunc(x) {
				return int64(x), true
			}
		case string:
			n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
			return n, err == nil
		}
	case "float":
		switch x := v.(type) {
		case int64:
			return float64(x), true
		case float64:
			return x, true
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
			return f, err == nil
		}
	case "bool":
		switch x := v.(type) {
		case bool:
			return x, true
		case string:
			switch strings.TrimSpace(x) {
			case "صح", "true":
				return true, true
			case "خطأ", "false":
				return false, true
			}
		}
	case "json":
		return v, true
	}
	return nil, false
}

// ---------------------------------------------------------------- values

// encode turns a field value into what SQLite stores.
func encode(kind string, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch kind {
	case "bool":
		if b, ok := v.(bool); ok {
			if b {
				return int64(1), nil
			}
			return int64(0), nil
		}
	case "int", "float", "text":
		return v, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// decode turns what SQLite returned into plain data.
func decode(kind string, v any) any {
	if b, ok := v.([]byte); ok {
		v = string(b)
	}
	if v == nil {
		return nil
	}
	switch kind {
	case "bool":
		if n, ok := v.(int64); ok {
			return n != 0
		}
	case "float":
		if n, ok := v.(int64); ok {
			return float64(n)
		}
	case "json":
		if str, ok := v.(string); ok {
			dec := json.NewDecoder(strings.NewReader(str))
			dec.UseNumber()
			var out any
			if dec.Decode(&out) == nil {
				return normalize(out)
			}
		}
	}
	return v
}

func normalize(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, _ := x.Float64()
		return f
	case []any:
		for i := range x {
			x[i] = normalize(x[i])
		}
	case map[string]any:
		for k := range x {
			x[k] = normalize(x[k])
		}
	}
	return v
}

// ---------------------------------------------------------------- rows

func (s *Store) schema(table string) ([]interp.Field, error) {
	f, ok := s.fields[table]
	if !ok {
		return nil, fmt.Errorf("الجدول «%s» غير مُعرّف", table)
	}
	return f, nil
}

func (s *Store) selectSQL(table string, fields []interp.Field) string {
	cols := []string{quote(idField)}
	for _, f := range fields {
		cols = append(cols, quote(f.Name))
	}
	return "SELECT " + strings.Join(cols, ", ") + " FROM " + quote(table)
}

func (s *Store) query(table, where string, args ...any) ([]interp.Row, error) {
	fields, err := s.schema(table)
	if err != nil {
		return nil, err
	}
	q := s.selectSQL(table, fields)
	if where != "" {
		q += " WHERE " + where
	}
	rows, err := s.db.Query(q+" ORDER BY "+quote(idField), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []interp.Row{}
	for rows.Next() {
		vals := make([]any, len(fields)+1)
		ptrs := make([]any, len(vals))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := interp.Row{idField: vals[0]}
		for i, f := range fields {
			row[f.Name] = decode(f.Kind, vals[i+1])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// Rows returns every row of a table, oldest first.
func (s *Store) Rows(table string) ([]interp.Row, error) { return s.query(table, "") }

// Insert adds a row and returns its id.
func (s *Store) Insert(table string, row interp.Row) (int64, error) {
	return s.insert(s.db, table, row)
}

type execer interface {
	Exec(string, ...any) (sql.Result, error)
}

func (s *Store) insert(db execer, table string, row interp.Row) (int64, error) {
	fields, err := s.schema(table)
	if err != nil {
		return 0, err
	}
	return insertRow(db, table, fields, row)
}

func insertRow(db execer, table string, fields []interp.Field, row interp.Row) (int64, error) {
	cols := []string{}
	marks := []string{}
	args := []any{}
	if id, ok := row[idField].(int64); ok && id > 0 {
		cols, marks, args = append(cols, quote(idField)), append(marks, "?"), append(args, id)
	}
	for _, f := range fields {
		v, err := encode(f.Kind, row[f.Name])
		if err != nil {
			return 0, err
		}
		cols, marks, args = append(cols, quote(f.Name)), append(marks, "?"), append(args, v)
	}
	res, err := db.Exec("INSERT INTO "+quote(table)+" ("+strings.Join(cols, ", ")+") VALUES ("+strings.Join(marks, ", ")+")", args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Update replaces a row; it reports false when there is no row with that id.
func (s *Store) Update(table string, id int64, row interp.Row) (bool, error) {
	fields, err := s.schema(table)
	if err != nil {
		return false, err
	}
	sets := []string{}
	args := []any{}
	for _, f := range fields {
		v, err := encode(f.Kind, row[f.Name])
		if err != nil {
			return false, err
		}
		sets, args = append(sets, quote(f.Name)+" = ?"), append(args, v)
	}
	if len(sets) == 0 {
		_, found, err := s.Get(table, id)
		return found, err
	}
	res, err := s.db.Exec("UPDATE "+quote(table)+" SET "+strings.Join(sets, ", ")+" WHERE "+quote(idField)+" = ?", append(args, id)...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Delete removes a row; it reports false when there is no row with that id.
func (s *Store) Delete(table string, id int64) (bool, error) {
	if _, err := s.schema(table); err != nil {
		return false, err
	}
	res, err := s.db.Exec("DELETE FROM "+quote(table)+" WHERE "+quote(idField)+" = ?", id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Count returns how many rows a table has.
func (s *Store) Count(table string) (int64, error) {
	if _, err := s.schema(table); err != nil {
		return 0, err
	}
	var n int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM " + quote(table)).Scan(&n)
	return n, err
}

// Get returns one row by id.
func (s *Store) Get(table string, id int64) (interp.Row, bool, error) {
	rows, err := s.query(table, quote(idField)+" = ?", id)
	if err != nil || len(rows) == 0 {
		return nil, false, err
	}
	return rows[0], true, nil
}

// Where returns the rows matching a condition compiled from «حيث».
func (s *Store) Where(table string, c *interp.Cond) ([]interp.Row, error) {
	fields, err := s.schema(table)
	if err != nil {
		return nil, err
	}
	kinds := map[string]string{}
	for _, f := range fields {
		kinds[f.Name] = f.Kind
	}
	var args []any
	where, err := condSQL(c, kinds, &args)
	if err != nil {
		return nil, err
	}
	return s.query(table, where, args...)
}

var sqlOps = map[string]string{"eq": "IS", "ne": "IS NOT", "lt": "<", "le": "<=", "gt": ">", "ge": ">="}

func condSQL(c *interp.Cond, kinds map[string]string, args *[]any) (string, error) {
	switch c.Op {
	case "and", "or":
		l, err := condSQL(c.L, kinds, args)
		if err != nil {
			return "", err
		}
		r, err := condSQL(c.R, kinds, args)
		if err != nil {
			return "", err
		}
		return "(" + l + " " + strings.ToUpper(c.Op) + " " + r + ")", nil
	case "not":
		l, err := condSQL(c.L, kinds, args)
		if err != nil {
			return "", err
		}
		return "(NOT " + l + ")", nil
	}
	kind, ok := kinds[c.Field]
	if !ok {
		return "", fmt.Errorf("لا يوجد حقل «%s»", c.Field)
	}
	if c.Op == "true" {
		return quote(c.Field) + " = 1", nil
	}
	op, ok := sqlOps[c.Op]
	if !ok {
		return "", errors.New("شرط غير معروف: " + c.Op)
	}
	v, err := encode(kind, c.Value)
	if err != nil {
		return "", err
	}
	*args = append(*args, v)
	return quote(c.Field) + " " + op + " ?", nil
}

// ---------------------------------------------------------------- importing JSON data

// ImportJSON reads tables from a data file written by an earlier dad (the JSON
// store). Each table is copied into the database when the program first
// declares it, keeping the ids.
func (s *Store) ImportJSON(path string) error {
	fs, err := interp.OpenFileStore(path)
	if err != nil {
		return err
	}
	tables, err := fs.TableNames()
	if err != nil {
		return err
	}
	s.pending = map[string][]interp.Row{}
	for _, t := range tables {
		rows, err := fs.Rows(t)
		if err != nil {
			return err
		}
		s.pending[t] = rows
	}
	s.importOf = path
	return nil
}

func (s *Store) importRows(tx *sql.Tx, table string, fields []interp.Field) error {
	rows, ok := s.pending[table]
	if !ok {
		return nil
	}
	for _, r := range rows {
		for _, f := range fields {
			if _, ok := r[f.Name]; !ok && f.HasDefault {
				r[f.Name] = f.Default
			}
			if v, ok := convertValue(r[f.Name], f.Kind); ok {
				r[f.Name] = v
			}
		}
		if _, err := insertRow(tx, table, fields, r); err != nil {
			return fmt.Errorf("تعذر نقل البيانات من «%s»: %v", s.importOf, err)
		}
	}
	delete(s.pending, table)
	s.Notes = append(s.Notes, fmt.Sprintf("نُقلت %d سجلات «%s» من %s إلى %s", len(rows), table, s.importOf, s.path))
	return nil
}

// FinishImport renames the old JSON file once all its tables were copied, so
// they are not imported again. It reports the tables still waiting.
func (s *Store) FinishImport() []string {
	if s.importOf == "" {
		return nil
	}
	var waiting []string
	for t := range s.pending {
		waiting = append(waiting, t)
	}
	sort.Strings(waiting)
	if len(waiting) == 0 {
		os.Rename(s.importOf, s.importOf+".imported")
		s.importOf = ""
	}
	return waiting
}
