package jrt

import "testing"

func TestListOfSpreadsReferenceArrays(t *testing.T) {
	type ctl struct{ n int }
	kids := []*ctl{{1}, {2}}
	if n := ListOf(kids).Size(); n != 2 {
		t.Fatalf("ListOf([]*ctl) size %d, want 2", n)
	}
	if n := ListOf([]string{"a", "b", "c"}).Size(); n != 3 {
		t.Fatalf("ListOf([]string) size %d, want 3", n)
	}
	if n := ListOf([]int32{1, 2}).Size(); n != 1 {
		t.Fatalf("ListOf([]int32) size %d, want 1 (a primitive array is one element)", n)
	}
	if n := ListOf(int32(1)).Size(); n != 1 {
		t.Fatalf("ListOf(int32) size %d, want 1", n)
	}
}
