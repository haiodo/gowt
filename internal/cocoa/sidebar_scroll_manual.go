//go:build darwin

package cocoa

import "github.com/ebitengine/purego"

type sidebarScrollState struct {
	window, host, edge uintptr
	insets             NSRect
	automatic          uintptr
}

var sidebarScrolls = map[uintptr]sidebarScrollState{}
var sidebarScrollEdgeClass uintptr

// SetSidebarScrollUnderToolbar lets top-edge source lists scroll beneath the toolbar material.
// Call after laying out the sidebar; false restores the scroll views' original inset policy.
func SetSidebarScrollUnderToolbar(view *NSView, on bool) {
	initAssociated()
	var walk func(uintptr)
	walk = func(v uintptr) {
		if v == 0 {
			return
		}
		if msg(v, "isKindOfClass:", class("NSScrollView")) != 0 {
			s, exists := sidebarScrolls[v]
			nv := NewNSViewOverload1(int64(v))
			r := nv.ConvertRect_toView_(nv.Bounds(), nil)
			w := msg(v, "window")
			doc := msg(v, "documentView")
			top := w != 0 && r.Y+r.Height >= NewNSViewOverload1(int64(msg(w, "contentView"))).Frame().Height-1
			eligible := on && top && msg(doc, "isKindOfClass:", class("NSTableView")) != 0
			if !eligible {
				if exists {
					restoreSidebarScroll(v, s)
				}
				return
			}
			if !exists {
				s.window = w
				s.automatic = msg(v, "automaticallyAdjustsContentInsets")
				OSObjc_msgSend_stretOverload9(&s.insets, int64(v), int64(sel("contentInsets")))
				s.edge = msg(msg(sidebarScrollMaterialClass(), "alloc"), "init")
				msg(s.edge, "setMaterial:", 10) // NSVisualEffectMaterialHeaderView
				msg(s.edge, "setBlendingMode:", 1)
				msg(s.edge, "setState:", 0) // follows the window's active state
				msg(s.edge, "setWantsLayer:", 1)
				msg(msg(s.edge, "layer"), "setMask:", msg(class("CAGradientLayer"), "layer"))
				msg(v, "retain")
				msg(v, "addSubview:", s.edge)
				msg(s.edge, "release")
			}
			s.host = uintptr(view.Id)
			s.window = w
			sidebarScrolls[v] = s
			msg(v, "setAutomaticallyAdjustsContentInsets:", 1)
			syncSidebarScrollEdge(v, s.edge)
			return
		}
		subs := msg(v, "subviews")
		for i, n := uintptr(0), msg(subs, "count"); i < n; i++ {
			walk(msg(subs, "objectAtIndex:", i))
		}
	}
	root := uintptr(view.Id)
	walk(root)
	for v, state := range sidebarScrolls {
		if state.host == root && v != root && msg(v, "isDescendantOf:", root)&0xff == 0 {
			restoreSidebarScroll(v, state)
		}
	}
}

func sidebarScrollMaterialClass() uintptr {
	if sidebarScrollEdgeClass == 0 {
		sidebarScrollEdgeClass = allocClass(class("NSVisualEffectView"), "GowtSidebarScrollMaterial", 0)
		addMethod(sidebarScrollEdgeClass, sel("hitTest:"), purego.NewCallback(func(self, _ uintptr, x, y float64) uintptr { return 0 }), "@@:{CGPoint=dd}")
		addMethod(sidebarScrollEdgeClass, sel("isAccessibilityElement"), purego.NewCallback(func(self, _ uintptr) uintptr { return 0 }), "B@:")
		addMethod(sidebarScrollEdgeClass, sel("accessibilityIsIgnored"), purego.NewCallback(func(self, _ uintptr) uintptr { return 1 }), "B@:")
		registerClass(sidebarScrollEdgeClass)
	}
	return sidebarScrollEdgeClass
}

func syncSidebarScrollEdge(v, edge uintptr) {
	transaction := class("CATransaction")
	msg(transaction, "begin")
	// Implicit mask animations trail behind the divider during a live drag.
	msg(transaction, "setDisableActions:", 1)
	defer msg(transaction, "commit")
	b := NewNSViewOverload1(int64(v)).Bounds()
	top := float64(ContentTopInset(NewNSViewOverload1(int64(v))))
	r := b
	r.Height = top + 16
	mask := uintptr(2 | 8) // width sizable, fixed to top in unflipped coordinates
	if msg(v, "isFlipped") == 0 {
		r.Y = b.Y + b.Height - r.Height
	} else {
		mask = 2 | 32
	}
	msg(edge, "setAutoresizingMask:", mask)
	msgRectOnly(edge, sel("setFrame:"), r)
	gradient := msg(msg(edge, "layer"), "mask")
	msgRectOnly(gradient, sel("setFrame:"), NSRect{Width: r.Width, Height: r.Height})
	colors := msg(class("NSMutableArray"), "array")
	for _, color := range []string{"whiteColor", "whiteColor", "clearColor"} {
		msg(colors, "addObject:", msg(msg(class("NSColor"), color), "CGColor"))
	}
	locations := msg(class("NSMutableArray"), "array")
	for _, f := range []float64{0, top / r.Height, 1} {
		msg(locations, "addObject:", uintptr(NSNumberNumberWithDouble(f).Id))
	}
	msg(gradient, "setColors:", colors)
	msg(gradient, "setLocations:", locations)
	OSObjc_msgSendOverload2(int64(gradient), int64(sel("setStartPoint:")), NSPoint{X: .5, Y: 1})
	OSObjc_msgSendOverload2(int64(gradient), int64(sel("setEndPoint:")), NSPoint{X: .5})
}

func restoreSidebarScroll(v uintptr, s sidebarScrollState) {
	msg(s.edge, "removeFromSuperview")
	msg(v, "setAutomaticallyAdjustsContentInsets:", s.automatic)
	msgRectOnly(v, sel("setContentInsets:"), s.insets)
	delete(sidebarScrolls, v)
	msg(v, "release")
}

// trafficLightShifts holds, per window, the left x of each standard window button as AppKit laid it
// out and the shift applied on top; SetTrafficLightInset is idempotent and reversible.
var trafficLightShifts = map[uintptr]*trafficLightShift{}

type trafficLightShift struct {
	base [3]float64
}

// SetTrafficLightInset moves the standard window buttons (traffic lights) dx points right of AppKit's
// layout, restoring AppKit's positions at 0. The bases are captured once, on a visible window: AppKit
// does not promise the button frames before the window is shown. Only a full-height sidebar calls
// this; the floating sidebar mode keeps AppKit's layout.
func SetTrafficLightInset(view *NSView, dx float64) {
	w := msg(uintptr(view.Id), "window")
	if w == 0 {
		return
	}
	t := trafficLightShifts[w]
	if t == nil {
		if dx == 0 {
			return
		}
		if msg(w, "isVisible")&0xff == 0 {
			return
		}
		t = &trafficLightShift{}
		for i := range t.base {
			t.base[i] = -1
		}
		trafficLightShifts[w] = t
	}
	for i := uintptr(0); i < 3; i++ {
		b := msg(w, "standardWindowButton:", i)
		if b == 0 {
			continue
		}
		v := NewNSViewOverload1(int64(b))
		f := v.Frame()
		if t.base[i] < 0 {
			t.base[i] = f.X
		}
		if nx := t.base[i] + dx; nx != f.X {
			OSObjc_msgSendOverload2(int64(b), int64(sel("setFrameOrigin:")), NSPoint{X: nx, Y: f.Y})
		}
	}
}
