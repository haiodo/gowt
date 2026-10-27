package gowt

import (
	"fmt"
	"strconv"

	"github.com/haiodo/gowt/swt"
)

// Scrollbars gives a StyledText vertical and horizontal scroll bars (they appear when needed).
func Scrollbars() Option { return style{id: sScrollbars, set: swt.V_SCROLL | swt.H_SCROLL} }

// StyledTextView is a multi-line editor with per-range styles. All offsets and lengths are in
// runes, i.e. indexes into []rune(Text()); the facade converts to the UTF-16 units SWT uses.
// Each conversion scans the text, O(n) per call: fine for documents up to some hundred
// kilobytes; for larger ones keep a line index and use Unwrap with SWT offsets.
//
// Not wrapped (use Unwrap): bullets other than LineNumbers, line alignment and indent, glyph metrics, underline
// styles and colors, block selection, verify and extended-modify listeners (undo), bidi
// segments, custom content.
type StyledTextView struct {
	base
	t     *swt.StyledText
	fonts *usedFonts
	nums  *lineNumberer // LineNumbers: the installed bullet listener
}

// TextStyle is the look of a range of a StyledText. Nil colors inherit the widget's.
type TextStyle struct {
	Foreground, Background *RGB
	Bold, Italic           bool
	Underline, Strikeout   bool
	Font                   *Font // a typeface other than the widget's, resolved as SetFont does
}

// StyleSpan is a TextStyle applied to Length runes from Start.
type StyleSpan struct {
	Start, Length int
	Style         TextStyle
}

// StyledText creates a multi-line styled editor.
// Options: Scrollbars, Wrap, ReadOnly, Border, NoBorder, NoBackground, Value, OnChange, LineNumbers.
func StyledText(opts ...Option) Option {
	return leaf("StyledText", sScrollbars|sWrap|sReadOnly|sBorder|sNoBorder, opts, func(p *swt.Composite) Widget {
		t := swt.NewStyledText(p, resolve(swt.NONE, opts))
		return &StyledTextView{base: base{&t.Control}, t: t, fonts: newUsedFonts(&t.Control)}
	})
}

// usedFonts holds the shared fonts a widget took for its content, each once, until it is disposed.
type usedFonts struct {
	a     *App
	fonts map[Font]*swt.Font
}

func newUsedFonts(c *swt.Control) *usedFonts {
	u := &usedFonts{appOf(c), map[Font]*swt.Font{}}
	c.AddDisposeListener(u)
	return u
}

// get returns the native font for k, a Font filled by from.
func (u *usedFonts) get(k Font) *swt.Font {
	sf, ok := u.fonts[k]
	if !ok {
		sf = u.a.font(k)
		u.fonts[k] = sf
	}
	return sf
}

func (u *usedFonts) WidgetDisposed(*swt.DisposeEvent) {
	for k := range u.fonts {
		u.a.releaseFont(k)
	}
}

// Unwrap returns the underlying swt.StyledText for the full API (offsets there are UTF-16 units).
func (t *StyledTextView) Unwrap() *swt.StyledText { return t.t }

// Text returns the whole content.
func (t *StyledTextView) Text() string { return t.t.GetText() }

// SetText replaces the whole content.
func (t *StyledTextView) SetText(s string) { t.t.SetText(s) }

// Append adds s at the end.
func (t *StyledTextView) Append(s string) { t.t.Append(s) }

// TrimHead deletes the oldest lines so that at most keep lines remain; TrimHead(0) clears the
// text. It does not touch the rest of the text and works on a read-only view.
func (t *StyledTextView) TrimHead(keep int) {
	n := int(t.t.GetLineCount()) - keep
	switch {
	case n <= 0:
	case keep <= 0:
		t.t.ReplaceTextRange(0, t.t.GetCharCount(), "")
	default:
		t.t.ReplaceTextRange(0, t.t.GetOffsetAtLine(int32(n)), "")
	}
}

// ScrollToEnd moves the caret to the end and scrolls it into view.
func (t *StyledTextView) ScrollToEnd() {
	t.t.SetCaretOffset(t.t.GetCharCount())
	t.t.ShowSelection()
}

// Insert adds s at the caret.
func (t *StyledTextView) Insert(s string) { t.t.Insert(s) }

// Replace replaces length runes from start with s.
func (t *StyledTextView) Replace(start, length int, s string) {
	text := t.t.GetText()
	u := runesToUnits(text, start)
	t.t.ReplaceTextRange(int32(u), int32(runesToUnits(text, start+length)-u), s)
}

// Selection returns the selected range; length is 0 when only the caret is set.
func (t *StyledTextView) Selection() (start, length int) {
	p := t.t.GetSelectionRange()
	text := t.t.GetText()
	start = unitsToRunes(text, int(p.X))
	return start, unitsToRunes(text, int(p.X+p.Y)) - start
}

// SetSelection selects length runes from start.
func (t *StyledTextView) SetSelection(start, length int) {
	text := t.t.GetText()
	u := runesToUnits(text, start)
	t.t.SetSelectionRange(int32(u), int32(runesToUnits(text, start+length)-u))
}

// SelectedText returns the selected text.
func (t *StyledTextView) SelectedText() string { return t.t.GetSelectionText() }

// Caret returns the caret position.
func (t *StyledTextView) Caret() int {
	return unitsToRunes(t.t.GetText(), int(t.t.GetCaretOffset()))
}

// SetCaret moves the caret.
func (t *StyledTextView) SetCaret(offset int) {
	t.t.SetCaretOffset(int32(runesToUnits(t.t.GetText(), offset)))
}

// Reveal scrolls so that the selection is visible.
func (t *StyledTextView) Reveal() { t.t.ShowSelection() }

// Lines returns the number of lines.
func (t *StyledTextView) Lines() int { return int(t.t.GetLineCount()) }

// LineAt returns the line index that holds the rune offset.
func (t *StyledTextView) LineAt(offset int) int {
	return int(t.t.GetLineAtOffset(int32(runesToUnits(t.t.GetText(), offset))))
}

// LineStart returns the rune offset of the first character of line.
func (t *StyledTextView) LineStart(line int) int {
	return unitsToRunes(t.t.GetText(), int(t.t.GetOffsetAtLine(int32(line))))
}

// SetTopLine scrolls so that line is the first visible one.
func (t *StyledTextView) SetTopLine(line int) { t.t.SetTopIndex(int32(line)) }

// SetEditable allows or forbids typing.
func (t *StyledTextView) SetEditable(on bool) { t.t.SetEditable(on) }

// SetWrap turns word wrapping on or off.
func (t *StyledTextView) SetWrap(on bool) { t.t.SetWordWrap(on) }

// SetTabs sets the tab width in spaces.
func (t *StyledTextView) SetTabs(n int) { t.t.SetTabs(int32(n)) }

// Cut moves the selection to the system clipboard.
func (t *StyledTextView) Cut() { t.t.Cut() }

// Copy copies the selection to the system clipboard.
func (t *StyledTextView) Copy() { t.t.Copy() }

// Paste inserts the system clipboard text at the caret.
func (t *StyledTextView) Paste() { t.t.Paste() }

// SetLineBackground colors count lines from line.
func (t *StyledTextView) SetLineBackground(line, count int, c RGB) {
	t.t.SetLineBackground(int32(line), int32(count), c.color())
}

// SetParagraphSpacing leaves before points of empty space above line (NSParagraphStyle
// paragraphSpacingBefore); 0 removes it. A SetLineBackground color stops at the gap, so the gap
// above the first line of a colored block shows the widget background. The space belongs to
// the line: it moves with it when text is inserted above, SetText and a replace of the whole text
// drop it, and it does nothing while an OnStyle listener is set.
func (t *StyledTextView) SetParagraphSpacing(line, before int) {
	t.t.SetLineVerticalIndent(int32(line), int32(before))
}

// SetStyle applies s to length runes from start, replacing the styles there.
func (t *StyledTextView) SetStyle(start, length int, s TextStyle) {
	text := t.t.GetText()
	u := runesToUnits(text, start)
	r := t.styleRange(s)
	r.Start = int32(u)
	r.Length = int32(runesToUnits(text, start+length) - u)
	t.t.SetStyleRange(r)
}

// ClearStyles removes every style.
func (t *StyledTextView) ClearStyles() {
	t.t.ReplaceStyleRanges(0, t.t.GetCharCount(), []*swt.StyleRange{})
}

// Styles returns the styled ranges in order. Font is not reported.
func (t *StyledTextView) Styles() []StyleSpan {
	text := t.t.GetText()
	var out []StyleSpan
	for _, r := range t.t.GetStyleRanges() {
		start := unitsToRunes(text, int(r.Start))
		out = append(out, StyleSpan{start, unitsToRunes(text, int(r.Start+r.Length)) - start, styleOf(r)})
	}
	return out
}

func (t *StyledTextView) styleRange(s TextStyle) *swt.StyleRange {
	r := swt.NewStyleRange()
	if s.Foreground != nil {
		r.Foreground = s.Foreground.color()
	}
	if s.Background != nil {
		r.Background = s.Background.color()
	}
	r.Underline = s.Underline
	r.Strikeout = s.Strikeout
	if s.Bold {
		r.FontStyle |= swt.BOLD
	}
	if s.Italic {
		r.FontStyle |= swt.ITALIC
	}
	if s.Font != nil {
		r.Font = t.font(*s.Font)
	}
	return r
}

func (t *StyledTextView) font(f Font) *swt.Font {
	return t.fonts.get(f.from(t.t.GetFont(), t.t.GetDisplay()))
}

func styleOf(r *swt.StyleRange) TextStyle {
	s := TextStyle{Bold: r.FontStyle&swt.BOLD != 0, Italic: r.FontStyle&swt.ITALIC != 0,
		Underline: r.Underline, Strikeout: r.Strikeout}
	if r.Foreground != nil {
		c := rgbOf(r.Foreground.GetRGB())
		s.Foreground = &c
	}
	if r.Background != nil {
		c := rgbOf(r.Background.GetRGB())
		s.Background = &c
	}
	return s
}

// OnChange runs f with the new content after every edit.
func (t *StyledTextView) OnChange(f func(text string)) {
	t.t.AddModifyListener(&modifier{func() { f(t.t.GetText()) }})
}

// OnSelect runs f with the selection after it changed.
func (t *StyledTextView) OnSelect(f func(start, length int)) {
	onSelect(t.t, func(*swt.SelectionEvent) { f(t.Selection()) })
}

// OnCaret runs f with the caret position after it moved.
func (t *StyledTextView) OnCaret(f func(offset int)) {
	t.t.AddCaretListener(&caretFunc{func() { f(t.Caret()) }})
}

type caretFunc struct{ f func() }

func (c *caretFunc) CaretMoved(*swt.CaretEvent) { c.f() }

// OnStyle runs f each time a line is drawn; the spans it returns style that line only. Start
// is counted in runes from the beginning of the line, so no whole-text conversion happens per
// repaint. f must not change the text.
func (t *StyledTextView) OnStyle(f func(line string) []StyleSpan) {
	t.t.AddLineStyleListener(&lineStyler{t, f})
}

type lineStyler struct {
	t *StyledTextView
	f func(line string) []StyleSpan
}

func (l *lineStyler) LineGetStyle(e *swt.LineStyleEvent) {
	for _, sp := range l.f(e.LineText) {
		u := runesToUnits(e.LineText, sp.Start)
		r := l.t.styleRange(sp.Style)
		r.Start = e.LineOffset + int32(u)
		r.Length = int32(runesToUnits(e.LineText, sp.Start+sp.Length) - u)
		e.Styles = append(e.Styles, r)
	}
}

// runesToUnits converts a rune index of s to a UTF-16 unit index; past the end it clamps.
func runesToUnits(s string, r int) int {
	n := 0
	for _, c := range s {
		if r <= 0 {
			break
		}
		r--
		n++
		if c >= 0x10000 {
			n++
		}
	}
	return n
}

// unitsToRunes converts a UTF-16 unit index of s to a rune index; an index inside a surrogate pair maps to the rune that starts it.
func unitsToRunes(s string, u int) int {
	n := 0
	for _, c := range s {
		w := 1
		if c >= 0x10000 {
			w = 2
		}
		if u < w {
			break
		}
		u -= w
		n++
	}
	return n
}

// LineNumbers shows the line number of every line in a gutter at the left of the text. The numbers
// are SWT bullets drawn with the lines themselves, so they scroll with the text, keep the line spacing
// and the paragraph spacing of each line, stay level with wrapped lines (one number per line) and follow
// the font and the theme. The gutter widens with the digits of the last line number.
func LineNumbers() Option {
	return hook(func(w Widget) error {
		t, ok := w.(*StyledTextView)
		if !ok {
			return fmt.Errorf("gowt: LineNumbers applies to StyledText, not to %s", nameOf(w.control()))
		}
		t.SetLineNumbers(true)
		return nil
	})
}

// NoBackground makes a StyledText paint no background of its own, so what is under it shows through:
// the material of a look.SetMaterial pane, a coloured Panel. Selected text and TextStyle backgrounds
// are still drawn. It is an alpha-0 background colour, which SWT draws as the background of the parent.
func NoBackground() Option {
	return hook(func(w Widget) error {
		t, ok := w.(*StyledTextView)
		if !ok {
			return fmt.Errorf("gowt: NoBackground applies to StyledText, not to %s", nameOf(w.control()))
		}
		// White, not black: the GTK port takes an all-zero GdkRGBA for a disposed Color.
		t.t.SetBackgroundWithColor(swt.NewColorDeviceRedGreenBlueAlpha(t.t.GetDisplay(), 255, 255, 255, 0))
		return nil
	})
}

// SetLineNumbers shows or hides the line number gutter, see LineNumbers.
func (t *StyledTextView) SetLineNumbers(on bool) {
	if on == (t.nums != nil) {
		return
	}
	if on {
		t.nums = &lineNumberer{t: t}
		t.t.AddLineStyleListener(t.nums)
	} else {
		t.t.RemoveLineStyleListener(t.nums)
		t.nums = nil
	}
	t.t.Redraw()
}

type lineNumberer struct {
	t     *StyledTextView
	font  *swt.Font // the font digit was measured with
	digit int32
}

func (l *lineNumberer) LineGetStyle(e *swt.LineStyleEvent) {
	t := l.t.t
	line := t.GetLineAtOffset(e.LineOffset)
	if f := t.GetFont(); f != l.font {
		gc := swt.NewGCDrawable(t)
		gc.SetFont(f)
		l.font, l.digit = f, gc.TextExtent("0").X
		gc.Dispose()
	}
	digit := l.digit
	digits := int32(len(strconv.Itoa(int(t.GetLineCount()))))
	st := swt.NewStyleRange()
	st.Foreground = SecondaryText.in(l.t.dark()).color()
	// Two digits at least, so the gutter does not jump between 9 and 10 lines; the margin is the renderer's.
	st.Metrics = swt.NewGlyphMetrics(0, 0, max(digits, 2)*digit+2*digit)
	e.Bullet = swt.NewBulletTypeStyle(swt.STBULLET_NUMBER, st)
	e.BulletIndex = line
}

func (t *StyledTextView) dark() bool {
	if a := appOf(&t.t.Control); a != nil {
		return a.dark
	}
	return systemDark()
}
