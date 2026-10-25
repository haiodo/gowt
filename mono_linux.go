package gowt

import "github.com/haiodo/gowt/internal/gtk"

// gsettingsMono reads the GNOME monospace font family in process (no gsettings binary needed), "Monospace"
// (fontconfig's alias) without GNOME. Ceiling: a sandbox without the schema falls back; the portal is the way out.
func gsettingsMono() string {
	v, ok := gtk.GSettingsString("org.gnome.desktop.interface", "monospace-font-name")
	if !ok {
		return "Monospace"
	}
	return monoFamily(v)
}
