//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

// GlassCornerRadius reads back the radius of the glass InstallBackdrop added for view, -1 without one.
func GlassCornerRadius(view *NSView) float64 {
	initAssociated()
	bd := getAssociated(uintptr(view.Id), backdropKey())
	if !respondsTo(bd, "cornerRadius") {
		return -1
	}
	var get func(self, sel uintptr) float64
	purego.RegisterFunc(&get, objcMsgSend)
	return get(bd, sel("cornerRadius"))
}
