package junit

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// JUnit's optional trailing message: a string or a Supplier<String> (func() string).
func message(msg []any) string {
	if len(msg) == 0 {
		return ""
	}
	if f, ok := msg[0].(func() string); ok {
		return f()
	}
	return fmt.Sprint(msg[0])
}

func failf(msg []any, format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	if m := message(msg); m != "" {
		s = m + " ==> " + s
	}
	panic(&AssertionFailed{Message: s})
}

func Fail(msg ...any) { panic(&AssertionFailed{Message: message(msg)}) }

func AssertTrue(cond bool, msg ...any) {
	if !cond {
		failf(msg, "expected: <true> but was: <false>")
	}
}

func AssertFalse(cond bool, msg ...any) {
	if cond {
		failf(msg, "expected: <false> but was: <true>")
	}
}

// IsNull is Java's null: a nil pointer/slice/map/func/interface, and "" - the port's stand-in
// for a null String (Go's string has no nil).
func IsNull(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func, reflect.Interface, reflect.Chan:
		return r.IsNil()
	case reflect.String:
		return r.Len() == 0
	}
	return false
}

func AssertNull(v any, msg ...any) {
	if !IsNull(v) {
		failf(msg, "expected: <null> but was: <%s>", show(v))
	}
}

func AssertNotNull(v any, msg ...any) {
	if IsNull(v) {
		failf(msg, "expected: not <null>")
	}
}

// same is Java's ==: a translated object is reached through different Go pointer types (an
// upcast is the address of the embedded superclass field, at offset 0), so compare addresses.
func same(a, b any) bool {
	if IsNull(a) || IsNull(b) {
		return IsNull(a) && IsNull(b)
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Kind() == reflect.Pointer && rb.Kind() == reflect.Pointer {
		return ra.Pointer() == rb.Pointer()
	}
	return ra.Type() == rb.Type() && ra.Comparable() && a == b
}

func AssertSame(expected, actual any, msg ...any) {
	if !same(expected, actual) {
		failf(msg, "expected: <%s> but was: <%s>", show(expected), show(actual))
	}
}

func AssertNotSame(expected, actual any, msg ...any) {
	if same(expected, actual) {
		failf(msg, "expected: not same but was: <%s>", show(actual))
	}
}

// Equal is Object.equals: the translated equals(Object) (Go Equals(any) bool) when there is one,
// identity for other objects, value equality otherwise.
func Equal(a, b any) bool {
	if IsNull(a) || IsNull(b) {
		return IsNull(a) && IsNull(b)
	}
	if e, ok := a.(interface{ Equals(any) bool }); ok {
		return e.Equals(b)
	}
	return same(a, b)
}

func AssertEquals(expected, actual any, msg ...any) {
	if !Equal(expected, actual) {
		failf(msg, "expected: <%s> but was: <%s>", show(expected), show(actual))
	}
}

func AssertNotEquals(unexpected, actual any, msg ...any) {
	if Equal(unexpected, actual) {
		failf(msg, "expected: not equal but was: <%s>", show(actual))
	}
}

func toFloat(v any) float64 { return reflect.ValueOf(v).Convert(reflect.TypeFor[float64]()).Float() }

func closeTo(e, a, delta float64) bool { return e == a || math.Abs(e-a) <= delta }

func AssertEqualsDelta(expected, actual, delta any, msg ...any) {
	if !closeTo(toFloat(expected), toFloat(actual), toFloat(delta)) {
		failf(msg, "expected: <%v> but was: <%v>", expected, actual)
	}
}

func AssertNotEqualsDelta(unexpected, actual, delta any, msg ...any) {
	if closeTo(toFloat(unexpected), toFloat(actual), toFloat(delta)) {
		failf(msg, "expected: not equal but was: <%v>", actual)
	}
}

// arrayDiff returns "" when both arrays (Go slices, possibly nested) are element-wise Equal.
func arrayDiff(e, a any, delta float64) string {
	if IsNull(e) || IsNull(a) {
		if IsNull(e) && IsNull(a) {
			return ""
		}
		return fmt.Sprintf("expected: <%s> but was: <%s>", show(e), show(a))
	}
	re, ra := reflect.ValueOf(e), reflect.ValueOf(a)
	if re.Len() != ra.Len() {
		return fmt.Sprintf("array lengths differ, expected: <%d> but was: <%d>", re.Len(), ra.Len())
	}
	for i := range re.Len() {
		x, y := re.Index(i).Interface(), ra.Index(i).Interface()
		switch {
		case re.Index(i).Kind() == reflect.Slice:
			if d := arrayDiff(x, y, delta); d != "" {
				return fmt.Sprintf("array contents differ at index [%d], %s", i, d)
			}
		case delta >= 0 && re.Index(i).CanFloat():
			if !closeTo(toFloat(x), toFloat(y), delta) {
				return fmt.Sprintf("array contents differ at index [%d], expected: <%v> but was: <%v>", i, x, y)
			}
		case !Equal(x, y):
			return fmt.Sprintf("array contents differ at index [%d], expected: <%s> but was: <%s>", i, show(x), show(y))
		}
	}
	return ""
}

func AssertArrayEquals(expected, actual any, msg ...any) {
	if d := arrayDiff(expected, actual, -1); d != "" {
		failf(msg, "%s", d)
	}
}

func AssertArrayEqualsDelta(expected, actual, delta any, msg ...any) {
	if d := arrayDiff(expected, actual, toFloat(delta)); d != "" {
		failf(msg, "%s", d)
	}
}

func AssertThrows[T any](fn func(), msg ...any) (res T) {
	defer func() {
		r := recover()
		if r == nil {
			failf(msg, "Expected %s to be thrown, but nothing was thrown.", reflect.TypeFor[T]())
		}
		v, ok := r.(T)
		if !ok {
			failf(msg, "Unexpected exception type thrown, expected: <%s> but was: <%T>: %v", reflect.TypeFor[T](), r, r)
		}
		res = v
	}()
	fn()
	return
}

func AssertDoesNotThrow(fn func(), msg ...any) {
	defer func() {
		if r := recover(); r != nil {
			failf(msg, "Unexpected exception thrown: %T: %v", r, r)
		}
	}()
	fn()
}

func AssertInstanceOf[T any](v any, msg ...any) T {
	t, ok := v.(T)
	if !ok {
		failf(msg, "Unexpected type, expected: <%s> but was: <%T>", reflect.TypeFor[T](), v)
	}
	return t
}

// AssertAll runs every executable and reports all failures together.
func AssertAll(fns ...func()) {
	var failures []string
	for _, fn := range fns {
		func() {
			defer func() {
				if r := recover(); r != nil {
					failures = append(failures, fmt.Sprint(r))
				}
			}()
			fn()
		}()
	}
	if len(failures) > 0 {
		panic(&AssertionFailed{Message: fmt.Sprintf("Multiple Failures (%d failures): %s", len(failures), strings.Join(failures, "; "))})
	}
}

func AssumeTrue(cond bool, msg ...any) {
	if !cond {
		panic(&Skipped{Reason: "Assumption failed: " + message(msg)})
	}
}

func AssumeFalse(cond bool, msg ...any) { AssumeTrue(!cond, msg...) }

// SkipIfEnv is @DisabledIfEnvironmentVariable(named, matches, disabledReason): reason when the
// variable is set and fully matches, else "".
func SkipIfEnv(name, pattern, reason string) string {
	v, ok := os.LookupEnv(name)
	if ok && regexp.MustCompile("^(?:"+pattern+")$").MatchString(v) {
		return reason
	}
	return ""
}

func show(v any) string {
	if IsNull(v) {
		return "null"
	}
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}
	if reflect.ValueOf(v).Kind() == reflect.Pointer {
		return fmt.Sprintf("%T@%p", v, v)
	}
	return fmt.Sprint(v)
}
