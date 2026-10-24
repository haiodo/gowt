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
  Lift draws the sprites at once, on the worker goroutines, but keeps them hidden: the node stays
  in the tiles until they are redrawn without it, then the sprite and the clean tiles appear in one
  host transaction, so the node is on screen once in every frame (no ghost, no flash). The drag
  starts a frame or two later. A plain move only repositions a sprite; a changed picture (a
  connector) is drawn again in that frame, all changed sprites in parallel. `Scene` ignores a
  picture equal to the node's current one (`Picture.Equal`) and only the last picture of a node in
  a Commit counts, so connectors built relative to their start point (see `synth`) are not drawn
  again when both their boxes move by the same amount. `comp.Land(nodes...)` returns the nodes;
  sprites stay until the tiles are redrawn.
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

### Input (macOS)

The view reads the canvas mouse, key and focus events from SWT and the scroll, pinch and smart-zoom
events from the NSEvent itself (SWT hands scrolls on as integers without phase or momentum). All
callbacks run on the UI goroutine, one handler per kind; set them in `init`.

```go
v.OnPointer(func(p *view.Pointer) {
	switch p.Kind { // PointerDown, PointerMove, PointerUp, PointerCancel
	case view.PointerDown:
		// p.Pos view points, p.Scene scene point, p.Node topmost node under it (nil on empty space),
		// p.Button 1/2/3, p.Count 2 on a double click, p.Mods Ctrl/Shift/Alt/Cmd
	case view.PointerMove:
		// p.Drag: a button is held, p.Button which; p.Target and p.Origin: node and scene point of the press
	}
})
v.OnScroll(func(s *view.Scroll) bool { return false }) // true = handled, false = the view pans
v.OnMagnify(func(m *view.Magnify) bool { return false }) // m.Factor multiplies the zoom; false = ZoomAt the pointer
v.OnSmartZoom(func(z *view.SmartZoom) bool { return false }) // false = toggle zoom 1 and the zoom before
v.OnKey(func(k *view.Key) { /* k.Down, k.Code (SWT key code), k.Char, k.Mods */ })
v.SetCursor(view.CursorGrab) // Arrow Hand Grab Grabbing Crosshair Text ResizeH ResizeV NotAllowed
v.Pan(dx, dy); v.SetZoomLimits(0.02, 64)
```

- **Capture.** After a press the moves and the release keep coming outside the view (AppKit keeps the
  mouse-down view as the target) until the last button is up. The release is also read from the NSEvent,
  because SWT queues it and it can be late or lost outside the window; the SWT one that follows is dropped. If the view loses focus with a button held, a
  `PointerCancel` arrives: land what you lifted. Escape during a press is a `PointerCancel` with `Aborted` set: put back what the drag moved. While a button is held the view also polls the hardware button state every 20 ms and reports the release if no event brought it. A macOS Ctrl-click is Button 1 with `Mods.Ctrl`.
- **Outside.** `Pointer.Outside` is set when `Pos` is outside the view. `Pos` and `Scene` stay exact there
  (negative or past the view size): the release comes from the NSEvent converted with `convertPoint:` in
  float64, the moves from SWT as integer points. The view imposes no rule; the app clamps, cancels or drops
  where released.
- **Auto-scroll (opt-in).** `stop := v.AutoScroll(view.AutoScrollOptions{Band, Ramp, Max, Curve})`, called
  from the `PointerDown` that starts a drag. While the pointer is within `Band` view points (default 32) of
  an edge or outside the view, the camera pans on every frame of the frame clock, also with the pointer
  standing still. Speed per axis is `Max * Curve(min(d/Ramp, 1))` points per second, with `d` the distance
  past the band start (defaults: Max 1000, Ramp 120, linear), so the step per frame is even for a fixed
  distance and capped. After each step the view sends a `PointerMove` with `Drag` set at the same view
  point, so `Scene` is the new point under the pointer and an app that moves the dragged node to
  `Scene + grab` keeps it under the pointer. It ends on release, on `PointerCancel` (Escape, focus loss)
  or `stop()`; the app puts back what it moved on `Aborted` as before. An app can also skip it and call
  `v.Pan` from its own frame callback (`OnFrameReport`).
- **Example drop rule (the demos, not the view).** Figma and Freeform keep the dragged object in view: on a
  release with `Outside` set the demos clamp `Pos` into the view inset by 8 points (`clampToView`), take
  `SceneAt` of it and move the object there, so the grab point lands at the nearest visible point and the
  object never ends offscreen. Escape restores the origin instead.
- **Scroll.** `Precise` deltas (trackpad, Magic Mouse) are view points, a wheel gives lines (16 points each
  in the built-in pan). `Phase` is the finger phase, `Momentum` the phase of the coasting after the fingers
  lift; both are `PhaseNone` for a wheel. A press or resting fingers (`PhaseMayBegin`) drop the momentum
  events that follow, as a native scroll view stops. Without a handler Cmd or Ctrl with scroll zooms.
- **Scene point.** `Scene = camera origin + view point / zoom`, the same as `Compositor.SceneAt`; a node
  drag is `Lift` on press, `Scene.Update` with the new position on each move, `Land` on release or cancel.
- **Keys** need the view focused; a press on the view focuses it.
- **Screen change.** A window moved to a screen with another backing scale is noticed through the window
  notifications, and the next frame re-rasters at the new scale.
- **Smart zoom** arrives only if Trackpad > Smart zoom is on in System Settings, and AppKit sends it only to a
  view that implements `smartMagnifyWithEvent:`; the view adds that method to its class. There is no other
  two-finger double-click event: an app that wants a fallback uses `Pointer.Count == 2`.
- Scroll, pinch and smart-zoom events are consumed by the view, so a `MouseWheel` or `Gesture` listener on
  the canvas does not see them. Diagonal resize cursors are not offered (AppKit has no public ones).

## Measured (M4 Max, macOS 26, CoreGraphics, no window)

The headless run is `go test ./gfx/cg -run CompositorHeadless -v`. Windowed numbers come from
`go run -tags gfxdemo ./gfx/view/demo`, which opens a window.

- 10k boxes and 20k connectors: 1.1 KB per node (G-0: 2.2 KB), 512 px tile 0.7-2.0 ms, 24 tiles of
  a viewport on 8 workers 15 ms, first frame of the view about 30 ms, hit test 0.4 us.
- Frame CPU in Frame: pan 0.1 ms, zoom 0.2 ms, drag of 1 node with 10 links about 0.9 ms (was 2.1),
  drag of 100 nodes with their links p50 1.0 ms, p99 1.8 ms (was 5.4 and 7.1); the Lift call that
  draws about 150 sprites takes 4 ms (was a 13 ms frame).

## Not there yet

Rotation and swipe gestures, text input (IME) in the canvas, the Windows and Linux backends, shadows
blur on the overlay, group opacity for node subtrees, accessibility of the canvas.
