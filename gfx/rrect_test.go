package gfx

import "testing"

func TestRRectRecognition(t *testing.T) {
	for _, tt := range []struct {
		name   string
		build  func() *Path
		wantOk bool
		want   [7]float64 // x, y, w, h, r0, r1(r2, r3 equal for these)
	}{
		{"rect", func() *Path { p := &Path{}; p.Rect(3, 4, 50, 20); return p }, true, [7]float64{3, 4, 50, 20, 0, 0, 0}},
		{"roundrect", func() *Path { p := &Path{}; p.RoundRect(3, 4, 50, 20, 6); return p }, true, [7]float64{3, 4, 50, 20, 6, 6, 6}},
		{"roundrect clamped", func() *Path { p := &Path{}; p.RoundRect(3, 4, 50, 20, 30); return p }, true, [7]float64{3, 4, 50, 20, 10, 10, 10}},
		{"roundrect4 distinct", func() *Path { p := &Path{}; p.RoundRect4(3, 4, 50, 20, [4]float64{2, 4, 6, 8}); return p }, true, [7]float64{3, 4, 50, 20, 2, 4, 6}},
		{"ellipse", func() *Path { p := &Path{}; p.Ellipse(10, 10, 8, 8); return p }, false, [7]float64{}},
		{"polygon", func() *Path {
			p := &Path{}
			p.MoveTo(0, 0)
			p.LineTo(10, 0)
			p.LineTo(10, 10)
			p.Close()
			return p
		}, false, [7]float64{}},
		{"shifted roundrect", func() *Path { p := &Path{}; p.RoundRect(-7.5, 100.25, 33.3, 17.7, 3.3); return p }, true, [7]float64{-7.5, 100.25, 33.3, 17.7, 3.3, 3.3, 3.3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			x, y, w, h, r, ok := tt.build().RRect()
			if ok != tt.wantOk {
				t.Fatalf("ok %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			got := [7]float64{x, y, w, h, r[0], r[1], r[2]}
			want := tt.want
			if tt.name == "roundrect4 distinct" {
				if r[3] != 8 {
					t.Errorf("r[3] %v, want 8", r[3])
				}
			} else if d := r[3] - r[0]; d > 1e-9 || d < -1e-9 {
				t.Errorf("r[3] %v, want %v", r[3], r[0])
			}
			for i := range got {
				if d := got[i] - want[i]; d > 1e-9 || d < -1e-9 {
					t.Errorf("field %d: got %v, want %v", i, got[i], want[i])
				}
			}
		})
	}
}
