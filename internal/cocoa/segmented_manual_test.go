//go:build darwin

package cocoa

import "testing"

// A segmented control off screen: it sizes itself, the action reaches Clicked with the segment, the
// small size shrinks it, and it draws its segments into a bitmap.
func TestSegmentedOffscreen(t *testing.T) { onMain(func() { segmentedOffscreen(t) }) }

func segmentedOffscreen(t *testing.T) {
	pool := msg(msg(class("NSAutoreleasePool"), "alloc"), "init")
	defer msg(pool, "drain")
	parent := msg(msg(class("NSView"), "alloc"), "init")
	defer msg(parent, "release")
	s := NewSegmented(NewNSViewOverload1(int64(parent)), 3)
	defer s.Release()
	for i, l := range []string{"Fixed", "Hug", "Fill"} {
		s.SetSegment(i, l, nil, "tip "+l)
	}
	regular := s.Fit()
	if regular.Width < 60 || regular.Height < 15 {
		t.Fatalf("fitting size %+v", regular)
	}
	var got []int
	s.Clicked = func(i int) { got = append(got, i) }
	s.Press(2)
	if len(got) != 1 || got[0] != 2 || s.Selected() != 2 {
		t.Errorf("press 2: clicked %v, selected %d", got, s.Selected())
	}
	s.Select(-1)
	if s.Selected() != -1 {
		t.Errorf("Select(-1) left %d selected", s.Selected())
	}
	if r := s.SegmentRole(0); r != "AXRadioButton" {
		t.Errorf("select-one segment role %q", r)
	}
	s.SetMomentary(true)
	if s.TrackingMode() != trackingMomentary || s.SegmentToolTip(1) != "tip Hug" || s.SegmentRole(0) != "AXButton" {
		t.Errorf("tracking %d, tooltip %q, role %q", s.TrackingMode(), s.SegmentToolTip(1), s.SegmentRole(0))
	}
	got = nil
	s.Press(1)
	s.Press(1)
	if len(got) != 2 || got[0] != 1 || got[1] != 1 || s.Selected() != -1 {
		t.Errorf("momentary presses of 1: clicked %v, selected %d", got, s.Selected())
	}
	s.SetSmall()
	if small := s.Fit(); s.ControlSize() != controlSizeSmall || small.Height >= regular.Height {
		t.Errorf("small: size %d, height %v (regular %v)", s.ControlSize(), small.Height, regular.Height)
	}

	f := s.Fit()
	r := NSRect{Width: f.Width, Height: f.Height}
	s.SetFrame(r)
	v := NewNSViewOverload1(int64(s.View))
	rep := v.BitmapImageRepForCachingDisplayInRect(r)
	if rep == nil {
		t.Fatal("no bitmap for the control")
	}
	v.CacheDisplayInRect(r, rep)
	w, h := int(msg(uintptr(rep.Id), "pixelsWide")), int(msg(uintptr(rep.Id), "pixelsHigh"))
	ink := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if NewNSColorOverload1(int64(msg(uintptr(rep.Id), "colorAtX:y:", uintptr(x), uintptr(y)))).AlphaComponent() > 0.3 {
				ink++
			}
		}
	}
	if ink < w*h/20 {
		t.Errorf("the control drew %d of %d pixels", ink, w*h)
	}
}
