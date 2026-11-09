//go:build linux

package cairo

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// matrix is cairo_matrix_t: x' = xx*x + xy*y + x0, y' = yx*x + yy*y + y0, the order of gfx.Affine.
type matrix struct{ xx, yx, xy, yy, x0, y0 float64 }

type pangoRect struct{ x, y, w, h int32 }

const (
	formatARGB32 = 0
	formatA8     = 2

	operatorClear  = 0
	operatorSource = 1
	operatorOver   = 2

	extendPad = 3

	antialiasGray  = 2
	hintStyleNone  = 1
	hintMetricsOff = 1

	fillRuleWinding = 0
	fillRuleEvenOdd = 1

	filterFast     = 0
	filterGood     = 1
	filterBilinear = 4
)

var (
	bindOnce sync.Once
	bindErr  error
)

// The cairo, pango and fontconfig entry points the renderer calls; contexts, surfaces
// and layouts travel as raw uintptr references. bind in bind_linux.go (purego) or
// bind_cgo_linux.go (opt-in cgo backend, build tag gowtcgo) fills them.
var (
	imageSurfaceCreate    func(format int32, w, h int32) uintptr
	imageSurfaceGetData   func(s uintptr) uintptr
	imageSurfaceGetStride func(s uintptr) int32
	surfaceDestroy        func(s uintptr)
	surfaceReference      func(s uintptr) uintptr
	surfaceFlush          func(s uintptr)
	surfaceMarkDirty      func(s uintptr)
	create                func(s uintptr) uintptr
	destroy               func(cr uintptr)
	save, restore         func(cr uintptr)
	translate, scale      func(cr uintptr, a, b float64)
	rotate                func(cr uintptr, a float64)
	transform             func(cr uintptr, m *matrix)
	getMatrix             func(cr uintptr, m *matrix)
	setMatrix             func(cr uintptr, m *matrix)
	identityMatrix        func(cr uintptr)
	newPath, closePath    func(cr uintptr)
	moveTo, lineTo        func(cr uintptr, x, y float64)
	curveTo               func(cr uintptr, x1, y1, x2, y2, x3, y3 float64)
	rectangle             func(cr uintptr, x, y, w, h float64)
	setSourceRGBA         func(cr uintptr, r, g, b, a float64)
	setSource             func(cr uintptr, p uintptr)
	setSourceSurface      func(cr uintptr, s uintptr, x, y float64)
	setFillRule           func(cr uintptr, r int32)
	setOperator           func(cr uintptr, op int32)
	fill, stroke, clip    func(cr uintptr)
	paint                 func(cr uintptr)
	paintWithAlpha        func(cr uintptr, a float64)
	maskSurface           func(cr uintptr, s uintptr, x, y float64)
	setLineWidth          func(cr uintptr, w float64)
	setLineCap            func(cr uintptr, c int32)
	setLineJoin           func(cr uintptr, j int32)
	setMiterLimit         func(cr uintptr, m float64)
	setDash               func(cr uintptr, d *float64, n int32, off float64)
	pushGroup             func(cr uintptr)
	popGroupToSource      func(cr uintptr)
	patternCreateLinear   func(x0, y0, x1, y1 float64) uintptr
	patternCreateRadial   func(x0, y0, r0, x1, y1, r1 float64) uintptr
	patternAddColorStop   func(p uintptr, off, r, g, b, a float64)
	patternSetExtend      func(p uintptr, e int32)
	patternDestroy        func(p uintptr)
	getSource             func(cr uintptr) uintptr
	patternSetFilter      func(p uintptr, f int32)
	fontOptionsCreate     func() uintptr
	pdfCreateForStream    func(write uintptr, closure uintptr, w, h float64) uintptr
	pdfSetSize            func(s uintptr, w, h float64)
	showPage              func(cr uintptr)
	surfaceFinish         func(s uintptr)
	surfaceStatus         func(s uintptr) int32
	fontOptionsSetAA      func(o uintptr, v int32)
	fontOptionsSetHint    func(o uintptr, v int32)
	fontOptionsSetMetrics func(o uintptr, v int32)

	fmNew               func() uintptr
	fmCreateContext     func(fm uintptr) uintptr
	ctxSetFontOptions   func(ctx, opts uintptr)
	ctxUpdate           func(cr, ctx uintptr)
	layoutNew           func(ctx uintptr) uintptr
	layoutSetText       func(l uintptr, s string, n int32)
	layoutSetFont       func(l, desc uintptr)
	layoutBaseline      func(l uintptr) int32
	layoutExtents       func(l uintptr, ink, logical *pangoRect)
	showLayout          func(cr, l uintptr)
	fdNew               func() uintptr
	fdSetFamily         func(d uintptr, f string)
	fdSetWeight         func(d uintptr, w int32)
	fdSetStyle          func(d uintptr, s int32)
	fdSetAbsSize        func(d uintptr, s float64)
	attrListNew         func() uintptr
	attrListInsert      func(list, attr uintptr)
	attrListUnref       func(list uintptr)
	attrLetterSpacing   func(units int32) uintptr
	layoutSetAttrs      func(l, list uintptr)
	objectUnref         func(o uintptr)
	fcPatternCreate     func() uintptr
	fcPatternDestroy    func(p uintptr)
	fcPatternAddString  func(p uintptr, obj, s string) int32
	fcConfigSubstitute  func(cfg, p uintptr, kind int32) int32
	fcDefaultSubstitute func(p uintptr)
	fcFontMatch         func(cfg, p uintptr, res *int32) uintptr
	fcPatternGetString  func(p uintptr, obj string, n int32, out *uintptr) int32
	fcPatternGetInteger func(p uintptr, obj string, n int32, out *int32) int32
	fcAppFontAddDir     func(cfg uintptr, dir string) int32
	fcAppFontAddFile    func(cfg uintptr, file string) int32

	// newer than the rest, nil when the library lacks them
	ctxSetRound     func(ctx uintptr, v int32)
	fcConfigChanged func(fm uintptr)
)

func open(name string) (uintptr, error) {
	return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

func ptr(a uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&a)) }

func cstr(a uintptr) string {
	if a == 0 {
		return ""
	}
	p := ptr(a)
	n := 0
	for *(*byte)(unsafe.Add(p, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(p), n))
}
