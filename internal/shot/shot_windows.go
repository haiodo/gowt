// Package shot captures a Shell's window for the snapshot commands (cmd/controlexample -snap, cmd/images).
package shot

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
	"unsafe"
)

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	gdi32      = syscall.NewLazyDLL("gdi32.dll")
	ntdll      = syscall.NewLazyDLL("ntdll.dll")
	uxtheme    = syscall.NewLazyDLL("uxtheme.dll")
	dwmapi     = syscall.NewLazyDLL("dwmapi.dll")
	dwmGetAttr = dwmapi.NewProc("DwmGetWindowAttribute")
	getRect    = user32.NewProc("GetWindowRect")
)

const (
	srccopy    = 0x00CC0020
	captureblt = 0x40000000 // layered windows too
)

type rect struct{ L, T, R, B int32 }

type bmi struct {
	Size                   uint32
	W, H                   int32
	Planes, BitCount       uint16
	Compression, SizeImage uint32
	XPels, YPels           int32
	ClrUsed, ClrImportant  uint32
}

// Capture writes the whole window hwnd as a PNG: title bar and DWM frame, without the invisible resize border
// (DWMWA_EXTENDED_FRAME_BOUNDS), and returns where it lies on the screen, in pixels. It copies that rectangle from the screen, so the window must be on top and
// on screen; the screen copy also holds the WebView2 child windows. GOWT_SNAP_FULL=1 uses PrintWindow
// (PW_RENDERFULLCONTENT) instead, which renders Mica and acrylic backdrops wrongly.
func Capture(hwnd int64, path string) (Frame, error) {
	var wr, fr rect
	getRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
	// DWMWA_EXTENDED_FRAME_BOUNDS is 9; it fails without DWM, then the window rect is the frame.
	if r, _, _ := dwmGetAttr.Call(uintptr(hwnd), 9, uintptr(unsafe.Pointer(&fr)), unsafe.Sizeof(fr)); r != 0 {
		fr = wr
	}
	fw, fh := int(fr.R-fr.L), int(fr.B-fr.T)
	if fw <= 0 || fh <= 0 {
		return Frame{}, fmt.Errorf("empty window")
	}
	user32.NewProc("UpdateWindow").Call(uintptr(hwnd))
	if Passive {
		const hwndTop, noSizeMoveActivate = 0, 0x0001 | 0x0002 | 0x0010
		user32.NewProc("SetWindowPos").Call(uintptr(hwnd), hwndTop, 0, 0, 0, 0, noSizeMoveActivate)
	} else {
		user32.NewProc("BringWindowToTop").Call(uintptr(hwnd))
		user32.NewProc("SetForegroundWindow").Call(uintptr(hwnd))
	}
	full := os.Getenv("GOWT_SNAP_FULL") == "1"
	ww, wh := fw, fh
	src, sx, sy := uintptr(0), int(fr.L), int(fr.T)
	if full {
		ww, wh = int(wr.R-wr.L), int(wr.B-wr.T)
		src = uintptr(hwnd)
		sx, sy = 0, 0
	}
	dc, _, _ := user32.NewProc("GetWindowDC").Call(src)
	if dc == 0 {
		return Frame{}, fmt.Errorf("GetWindowDC failed")
	}
	defer user32.NewProc("ReleaseDC").Call(src, dc)
	mem, _, _ := gdi32.NewProc("CreateCompatibleDC").Call(dc)
	if mem == 0 {
		return Frame{}, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer gdi32.NewProc("DeleteDC").Call(mem)
	bmp, _, _ := gdi32.NewProc("CreateCompatibleBitmap").Call(dc, uintptr(ww), uintptr(wh))
	if bmp == 0 {
		return Frame{}, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer gdi32.NewProc("DeleteObject").Call(bmp)
	old, _, _ := gdi32.NewProc("SelectObject").Call(mem, bmp)
	defer gdi32.NewProc("SelectObject").Call(mem, old)
	if full {
		if r, _, _ := user32.NewProc("PrintWindow").Call(uintptr(hwnd), mem, 2); r == 0 {
			return Frame{}, fmt.Errorf("PrintWindow failed")
		}
	} else if r, _, _ := gdi32.NewProc("BitBlt").Call(mem, 0, 0, uintptr(ww), uintptr(wh), dc, uintptr(sx), uintptr(sy), srccopy|captureblt); r == 0 {
		return Frame{}, fmt.Errorf("BitBlt failed")
	}
	hdr := bmi{Size: uint32(unsafe.Sizeof(bmi{})), W: int32(ww), H: -int32(wh), Planes: 1, BitCount: 32}
	buf := make([]byte, ww*wh*4)
	// GetDIBits must not see the bitmap selected into a DC: some display drivers crash on it.
	gdi32.NewProc("SelectObject").Call(mem, old)
	if r, _, _ := gdi32.NewProc("GetDIBits").Call(mem, bmp, 0, uintptr(wh), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&hdr)), 0); r == 0 {
		return Frame{}, fmt.Errorf("GetDIBits failed")
	}
	ox, oy := 0, 0
	if full {
		ox, oy = int(fr.L-wr.L), int(fr.T-wr.T)
	}
	dpi, _, _ := user32.NewProc("GetDpiForWindow").Call(uintptr(hwnd))
	radius := 8 * int(dpi) / 96 // Windows 11 rounds the window corners; the screen copy holds the desktop behind them
	img := image.NewNRGBA(image.Rect(0, 0, fw, fh))
	for y := 0; y < fh; y++ {
		for x := 0; x < fw; x++ {
			si, dst := ((y+oy)*ww+x+ox)*4, img.PixOffset(x, y)
			img.Pix[dst], img.Pix[dst+1], img.Pix[dst+2], img.Pix[dst+3] = buf[si+2], buf[si+1], buf[si], 255
			if outsideCorner(x, y, fw, fh, radius) {
				img.Pix[dst+3] = 0
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return Frame{}, err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return Frame{}, err
	}
	return Frame{float64(fr.L), float64(fr.T), float64(fw), float64(fh)}, nil
}

// outsideCorner reports whether the pixel lies outside the rounded corner arc of a w x h window.
func outsideCorner(x, y, w, h, r int) bool {
	cx, cy := x, y
	switch {
	case x < r && y < r:
		cx, cy = r, r
	case x >= w-r && y < r:
		cx, cy = w-r-1, r
	case x < r && y >= h-r:
		cx, cy = r, h-r-1
	case x >= w-r && y >= h-r:
		cx, cy = w-r-1, h-r-1
	default:
		return false
	}
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy > r*r
}

// Meta is what the pixels depend on besides the code: snapcheck compares it before comparing images.
// Wine's version is part of it: its themed drawing differs between releases and from real Windows.
func Meta(hwnd int64) string {
	var vi [37]uint32 // OSVERSIONINFOEXW: size, major, minor, build, platform, ...
	vi[0] = uint32(unsafe.Sizeof(vi))
	ntdll.NewProc("RtlGetVersion").Call(uintptr(unsafe.Pointer(&vi)))
	env := "windows"
	if p := ntdll.NewProc("wine_get_version"); p.Find() == nil {
		v, _, _ := p.Call()
		env = "wine " + cString(v)
	}
	dpi, _, _ := user32.NewProc("GetDpiForWindow").Call(uintptr(hwnd))
	themed, _, _ := uxtheme.NewProc("IsThemeActive").Call()
	w, _, _ := user32.NewProc("GetSystemMetrics").Call(0)
	h, _, _ := user32.NewProc("GetSystemMetrics").Call(1)
	return fmt.Sprintf("os=windows\nenv=%s\nwindows=%d.%d.%d\ndpi=%d\ntheme=%v\nscreen=%dx%d\n", env, vi[1], vi[2], vi[3], dpi, themed != 0, w, h)
}

func cString(p uintptr) string {
	var b []byte
	for ; ; p++ {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(nil), p))
		if c == 0 {
			return string(b)
		}
		b = append(b, c)
	}
}
