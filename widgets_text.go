package gowt

import "github.com/haiodo/gowt/swt"

// Scrollbars gives a StyledText vertical and horizontal scroll bars (they appear when needed).
func Scrollbars() Option { return Option{style: swt.V_SCROLL | swt.H_SCROLL} }

// StyledText is a multi-line editor with per-range styles. All offsets and lengths are in
// runes, i.e. indexes into []rune(Text()); the facade converts to the UTF-16 units SWT uses.
// Each conversion scans the text, O(n) per call: fine for documents up to some hundred
// kilobytes; for larger ones keep a line index and use Unwrap with SWT offsets.
//
// Not wrapped (use Unwrap): bullets, line alignment and indent, glyph metrics, underline
// styles and colors, block selection, verify and extended-modify listeners (undo), bidi
// segments, custom content.
type StyledText struct {
	t     *swt.StyledText
	fonts map[Font]*swt.Font
}

// TextStyle is the look of a range of a StyledText. Nil colors inherit the widget's.
type TextStyle struct {
	Foreground, Background *RGB
	Bold, Italic           bool
	Underline, Strikeout   bool
	Font                   *Font // a typeface other than the widget's; the widget owns the native font
}

// StyleSpan is a TextStyle applied to Length runes from Start.
type StyleSpan struct {
	Start, Length int
	Style         TextStyle
}

// StyledText adds a multi-line styled editor; options: Scrollbars, Wrap, ReadOnly, Border.
func (p *panel) StyledText(opts ...Option) *StyledText {
	t := swt.NewStyledText(p.c, resolve(swt.NONE, opts))
	applyOpts(&t.Control, opts)
	w := &StyledText{t: t, fonts: map[Font]*swt.Font{}}
	t.AddDisposeListener(&fontDisposer{w})
	return w
}

type fontDisposer struct{ w *StyledText }

func (d *fontDisposer) WidgetDisposed(*swt.DisposeEvent) {
	for _, f := range d.w.fonts {
		f.Dispose()
	}
}

func (t *StyledText) control() *swt.Control { return &t.t.Control }

// Unwrap returns the underlying swt.StyledText for the full API (offsets there are UTF-16 units).
func (t *StyledText) Unwrap() *swt.StyledText { return t.t }

// Text returns the whole content.
func (t *StyledText) Text() string { return t.t.GetText() }

// SetText replaces the whole content.
func (t *StyledText) SetText(s string) { t.t.SetText(s) }

// Append adds s at the end.
func (t *StyledText) Append(s string) { t.t.Append(s) }

// Insert adds s at the caret.
func (t *StyledText) Insert(s string) { t.t.Insert(s) }

// Replace replaces length runes from start with s.
func (t *StyledText) Replace(start, length int, s string) {
	text := t.t.GetText()
	u := runesToUnits(text, start)
	t.t.ReplaceTextRange(int32(u), int32(runesToUnits(text, start+length)-u), s)
}

// Selection returns the selected range; length is 0 when only the caret is set.
func (t *StyledText) Selection() (start, length int) {
	p := t.t.GetSelectionRange()
	text := t.t.GetText()
	start = unitsToRunes(text, int(p.X))
	return start, unitsToRunes(text, int(p.X+p.Y)) - start
}

// SetSelection selects length runes from start.
func (t *StyledText) SetSelection(start, length int) {
	text := t.t.GetText()
	u := runesToUnits(text, start)
	t.t.SetSelectionRange(int32(u), int32(runesToUnits(text, start+length)-u))
}

// SelectedText returns the selected text.
func (t *StyledText) SelectedText() string { return t.t.GetSelectionText() }

// Caret returns the caret position.
func (t *StyledText) Caret() int {
	return unitsToRunes(t.t.GetText(), int(t.t.GetCaretOffset()))
}

// SetCaret moves the caret.
func (t *StyledText) SetCaret(offset int) {
	t.t.SetCaretOffset(int32(runesToUnits(t.t.GetText(), offset)))
}

// Reveal scrolls so that the selection is visible.
func (t *StyledText) Reveal() { t.t.ShowSelection() }

// Lines returns the number of lines.
func (t *StyledText) Lines() int { return int(t.t.GetLineCount()) }

// LineAt returns the line index that holds the rune offset.
func (t *StyledText) LineAt(offset int) int {
	return int(t.t.GetLineAtOffset(int32(runesToUnits(t.t.GetText(), offset))))
}

// LineStart returns the rune offset of the first character of line.
func (t *StyledText) LineStart(line int) int {
	return unitsToRunes(t.t.GetText(), int(t.t.GetOffsetAtLine(int32(line))))
}

// SetTopLine scrolls so that line is the first visible one.
func (t *StyledText) SetTopLine(line int) { t.t.SetTopIndex(int32(line)) }

// SetEditable allows or forbids typing.
func (t *StyledText) SetEditable(on bool) { t.t.SetEditable(on) }

// SetWrap turns word wrapping on or off.
func (t *StyledText) SetWrap(on bool) { t.t.SetWordWrap(on) }

// SetTabs sets the tab width in spaces.
func (t *StyledText) SetTabs(n int) { t.t.SetTabs(int32(n)) }

// Cut moves the selection to the system clipboard.
func (t *StyledText) Cut() { t.t.Cut() }

// Copy copies the selection to the system clipboard.
func (t *StyledText) Copy() { t.t.Copy() }

// Paste inserts the system clipboard text at the caret.
func (t *StyledText) Paste() { t.t.Paste() }

// SetLineBackground colors count lines from line.
func (t *StyledText) SetLineBackground(line, count int, c RGB) {
	t.t.SetLineBackground(int32(line), int32(count), c.color())
}

// SetStyle applies s to length runes from start, replacing the styles there.
func (t *StyledText) SetStyle(start, length int, s TextStyle) {
	text := t.t.GetText()
	u := runesToUnits(text, start)
	r := t.styleRange(s)
	r.Start = int32(u)
	r.Length = int32(runesToUnits(text, start+length) - u)
	t.t.SetStyleRange(r)
}

// ClearStyles removes every style.
func (t *StyledText) ClearStyles() {
	t.t.ReplaceStyleRanges(0, t.t.GetCharCount(), []*swt.StyleRange{})
}

// Styles returns the styled ranges in order. Font is not reported.
func (t *StyledText) Styles() []StyleSpan {
	text := t.t.GetText()
	var out []StyleSpan
	for _, r := range t.t.GetStyleRanges() {
		start := unitsToRunes(text, int(r.Start))
		out = append(out, StyleSpan{start, unitsToRunes(text, int(r.Start+r.Length)) - start, styleOf(r)})
	}
	return out
}

func (t *StyledText) styleRange(s TextStyle) *swt.StyleRange {
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

func (t *StyledText) font(f Font) *swt.Font {
	if sf, ok := t.fonts[f]; ok {
		return sf
	}
	var st int32
	if f.Bold {
		st |= swt.BOLD
	}
	if f.Italic {
		st |= swt.ITALIC
	}
	sf := swt.NewFontDeviceNameHeightStyle(t.t.GetDisplay(), f.Name, int32(f.Size), st)
	t.fonts[f] = sf
	return sf
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
func (t *StyledText) OnChange(f func(text string)) {
	t.t.AddModifyListener(&modifier{func() { f(t.t.GetText()) }})
}

// OnSelect runs f with the selection after it changed.
func (t *StyledText) OnSelect(f func(start, length int)) {
	onSelect(t.t, func(*swt.SelectionEvent) { f(t.Selection()) })
}

// OnCaret runs f with the caret position after it moved.
func (t *StyledText) OnCaret(f func(offset int)) {
	t.t.AddCaretListener(&caretFunc{func() { f(t.Caret()) }})
}

type caretFunc struct{ f func() }

func (c *caretFunc) CaretMoved(*swt.CaretEvent) { c.f() }

// OnStyle runs f each time a line is drawn; the spans it returns style that line only. Start
// is counted in runes from the beginning of the line, so no whole-text conversion happens per
// repaint. f must not change the text.
func (t *StyledText) OnStyle(f func(line string) []StyleSpan) {
	t.t.AddLineStyleListener(&lineStyler{t, f})
}

type lineStyler struct {
	t *StyledText
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
