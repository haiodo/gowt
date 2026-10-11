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

func TestCaseMappingKeepsLoneSurrogate(t *testing.T) {
	hi := Substring("a\U0001F600b", 0, 2)
	if got := ToUpperCase(hi + "x"); got != "A"+Substring("\U0001F600", 0, 1)+"X" || StringLength(got) != 3 {
		t.Fatalf("upper: %q", got)
	}
	lo := Substring("\U0001F600B", 1, -1)
	if got := ToLowerCase(lo); StringLength(got) != 2 || ToChars(got)[1] != 'b' || ToChars(got)[0] != ToChars(lo)[0] {
		t.Fatalf("lower: %q", got)
	}
	if ToUpperCase("привет") != "ПРИВЕТ" {
		t.Fatal("valid text")
	}
}

func TestStringBuilderUnits(t *testing.T) {
	sb := NewStringBuilder("привет")
	if sb.Length() != 6 || sb.CharAt(0) != 'п' {
		t.Fatal("cyrillic", sb.Length(), sb.CharAt(0))
	}
	sb.Insert(3, "X")
	if sb.ToString() != "приXвет" {
		t.Fatal("insert", sb.ToString())
	}
	sb.Delete(1, 3).DeleteCharAt(0)
	if sb.ToString() != "Xвет" || sb.IndexOf("ет") != 2 {
		t.Fatal("delete", sb.ToString())
	}
	sb.SetLength(2)
	if sb.ToString() != "Xв" {
		t.Fatal("setLength", sb.ToString())
	}
	sb.SetLength(3)
	if sb.Length() != 3 || sb.CharAt(2) != 0 {
		t.Fatal("setLength pad")
	}

	e := NewStringBuilder("a\U0001F600b")
	if e.Length() != 4 || e.CharAt(1) != 0xD83D || e.CharAt(2) != 0xDE00 {
		t.Fatal("emoji", e.Length())
	}
	e.Insert(2, "-")
	if StringLength(e.ToString()) != 5 || e.CharAt(2) != '-' {
		t.Fatal("insert inside a pair")
	}
	e.DeleteCharAt(2)
	if e.ToString() != "a\U0001F600b" {
		t.Fatalf("pair restored: %q", e.ToString())
	}
	e.Replace(1, 3, "z")
	if e.ToString() != "azb" {
		t.Fatal("replace", e.ToString())
	}
	e.Append([]uint16{0xD83D}).Append([]uint16{0xDE00})
	if e.Length() != 5 || e.ToString() != "azb\U0001F600" {
		t.Fatal("append halves", e.Length())
	}
	dst := make([]uint16, 2)
	e.GetChars(3, 5, dst, 0)
	if dst[0] != 0xD83D || dst[1] != 0xDE00 {
		t.Fatal("getChars")
	}
}
