package interp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Row is one record of a table, as plain JSON-ready values keyed by field name.
// Every row has "رقم", its id, assigned by the store.
type Row = map[string]any

// Store keeps the rows of every جدول. The interpreter only uses this interface, so
// storage can move to SQLite without changing Ḍād programs.
type Store interface {
	Rows(table string) ([]Row, error)
	Insert(table string, row Row) (int64, error)
	Update(table string, id int64, row Row) (bool, error)
	Delete(table string, id int64) (bool, error)
}

// ---------------------------------------------------------------- in memory

type memTable struct {
	next int64
	rows map[int64]Row
}

// MemoryStore keeps tables in memory. It is used by the playground and tests.
type MemoryStore struct {
	mu     sync.Mutex
	tables map[string]*memTable
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{tables: map[string]*memTable{}} }

func (s *MemoryStore) table(name string) *memTable {
	t := s.tables[name]
	if t == nil {
		t = &memTable{next: 1, rows: map[int64]Row{}}
		s.tables[name] = t
	}
	return t
}

func (s *MemoryStore) Rows(table string) ([]Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.table(table)
	ids := make([]int64, 0, len(t.rows))
	for id := range t.rows {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]Row, 0, len(ids))
	for _, id := range ids {
		out = append(out, copyRow(t.rows[id]))
	}
	return out, nil
}

func (s *MemoryStore) Insert(table string, row Row) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.table(table)
	id := t.next
	t.next++
	r := copyRow(row)
	r["رقم"] = id
	t.rows[id] = r
	return id, nil
}

func (s *MemoryStore) Update(table string, id int64, row Row) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.table(table)
	if _, ok := t.rows[id]; !ok {
		return false, nil
	}
	r := copyRow(row)
	r["رقم"] = id
	t.rows[id] = r
	return true, nil
}

func (s *MemoryStore) Delete(table string, id int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.table(table)
	if _, ok := t.rows[id]; !ok {
		return false, nil
	}
	delete(t.rows, id)
	return true, nil
}

func copyRow(r Row) Row {
	out := make(Row, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}

// ---------------------------------------------------------------- JSON file

// FileStore keeps every table in one JSON file, written after each change.
// It is meant for development; a later version moves the data to SQLite.
type FileStore struct {
	path string
	mem  *MemoryStore
}

type fileFormat struct {
	Note   string                `json:"_"`
	Tables map[string]*fileTable `json:"tables"`
}

type fileTable struct {
	Next int64 `json:"next"`
	Rows []Row `json:"rows"`
}

// OpenFileStore reads the data file at path, if it exists.
func OpenFileStore(path string) (*FileStore, error) {
	s := &FileStore{path: path, mem: NewMemoryStore()}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var f fileFormat
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("ملف البيانات «%s» تالف: %v", path, err)
	}
	for name, t := range f.Tables {
		mt := s.mem.table(name)
		mt.next = t.Next
		for _, r := range t.Rows {
			r = normalizeJSONRow(r)
			id, _ := r["رقم"].(int64)
			if id <= 0 {
				continue
			}
			mt.rows[id] = r
			if id >= mt.next {
				mt.next = id + 1
			}
		}
	}
	return s, nil
}

// Path is the data file's location.
func (s *FileStore) Path() string { return s.path }

func (s *FileStore) Rows(table string) ([]Row, error) { return s.mem.Rows(table) }

func (s *FileStore) Insert(table string, row Row) (int64, error) {
	id, _ := s.mem.Insert(table, row)
	return id, s.save()
}

func (s *FileStore) Update(table string, id int64, row Row) (bool, error) {
	ok, _ := s.mem.Update(table, id, row)
	if !ok {
		return false, nil
	}
	return true, s.save()
}

func (s *FileStore) Delete(table string, id int64) (bool, error) {
	ok, _ := s.mem.Delete(table, id)
	if !ok {
		return false, nil
	}
	return true, s.save()
}

// save writes the whole file through a temporary file, so a crash never leaves it half written.
func (s *FileStore) save() error {
	s.mem.mu.Lock()
	f := fileFormat{
		Note:   "Ḍād data file, written by dad. Edit only while dad isn't running.",
		Tables: map[string]*fileTable{},
	}
	for name, t := range s.mem.tables {
		ft := &fileTable{Next: t.next, Rows: []Row{}}
		ids := make([]int64, 0, len(t.rows))
		for id := range t.rows {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			ft.Rows = append(ft.Rows, t.rows[id])
		}
		f.Tables[name] = ft
	}
	s.mem.mu.Unlock()
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// normalizeJSONRow turns json.Number values into int64 or float64.
func normalizeJSONRow(r Row) Row {
	out := make(Row, len(r))
	for k, v := range r {
		out[k] = normalizeJSON(v)
	}
	return out
}

func normalizeJSON(v any) any {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, _ := x.Float64()
		return f
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalizeJSON(e)
		}
		return out
	case map[string]any:
		return normalizeJSONRow(x)
	}
	return v
}
