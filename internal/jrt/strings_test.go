package jrt

import "testing"

func TestCyrillicIndices(t *testing.T) {
	s := "привет мир, привет"
	if StringLength(s) != 18 {
		t.Fatal("length", StringLength(s))
	}
	i, j := IndexFrom(s, "мир", 0), LastIndexOf(s, "привет")
	if i != 7 || j != 12 || IndexFrom(s, "привет", 1) != 12 || IndexFrom(s, "я", 0) != -1 {
		t.Fatal("indexOf", i, j)
	}
	if Substring(s, 0, i) != "привет " || Substring(s, j, -1) != "привет" {
		t.Fatal("substring round-trip")
	}
	if LastIndexOf(s, "т") != 17 {
		t.Fatal("lastIndexOf char")
	}
}
