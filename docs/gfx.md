# gfx: retained 2D canvas (experimental)

`gowt/gfx` is a scene of nodes drawn from cached tiles, for editors with many shapes: diagrams, UI
mockups. It is **experimental**: the API may change without notice, and only macOS has a backend so
far (CoreGraphics, CoreText, CALayer). The scene, pictures, geometry, SVG and the compositor logic are
pure Go and build on every OS; Windows and Linux backends plug in behind `gfx.Renderer` and `gfx.Host`.

Packages:

| Package | What |
|---|---|
| `gfx` | paths and curve geometry, `Picture`, `Scene`, `Compositor`, animations, SVG, offscreen rendering |
| `gfx/cg` | macOS: `Renderer` (CoreGraphics + CoreText), `Host` (CALayer), `DisplayLink` (CVDisplayLink) |
| `gfx/view` | the gowt widget: `view.New(scene, init, opts...)` |
| `gfx/synth` | a synthetic 10k-box document for benchmarks |

## Scene

```go
scene := gfx.NewScene(gfx.R(-1000, -1000, 20000, 20000), 256) // world bounds, grid cell

n := gfx.NewNode(1)
scene.Update(func(tx *gfx.Tx) { // any goroutine; applied whole by the next frame
	tx.Add(nil, n)             // nil: under the root
	tx.SetPicture(n, pic)      // drawn in the node's own coordinates
	tx.SetShape(n, shape, gfx.HitFill)
	tx.Move(n, 100, 40)        // or SetTransform(n, affine)
	tx.SetOpacity(n, 0.8)
	tx.ClipRRect(frame, gfx.R(0, 0, 300, 200), 8) // clips frame and its subtree
})
```

- A node has a transform relative to its parent, opacity, an optional clip (rectangle, rounded
  rectangle or path), a Z among siblings and a `Picture`. Children draw after their parent.
  Moving a frame moves its subtree without touching the children's pictures.
- `Scene.Commit` (the compositor calls it every frame) applies the queued transactions and returns
  the scene rectangles whose pixels changed: the old and new bounds of the nodes that changed. Only
  tiles under them are redrawn.
- `Scene.HitTest(point, tolerance)` goes from the top: bounds, then the exact fill or stroke of
  the node's shape, then every clip of its ancestors. The tolerance is in scene units (pixels / zoom).
- Opacity of a node multiplies down the tree and is applied per node, not to a group as a whole:
  overlapping children of a half-transparent frame show through each other. Use `BeginLayer` in a
  picture for a true group alpha.

## Pictures

`gfx.NewRecorder()` records `Save`, `Restore`, `Translate`, `Scale`, `Rotate`, `Transform`,
`ClipRect`, `ClipRRect`, `ClipPath`, `FillPath`, `StrokePath` (dash, caps, joins), `FillGradient` and
`StrokeGradient` (linear, radial), `FillShadowRRect`, `Text`, `DrawImage`, `BeginLayer`/`EndLayer`
(group alpha). The result is an immutable `*Picture` in a compact byte stream (about 0.3 KB for
a box with shadow, border and two labels), with bounds that follow transforms, clips and strokes.
Text bounds are measured by CoreText when `cg.NewRenderer` ran before recording
(`gfx.DefaultMeasurer`), else estimated. `Picture.Lod` replaces a picture when it is small on screen.

Output without a window:

```go
r := cg.NewRenderer(cg.Options{TextCache: true, ShadowSprite: true, LOD: true})
raster, err := gfx.RenderSubtree(r, scene, node, gfx.Rect{}, 2, gfx.Color{R: 1, G: 1, B: 1, A: 1})
png, _ := raster.PNG()
gfx.WriteSubtreeSVG(w, scene, node, gfx.Rect{}, 1) // pure Go, no renderer
```

## Compositor

```go
comp := gfx.NewCompositor(scene, renderer, host, gfx.Options{})
comp.SetViewport(wPoints, hPoints, devicePixelsPerPoint)
comp.SetCamera(gfx.Camera{Origin: gfx.Point{X: 0, Y: 0}, Zoom: 1})
comp.Frame(frameInfo) // on the UI goroutine, once per display frame
```

- **Tiles**: 512 device pixel tiles, drawn by a worker pool (default NumCPU-2, at most 6, so the UI
  thread keeps a core), visible tiles first, one ring of margin. Zoom levels are powers of two
  of device pixels per scene unit. During a zoom gesture the existing tiles are scaled; 120 ms after
  the last zoom change the tiles are drawn again at the new level, and the old ones leave when the
  new ones cover the view. `Options.ExactZoom` draws at exactly the zoom instead.
- **Pan** moves tile edges by whole device pixels; neighbours meet exactly.
- **Active layer**: `comp.Lift(nodes...)` takes nodes (with subtrees) out of the tiles into
  sprites that follow their nodes. Pass the dragged nodes and the connectors that depend on them.
  A sprite is drawn once; a plain move only repositions it, a picture change (a connector) redraws
  it on that frame. `comp.Land(nodes...)` returns them; sprites stay until the tiles are redrawn.
  Until the tiles under a lifted node are redrawn, the old pixels show as a ghost for a frame or two.
- **Overlay**: `comp.NewSprite(pic, space, pos)` for selection, handles, remote cursors. Spaces:
  `SpaceScene` (zooms with the scene), `SpacePinned` (anchored to a scene point, screen size),
  `SpaceScreen`.
- **Animation** runs on the frame clock (`FrameInfo.Time`, the display time of the frame): `MoveTo`,
  `FadeTo`, `RotateTo`, `ScaleTo` with `Linear`, `EaseIn`, `EaseOut`, `EaseInOut` or `CubicBezier`,
  and `FlyAlong(path, duration, ease, orient, done)` for a cursor along a cubic path. Set a goal
  once from any goroutine; no code runs per frame in the application.
- `comp.OnFrame(f)` runs first in every frame with the frame clock, for model animations. While f is set the view
  gets a frame on every display tick; `OnFrame(nil)` lets an idle scene stop.
  `comp.NeedsFrame()` is cheap and goroutine-safe; the macOS widget polls it from the display link.

## The widget (macOS)

```go
var doc *synth.Doc // or your own scene
app.Window("title", view.New(scene, func(v *view.View) {
	v.Compositor().SetCamera(gfx.Camera{Zoom: 1})
	v.Canvas().OnMouseDown(func(m g.Mouse) { n := v.HitTest(m.X, m.Y) /* ... */ })
}, g.Grow()))
```

`view.New` builds a `g.Canvas` made layer-backed, a `cg.Host`, a compositor and a CVDisplayLink;
frames are posted to the UI thread only while `NeedsFrame` is true. Elsewhere `view.New` gives a
plain canvas and never calls `init`.

## Measured (M4 Max, macOS 26, CoreGraphics, no window)

The headless run is `go test ./gfx/cg -run CompositorHeadless -v`. Windowed numbers come from
`go run -tags gfxdemo ./gfx/view/demo`, which opens a window.

- 10k boxes and 20k connectors: 1.1 KB per node (G-0: 2.2 KB), 512 px tile 0.7-2.0 ms, 24 tiles of
  a viewport on 8 workers 15 ms, first frame of the view about 30 ms, hit test 0.4 us.
- Frame CPU in Frame: pan 0.1 ms, zoom 0.2 ms, drag of 1 node with 14 links about 2 ms,
  drag of 100 nodes with their links about 5 ms (max 20 ms on the frame that lifts them).

## Not there yet

Input beyond the canvas mouse listeners (wheel, pinch, key), the Windows and Linux backends, shadows
blur on the overlay, group opacity for node subtrees, accessibility of the canvas.
