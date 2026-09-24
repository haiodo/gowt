package jrt

import "sync"

// hasher is what Java's HashMap uses for key identity: translated classes carry
// HashCode/Equals (cocoa's id compares by native handle, not by Go pointer).
type hasher interface {
	HashCode() int32
	Equals(other any) bool
}

type mapEntry struct {
	key, value any
}

// Map stands in for java.util.Map/HashMap: hashCode buckets with an equals scan, or plain Go
// equality for keys without HashCode/Equals.
type Map struct {
	buckets map[any][]mapEntry
}

func NewMap() *Map { return &Map{buckets: map[any][]mapEntry{}} }

func bucketKey(k any) any {
	if h, ok := k.(hasher); ok {
		return h.HashCode()
	}
	return k
}

func keysEqual(a, b any) bool {
	if h, ok := a.(hasher); ok {
		return h.Equals(b)
	}
	return a == b
}

func (m *Map) find(k any) (any, int) {
	bk := bucketKey(k)
	for i, e := range m.buckets[bk] {
		if keysEqual(e.key, k) {
			return bk, i
		}
	}
	return bk, -1
}

func (m *Map) Get(k any) any {
	bk, i := m.find(k)
	if i < 0 {
		return nil
	}
	return m.buckets[bk][i].value
}

func (m *Map) ContainsKey(k any) bool {
	_, i := m.find(k)
	return i >= 0
}

func (m *Map) Put(k, v any) any {
	bk, i := m.find(k)
	if i < 0 {
		m.buckets[bk] = append(m.buckets[bk], mapEntry{k, v})
		return nil
	}
	old := m.buckets[bk][i].value
	m.buckets[bk][i].value = v
	return old
}

func (m *Map) Remove(k any) any {
	bk, i := m.find(k)
	if i < 0 {
		return nil
	}
	old := m.buckets[bk][i].value
	m.buckets[bk] = append(m.buckets[bk][:i], m.buckets[bk][i+1:]...)
	if len(m.buckets[bk]) == 0 {
		delete(m.buckets, bk)
	}
	return old
}

func (m *Map) Size() int32 {
	n := 0
	for _, b := range m.buckets {
		n += len(b)
	}
	return int32(n)
}

func (m *Map) IsEmpty() bool { return len(m.buckets) == 0 }

func (m *Map) Clear() { m.buckets = map[any][]mapEntry{} }

// List stands in for java.util.List/ArrayList and ConcurrentLinkedQueue. The mutex covers the
// queue use (Synchronizer.asyncExec may be called off the UI goroutine).
type List struct {
	mu    sync.Mutex
	items []any
}

func NewList() *List { return &List{} }

func (l *List) Add(v any) bool {
	l.mu.Lock()
	l.items = append(l.items, v)
	l.mu.Unlock()
	return true
}

func (l *List) Get(i int32) any {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.items[i]
}

func (l *List) Peek() any {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.items) == 0 {
		return nil
	}
	return l.items[0]
}

func (l *List) Poll() any {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.items) == 0 {
		return nil
	}
	v := l.items[0]
	l.items = l.items[1:]
	return v
}

func (l *List) Size() int32 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int32(len(l.items))
}

func (l *List) IsEmpty() bool { return l.Size() == 0 }

func (l *List) Clear() {
	l.mu.Lock()
	l.items = nil
	l.mu.Unlock()
}

func (l *List) AddAll(other *List) bool {
	for _, v := range other.items {
		l.Add(v)
	}
	return true
}

// Remove is java.util.List.remove(Object): drops the first equal element.
func (l *List) Remove(v any) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, e := range l.items {
		if keysEqual(e, v) {
			l.items = append(l.items[:i], l.items[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveIf takes Java's Predicate as a func(any) bool; typed any because a translated
// method-reference argument has no Go type of its own yet.
func (l *List) RemoveIf(predicate any) bool {
	pred := predicate.(func(any) bool)
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.items[:0]
	for _, v := range l.items {
		if !pred(v) {
			kept = append(kept, v)
		}
	}
	removed := len(kept) != len(l.items)
	l.items = kept
	return removed
}
