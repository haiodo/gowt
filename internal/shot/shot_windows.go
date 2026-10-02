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
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
	getRect  = user32.NewProc("GetWindowRect")
	getCRect = user32.NewProc("GetClientRect")
	toScreen = user32.NewProc("ClientToScreen")
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

// WindowPNG writes the client area of the toplevel window hwnd as a PNG, copied from its window DC
// (the window must be on top and on screen).
func WindowPNG(hwnd int64, path string) error {
	var wr, cr rect
	pt := [2]int32{}
	getRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
	getCRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&cr)))
	toScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
	ww, wh := int(wr.R-wr.L), int(wr.B-wr.T)
	cw, ch := int(cr.R), int(cr.B)
	if cw <= 0 || ch <= 0 {
		return fmt.Errorf("empty client area")
	}
	user32.NewProc("UpdateWindow").Call(uintptr(hwnd))
	dc, _, _ := user32.NewProc("GetWindowDC").Call(uintptr(hwnd))
	defer user32.NewProc("ReleaseDC").Call(uintptr(hwnd), dc)
	mem, _, _ := gdi32.NewProc("CreateCompatibleDC").Call(dc)
	defer gdi32.NewProc("DeleteDC").Call(mem)
	bmp, _, _ := gdi32.NewProc("CreateCompatibleBitmap").Call(dc, uintptr(ww), uintptr(wh))
	defer gdi32.NewProc("DeleteObject").Call(bmp)
	gdi32.NewProc("SelectObject").Call(mem, bmp)
	// SRCCOPY. PrintWindow(PW_RENDERFULLCONTENT) left Wine blocked in the next DispatchMessage.
	if r, _, _ := gdi32.NewProc("BitBlt").Call(mem, 0, 0, uintptr(ww), uintptr(wh), dc, 0, 0, 0x00CC0020); r == 0 {
		return fmt.Errorf("BitBlt failed")
	}
	hdr := bmi{Size: uint32(unsafe.Sizeof(bmi{})), W: int32(ww), H: -int32(wh), Planes: 1, BitCount: 32}
	buf := make([]byte, ww*wh*4)
	if r, _, _ := gdi32.NewProc("GetDIBits").Call(mem, bmp, 0, uintptr(wh), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&hdr)), 0); r == 0 {
		return fmt.Errorf("GetDIBits failed")
	}
	ox, oy := int(pt[0]-wr.L), int(pt[1]-wr.T)
	img := image.NewNRGBA(image.Rect(0, 0, cw, ch))
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			src, dst := ((y+oy)*ww+x+ox)*4, img.PixOffset(x, y)
			img.Pix[dst], img.Pix[dst+1], img.Pix[dst+2], img.Pix[dst+3] = buf[src+2], buf[src+1], buf[src], 255
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
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
