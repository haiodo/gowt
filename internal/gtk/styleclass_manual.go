//go:build linux

package gtk

import "unsafe"

var (
	gtkHasClass      = lz[func(uintptr, unsafe.Pointer) int32]{sym: "gtk_style_context_has_class"}
	gtkGetAccessible = lz[func(uintptr) uintptr]{sym: "gtk_widget_get_accessible"}
	atkSetName       = lz[func(uintptr, unsafe.Pointer)]{sym: "atk_object_set_name"}
	gtkButtonImage   = lz[func(uintptr) uintptr]{sym: "gtk_button_get_image"}
)

// SetButtonIconName shows the named theme icon in the image of a GTK 3 button that has one, such as
// SWT's arrow button. GTK 4 keeps the image.
func SetButtonIconName(button int64, name string) {
	if GTKGTK4 {
		return
	}
	if img := gtkButtonImage.get()(uintptr(button)); img != 0 {
		GTK3Gtk_image_set_from_icon_name(int64(img), cs(name), GTKGTK_ICON_SIZE_MENU)
	}
}

// AddStyleClass adds a CSS class such as "flat" or "circular" to the widget's style context.
func AddStyleClass(widget int64, class string) {
	GTKGtk_style_context_add_class(GTKGtk_widget_get_style_context(widget), cs(class))
}

// HasStyleClass reports whether the widget's style context has the CSS class.
func HasStyleClass(widget int64, class string) bool {
	return gtkHasClass.get()(uintptr(GTKGtk_widget_get_style_context(widget)), sp(cs(class))) != 0
}

// SetAccessibleName sets the name a screen reader announces for the widget (ATK).
func SetAccessibleName(widget int64, name string) {
	if a := gtkGetAccessible.get()(uintptr(widget)); a != 0 {
		atkSetName.get()(a, sp(cs(name)))
	}
}
