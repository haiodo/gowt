//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

var panelClass uintptr

// InstallPanelBackdrop is InstallBackdrop for a SetGlass panel: the glass is the lowest child of view
// itself, not a sibling below it. On macOS 26.5 a sibling glass drew a square with a shadow and ignored
// its corner radius; as a child it is rounded. A panel in a glass group keeps the sibling glass.
func InstallPanelBackdrop(view *NSView, kind int) bool {
	return installBackdrop(view, kind, false, true)
}

// panelGlassClass is NSGlassEffectView that lets every click through to the panel under it.
func panelGlassClass() uintptr {
	if panelClass == 0 {
		noHit := purego.NewCallback(func(self, _ uintptr, x, y float64) uintptr { return 0 })
		panelClass = allocClass(class("NSGlassEffectView"), "GowtGlassPanel", 0)
		addMethod(panelClass, sel("hitTest:"), noHit, "@@:{CGPoint=dd}")
		registerClass(panelClass)
	}
	return panelClass
}

// panelHost is the view that takes the children of v: the content view of an NSBox, v itself otherwise.
func panelHost(v uintptr) uintptr {
	if c := class("NSBox"); c != 0 && msg(v, "isKindOfClass:", c) != 0 {
		return msg(v, "contentView")
	}
	return v
}

func panelGlassInside(v, bd uintptr) bool {
	sv := msg(bd, "superview")
	return sv != 0 && sv == panelHost(v)
}

func placePanelGlass(v, bd uintptr) {
	host := panelHost(v)
	msgRectOnly(bd, sel("setFrame:"), NewNSViewOverload1(int64(host)).Bounds())
	msg(host, "addSubview:positioned:relativeTo:", bd, windowBelowArg, 0)
}

func syncPanelGlass(v, bd uintptr) {
	host := panelHost(v)
	msgRectOnly(bd, sel("setFrame:"), NewNSViewOverload1(int64(host)).Bounds())
	lowerPanelGlass(host, bd)
}

// lowerPanelGlass puts the glass back under the children: SWT adds a new child at the bottom of its
// parent, which is below the glass.
func lowerPanelGlass(host, bd uintptr) {
	subs := msg(host, "subviews")
	if msg(subs, "count") > 1 && msg(subs, "objectAtIndex:", 0) != bd {
		msg(bd, "retain")
		msg(bd, "removeFromSuperview")
		msg(host, "addSubview:positioned:relativeTo:", bd, windowBelowArg, 0)
		msg(bd, "release")
	}
}

// KeepPanelBackdropBelow lowers the glass InstallPanelBackdrop put in view under the children added since.
func KeepPanelBackdropBelow(view *NSView) {
	initAssociated()
	v := uintptr(view.Id)
	if bd := getAssociated(v, backdropKey()); bd != 0 && panelGlassInside(v, bd) {
		lowerPanelGlass(panelHost(v), bd)
	}
}

var msgToF64 func(self, sel uintptr) float64

// PanelGlassClip reads the corner radius and masksToBounds of the layer of the glass InstallPanelBackdrop
// put in view; ok is false without such glass.
func PanelGlassClip(view *NSView) (radius float64, masks, ok bool) {
	initAssociated()
	v := uintptr(view.Id)
	bd := getAssociated(v, backdropKey())
	if bd == 0 || !panelGlassInside(v, bd) || !respondsTo(msg(bd, "layer"), "cornerRadius") {
		return 0, false, false
	}
	if msgToF64 == nil {
		purego.RegisterFunc(&msgToF64, objcMsgSend)
	}
	l := msg(bd, "layer")
	return msgToF64(l, sel("cornerRadius")), msg(l, "masksToBounds")&0xff != 0, true
}
