#include <stdint.h>
#include <objc/message.h>

uintptr_t gowtBenchMsgSend(uintptr_t object, uintptr_t selector) {
	return ((uintptr_t (*)(uintptr_t, uintptr_t))objc_msgSend)(object, selector);
}

extern uintptr_t GowtBenchGoCallback(uintptr_t arg0, uintptr_t arg1);

uintptr_t gowtBenchCallbackRoundTrip(uintptr_t arg0, uintptr_t arg1) {
	return GowtBenchGoCallback(arg0, arg1);
}
