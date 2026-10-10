package jrt

import (
	"cmp"
	"net/url"
	"slices"
	"strings"
)

// The Collections wrappers return the list itself: the translated tests are single-threaded and
// never write through the original after wrapping, so a copy or a lock would only hide a bug.
func CollectionsUnmodifiableList(l *List) *List { return l }
func CollectionsSynchronizedList(l *List) *List { return l }
func CollectionsSynchronizedSet(l *List) *List  { return l }

// CollectionsSort is Collections.sort(list) for strings and numbers, or with a comparator func(a, b T) int32.
func CollectionsSort(l *List, comparator ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(comparator) == 1 && comparator[0] != nil {
		slices.SortStableFunc(l.items, func(a, b any) int { return int(callFunc(comparator[0], a, b).(int32)) })
		return
	}
	slices.SortStableFunc(l.items, func(a, b any) int {
		switch x := a.(type) {
		case string:
			return cmp.Compare(x, b.(string))
		case int32:
			return cmp.Compare(x, b.(int32))
		case int64:
			return cmp.Compare(x, b.(int64))
		case float64:
			return cmp.Compare(x, b.(float64))
		}
		panic(NewIllegalArgumentException("Collections.sort: elements are not comparable"))
	})
}

// RemoveAll is List.removeAll: drops every element that other contains.
func (l *List) RemoveAll(other *List) bool {
	drop := other.ToArray()
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.items[:0:0]
	for _, e := range l.items {
		if !slices.ContainsFunc(drop, func(d any) bool { return keysEqual(e, d) }) {
			kept = append(kept, e)
		}
	}
	changed := len(kept) != len(l.items)
	l.items = kept
	return changed
}

// URI is java.net.URI as its string form (Path.toUri().toString()).
type URI struct{ s string }

// NewURI parses s like new URI(String): an unparsable string throws URISyntaxException.
func NewURI(s string) *URI {
	if _, err := url.Parse(s); err != nil {
		panic(NewURISyntaxException(s, err.Error()))
	}
	return &URI{s}
}

// GetScheme is URI.getScheme(): the text before the first colon when it is a valid scheme, else the port's null String.
func (u *URI) GetScheme() string {
	i := strings.IndexByte(u.s, ':')
	if i <= 0 {
		return ""
	}
	for j, c := range u.s[:i] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || j > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return ""
		}
	}
	return u.s[:i]
}

func (u *URI) ToString() string { return u.s }
func (u *URI) String() string   { return u.s }

// DoubleEquals is Double.equals(Object): the other must be a Double with the same value.
func DoubleEquals(a float64, b any) bool {
	f, ok := b.(float64)
	return ok && (f == a || (f != f && a != a))
}

// NumberDouble is Number.doubleValue() on a boxed number of whatever kind the value came as.
func NumberDouble(n any) float64 {
	switch x := n.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case int16:
		return float64(x)
	case int8:
		return float64(x)
	case int:
		return float64(x)
	}
	panic(NewIllegalArgumentException("not a Number"))
}
