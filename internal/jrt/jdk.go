package jrt

import (
	"cmp"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// StringBuilder is java.lang.StringBuilder. Indices are byte offsets, like the port's String.length().
type StringBuilder struct{ b []byte }

// NewStringBuilder covers (), (int capacity) and (String).
func NewStringBuilder(init ...any) *StringBuilder {
	sb := &StringBuilder{}
	if len(init) == 1 {
		if s, ok := init[0].(string); ok {
			sb.b = []byte(s)
		}
	}
	return sb
}

func sbText(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case uint16:
		return string(utf16.Decode([]uint16{x}))
	case []uint16:
		return string(utf16.Decode(x))
	}
	return fmt.Sprint(v)
}

func (s *StringBuilder) Append(v any) *StringBuilder {
	s.b = append(s.b, sbText(v)...)
	return s
}

func (s *StringBuilder) Insert(at int32, v any) *StringBuilder {
	s.b = append(s.b[:at], append([]byte(sbText(v)), s.b[at:]...)...)
	return s
}

func (s *StringBuilder) Length() int32         { return int32(len(s.b)) }
func (s *StringBuilder) IsEmpty() bool         { return len(s.b) == 0 }
func (s *StringBuilder) ToString() string      { return string(s.b) }
func (s *StringBuilder) String() string        { return string(s.b) }
func (s *StringBuilder) CharAt(i int32) uint16 { return uint16(s.b[i]) }

func (s *StringBuilder) SetLength(n int32) {
	for int32(len(s.b)) < n {
		s.b = append(s.b, 0)
	}
	s.b = s.b[:n]
}

func (s *StringBuilder) DeleteCharAt(i int32) *StringBuilder {
	s.b = append(s.b[:i], s.b[i+1:]...)
	return s
}

func (s *StringBuilder) Delete(start, end int32) *StringBuilder {
	end = min(end, int32(len(s.b)))
	s.b = append(s.b[:start], s.b[end:]...)
	return s
}

func (s *StringBuilder) IndexOf(sub string) int32 { return int32(strings.Index(string(s.b), sub)) }

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
	run  Runnable
	done chan struct{}
}

func NewThread(r Runnable) *Thread { return &Thread{run: r, done: make(chan struct{})} }

// Run lets a Thread be passed as a Runnable; it runs the body in place.
func (t *Thread) Run() { t.run.Run() }

func ThreadStart(t any) {
	th := t.(*Thread)
	go func() {
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
	return int32(len(utf16.Encode([]rune(s))))
}

// Substring is String.substring(start, end) over UTF-16 indices; end < 0 means to the end.
func Substring(s string, start, end int32) string {
	if isASCII(s) {
		if end < 0 {
			end = int32(len(s))
		}
		return s[start:end]
	}
	u := utf16.Encode([]rune(s))
	if end < 0 {
		end = int32(len(u))
	}
	return string(utf16.Decode(u[start:end]))
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
