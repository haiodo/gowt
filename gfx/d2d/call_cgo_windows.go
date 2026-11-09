//go:build windows && cgo && gowtcgo

// Opt-in cgo backend for the COM call path: instead of packing the arguments into uintptr
// registers for syscall.SyscallN (and the arm64 float thunk), the arguments cross into C with
// their kinds, and one typed C call per signature dispatches them - floats land in the float
// registers, 2-float structs travel as one value. The default backend stays call_windows_*.go.

package d2d

/*
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// The Go side of call packs into this layout (see cgoCallArgs).
typedef struct {
	uint32_t sig;
	uint32_t pad;
	uintptr_t x[12];
	double f[12];
} wtArgs;

// A 2-float struct argument (D2D1_POINT_2F and friends): one 8-byte slot on amd64, two float
// registers on arm64 - the C compiler sorts the ABI out from the struct type.
typedef struct { float x, y; } wtPt2;
static wtPt2 wtPt2Bits(uintptr_t v) { wtPt2 p; memcpy(&p, &v, 8); return p; }

// sig encodes (argument count, per-argument kind): sig = len, then sig = sig*4 + kind per
// argument, kInt = 0, kF32 = 1, kPt2 = 2. x[] holds the integer (and packed point) arguments
// in order, f[] the float arguments. Every signature vcall is called with must have a case
// here - the default aborts so a new call shape is noticed at development time.
uint64_t gowtVCall(uintptr_t fn, uintptr_t vargs) {
	const wtArgs *a = (const wtArgs *)vargs;
	const uintptr_t *x = a->x;
	const double *f = a->f;
	#define WP(i) ((void *)x[(i)])
	#define K1(base, d) ((base) * 4 + (d))
	switch (a->sig) {
	case K1(1u, 0):
		return ((uint64_t (*)(void *))fn)(WP(0));
	case K1(K1(2u, 0), 0):
		return ((uint64_t (*)(void *, void *))fn)(WP(0), WP(1));
	case K1(K1(2u, 0), 2):
		return ((uint64_t (*)(void *, wtPt2))fn)(WP(0), wtPt2Bits(x[1]));
	case K1(K1(K1(3u, 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *))fn)(WP(0), WP(1), WP(2));
	case K1(K1(K1(3u, 0), 1), 1):
		return ((uint64_t (*)(void *, float, float))fn)(WP(0), (float)f[0], (float)f[1]);
	case K1(K1(K1(3u, 0), 2), 0):
		return ((uint64_t (*)(void *, wtPt2, void *))fn)(WP(0), wtPt2Bits(x[1]), WP(2));
	case K1(K1(K1(K1(4u, 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *))fn)(WP(0), WP(1), WP(2), WP(3));
	case K1(K1(K1(K1(K1(5u, 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *))fn)(WP(0), WP(1), WP(2), WP(3), WP(4));
	case K1(K1(K1(K1(K1(5u, 0), 0), 0), 1), 0):
		return ((uint64_t (*)(void *, void *, void *, float, void *))fn)(WP(0), WP(1), WP(2), (float)f[0], WP(3));
	case K1(K1(K1(K1(K1(5u, 0), 1), 0), 1), 0):
		return ((uint64_t (*)(void *, float, void *, float, void *))fn)(WP(0), (float)f[0], WP(1), (float)f[1], WP(2));
	case K1(K1(K1(K1(K1(5u, 0), 1), 1), 1), 0):
		return ((uint64_t (*)(void *, float, float, float, void *))fn)(WP(0), (float)f[0], (float)f[1], (float)f[2], WP(1));
	case K1(K1(K1(K1(K1(5u, 0), 2), 0), 0), 0):
		return ((uint64_t (*)(void *, wtPt2, void *, void *, void *))fn)(WP(0), wtPt2Bits(x[1]), WP(2), WP(3), WP(4));
	case K1(K1(K1(K1(K1(K1(6u, 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *))fn)(WP(0), WP(1), WP(2), WP(3), WP(4), WP(5));
	case K1(K1(K1(K1(K1(K1(6u, 0), 0), 0), 1), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, float, void *, void *))fn)(WP(0), WP(1), WP(2), (float)f[0], WP(3), WP(4));
	case K1(K1(K1(K1(K1(K1(K1(7u, 0), 0), 0), 0), 0), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, void *))fn)(WP(0), WP(1), WP(2), WP(3), WP(4), WP(5), WP(6));
	case K1(K1(K1(K1(K1(K1(K1(7u, 0), 0), 0), 0), 1), 1), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, float, float, void *))fn)(WP(0), WP(1), WP(2), WP(3), (float)f[0], (float)f[1], WP(4));
	case K1(K1(K1(K1(K1(K1(K1(K1(K1(9u, 0), 0), 0), 0), 0), 0), 1), 0), 0):
		return ((uint64_t (*)(void *, void *, void *, void *, void *, void *, float, void *, void *))fn)(WP(0), WP(1), WP(2), WP(3), WP(4), WP(5), (float)f[0], WP(6), WP(7));
	}
	fprintf(stderr, "gfx/d2d: gowtVCall: unhandled signature %u\n", a->sig);
	abort();
}
*/
import "C"

import (
	"math"
	"runtime"
	"unsafe"
)

// cgoCallArgs mirrors wtArgs in the preamble; only a pointer crosses the boundary.
type cgoCallArgs struct {
	sig uint32
	pad uint32
	x   [12]uintptr
	f   [12]float64
}

func call(fn uintptr, args ...arg) uintptr {
	var a cgoCallArgs
	a.sig = uint32(len(args))
	ni, nf := 0, 0
	for _, v := range args {
		switch v.kind {
		case kInt, kPt2:
			a.sig = a.sig*4 + uint32(v.kind)
			a.x[ni] = v.v
			ni++
		case kF32:
			a.sig = a.sig*4 + 1
			a.f[nf] = float64(math.Float32frombits(uint32(v.v)))
			nf++
		}
	}
	// The block goes across as an integer: a Go-pointer argument would escape it
	// to the heap per call; there are no callbacks on this path, so the stack block is
	// stable for the duration of the call.
	r := uint64(C.gowtVCall(C.uintptr_t(fn), C.uintptr_t(uintptr(unsafe.Pointer(&a)))))
	runtime.KeepAlive(args)
	runtime.KeepAlive(&a)
	return uintptr(r)
}
