//go:build darwin && cgo

package cgobench

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

func BenchmarkMsgSendPurego(b *testing.B) {
	class := cocoa.OSObjc_lookUpClass("NSObject")
	selector := cocoa.OSSel_registerName("hash")
	for i := 0; i < b.N; i++ {
		cocoa.OSObjc_msgSend(class, selector)
	}
}

func BenchmarkMsgSendCgo(b *testing.B) {
	class := cocoa.OSObjc_lookUpClass("NSObject")
	selector := cocoa.OSSel_registerName("hash")
	for i := 0; i < b.N; i++ {
		msgSend(uintptr(class), uintptr(selector))
	}
}

func BenchmarkCallbackRoundTripPurego(b *testing.B) {
	callback := cocoa.NewCallbackN(2, func([]int64) int64 { return 0 })
	for i := 0; i < b.N; i++ {
		cocoa.OSCall(callback, 1, 2)
	}
}

func BenchmarkCallbackRoundTripCgo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		callbackRoundTrip(1, 2)
	}
}
