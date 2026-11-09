//go:build linux && cgo && gowtcgo

// Opt-in cgo backend for the cairo renderer: the same entry points as bind_linux.go,
// but every call goes through an exactly typed C wrapper - no purego/libffi on the
// render path. Libraries are still opened lazily at bind (a system without them gets
// the error from NewRenderer, same as the default backend) and symbols are resolved
// with dlsym at bind time, so the optional newer ones stay nil when absent.

package cairo

/*
#include <stdint.h>
#include <stdlib.h>

static uintptr_t gowt_image_surface_create(uintptr_t fn, int32_t f, int32_t w, int32_t h) {
	return ((uintptr_t (*)(int32_t, int32_t, int32_t))fn)(f, w, h);
}
static uintptr_t gowt_image_surface_get_data(uintptr_t fn, uintptr_t s) {
	return ((uintptr_t (*)(uintptr_t))fn)(s);
}
static int32_t gowt_image_surface_get_stride(uintptr_t fn, uintptr_t s) {
	return ((int32_t (*)(uintptr_t))fn)(s);
}
static void gowt_surface_destroy(uintptr_t fn, uintptr_t s) { ((void (*)(uintptr_t))fn)(s); }
static uintptr_t gowt_surface_reference(uintptr_t fn, uintptr_t s) {
	return ((uintptr_t (*)(uintptr_t))fn)(s);
}
static void gowt_surface_flush(uintptr_t fn, uintptr_t s) { ((void (*)(uintptr_t))fn)(s); }
static void gowt_surface_mark_dirty(uintptr_t fn, uintptr_t s) { ((void (*)(uintptr_t))fn)(s); }
static uintptr_t gowt_create(uintptr_t fn, uintptr_t s) { return ((uintptr_t (*)(uintptr_t))fn)(s); }
static void gowt_destroy(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_save(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_restore(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_translate(uintptr_t fn, uintptr_t cr, double a, double b) {
	((void (*)(uintptr_t, double, double))fn)(cr, a, b);
}
static void gowt_scale(uintptr_t fn, uintptr_t cr, double a, double b) {
	((void (*)(uintptr_t, double, double))fn)(cr, a, b);
}
static void gowt_rotate(uintptr_t fn, uintptr_t cr, double a) { ((void (*)(uintptr_t, double))fn)(cr, a); }
static void gowt_transform(uintptr_t fn, uintptr_t cr, void *m) {
	((void (*)(uintptr_t, void *))fn)(cr, m);
}
static void gowt_get_matrix(uintptr_t fn, uintptr_t cr, void *m) {
	((void (*)(uintptr_t, void *))fn)(cr, m);
}
static void gowt_set_matrix(uintptr_t fn, uintptr_t cr, void *m) {
	((void (*)(uintptr_t, void *))fn)(cr, m);
}
static void gowt_identity_matrix(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_new_path(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_close_path(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_move_to(uintptr_t fn, uintptr_t cr, double x, double y) {
	((void (*)(uintptr_t, double, double))fn)(cr, x, y);
}
static void gowt_line_to(uintptr_t fn, uintptr_t cr, double x, double y) {
	((void (*)(uintptr_t, double, double))fn)(cr, x, y);
}
static void gowt_curve_to(uintptr_t fn, uintptr_t cr, double x1, double y1, double x2, double y2, double x3, double y3) {
	((void (*)(uintptr_t, double, double, double, double, double, double))fn)(cr, x1, y1, x2, y2, x3, y3);
}
static void gowt_rectangle(uintptr_t fn, uintptr_t cr, double x, double y, double w, double h) {
	((void (*)(uintptr_t, double, double, double, double))fn)(cr, x, y, w, h);
}
static void gowt_set_source_rgba(uintptr_t fn, uintptr_t cr, double r, double g, double b, double a) {
	((void (*)(uintptr_t, double, double, double, double))fn)(cr, r, g, b, a);
}
static void gowt_set_source(uintptr_t fn, uintptr_t cr, uintptr_t p) {
	((void (*)(uintptr_t, uintptr_t))fn)(cr, p);
}
static void gowt_set_source_surface(uintptr_t fn, uintptr_t cr, uintptr_t s, double x, double y) {
	((void (*)(uintptr_t, uintptr_t, double, double))fn)(cr, s, x, y);
}
static void gowt_set_fill_rule(uintptr_t fn, uintptr_t cr, int32_t r) {
	((void (*)(uintptr_t, int32_t))fn)(cr, r);
}
static void gowt_set_operator(uintptr_t fn, uintptr_t cr, int32_t op) {
	((void (*)(uintptr_t, int32_t))fn)(cr, op);
}
static void gowt_fill(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_stroke(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_clip(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_paint(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_paint_with_alpha(uintptr_t fn, uintptr_t cr, double a) {
	((void (*)(uintptr_t, double))fn)(cr, a);
}
static void gowt_mask_surface(uintptr_t fn, uintptr_t cr, uintptr_t s, double x, double y) {
	((void (*)(uintptr_t, uintptr_t, double, double))fn)(cr, s, x, y);
}
static void gowt_set_line_width(uintptr_t fn, uintptr_t cr, double w) {
	((void (*)(uintptr_t, double))fn)(cr, w);
}
static void gowt_set_line_cap(uintptr_t fn, uintptr_t cr, int32_t c) {
	((void (*)(uintptr_t, int32_t))fn)(cr, c);
}
static void gowt_set_line_join(uintptr_t fn, uintptr_t cr, int32_t j) {
	((void (*)(uintptr_t, int32_t))fn)(cr, j);
}
static void gowt_set_miter_limit(uintptr_t fn, uintptr_t cr, double m) {
	((void (*)(uintptr_t, double))fn)(cr, m);
}
static void gowt_set_dash(uintptr_t fn, uintptr_t cr, void *d, int32_t n, double off) {
	((void (*)(uintptr_t, void *, int32_t, double))fn)(cr, d, n, off);
}
static void gowt_push_group(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_pop_group_to_source(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static uintptr_t gowt_pattern_create_linear(uintptr_t fn, double x0, double y0, double x1, double y1) {
	return ((uintptr_t (*)(double, double, double, double))fn)(x0, y0, x1, y1);
}
static uintptr_t gowt_pattern_create_radial(uintptr_t fn, double x0, double y0, double r0, double x1, double y1, double r1) {
	return ((uintptr_t (*)(double, double, double, double, double, double))fn)(x0, y0, r0, x1, y1, r1);
}
static void gowt_pattern_add_color_stop(uintptr_t fn, uintptr_t p, double off, double r, double g, double b, double a) {
	((void (*)(uintptr_t, double, double, double, double, double))fn)(p, off, r, g, b, a);
}
static void gowt_pattern_set_extend(uintptr_t fn, uintptr_t p, int32_t e) {
	((void (*)(uintptr_t, int32_t))fn)(p, e);
}
static void gowt_pattern_destroy(uintptr_t fn, uintptr_t p) { ((void (*)(uintptr_t))fn)(p); }
static uintptr_t gowt_get_source(uintptr_t fn, uintptr_t cr) { return ((uintptr_t (*)(uintptr_t))fn)(cr); }
static void gowt_pattern_set_filter(uintptr_t fn, uintptr_t p, int32_t f) {
	((void (*)(uintptr_t, int32_t))fn)(p, f);
}
static uintptr_t gowt_font_options_create(uintptr_t fn) { return ((uintptr_t (*)(void))fn)(); }
static uintptr_t gowt_pdf_create_for_stream(uintptr_t fn, uintptr_t w, uintptr_t closure, double wd, double h) {
	return ((uintptr_t (*)(uintptr_t, uintptr_t, double, double))fn)(w, closure, wd, h);
}
static void gowt_pdf_set_size(uintptr_t fn, uintptr_t s, double w, double h) {
	((void (*)(uintptr_t, double, double))fn)(s, w, h);
}
static void gowt_show_page(uintptr_t fn, uintptr_t cr) { ((void (*)(uintptr_t))fn)(cr); }
static void gowt_surface_finish(uintptr_t fn, uintptr_t s) { ((void (*)(uintptr_t))fn)(s); }
static int32_t gowt_surface_status(uintptr_t fn, uintptr_t s) { return ((int32_t (*)(uintptr_t))fn)(s); }
static void gowt_font_options_set_aa(uintptr_t fn, uintptr_t o, int32_t v) {
	((void (*)(uintptr_t, int32_t))fn)(o, v);
}
static void gowt_font_options_set_hint(uintptr_t fn, uintptr_t o, int32_t v) {
	((void (*)(uintptr_t, int32_t))fn)(o, v);
}
static void gowt_font_options_set_metrics(uintptr_t fn, uintptr_t o, int32_t v) {
	((void (*)(uintptr_t, int32_t))fn)(o, v);
}
static uintptr_t gowt_fm_new(uintptr_t fn) { return ((uintptr_t (*)(void))fn)(); }
static uintptr_t gowt_fm_create_context(uintptr_t fn, uintptr_t fm) {
	return ((uintptr_t (*)(uintptr_t))fn)(fm);
}
static void gowt_ctx_set_font_options(uintptr_t fn, uintptr_t ctx, uintptr_t opts) {
	((void (*)(uintptr_t, uintptr_t))fn)(ctx, opts);
}
static void gowt_ctx_update(uintptr_t fn, uintptr_t cr, uintptr_t ctx) {
	((void (*)(uintptr_t, uintptr_t))fn)(cr, ctx);
}
static uintptr_t gowt_layout_new(uintptr_t fn, uintptr_t ctx) { return ((uintptr_t (*)(uintptr_t))fn)(ctx); }
static void gowt_layout_set_text(uintptr_t fn, uintptr_t l, char *s, int32_t n) {
	((void (*)(uintptr_t, char *, int32_t))fn)(l, s, n);
}
static void gowt_layout_set_font(uintptr_t fn, uintptr_t l, uintptr_t desc) {
	((void (*)(uintptr_t, uintptr_t))fn)(l, desc);
}
static int32_t gowt_layout_baseline(uintptr_t fn, uintptr_t l) { return ((int32_t (*)(uintptr_t))fn)(l); }
static void gowt_layout_extents(uintptr_t fn, uintptr_t l, void *ink, void *logical) {
	((void (*)(uintptr_t, void *, void *))fn)(l, ink, logical);
}
static void gowt_show_layout(uintptr_t fn, uintptr_t cr, uintptr_t l) {
	((void (*)(uintptr_t, uintptr_t))fn)(cr, l);
}
static uintptr_t gowt_fd_new(uintptr_t fn) { return ((uintptr_t (*)(void))fn)(); }
static void gowt_fd_set_family(uintptr_t fn, uintptr_t d, char *f) {
	((void (*)(uintptr_t, char *))fn)(d, f);
}
static void gowt_fd_set_weight(uintptr_t fn, uintptr_t d, int32_t w) {
	((void (*)(uintptr_t, int32_t))fn)(d, w);
}
static void gowt_fd_set_style(uintptr_t fn, uintptr_t d, int32_t s) {
	((void (*)(uintptr_t, int32_t))fn)(d, s);
}
static void gowt_fd_set_abs_size(uintptr_t fn, uintptr_t d, double s) {
	((void (*)(uintptr_t, double))fn)(d, s);
}
static uintptr_t gowt_attr_list_new(uintptr_t fn) { return ((uintptr_t (*)(void))fn)(); }
static void gowt_attr_list_insert(uintptr_t fn, uintptr_t list, uintptr_t attr) {
	((void (*)(uintptr_t, uintptr_t))fn)(list, attr);
}
static void gowt_attr_list_unref(uintptr_t fn, uintptr_t list) { ((void (*)(uintptr_t))fn)(list); }
static uintptr_t gowt_attr_letter_spacing(uintptr_t fn, int32_t units) {
	return ((uintptr_t (*)(int32_t))fn)(units);
}
static void gowt_layout_set_attrs(uintptr_t fn, uintptr_t l, uintptr_t list) {
	((void (*)(uintptr_t, uintptr_t))fn)(l, list);
}
static void gowt_object_unref(uintptr_t fn, uintptr_t o) { ((void (*)(uintptr_t))fn)(o); }
static uintptr_t gowt_fc_pattern_create(uintptr_t fn) { return ((uintptr_t (*)(void))fn)(); }
static void gowt_fc_pattern_destroy(uintptr_t fn, uintptr_t p) { ((void (*)(uintptr_t))fn)(p); }
static int32_t gowt_fc_pattern_add_string(uintptr_t fn, uintptr_t p, char *obj, char *s) {
	return ((int32_t (*)(uintptr_t, char *, char *))fn)(p, obj, s);
}
static int32_t gowt_fc_config_substitute(uintptr_t fn, uintptr_t cfg, uintptr_t p, int32_t kind) {
	return ((int32_t (*)(uintptr_t, uintptr_t, int32_t))fn)(cfg, p, kind);
}
static void gowt_fc_default_substitute(uintptr_t fn, uintptr_t p) { ((void (*)(uintptr_t))fn)(p); }
static uintptr_t gowt_fc_font_match(uintptr_t fn, uintptr_t cfg, uintptr_t p, void *res) {
	return ((uintptr_t (*)(uintptr_t, uintptr_t, void *))fn)(cfg, p, res);
}
static int32_t gowt_fc_pattern_get_string(uintptr_t fn, uintptr_t p, char *obj, int32_t n, void *out) {
	return ((int32_t (*)(uintptr_t, char *, int32_t, void *))fn)(p, obj, n, out);
}
static int32_t gowt_fc_pattern_get_integer(uintptr_t fn, uintptr_t p, char *obj, int32_t n, void *out) {
	return ((int32_t (*)(uintptr_t, char *, int32_t, void *))fn)(p, obj, n, out);
}
static int32_t gowt_fc_app_font_add_dir(uintptr_t fn, uintptr_t cfg, char *dir) {
	return ((int32_t (*)(uintptr_t, char *))fn)(cfg, dir);
}
static int32_t gowt_fc_app_font_add_file(uintptr_t fn, uintptr_t cfg, char *file) {
	return ((int32_t (*)(uintptr_t, char *))fn)(cfg, file);
}
static void gowt_ctx_set_round(uintptr_t fn, uintptr_t ctx, int32_t v) {
	((void (*)(uintptr_t, int32_t))fn)(ctx, v);
}
static void gowt_fc_config_changed(uintptr_t fn, uintptr_t fm) { ((void (*)(uintptr_t))fn)(fm); }
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// p_* holds the dlsym'd address per entry point; the closures below pass it to the
// typed C wrapper as the first argument.
var (
	p_image_surface_create, p_image_surface_get_data, p_image_surface_get_stride     uintptr
	p_surface_destroy, p_surface_reference, p_surface_flush, p_surface_mark_dirty    uintptr
	p_create, p_destroy, p_save, p_restore, p_translate, p_scale, p_rotate           uintptr
	p_transform, p_get_matrix, p_set_matrix, p_identity_matrix                       uintptr
	p_new_path, p_close_path, p_move_to, p_line_to, p_curve_to, p_rectangle          uintptr
	p_set_source_rgba, p_set_source, p_set_source_surface                            uintptr
	p_set_fill_rule, p_set_operator, p_fill, p_stroke, p_clip                        uintptr
	p_paint, p_paint_with_alpha, p_mask_surface                                      uintptr
	p_set_line_width, p_set_line_cap, p_set_line_join, p_set_miter_limit, p_set_dash uintptr
	p_push_group, p_pop_group_to_source                                              uintptr
	p_pattern_create_linear, p_pattern_create_radial, p_pattern_add_color_stop       uintptr
	p_pattern_set_extend, p_pattern_destroy, p_get_source, p_pattern_set_filter      uintptr
	p_font_options_create, p_pdf_create_for_stream, p_pdf_set_size                   uintptr
	p_show_page, p_surface_finish, p_surface_status                                  uintptr
	p_font_options_set_aa, p_font_options_set_hint, p_font_options_set_metrics       uintptr
	p_fm_new, p_fm_create_context, p_ctx_set_font_options, p_ctx_update              uintptr
	p_layout_new, p_layout_set_text, p_layout_set_font, p_layout_baseline            uintptr
	p_layout_extents, p_show_layout                                                  uintptr
	p_fd_new, p_fd_set_family, p_fd_set_weight, p_fd_set_style, p_fd_set_abs_size    uintptr
	p_attr_list_new, p_attr_list_insert, p_attr_list_unref, p_attr_letter_spacing    uintptr
	p_layout_set_attrs, p_object_unref                                               uintptr
	p_fc_pattern_create, p_fc_pattern_destroy, p_fc_pattern_add_string               uintptr
	p_fc_config_substitute, p_fc_default_substitute, p_fc_font_match                 uintptr
	p_fc_pattern_get_string, p_fc_pattern_get_integer                                uintptr
	p_fc_app_font_add_dir, p_fc_app_font_add_file                                    uintptr
	p_ctx_set_round, p_fc_config_changed                                             uintptr
)

func cairoSym(h uintptr, name string) (uintptr, error) {
	a, err := purego.Dlsym(h, name)
	if err != nil || a == 0 {
		return 0, fmt.Errorf("gfx/cairo: %s not found", name)
	}
	return a, nil
}

func bind() (err error) {
	bindOnce.Do(func() {
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
		must := func(lib uintptr, name string) uintptr {
			a, e := cairoSym(lib, name)
			if e != nil {
				panic(e)
			}
			return a
		}
		defer func() {
			if r := recover(); r != nil {
				bindErr = fmt.Errorf("gfx/cairo: %v", r)
			}
		}()
		p_image_surface_create = must(c, "cairo_image_surface_create")
		p_image_surface_get_data = must(c, "cairo_image_surface_get_data")
		p_image_surface_get_stride = must(c, "cairo_image_surface_get_stride")
		p_surface_destroy = must(c, "cairo_surface_destroy")
		p_surface_reference = must(c, "cairo_surface_reference")
		p_surface_flush = must(c, "cairo_surface_flush")
		p_surface_mark_dirty = must(c, "cairo_surface_mark_dirty")
		p_create = must(c, "cairo_create")
		p_destroy = must(c, "cairo_destroy")
		p_save = must(c, "cairo_save")
		p_restore = must(c, "cairo_restore")
		p_translate = must(c, "cairo_translate")
		p_scale = must(c, "cairo_scale")
		p_rotate = must(c, "cairo_rotate")
		p_transform = must(c, "cairo_transform")
		p_get_matrix = must(c, "cairo_get_matrix")
		p_set_matrix = must(c, "cairo_set_matrix")
		p_identity_matrix = must(c, "cairo_identity_matrix")
		p_new_path = must(c, "cairo_new_path")
		p_close_path = must(c, "cairo_close_path")
		p_move_to = must(c, "cairo_move_to")
		p_line_to = must(c, "cairo_line_to")
		p_curve_to = must(c, "cairo_curve_to")
		p_rectangle = must(c, "cairo_rectangle")
		p_set_source_rgba = must(c, "cairo_set_source_rgba")
		p_set_source = must(c, "cairo_set_source")
		p_set_source_surface = must(c, "cairo_set_source_surface")
		p_set_fill_rule = must(c, "cairo_set_fill_rule")
		p_set_operator = must(c, "cairo_set_operator")
		p_fill = must(c, "cairo_fill")
		p_stroke = must(c, "cairo_stroke")
		p_clip = must(c, "cairo_clip")
		p_paint = must(c, "cairo_paint")
		p_paint_with_alpha = must(c, "cairo_paint_with_alpha")
		p_mask_surface = must(c, "cairo_mask_surface")
		p_set_line_width = must(c, "cairo_set_line_width")
		p_set_line_cap = must(c, "cairo_set_line_cap")
		p_set_line_join = must(c, "cairo_set_line_join")
		p_set_miter_limit = must(c, "cairo_set_miter_limit")
		p_set_dash = must(c, "cairo_set_dash")
		p_push_group = must(c, "cairo_push_group")
		p_pop_group_to_source = must(c, "cairo_pop_group_to_source")
		p_pattern_create_linear = must(c, "cairo_pattern_create_linear")
		p_pattern_create_radial = must(c, "cairo_pattern_create_radial")
		p_pattern_add_color_stop = must(c, "cairo_pattern_add_color_stop_rgba")
		p_pattern_set_extend = must(c, "cairo_pattern_set_extend")
		p_pattern_destroy = must(c, "cairo_pattern_destroy")
		p_get_source = must(c, "cairo_get_source")
		p_pattern_set_filter = must(c, "cairo_pattern_set_filter")
		p_font_options_create = must(c, "cairo_font_options_create")
		p_pdf_create_for_stream = must(c, "cairo_pdf_surface_create_for_stream")
		p_pdf_set_size = must(c, "cairo_pdf_surface_set_size")
		p_show_page = must(c, "cairo_show_page")
		p_surface_finish = must(c, "cairo_surface_finish")
		p_surface_status = must(c, "cairo_surface_status")
		p_font_options_set_aa = must(c, "cairo_font_options_set_antialias")
		p_font_options_set_hint = must(c, "cairo_font_options_set_hint_style")
		p_font_options_set_metrics = must(c, "cairo_font_options_set_hint_metrics")
		p_fm_new = must(pc, "pango_cairo_font_map_new")
		p_ctx_set_font_options = must(pc, "pango_cairo_context_set_font_options")
		p_ctx_update = must(pc, "pango_cairo_update_context")
		p_show_layout = must(pc, "pango_cairo_show_layout")
		p_fm_create_context = must(p, "pango_font_map_create_context")
		p_layout_new = must(p, "pango_layout_new")
		p_layout_set_text = must(p, "pango_layout_set_text")
		p_layout_set_font = must(p, "pango_layout_set_font_description")
		p_layout_baseline = must(p, "pango_layout_get_baseline")
		p_layout_extents = must(p, "pango_layout_get_extents")
		p_fd_new = must(p, "pango_font_description_new")
		p_fd_set_family = must(p, "pango_font_description_set_family")
		p_fd_set_weight = must(p, "pango_font_description_set_weight")
		p_fd_set_style = must(p, "pango_font_description_set_style")
		p_fd_set_abs_size = must(p, "pango_font_description_set_absolute_size")
		p_attr_list_new = must(p, "pango_attr_list_new")
		p_attr_list_insert = must(p, "pango_attr_list_insert")
		p_attr_list_unref = must(p, "pango_attr_list_unref")
		p_attr_letter_spacing = must(p, "pango_attr_letter_spacing_new")
		p_layout_set_attrs = must(p, "pango_layout_set_attributes")
		p_object_unref = must(gobj, "g_object_unref")
		p_fc_pattern_create = must(fc, "FcPatternCreate")
		p_fc_pattern_destroy = must(fc, "FcPatternDestroy")
		p_fc_pattern_add_string = must(fc, "FcPatternAddString")
		p_fc_config_substitute = must(fc, "FcConfigSubstitute")
		p_fc_default_substitute = must(fc, "FcDefaultSubstitute")
		p_fc_font_match = must(fc, "FcFontMatch")
		p_fc_pattern_get_string = must(fc, "FcPatternGetString")
		p_fc_pattern_get_integer = must(fc, "FcPatternGetInteger")
		p_fc_app_font_add_file = must(fc, "FcConfigAppFontAddFile")
		p_fc_app_font_add_dir = must(fc, "FcConfigAppFontAddDir")
		if a, e := purego.Dlsym(p, "pango_context_set_round_glyph_positions"); e == nil && a != 0 {
			p_ctx_set_round = a
		}
		if a, e := purego.Dlsym(p, "pango_fc_font_map_config_changed"); e == nil && a != 0 {
			p_fc_config_changed = a
		}
		if a, e := purego.Dlsym(pc, "pango_fc_font_map_config_changed"); e == nil && a != 0 {
			p_fc_config_changed = a
		}

		imageSurfaceCreate = func(format int32, w, h int32) uintptr {
			return uintptr(C.gowt_image_surface_create(C.uintptr_t(p_image_surface_create), C.int32_t(format), C.int32_t(w), C.int32_t(h)))
		}
		imageSurfaceGetData = func(s uintptr) uintptr {
			return uintptr(C.gowt_image_surface_get_data(C.uintptr_t(p_image_surface_get_data), C.uintptr_t(s)))
		}
		imageSurfaceGetStride = func(s uintptr) int32 {
			return int32(C.gowt_image_surface_get_stride(C.uintptr_t(p_image_surface_get_stride), C.uintptr_t(s)))
		}
		surfaceDestroy = func(s uintptr) { C.gowt_surface_destroy(C.uintptr_t(p_surface_destroy), C.uintptr_t(s)) }
		surfaceReference = func(s uintptr) uintptr {
			return uintptr(C.gowt_surface_reference(C.uintptr_t(p_surface_reference), C.uintptr_t(s)))
		}
		surfaceFlush = func(s uintptr) { C.gowt_surface_flush(C.uintptr_t(p_surface_flush), C.uintptr_t(s)) }
		surfaceMarkDirty = func(s uintptr) { C.gowt_surface_mark_dirty(C.uintptr_t(p_surface_mark_dirty), C.uintptr_t(s)) }
		create = func(s uintptr) uintptr { return uintptr(C.gowt_create(C.uintptr_t(p_create), C.uintptr_t(s))) }
		destroy = func(cr uintptr) { C.gowt_destroy(C.uintptr_t(p_destroy), C.uintptr_t(cr)) }
		save = func(cr uintptr) { C.gowt_save(C.uintptr_t(p_save), C.uintptr_t(cr)) }
		restore = func(cr uintptr) { C.gowt_restore(C.uintptr_t(p_restore), C.uintptr_t(cr)) }
		translate = func(cr uintptr, a, b float64) {
			C.gowt_translate(C.uintptr_t(p_translate), C.uintptr_t(cr), C.double(a), C.double(b))
		}
		scale = func(cr uintptr, a, b float64) {
			C.gowt_scale(C.uintptr_t(p_scale), C.uintptr_t(cr), C.double(a), C.double(b))
		}
		rotate = func(cr uintptr, a float64) { C.gowt_rotate(C.uintptr_t(p_rotate), C.uintptr_t(cr), C.double(a)) }
		transform = func(cr uintptr, m *matrix) {
			C.gowt_transform(C.uintptr_t(p_transform), C.uintptr_t(cr), unsafe.Pointer(m))
		}
		getMatrix = func(cr uintptr, m *matrix) {
			C.gowt_get_matrix(C.uintptr_t(p_get_matrix), C.uintptr_t(cr), unsafe.Pointer(m))
		}
		setMatrix = func(cr uintptr, m *matrix) {
			C.gowt_set_matrix(C.uintptr_t(p_set_matrix), C.uintptr_t(cr), unsafe.Pointer(m))
		}
		identityMatrix = func(cr uintptr) { C.gowt_identity_matrix(C.uintptr_t(p_identity_matrix), C.uintptr_t(cr)) }
		newPath = func(cr uintptr) { C.gowt_new_path(C.uintptr_t(p_new_path), C.uintptr_t(cr)) }
		closePath = func(cr uintptr) { C.gowt_close_path(C.uintptr_t(p_close_path), C.uintptr_t(cr)) }
		moveTo = func(cr uintptr, x, y float64) {
			C.gowt_move_to(C.uintptr_t(p_move_to), C.uintptr_t(cr), C.double(x), C.double(y))
		}
		lineTo = func(cr uintptr, x, y float64) {
			C.gowt_line_to(C.uintptr_t(p_line_to), C.uintptr_t(cr), C.double(x), C.double(y))
		}
		curveTo = func(cr uintptr, x1, y1, x2, y2, x3, y3 float64) {
			C.gowt_curve_to(C.uintptr_t(p_curve_to), C.uintptr_t(cr), C.double(x1), C.double(y1), C.double(x2), C.double(y2), C.double(x3), C.double(y3))
		}
		rectangle = func(cr uintptr, x, y, w, h float64) {
			C.gowt_rectangle(C.uintptr_t(p_rectangle), C.uintptr_t(cr), C.double(x), C.double(y), C.double(w), C.double(h))
		}
		setSourceRGBA = func(cr uintptr, r, g, b, a float64) {
			C.gowt_set_source_rgba(C.uintptr_t(p_set_source_rgba), C.uintptr_t(cr), C.double(r), C.double(g), C.double(b), C.double(a))
		}
		setSource = func(cr uintptr, p uintptr) {
			C.gowt_set_source(C.uintptr_t(p_set_source), C.uintptr_t(cr), C.uintptr_t(p))
		}
		setSourceSurface = func(cr uintptr, s uintptr, x, y float64) {
			C.gowt_set_source_surface(C.uintptr_t(p_set_source_surface), C.uintptr_t(cr), C.uintptr_t(s), C.double(x), C.double(y))
		}
		setFillRule = func(cr uintptr, r int32) {
			C.gowt_set_fill_rule(C.uintptr_t(p_set_fill_rule), C.uintptr_t(cr), C.int32_t(r))
		}
		setOperator = func(cr uintptr, op int32) {
			C.gowt_set_operator(C.uintptr_t(p_set_operator), C.uintptr_t(cr), C.int32_t(op))
		}
		fill = func(cr uintptr) { C.gowt_fill(C.uintptr_t(p_fill), C.uintptr_t(cr)) }
		stroke = func(cr uintptr) { C.gowt_stroke(C.uintptr_t(p_stroke), C.uintptr_t(cr)) }
		clip = func(cr uintptr) { C.gowt_clip(C.uintptr_t(p_clip), C.uintptr_t(cr)) }
		paint = func(cr uintptr) { C.gowt_paint(C.uintptr_t(p_paint), C.uintptr_t(cr)) }
		paintWithAlpha = func(cr uintptr, a float64) {
			C.gowt_paint_with_alpha(C.uintptr_t(p_paint_with_alpha), C.uintptr_t(cr), C.double(a))
		}
		maskSurface = func(cr uintptr, s uintptr, x, y float64) {
			C.gowt_mask_surface(C.uintptr_t(p_mask_surface), C.uintptr_t(cr), C.uintptr_t(s), C.double(x), C.double(y))
		}
		setLineWidth = func(cr uintptr, w float64) {
			C.gowt_set_line_width(C.uintptr_t(p_set_line_width), C.uintptr_t(cr), C.double(w))
		}
		setLineCap = func(cr uintptr, c int32) {
			C.gowt_set_line_cap(C.uintptr_t(p_set_line_cap), C.uintptr_t(cr), C.int32_t(c))
		}
		setLineJoin = func(cr uintptr, j int32) {
			C.gowt_set_line_join(C.uintptr_t(p_set_line_join), C.uintptr_t(cr), C.int32_t(j))
		}
		setMiterLimit = func(cr uintptr, m float64) {
			C.gowt_set_miter_limit(C.uintptr_t(p_set_miter_limit), C.uintptr_t(cr), C.double(m))
		}
		setDash = func(cr uintptr, d *float64, n int32, off float64) {
			C.gowt_set_dash(C.uintptr_t(p_set_dash), C.uintptr_t(cr), unsafe.Pointer(d), C.int32_t(n), C.double(off))
		}
		pushGroup = func(cr uintptr) { C.gowt_push_group(C.uintptr_t(p_push_group), C.uintptr_t(cr)) }
		popGroupToSource = func(cr uintptr) { C.gowt_pop_group_to_source(C.uintptr_t(p_pop_group_to_source), C.uintptr_t(cr)) }
		patternCreateLinear = func(x0, y0, x1, y1 float64) uintptr {
			return uintptr(C.gowt_pattern_create_linear(C.uintptr_t(p_pattern_create_linear), C.double(x0), C.double(y0), C.double(x1), C.double(y1)))
		}
		patternCreateRadial = func(x0, y0, r0, x1, y1, r1 float64) uintptr {
			return uintptr(C.gowt_pattern_create_radial(C.uintptr_t(p_pattern_create_radial), C.double(x0), C.double(y0), C.double(r0), C.double(x1), C.double(y1), C.double(r1)))
		}
		patternAddColorStop = func(p uintptr, off, r, g, b, a float64) {
			C.gowt_pattern_add_color_stop(C.uintptr_t(p_pattern_add_color_stop), C.uintptr_t(p), C.double(off), C.double(r), C.double(g), C.double(b), C.double(a))
		}
		patternSetExtend = func(p uintptr, e int32) {
			C.gowt_pattern_set_extend(C.uintptr_t(p_pattern_set_extend), C.uintptr_t(p), C.int32_t(e))
		}
		patternDestroy = func(p uintptr) { C.gowt_pattern_destroy(C.uintptr_t(p_pattern_destroy), C.uintptr_t(p)) }
		getSource = func(cr uintptr) uintptr {
			return uintptr(C.gowt_get_source(C.uintptr_t(p_get_source), C.uintptr_t(cr)))
		}
		patternSetFilter = func(p uintptr, f int32) {
			C.gowt_pattern_set_filter(C.uintptr_t(p_pattern_set_filter), C.uintptr_t(p), C.int32_t(f))
		}
		fontOptionsCreate = func() uintptr { return uintptr(C.gowt_font_options_create(C.uintptr_t(p_font_options_create))) }
		pdfCreateForStream = func(write uintptr, closure uintptr, w, h float64) uintptr {
			return uintptr(C.gowt_pdf_create_for_stream(C.uintptr_t(p_pdf_create_for_stream), C.uintptr_t(write), C.uintptr_t(closure), C.double(w), C.double(h)))
		}
		pdfSetSize = func(s uintptr, w, h float64) {
			C.gowt_pdf_set_size(C.uintptr_t(p_pdf_set_size), C.uintptr_t(s), C.double(w), C.double(h))
		}
		showPage = func(cr uintptr) { C.gowt_show_page(C.uintptr_t(p_show_page), C.uintptr_t(cr)) }
		surfaceFinish = func(s uintptr) { C.gowt_surface_finish(C.uintptr_t(p_surface_finish), C.uintptr_t(s)) }
		surfaceStatus = func(s uintptr) int32 {
			return int32(C.gowt_surface_status(C.uintptr_t(p_surface_status), C.uintptr_t(s)))
		}
		fontOptionsSetAA = func(o uintptr, v int32) {
			C.gowt_font_options_set_aa(C.uintptr_t(p_font_options_set_aa), C.uintptr_t(o), C.int32_t(v))
		}
		fontOptionsSetHint = func(o uintptr, v int32) {
			C.gowt_font_options_set_hint(C.uintptr_t(p_font_options_set_hint), C.uintptr_t(o), C.int32_t(v))
		}
		fontOptionsSetMetrics = func(o uintptr, v int32) {
			C.gowt_font_options_set_metrics(C.uintptr_t(p_font_options_set_metrics), C.uintptr_t(o), C.int32_t(v))
		}
		fmNew = func() uintptr { return uintptr(C.gowt_fm_new(C.uintptr_t(p_fm_new))) }
		fmCreateContext = func(fm uintptr) uintptr {
			return uintptr(C.gowt_fm_create_context(C.uintptr_t(p_fm_create_context), C.uintptr_t(fm)))
		}
		ctxSetFontOptions = func(ctx, opts uintptr) {
			C.gowt_ctx_set_font_options(C.uintptr_t(p_ctx_set_font_options), C.uintptr_t(ctx), C.uintptr_t(opts))
		}
		ctxUpdate = func(cr, ctx uintptr) {
			C.gowt_ctx_update(C.uintptr_t(p_ctx_update), C.uintptr_t(cr), C.uintptr_t(ctx))
		}
		layoutNew = func(ctx uintptr) uintptr {
			return uintptr(C.gowt_layout_new(C.uintptr_t(p_layout_new), C.uintptr_t(ctx)))
		}
		layoutSetText = func(l uintptr, s string, n int32) {
			cs := C.CString(s)
			defer C.free(unsafe.Pointer(cs))
			C.gowt_layout_set_text(C.uintptr_t(p_layout_set_text), C.uintptr_t(l), cs, C.int32_t(n))
		}
		layoutSetFont = func(l, desc uintptr) {
			C.gowt_layout_set_font(C.uintptr_t(p_layout_set_font), C.uintptr_t(l), C.uintptr_t(desc))
		}
		layoutBaseline = func(l uintptr) int32 {
			return int32(C.gowt_layout_baseline(C.uintptr_t(p_layout_baseline), C.uintptr_t(l)))
		}
		layoutExtents = func(l uintptr, ink, logical *pangoRect) {
			C.gowt_layout_extents(C.uintptr_t(p_layout_extents), C.uintptr_t(l), unsafe.Pointer(ink), unsafe.Pointer(logical))
		}
		showLayout = func(cr, l uintptr) {
			C.gowt_show_layout(C.uintptr_t(p_show_layout), C.uintptr_t(cr), C.uintptr_t(l))
		}
		fdNew = func() uintptr { return uintptr(C.gowt_fd_new(C.uintptr_t(p_fd_new))) }
		fdSetFamily = func(d uintptr, f string) {
			cs := C.CString(f)
			defer C.free(unsafe.Pointer(cs))
			C.gowt_fd_set_family(C.uintptr_t(p_fd_set_family), C.uintptr_t(d), cs)
		}
		fdSetWeight = func(d uintptr, w int32) {
			C.gowt_fd_set_weight(C.uintptr_t(p_fd_set_weight), C.uintptr_t(d), C.int32_t(w))
		}
		fdSetStyle = func(d uintptr, s int32) {
			C.gowt_fd_set_style(C.uintptr_t(p_fd_set_style), C.uintptr_t(d), C.int32_t(s))
		}
		fdSetAbsSize = func(d uintptr, s float64) {
			C.gowt_fd_set_abs_size(C.uintptr_t(p_fd_set_abs_size), C.uintptr_t(d), C.double(s))
		}
		attrListNew = func() uintptr { return uintptr(C.gowt_attr_list_new(C.uintptr_t(p_attr_list_new))) }
		attrListInsert = func(list, attr uintptr) {
			C.gowt_attr_list_insert(C.uintptr_t(p_attr_list_insert), C.uintptr_t(list), C.uintptr_t(attr))
		}
		attrListUnref = func(list uintptr) { C.gowt_attr_list_unref(C.uintptr_t(p_attr_list_unref), C.uintptr_t(list)) }
		attrLetterSpacing = func(units int32) uintptr {
			return uintptr(C.gowt_attr_letter_spacing(C.uintptr_t(p_attr_letter_spacing), C.int32_t(units)))
		}
		layoutSetAttrs = func(l, list uintptr) {
			C.gowt_layout_set_attrs(C.uintptr_t(p_layout_set_attrs), C.uintptr_t(l), C.uintptr_t(list))
		}
		objectUnref = func(o uintptr) { C.gowt_object_unref(C.uintptr_t(p_object_unref), C.uintptr_t(o)) }
		fcPatternCreate = func() uintptr { return uintptr(C.gowt_fc_pattern_create(C.uintptr_t(p_fc_pattern_create))) }
		fcPatternDestroy = func(p uintptr) { C.gowt_fc_pattern_destroy(C.uintptr_t(p_fc_pattern_destroy), C.uintptr_t(p)) }
		fcPatternAddString = func(p uintptr, obj, s string) int32 {
			cobj := C.CString(obj)
			defer C.free(unsafe.Pointer(cobj))
			cs := C.CString(s)
			defer C.free(unsafe.Pointer(cs))
			return int32(C.gowt_fc_pattern_add_string(C.uintptr_t(p_fc_pattern_add_string), C.uintptr_t(p), cobj, cs))
		}
		fcConfigSubstitute = func(cfg, p uintptr, kind int32) int32 {
			return int32(C.gowt_fc_config_substitute(C.uintptr_t(p_fc_config_substitute), C.uintptr_t(cfg), C.uintptr_t(p), C.int32_t(kind)))
		}
		fcDefaultSubstitute = func(p uintptr) { C.gowt_fc_default_substitute(C.uintptr_t(p_fc_default_substitute), C.uintptr_t(p)) }
		fcFontMatch = func(cfg, p uintptr, res *int32) uintptr {
			return uintptr(C.gowt_fc_font_match(C.uintptr_t(p_fc_font_match), C.uintptr_t(cfg), C.uintptr_t(p), unsafe.Pointer(res)))
		}
		fcPatternGetString = func(p uintptr, obj string, n int32, out *uintptr) int32 {
			cobj := C.CString(obj)
			defer C.free(unsafe.Pointer(cobj))
			return int32(C.gowt_fc_pattern_get_string(C.uintptr_t(p_fc_pattern_get_string), C.uintptr_t(p), cobj, C.int32_t(n), unsafe.Pointer(out)))
		}
		fcPatternGetInteger = func(p uintptr, obj string, n int32, out *int32) int32 {
			cobj := C.CString(obj)
			defer C.free(unsafe.Pointer(cobj))
			return int32(C.gowt_fc_pattern_get_integer(C.uintptr_t(p_fc_pattern_get_integer), C.uintptr_t(p), cobj, C.int32_t(n), unsafe.Pointer(out)))
		}
		fcAppFontAddDir = func(cfg uintptr, dir string) int32 {
			cs := C.CString(dir)
			defer C.free(unsafe.Pointer(cs))
			return int32(C.gowt_fc_app_font_add_dir(C.uintptr_t(p_fc_app_font_add_dir), C.uintptr_t(cfg), cs))
		}
		fcAppFontAddFile = func(cfg uintptr, file string) int32 {
			cs := C.CString(file)
			defer C.free(unsafe.Pointer(cs))
			return int32(C.gowt_fc_app_font_add_file(C.uintptr_t(p_fc_app_font_add_file), C.uintptr_t(cfg), cs))
		}
		ctxSetRound = nil
		if p_ctx_set_round != 0 {
			ctxSetRound = func(ctx uintptr, v int32) {
				C.gowt_ctx_set_round(C.uintptr_t(p_ctx_set_round), C.uintptr_t(ctx), C.int32_t(v))
			}
		}
		fcConfigChanged = nil
		if p_fc_config_changed != 0 {
			fcConfigChanged = func(fm uintptr) { C.gowt_fc_config_changed(C.uintptr_t(p_fc_config_changed), C.uintptr_t(fm)) }
		}
	})
	return bindErr
}
