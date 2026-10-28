package swt

import (
	"math/rand"
	"runtime"
	"slices"
	"testing"

	"github.com/haiodo/gowt/internal/jrt"
)

const (
	tSmile = "\U0001F600"
	tZWJ   = "\U0001F468‍\U0001F469‍\U0001F467" // 8 units
	tFlag  = "\U0001F1FA\U0001F1F8"             // 4 units
)

func guiDisplay(t *uiT) *Display {
	d := NewDisplay()
	t.Cleanup(d.Dispose)
	return d
}

func pump(d *Display) {
	for d.ReadAndDispatch() {
	}
}

func newEmojiStyledText(t *uiT, style int32, text string) (*Display, *Shell, *StyledText) {
	d := guiDisplay(t)
	sh := NewShellDisplayStyle(d, SHELL_TRIM)
	sh.SetSize(300, 200)
	st := NewStyledText(&sh.Composite, style)
	st.SetText(text)
	sh.Open()
	pump(d)
	return d, sh, st
}

func TestStyledTextEmojiEditing(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		text := "a" + tSmile + "b" + tZWJ + "c" + tFlag + "d\nsecond " + tSmile
		d, sh, st := newEmojiStyledText(t, MULTI|V_SCROLL, text)
		defer sh.Dispose()
		if got := st.GetCharCount(); got != 1+2+1+8+1+4+1+1+7+2 {
			t.Fatalf("char count %d", got)
		}
		if got := st.GetText(); got != text {
			t.Fatalf("text %q", got)
		}
		// typing: inserts around, between and after pairs
		st.SetCaretOffset(3)
		st.Insert("X")
		st.ReplaceTextRange(0, 0, tSmile)
		pump(d)
		if got := st.GetLine(0); got != tSmile+"a"+tSmile+"Xb"+tZWJ+"c"+tFlag+"d" {
			t.Fatalf("after typing %q", got)
		}
		// caret moves over whole clusters' units one unit at a time without panics
		st.SetCaretOffset(0)
		for i := int32(0); i < st.GetCharCount(); i++ {
			st.InvokeAction(STCOLUMN_NEXT)
			pump(d)
			if st.GetCaretOffset() < i {
				t.Fatalf("caret went back at %d: %d", i, st.GetCaretOffset())
			}
		}
		for st.GetCaretOffset() > 0 {
			before := st.GetCaretOffset()
			st.InvokeAction(STCOLUMN_PREVIOUS)
			if st.GetCaretOffset() >= before {
				t.Fatalf("caret stuck at %d", before)
			}
		}
		// delete and backspace over emoji
		st.SetText("x" + tSmile + "y" + tZWJ + "z")
		st.SetCaretOffset(1)
		st.InvokeAction(STDELETE_NEXT)
		if got := st.GetText(); got != "xy"+tZWJ+"z" {
			t.Fatalf("delete: %q", got)
		}
		st.SetCaretOffset(2 + 8)
		st.InvokeAction(STDELETE_PREVIOUS)
		if got := st.GetText(); got != "xyz" {
			t.Fatalf("backspace: %q", got)
		}
		// selection
		st.SetText("a" + tFlag + "b")
		st.SetSelectionRange(1, 4)
		if got := st.GetSelectionText(); got != tFlag {
			t.Fatalf("selection %q", got)
		}
		// a selection cutting a pair in half must not panic
		st.SetSelectionRange(0, 2)
		_ = st.GetSelectionText()
		st.SetSelectionRange(2, 1)
		_ = st.GetSelectionText()
		pump(d)
	})
}

func TestStyledTextEmojiClipboard(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		d, sh, st := newEmojiStyledText(t, MULTI, "a"+tSmile+tZWJ+"b")
		defer sh.Dispose()
		st.SetSelectionRange(1, 10)
		st.Copy()
		st.SetCaretOffset(st.GetCharCount())
		st.Paste()
		pump(d)
		if got := st.GetText(); got != "a"+tSmile+tZWJ+"b"+tSmile+tZWJ {
			t.Fatalf("pasted %q", got)
		}
		st.SetSelectionRange(1, 10)
		st.Cut()
		pump(d)
		if got := st.GetText(); got != "ab"+tSmile+tZWJ {
			t.Fatalf("cut %q", got)
		}
	})
}

func TestStyledTextEmojiWrapAndStyles(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		text := ""
		for i := 0; i < 40; i++ {
			text += tSmile + tFlag + "ab "
		}
		d, sh, st := newEmojiStyledText(t, MULTI|WRAP|V_SCROLL, text)
		defer sh.Dispose()
		pump(d)
		n := st.GetCharCount()
		for off := int32(0); off <= n; off++ {
			l := st.GetLineAtOffset(off)
			if l < 0 || l >= st.GetLineCount() {
				t.Fatalf("line %d at %d", l, off)
			}
			st.GetLocationAtOffset(off)
			if off < n {
				st.GetTextBounds(off, n-1)
			}
		}
		st.GetOffsetAtPoint(NewPoint(5, 5))
		st.GetOffsetAtPoint(NewPoint(250, 60))
		// style ranges starting or ending inside a pair
		for _, r := range [][2]int32{{1, 1}, {0, 1}, {1, 3}, {3, 1}, {2, 4}, {n - 1, 1}} {
			sr := NewStyleRange()
			sr.Start, sr.Length = r[0], r[1]
			sr.FontStyle = BOLD
			st.SetStyleRange(sr)
			st.GetStyleRangeAtOffset(r[0])
			pump(d)
		}
		st.Redraw()
		pump(d)
		st.SetText("")
		pump(d)
	})
}

func TestTextComboLabelEmoji(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		d := guiDisplay(t)
		sh := NewShellDisplayStyle(d, SHELL_TRIM)
		defer sh.Dispose()
		txt := NewText(&sh.Composite, BORDER)
		multi := NewText(&sh.Composite, MULTI|BORDER)
		cb := NewCombo(&sh.Composite, BORDER)
		lb := NewLabel(&sh.Composite, NONE)
		s := "a" + tSmile + "b" + tZWJ + "c" + tFlag
		txt.SetText(s)
		multi.SetText(s + "\n" + s)
		cb.SetText(s)
		cb.Add(s)
		lb.SetText(s)
		sh.Open()
		pump(d)
		if got := txt.GetText(); got != s {
			t.Fatalf("Text %q", got)
		}
		// Windows Text stores line breaks as CRLF and returns them, as SWT does.
		nl := "\n"
		if runtime.GOOS == "windows" {
			nl = "\r\n"
		}
		if got := multi.GetText(); got != s+nl+s {
			t.Fatalf("multi %q", got)
		}
		if got := cb.GetText(); got != s {
			t.Fatalf("Combo %q", got)
		}
		if got := lb.GetText(); got != s {
			t.Fatalf("Label %q", got)
		}
		if got := txt.GetCharCount(); got != 1+2+1+8+1+4 {
			t.Fatalf("Text count %d", got)
		}
		txt.SetSelectionStartEnd(1, 3)
		if got := txt.GetSelectionText(); got != tSmile {
			t.Fatalf("Text selection %q", got)
		}
		txt.Insert("Z")
		if got := txt.GetText(); got != "a"+"Z"+"b"+tZWJ+"c"+tFlag {
			t.Fatalf("Text insert %q", got)
		}
		pump(d)
	})
}

// Random edits through the widget, some cutting a pair in two, against a []uint16 model.
func TestStyledTextEmojiRandom(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		pieces := []string{"a", "b", "\n", tSmile, tZWJ, tFlag, "xyz", " word "}
		d, sh, st := newEmojiStyledText(t, MULTI|WRAP|V_SCROLL, "")
		defer sh.Dispose()
		r := rand.New(rand.NewSource(7))
		var model []uint16
		for op := 0; op < 400; op++ {
			ins := ""
			for i := r.Intn(3); i > 0; i-- {
				ins += pieces[r.Intn(len(pieces))]
			}
			start := r.Intn(len(model) + 1)
			n := r.Intn(len(model) - start + 1)
			if n > 6 {
				n = r.Intn(7)
			}
			st.ReplaceTextRange(int32(start), int32(n), ins)
			model = append(append(append([]uint16{}, model[:start]...), jrt.ToChars(ins)...), model[start+n:]...)
			cnt := int32(len(model))
			if st.GetCharCount() != cnt {
				t.Fatalf("op %d: count %d, model %d", op, st.GetCharCount(), cnt)
			}
			if cnt > 0 {
				sr := NewStyleRange()
				sr.Start = int32(r.Intn(len(model)))
				sr.Length = 1 + int32(r.Intn(int(cnt-sr.Start)))
				sr.FontStyle = BOLD
				st.SetStyleRange(sr)
				st.SetSelectionRange(int32(r.Intn(len(model))), int32(r.Intn(3)))
				_ = st.GetSelectionText()
				st.InvokeAction(STCOLUMN_NEXT)
				st.InvokeAction(STSELECT_COLUMN_PREVIOUS)
				st.GetLocationAtOffset(st.GetCaretOffset())
			}
			pump(d)
			if got := jrt.ToChars(st.GetText()); !slices.Equal(got, model) {
				t.Fatalf("op %d: text %x, model %x", op, got, model)
			}
		}
	})
}

func TestTextEmojiRandom(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		pieces := []string{"a", tSmile, tZWJ, tFlag, "xyz"}
		d := guiDisplay(t)
		sh := NewShellDisplayStyle(d, SHELL_TRIM)
		defer sh.Dispose()
		txt := NewText(&sh.Composite, BORDER)
		multi := NewText(&sh.Composite, MULTI|BORDER)
		cb := NewCombo(&sh.Composite, BORDER)
		sh.Open()
		r := rand.New(rand.NewSource(3))
		for op := 0; op < 200; op++ {
			ins := pieces[r.Intn(len(pieces))] + pieces[r.Intn(len(pieces))]
			for _, w := range []*Text{txt, multi} {
				n := w.GetCharCount()
				a := int32(r.Intn(int(n) + 1))
				w.SetSelectionStartEnd(a, a+int32(r.Intn(int(n-a)+1)))
				w.Insert(ins)
				_ = w.GetText()
				_ = w.GetSelectionText()
			}
			n := jrt.StringLength(cb.GetText())
			a := int32(r.Intn(int(n) + 1))
			cb.SetSelection(NewPoint(a, a+int32(r.Intn(int(n-a)+1))))
			cb.SetText(cb.GetText() + ins)
			pump(d)
		}
	})
}
