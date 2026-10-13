package app

import "testing"

func TestRecordValueSemantics(t *testing.T) {
	if !NewKeys().Same() || NewPt(1, 2).Equals(NewPt(2, 1)) || NewPt(1, 2).Equals(nil) {
		t.Fatal("equals")
	}
	if got := NewKeys().Lookup(); got != "aPt[x=3, y=4]33" {
		t.Fatal(got)
	}
}
