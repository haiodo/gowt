//go:build darwin

// Hand-written: the placeholder of a multi-line text view.
package cocoa

import (
	"math"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	placeholderOnce sync.Once
	phPoint         func(self, sel uintptr) NSPoint
	phSize          func(self, sel uintptr) NSSize
	phFloat         func(self, sel uintptr) float64
)

const lineBreakByClipping = 2 // NSLineBreakByClipping

// DrawTextViewPlaceholder draws s as AppKit draws a text field placeholder, where the first character
// of the empty view goes; NSTextView has no public one. Call it while the view draws (SWT Paint).
func DrawTextViewPlaceholder(view *NSView, s string) {
	v := uintptr(view.Id)
	if s == "" || msg(msg(v, "textStorage"), "length") != 0 || msg(v, "hasMarkedText")&0xff != 0 {
		return
	}
	placeholderOnce.Do(func() {
		objcInit()
		purego.RegisterFunc(&phPoint, objcMsgSend)
		purego.RegisterFunc(&phSize, objcMsgSend)
		purego.RegisterFunc(&phFloat, objcMsgSend)
	})
	origin := phPoint(v, sel("textContainerOrigin"))
	container := msg(v, "textContainer")
	pad := phFloat(container, sel("lineFragmentPadding"))
	bounds := view.Bounds()
	width := math.Min(phSize(container, sel("containerSize")).Width, bounds.Width-origin.X)
	para := msg(msg(msg(class("NSMutableParagraphStyle"), "alloc"), "init"), "autorelease")
	msg(para, "setAlignment:", msg(v, "alignment"))
	if msg(container, "widthTracksTextView")&0xff == 0 {
		msg(para, "setLineBreakMode:", lineBreakByClipping)
	}
	keys := [3]uintptr{uintptr(OSNSFontAttributeName()), uintptr(OSNSForegroundColorAttributeName()), uintptr(OSNSParagraphStyleAttributeName())}
	values := [3]uintptr{msg(v, "font"), msg(class("NSColor"), "placeholderTextColor"), para}
	attrs := msg(class("NSDictionary"), "dictionaryWithObjects:forKeys:count:", uintptr(unsafe.Pointer(&values[0])), uintptr(unsafe.Pointer(&keys[0])), 3)
	runtime.KeepAlive(keys)
	runtime.KeepAlive(values)
	str := msg(msg(msg(class("NSAttributedString"), "alloc"), "initWithString:attributes:", nsString(s), attrs), "autorelease")
	NewNSAttributedStringOverload1(int64(str)).DrawInRect(NSRect{
		X:      origin.X + pad,
		Y:      origin.Y,
		Width:  math.Max(width-2*pad, 0),
		Height: math.Max(bounds.Height-origin.Y, 0),
	})
}
