package jrt

import (
	"math"
	"testing"
)

func TestAssert(t *testing.T) {
	AssertIsTrue(true)
	defer func() {
		if _, ok := recover().(*AssertionFailedException); !ok {
			t.Fatal("isTrue(false) must panic AssertionFailedException")
		}
	}()
	AssertIsTrue(false, "x")
}

func TestListenerListAndSafeRunner(t *testing.T) {
	l := NewListenerList()
	a, b := new(int), new(int)
	l.Add(a)
	l.Add(a)
	l.Add(b)
	if l.Size() != 2 {
		t.Fatal("size", l.Size())
	}
	l.Remove(a)
	if ls := l.GetListeners(); len(ls) != 1 || ls[0] != any(b) {
		t.Fatal("remove", ls)
	}
	var got error
	SafeRunnerRun(&safe{func() { panic(NewIllegalArgumentException("boom")) }, &got})
	if got == nil || got.Error() != "boom" {
		t.Fatal("handler not called", got)
	}
}

type safe struct {
	run func()
	got *error
}

func (s *safe) Run()                    { s.run() }
func (s *safe) HandleException(e error) { *s.got = e }

func TestDoubleToInt(t *testing.T) {
	for in, want := range map[float64]int32{math.NaN(): 0, 3.9: 3, -3.9: -3, 1e20: math.MaxInt32, -1e20: math.MinInt32} {
		if got := DoubleToInt(in); got != want {
			t.Errorf("DoubleToInt(%v) = %d, want %d", in, got, want)
		}
	}
}
