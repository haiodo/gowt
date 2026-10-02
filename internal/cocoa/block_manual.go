//go:build darwin

// ObjC blocks over purego: a block literal is {isa, flags, reserved, invoke, descriptor}. Ours are
// global blocks (isa _NSConcreteGlobalBlock, BLOCK_IS_GLOBAL), so Block_copy/Block_release never
// copy or free them and the runtime only ever reads the header. invoke is a purego callback shared
// by all blocks with the same argument count; it finds the Go closure by the block address.
// Limits: word-sized arguments only (ids, pointers, ints, BOOL; no float or struct), void result,
// each block runs at most once. For calling a block someone else made, read invoke at offset 16.
package cocoa

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const blockIsGlobal = 1 << 28

type blockDescriptor struct{ reserved, size uintptr }

var blockDesc = blockDescriptor{0, 32}

// goBlock's first field is the literal, so &goBlock is the block pointer.
type goBlock struct {
	isa      uintptr
	flags    int32
	reserved int32
	invoke   uintptr
	desc     uintptr
	fn       func(args []uintptr)
}

var (
	blockMu      sync.Mutex
	blockLive    = map[uintptr]*goBlock{}
	blockFree    []*goBlock // never freed: a late Block_release may still read the header
	blockInvoke  = map[int]uintptr{}
	blockIsaOnce sync.Once
	blockIsa     uintptr
)

// NewBlock returns a block pointer to pass where an ObjC API wants a block with nargs word
// arguments; fn runs once when the callee invokes it, and the block is then recycled.
func NewBlock(nargs int, fn func(args []uintptr)) uintptr {
	ensureFrameworks()
	blockIsaOnce.Do(func() {
		blockIsa, _ = purego.Dlsym(purego.RTLD_DEFAULT, "_NSConcreteGlobalBlock")
	})
	blockMu.Lock()
	defer blockMu.Unlock()
	inv, ok := blockInvoke[nargs]
	if !ok {
		inv = purego.NewCallback(blockInvoker(nargs))
		blockInvoke[nargs] = inv
	}
	var b *goBlock
	if n := len(blockFree); n > 0 {
		b, blockFree = blockFree[n-1], blockFree[:n-1]
	} else {
		b = &goBlock{}
	}
	*b = goBlock{isa: blockIsa, flags: blockIsGlobal, invoke: inv, desc: uintptr(unsafe.Pointer(&blockDesc)), fn: fn}
	p := uintptr(unsafe.Pointer(b))
	blockLive[p] = b
	return p
}

func blockRun(self uintptr, args ...uintptr) {
	blockMu.Lock()
	b := blockLive[self]
	delete(blockLive, self)
	blockMu.Unlock()
	if b == nil {
		return
	}
	fn := b.fn
	b.fn = nil
	blockMu.Lock()
	blockFree = append(blockFree, b)
	blockMu.Unlock()
	fn(args)
}

func blockInvoker(n int) any {
	switch n {
	case 0:
		return func(self uintptr) { blockRun(self) }
	case 1:
		return func(self, a uintptr) { blockRun(self, a) }
	case 2:
		return func(self, a, b uintptr) { blockRun(self, a, b) }
	case 3:
		return func(self, a, b, c uintptr) { blockRun(self, a, b, c) }
	}
	panic("gowt/internal/cocoa: NewBlock supports 0..3 arguments")
}
