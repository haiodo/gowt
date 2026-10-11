// The Callback bridge (see README "Callback design" and "Round 20 (win32)").
package swt

import (
	"sync"

	"github.com/haiodo/gowt/internal/win32"
)

// Callback: `new Callback(obj, "method", argCount)` is translated into NewCallbackFn with a Go closure
// calling the named method; the native half is win32.NewCallbackN.
type Callback struct {
	address int64
	slot    *callbackSlot
}

// callbackSlot is one native callback (syscall slots are never freed, pool of 2000) whose Go target can be swapped,
// so Dispose can hand the slot to the next NewCallbackFn with the same argCount.
type callbackSlot struct {
	argCount int32
	address  int64
	fn       func(args []int64) int64
}

var (
	callbackSlotsMu   sync.Mutex
	callbackSlotsFree = map[int32][]*callbackSlot{}
)

// Ceiling: a window procedure still registered at the address after Dispose would reach the next owner of the slot.
func NewCallbackFn(fn func(args []int64) int64, argCount int32) *Callback {
	callbackSlotsMu.Lock()
	var slot *callbackSlot
	if free := callbackSlotsFree[argCount]; len(free) > 0 {
		slot = free[len(free)-1]
		callbackSlotsFree[argCount] = free[:len(free)-1]
	}
	callbackSlotsMu.Unlock()
	if slot == nil {
		slot = &callbackSlot{argCount: argCount}
		slot.address = win32.NewCallbackN(int(argCount), func(args []int64) int64 {
			callbackSlotsMu.Lock()
			target := slot.fn
			callbackSlotsMu.Unlock()
			if target == nil {
				return 0
			}
			return target(args)
		})
	}
	callbackSlotsMu.Lock()
	slot.fn = fn
	callbackSlotsMu.Unlock()
	return &Callback{address: slot.address, slot: slot}
}

func (c *Callback) GetAddress() int64 { return c.address }

// Typed callbacks (NewCallbackTyped) keep their slot.
func (c *Callback) Dispose() {
	if c.slot == nil {
		return
	}
	callbackSlotsMu.Lock()
	defer callbackSlotsMu.Unlock()
	c.slot.fn = nil
	callbackSlotsFree[c.slot.argCount] = append(callbackSlotsFree[c.slot.argCount], c.slot)
	c.slot = nil
}

func CallbackGetEntryCount() int32 { return 0 }

// NewCallbackTyped: `new Callback(obj, "m", n)` whose method takes typed arguments (int32, ...).
func NewCallbackTyped(fn any) *Callback { return &Callback{address: win32.NewCallbackAny(fn)} }

// Callback.setEnabled: callbacks need no gate here.
func CallbackSetEnabled(enabled bool) {}
