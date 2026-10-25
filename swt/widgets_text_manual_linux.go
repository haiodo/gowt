package swt

import "github.com/haiodo/gowt/internal/gtk"

// UseThemeBackground: SWT bakes the colour read at creation into a frameless entry's CSS, which stays
// light when the portal switches the theme to dark afterwards; the theme colour follows it.
func (this *Text) UseThemeBackground() {
	if gtk.GTKGTK4 || (this.style&SINGLE) == 0 || (this.style&BORDER) != 0 {
		return
	}
	context := gtk.GTKGtk_widget_get_style_context(this.Handle)
	this.Gtk_css_provider_load_from_css(context, "entry {border: solid; background: @theme_base_color;}")
}
