package fluent

import "testing"

func TestLayerAndCard(t *testing.T) {
	light, dark := Palette{Black: RGB{0, 0, 0}}, Palette{Dark: true, Black: RGB{255, 255, 255}}
	if got := light.Layer(); got != (RGB{0xF3, 0xF3, 0xF3}) {
		t.Errorf("light layer %v", got)
	}
	if got := dark.Layer(); got != (RGB{0x20, 0x20, 0x20}) {
		t.Errorf("dark layer %v", got)
	}
	// White at 70% over F3 is FB; at 5% over 20 is 2B.
	if got := light.Card(); got != (RGB{0xFB, 0xFB, 0xFB}) {
		t.Errorf("light card %v", got)
	}
	if got := dark.Card(); got != (RGB{0x2B, 0x2B, 0x2B}) {
		t.Errorf("dark card %v", got)
	}
}

func TestRowFill(t *testing.T) {
	p := Palette{Accent: RGB{0, 0x67, 0xC0}, Black: RGB{0, 0, 0}}
	bg := RGB{255, 255, 255}
	if got := rowFill(p, bg, true, false, true); got != p.Accent.over(0.20, bg) {
		t.Errorf("focused selection %v", got)
	}
	if got := rowFill(p, bg, true, true, true); got != p.Accent.over(0.28, bg) {
		t.Errorf("hot focused selection %v", got)
	}
	if got := rowFill(p, bg, true, false, false); got != p.Black.over(0.09, bg) {
		t.Errorf("unfocused selection %v", got)
	}
	if got := rowFill(p, bg, false, true, true); got != p.Black.over(0.05, bg) {
		t.Errorf("hover %v", got)
	}
}
