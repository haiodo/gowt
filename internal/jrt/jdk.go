package jrt

import (
	"cmp"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

// StringBuilder is java.lang.StringBuilder. It holds UTF-16 units, so indices and Length match Java.
type StringBuilder struct{ u []uint16 }

// NewStringBuilder covers (), (int capacity) and (String).
func NewStringBuilder(init ...any) *StringBuilder {
	sb := &StringBuilder{}
	if len(init) == 1 {
		if s, ok := init[0].(string); ok {
			sb.u = ToChars(s)
		}
	}
	return sb
}

func sbUnits(v any) []uint16 {
	switch x := v.(type) {
	case uint16:
		return []uint16{x}
	case []uint16:
		return x
	case nil:
		return ToChars("null")
	case string:
		return ToChars(x)
	}
	return ToChars(fmt.Sprint(v))
}

// Append covers append(x) and append(char[], offset, len).
func (s *StringBuilder) Append(v any, offsetLen ...int32) *StringBuilder {
	if chars, ok := v.([]uint16); ok && len(offsetLen) == 2 {
		v = chars[offsetLen[0] : offsetLen[0]+offsetLen[1]]
	}
	s.u = append(s.u, sbUnits(v)...)
	return s
}

func (s *StringBuilder) Insert(at int32, v any) *StringBuilder {
	s.u = slices.Insert(s.u, int(at), sbUnits(v)...)
	return s
}

func (s *StringBuilder) Length() int32         { return int32(len(s.u)) }
func (s *StringBuilder) IsEmpty() bool         { return len(s.u) == 0 }
func (s *StringBuilder) ToString() string      { return FromChars(s.u) }
func (s *StringBuilder) String() string        { return FromChars(s.u) }
func (s *StringBuilder) CharAt(i int32) uint16 { return s.u[i] }

func (s *StringBuilder) SetLength(n int32) {
	for int32(len(s.u)) < n {
		s.u = append(s.u, 0)
	}
	s.u = s.u[:n]
}

func (s *StringBuilder) DeleteCharAt(i int32) *StringBuilder {
	s.u = slices.Delete(s.u, int(i), int(i)+1)
	return s
}

func (s *StringBuilder) Delete(start, end int32) *StringBuilder {
	end = min(end, int32(len(s.u)))
	s.u = slices.Delete(s.u, int(start), int(end))
	return s
}

// GetChars is getChars(srcBegin, srcEnd, dst, dstBegin).
func (s *StringBuilder) GetChars(srcBegin, srcEnd int32, dst []uint16, dstBegin int32) {
	copy(dst[dstBegin:], s.u[srcBegin:srcEnd])
}

func (s *StringBuilder) IndexOf(sub string) int32 { return IndexFrom(s.String(), sub, 0) }

// Random is java.util.Random's 48-bit LCG, so a fixed seed gives Java's sequence.
type Random struct{ seed int64 }

const randMask = 1<<48 - 1

func NewRandom(seed ...int64) *Random {
	if len(seed) == 0 {
		seed = []int64{time.Now().UnixNano()}
	}
	return &Random{(seed[0] ^ 0x5DEECE66D) & randMask}
}

func (r *Random) next(bits uint) int32 {
	r.seed = (r.seed*0x5DEECE66D + 0xB) & randMask
	return int32(r.seed >> (48 - bits))
}

// NextBytes fills bytes like java.util.Random.nextBytes.
func (r *Random) NextBytes(bytes []int8) {
	for i := 0; i < len(bytes); {
		for rnd, n := r.NextInt(), min(len(bytes)-i, 4); n > 0; n-- {
			bytes[i] = int8(rnd)
			i++
			rnd >>= 8
		}
	}
}

// NextInt covers nextInt() and nextInt(bound).
func (r *Random) NextInt(bound ...int32) int32 {
	if len(bound) == 0 {
		return r.next(32)
	}
	n := bound[0]
	if n <= 0 {
		panic(NewIllegalArgumentException("bound must be positive"))
	}
	if n&-n == n {
		return int32((int64(n) * int64(r.next(31))) >> 31)
	}
	for {
		bits := r.next(31)
		if val := bits % n; bits-val+(n-1) >= 0 {
			return val
		}
	}
}

// Locale is java.util.Locale reduced to its toString() form; only ENGLISH is needed so far.
type Locale struct{ tag string }

var LocaleENGLISH = &Locale{"en"}
var LocaleGERMAN = &Locale{"de"}

func LocaleDefault() *Locale { return &Locale{LocaleLanguage(nil)} }

func (l *Locale) ToString() string    { return l.tag }
func (l *Locale) GetLanguage() string { return l.tag }
func (l *Locale) GetCountry() string  { return "" }
func (l *Locale) GetVariant() string  { return "" }

// Thread is java.lang.Thread for `new Thread(runnable)` + start/join; the port's `Thread` type
// itself stays a bare any (Display.thread), see Manual.
type Thread struct {
	run     Runnable
	done    chan struct{}
	started atomic.Bool
}

func NewThread(r Runnable) *Thread { return &Thread{run: r, done: make(chan struct{})} }

// Run lets a Thread be passed as a Runnable; it runs the body in place.
func (t *Thread) Run() { t.run.Run() }

func ThreadStart(t any) {
	th, ok := t.(*Thread)
	if !ok {
		// A class extending Thread is translated to a struct with Run, not embedding Thread.
		th = NewThread(t.(Runnable))
	}
	th.started.Store(true)
	go func() {
		// A thread may own a Display, and the toolkit checks the OS thread.
		runtime.LockOSThread()
		defer close(th.done)
		// An uncaught exception ends only its own Java thread.
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "Exception in thread:", r)
			}
		}()
		th.run.Run()
	}()
}

func ThreadJoin(t any) { <-t.(*Thread).done }

// ThreadIsAlive is Thread.isAlive: started and not yet finished.
func ThreadIsAlive(t any) bool {
	th := t.(*Thread)
	select {
	case <-th.done:
		return false
	default:
		return th.started.Load()
	}
}

// ParseDouble is Double.parseDouble.
func ParseDouble(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		panic(&NumberFormatException{Input: s})
	}
	return f
}

// ParseFloat is Float.parseFloat.
func ParseFloat(s string) float32 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
	if err != nil {
		panic(&NumberFormatException{Input: s})
	}
	return float32(f)
}

// HashCodeOf is Object.hashCode() on a receiver whose static type has no Go method for it.
func HashCodeOf(x any) int32 {
	if h, ok := x.(interface{ HashCode() int32 }); ok {
		return h.HashCode()
	}
	return IdentityHashCode(x)
}

// ComparingInt is Comparator.comparingInt.
func ComparingInt[T any](key func(T) int32) func(T, T) int32 {
	return func(a, b T) int32 { return int32(cmp.Compare(key(a), key(b))) }
}

// ThenComparing is Comparator.thenComparing(Comparator) or thenComparing(Function): next is a
// func(T, T) int32 or a key extractor returning an ordered value.
func ThenComparing[T any](first func(T, T) int32, next any) func(T, T) int32 {
	second, ok := next.(func(T, T) int32)
	if !ok {
		key := reflect.ValueOf(next)
		second = func(a, b T) int32 {
			ka := key.Call([]reflect.Value{reflect.ValueOf(a)})[0].Interface()
			kb := key.Call([]reflect.Value{reflect.ValueOf(b)})[0].Interface()
			return compareOrdered(ka, kb)
		}
	}
	return func(a, b T) int32 {
		if c := first(a, b); c != 0 {
			return c
		}
		return second(a, b)
	}
}

func compareOrdered(a, b any) int32 {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	switch {
	case va.CanInt():
		return int32(cmp.Compare(va.Int(), vb.Int()))
	case va.CanFloat():
		return int32(cmp.Compare(va.Float(), vb.Float()))
	}
	return int32(strings.Compare(va.String(), vb.String()))
}

// NullString stands for a Java null String argument written literally in a test; the ported
// null-argument guards compare against it, since an ordinary "" is a legal String.
const NullString = "\x00null"

// NullToEmpty turns the NullString sentinel back into "" where a null String result is used as a plain value.
func NullToEmpty(s string) string {
	if s == NullString {
		return ""
	}
	return s
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// StringLength is String.length(): UTF-16 code units, so a length sizes a getChars buffer.
func StringLength(s string) int32 {
	if isASCII(s) {
		return int32(len(s))
	}
	return int32(len(ToChars(s)))
}

// Ceiling: a pair cut by Substring and joined with Go + is two 3-byte units, not the 4-byte rune. Length,
// index, hash and compareTo are right; ==, map keys and HasPrefix see different bytes. Upgrade: a
// jrt.Concat in the generator that re-encodes the joined units.
// ToChars is String.toCharArray(): the UTF-16 units of s. A lone surrogate is kept as WTF-8
// (ED A0..BF xx), which Substring produces when it cuts an emoji pair; two such halves
// concatenated with + read back as the pair.
func ToChars(s string) []uint16 {
	u := make([]uint16, 0, len(s))
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			u = append(u, uint16(c))
			i++
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 && c == 0xED && i+2 < len(s) && s[i+1] >= 0xA0 && s[i+1] <= 0xBF && s[i+2]&0xC0 == 0x80 {
			u = append(u, uint16(0xD000|rune(s[i+1]&0x3F)<<6|rune(s[i+2]&0x3F)))
			i += 3
			continue
		}
		if r >= 0x10000 {
			r1, r2 := utf16.EncodeRune(r)
			u = append(u, uint16(r1), uint16(r2))
		} else {
			u = append(u, uint16(r))
		}
		i += n
	}
	return u
}

type charsEntry struct {
	data  *byte
	n     int
	chars []uint16 // nil: s is ASCII, index bytes directly
}

var charsCache atomic.Pointer[charsEntry]

// Ceiling: one cached string, strings over 16 MiB are converted on every call. Upgrade: a small per-string table.
const charsCacheMax = 16 << 20

// CharAt is String.charAt(i): the i-th UTF-16 unit of s, amortized O(1) in a loop over one string.
// Out of range panics like the ToChars(s)[i] it replaces.
func CharAt(s string, i int32) uint16 {
	if len(s) <= 16 {
		if isASCII(s) {
			return uint16(s[i])
		}
		return ToChars(s)[i]
	}
	if len(s) > charsCacheMax {
		return ToChars(s)[i]
	}
	d := unsafe.StringData(s)
	e := charsCache.Load()
	if e == nil || e.data != d || e.n != len(s) {
		e = &charsEntry{data: d, n: len(s)}
		if !isASCII(s) {
			e.chars = ToChars(s)
		}
		charsCache.Store(e)
	}
	if e.chars == nil {
		return uint16(s[i])
	}
	return e.chars[i]
}

// FromChars is new String(char[]): a valid pair becomes one rune, a lone surrogate WTF-8 (see ToChars).
func FromChars(u []uint16) string {
	b := make([]byte, 0, len(u))
	for i := 0; i < len(u); i++ {
		r := rune(u[i])
		switch {
		case utf16.IsSurrogate(r) && r < 0xDC00 && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] < 0xE000:
			b = utf8.AppendRune(b, utf16.DecodeRune(r, rune(u[i+1])))
			i++
		case utf16.IsSurrogate(r):
			b = append(b, 0xED, byte(0x80|r>>6&0x3F), byte(0x80|r&0x3F))
		default:
			b = utf8.AppendRune(b, r)
		}
	}
	return string(b)
}

// ValidUTF8 is for text handed to the OS: surrogate pairs held as two WTF-8 halves are joined,
// a lone surrogate becomes U+FFFD, since Cocoa and GTK reject invalid UTF-8.
func ValidUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	u := ToChars(s)
	for i := 0; i < len(u); i++ {
		if u[i] >= 0xD800 && u[i] < 0xDC00 && i+1 < len(u) && u[i+1] >= 0xDC00 && u[i+1] < 0xE000 {
			i++
		} else if utf16.IsSurrogate(rune(u[i])) {
			u[i] = 0xFFFD
		}
	}
	return FromChars(u)
}

// Substring is String.substring(start, end) over UTF-16 indices; end < 0 means to the end.
func Substring(s string, start, end int32) string {
	if isASCII(s) {
		if end < 0 {
			end = int32(len(s))
		}
		return s[start:end]
	}
	u := ToChars(s)
	if end < 0 {
		end = int32(len(u))
	}
	return FromChars(u[start:end])
}

// Optional is java.util.Optional over any: erased generics, so a caller casts Get's result.
type Optional struct {
	v  any
	ok bool
}

func OptionalOf(v any) *Optional { return &Optional{v, true} }

// OptionalOfNullable: a nil interface or typed nil pointer is empty.
func OptionalOfNullable(v any) *Optional {
	if IsNil(v) {
		return &Optional{}
	}
	return &Optional{v, true}
}

func OptionalEmpty() *Optional { return &Optional{} }

func (o *Optional) IsPresent() bool { return o.ok }
func (o *Optional) IsEmpty() bool   { return !o.ok }

func (o *Optional) Get() any {
	if !o.ok {
		panic(&NoSuchElementException{})
	}
	return o.v
}

func (o *Optional) OrElse(other any) any {
	if o.ok {
		return o.v
	}
	return other
}

// NoSuchElementException is java.util.NoSuchElementException.
type NoSuchElementException struct{}

func (e *NoSuchElementException) Error() string { return "No value present" }

// mapWTF8 applies f to the valid chunks of s; a lone surrogate (WTF-8) passes through, since
// strings.ToUpper and strings.ToLower turn it into U+FFFD.
func mapWTF8(s string, f func(string) string) string {
	if utf8.ValidString(s) {
		return f(s)
	}
	var b strings.Builder
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == 0xED && i+2 < len(s) && s[i+1] >= 0xA0 && s[i+1] <= 0xBF && s[i+2]&0xC0 == 0x80 {
			b.WriteString(f(s[start:i]))
			b.WriteString(s[i : i+3])
			i += 2
			start = i + 1
		}
	}
	b.WriteString(f(s[start:]))
	return b.String()
}

// ToUpperCase is String.toUpperCase().
func ToUpperCase(s string) string { return mapWTF8(s, strings.ToUpper) }

// ToLowerCase is String.toLowerCase().
func ToLowerCase(s string) string { return mapWTF8(s, strings.ToLower) }
