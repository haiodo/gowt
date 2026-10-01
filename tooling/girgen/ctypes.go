package main

import "strings"

// Go-side kinds of the generated wrappers. Handles and pointers are int64 (SWT stores them as
// Java longs), C ints int32, gboolean bool, C strings []int8, out arrays slices.
var prim = map[string]string{
	"gint": "int32", "int": "int32", "guint": "int32", "unsigned int": "int32", "unsigned": "int32",
	"gint32": "int32", "guint32": "int32", "gunichar": "int32", "GQuark": "int32", "uint32_t": "int32",
	"gshort": "int16", "gushort": "int16", "gint16": "int16", "guint16": "int16", "short": "int16",
	"gchar": "int8", "char": "int8", "guchar": "int8", "gint8": "int8", "guint8": "int8", "unsigned char": "int8",
	"gint64": "int64", "guint64": "int64", "glong": "int64", "gulong": "int64", "long": "int64", "unsigned long": "int64",
	"gsize": "int64", "gssize": "int64", "goffset": "int64", "gintptr": "int64", "guintptr": "int64", "GType": "int64",
	"size_t": "int64", "ssize_t": "int64", "gpointer": "int64", "gconstpointer": "int64", "Window": "int64", "Atom": "int64",
	"Time": "int64", "Drawable": "int64", "Pixmap": "int64", "Cursor": "int64", "KeySym": "int64", "XID": "int64",
	"gdouble": "float64", "double": "float64", "gfloat": "float32", "float": "float32",
	"gboolean": "bool", "cairo_bool_t": "bool", "Bool": "bool", "bool": "bool",
	"time_t": "int64", "pid_t": "int32", "GPid": "int32", "uid_t": "int32",
	"guint8*": "", "void": "",
}

// normCType drops const/volatile, struct and spacing so "const GtkWidget *" becomes ("GtkWidget", 1).
func normCType(ct string) (string, int) {
	ct = strings.ReplaceAll(ct, "*", " * ")
	var out []string
	stars := 0
	for _, f := range strings.Fields(ct) {
		switch f {
		case "const", "volatile", "struct", "restrict", "G_GNUC_MAY_ALIAS", "_Xconst":
		case "*":
			stars++
		default:
			out = append(out, f)
		}
	}
	return strings.Join(out, " "), stars
}

// primOf resolves aliases and enums down to a primitive Go type name, "" when it is a struct or opaque.
func (ix *Index) primOf(base string) string {
	for i := 0; i < 8; i++ {
		if t, ok := prim[base]; ok {
			return t
		}
		if ix.Enums[base] {
			return "int32"
		}
		t, ok := ix.Aliases[base]
		if !ok || t == base {
			break
		}
		if strings.Contains(t, "*") {
			return "int64"
		}
		base = t
	}
	if _, ok := prim[base]; ok {
		return prim[base]
	}
	return ""
}

// structOf returns the record behind a (possibly aliased) C type name.
func (ix *Index) structOf(base string) *Record {
	for i := 0; i < 8; i++ {
		if r, ok := ix.Types[base]; ok {
			return r
		}
		t, ok := ix.Aliases[base]
		if !ok || t == base {
			return nil
		}
		base = t
	}
	return nil
}
