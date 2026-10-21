//go:build linux

package gtk

var (
	gtkTextViewSetLeftMargin   = lz[func(uintptr, int32)]{sym: "gtk_text_view_set_left_margin"}
	gtkTextViewSetRightMargin  = lz[func(uintptr, int32)]{sym: "gtk_text_view_set_right_margin"}
	gtkTextViewSetTopMargin    = lz[func(uintptr, int32)]{sym: "gtk_text_view_set_top_margin"}
	gtkTextViewSetBottomMargin = lz[func(uintptr, int32)]{sym: "gtk_text_view_set_bottom_margin"}
)

const pangoScale = 1024

// SetTextViewMargins puts x between the text and the left and right edge, y above and below it.
func SetTextViewMargins(view int64, x, y int32) {
	gtkTextViewSetLeftMargin.get()(uintptr(view), x)
	gtkTextViewSetRightMargin.get()(uintptr(view), x)
	gtkTextViewSetTopMargin.get()(uintptr(view), y)
	gtkTextViewSetBottomMargin.get()(uintptr(view), y)
}

// TextViewContentHeight is the height text takes in the font of view, wrapped by words and then
// characters in width pixels; an empty text counts one line. GtkTextView measures only at the width it
// has, so a Pango layout of the view's own context stands in for it.
func TextViewContentHeight(view int64, text string, width int32) int32 {
	layout := GTKGtk_widget_create_pango_layout(view, cs(text))
	defer OSG_object_unref(layout)
	OSPango_layout_set_width(layout, max(width, 1)*pangoScale)
	OSPango_layout_set_wrap(layout, OSPANGO_WRAP_WORD_CHAR)
	// GtkTextView rounds each line up to whole pixels; the pixel size of the whole layout rounds once
	// and comes out a few pixels short for several lines.
	var total int32
	for i := range OSPango_layout_get_line_count(layout) {
		var logical PangoRectangle
		OSPango_layout_line_get_extents(OSPango_layout_get_line(layout, i), nil, &logical)
		total += (logical.Height + pangoScale - 1) / pangoScale
	}
	return total
}
