package gowt

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/haiodo/gowt/swt"
)

func TestFontResolve(t *testing.T) {
	cases := []struct {
		f    Font
		want Font
	}{
		{Font{}, Font{Name: "cur", Size: 12}},
		{Font{Scale: 1.5}, Font{Name: "cur", Size: 20}},  // 13 * 1.5 rounds up
		{Font{Scale: 0.01}, Font{Name: "cur", Size: 1}},  // never 0, which would mean "keep"
		{Font{Size: 9, Scale: 2}, Font{Name: "cur", Size: 9}},
		{Font{Name: "Mono", Bold: true}, Font{Name: "Mono", Size: 12, Bold: true}},
		{Font{Weight: Semibold, Bold: true}, Font{Name: "cur", Size: 12, Weight: Semibold}},
	}
	for _, c := range cases {
		if got := c.f.resolve("cur", 12, 13); got != c.want {
			t.Errorf("%+v.resolve = %+v, want %+v", c.f, got, c.want)
		}
	}
}

func TestAdaptiveIn(t *testing.T) {
	a := Adaptive{Light: RGB{1, 2, 3}, Dark: RGB{4, 5, 6}}
	var p Paint = a
	if p.in(false) != a.Light || p.in(true) != a.Dark {
		t.Errorf("Adaptive.in: %v %v", p.in(false), p.in(true))
	}
	if (RGB{7, 8, 9}).in(true) != (RGB{7, 8, 9}) {
		t.Error("RGB.in must ignore the theme")
	}
}

func bg(c *swt.Control) RGB { return rgbOf(c.GetBackground().GetRGB()) }
func fg(c *swt.Control) RGB { return rgbOf(c.GetForeground().GetRGB()) }

func TestAdaptiveRepaint(t *testing.T) {
	surface := Adaptive{Light: RGB{250, 250, 250}, Dark: RGB{30, 30, 30}}
	ink := Adaptive{Light: RGB{20, 20, 20}, Dark: RGB{230, 230, 230}}
	guiRun(t, func(app *App, w *Window) {
		app.repaint(false)
		box := put[*Panel](w, VBox(Background(surface)))
		l := put[*LabelView](box, Label("x", Foreground(ink)))
		plain := put[*ButtonView](w, Button("b", Background(RGB{1, 2, 3})))
		if bg(box.ctl) != surface.Light || fg(l.ctl) != ink.Light || bg(l.ctl) != surface.Light {
			t.Errorf("light: box %v, label %v on %v", bg(box.ctl), fg(l.ctl), bg(l.ctl))
		}
		if len(app.paints) != 2 {
			t.Errorf("%d controls kept for a theme switch, want the 2 with an Adaptive", len(app.paints))
		}
		app.repaint(true)
		if bg(box.ctl) != surface.Dark || fg(l.ctl) != ink.Dark || bg(l.ctl) != surface.Dark || bg(plain.ctl) != (RGB{1, 2, 3}) {
			t.Errorf("dark: box %v, label %v on %v, button %v", bg(box.ctl), fg(l.ctl), bg(l.ctl), bg(plain.ctl))
		}
		// A later RGB wins over the Adaptive on the next switch.
		l.SetForeground(RGB{9, 9, 9})
		app.repaint(false)
		if fg(l.ctl) != (RGB{9, 9, 9}) {
			t.Errorf("RGB after Adaptive: %v after a switch", fg(l.ctl))
		}
		box.SetBackground(nil)
		if box.c.GetBackgroundMode() != swt.INHERIT_NONE {
			t.Error("SetBackground(nil) left INHERIT_DEFAULT")
		}
		box.Dispose()
		if len(app.paints) != 0 {
			t.Errorf("%d controls kept after Dispose", len(app.paints))
		}
	})
}

func TestFontCacheSharing(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		sys := app.SystemFont()
		a := put[*LabelView](w, Label("a", Typeface(Font{Scale: 2, Weight: Semibold})))
		b := put[*LabelView](w, Label("b"))
		b.SetFont(Font{Scale: 2, Weight: Semibold})
		if a.l.GetFont() != b.l.GetFont() || len(app.fonts) != 1 {
			t.Fatalf("equal fonts not shared: %d native fonts", len(app.fonts))
		}
		shared := a.l.GetFont()
		if h := int(shared.GetFontData()[0].GetHeight()); h != 2*sys.Size {
			t.Errorf("Scale 2: height %d, system %d", h, sys.Size)
		}
		cv := put[*CanvasView](w, Canvas(nil))
		cv.fonts.get(Font{Scale: 2, Weight: Semibold}.from(cv.c.GetFont(), cv.c.GetDisplay()))
		if len(app.fonts) != 1 || app.fonts[ownedFonts[a.ctl]].refs != 3 {
			t.Errorf("canvas did not share the font: %d fonts", len(app.fonts))
		}
		a.Dispose()
		b.SetFont(Font{Size: 30})
		if shared.IsDisposed() {
			t.Error("font released while the canvas still uses it")
		}
		cv.Dispose()
		if !shared.IsDisposed() || len(app.fonts) != 1 {
			t.Errorf("after the last user: disposed %v, %d fonts left, want only b's", shared.IsDisposed(), len(app.fonts))
		}
		b.Dispose()
		if len(app.fonts) != 0 || len(ownedFonts) != 0 {
			t.Errorf("%d fonts, %d owned left", len(app.fonts), len(ownedFonts))
		}
	})
}

func TestImageOwnership(t *testing.T) {
	var kept *Image
	guiRun(t, func(app *App, w *Window) {
		var buf bytes.Buffer
		if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
			t.Fatal(err)
		}
		early, err := app.ImageFrom(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		kept, _ = app.ImageFrom(bytes.NewReader(buf.Bytes()))
		early.Dispose()
		early.Dispose()
		if !early.i.IsDisposed() || len(app.images) != 1 {
			t.Errorf("Dispose: disposed %v, %d images listed, want 1", early.i.IsDisposed(), len(app.images))
		}
	})
	if kept != nil && !kept.i.IsDisposed() {
		t.Error("the App did not dispose its image when Run returned")
	}
}
