//go:build windows

package win32

import "testing"

// A COMObject is a pointer to a vtable pointer; slot i of the vtable holds the thunk of callback<i>.
func TestCOMObjectVtableLayout(t *testing.T) {
	obj := NewCOMObject([]int32{2, 0, 0, 1})
	defer obj.Dispose()

	pVtable := make([]int64, 1)
	OSMoveMemoryOverload7(pVtable, obj.PpVtable, CPTR_SIZEOF)
	if pVtable[0] == 0 {
		t.Fatal("vtable pointer is nil")
	}
	slots := make([]int64, 4)
	OSMoveMemoryOverload7(slots, pVtable[0], 4*CPTR_SIZEOF)
	seen := map[int64]bool{}
	for i, s := range slots {
		if s == 0 || seen[s] {
			t.Errorf("slot %d = %#x: nil or shared with another slot", i, s)
		}
		seen[s] = true
	}
}
