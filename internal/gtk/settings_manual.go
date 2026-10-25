//go:build linux

package gtk

import "unsafe"

var (
	gSettingsSchemaSourceGetDefault = lz[func() uintptr]{sym: "g_settings_schema_source_get_default"}
	gSettingsSchemaSourceLookup     = lz[func(uintptr, unsafe.Pointer, int32) uintptr]{sym: "g_settings_schema_source_lookup"}
	gSettingsSchemaHasKey           = lz[func(uintptr, unsafe.Pointer) int32]{sym: "g_settings_schema_has_key"}
	gSettingsSchemaUnref            = lz[func(uintptr)]{sym: "g_settings_schema_unref"}
	gSettingsNew                    = lz[func(unsafe.Pointer) uintptr]{sym: "g_settings_new"}
	gSettingsGetString              = lz[func(uintptr, unsafe.Pointer) uintptr]{sym: "g_settings_get_string"}
)

// GSettingsString reads a string key of an installed schema; ok is false when the schema or the key is
// missing (g_settings_new would abort on a missing schema).
func GSettingsString(schema, key string) (value string, ok bool) {
	src := gSettingsSchemaSourceGetDefault.get()()
	if src == 0 {
		return "", false
	}
	sch := gSettingsSchemaSourceLookup.get()(src, sp(cs(schema)), 1)
	if sch == 0 {
		return "", false
	}
	defer gSettingsSchemaUnref.get()(sch)
	if gSettingsSchemaHasKey.get()(sch, sp(cs(key))) == 0 {
		return "", false
	}
	st := gSettingsNew.get()(sp(cs(schema)))
	if st == 0 {
		return "", false
	}
	defer OSG_object_unref(int64(st))
	p := gSettingsGetString.get()(st, sp(cs(key)))
	if p == 0 {
		return "", false
	}
	defer OSG_free(int64(p))
	return ConverterCCharPtrToJavaString(int64(p), false), true
}

// PangoFamily is the family of a Pango font description string such as "Ubuntu Sans Mono 11".
func PangoFamily(desc string) string {
	d := OSPango_font_description_from_string(cs(desc))
	if d == 0 {
		return ""
	}
	defer OSPango_font_description_free(d)
	return ConverterCCharPtrToJavaString(OSPango_font_description_get_family(d), false)
}

var gtkFrameSetLabelAlign = lz[func(uintptr, float32, float32)]{sym: "gtk_frame_set_label_align"}

// SetFrameLabelAlign positions the label of a GtkFrame; yalign 0 puts it above the frame line.
func SetFrameLabelAlign(frame int64, xalign, yalign float32) {
	gtkFrameSetLabelAlign.get()(uintptr(frame), xalign, yalign)
}
