// Hand-written opaque stubs for the rest of manual.txt's list: IME (Composite/
// Canvas/Decorations/Shell/Button's own field types, not translated yet), plus the graphics/
// accessibility leaf types Control.java reads fields off of. Composite/Canvas/Decorations/Shell
// are real as of Round 5, Menu/MenuItem as of Round 7, ScrollBar as of Round 8, Caret as of Round 10.
package swt

import "github.com/haiodo/gowt/internal/cocoa"

// IME: Canvas.java null-checks it before every use, so a nil *IME is "no IME". Every param is a
// raw objc-runtime int64 handle (id/SEL/pointer), matching IME.java's native-bridge signatures.
type IME struct {
	startOffset int32
}

// StyledText builds one and reads the composition back; with no input method there is no composition.
func (i *IME) AddListener(eventType int32, listener Listener) {}

func NewIME(parent *Canvas, style int32) *IME { return &IME{} }

func (i *IME) GetCaretOffset() int32             { return 0 }
func (i *IME) GetCommitCount() int32             { return 0 }
func (i *IME) GetCompositionOffset() int32       { return -1 }
func (i *IME) GetRanges() []int32                { return nil }
func (i *IME) GetStyles() []*TextStyle           { return nil }
func (i *IME) GetText() string                   { return "" }
func (i *IME) GetWideCaret() bool                { return false }
func (i *IME) SetCompositionOffset(offset int32) {}

func (i *IME) IsDisposed() bool                                                       { return i == nil }
func (i *IME) AttributedSubstringFromRange(id int64, sel int64, rangePtr int64) int64 { return 0 }
func (i *IME) CharacterIndexForPoint(id int64, sel int64, point int64) int64          { return 0 }
func (i *IME) FirstRectForCharacterRange(id int64, sel int64, r int64) cocoa.NSRect {
	return cocoa.NSRect{}
}
func (i *IME) HasMarkedText(id int64, sel int64) bool          { return false }
func (i *IME) IsInlineEnabled() bool                           { return false }
func (i *IME) InsertText(id int64, sel int64, str int64) bool  { return true }
func (i *IME) MarkedRange(id int64, sel int64) cocoa.NSRange   { return cocoa.NSRange{} }
func (i *IME) Release(bool)                                    {}
func (i *IME) Reskin(flags int32)                              {}
func (i *IME) SelectedRange(id int64, sel int64) cocoa.NSRange { return cocoa.NSRange{} }
func (i *IME) SetMarkedText_selectedRange(id int64, sel int64, str int64, selRange int64) bool {
	return true
}
func (i *IME) ValidAttributesForMarkedText(id int64, sel int64) int64 { return 0 }

// org.eclipse.swt.internal.WidgetSpy: creation/disposal tracking, off by default (matches the
// real class's own isEnabled starting false).
var WidgetSpyIsEnabled bool

type widgetSpy struct{}

func (widgetSpy) WidgetCreated(w *Widget)  {}
func (widgetSpy) WidgetDisposed(w *Widget) {}

func WidgetSpyGetInstance() widgetSpy { return widgetSpy{} }
