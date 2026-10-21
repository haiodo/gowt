//go:build linux

package gtk

import "unsafe"

var (
	gtkHasClass      = lz[func(uintptr, unsafe.Pointer) int32]{sym: "gtk_style_context_has_class"}
	gtkGetAccessible = lz[func(uintptr) uintptr]{sym: "gtk_widget_get_accessible"}
	atkSetName       = lz[func(uintptr, unsafe.Pointer)]{sym: "atk_object_set_name"}
)

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
