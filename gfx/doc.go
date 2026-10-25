// Package gfx is an experimental retained 2D canvas for gowt: a tree of nodes holding recorded
// Pictures, a compositor that shows it from cached tiles, and the geometry under both.
//
// The package is pure Go and builds on every OS. Rasterizers and layer hosts live in sub-packages:
// gfx/cg (macOS), gfx/cairo (Linux), gfx/d2d (Windows), gfx/view (the gowt widget), gfx/offscreen
// (PNG without a window), gfx/layout (auto-layout) and gfx/pen (.pen designs). Most programs need
// only gfx, gfx/view or gfx/offscreen. The API may change without notice; docs/gfx.md has a quick
// start and the overview.
//
//   - Path, Measure: curves, exact bounds, flattening, length, point and tangent by length,
//     nearest point, split.
//   - Recorder and Picture: drawing commands in a compact byte stream, with bounds, gradients,
//     clips, layers with alpha, shadows, images and text measured by the backend.
//   - Scene and Node: transforms relative to the parent, opacity, clips, order among siblings, a
//     grid index, hit-testing that honors ancestor clips, and transactions (Scene.Update, applied
//     by Scene.Commit) callable from any goroutine.
//   - Compositor: static tiles drawn by a bounded worker pool, an active layer for dragged nodes,
//     an overlay of animated Sprites, all placed on whole device pixels on a Host.
//   - RenderSubtree, Picture.WriteSVG and WriteSubtreeSVG: output without a window.
package gfx
