package swt

import "testing"

func TestTextIconBGRA(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		pix := textIconBGRA(16, cancel, false)
		if len(pix) != 16*16*4 {
			t.Fatalf("cancel=%v: len %d", cancel, len(pix))
		}
		var opaque, partial, clear int
		for i := 3; i < len(pix); i += 4 {
			switch pix[i] {
			case 0:
				clear++
			case 255:
				opaque++
			default:
				partial++
			}
		}
		if opaque == 0 || partial == 0 || clear == 0 {
			t.Errorf("cancel=%v: opaque %d partial %d clear %d", cancel, opaque, partial, clear)
		}
	}
	if textIconBGRA(16, false, true)[0] <= textIconBGRA(16, false, false)[0] {
		t.Error("dark icon must be lighter than light icon")
	}
}
