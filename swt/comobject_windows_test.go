package swt

import (
	"testing"

	"github.com/haiodo/gowt/internal/win32"
)

// A COMObject is a pointer to a vtable pointer; slot i of the vtable holds the thunk of callback<i>.
func TestCOMObjectVtableLayout(t *testing.T) {
	obj := NewCOMObject([]int32{2, 0, 0, 1})
	defer obj.Dispose()

	pVtable := make([]int64, 1)
	win32.OSMoveMemoryOverload7(pVtable, obj.PpVtable, win32.CPTR_SIZEOF)
	if pVtable[0] == 0 {
		t.Fatal("vtable pointer is nil")
	}
	slots := make([]int64, 4)
	win32.OSMoveMemoryOverload7(slots, pVtable[0], 4*win32.CPTR_SIZEOF)
	seen := map[int64]bool{}
	for i, s := range slots {
		if s == 0 || seen[s] {
			t.Errorf("slot %d = %#x: nil or shared with another slot", i, s)
		}
		seen[s] = true
	}
}

type comObjectMethod9 struct {
	COMObject
}

func (this *comObjectMethod9) method9_(args []int64) int64 { return args[0] + 1 }

// Accessible's anonymous COMObjects override methods no named subclass does (method9 is get_accChild): the
// vtable thunk must still reach the override, not COMObject's E_NOTIMPL.
func TestCOMObjectCallbackReachesAnonymousOverride(t *testing.T) {
	obj := &comObjectMethod9{}
	obj.impl = obj
	obj.initCOMObject([]int32{2, 0, 0, 1, 1, 1, 1, 1, 1, 12})
	defer obj.Dispose()
	if got := COMObjectCallback9([]int64{obj.PpVtable, 41}); got != 42 {
		t.Fatalf("callback9 = %#x, want 42 from the override", got)
	}
}
