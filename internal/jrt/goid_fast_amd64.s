#include "textflag.h"

// func gword(off uintptr) uint64 returns the word at byte offset off of the running goroutine's g.
TEXT ·gword(SB),NOSPLIT,$0-16
	MOVQ (TLS), AX
	MOVQ off+0(FP), CX
	MOVQ (AX)(CX*1), AX
	MOVQ AX, ret+8(FP)
	RET
