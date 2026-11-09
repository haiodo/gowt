//go:build windows && gowtcgo && cgo

// Hand-written half of the opt-in cgo backend (tooling/j2go README "Native methods"):
// the export resolver behind every generated <unit>_cgo.go wrapper. gowt_find mirrors
// dll_manual.go's Lookup - the W variant first, across the package's DLL list, missing
// dynamic exports resolving to NULL (the generated wrapper then returns 0).
//
// The DLL list is compiled into this file, not passed from Go at package init: generated
// Java statics (GUID constants in win32_com.go) call natives from package-level variable
// initialization, whose order relative to any other var is not guaranteed - the resolver
// must be ready on the very first call. internal/win32 tests compare the list against
// dllNames (dll_manual.go), the default backend's single source of truth.

package win32

/*
#include <stdlib.h>
#include <stdint.h>

typedef void *HMODULE;
__declspec(dllimport) HMODULE __stdcall GetModuleHandleW(const unsigned short *name);
__declspec(dllimport) HMODULE __stdcall LoadLibraryW(const unsigned short *name);
__declspec(dllimport) void *__stdcall GetProcAddress(HMODULE mod, const char *name);

// Keep in sync with dllNames (dll_manual.go): dllcgo_list_test.go compares the two.
static const char *gowt_dll_names[] = {
	"user32.dll", "gdi32.dll", "kernel32.dll", "comctl32.dll", "comdlg32.dll", "shell32.dll",
	"ole32.dll", "oleaut32.dll", "uxtheme.dll", "imm32.dll", "msimg32.dll", "advapi32.dll",
	"usp10.dll", "dwmapi.dll", "shlwapi.dll", "oleacc.dll", "wininet.dll", "propsys.dll",
	"urlmon.dll", "shcore.dll", "winspool.drv", "gdiplus.dll", "ucrtbase.dll", "ntdll.dll",
};
#define gowt_dll_count (int)(sizeof(gowt_dll_names) / sizeof(gowt_dll_names[0]))

static HMODULE gowt_dlls[32];
static int gowt_dlls_ready;

// Loads lazily on the first export lookup: package-level initializers of generated files
// call natives before any init() runs. Concurrent first calls race benignly - every
// writer computes the same handles.
static void gowt_dlls_load(void) {
	for (int i = 0; i < gowt_dll_count; i++) {
		unsigned short w[64];
		const char *s = gowt_dll_names[i];
		int j = 0;
		while (s[j] != 0 && j < 63) { w[j] = (unsigned char)s[j]; j++; }
		w[j] = 0;
		HMODULE m = GetModuleHandleW(w);
		if (m == 0) m = LoadLibraryW(w);
		gowt_dlls[i] = m;
	}
	gowt_dlls_ready = 1;
}

void *gowt_find(const char *name, int dynamic) {
	(void)dynamic;
	if (!gowt_dlls_ready) gowt_dlls_load();
	// Every export name fits; Lookup's W variant goes first.
	char buf[128];
	for (int suffix = 0; suffix < 2; suffix++) {
		int j = 0;
		while (name[j] != 0 && j < 120) { buf[j] = name[j]; j++; }
		if (suffix == 0) buf[j++] = 'W';
		buf[j] = 0;
		for (int i = 0; i < gowt_dll_count; i++) {
			if (gowt_dlls[i] == 0) continue;
			void *p = GetProcAddress(gowt_dlls[i], buf);
			if (p != 0) return p;
		}
	}
	return 0;
}

// dllcgo_list_test.go compares the compiled-in list with dllNames.
int gowt_dllcgo_count(void) { return gowt_dll_count; }

const char *gowt_dllcgo_name(int i) {
	return (i >= 0 && i < gowt_dll_count) ? gowt_dll_names[i] : 0;
}
*/
import "C"

func gowtDLLcgoCount() int { return int(C.gowt_dllcgo_count()) }

func gowtDLLcgoName(i int) string {
	p := C.gowt_dllcgo_name(C.int(i))
	if p == nil {
		return ""
	}
	return C.GoString(p)
}
