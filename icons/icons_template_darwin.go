package icons

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// markTemplate sets NSImage isTemplate. swt.Image keeps one NSImage for its life and swaps only its
// representations when the zoom changes, so the flag stays.
func markTemplate(img *swt.Image) {
	cocoa.Msg(img.Handle.Id, "setTemplate:", 1)
}

func isTemplate(img *swt.Image) bool {
	return cocoa.Msg(img.Handle.Id, "isTemplate")&0xff != 0
}
