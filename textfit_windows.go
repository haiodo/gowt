package gowt

import (
	"unicode/utf16"

	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/internal/win32"
	"github.com/haiodo/gowt/swt"
)

const emSetRect = 0xb3 // EM_SETRECT

// nativeInset sets the formatting rectangle of the edit to its client area less the inset. The edit
// resets it when it is resized, so a Resize listener, added once, sets it again.
func nativeInset(t *swt.Text, x, y int) {
	const key = "gowt.inset"
	in, _ := t.GetDataKey(key).(*[2]int)
	if in == nil {
		in = new([2]int)
		t.SetDataKeyValue(key, in)
		t.AddControlListener(swt.ControlListenerControlResizedAdapter(func(*swt.ControlEvent) { setFormatRect(t, in[0], in[1]) }))
	}
	*in = [2]int{x, y}
	setFormatRect(t, x, y)
}

func setFormatRect(t *swt.Text, x, y int) {
	z := t.GetAutoscalingZoom()
	dx, dy := swt.DPIUtilPointToPixel(int32(x), z), swt.DPIUtilPointToPixel(int32(y), z)
	r := win32.NewRECT()
	win32.OSGetClientRect(t.Handle, r)
	r.Left, r.Top, r.Right, r.Bottom = r.Left+dx, r.Top+dy, r.Right-dx, r.Bottom-dy
	win32.OSSendMessageOverload13(t.Handle, emSetRect, 0, r)
}

// nativeContentHeight measures the text with DrawText, wrapped at width less the horizontal inset.
func nativeContentHeight(t *swt.Text, width, x, y int) int {
	z := t.GetAutoscalingZoom()
	hdc := win32.OSGetDC(t.Handle)
	defer win32.OSReleaseDC(t.Handle, hdc)
	old := win32.OSSelectObject(hdc, win32.OSSendMessage(t.Handle, win32.OSWM_GETFONT, 0, nil))
	defer win32.OSSelectObject(hdc, old)
	s := t.GetText()
	// DrawText leaves out the line after a final line break, and measures an empty string as no height.
	if s == "" || s[len(s)-1] == '\n' {
		s += " "
	}
	r := win32.NewRECT()
	r.Right = swt.DPIUtilPointToPixel(int32(width-2*x), z)
	u := append(utf16.Encode([]rune(s)), 0)
	win32.OSDrawTextHDCLpStringNCountLpRectUFormat(hdc, u, -1, r, win32.OSDT_CALCRECT|win32.OSDT_WORDBREAK|win32.OSDT_EDITCONTROL|win32.OSDT_NOPREFIX)
	return int(swt.DPIUtilPixelToPoint(r.Bottom-r.Top, z)) + 2*y + fluent.ExtraHeight(t)
}

func nativeWidth(t *swt.Text) int { return int(t.GetClientArea().Width) }
