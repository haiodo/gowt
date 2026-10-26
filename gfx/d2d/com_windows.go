//go:build windows

package d2d

import (
	"fmt"
	"math"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const ptrSize = unsafe.Sizeof(uintptr(0))

const (
	kInt uint8 = iota
	kF32
	kPt2
)

// arg is one argument of a COM call: an integer or pointer, a float, or a struct of two floats.
type arg struct {
	kind uint8
	v    uintptr        // kF32: the bits; kPt2: x bits, then y bits above
	hold unsafe.Pointer // keeps the object behind a pointer argument alive until the call returns
}

func ui(v uintptr) arg { return arg{kInt, v, nil} }
func fl(v float64) arg { return arg{kF32, uintptr(math.Float32bits(float32(v))), nil} }
func pt2(x, y float64) arg {
	return arg{kPt2, uintptr(math.Float32bits(float32(x))) | uintptr(math.Float32bits(float32(y)))<<32, nil}
}

// ref passes the address of v. v is forced onto the heap: a stack object could move between the
// conversion to an integer and the call.
func ref[T any](v *T) arg {
	escape(v)
	return arg{kInt, uintptr(unsafe.Pointer(v)), unsafe.Pointer(v)}
}

// wstr passes s as a NUL terminated UTF-16 string.
func wstr(s string) arg { return ref(utf16Z(s)) }

// buf passes the elements of s, or NULL for an empty slice.
func buf[T any](s []T) arg {
	if len(s) == 0 {
		return ui(0)
	}
	return ref(&s[0])
}

var (
	never bool
	sink  any
)

//go:noinline
func escape(v any) {
	if never {
		sink = v
	}
}

// ptr converts an address coming from native code back to a pointer.
func ptr(a uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&a)) }

type guid struct {
	d1 uint32
	d2 uint16
	d3 uint16
	d4 [8]byte
}

func mkGUID(d1 uint32, d2, d3 uint16, d4 ...byte) *guid {
	g := &guid{d1: d1, d2: d2, d3: d3}
	copy(g.d4[:], d4)
	return g
}

var (
	iidD2DFactory1       = mkGUID(0xbb12d362, 0xdaee, 0x4b9a, 0xaa, 0x1d, 0x14, 0xba, 0x40, 0x1c, 0xfa, 0x1f)
	iidDXGIDevice        = mkGUID(0x54ec77fa, 0x1377, 0x44e6, 0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c)
	iidD3D11Multi        = mkGUID(0x9b7e4e00, 0x342c, 0x4106, 0xa1, 0x9f, 0x4f, 0x27, 0x04, 0xf6, 0x89, 0xf0)
	iidDWriteFactory5    = mkGUID(0x958db99a, 0xbe2a, 0x4f09, 0xaf, 0x7d, 0x65, 0x18, 0x98, 0x03, 0xd1, 0xd3)
	iidDWriteTextLayout1 = mkGUID(0x9064d822, 0x80a7, 0x465c, 0xa9, 0x86, 0xdf, 0x65, 0xf7, 0x8b, 0x8f, 0xeb)
	iidDWriteFactory     = mkGUID(0xb859ee5a, 0xd838, 0x4b5b, 0xa2, 0xe8, 0x1a, 0xdc, 0x7d, 0x93, 0xdb, 0x48)
)

var (
	d2d1      = syscall.NewLazyDLL("d2d1.dll")
	dwrite    = syscall.NewLazyDLL("dwrite.dll")
	d3d11     = syscall.NewLazyDLL("d3d11.dll")
	procD2D1F = d2d1.NewProc("D2D1CreateFactory")
	procDWF   = dwrite.NewProc("DWriteCreateFactory")
	procD3D   = d3d11.NewProc("D3D11CreateDevice")
)

func hrErr(what string, hr uintptr) error {
	return fmt.Errorf("gfx/d2d: %s failed: HRESULT 0x%08x", what, uint32(hr))
}

func failed(hr uintptr) bool { return int32(hr) < 0 }

// vcall calls slot of the COM object obj.
func vcall(obj uintptr, slot int, args ...arg) uintptr {
	vt := *(*uintptr)(ptr(obj))
	fn := *(*uintptr)(unsafe.Add(ptr(vt), uintptr(slot)*ptrSize))
	all := make([]arg, 0, len(args)+1)
	all = append(all, ui(obj))
	all = append(all, args...)
	return uintptr(uint32(call(fn, all...)))
}

func release(obj uintptr) {
	if obj != 0 {
		vcall(obj, 2)
	}
}

func queryInterface(obj uintptr, iid *guid) (uintptr, error) {
	var out uintptr
	if hr := vcall(obj, 0, ref(iid), ref(&out)); failed(hr) {
		return 0, hrErr("QueryInterface", hr)
	}
	return out, nil
}

// utf16Z is s as a NUL terminated UTF-16 string.
func utf16Z(s string) *uint16 {
	w := utf16.Encode([]rune(s))
	w = append(w, 0)
	return &w[0]
}

func utf16Slice(s string) []uint16 { return utf16.Encode([]rune(s)) }

func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	var w []uint16
	for i := uintptr(0); ; i += 2 {
		c := *(*uint16)(ptr(p + i))
		if c == 0 {
			break
		}
		w = append(w, c)
	}
	return string(utf16.Decode(w))
}
