//go:build windows && arm64

#include "textflag.h"

// Called by SyscallN with R0 = *thunkBlock; loads d0-d7 and x0-x7 and tail-calls the target.
TEXT ·gdipFloatThunk(SB),NOSPLIT|NOFRAME,$0-0
	MOVD	R0, R16
	FMOVD	72(R16), F0
	FMOVD	80(R16), F1
	FMOVD	88(R16), F2
	FMOVD	96(R16), F3
	FMOVD	104(R16), F4
	FMOVD	112(R16), F5
	FMOVD	120(R16), F6
	FMOVD	128(R16), F7
	MOVD	0(R16), R17
	MOVD	8(R16), R0
	MOVD	16(R16), R1
	MOVD	24(R16), R2
	MOVD	32(R16), R3
	MOVD	40(R16), R4
	MOVD	48(R16), R5
	MOVD	56(R16), R6
	MOVD	64(R16), R7
	B	(R17)

DATA ·gdipFloatThunkAddr+0(SB)/8, $·gdipFloatThunk(SB)
GLOBL ·gdipFloatThunkAddr(SB), NOPTR, $8
