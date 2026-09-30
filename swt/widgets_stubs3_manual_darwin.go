// Round 6 manual stubs: the Callback bridge (see README "Callback design") and the cocoa-only
// org.eclipse.swt.internal helpers; the platform-neutral stand-ins are in widgets_stubs3_manual.go.
package swt

import (
	"github.com/haiodo/gowt/internal/cocoa"
)

// Callback: `new Callback(obj, "method", argCount)` is translated into NewCallbackFn with a Go
// closure calling the named method; the native half lives in internal/cocoa/callback_manual.go.
type Callback struct {
	address int64
}

func NewCallbackFn(fn func(args []int64) int64, argCount int32) *Callback {
	return &Callback{address: cocoa.NewCallbackN(int(argCount), fn)}
}

func (c *Callback) GetAddress() int64 { return c.address }

// purego callbacks can't be freed; the slot stays allocated (pool of 2000, SWT creates ~20).
func (c *Callback) Dispose() {}

// Callback.getEntryCount: JNI re-entry depth, only compared to 0 to decide on autorelease pools.
func CallbackGetEntryCount() int32 { return 0 }

// BidiUtil (emulated/bidi): no bidi support, text direction is left as is.
func BidiUtilResolveTextDirection(text string) int32 { return NONE }
