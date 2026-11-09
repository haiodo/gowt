//go:build windows && gowtcgo && cgo

// Opt-in cgo backend for the GDI+ flat API calls (gp/gpOut/gpInt in gdip_manual.go): the
// float-tagged arguments cross into C with their kinds and one typed C call per signature
// dispatches them, so on arm64 the floats reach the float registers without the
// gdipFloatThunk assembly (excluded in this build), and on amd64 no SyscallN packing runs.
// The default backends stay gdip_call_windows_{amd64,arm64}.go.

package win32

/*
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

// The Go side of gdipCall packs into this layout (see gdipCallArgs); f32-tagged arguments
// become float, everything else a pointer-sized integer.
typedef struct {
	uint32_t sig;
	uint32_t pad;
	uintptr_t x[16];
	double f[12];
} gdipArgs;

// sig encodes (argument count, per-argument kind): sig = len, then sig = sig*4 + kind per
// argument, int = 0, float32 = 1. x[] holds the integer arguments in order, f[] the floats.
// Every gp call shape must have a case here - the default aborts so a new shape is noticed
// at development time.
uint64_t gowtGdipCall(uintptr_t fn, uintptr_t vargs) {
	const gdipArgs *a = (const gdipArgs *)vargs;
	const uintptr_t *x = a->x;
	const double *f = a->f;
	#define GP(i) ((void *)x[(i)])
	#define K1(base, d) ((base) * 4 + (d))
	switch (a->sig) {
	case 0u:
		return ((uint64_t (*)())fn)();
	case K1(1u, 0):
		return ((uint64_t (*)(void *))fn)(GP(0));
	case K1(K1(2u, 0), 0):
		return ((uint64_t (*)(void *, void *))fn)(GP(0), GP(1));
	case K1(K1(2u, 0), 1):
		return ((uint64_t (*)(void *, float))fn)(GP(0), (float)f[0]);
	case K1(K1(K1(3u, 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *))fn)(GP(0), GP(1), GP(2));
	case K1(K1(K1(3u, 0), 1), 0):
		return ((uint64_t (*)(void *, float, void *))fn)(GP(0), (float)f[0], GP(1));
	case K1(K1(K1(3u, 0), 0), 1):
		return ((uint64_t (*)(void *, void *, float))fn)(GP(0), GP(1), (float)f[0]);
	case K1(K1(K1(K1(4u, 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3));
	case K1(K1(K1(K1(4u, 0), 1), 0), 0):
		return ((uint64_t (*)(void *, float, void *, void *))fn)(GP(0), (float)f[0], GP(1), GP(2));
	case K1(K1(K1(K1(4u, 0), 1), 1), 0):
		return ((uint64_t (*)(void *, float, float, void *))fn)(GP(0), (float)f[0], (float)f[1], GP(1));
	case K1(K1(K1(K1(K1(5u, 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4));
	case K1(K1(K1(K1(K1(5u, 0), 1), 0), 0), 0):
		return ((uint64_t (*)(void *, float, void *, void *, void *))fn)(GP(0), (float)f[0], GP(1), GP(2), GP(3));
	case K1(K1(K1(K1(K1(5u, 0), 1), 1), 0), 0):
		return ((uint64_t (*)(void *, float, float, void *, void *))fn)(GP(0), (float)f[0], (float)f[1], GP(1), GP(2));
	case K1(K1(K1(K1(K1(5u, 0), 1), 1), 1), 1):
		return ((uint64_t (*)(void *, float, float, float, float))fn)(GP(0), (float)f[0], (float)f[1], (float)f[2], (float)f[3]);
	case K1(K1(K1(K1(K1(K1(6u, 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), GP(5));
	case K1(K1(K1(K1(K1(K1(6u, 0), 1), 1), 0), 0), 0):
		return ((uint64_t (*)(void *, float, float, void *, void *, void *))fn)(GP(0), (float)f[0], (float)f[1], GP(1), GP(2), GP(3));
	case K1(K1(K1(K1(K1(K1(K1(7u, 0), 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), GP(5), GP(6));
	case K1(K1(K1(K1(K1(K1(K1(7u, 0), 0), 1), 1), 1), 1), 0):
		return ((uint64_t (*)(void *, void *, float, float, float, float, void *))fn)(GP(0), GP(1), (float)f[0], (float)f[1], (float)f[2], (float)f[3], GP(2));
	case K1(K1(K1(K1(K1(K1(K1(7u, 0), 1), 1), 1), 1), 1), 1):
		return ((uint64_t (*)(void *, float, float, float, float, float, float))fn)(GP(0), (float)f[0], (float)f[1], (float)f[2], (float)f[3], (float)f[4], (float)f[5]);
	case K1(K1(K1(K1(K1(K1(K1(7u, 1), 1), 1), 1), 1), 1), 0):
		return ((uint64_t (*)(float, float, float, float, float, float, void *))fn)((float)f[0], (float)f[1], (float)f[2], (float)f[3], (float)f[4], (float)f[5], GP(0));
	case K1(K1(K1(K1(K1(K1(K1(K1(8u, 0), 0), 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), GP(5), GP(6), GP(7));
	case K1(K1(K1(K1(K1(K1(K1(K1(8u, 0), 0), 0), 0), 0), 0), 1), 1):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, float, float))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), GP(5), (float)f[0], (float)f[1]);
	case K1(K1(K1(K1(K1(K1(K1(K1(8u, 0), 0), 0), 0), 0), 1), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, float, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), (float)f[0], GP(5), GP(6));
	case K1(K1(K1(K1(K1(K1(K1(K1(K1(9u, 0), 0), 0), 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), GP(5), GP(6), GP(7), GP(8));
	case K1(K1(K1(K1(K1(K1(K1(K1(K1(9u, 0), 1), 1), 1), 1), 1), 1), 1), 1):
		return ((uint64_t (*)(void *, float, float, float, float, float, float, float, float))fn)(GP(0), (float)f[0], (float)f[1], (float)f[2], (float)f[3], (float)f[4], (float)f[5], (float)f[6], (float)f[7]);
	case K1(K1(K1(K1(K1(K1(K1(K1(K1(K1(K1(K1(K1(K1(14u, 0), 0), 0), 0), 0), 0), 0), 0), 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, void *, void *, void *, void *, void *, void *, void *, void *))fn)(GP(0), GP(1), GP(2), GP(3), GP(4), GP(5), GP(6), GP(7), GP(8), GP(9), GP(10), GP(11), GP(12), GP(13));
	}
	fprintf(stderr, "win32: gowtGdipCall: unhandled signature %u\n", a->sig);
	abort();
}
*/
import "C"

import (
	"math"
	"runtime"
	"unsafe"
)

// gdipCallArgs mirrors gdipArgs in the preamble; only a pointer crosses the boundary.
type gdipCallArgs struct {
	sig uint32
	pad uint32
	x   [16]uintptr
	f   [12]float64
}

func gdipCall(fn uintptr, args []uintptr) uintptr {
	var a gdipCallArgs
	a.sig = uint32(len(args))
	ni, nf := 0, 0
	for _, v := range args {
		if uint64(v)>>32 == f32Tag {
			a.sig = a.sig*4 + 1
			a.f[nf] = float64(math.Float32frombits(uint32(v)))
			nf++
		} else {
			a.sig *= 4
			a.x[ni] = v
			ni++
		}
	}
	// See gfx/d2d call_cgo_windows.go: an integer crosses, the block stays on the
	// stack; GDI+ flat calls made here carry no Go callbacks.
	r := uint64(C.gowtGdipCall(C.uintptr_t(fn), C.uintptr_t(uintptr(unsafe.Pointer(&a)))))
	runtime.KeepAlive(args)
	runtime.KeepAlive(&a)
	return uintptr(r)
}
