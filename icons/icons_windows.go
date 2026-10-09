package icons

import (
	"sync"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

// Segoe Fluent Icons ships with Windows 11, Segoe MDL2 Assets with Windows 10; the code points
// in the dictionary exist in both. Neither font is embedded.
func iconFont(a *gowt.App) string {
	iconFontOnce.Do(func() { iconFontName = findIconFont(a) })
	return iconFontName
}

var (
	iconFontOnce sync.Once
	iconFontName string
)

func findIconFont(a *gowt.App) string {
	for _, f := range []string{"Segoe Fluent Icons", "Segoe MDL2 Assets"} {
		if len(a.Unwrap().GetFontList(f, true)) > 0 {
			return f
		}
	}
	return ""
}

// systemIcon draws the glyph white on black into an image and reads the brightness back as the
// alpha mask, because a GDI bitmap has no alpha channel.
func systemIcon(a *gowt.App, e iconEntry, px int, c gowt.RGB) *swt.ImageData {
	face := iconFont(a)
	if face == "" || px <= 0 {
		return nil
	}
	img := swt.NewImageDeviceWidthHeight(a.Unwrap(), int32(px), int32(px))
	defer img.Dispose()
	// The image is in device pixels, so the 96 dpi point size is px*3/4.
	font := swt.NewFontDeviceNameHeightStyle(a.Unwrap(), face, int32(max(1, px*3/4)), swt.NORMAL)
	defer font.Dispose()
	gc := swt.NewGCDrawable(img)
	defer gc.Dispose()
	gc.SetBackground(a.Unwrap().GetSystemColor(swt.COLOR_BLACK))
	gc.SetForeground(a.Unwrap().GetSystemColor(swt.COLOR_WHITE))
	gc.SetFont(font)
	gc.FillRectangle(0, 0, int32(px), int32(px))
	s := string(e.fluent)
	ext := gc.TextExtent(s)
	gc.DrawText(s, (int32(px)-ext.X)/2, (int32(px)-ext.Y)/2)

	data := img.GetImageData()
	return swt.NewImageDataFromNRGBA(tintAlpha(px, c, func(x, y int) uint8 {
		rgb := data.Palette.GetRGB(data.GetPixel(int32(x), int32(y)))
		return uint8((rgb.Red + rgb.Green + rgb.Blue) / 3)
	}))
}
