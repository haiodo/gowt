//go:build windows

package win32

import "testing"

// ScriptStringOut takes the handle ScriptStringAnalyse stored, not the address of its slot.
func TestScriptStringOut(t *testing.T) {
	hdc := OSGetDC(0)
	if hdc == 0 {
		t.Skip("no screen DC")
	}
	defer OSReleaseDC(0, hdc)
	heap := OSGetProcessHeap()
	chars := []uint16{'a', 'b'}
	str := OSHeapAlloc(heap, OSHEAP_ZERO_MEMORY, 4)
	defer OSHeapFree(heap, 0, str)
	OSMoveMemoryOverload12(str, chars, 4)
	ssa := OSHeapAlloc(heap, OSHEAP_ZERO_MEMORY, OSSCRIPT_STRING_ANALYSIS_sizeof())
	defer OSHeapFree(heap, 0, ssa)
	mdc := OSCreateEnhMetaFile(hdc, nil, nil, nil)
	if rc := OSScriptStringAnalyse(mdc, str, 2, 0, -1, OSSSA_METAFILE|OSSSA_FALLBACK|OSSSA_GLYPHS|OSSSA_LINK, 0, nil, nil, 0, 0, 0, ssa); rc != OSS_OK {
		t.Fatalf("ScriptStringAnalyse = %#x", rc)
	}
	if rc := OSScriptStringOut(ssa, 0, 0, 0, nil, 0, 0, false); rc != OSS_OK {
		t.Errorf("ScriptStringOut = %#x, want S_OK", rc)
	}
	OSScriptStringFree(ssa)
	OSDeleteEnhMetaFile(OSCloseEnhMetaFile(mdc))
}
