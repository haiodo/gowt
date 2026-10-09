//go:build linux

package webkit

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

// JavaScriptCore installs its own signal handlers (GC thread suspension, SIGSEGV/SIGBUS faults) without
// SA_ONSTACK, some of them lazily on the first JS evaluation, and Go dies with "non-Go code set up
// signal handler without SA_ONSTACK flag" on the next signal it handles. fixSignalFlags adds the flag
// after every point where WebKit may have installed one.
const (
	saOnstack = 0x08000000
	// glibc struct sigaction: handler at 0, 128-byte mask, int flags at 136, restorer at 144.
	sigactionSize  = 152
	sigactionFlags = 136
)

var libcSigaction func(sig int32, act, old unsafe.Pointer) int32

func prepareSignals() {
	libc, err := purego.Dlopen("libc.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&libcSigaction, libc, "sigaction")
}

func fixSignalFlags() {
	if libcSigaction == nil {
		return
	}
	for sig := int32(1); sig <= 64; sig++ {
		var sa [sigactionSize]byte
		if libcSigaction(sig, nil, unsafe.Pointer(&sa[0])) != 0 {
			continue
		}
		handler := *(*uintptr)(unsafe.Pointer(&sa[0]))
		flags := (*int32)(unsafe.Pointer(&sa[sigactionFlags]))
		if handler > 1 && *flags&saOnstack == 0 {
			*flags |= saOnstack
			libcSigaction(sig, unsafe.Pointer(&sa[0]), nil)
		}
	}
}
