package swt

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/haiodo/gowt/internal/jrt"
)

// Emoji are two UTF-16 units; every offset the content takes or returns is in units.
func TestDefaultContentSurrogates(t *testing.T) {
	const smile = "\U0001F600"
	c := newDefaultContent()
	c.SetText("ab" + smile + "cd\n" + smile + "x")
	if got := c.GetCharCount(); got != 10 {
		t.Fatalf("char count %d, want 10", got)
	}
	if got := c.GetLine(0); got != "ab"+smile+"cd" {
		t.Fatalf("line 0 %q", got)
	}
	c.ReplaceTextRange(2, 2, "")
	if got := c.GetLine(0); got != "abcd" {
		t.Fatalf("after delete %q", got)
	}
	c.ReplaceTextRange(1, 0, smile+smile)
	if got := c.GetLine(0); got != "a"+smile+smile+"bcd" {
		t.Fatalf("after insert %q", got)
	}
	if got := c.GetLine(1); got != smile+"x" {
		t.Fatalf("line 1 %q", got)
	}
	if got := c.GetTextRange(1, 4); got != smile+smile {
		t.Fatalf("range %q", got)
	}
}

// Random edits, some cutting a pair in half, against a plain []uint16 model.
func TestDefaultContentSurrogatesRandom(t *testing.T) {
	pieces := []string{"a", "b", "\n", "\U0001F600", "\U0001F468‍\U0001F469", "\U0001F1FA\U0001F1F8", "xyz"}
	r := rand.New(rand.NewSource(1))
	for round := 0; round < 300; round++ {
		c := newDefaultContent()
		model := []uint16{}
		for op := 0; op < 40; op++ {
			ins := ""
			for i := r.Intn(3); i > 0; i-- {
				ins += pieces[r.Intn(len(pieces))]
			}
			start := r.Intn(len(model) + 1)
			n := r.Intn(len(model) - start + 1)
			if n > 3 {
				n = r.Intn(4)
			}
			c.ReplaceTextRange(int32(start), int32(n), ins)
			model = append(append(append([]uint16{}, model[:start]...), jrt.ToChars(ins)...), model[start+n:]...)
			if got := c.GetCharCount(); int(got) != len(model) {
				t.Fatalf("round %d op %d: count %d, model %d", round, op, got, len(model))
			}
			for l := int32(0); l < c.GetLineCount(); l++ {
				c.GetLine(l)
			}
			if got := jrt.ToChars(c.GetTextRange(0, int32(len(model)))); !slices.Equal(got, model) {
				t.Fatalf("round %d op %d: text %x, model %x", round, op, got, model)
			}
		}
	}
}
