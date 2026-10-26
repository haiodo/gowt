package gowt

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/haiodo/gowt/internal/pdfcheck"
	"github.com/haiodo/gowt/swt"
)

// printed is what a job run saw on the UI thread.
type printed struct {
	pages []PrintPage
	err   error
}

// printRun prints pages pages into a PDF file with draw and returns the pages as the job reported
// them and the file; it needs a display for the toolkit but opens no window.
func printRun(t *testing.T, o PrintOptions, pages int, draw func(*PrintPage)) (printed, []byte) {
	t.Helper()
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	path := filepath.Join(t.TempDir(), "out.pdf")
	var got printed
	var runErr error
	onMain(func() {
		runErr = Run(func(app *App) {
			job, err := PrintToPDF(path, o)
			if err != nil {
				got.err = err
				return
			}
			got.err = job.Print(pages, func(p *PrintPage) {
				cp := *p
				got.pages = append(got.pages, cp)
				draw(p)
			})
		})
	})
	if got.err == nil {
		got.err = runErr
	}
	data, _ := os.ReadFile(path)
	return got, data
}

func TestPrintToPDFPagesAndText(t *testing.T) {
	got, data := printRun(t, PrintOptions{Title: "gowt print test", Margin: 36}, 3, func(p *PrintPage) {
		gc := p.GC()
		gc.SetForeground(swt.NewColorRedGreenBlue(200, 0, 0))
		a := p.Area
		gc.DrawRectangle(int32(a.X), int32(a.Y), int32(a.W), int32(a.H)) // the area, in points
		gc.DrawText("Page text", int32(a.X+10), int32(a.Y+10))
	})
	if got.err != nil {
		t.Skip(got.err) // no PDF printer here (Windows without Microsoft Print to PDF)
	}
	if len(got.pages) != 3 {
		t.Fatalf("draw ran for %d pages, want 3", len(got.pages))
	}
	p := got.pages[0]
	if p.Number != 1 || got.pages[2].Number != 3 {
		t.Errorf("page numbers %d..%d", p.Number, got.pages[2].Number)
	}
	if p.Paper.W < 400 || p.Paper.H < 500 || p.Paper.W > p.Paper.H {
		t.Errorf("paper %+v is not a portrait sheet", p.Paper)
	}
	if !(p.Printable.W <= p.Paper.W && p.Printable.H <= p.Paper.H && p.Printable.W > p.Paper.W/2) {
		t.Errorf("printable %+v on paper %+v", p.Printable, p.Paper)
	}
	if math.Abs(p.Area.W-(p.Printable.W-72)) > 0.01 || p.Area.X != 36 {
		t.Errorf("area %+v, want the printable area less a 36 point margin", p.Area)
	}
	info, err := pdfcheck.Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Pages) != 3 {
		t.Fatalf("the file has %d pages, want 3", len(info.Pages))
	}
	if sz := info.Pages[0]; math.Abs(sz.W-p.Paper.W) > 1.5 || math.Abs(sz.H-p.Paper.H) > 1.5 {
		t.Errorf("the page is %vx%v in the file, the job said %vx%v", sz.W, sz.H, p.Paper.W, p.Paper.H)
	}
	if len(info.Fonts) == 0 {
		t.Error("the text is not text in the file")
	}
}

func TestPrintMarginTooLarge(t *testing.T) {
	got, _ := printRun(t, PrintOptions{Margin: 1000}, 1, func(*PrintPage) { t.Error("drew on a page with no area") })
	if got.err == nil {
		t.Error("a margin larger than the page was accepted")
	}
}
