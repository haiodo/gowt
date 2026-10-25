//go:build linux

package cairo

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// matrix is cairo_matrix_t: x' = xx*x + xy*y + x0, y' = yx*x + yy*y + y0, the order of gfx.Affine.
type matrix struct{ xx, yx, xy, yy, x0, y0 float64 }

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

	filterGood = 1
)

var (
	bindOnce sync.Once
	bindErr  error
)

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

type pangoRect struct{ x, y, w, h int32 }

func open(name string) (uintptr, error) {
	return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

// bind loads the libraries once. A system without them gets the error from NewRenderer.
func bind() (err error) {
	bindOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				bindErr = fmt.Errorf("gfx/cairo: %v", r)
			}
		}()
		var h [5]uintptr
		for i, n := range []string{"libcairo.so.2", "libpango-1.0.so.0", "libpangocairo-1.0.so.0", "libgobject-2.0.so.0", "libfontconfig.so.1"} {
			lib, e := open(n)
			if e != nil {
				bindErr = fmt.Errorf("gfx/cairo: %w", e)
				return
			}
			h[i] = lib
		}
		c, p, pc, gobj, fc := h[0], h[1], h[2], h[3], h[4]
		for _, b := range []struct {
			lib uintptr
			fns map[string]any
		}{
			{c, map[string]any{
				"cairo_image_surface_create": &imageSurfaceCreate, "cairo_image_surface_get_data": &imageSurfaceGetData,
				"cairo_image_surface_get_stride": &imageSurfaceGetStride, "cairo_surface_destroy": &surfaceDestroy,
				"cairo_surface_reference": &surfaceReference, "cairo_surface_flush": &surfaceFlush,
				"cairo_surface_mark_dirty": &surfaceMarkDirty, "cairo_create": &create, "cairo_destroy": &destroy,
				"cairo_save": &save, "cairo_restore": &restore, "cairo_translate": &translate, "cairo_scale": &scale,
				"cairo_rotate": &rotate, "cairo_transform": &transform, "cairo_get_matrix": &getMatrix,
				"cairo_set_matrix": &setMatrix, "cairo_identity_matrix": &identityMatrix, "cairo_new_path": &newPath,
				"cairo_close_path": &closePath, "cairo_move_to": &moveTo, "cairo_line_to": &lineTo,
				"cairo_curve_to": &curveTo, "cairo_rectangle": &rectangle, "cairo_set_source_rgba": &setSourceRGBA,
				"cairo_set_source": &setSource, "cairo_set_source_surface": &setSourceSurface,
				"cairo_set_fill_rule": &setFillRule, "cairo_set_operator": &setOperator, "cairo_fill": &fill,
				"cairo_stroke": &stroke, "cairo_clip": &clip, "cairo_paint": &paint,
				"cairo_paint_with_alpha": &paintWithAlpha, "cairo_mask_surface": &maskSurface,
				"cairo_set_line_width": &setLineWidth, "cairo_set_line_cap": &setLineCap,
				"cairo_set_line_join": &setLineJoin, "cairo_set_miter_limit": &setMiterLimit, "cairo_set_dash": &setDash,
				"cairo_push_group": &pushGroup, "cairo_pop_group_to_source": &popGroupToSource,
				"cairo_pattern_create_linear": &patternCreateLinear, "cairo_pattern_create_radial": &patternCreateRadial,
				"cairo_pattern_add_color_stop_rgba": &patternAddColorStop, "cairo_pattern_set_extend": &patternSetExtend,
				"cairo_pattern_destroy": &patternDestroy, "cairo_get_source": &getSource,
				"cairo_pattern_set_filter": &patternSetFilter, "cairo_font_options_create": &fontOptionsCreate,
				"cairo_font_options_set_antialias": &fontOptionsSetAA, "cairo_font_options_set_hint_style": &fontOptionsSetHint,
				"cairo_font_options_set_hint_metrics": &fontOptionsSetMetrics,
			}},
			{pc, map[string]any{
				"pango_cairo_font_map_new": &fmNew, "pango_cairo_context_set_font_options": &ctxSetFontOptions,
				"pango_cairo_update_context": &ctxUpdate, "pango_cairo_show_layout": &showLayout,
			}},
			{p, map[string]any{
				"pango_font_map_create_context": &fmCreateContext,
				"pango_layout_new":              &layoutNew, "pango_layout_set_text": &layoutSetText,
				"pango_layout_set_font_description": &layoutSetFont, "pango_layout_get_baseline": &layoutBaseline,
				"pango_layout_get_extents": &layoutExtents, "pango_font_description_new": &fdNew,
				"pango_font_description_set_family": &fdSetFamily, "pango_font_description_set_weight": &fdSetWeight,
				"pango_font_description_set_style": &fdSetStyle, "pango_font_description_set_absolute_size": &fdSetAbsSize,
			}},
			{gobj, map[string]any{"g_object_unref": &objectUnref}},
			{fc, map[string]any{
				"FcPatternCreate": &fcPatternCreate, "FcPatternDestroy": &fcPatternDestroy,
				"FcPatternAddString": &fcPatternAddString, "FcConfigSubstitute": &fcConfigSubstitute,
				"FcDefaultSubstitute": &fcDefaultSubstitute, "FcFontMatch": &fcFontMatch,
				"FcPatternGetString": &fcPatternGetString, "FcPatternGetInteger": &fcPatternGetInteger,
				"FcConfigAppFontAddFile": &fcAppFontAddFile, "FcConfigAppFontAddDir": &fcAppFontAddDir,
			}},
		} {
			for name, fn := range b.fns {
				purego.RegisterLibFunc(fn, b.lib, name)
			}
		}
		for _, o := range []struct {
			fn   any
			name string
			libs []uintptr
		}{
			{&ctxSetRound, "pango_context_set_round_glyph_positions", []uintptr{p}},
			{&fcConfigChanged, "pango_fc_font_map_config_changed", []uintptr{p, pc}},
		} {
			for _, l := range o.libs {
				if a, e := purego.Dlsym(l, o.name); e == nil {
					purego.RegisterFunc(o.fn, a)
					break
				}
			}
		}
	})
	return bindErr
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
