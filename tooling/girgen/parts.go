package main

import (
	"strings"

	"github.com/haiodo/gowt/tooling/genplan"
)

// familyOf groups a generated binding name into a navigation family: the first name
// segment after the library prefix, so GTKGtk_tree_view_new -> tree, GTK4Gtk_widget_show
// -> widget, GTKGTK_IS_WIDGET -> is, OSG_object_ref -> object. Names without a segment
// keep their sanitized whole name.
func familyOf(name string) string {
	toks := strings.Split(name, "_")
	if len(toks) < 2 {
		return genplan.Sanitize(name)
	}
	return strings.ToLower(toks[1])
}

// ownerOf is the sub-family a split part is disambiguated by: the first two segments
// after the library prefix (tree_view, widget_show, ...).
func ownerOf(name string) string {
	toks := strings.Split(name, "_")
	if len(toks) < 3 {
		return strings.ToLower(name)
	}
	return strings.ToLower(toks[1] + "_" + toks[2])
}

// needsUnsafe mirrors the per-file import rule of the single-file layout: a part
// imports unsafe exactly when a selector on it appears in the text.
func needsUnsafe(text string) bool {
	i := 0
	for i < len(text) {
		j := strings.Index(text[i:], "unsafe.")
		if j < 0 {
			return false
		}
		start := i + j
		if start == 0 || !isIdentByte(text[start-1]) {
			return true
		}
		i = start + len("unsafe.")
	}
	return false
}

func isIdentByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}
