package icons

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func systemIcon(a *gowt.App, e iconEntry, px int, c gowt.RGB) *swt.ImageData {
	alpha := cocoa.SymbolAlpha(e.sf, px)
	if alpha == nil {
		return nil
	}
	return swt.NewImageDataFromNRGBA(tintAlpha(px, c, func(x, y int) uint8 { return alpha[y*px+x] }))
}
