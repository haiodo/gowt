# Printing and PDF

Example: [`examples/print`](../examples/print).

Two things are here: the system print flow (`Window.PrintDialog`, `PrintJob`, `PrintPage`) with the package `gfx/printing` that draws gfx scenes on its pages, and PDF export of a gfx scene (`gfx.PDFRenderer`, `offscreen.PDF`). They share the page model: points (1/72 inch), y down, the origin at the top left.

## Print dialog and pages

```go
job, ok := w.PrintDialog(gowt.PrintOptions{Title: "Map", Margin: 36})
if !ok {
	return // cancelled
}
err := job.Print(pages, func(p *gowt.PrintPage) {
	printing.Scene(p, scene, nil, gfx.Rect{}, printing.Options{}) // or p.GC() for SWT drawing
})
```

- `PrintDialog` shows the system dialog (GTK print dialog, `NSPrintPanel` settings through SWT, the Windows common dialog). The printer, paper, orientation and page range the user chose are in the `PrintJob`; `job.PageRange()` tells the range.
- `Print(pages, draw)` starts the job and calls `draw` for each page of the range (clamped to `pages`), in order. A panic in `draw` cancels the job. A job prints once.
- A `PrintPage` has `Number`, `Paper` (the sheet), `Printable` (what the printer can reach, inside the sheet) and `Area`, the printable area less `PrintOptions.Margin`. All are in points; `Area` is in the coordinates of the page, whose origin is the top left corner of the printable area.
- `p.GC()` is an `*swt.GC` in points with that origin: text, lines, images, paths. `p.Native()` is the native context for `gfx/printing`. Use one of the two on a page.

### gfx scenes on a page

`printing.Scene(p, scene, root, area, opts)` and `printing.Picture(p, pic, opts)` fit the scene (or the area of it) into `p.Area`, centred, as large as it goes (`Options.Scale` caps it; `Options.Background` fills the scene area). `printing.Page(p)` is the `gfx.Page` itself, for several `Draw` calls with your own layout.

| OS | How the page is drawn |
|---|---|
| macOS | CoreGraphics and CoreText into the print operation's context: vectors and text, fonts embedded by the print system |
| Linux | cairo and Pango into the GTK print job's cairo context: vectors and text |
| Windows | Direct2D has no printer target, so the page is rendered at the printer's resolution in bands of 1024 rows and drawn as bitmaps: exact, not vector (a PDF made by the Microsoft Print to PDF driver holds images) |

A gfx page is drawn with the picture's own fonts. Shadows, gradients and group opacity are what the backend draws on the screen.

### Printing to a PDF file

`gowt.PrintToPDF(path, opts)` makes a job that writes a PDF file through the same print path, with no dialog: macOS saves the job as PDF, GTK uses its file backend, Windows prints to the "Microsoft Print to PDF" printer (the job returns an error without it). On macOS the print system needs one printer installed to start a job at all, even for a PDF: without one, `Print` returns an error. It is for tests and for "print to file" buttons; PDF export of a scene below is the way to write a PDF with no printer at all.

In a QA build `PrintDialog` is not shown: the scripted reply of kind `print` is the path of a PDF file the job writes ([qa.md](qa.md)).

## PDF export

```go
f, _ := os.Create("scene.pdf")
err := offscreen.PDF(f, scene, nil, gfx.PDFOptions{PageW: 595, PageH: 842, Margin: 36})
```

`gfx.PDFOptions` places the area of the scene on a page: with no `PageW` and `PageH` the page is the size of the area at `Scale` (default 1 point per scene unit) plus `Margin`; with both set the area is centred and scaled to the largest scale that fits (not past `Scale` when that is set). `Background` fills behind the content. `gfx.WriteSubtreePDF` and `gfx.WritePicturePDF` take a `gfx.PDFRenderer`; `offscreen.NewPDF(w)` gives a `gfx.Document` for several pages: `NewPage(w, h).Draw(scale, origin, items)`, then `Close`.

| OS | Backend | Text | Shadows |
|---|---|---|---|
| macOS | `CGPDFContext` (`gfx/cg`) | text, fonts embedded by CoreGraphics | CoreGraphics blur, rasterized in the file |
| Linux | cairo PDF surface (`gfx/cairo`) | text, Pango shapes it, cairo embeds subsets | rasterized |
| Windows | `gfx/pdf`, a pure Go writer | text, the glyphs used are embedded from the font file DirectWrite's choice points at; no shaping, kerning from the `kern` table only | stacked translucent strokes, not a Gaussian blur |

Why the Windows writer instead of the "Microsoft Print to PDF" printer: that driver is an optional Windows feature, shows a file dialog unless a path is given, and takes a bitmap from Direct2D (see above), so its PDF holds images. The pure Go writer keeps vectors and selectable text, runs on every OS (its output is checked on all of them), and costs shaping: complex scripts and ligatures are not shaped, a glyph the font lacks is drawn as its notdef, and a font that is not a TrueType file with embedding allowed is drawn as outlines. Alpha of a gradient stop is not used (the first stop's alpha stands for all when they agree).

Tests read the files back with `internal/pdfcheck` (pages, sizes, fonts, images) and, on macOS, render them with CoreGraphics and compare pixels with the bitmap renderer.
