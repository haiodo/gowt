package look

import (
	"testing"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/swt"
)

// The pill fill is black on a light bar, white on a dark one: one alpha scheme serves both themes,
// the Harness #00000008..#0000001F light and #FFFFFF14/#FFFFFF1F dark.
func TestPillInk(t *testing.T) {
	light := swt.RGB{Red: 255, Green: 255, Blue: 255}
	dark := swt.RGB{Red: 30, Green: 31, Blue: 34}
	if got := pillInk(light); got != (swt.RGB{}) {
		t.Errorf("light bar: ink %v, want black", got)
	}
	if got := pillInk(dark); got != (swt.RGB{Red: 255, Green: 255, Blue: 255}) {
		t.Errorf("dark bar: ink %v, want white", got)
	}
}

func TestPillAlpha(t *testing.T) {
	for _, c := range []struct {
		state pillState
		alpha int32
	}{
		{pillRest, 0x0A},
		{pillHover, 0x14},
		{pillPressed, 0x1F},
	} {
		if got := pillAlpha(c.state); got != c.alpha {
			t.Errorf("state %d: alpha %#04x, want %#04x", c.state, got, c.alpha)
		}
	}
}

// The radius is the same for every button of one height: about 11 points, never more than a half
// circle, so a hover fill never changes the button size.
func TestPillRadius(t *testing.T) {
	if got, want := pillRadius(int32(gowt.Spacing.ControlHeight)), int32(gowt.Scaled(11)); got != want {
		t.Errorf("control height: radius %d, want %d", got, want)
	}
	if got := pillRadius(16); got != 8 {
		t.Errorf("height 16: radius %d, want the half-circle cap at 8", got)
	}
}

// Pressed beats hover, and a disabled segment stays at rest and never looks pressed.
func TestPillSegState(t *testing.T) {
	s := &pillSeg{enabled: true}
	if got := s.state(); got != pillRest {
		t.Errorf("idle: state %d, want rest", got)
	}
	s.hover = true
	if got := s.state(); got != pillHover {
		t.Errorf("hover: state %d, want hover", got)
	}
	s.pressed = true
	if got := s.state(); got != pillPressed {
		t.Errorf("pressed under the pointer: state %d, want pressed", got)
	}
	s.enabled = false
	if got := s.state(); got != pillRest {
		t.Errorf("disabled: state %d, want rest", got)
	}
}

func TestRGBOfHex(t *testing.T) {
	c := rgbOfHex(hexRGB(swt.RGB{Red: 0x1C, Green: 0x1C, Blue: 0x1E}))
	if *c != (swt.RGB{Red: 0x1C, Green: 0x1C, Blue: 0x1E}) {
		t.Errorf("rgbOfHex(hexRGB) = %v, want the colour back", *c)
	}
}
