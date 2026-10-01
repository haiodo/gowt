// The Callback bridge (see README "Callback design" and "Round 20 (win32)").
package swt

import "github.com/haiodo/gowt/internal/win32"

// Callback: `new Callback(obj, "method", argCount)` is translated into NewCallbackFn with a Go closure
// calling the named method; the native half is win32.NewCallbackN.
type Callback struct {
	address int64
}

func NewCallbackFn(fn func(args []int64) int64, argCount int32) *Callback {
	return &Callback{address: win32.NewCallbackN(int(argCount), fn)}
}

func (c *Callback) GetAddress() int64 { return c.address }

// syscall callbacks cannot be freed; the slot stays allocated (pool of 2000, SWT creates ~25).
func (c *Callback) Dispose() {}

func CallbackGetEntryCount() int32 { return 0 }

// NewCallbackTyped: `new Callback(obj, "m", n)` whose method takes typed arguments (int32, ...).
func NewCallbackTyped(fn any) *Callback { return &Callback{address: win32.NewCallbackAny(fn)} }

// Callback.setEnabled: callbacks need no gate here.
func CallbackSetEnabled(enabled bool) {}
