//go:build windows && gowtcgo && cgo

package win32

import "testing"

// The cgo resolver's compiled-in DLL list must mirror dllNames (dll_manual.go), the
// default backend's Lookup order: a drift would resolve exports differently between
// the backends.

func TestResolverDLLListMatches(t *testing.T) {
	if n := gowtDLLcgoCount(); n != len(dllNames) {
		t.Fatalf("gowt_dll_names has %d entries, dllNames has %d", n, len(dllNames))
	}
	for i, want := range dllNames {
		if got := gowtDLLcgoName(i); got != want {
			t.Fatalf("gowt_dll_names[%d] = %q, dllNames[%d] = %q", i, got, i, want)
		}
	}
}
