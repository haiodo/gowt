//go:build darwin

// Hand-written: an NSSegmentedControl inside an SWT composite (gowt.Segmented). One run-time ObjC
// class is the target of every control's action.
package cocoa

import "github.com/ebitengine/purego"

const (
	trackingSelectOne       = 0 // NSSegmentSwitchTrackingSelectOne
	trackingMomentary       = 2 // NSSegmentSwitchTrackingMomentary
	distributionFillEqually = 2 // NSSegmentDistributionFillEqually, macOS 10.13
)

// Segmented is a native segmented control; Clicked runs with the segment the user pressed.
type Segmented struct {
	View    uintptr
	Clicked func(i int)
}

var (
	segmentedViews  = map[uintptr]*Segmented{}
	segmentedTarget uintptr
	segmentedSize   func(self, sel uintptr) NSSize
)

// NewSegmented adds a control of n segments to parent, the segments as wide as the widest, in the
// select-one mode.
func NewSegmented(parent *NSView, n int) *Segmented {
	if segmentedTarget == 0 {
		click := purego.NewCallback(func(_, _, sender uintptr) {
			defer enterCallback()()
			if s := segmentedViews[sender]; s != nil && s.Clicked != nil {
				s.Clicked(int(int64(msg(sender, "selectedSegment"))))
			}
		})
		cls := newClass("GowtSegmentedTarget", nil, map[string]objcMethod{"gowtSegment:": {click, "v@:@"}})
		segmentedTarget = msg(msg(cls, "alloc"), "init")
		purego.RegisterFunc(&segmentedSize, objcMsgSend)
	}
	v := msg(msg(class("NSSegmentedControl"), "alloc"), "init")
	msg(v, "setSegmentCount:", uintptr(n))
	msg(v, "setTrackingMode:", trackingSelectOne)
	if respondsTo(v, "setSegmentDistribution:") {
		msg(v, "setSegmentDistribution:", distributionFillEqually)
	}
	msg(v, "setTarget:", segmentedTarget)
	msg(v, "setAction:", sel("gowtSegment:"))
	msg(uintptr(parent.Id), "addSubview:", v)
	msg(v, "release")
	s := &Segmented{View: v}
	segmentedViews[v] = s
	return s
}

// SetSegment gives segment i its label, image (nil for none) and tooltip. An image without a label
// is described to VoiceOver by the tooltip, on a copy so a shared image keeps its own description.
func (s *Segmented) SetSegment(i int, label string, image *NSImage, tooltip string) {
	msg(s.View, "setLabel:forSegment:", nsString(label), uintptr(i))
	if image != nil {
		img := uintptr(image.Id)
		if label == "" && tooltip != "" {
			img = msg(msg(img, "copy"), "autorelease")
			msg(img, "setAccessibilityDescription:", nsString(tooltip))
		}
		msg(s.View, "setImage:forSegment:", img, uintptr(i))
	}
	if tooltip != "" {
		msg(s.View, "setToolTip:forSegment:", nsString(tooltip), uintptr(i))
	}
}

// Selected returns the selected segment, -1 for none.
func (s *Segmented) Selected() int { return int(int64(msg(s.View, "selectedSegment"))) }

// Select selects segment i; -1 clears the selection.
func (s *Segmented) Select(i int) { msg(s.View, "setSelectedSegment:", uintptr(int64(i))) }

// SetMomentary makes the segments buttons that do not stay selected.
func (s *Segmented) SetMomentary(on bool) {
	mode := uintptr(trackingSelectOne)
	if on {
		mode = trackingMomentary
	}
	msg(s.View, "setTrackingMode:", mode)
}

// SetSmall gives the control the small control size and its font.
func (s *Segmented) SetSmall() { setSmall(s.View) }

// SetEnabled enables or dims the whole control.
func (s *Segmented) SetEnabled(on bool) { msg(s.View, "setEnabled:", boolArg(on)) }

// Fit returns the size the control asks for.
func (s *Segmented) Fit() NSSize { return segmentedSize(s.View, sel("fittingSize")) }

// SetFrame places the control in its parent's coordinates.
func (s *Segmented) SetFrame(r NSRect) { msgRectOnly(s.View, sel("setFrame:"), r) }

// Press presses segment i as VoiceOver does, through the accessibility element of the segment: a
// momentary control ignores setSelectedSegment:, so a selection followed by the action reports -1.
func (s *Segmented) Press(i int) {
	kids := msg(msg(s.View, "cell"), "accessibilityChildren")
	if kids == 0 || i < 0 || i >= int(msg(kids, "count")) {
		return
	}
	msg(msg(kids, "objectAtIndex:", uintptr(i)), "accessibilityPerformPress")
}

// SegmentRole is the accessibility role of segment i: AXRadioButton, or AXButton when momentary.
func (s *Segmented) SegmentRole(i int) string {
	kids := msg(msg(s.View, "cell"), "accessibilityChildren")
	if kids == 0 || i < 0 || i >= int(msg(kids, "count")) {
		return ""
	}
	return goString(msg(msg(kids, "objectAtIndex:", uintptr(i)), "accessibilityRole"))
}

// SegmentCount, SegmentLabel, TrackingMode and ControlSize read the control back for tests.
func (s *Segmented) SegmentCount() int { return int(msg(s.View, "segmentCount")) }

func (s *Segmented) SegmentLabel(i int) string {
	return goString(msg(s.View, "labelForSegment:", uintptr(i)))
}

func (s *Segmented) SegmentToolTip(i int) string {
	return goString(msg(s.View, "toolTipForSegment:", uintptr(i)))
}

func (s *Segmented) TrackingMode() int { return int(msg(s.View, "trackingMode")) }

func (s *Segmented) ControlSize() int { return int(msg(s.View, "controlSize")) }

// Frame is the control's frame in its parent.
func (s *Segmented) Frame() NSRect { return NewNSViewOverload1(int64(s.View)).Frame() }

// Release forgets the control; its parent view owns it.
func (s *Segmented) Release() { delete(segmentedViews, s.View) }
