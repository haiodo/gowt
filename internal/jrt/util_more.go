package jrt

import (
	"os"
	"path/filepath"
	"reflect"
)

// entries lists the map's pairs in EntrySet order.
func (m *Map) entries() []*MapEntry {
	var out []*MapEntry
	for _, e := range m.EntrySet().ToArray() {
		out = append(out, e.(*MapEntry))
	}
	return out
}

// Values is Map.values(): a snapshot in EntrySet order.
func (m *Map) Values() *List {
	l := NewList()
	for _, e := range m.entries() {
		l.Add(e.value)
	}
	return l
}

// ComputeIfAbsent is Map.computeIfAbsent: fn is a Go func of the key returning the value.
func (m *Map) ComputeIfAbsent(k any, fn any) any {
	if v := m.Get(k); !IsNil(v) {
		return v
	}
	v := callFunc(fn, k)
	if !IsNil(v) {
		m.Put(k, v)
	}
	return v
}

// callFunc calls a Go func value with erased arguments and returns its first result (nil if none).
func callFunc(fn any, args ...any) any {
	f := reflect.ValueOf(fn)
	in := make([]reflect.Value, len(args))
	for i, a := range args {
		in[i] = reflect.Zero(f.Type().In(i))
		if a != nil {
			in[i] = reflect.ValueOf(a)
		}
	}
	out := f.Call(in)
	if len(out) == 0 {
		return nil
	}
	return out[0].Interface()
}

// Iterator is java.util.Iterator over a List snapshot; Remove deletes the last returned element from the list.
type Iterator struct {
	list  *List
	items []any
	next  int
}

func (l *List) Iterator() *Iterator { return &Iterator{list: l, items: l.ToArray()} }

func (it *Iterator) HasNext() bool { return it.next < len(it.items) }

func (it *Iterator) Next() any {
	if !it.HasNext() {
		panic(&NoSuchElementException{})
	}
	it.next++
	return it.items[it.next-1]
}

func (it *Iterator) Remove() { it.list.Remove(it.items[it.next-1]) }

// ListCopyOf is List.copyOf.
func ListCopyOf(l *List) *List { return &List{items: l.ToArray()} }

// OrElseGet is Optional.orElseGet: supplier is a Go func returning the value.
func (o *Optional) OrElseGet(supplier any) any {
	if o.ok {
		return o.v
	}
	return callFunc(supplier)
}

// Equals is Path.equals.
func (p *Path) Equals(other *Path) bool { return other != nil && p.s == other.s }

// Stream is List.stream() for the one chain Region.toString builds; the stages are not run.
func (l *List) Stream() *List { return l }

// ListFiles is File.listFiles(): the directory's entries, nil if it cannot be read.
func (f *File) ListFiles() []*File {
	entries, err := os.ReadDir(f.path)
	if err != nil {
		return nil
	}
	out := make([]*File, len(entries))
	for i, e := range entries {
		out[i] = &File{filepath.Join(f.path, e.Name())}
	}
	return out
}

// Delete is File.delete().
func (f *File) Delete() bool { return os.Remove(f.path) == nil }
