package gowt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/haiodo/gowt/swt"
)

// PrintOptions are the settings of a print job that the system dialog does not ask for.
type PrintOptions struct {
	// Title is the name of the job in the printer queue; empty is "gowt".
	Title string
	// Margin is kept free on every side inside the area the printer can reach, in points.
	Margin float64
}

// PrintArea is a rectangle on a page in points (1/72 inch), y down.
type PrintArea struct{ X, Y, W, H float64 }

// PrintJob is a printer with its settings: chosen in the system dialog (Window.PrintDialog) or
// the file a job is written to (PrintToPDF). Print draws the pages.
type PrintJob struct {
	data  *swt.PrinterData
	opts  PrintOptions
	first int // first page of the range chosen in the dialog, 1-based
	last  int // last page of that range, 0 for the end of the document
}

// PrintDialog shows the system print dialog over w and returns the job it describes; ok is false
// when the user cancelled. In a QA build the dialog is not shown: the scripted reply of kind "print"
// is the path of a PDF file to print into (see docs/qa.md).
func (w *Window) PrintDialog(o PrintOptions) (job *PrintJob, ok bool) {
	if r, cancel, handled := scripted(dialogCall{Kind: "print", Title: o.Title}); handled {
		var path string
		if cancel || !replyAs("print", r, &path) {
			return nil, false
		}
		j, err := PrintToPDF(path, o)
		if err != nil {
			qaDialogFailed(err.Error())
			return nil, false
		}
		return j, true
	}
	dlg := swt.NewPrintDialog(w.shell)
	if o.Title != "" {
		dlg.SetText(o.Title)
	}
	data := dlg.Open()
	if data == nil {
		return nil, false
	}
	j := &PrintJob{data: data, opts: o, first: 1}
	if data.Scope == swt.PrinterDataPAGE_RANGE {
		j.first, j.last = int(data.StartPage), int(data.EndPage)
	}
	return j, true
}

// PrintToPDF makes a job that writes a PDF file instead of printing: macOS and Linux save the job
// as PDF, Windows prints to the "Microsoft Print to PDF" printer with the file name set, which
// needs that printer (a Windows feature that is on by default). The file is written by Print.
func PrintToPDF(path string, o PrintOptions) (*PrintJob, error) {
	if path == "" {
		return nil, errors.New("gowt: PrintToPDF needs a file name")
	}
	data := swt.NewPrinterData()
	for _, p := range swt.PrinterGetPrinterList() {
		if isPDFPrinter(p.Name) {
			data = p
			break
		}
	}
	if pdfPrinterRequired && !isPDFPrinter(data.Name) {
		return nil, errors.New(`gowt: no "Microsoft Print to PDF" printer is installed`)
	}
	data.PrintToFile = true
	data.FileName = path
	return &PrintJob{data: data, opts: o, first: 1}, nil
}

func isPDFPrinter(name string) bool { return strings.EqualFold(name, "Microsoft Print to PDF") }

// PageRange is the range of pages chosen in the dialog: first is 1-based and last is 0 for the end
// of the document. A job from PrintToPDF prints everything.
func (j *PrintJob) PageRange() (first, last int) { return j.first, j.last }

// PrintPage is one page being drawn. Draw through GC, or through a gfx page of the package
// gfx/printing, which takes Native. Coordinates are points from the top left corner of the area the
// printer can reach (Printable); the part inside the margin of PrintOptions is Area.
type PrintPage struct {
	Number    int       // 1-based
	Paper     PrintArea // the sheet; X and Y are 0
	Printable PrintArea // what the printer can reach, on the sheet
	Area      PrintArea // Printable less the margin, in the coordinates of the page
	gc        *swt.GC   // dots of the printer
	f         float64   // dots per point
	gcOn      bool      // GC set the point transform
	printer   *swt.Printer
}

// GC returns a graphics context whose unit is the point and whose origin is the top left corner
// of the printable area. Use it instead of Native on a page, not both.
func (p *PrintPage) GC() *swt.GC {
	if !p.gcOn {
		p.gcOn = true
		p.gc.SetAdvanced(true)
		t := swt.NewTransform(p.printer)
		t.Scale(float32(p.f), float32(p.f))
		p.gc.SetTransform(t)
		t.Dispose()
	}
	return p.gc
}

// NativeContext is the drawing context of a page as the platform has it, for a gfx page.
type NativeContext struct {
	// Handle is the cairo_t on Linux, the CGContextRef on macOS and the printer HDC on Windows. It is
	// valid until the draw callback returns.
	Handle uintptr
	// UnitsPerPoint is how many units of the context make a point: the printer's dots per 72nd of
	// an inch. The origin is the top left corner of the printable area, y down on Linux; macOS
	// may give y up, then Height flips it.
	UnitsPerPoint float64
	Width, Height float64 // size of the printable area in those units
	// GC is the same context as an SWT graphics context, without the point transform GC sets.
	GC *swt.GC
}

// Native returns the native context of the page, for package gfx/printing. A page is drawn through
// GC or through Native, not both.
func (p *PrintPage) Native() NativeContext {
	if p.gcOn {
		p.gcOn = false
		p.gc.SetTransform(nil)
	}
	return NativeContext{Handle: gcHandle(p.gc), UnitsPerPoint: p.f, Width: p.Printable.W * p.f, Height: p.Printable.H * p.f, GC: p.gc}
}

// Print runs the job: it asks the printer for pages first..last of a document of pages pages (the
// dialog's range, clamped to the document) and calls draw for each, in order. The job ends when
// draw has returned for the last page, or is cancelled when draw panics. The printer is released
// by Print; a job prints once.
func (j *PrintJob) Print(pages int, draw func(*PrintPage)) (err error) {
	if j.data == nil {
		return errors.New("gowt: the print job was already used")
	}
	data := j.data
	j.data = nil
	printer := swt.NewPrinterData_(data)
	defer printer.Dispose()
	title := j.opts.Title
	if title == "" {
		title = "gowt"
	}
	if !printer.StartJob(title) {
		return errors.New("gowt: the printer did not start the job")
	}
	finished := false
	defer func() {
		if !finished {
			printer.CancelJob()
		}
	}()
	first, last := j.first, j.last
	if first < 1 {
		first = 1
	}
	if last < 1 || last > pages {
		last = pages
	}
	dpi := printer.GetDPI()
	f := float64(dpi.X) / 72
	client := printer.GetClientArea()
	trim := printer.ComputeTrim(client.X, client.Y, client.Width, client.Height)
	pt := func(v int32) float64 { return float64(v) / f }
	paper := PrintArea{W: pt(trim.Width), H: pt(trim.Height)}
	printable := PrintArea{X: pt(client.X - trim.X), Y: pt(client.Y - trim.Y), W: pt(client.Width), H: pt(client.Height)}
	m := max(j.opts.Margin, 0)
	area := PrintArea{X: m, Y: m, W: printable.W - 2*m, H: printable.H - 2*m}
	if area.W <= 0 || area.H <= 0 {
		return fmt.Errorf("gowt: the margin of %v points leaves nothing of the %.0fx%.0f point printable area", m, printable.W, printable.H)
	}
	for n := first; n <= last; n++ {
		if !printer.StartPage() {
			return fmt.Errorf("gowt: the printer did not start page %d", n)
		}
		gc := swt.NewGCDrawable(printer)
		page := &PrintPage{Number: n, Paper: paper, Printable: printable, Area: area, gc: gc, f: f, printer: printer}
		func() {
			defer gc.Dispose()
			draw(page)
		}()
		printer.EndPage()
	}
	finished = true
	printer.EndJob()
	return nil
}
