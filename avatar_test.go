package gowt

import "testing"

func TestAvatarImages(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		red := RGB{200, 30, 30}
		av := app.AvatarImage("AB", red, 40)
		cr := app.CircleImage(av, 24)
		for name, img := range map[string]*Image{"AvatarImage": av, "CircleImage": cr} {
			want := 40
			if img == cr {
				want = 24
			}
			if w, h := img.Size(); w != want || h != want {
				t.Errorf("%s: size %dx%d, want %d", name, w, h, want)
			}
			d := img.Unwrap().GetImageDataZoom(100)
			n := d.Width
			if n < int32(want) {
				t.Fatalf("%s: %d pixels wide", name, n)
			}
			if a := d.GetAlpha(0, 0); a != 0 {
				t.Errorf("%s: corner alpha %d, want 0", name, a)
			}
			if a := d.GetAlpha(n/2, n/2); a != 255 {
				t.Errorf("%s: centre alpha %d, want 255", name, a)
			}
		}
		put[*LabelView](w, Label("")).SetImage(av)
	})
}
