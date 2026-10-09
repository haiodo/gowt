//go:build windows

// Hand-written half of COMObject.java: its constructor picks a numbered static method callbackN by name through
// Callback's reflection, which has no Go form (see InvocationEmitter.emitCallback).
package win32

// Callback is the address of a stdcall thunk made by NewCallbackN.
type Callback struct {
	address int64
}

func (c *Callback) GetAddress() int64 { return c.address }

// NewLONG is the key of COMObject's ObjectMap; an int64 compares by value where the pointer to Java's LONG would not.
func NewLONG(value int64) int64 { return value }

var comObjectCallbacks = [...]func([]int64) int64{COMObjectCallback0, COMObjectCallback1, COMObjectCallback2, COMObjectCallback3, COMObjectCallback4, COMObjectCallback5, COMObjectCallback6, COMObjectCallback7, COMObjectCallback8, COMObjectCallback9, COMObjectCallback10, COMObjectCallback11, COMObjectCallback12, COMObjectCallback13, COMObjectCallback14, COMObjectCallback15, COMObjectCallback16, COMObjectCallback17, COMObjectCallback18, COMObjectCallback19, COMObjectCallback20, COMObjectCallback21, COMObjectCallback22, COMObjectCallback23, COMObjectCallback24, COMObjectCallback25, COMObjectCallback26, COMObjectCallback27, COMObjectCallback28, COMObjectCallback29, COMObjectCallback30, COMObjectCallback31, COMObjectCallback32, COMObjectCallback33, COMObjectCallback34, COMObjectCallback35, COMObjectCallback36, COMObjectCallback37, COMObjectCallback38, COMObjectCallback39, COMObjectCallback40, COMObjectCallback41, COMObjectCallback42, COMObjectCallback43, COMObjectCallback44, COMObjectCallback45, COMObjectCallback46, COMObjectCallback47, COMObjectCallback48, COMObjectCallback49, COMObjectCallback50, COMObjectCallback51, COMObjectCallback52, COMObjectCallback53, COMObjectCallback54, COMObjectCallback55, COMObjectCallback56, COMObjectCallback57, COMObjectCallback58, COMObjectCallback59, COMObjectCallback60, COMObjectCallback61, COMObjectCallback62, COMObjectCallback63, COMObjectCallback64, COMObjectCallback65, COMObjectCallback66, COMObjectCallback67, COMObjectCallback68, COMObjectCallback69, COMObjectCallback70, COMObjectCallback71, COMObjectCallback72, COMObjectCallback73, COMObjectCallback74, COMObjectCallback75, COMObjectCallback76, COMObjectCallback77, COMObjectCallback78, COMObjectCallback79}

// COMObjectNewCallback returns the thunk of callback<index> taking argCount words (the COM object pointer included).
func COMObjectNewCallback(prefix string, index int32, argCount int32) *Callback {
	return &Callback{address: NewCallbackN(int(argCount), comObjectCallbacks[index])}
}
