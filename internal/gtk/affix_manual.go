//go:build linux

package gtk

import "fmt"

// EntryAffix is what SetEntryAffix shows inside the frame of a GtkEntry around its text.
type EntryAffix struct {
	Prefix, Suffix string
	Image          int64   // a cairo surface with its device scale, drawn instead of Prefix
	ImageW, ImageH float64 // its size in widget pixels
}

// affixGap is the space in pixels between an affix and the text.
const affixGap = 6

// dimLabel is the opacity of Adwaita's .dim-label, the secondary text of GTK.
const dimLabel = 0.55

type entryLook struct {
	provider                    int64
	pad                         GtkBorder // the theme's padding, before the affixes widened it
	small                       bool
	affix                       *EntryAffix
	lead                        int32 // pixels the prefix takes, with its gap
	trail                       int32
	draw                        int64 // the draw callback, 0 before SetEntryAffix
	handler                     int64
	styled                      int64 // the style-updated callback
	styledID                    int64
	changed                     func()
	fill                        bool
	red, green, blue            int32
	stroke                      bool
	sred, sgreen, sblue, salpha int32
	radius                      int
	padX                        int
}

var entryLooks = map[int64]*entryLook{}

// lookOf returns the state of entry; changed, the latest given, runs when GTK has applied a new style. GTK applies
// the CSS of SetEntrySmall and SetEntryAffix on the next frame, invalidating the style context does
// not hasten it: until then the entry reports its old padding.
func lookOf(entry int64, changed func()) *entryLook {
	l := entryLooks[entry]
	if l != nil {
		l.changed = changed
	} else {
		l = &entryLook{changed: changed}
		GTK3Gtk_style_context_get_padding(GTKGtk_widget_get_style_context(entry), GTKGtk_widget_get_state_flags(entry), &l.pad)
		entryLooks[entry] = l
		l.styled = NewCallbackN(2, func(args []int64) int64 {
			if l := entryLooks[args[0]]; l != nil && l.changed != nil {
				l.changed()
			}
			return 0
		})
		l.styledID = int64(gSignalConnectData.get()(uintptr(entry), sp(cs("style-updated")), uintptr(l.styled), 0, 0, gConnectAfter))
	}
	return l
}

// look returns the state of entry without replacing its changed callback.
func look(entry int64) *entryLook {
	if l := entryLooks[entry]; l != nil {
		return l
	}
	return lookOf(entry, nil)
}

// SetEntryFill gives the entry the fill and corner radius of a FieldFill, without the theme's frame.
func SetEntryFill(entry int64, red, green, blue int32, radius int) {
	l := look(entry)
	l.fill, l.red, l.green, l.blue, l.radius = true, red, green, blue, radius
	l.css(entry)
}

// SetEntryStroke draws a border of the RGBA colour around the entry fill.
func SetEntryStroke(entry int64, red, green, blue, alpha int32) {
	l := look(entry)
	l.stroke, l.sred, l.sgreen, l.sblue, l.salpha = true, red, green, blue, alpha
	l.css(entry)
}

// SetEntryInset widens the padding between the entry and its text by x pixels on each side.
func SetEntryInset(entry int64, x int) {
	l := look(entry)
	l.padX = x
	l.css(entry)
}

// SetEntrySmall gives the entry the compact height of an inspector field; changed runs once the
// entry has the new size.
func SetEntrySmall(entry int64, changed func()) {
	l := lookOf(entry, changed)
	l.small = true
	l.css(entry)
}

// SetEntryAffix pads the entry for a and draws a after the entry has drawn itself, on the line of its
// text, in the text colour at the opacity of .dim-label; changed runs once the entry has the new size.
// It returns the pixels the affixes add to the width, which the entry reports only from then on.
func SetEntryAffix(entry int64, a EntryAffix, changed func()) (width int32) {
	l := lookOf(entry, changed)
	l.affix = &a
	l.lead, l.trail = 0, 0
	if a.Image != 0 {
		l.lead = int32(a.ImageW+0.5) + affixGap
	} else if a.Prefix != "" {
		l.lead = textWidth(entry, a.Prefix) + affixGap
	}
	if a.Suffix != "" {
		l.trail = textWidth(entry, a.Suffix) + affixGap
	}
	l.css(entry)
	if l.draw == 0 {
		l.draw = NewCallbackN(3, func(args []int64) int64 {
			if l := entryLooks[args[0]]; l != nil && l.affix != nil {
				l.paint(args[0], args[1])
			}
			return 0
		})
		l.handler = int64(gSignalConnectData.get()(uintptr(entry), sp(cs("draw")), uintptr(l.draw), 0, 0, gConnectAfter))
	}
	GTKGtk_widget_queue_resize(entry)
	return l.lead + l.trail
}

// EntryValueStart is the x in widget pixels where the text of the entry starts after its prefix.
func EntryValueStart(entry int64) int32 {
	l := entryLooks[entry]
	if l == nil {
		return 0
	}
	var b GtkBorder
	GTK3Gtk_style_context_get_border(GTKGtk_widget_get_style_context(entry), GTKGtk_widget_get_state_flags(entry), &b)
	return b.Left + l.pad.Left + l.lead
}

// ReleaseEntryLook disconnects the drawing of the affixes; call it when the entry is destroyed.
func ReleaseEntryLook(entry int64) {
	l := entryLooks[entry]
	if l == nil {
		return
	}
	delete(entryLooks, entry)
	OSG_signal_handler_disconnect(entry, l.styledID)
	FreeCallback(l.styled)
	if l.draw != 0 {
		OSG_signal_handler_disconnect(entry, l.handler)
		FreeCallback(l.draw)
	}
}

func (l *entryLook) css(entry int64) {
	css := fmt.Sprintf("entry { padding-left: %dpx; padding-right: %dpx; }", l.pad.Left+l.lead+int32(l.padX), l.pad.Right+l.trail+int32(l.padX))
	if l.small {
		css += " entry { min-height: 24px; padding-top: 0; padding-bottom: 0; }"
	}
	if l.fill {
		css += fmt.Sprintf(" entry { background-color: rgb(%d,%d,%d); border: none; box-shadow: none; border-radius: %dpx; }", l.red, l.green, l.blue, l.radius)
	}
	if l.stroke {
		css += fmt.Sprintf(" entry { border: 1px solid rgba(%d,%d,%d,%.3f); border-radius: %dpx; }", l.sred, l.sgreen, l.sblue, float64(l.salpha)/255, l.radius)
	}
	l.provider = SetWidgetCSS(entry, css, l.provider)

}

func (l *entryLook) paint(entry, cr int64) {
	ctx := GTKGtk_widget_get_style_context(entry)
	state := GTKGtk_widget_get_state_flags(entry)
	var b GtkBorder
	GTK3Gtk_style_context_get_border(ctx, state, &b)
	var c GdkRGBA
	GTK3Gtk_style_context_get_color(ctx, state, &c)
	ys := make([]int32, 1)
	GTK3Gtk_entry_get_layout_offsets(entry, make([]int32, 1), ys)
	a := l.affix
	x := float64(b.Left + l.pad.Left)
	switch {
	case a.Image != 0:
		h := float64(lineHeight(entry))
		CairoCairo_set_source_surface(cr, a.Image, x, float64(ys[0])+(h-a.ImageH)/2)
		CairoCairo_paint_with_alpha(cr, 1)
	case a.Prefix != "":
		showText(entry, cr, a.Prefix, x, float64(ys[0]), c)
	}
	if a.Suffix != "" {
		var alloc GtkAllocation
		GTKGtk_widget_get_allocation(entry, &alloc)
		sx := alloc.Width - b.Right - l.pad.Right - (l.trail - affixGap)
		showText(entry, cr, a.Suffix, float64(sx), float64(ys[0]), c)
	}
}

// showText draws s in the entry's font with its top left corner at x, y.
func showText(entry, cr int64, s string, x, y float64, c GdkRGBA) {
	layout := GTKGtk_widget_create_pango_layout(entry, cs(s))
	defer OSG_object_unref(layout)
	CairoCairo_set_source_rgba(cr, c.Red, c.Green, c.Blue, c.Alpha*dimLabel)
	CairoCairo_move_to(cr, x, y)
	OSPango_cairo_show_layout(cr, layout)
}

func textWidth(entry int64, s string) int32 {
	layout := GTKGtk_widget_create_pango_layout(entry, cs(s))
	defer OSG_object_unref(layout)
	w, h := make([]int32, 1), make([]int32, 1)
	OSPango_layout_get_pixel_size(layout, w, h)
	return w[0]
}

func lineHeight(entry int64) int32 {
	layout := GTKGtk_widget_create_pango_layout(entry, cs("0"))
	defer OSG_object_unref(layout)
	w, h := make([]int32, 1), make([]int32, 1)
	OSPango_layout_get_pixel_size(layout, w, h)
	return h[0]
}
