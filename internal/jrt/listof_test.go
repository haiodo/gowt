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

func TestTreeSetAddSortsAndDedups(t *testing.T) {
	s := NewTreeSet()
	for _, v := range []int32{5, 1, 3, 5, 0, 3, 9} {
		s.Add(v)
	}
	got := s.ToArray()
	want := []any{int32(0), int32(1), int32(3), int32(5), int32(9)}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if s.Add(int32(3)) || !s.Add(int32(4)) {
		t.Fatal("Add must report false for a present element and true for a new one")
	}
	if s.Higher(int32(4)) != int32(5) || s.Lower(int32(4)) != int32(3) {
		t.Fatalf("Higher/Lower around 4: %v %v", s.Higher(int32(4)), s.Lower(int32(4)))
	}
}
