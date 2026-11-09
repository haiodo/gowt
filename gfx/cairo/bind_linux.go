//go:build linux && (!cgo || !gowtcgo)

package cairo

import (
	"fmt"

	"github.com/ebitengine/purego"
)

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
				"cairo_pdf_surface_create_for_stream": &pdfCreateForStream, "cairo_pdf_surface_set_size": &pdfSetSize,
				"cairo_show_page": &showPage, "cairo_surface_finish": &surfaceFinish, "cairo_surface_status": &surfaceStatus,
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
				"pango_attr_list_new": &attrListNew, "pango_attr_list_insert": &attrListInsert,
				"pango_attr_list_unref": &attrListUnref, "pango_attr_letter_spacing_new": &attrLetterSpacing,
				"pango_layout_set_attributes": &layoutSetAttrs,
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
