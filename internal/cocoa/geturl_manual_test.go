//go:build darwin

package cocoa

import (
	"runtime"
	"testing"
	"unsafe"
)

// A GURL Apple event dispatched by NSAppleEventManager, as for a link click, reaches fn.
func TestHandleGetURL(t *testing.T) {
	onMain(func() {
		var got []string
		HandleGetURL(func(u string) { got = append(got, u) })
		ev := msg(class("NSAppleEventDescriptor"), "appleEventWithEventClass:eventID:targetDescriptor:returnID:transactionID:",
			aeGetURL, aeGetURL, 0, ^uintptr(0), 0)
		msg(ev, "setParamDescriptor:forKeyword:", msg(class("NSAppleEventDescriptor"), "descriptorWithString:", nsString("tenniarb://open?id=1")), keyDirectObject)
		reply := [3]uint32{'n'<<24 | 'u'<<16 | 'l'<<8 | 'l'} // AEDesc, #pragma pack(2): type, then the handle at offset 4
		mgr := msg(class("NSAppleEventManager"), "sharedAppleEventManager")
		if err := int16(msg(mgr, "dispatchRawAppleEvent:withRawReply:handlerRefCon:", msg(ev, "aeDesc"), uintptr(unsafe.Pointer(&reply)), 0)); err != 0 {
			t.Fatalf("dispatchRawAppleEvent: OSErr %d", err)
		}
		runtime.KeepAlive(&reply)
		if len(got) != 1 || got[0] != "tenniarb://open?id=1" {
			t.Errorf("handler got %q, want the one URL", got)
		}
	})
}
