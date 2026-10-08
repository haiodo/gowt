package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

func systemIcon(a *App, e iconEntry, px int, c RGB) *swt.ImageData {
	alpha := cocoa.SymbolAlpha(e.sf, px)
	if alpha == nil {
		return nil
	}
	return swt.NewImageDataFromNRGBA(tintAlpha(px, c, func(x, y int) uint8 { return alpha[y*px+x] }))
}
