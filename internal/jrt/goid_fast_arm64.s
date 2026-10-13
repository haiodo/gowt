#include "textflag.h"

// func gword(off uintptr) uint64 returns the word at byte offset off of the running goroutine's g.
TEXT ·gword(SB),NOSPLIT,$0-16
	MOVD off+0(FP), R0
	MOVD (g)(R0), R0
	MOVD R0, ret+8(FP)
	RET
