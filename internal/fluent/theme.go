//go:build windows

package fluent

import (
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/haiodo/gowt/swt"
)

// RGB is an opaque colour.
type RGB struct{ R, G, B uint8 }

// over is c at opacity a on top of bg.
func (c RGB) over(a float64, bg RGB) RGB {
	m := func(f, b uint8) uint8 { return uint8(float64(f)*a + float64(b)*(1-a) + 0.5) }
	return RGB{m(c.R, bg.R), m(c.G, bg.G), m(c.B, bg.B)}
}

// Palette holds the WinUI 3 colours of the light or the dark theme and the system accent.
type Palette struct {
	Dark     bool
	Accent   RGB
	OnAccent RGB
	Text     RGB
	Disabled RGB
	Fill     RGB // input surface
	Black    RGB // the colour that tints light surfaces: black on light, white on dark
}

const darkContentKey = "org.eclipse.swt.internal.win32.useDarkModeExplorerTheme"

var (
	accentMu    sync.Mutex
	accentAt    time.Time
	accentCache RGB
)

// accent reads the colour of Settings > Personalisation > Colours from the DWM key, which the
// system keeps for the current user; UISettings needs WinRT. The blue of Windows 11 is the fallback.
func accent(dark bool) RGB {
	accentMu.Lock()
	defer accentMu.Unlock()
	if time.Since(accentAt) > 2*time.Second {
		accentAt = time.Now()
		accentCache = RGB{0x00, 0x67, 0xC0}
		if dark {
			accentCache = RGB{0x4C, 0xC2, 0xFF}
		}
		key, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\DWM`)
		val, _ := syscall.UTF16PtrFromString("AccentColor")
		var v, size uint32 = 0, 4
		const hkcu, rrfDword = 0x80000001, 0x10
		if call(pRegGetValue, hkcu, uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(val)), rrfDword, 0,
			uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&size))) == 0 {
			// 0xAABBGGRR
			a := RGB{uint8(v), uint8(v >> 8), uint8(v >> 16)}
			if dark {
				a = RGB{255, 255, 255}.over(0.25, a)
			}
			accentCache = a
		}
	}
	return accentCache
}

// luminance is the relative brightness 0..1 of c.
func luminance(c RGB) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

// Current returns the palette for the content of d: dark when look.SetDarkContent turned it on.
func Current(d *swt.Display) Palette {
	dark, _ := d.GetData(darkContentKey).(bool)
	p := Palette{Dark: dark, Accent: accent(dark)}
	if luminance(p.Accent) > 0.6 {
		p.OnAccent = RGB{0, 0, 0}
	} else {
		p.OnAccent = RGB{255, 255, 255}
	}
	if dark {
		p.Text, p.Disabled, p.Fill, p.Black = RGB{255, 255, 255}, RGB{0x8A, 0x8A, 0x8A}, RGB{0x2D, 0x2D, 0x2D}, RGB{255, 255, 255}
	} else {
		p.Text, p.Disabled, p.Fill, p.Black = RGB{0x1A, 0x1A, 0x1A}, RGB{0x8A, 0x8A, 0x8A}, RGB{255, 255, 255}, RGB{0, 0, 0}
	}
	return p
}

// Layer is the colour of the window background in WinUI 3: F3F3F3 light, 202020 dark.
func (p Palette) Layer() RGB {
	if p.Dark {
		return RGB{0x20, 0x20, 0x20}
	}
	return RGB{0xF3, 0xF3, 0xF3}
}

// Card is the fill of a card on the window background: white at 70% in the light theme, white at 5% in
// the dark one (CardBackgroundFillColorDefault).
func (p Palette) Card() RGB {
	if p.Dark {
		return RGB{255, 255, 255}.over(0.05, p.Layer())
	}
	return RGB{255, 255, 255}.over(0.70, p.Layer())
}

// Background gives the window the Fluent background; labels, groups and tab pages inherit it, inputs
// and tables keep their own. Windows 11 has no system colour for it, GetSysColor still answers F0F0F0.
func Background(c *swt.Composite) {
	l := Current(c.GetDisplay()).Layer()
	c.SetBackgroundWithColor(swt.NewColorRedGreenBlue(int32(l.R), int32(l.G), int32(l.B)))
	c.SetBackgroundMode(swt.INHERIT_DEFAULT)
}

// Stroke is the hairline of a surface: 12% of the tint over bg.
func (p Palette) Stroke(bg RGB) RGB { return p.Black.over(0.12, bg) }

// StrongStroke is the bottom edge of an input at rest, WinUI's ControlStrongStroke.
func (p Palette) StrongStroke(bg RGB) RGB { return p.Black.over(0.45, bg) }

// parentBackground is the colour behind c: the background of its parent.
func parentBackground(c *swt.Control) RGB {
	p := c.GetParent()
	if p == nil {
		return Current(c.GetDisplay()).Layer()
	}
	col := p.GetBackground()
	rgb := col.GetRGB()
	return RGB{uint8(rgb.Red), uint8(rgb.Green), uint8(rgb.Blue)}
}
