package main

import (
	"strings"
	"testing"
)

func TestCgoFileText(t *testing.T) {
	natives := []cgoNative{
		{name: "GTKGtk_widget_show", csym: "gtk_widget_show", params: []string{"uintptr"}},
		{name: "CairoCairo_matrix_init", csym: "cairo_matrix_init", params: []string{"unsafe.Pointer", "float64", "float64", "float64", "float64", "float64", "float64"}},
		{name: "GDKGdk_screen_get_resolution", csym: "gdk_screen_get_resolution", params: []string{"uintptr"}, ret: "float64"},
		{name: "OSG_object_ref", csym: "g_object_ref", params: []string{"uintptr"}, ret: "uintptr"},
		{name: "GTKGtk_get_major_version", csym: "gtk_get_major_version", ret: "int32"},
		{name: "GTKGtk_init_check", csym: "gtk_init_check", params: []string{"unsafe.Pointer", "unsafe.Pointer"}, ret: "int32"},
		{name: "GDKGdk_flush", csym: "gdk_flush"},
	}
	got := cgoFileText(natives)
	for _, want := range []string{
		"//go:build linux && gowtcgo && cgo",
		"package gtk",
		"#include <stdint.h>",
		"static void gowt_GTKGtk_widget_show(uintptr_t fn, uintptr_t a0) {\n\t((void (*)(uintptr_t))fn)(a0);\n}",
		"static double gowt_GDKGdk_screen_get_resolution(uintptr_t fn, uintptr_t a0) {\n\treturn ((double (*)(uintptr_t))fn)(a0);\n}",
		"static int32_t gowt_GTKGtk_get_major_version(uintptr_t fn) {\n\treturn ((int32_t (*)(void))fn)();\n}",
		"static void gowt_GDKGdk_flush(uintptr_t fn) {\n\t((void (*)(void))fn)();\n}",
		`func cgoBind_CairoCairo_matrix_init(addr uintptr) func(a0 unsafe.Pointer, a1 float64, a2 float64, a3 float64, a4 float64, a5 float64, a6 float64) {`,
		"return uintptr(C.gowt_OSG_object_ref(C.uintptr_t(addr), C.uintptr_t(a0)))",
		"return int32(C.gowt_GTKGtk_get_major_version(C.uintptr_t(addr)))",
		`cgoNativeRegistry["GTKGtk_widget_show"] = cgoBind_GTKGtk_widget_show`,
		"import \"unsafe\"",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("cgo file missing %q\n---\n%s", want, got)
		}
	}
	// void wrappers must not return.
	if strings.Contains(got, "return ((void (*)(uintptr_t))fn)(a0);") {
		t.Error("void wrapper returns a value")
	}
}

func TestCgoFileTextNoUnsafe(t *testing.T) {
	got := cgoFileText([]cgoNative{{name: "GTKGtk_get_major_version", csym: "gtk_get_major_version", ret: "int32"}})
	if strings.Contains(got, "import \"unsafe\"") {
		t.Error("file without pointer params must not import unsafe")
	}
	if !strings.Contains(got, "import \"C\"") {
		t.Error("file must import C")
	}
}

func TestCgoFileTextEmpty(t *testing.T) {
	if got := cgoFileText(nil); got != "" {
		t.Errorf("no natives must give no file, got %q", got)
	}
}

func TestCgoCTypePanicsOnUnknown(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("unknown raw type must panic")
		}
	}()
	cgoFileText([]cgoNative{{name: "X", csym: "x", params: []string{"string"}}})
}

// The default wrapper's lz variable must carry the binding name the cgo registry
// resolves (sym stays for the purego backend and the cgo symbol check).
func TestFuncChunkCarriesBindingName(t *testing.T) {
	g := &gen{shapes: &Shapes{Wrap: map[string]bool{}}, unpacks: map[string]bool{}, natives: map[string][]cgoNative{}}
	sig := Sig{Params: []GoParam{{Name: "obj", Type: "int64"}}, Ret: "bool", CRet: "int32"}
	ch := g.funcChunk("GTKGtk_widget_show", "gtk_widget_show", sig)
	got := ch.Text
	if !strings.Contains(got, `var r_GTKGtk_widget_show = lz[func(uintptr) int32]{name: "GTKGtk_widget_show", sym: "gtk_widget_show"}`) {
		t.Errorf("lz var misses name field:\n%s", got)
	}
	nat := g.natives[got]
	if len(nat) != 1 {
		t.Fatalf("funcChunk recorded %d natives, want 1", len(nat))
	}
	if nat[0].name != "GTKGtk_widget_show" || nat[0].csym != "gtk_widget_show" || nat[0].ret != "int32" || len(nat[0].params) != 1 || nat[0].params[0] != "uintptr" {
		t.Errorf("unexpected native record: %+v", nat[0])
	}
}
