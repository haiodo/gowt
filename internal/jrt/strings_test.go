package jrt

import (
	"testing"
	"unicode/utf8"
)

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

func TestSurrogatePairSplit(t *testing.T) {
	s := "a\U0001F600b"
	if StringLength(s) != 4 {
		t.Fatal("length", StringLength(s))
	}
	hi, lo := Substring(s, 0, 2), Substring(s, 2, -1)
	if StringLength(hi) != 2 || StringLength(lo) != 2 {
		t.Fatal("halves", StringLength(hi), StringLength(lo))
	}
	if StringLength(hi+lo) != 4 || FromChars(ToChars(hi+lo)) != s || Substring(hi+lo, 1, 3) != "\U0001F600" {
		t.Fatal("concat of halves")
	}
	if StringLength(Substring(s, 1, 2)+Substring(s, 2, 3)) != 2 {
		t.Fatal("lone halves")
	}
	if got := ValidUTF8(hi + lo); got != s {
		t.Fatalf("pair at the OS boundary: %q", got)
	}
	if got := ValidUTF8(hi); got != "a�" || !utf8.ValidString(got) {
		t.Fatalf("lone surrogate at the OS boundary: %q", got)
	}
	if got := ValidUTF8(lo + hi); got != "\uFFFDba\uFFFD" {
		t.Fatalf("reversed halves: %q", got)
	}
}
