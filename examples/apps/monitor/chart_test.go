package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/offscreen"
)

func testPalette(dark bool) *palette {
	card := gfx.Color{R: 0.93, G: 0.93, B: 0.94, A: 1}
	if dark {
		card = gfx.Color{R: 0.17, G: 0.17, B: 0.18, A: 1}
	}
	return newPalette(dark, card, gfx.RGB(10, 132, 255), 11)
}

func cpuSpec() chartSpec {
	return chartSpec{
		names: []string{"System", "User"}, stacked: true, total: "CPU", fixedMax: 1,
		fracs: []float64{0, 0.25, 0.5, 0.75, 1}, widest: "100%",
		color: func(p *palette, i int) gfx.Color { return []gfx.Color{p.second, p.accent}[i] },
		axis:  func(v float64) string { return pct(v) },
		value: func(v float64) string { return pct(v) },
	}
}

func netSpec() chartSpec {
	return chartSpec{
		names: []string{"Sent", "Received"}, minMax: 100e3, fracs: []float64{0, 0.5, 1}, widest: "000 KB/s",
		color: func(p *palette, i int) gfx.Color { return []gfx.Color{p.second, p.accent}[i] },
		axis:  rate, value: rate,
	}
}

// feed pushes n samples one second apart ending at the returned time.
func feed(c *chart, n int, f func(i int) []float64) {
	t := time.Date(2026, 10, 26, 14, 0, 0, 0, time.Local)
	for i := 0; i < n; i++ {
		c.push(t.Add(time.Duration(i)*time.Second), f(i)...)
	}
	c.tr = c.trTo
}

func cpuWave(i int) []float64 {
	u := 0.28 + 0.18*math.Sin(float64(i)/7) + 0.08*math.Sin(float64(i)/2.3)
	s := 0.07 + 0.04*math.Sin(float64(i)/5+1)
	return []float64{s, max(u, 0.02)}
}

// preview draws what the view shows: the tape under the camera, the axis, and the crosshair at
// hoverX (negative: none). The compositor itself needs a window, so the same pictures are placed
// by hand.
func preview(t *testing.T, c *chart, dark bool, hoverX float64, name string) {
	t.Helper()
	dir := os.Getenv("MONITOR_PNG_DIR")
	if dir == "" {
		return
	}
	r, err := offscreen.New(offscreen.Options{})
	if err != nil {
		t.Skip(err)
	}
	sc := gfx.NewScene()
	clip, tape, axis := gfx.NewNode(1), gfx.NewNode(2), gfx.NewNode(3)
	pps := c.pps()
	origin := c.tr*pps - (c.w - rightPad)
	sc.Update(func(tx *gfx.Tx) {
		tx.Add(nil, clip)
		tx.ClipRect(clip, gfx.R(0, 0, c.w, c.h))
		tx.Add(clip, tape)
		tx.SetPicture(tape, c.pic)
		tx.Move(tape, -origin, 0)
		tx.Add(clip, axis)
		tx.SetPicture(axis, c.axis)
		for i := range c.live {
			n := gfx.NewNode(10 + i)
			rec := gfx.NewRecorder()
			rec.FillEllipse(gfx.R(-4, -4, 8, 8), c.spec.color(c.pal, i))
			rec.StrokeEllipse(gfx.R(-4, -4, 8, 8), gfx.StrokeStyle{Width: 2}, c.pal.card)
			tx.Add(clip, n)
			tx.SetPicture(n, rec.Finish())
			tx.Move(n, c.w-rightPad, c.yOf(c.curves[i].at(c.tr)))
		}
		if hoverX >= 0 {
			c.hover = &gfx.Point{X: hoverX}
			k := len(c.ts) - 1 - int((c.w-rightPad-hoverX)/pps+0.5)
			x := c.w - rightPad - (c.tr-c.ts[k])*pps
			ln := gfx.NewNode(30)
			rec := gfx.NewRecorder()
			var p gfx.Path
			p.MoveTo(0, 0)
			p.LineTo(0, c.h)
			rec.StrokePath(&p, gfx.StrokeStyle{Width: 1}, c.pal.label)
			tx.Add(clip, ln)
			tx.SetPicture(ln, rec.Finish())
			tx.Move(ln, math.Round(x)+0.5, 0)
			b := gfx.NewNode(31)
			tx.Add(clip, b)
			tx.SetPicture(b, c.bubble(k))
			tx.Move(b, x+12, 6)
		}
	})
	bg := gfx.Color{R: c.pal.card.R, G: c.pal.card.G, B: c.pal.card.B, A: 1}
	ras, err := gfx.RenderSubtree(r, sc, clip, gfx.R(0, 0, c.w, c.h), 2, bg)
	if err != nil {
		t.Fatal(err)
	}
	png, _ := ras.PNG()
	if err := os.WriteFile(filepath.Join(dir, name+".png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewCharts(t *testing.T) {
	for _, dark := range []bool{false, true} {
		suffix := map[bool]string{false: "light", true: "dark"}[dark]
		c := newChart(cpuSpec(), testPalette(dark), time.Minute)
		c.w, c.h = 760, 200
		c.restyle()
		feed(c, 95, cpuWave)
		c.tr = c.trTo - 0.3 // mid-step
		c.rerecord()
		preview(t, c, dark, 500, "cpu-"+suffix)

		n := newChart(netSpec(), testPalette(dark), 5*time.Minute)
		n.w, n.h = 560, 180
		n.restyle()
		feed(n, 400, func(i int) []float64 {
			burst := 0.0
			if i > 250 && i < 300 {
				burst = 2.4e6 * math.Sin(float64(i-250)/50*math.Pi)
			}
			return []float64{30e3 + 20e3*math.Sin(float64(i)/9) + burst*0.2, 120e3 + 80e3*math.Sin(float64(i)/13) + burst}
		})
		n.yMax = n.yTarget
		n.rerecord()
		preview(t, n, dark, -1, "net-"+suffix)
	}
}

func TestStepEaseEnds(t *testing.T) {
	if stepEase(0) != 0 || stepEase(1) != 1 {
		t.Fatal("ease must run from 0 to 1")
	}
}

func TestSplineMonotone(t *testing.T) {
	xs := []float64{0, 1, 2, 3, 4}
	ys := []float64{0, 0, 1, 1, 0.2}
	s := newSpline(xs, ys)
	for x := 0.0; x <= 4; x += 0.05 {
		if v := s.at(x); v < -1e-9 || v > 1+1e-9 {
			t.Fatalf("overshoot at %v: %v", x, v)
		}
	}
	if s.at(2) != 1 || s.at(9) != 0.2 {
		t.Fatalf("spline must pass the samples")
	}
}

func benchChart() *chart {
	c := newChart(cpuSpec(), testPalette(false), time.Minute)
	c.w, c.h = 760, 200
	c.restyle()
	t := time.Date(2026, 10, 26, 14, 0, 0, 0, time.Local)
	for i := 0; i < 70; i++ {
		c.push(t.Add(time.Duration(i)*time.Second), cpuWave(i)...)
	}
	return c
}

// BenchmarkRecordTape is what one new sample costs on the UI thread: the tape picture again.
func BenchmarkRecordTape(b *testing.B) {
	c := benchChart()
	for i := 0; i < b.N; i++ {
		c.rerecord()
	}
}

// BenchmarkRasterTape is what the tile workers pay for the same sample: the visible area at 2x.
func BenchmarkRasterTape(b *testing.B) {
	r, err := offscreen.New(offscreen.Options{})
	if err != nil {
		b.Skip(err)
	}
	c := benchChart()
	c.scene.Commit()
	area := gfx.R(c.tr*c.pps()-(c.w-rightPad), 0, c.w, c.h)
	for i := 0; i < b.N; i++ {
		if _, err := gfx.RenderSubtree(r, c.scene, c.node, area, 2, gfx.Color{}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLabelEvery(t *testing.T) {
	for _, c := range []struct{ gap, w, want float64 }{{50, 60, 2}, {30, 60, 2}, {25, 60, 3}, {12, 60, 6}, {4, 60, 12}} {
		if got := labelEvery(c.gap, c.w); got != c.want {
			t.Errorf("labelEvery(%v, %v) = %v, want %v", c.gap, c.w, got, c.want)
		}
	}
}

// A 60 s window keeps at least two time labels on a 180 pt plot and three on a 226 pt one, inside
// the plot and apart, whatever second the window ends on.
func TestTimeLabelsOnNarrowPlots(t *testing.T) {
	// 44 pt is "15:04:05" in the macOS caption font, 50 pt leaves room for wider ones
	for _, c := range []struct{ plotW, labelW, want float64 }{{180, 50, 2}, {226, 44, 3}, {260, 50, 3}} {
		labelW := c.labelW
		pps := (c.plotW - rightPad) / 60
		every := labelEvery(10*pps, labelW+12)
		for end := 1000.0; end < 1060; end++ {
			hi := end*pps + rightPad
			lo := hi - c.plotW
			var ls []timeLabel
			for tk := math.Floor(lo/pps/10) * 10; tk*pps <= hi; tk += 10 {
				if x := tk * pps; x >= lo && math.Mod(tk/10, every) == 0 {
					ls = append(ls, timeLabel{"", x - labelW/2, labelW})
				}
			}
			got := placeLabels(ls, lo, hi)
			if float64(len(got)) < c.want {
				t.Errorf("%v pt plot ending at %vs: %d labels, want %v", c.plotW, end, len(got), c.want)
			}
			for i, l := range got {
				if l.x < lo || l.x+l.w > hi || i > 0 && l.x < got[i-1].x+got[i-1].w+4 {
					t.Errorf("%v pt plot ending at %vs: label %d at %.0f..%.0f, plot %.0f..%.0f", c.plotW, end, i, l.x, l.x+l.w, lo, hi)
				}
			}
		}
	}
}

// Whatever the system accent, the series of one chart never share a colour.
func TestSeriesColoursDistinct(t *testing.T) {
	accents := map[string]gfx.Color{
		"blue": gfx.RGB(0, 122, 255), "orange": gfx.RGB(255, 149, 0), "graphite": gfx.RGB(142, 142, 147),
		"red": gfx.RGB(255, 59, 48), "yellow": gfx.RGB(255, 204, 0), "green": gfx.RGB(52, 199, 89),
		"purple": gfx.RGB(175, 82, 222), "pink": gfx.RGB(255, 45, 85),
	}
	for name, a := range accents {
		for _, dark := range []bool{false, true} {
			p := newPalette(dark, gfx.RGB(240, 240, 240), a, 11)
			cs := []gfx.Color{p.accent, p.second, p.third}
			for i := range cs {
				for j := i + 1; j < len(cs); j++ {
					if near(cs[i], cs[j]) || cs[i] == cs[j] {
						t.Errorf("accent %s (dark %v): series %d and %d are both %v", name, dark, i, j, cs[i])
					}
				}
			}
		}
	}
	if p := newPalette(false, gfx.RGB(240, 240, 240), accents["orange"], 11); p.second != gfx.RGB(0, 122, 255) {
		t.Errorf("orange accent: second series %v, want blue", p.second)
	}
}

func TestNiceCeil(t *testing.T) {
	for in, want := range map[float64]float64{0: 1, 1: 1, 1.1: 2, 2.2: 2.5, 4.9: 5, 51: 100, 90e3: 100e3} {
		if got := niceCeil(in); got != want {
			t.Errorf("niceCeil(%v) = %v, want %v", in, got, want)
		}
	}
}

// Samples that arrive before the first size have no curves yet; the hover must wait for them.
func TestPlaceHoverBeforeFirstSize(t *testing.T) {
	spec := cpuSpec()
	spec.stacked = true
	c := newChart(spec, testPalette(false), time.Minute)
	feed(c, 5, cpuWave)
	if len(c.ts) == 0 || len(c.curves) != 0 {
		t.Fatalf("setup: want samples without curves, got %d samples, %d curves", len(c.ts), len(c.curves))
	}
	c.w, c.h = 760, 200 // the size is known, the tape is not recorded yet
	c.margin = 40
	c.hover = &gfx.Point{X: c.w - rightPad, Y: 50}
	c.placeHover()
	if c.hoverShown {
		t.Fatal("hover shown before the curves exist")
	}
}
