package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/examples/apps/monitor/sampler"
	"github.com/haiodo/gowt/internal/shot"
)

var mainq = make(chan func())

func init() { runtime.LockOSThread() }

// GTK wants the UI on the main thread: the tests run on their own goroutines and hand the window over.
func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func onMain(f func()) {
	ok := make(chan struct{})
	mainq <- func() { defer close(ok); f() }
	<-ok
}

func inkIn(img image.Image, r image.Rectangle) int {
	bg := img.At(r.Min.X, r.Min.Y)
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.At(x, y) != bg {
				n++
			}
		}
	}
	return n
}

// After a few updates every card still shows its title: the labels that were never updated and the
// charts between two samples are what GTK blanked before App.repaintShell.
func TestCardsKeepTheirTitles(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	var snapErr error // not t.Fatal: this runs on the UI thread, where Goexit would hang the test
	var ink []int
	rows := -1
	demo = true // the table leaves out this process, the second one below
	defer func() { demo = false }()
	dir := t.TempDir()
	var err error
	onMain(func() {
		err = gowt.Run(func(app *gowt.App) {
			m := &monitor{app: app, host: sampler.Host{Name: "test", OS: "Linux", Cores: 4, Memory: 8 << 30}, sortCol: 1, sortDesc: true, first: true}
			m.build()
			t0 := time.Now()
			snapshot := func(i int) sampler.Snapshot {
				f := float64(i) / 10
				return sampler.Snapshot{
					Time: t0.Add(time.Duration(i) * time.Second), CPU: 0.2 + f, User: 0.15 + f, System: 0.05,
					Cores:  []float64{0.1, 0.2 + f, 0.3, 0.4},
					Mem:    sampler.Memory{Total: 8 << 30, Used: 3 << 30, App: 3 << 30, Cached: 2 << 30, Free: 3 << 30, SwapTotal: 1 << 30},
					RxRate: 1e5 * f, TxRate: 5e4, Uptime: time.Hour, Procs: []sampler.Process{{PID: 1, Name: "init", CPU: 1, RSS: 1 << 20}, {PID: os.Getpid(), Name: "monitor", CPU: 120, RSS: 1 << 20}},
				}
			}
			var step func(i int)
			step = func(i int) {
				app.After(700*time.Millisecond, func() {
					if i < 5 {
						m.apply(snapshot(i))
						step(i + 1)
						return
					}
					img := image.Image(image.NewNRGBA(image.Rect(0, 0, 1, 1)))
					path := filepath.Join(dir, "w.png")
					if snapErr = shot.WindowPNG(m.win.Unwrap().Handle, path); snapErr == nil {
						f, _ := os.Open(path)
						img, snapErr = png.Decode(f)
						f.Close()
					}
					client := m.win.Unwrap().GetClientArea()
					k := float64(img.Bounds().Dx()) / float64(client.Width)
					dy := img.Bounds().Dy() - int(float64(client.Height)*k+0.5)
					for _, c := range m.cards[:3] { // CPU, Memory, Network: a title at the top left of each
						sh, ctl := &m.win.Unwrap().Control, &c.Unwrap().Control
						p := ctl.GetDisplay().MapFromToXY(ctl, sh, 0, 0)
						at := func(v int32) int { return int(float64(v)*k + 0.5) }
						ink = append(ink, inkIn(img, image.Rect(at(p.X+8), dy+at(p.Y+6), at(p.X+90), dy+at(p.Y+30))))
					}
					rows = m.table.Len()
					m.win.Close()
				})
			}
			step(0)
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapErr != nil {
		t.Fatal(snapErr)
	}
	if rows != 1 {
		t.Errorf("under GOWT_DEMO the table has %d rows, want 1: its own process is not left out", rows)
	}
	for i, n := range ink {
		if n < 30 {
			t.Errorf("card %d: %d pixels in the title area, the title is blank", i, n)
		}
	}
}
