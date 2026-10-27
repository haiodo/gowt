//go:build darwin

// Hand-written: inset, transparency and content height of a multi-line text view, and the fill of a
// borderless text field on glass.
package cocoa

import (
	"math"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	textViewOnce sync.Once
	tvSetSize    func(self, sel uintptr, s NSSize)
	tvFloat      func(self, sel, font uintptr) float64
)

func textViewInit() {
	textViewOnce.Do(func() {
		objcInit()
		purego.RegisterFunc(&tvSetSize, objcMsgSend)
		purego.RegisterFunc(&tvFloat, objcMsgSend)
	})
}

// SetTextViewInset sets textContainerInset: x left and right, y above and below.
// NSTextView has no per-side inset.
func SetTextViewInset(view *NSView, x, y float64) {
	textViewInit()
	tvSetSize(uintptr(view.Id), sel("setTextContainerInset:"), NSSize{Width: x, Height: y})
}

// ClearTextViewBackground stops an NSTextView and the NSScrollView around it (and its clip view) from
// painting a background, so a glass behind them shows.
func ClearTextViewBackground(view *NSView) {
	v := uintptr(view.Id)
	msg(v, "setDrawsBackground:", 0)
	if sv := msg(v, "enclosingScrollView"); sv != 0 {
		msg(sv, "setDrawsBackground:", 0)
		msg(msg(sv, "contentView"), "setDrawsBackground:", 0)
	}
}

// SetGlassFieldFill keeps textBackgroundColor under the view's light appearance and clears it under a
// dark one, where it is an opaque (30,30,30) rectangle over the glass.
func SetGlassFieldFill(view *NSView) {
	v := uintptr(view.Id)
	fill := msg(class("NSColor"), "textBackgroundColor")
	if appearanceDark(msg(v, "effectiveAppearance")) {
		fill = msg(class("NSColor"), "clearColor")
	}
	msg(v, "setBackgroundColor:", fill)
}

// TextViewContentHeight is the height of the text wrapped at width plus 2*insetY, one line at least.
// A layout manager of its own measures it, so the view stays as it is.
func TextViewContentHeight(view *NSView, width, insetX, insetY float64) float64 {
	textViewInit()
	v := uintptr(view.Id)
	storage := msg(v, "textStorage")
	// The view's own layoutManager is not asked: that moves an NSTextView from TextKit 2 to TextKit 1.
	lm := NewNSLayoutManagerOverload1(int64(msg(msg(class("NSLayoutManager"), "alloc"), "init")))
	defer msg(uintptr(lm.Id), "release")
	h := 0.0
	if font := msg(v, "font"); font != 0 {
		h = tvFloat(uintptr(lm.Id), sel("defaultLineHeightForFont:"), font)
	}
	if msg(storage, "length") != 0 {
		pad := tvFloat(msg(v, "textContainer"), sel("lineFragmentPadding"), 0) // no argument: the third register is ignored
		tc := NewNSTextContainerOverload1(int64(msg(class("NSTextContainer"), "alloc"))).InitWithContainerSize(NSSize{Width: math.Max(width-2*insetX, 1), Height: 1e7})
		tc.SetLineFragmentPadding(pad)
		lm.AddTextContainer(tc)
		msg(storage, "addLayoutManager:", uintptr(lm.Id))
		lm.GlyphRangeForTextContainer(tc)
		h = math.Max(h, lm.UsedRectForTextContainer(tc).Height)
		msg(storage, "removeLayoutManager:", uintptr(lm.Id))
		msg(uintptr(tc.Id), "release")
	}
	return math.Ceil(h + 2*insetY)
}
