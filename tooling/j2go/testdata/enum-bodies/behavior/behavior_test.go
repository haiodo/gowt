package app

import "testing"

func TestBodies(t *testing.T) {
	if OpPLUS.Apply(2, 3) != 5 || OpTIMES.Apply(2, 3) != 6 || OpNEG.Apply(2, 3) != -2 {
		t.Fatal("apply")
	}
	if OpTIMES.String() != "times *" || OpPLUS.String() != "PLUS" || OpFold(OpTIMES, []int32{2, 3, 4}) != 24 {
		t.Fatal("string/fold")
	}
	if OpValueOf("NEG") != OpNEG || len(OpValues()) != 3 || OpPLUS.CompareTo(OpNEG) != -2 {
		t.Fatal("values")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("valueOf must panic")
			}
		}()
		OpValueOf("nope")
	}()
	if ShapeCIRCLE.Label() != "round" {
		t.Fatal("shape")
	}
}
