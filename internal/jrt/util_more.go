package jrt

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
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

// upcast walks a translated class pointer down its field-0 superclass embedding until it fits t (Java passes
// a subclass where the func takes a superclass); v is returned unchanged when nothing fits.
func upcast(v reflect.Value, t reflect.Type) reflect.Value {
	for w := v; w.Kind() == reflect.Ptr && w.Elem().Kind() == reflect.Struct && w.Elem().NumField() > 0; {
		if w.Type().AssignableTo(t) {
			return w
		}
		f := w.Elem().Field(0)
		if !f.CanAddr() || !w.Type().Elem().Field(0).Anonymous {
			break
		}
		w = f.Addr()
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
			in[i] = upcast(reflect.ValueOf(a), f.Type().In(i))
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

// A Stream is the *List itself; every stage is eager and returns a new list.
func (l *List) Stream() *List { return l }

func (l *List) Map(fn any) *List {
	out := NewList()
	for _, v := range l.ToArray() {
		out.Add(callFunc(fn, v))
	}
	return out
}

func (l *List) Filter(pred any) *List {
	out := NewList()
	for _, v := range l.ToArray() {
		if callFunc(pred, v).(bool) {
			out.Add(v)
		}
	}
	return out
}

func (l *List) Skip(n int64) *List {
	items := l.ToArray()
	if int(n) > len(items) {
		n = int64(len(items))
	}
	return &List{items: items[n:]}
}

func (l *List) ToList() *List { return &List{items: l.ToArray()} }

// Collect covers joining, toList and toSet; the caller asserts the Java result type.
func (l *List) Collect(c *Collector) any {
	if c.kind == "" {
		parts := make([]string, 0, l.Size())
		for _, v := range l.ToArray() {
			parts = append(parts, fmt.Sprint(v))
		}
		return strings.Join(parts, c.sep)
	}
	out := NewList()
	for _, v := range l.ToArray() {
		if c.kind == "list" || !out.Contains(v) {
			out.Add(v)
		}
	}
	return out
}

// TreeSet is a sorted set of int32 or string elements (Image.ImageHandleManager's zoom set).
type TreeSet struct{ *List }

func lessAny(a, b any) bool {
	if x, ok := a.(int32); ok {
		return x < b.(int32)
	}
	return fmt.Sprint(a) < fmt.Sprint(b)
}

func NewTreeSet(from ...any) *TreeSet {
	t := &TreeSet{NewList()}
	if len(from) == 1 {
		if src, ok := from[0].(*List); ok {
			t.AddAll(src)
		}
	}
	return t
}

func (t *TreeSet) Add(a ...any) bool {
	if t.Contains(a[0]) {
		return false
	}
	t.List.Add(a[0])
	sort.SliceStable(t.items, func(i, j int) bool { return lessAny(t.items[i], t.items[j]) })
	return true
}

func (t *TreeSet) AddAll(other *List) bool {
	for _, v := range other.ToArray() {
		t.Add(v)
	}
	return true
}

// Higher is the least element strictly greater than v, nil if none.
func (t *TreeSet) Higher(v any) any {
	for _, e := range t.items {
		if lessAny(v, e) {
			return e
		}
	}
	return nil
}

// Lower is the greatest element strictly less than v, nil if none.
func (t *TreeSet) Lower(v any) any {
	var r any
	for _, e := range t.items {
		if lessAny(e, v) {
			r = e
		}
	}
	return r
}

// ParseIntRadix is Integer.parseInt(s, radix).
func ParseIntRadix(s string, radix int32) int32 {
	n, err := strconv.ParseInt(s, int(radix), 32)
	if err != nil {
		panic(&NumberFormatException{Input: s})
	}
	return int32(n)
}

// Directionality is Character.getDirectionality for the blocks SWT's BidiUtil tells apart:
// Hebrew 1, Arabic 2, other letters 0, digits 3, space 12, else 13 (other neutral).
// Ceiling: no full UCD bidi class table; swap in golang.org/x/text/unicode/bidi if needed.
func Directionality(c rune) int8 {
	switch {
	case c >= 0x0590 && c <= 0x05FF, c >= 0xFB1D && c <= 0xFB4F:
		return 1
	case c >= 0x0600 && c <= 0x07BF, c >= 0x0750 && c <= 0x077F, c >= 0xFB50 && c <= 0xFDFF, c >= 0xFE70 && c <= 0xFEFF:
		return 2
	case unicode.IsDigit(c):
		return 3
	case unicode.IsLetter(c):
		return 0
	case c == ' ':
		return 12
	}
	return 13
}

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

// IsAssignableFrom is Class.isAssignableFrom for the reflect types the translator uses as Class values.
func IsAssignableFrom(to, from reflect.Type) bool {
	if to == nil || from == nil {
		return false
	}
	return from == to || to.Kind() == reflect.Interface && from.Implements(to) || from.AssignableTo(to)
}

// Compute is Map.compute(key, (k, old) -> new): a nil result removes the key.
func (m *Map) Compute(k any, fn any) any {
	v := callFunc(fn, k, m.Get(k))
	if IsNil(v) {
		m.Remove(k)
		return nil
	}
	m.Put(k, v)
	return v
}

// GetAsInt is OptionalInt.getAsInt (an OptionalInt is an Optional of an int32).
func (o *Optional) GetAsInt() int32 { return Cast[int32](o.Get()) }

func (l *List) AnyMatch(pred any) bool {
	for _, v := range l.ToArray() {
		if callFunc(pred, v).(bool) {
			return true
		}
	}
	return false
}

// StringCompareTo is String.compareTo: UTF-16 code units, the difference of the first mismatch or of the lengths.
func StringCompareTo(a, b string) int32 {
	x, y := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			return int32(x[i]) - int32(y[i])
		}
	}
	return int32(len(x) - len(y))
}

// IsHighSurrogate is Character.isHighSurrogate.
func IsHighSurrogate(c rune) bool { return c >= 0xD800 && c <= 0xDBFF }
