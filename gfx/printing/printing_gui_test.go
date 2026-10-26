package printing_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
	"github.com/haiodo/gowt/gfx/printing"
	"github.com/haiodo/gowt/internal/pdfcheck"
)

var mainq = make(chan func())

// AppKit needs the main thread, which gowt's init pinned to this one.
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

// scene has a red box at the top left, a blue one at the bottom right and a text line.
func scene() *gfx.Scene {
	r := gfx.NewRecorder()
	r.FillRect(gfx.R(0, 0, 100, 50), gfx.RGB(220, 0, 0))
	r.FillRect(gfx.R(300, 250, 100, 50), gfx.RGB(0, 0, 220))
	r.Text("Printed scene", 120, 150, gfx.Font{Size: 24}, gfx.Black)
	s := gfx.NewScene()
	n := gfx.NewNode(1)
	s.Update(func(tx *gfx.Tx) {
		tx.Add(nil, n)
		tx.SetPicture(n, r.Finish())
	})
	return s
}

func TestPrintSceneToPDF(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	path := filepath.Join(t.TempDir(), "scene.pdf")
	var perr error
	var pages int
	var paper gowt.PrintArea
	var runErr error
	onMain(func() {
		runErr = gowt.Run(func(app *gowt.App) {
			job, err := gowt.PrintToPDF(path, gowt.PrintOptions{Title: "scene", Margin: 36})
			if err != nil {
				perr = err
				return
			}
			s := scene()
			perr = job.Print(2, func(p *gowt.PrintPage) {
				pages++
				paper = p.Paper
				if err := printing.Scene(p, s, nil, gfx.Rect{}, printing.Options{Background: gfx.RGBA(255, 255, 220, 1)}); err != nil {
					perr = err
				}
			})
		})
	})
	if perr != nil {
		t.Skip(perr) // no PDF printer here
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("GFX_PDF_OUT"); dir != "" {
		os.WriteFile(filepath.Join(dir, "printing_scene.pdf"), data, 0o644)
	}
	info, err := pdfcheck.Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 || len(info.Pages) != 2 {
		t.Fatalf("drew %d pages, the file has %d, want 2", pages, len(info.Pages))
	}
	if sz := info.Pages[0]; math.Abs(sz.W-paper.W) > 1.5 || math.Abs(sz.H-paper.H) > 1.5 {
		t.Errorf("the page is %vx%v in the file, the job said %vx%v", sz.W, sz.H, paper.W, paper.H)
	}
	if len(info.Fonts) == 0 && info.Images == 0 {
		t.Error("the file has neither text nor images")
	}
}

func TestPrintNothingToPrint(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	path := filepath.Join(t.TempDir(), "empty.pdf")
	var drawErr error
	var started bool
	onMain(func() {
		gowt.Run(func(app *gowt.App) {
			job, err := gowt.PrintToPDF(path, gowt.PrintOptions{})
			if err != nil {
				return
			}
			started = true
			job.Print(1, func(p *gowt.PrintPage) {
				drawErr = printing.Scene(p, gfx.NewScene(), nil, gfx.Rect{}, printing.Options{})
			})
		})
	})
	if started && drawErr == nil {
		t.Error("an empty scene printed without an error")
	}
}
