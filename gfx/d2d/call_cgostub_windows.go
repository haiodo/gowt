//go:build windows && cgo && gowtcgo

package d2d

// Test stubs for the cgo dispatcher (call_cgo_windows_test.go): Go does not allow import "C"
// in _test.go files, so the C functions and their address accessors live here and the test
// calls the pure-Go accessors.

/*
#include <stdint.h>
#include <string.h>

static uint64_t bits(double v) { uint64_t r; memcpy(&r, &v, 8); return r; }

// sig I,F,F (the ID2D1DeviceContext SetDpi shape).
uint64_t gowtTestSumFF(uintptr_t self, float a, float b) { return bits((double)a + (double)b); }

// sig I,F,I,F,I (the IDWriteTextLayout1 SetCharacterSpacing shape).
uint64_t gowtTestMixed(uintptr_t self, float a, uintptr_t i, float b, uintptr_t j) {
	return bits((double)a*10 + (double)b + (double)(i + j));
}

typedef struct { float x, y; } wtPt2T;

// sig I,P (the geometry sink BeginFigure shape).
uint64_t gowtTestPt(uintptr_t self, wtPt2T p) { return bits((double)p.x + (double)p.y); }

static uintptr_t gowtTestAddrSumFF(void) { return (uintptr_t)gowtTestSumFF; }
static uintptr_t gowtTestAddrMixed(void) { return (uintptr_t)gowtTestMixed; }
static uintptr_t gowtTestAddrPt(void) { return (uintptr_t)gowtTestPt; }
*/
import "C"

func testAddrSumFF() uintptr { return uintptr(C.gowtTestAddrSumFF()) }

func testAddrMixed() uintptr { return uintptr(C.gowtTestAddrMixed()) }

func testAddrPt() uintptr { return uintptr(C.gowtTestAddrPt()) }
