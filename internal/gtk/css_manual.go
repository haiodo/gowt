//go:build linux

package gtk

// USER, because SWT's per-widget colour providers sit at APPLICATION and would win over these rules.
const cssPriorityApplication = 800

func newProvider(css string) int64 {
	p := GTKGtk_css_provider_new()
	GTK3Gtk_css_provider_load_from_data(p, cs(css), -1, nil)
	return p
}

// AddGlobalCSS applies css to every widget of the default screen.
func AddGlobalCSS(css string) {
	GTK3Gtk_style_context_add_provider_for_screen(GDKGdk_screen_get_default(), newProvider(css), cssPriorityApplication)
}

// SetWidgetCSS loads css for the widget alone; the returned provider can be passed back to replace it.
func SetWidgetCSS(widget int64, css string, provider int64) int64 {
	if provider == 0 {
		provider = newProvider(css)
		GTKGtk_style_context_add_provider(GTKGtk_widget_get_style_context(widget), provider, cssPriorityApplication)
		return provider
	}
	GTK3Gtk_css_provider_load_from_data(provider, cs(css), -1, nil)
	return provider
}

// RemoveStyleClass removes a CSS class from the widget's style context.
func RemoveStyleClass(widget int64, class string) {
	GTKGtk_style_context_remove_class(GTKGtk_widget_get_style_context(widget), cs(class))
}

// RenderFrame draws the CSS border of the widget over a w x h area of cr, from the origin.
func RenderFrame(widget, cr int64, w, h float64) {
	GTKGtk_render_frame(GTKGtk_widget_get_style_context(widget), cr, 0, 0, w, h)
}

// Parent is the GTK parent of the widget.
func Parent(widget int64) int64 { return GTKGtk_widget_get_parent(widget) }

// SetScrollPolicyAutomatic shows the bars of a scrolled window on overflow only, as overlay bars.
func SetScrollPolicyAutomatic(scrolled int64) {
	GTKGtk_scrolled_window_set_policy(scrolled, GTKGTK_POLICY_AUTOMATIC, GTKGTK_POLICY_AUTOMATIC)
	GTKGtk_scrolled_window_set_overlay_scrolling(scrolled, true)
}
