//go:build linux

package gtk

const pangoEllipsizeEnd = 3

// wrapGTKGtk_cell_renderer_text_new ends a text too long for its cell with an ellipsis; SWT's renderer clips it.
func wrapGTKGtk_cell_renderer_text_new(r int64) int64 {
	if r != 0 {
		OSG_object_setOverload3(r, cs("ellipsize"), pangoEllipsizeEnd, 0)
	}
	return r
}
