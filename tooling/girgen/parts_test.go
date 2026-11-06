package main

import "testing"

func TestFamilyOf(t *testing.T) {
	cases := map[string]string{
		"GTKGtk_tree_view_new":     "tree",
		"GTK3Gtk_widget_show":      "widget",
		"GTK4Gtk_widget_show":      "widget",
		"GTK4Gdk_content_register": "content",
		"GDKGdk_window_new":        "window",
		"OSG_object_ref":           "object",
		"OSG_dbus_connection_new":  "dbus",
		"CairoCairo_set_source":    "set",
		"OSPango_layout_get":       "layout",
		"GTKGTK_IS_WIDGET":         "is",
		"GrapheneGraphene_vec":     "vec",
		"OSXSynchronize":           "osxsynchronize",
	}
	for name, want := range cases {
		if got := familyOf(name); got != want {
			t.Errorf("familyOf(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestOwnerOf(t *testing.T) {
	cases := map[string]string{
		"GTKGtk_tree_view_new": "tree_view",
		"GTKGtk_widget_show":   "widget_show",
		"OSXSynchronize":       "osxsynchronize",
	}
	for name, want := range cases {
		if got := ownerOf(name); got != want {
			t.Errorf("ownerOf(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNeedsUnsafe(t *testing.T) {
	if !needsUnsafe("var r_x = lz[func(unsafe.Pointer)]{}\n") {
		t.Error("selector on unsafe not detected")
	}
	if needsUnsafe("func f() { myunsafe.Value() }\n") {
		t.Error("identifier merely ending in the package name needs no import")
	}
	if needsUnsafe("func f() { g(x) }\n") {
		t.Error("plain body needs no import")
	}
}
