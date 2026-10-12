//go:build darwin

package cocoa

import "testing"

func BenchmarkMsgSend(b *testing.B) {
	cls := OSObjc_lookUpClass("NSObject")
	sel := OSSel_registerName("hash")
	for i := 0; i < b.N; i++ {
		OSObjc_msgSend(cls, sel)
	}
}

func BenchmarkMsgSendCallbackRoundTrip(b *testing.B) {
	fn := NewCallbackN(2, func(a []int64) int64 { return 0 })
	for i := 0; i < b.N; i++ {
		OSCall(fn, 1, 2)
	}
}
