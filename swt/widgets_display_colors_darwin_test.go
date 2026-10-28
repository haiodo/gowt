package swt

import "testing"

// SWT writes these as 236/255f: j2go once dropped the f, and Go folded
// the division to 0 (black tooltips).
func TestWidgetColorRGBFloatLiterals(t *testing.T) {
	d := &Display{}
	for _, c := range []struct {
		id  int32
		rgb [3]float64
	}{
		{COLOR_INFO_BACKGROUND, [3]float64{236, 235, 236}},
		{COLOR_WIDGET_NORMAL_SHADOW, [3]float64{159, 159, 159}},
		{COLOR_WIDGET_LIGHT_SHADOW, [3]float64{232, 232, 232}},
	} {
		got := d.GetWidgetColorRGB(c.id)
		for i, v := range c.rgb {
			if got[i] != v/255 {
				t.Errorf("color %d component %d = %v, want %v", c.id, i, got[i], v/255)
			}
		}
	}
}

func TestSystemColorInfoBackgroundNotBlack(t0 *testing.T) {
	onUI(t0, func(t *uiT) {
		c := guiDisplay(t).GetSystemColor(COLOR_INFO_BACKGROUND)
		if c.GetRed() == 0 && c.GetGreen() == 0 && c.GetBlue() == 0 {
			t.Fatalf("COLOR_INFO_BACKGROUND is black")
		}
	})
}
