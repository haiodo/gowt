//go:build darwin && cgo

package cgobench

/*
#cgo LDFLAGS: -framework Foundation
#include <stdint.h>

uintptr_t gowtBenchMsgSend(uintptr_t object, uintptr_t selector);
uintptr_t gowtBenchCallbackRoundTrip(uintptr_t arg0, uintptr_t arg1);
*/
import "C"

//export GowtBenchGoCallback
func GowtBenchGoCallback(arg0, arg1 C.uintptr_t) C.uintptr_t {
	return 0
}

func msgSend(object, selector uintptr) uintptr {
	return uintptr(C.gowtBenchMsgSend(C.uintptr_t(object), C.uintptr_t(selector)))
}

func callbackRoundTrip(arg0, arg1 uintptr) uintptr {
	return uintptr(C.gowtBenchCallbackRoundTrip(C.uintptr_t(arg0), C.uintptr_t(arg1)))
}
