# Graphics and images

Example: [`examples/graphics`](../examples/graphics).

## Canvas and GC

`parent.Canvas(onPaint func(g *gowt.GC), opts...)` is an empty area drawn by your callback. The callback runs whenever the system needs a repaint; draw the whole area each time. After your model changes call `canvas.Redraw()`. The canvas is double buffered. `canvas.Size()` returns the drawable width and height.

```go
canvas := w.Canvas(func(g *gowt.GC) {
	g.SetFill(gowt.RGB{R: 30, G: 120, B: 220})
	g.FillOval(10, 10, 100, 60)
	g.SetColor(gowt.RGB{R: 40, G: 40, B: 40})
	g.Text("hello", 20, 20)
}, gowt.Cell(gowt.GridCell{Width: 400, Height: 200}))
```

The `*GC` is valid only inside the callback. There is nothing to dispose. Give the canvas a size with `Cell(GridCell{Width, Height})`, or it will be small.

| Method | Draws |
|---|---|
| `SetColor(RGB)` | colour of lines and text |
| `SetFill(RGB)` | colour of filled shapes and text background |
| `SetLineWidth(w)` | outline width in pixels |
| `Line`, `Rect`, `Oval` | outlines |
| `FillRect`, `FillOval` | filled shapes |
| `Text(s, x, y)`, `TextSize(s)` | text drawn over the fill colour; measure with `TextSize` |
| `Image(img, x, y)` | an image |

`RGB{R, G, B}` are `uint8`. Not wrapped: paths, gradients, transforms, clipping, drawing into an off-screen image. `Unwrap()` gives the `*swt.GC` with all of them ([swt-direct.md](swt-direct.md)).

## Images

| Call | Source |
|---|---|
| `app.LoadImage(path)` | a file: PNG, JPEG, GIF, BMP, ICO, TIFF |
| `app.ImageFrom(r io.Reader)` | any reader, same formats |
| `app.ImageFromProvider(p)` | a zoom-aware provider such as `svg.NewImageDataProvider` |

Both loaders return an `error`. Use the image with `Label.SetImage`, `Button.SetImage`, `ToolItem.SetImage`, `Node.SetImage`, `TableRow.SetImage(col, img)`, `TrayIcon.SetImage` or `GC.Image`.

Ownership: the App disposes every image it created when `Run` returns. Call `img.Dispose()` earlier if you create many short-lived images. Widgets neither copy nor dispose an image, so keep it alive while it is shown.

## SVG

Import `github.com/haiodo/gowt/svg` to rasterize SVG at every zoom (paths, basic shapes, gradients, strokes; no text, filters or masks):

```go
img := app.ImageFromProvider(svg.NewImageDataProvider(data, 24, 24, "#333333"))
```

`color` is what `currentColor` means in a monochrome icon. For a new theme colour build a new provider. `svg.Recolor(data, color)` returns recoloured bytes. Named system icons are in [icons.md](icons.md).
