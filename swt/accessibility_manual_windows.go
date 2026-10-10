package swt

import (
	"fmt"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/win32"
)

// SetNumberVARIANT is Accessible.setNumberVARIANT: the VARIANT type follows the Java box (Double, Float, Long, else int).
func (this *Accessible) SetNumberVARIANT(variant int64, number any) {
	switch n := number.(type) {
	case nil:
		win32.OSMoveMemoryOverload80(variant, []int16{win32.COMVT_EMPTY}, 2)
		win32.OSMoveMemoryOverload13(variant+8, []int32{0}, 4)
	case float64:
		win32.OSMoveMemoryOverload80(variant, []int16{win32.COMVT_R8}, 2)
		win32.OSMoveMemoryOverload77(variant+8, []float64{n}, 8)
	case float32:
		win32.OSMoveMemoryOverload80(variant, []int16{win32.COMVT_R4}, 2)
		win32.OSMoveMemoryOverload78(variant+8, []float32{n}, 4)
	case int64:
		win32.OSMoveMemoryOverload80(variant, []int16{win32.COMVT_I8}, 2)
		win32.OSMoveMemoryOverload79(variant+8, []int64{n}, 8)
	default:
		win32.OSMoveMemoryOverload80(variant, []int16{win32.COMVT_I4}, 2)
		win32.OSMoveMemoryOverload13(variant+8, []int32{int32(jrt.NumberDouble(number))}, 4)
	}
}

func (this *Accessible) String() string {
	role := this.GetRole()
	if role == 0 {
		role = this.GetDefaultRole()
	}
	return fmt.Sprintf("Accessible(%s)", this.GetRoleString(role))
}
