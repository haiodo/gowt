package jrt

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"
)

// File is java.io.File over a path.
type File struct{ path string }

// NewFile covers File(path) and File(parent, child); a parent may be a *File or a string.
func NewFile(parts ...any) *File {
	ps := make([]string, len(parts))
	for i, p := range parts {
		ps[i] = pathOf(p)
	}
	return &File{filepath.Join(ps...)}
}

func pathOf(f any) string {
	switch v := f.(type) {
	case string:
		return v
	case *File:
		return v.path
	}
	panic(NewIllegalArgumentException("not a file name"))
}

func (f *File) GetPath() string { return f.path }
func (f *File) Exists() bool    { _, err := os.Stat(f.path); return err == nil }
func (f *File) IsFile() bool    { st, err := os.Stat(f.path); return err == nil && st.Mode().IsRegular() }
func (f *File) IsDirectory() bool {
	st, err := os.Stat(f.path)
	return err == nil && st.IsDir()
}
func (f *File) Mkdirs() bool { return os.MkdirAll(f.path, 0o755) == nil }

// Reader stands for BufferedReader, InputStreamReader and FileReader alike.
type Reader struct{ r io.Reader }

func NewBufferedReader(r any) *Reader { return r.(*Reader) }

func NewInputStreamReader(in InputStream) *Reader {
	if in == nil {
		panic(NewIOException("no stream"))
	}
	return &Reader{strings.NewReader(string(toBytes(ReadAllBytes(in))))}
}

func NewFileReader(file any) *Reader {
	f, err := os.Open(pathOf(file))
	if err != nil {
		panic(NewIOException(err.Error()))
	}
	return &Reader{f}
}

func toBytes(b []int8) []byte {
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = byte(v)
	}
	return out
}

// Lines is BufferedReader.lines(): a stream is an eager *List here.
func (r *Reader) Lines() *List {
	l := NewList()
	sc := bufio.NewScanner(r.r)
	for sc.Scan() {
		l.Add(sc.Text())
	}
	return l
}

// ReadLine is BufferedReader.readLine(): the next line, "" at the end (Java's null).
func (r *Reader) ReadLine() string {
	var sb []byte
	one := make([]byte, 1)
	for {
		n, err := r.r.Read(one)
		if n == 0 || err != nil || one[0] == '\n' {
			return string(sb)
		}
		sb = append(sb, one[0])
	}
}

func (r *Reader) Close() {
	if c, ok := r.r.(io.Closer); ok {
		c.Close()
	}
}

// Collector is Collectors.joining(sep), toList() or toSet() (kind "list"/"set"; sets keep insertion order).
type Collector struct {
	sep, kind string
	key       any
}

func CollectorsJoining(sep string) *Collector { return &Collector{sep: sep} }

// CollectorsSummingInt is Collectors.summingInt(ToIntFunction).
func CollectorsSummingInt(key any) *Collector { return &Collector{kind: "sumint", key: key} }
func CollectorsToList() *Collector            { return &Collector{kind: "list"} }
func CollectorsToSet() *Collector             { return &Collector{kind: "set"} }

// Properties (a Map): key=value lines in, sorted key=value lines out.
func (m *Map) Load(in InputStream) {
	for _, line := range strings.Split(string(toBytes(ReadAllBytes(in))), "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(line, "#") {
			m.Put(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
}

func (m *Map) GetProperty(key string) string {
	if v, ok := m.Get(key).(string); ok {
		return v
	}
	return ""
}

func (m *Map) Store(out OutputStream, comments string) {
	var lines []string
	for _, e := range m.EntrySet().ToArray() {
		lines = append(lines, e.(*MapEntry).key.(string)+"="+e.(*MapEntry).value.(string))
	}
	sort.Strings(lines)
	b := []byte(strings.Join(lines, "\n") + "\n")
	s := make([]int8, len(b))
	for i, v := range b {
		s[i] = int8(v)
	}
	out.WriteRange(s, 0, int32(len(s)))
}

func (m *Map) KeySet() *List {
	var keys []any
	for _, e := range m.EntrySet().ToArray() {
		keys = append(keys, e.(*MapEntry).key)
	}
	return ListOf(keys...)
}

func (sb *StringBuilder) Replace(start, end int32, s string) *StringBuilder {
	sb.b = append(append(append([]byte{}, sb.b[:start]...), s...), sb.b[min(int(end), len(sb.b)):]...)
	return sb
}

func LocaleForLanguageTag(tag string) *Locale { return &Locale{tag} }

// Get waits for the future and returns nothing: the ported futures carry no value (GTK 4 only).
func (f *CompletableFuture) Get() any {
	for !f.IsDone() {
		Yield()
	}
	return nil
}

// The reader classes share one Go type; each constructor keeps its Java name.
type (
	BufferedReader    = Reader
	InputStreamReader = Reader
	FileReader        = Reader
)

// java.io.File.separator / separatorChar.
const (
	FileSeparator            = "/"
	FileSeparatorChar uint16 = '/'
)

// Pattern and Matcher are java.util.regex over Go's RE2 (no backreferences or lookaround).
type Pattern struct{ re *regexp.Regexp }

// \p{Punct} (Java) is [:punct:] inside a Go class.
func PatternCompile(regex string) *Pattern {
	return &Pattern{regexp.MustCompile(strings.ReplaceAll(regex, `\p{Punct}`, `[:punct:]`))}
}

type Matcher struct {
	p    *Pattern
	s    string
	from int
	m    []string
	lo   int // byte offsets of the last match
	hi   int
}

func (p *Pattern) Matcher(s string) *Matcher { return &Matcher{p: p, s: s} }

func (m *Matcher) Find() bool {
	loc := m.p.re.FindStringSubmatchIndex(m.s[m.from:])
	if loc == nil {
		return false
	}
	m.m = m.m[:0]
	for i := 0; i < len(loc); i += 2 {
		if loc[i] < 0 {
			m.m = append(m.m, "")
		} else {
			m.m = append(m.m, m.s[m.from+loc[i]:m.from+loc[i+1]])
		}
	}
	m.lo, m.hi = m.from+loc[0], m.from+loc[1]
	m.from += max(loc[1], 1)
	return true
}

// Start and End are UTF-16 offsets of the last match, as in Java.
func (m *Matcher) Start() int32 { return int32(len(utf16.Encode([]rune(m.s[:m.lo])))) }
func (m *Matcher) End() int32   { return int32(len(utf16.Encode([]rune(m.s[:m.hi])))) }

func (m *Matcher) Group(i ...int32) string {
	if len(i) == 0 {
		return m.m[0]
	}
	return m.m[i[0]]
}

// ArraysFill is Arrays.fill(a, v); ArraysFillRange fills [from, to).
func ArraysFill[T any](a []T, v T) { ArraysFillRange(a, 0, int32(len(a)), v) }

func ArraysFillRange[T any](a []T, from, to int32, v T) {
	for i := from; i < to; i++ {
		a[i] = v
	}
}

// ReplaceFirst and ReplaceAll are String.replaceFirst/replaceAll ($1 group references as in Java).
func ReplaceFirst(s, regex, repl string) string {
	re := regexp.MustCompile(regex)
	loc := re.FindStringSubmatchIndex(s)
	if loc == nil {
		return s
	}
	return s[:loc[0]] + string(re.ExpandString(nil, javaRepl(repl), s, loc)) + s[loc[1]:]
}

func ReplaceAll(s, regex, repl string) string {
	return regexp.MustCompile(regex).ReplaceAllString(s, javaRepl(repl))
}

// javaRepl turns Java's $1 into Go's ${1}.
func javaRepl(r string) string { return regexp.MustCompile(`\$(\d+)`).ReplaceAllString(r, "$${$1}") }

// LinkedList / deque members on List.
func (l *List) GetFirst() any { return l.Get(0) }
func (l *List) GetLast() any  { return l.Get(l.Size() - 1) }

func (l *List) RemoveFirst() any {
	l.mu.Lock()
	defer l.mu.Unlock()
	v := l.items[0]
	l.items = l.items[1:]
	return v
}

func (l *List) IndexOf(v any) int32 {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, e := range l.items {
		if keysEqual(e, v) {
			return int32(i)
		}
	}
	return -1
}
