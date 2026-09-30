package cocoa

import "testing"

func TestIntersectionRect(t *testing.T) {
	a := NSRect{0, 0, 200, 200}
	b := NSRect{5, 10, 20, 30}
	OSNSIntersectionRect(&a, &a, &b)
	if a != b {
		t.Fatalf("intersection = %v, want %v", a, b)
	}
	c := NSRect{100, 100, 1, 1}
	OSNSIntersectionRect(&a, &a, &c)
	if a != (NSRect{}) {
		t.Fatalf("disjoint intersection = %v, want zero", a)
	}
}
