package gowt

import "github.com/haiodo/gowt/swt"

// tabClose is the box of the close button, the default renderer's size: the 16 px button and its trim.
const tabClose = 18

// tabRenderer draws a CTabFolder as a modern document tab bar: a flat strip with a hairline, the
// selected tab merging with the content below, a close button on the selected and the hovered tab.
// It subclasses the SWT renderer as SWT intends (CTabFolder.setRenderer); an override reaches only
// calls that CTabFolder makes, not the renderer's own nested calls (docs/internals.md).
type tabRenderer struct {
	*swt.CTabFolderRenderer
	f      *swt.CTabFolder
	colors map[RGB]*swt.Color
}

// newTabRenderer installs the renderer on f and sets the tab height from the density tokens.
func newTabRenderer(f *swt.CTabFolder) *tabRenderer {
	r := &tabRenderer{f: f, colors: map[RGB]*swt.Color{}}
	r.CTabFolderRenderer = swt.NewCTabFolderRenderer(f)
	r.SetImpl_(r)
	f.SetRenderer(r)
	f.SetTabHeight(int32(Spacing.ControlHeight - 1))
	return r
}

func mixRGB(a, b RGB, t float64) RGB {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return RGB{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B)}
}

type tabPalette struct{ base, strip, hover, fg, dim, line, accent RGB }

func (r *tabRenderer) palette() tabPalette {
	dark := systemDark()
	if a := appOf(r.f); a != nil {
		dark = a.dark
	}
	d := r.f.GetDisplay()
	p := tabPalette{base: rgbOf(r.f.GetBackground().GetRGB()), fg: rgbOf(r.f.GetForeground().GetRGB())}
	shade := 0.05
	if dark {
		shade = 0.25
	}
	p.strip = mixRGB(p.base, RGB{}, shade)
	p.hover = mixRGB(p.strip, p.fg, 0.08)
	p.dim = secondaryText()
	p.line = tabSeparator(d, p.strip, p.fg, dark)
	p.accent = tabAccent(d, dark)
	return p
}

func (r *tabRenderer) c(v RGB) *swt.Color {
	col := r.colors[v]
	if col == nil {
		col = v.color()
		r.colors[v] = col
	}
	return col
}

func (r *tabRenderer) fill(gc *swt.GC, v RGB, x, y, w, h int32) {
	gc.SetBackground(r.c(v))
	gc.FillRectangle(x, y, w, h)
}

func (r *tabRenderer) bottom() bool { return r.f.GetTabPosition() == swt.BOTTOM }

func (r *tabRenderer) Dispose_() {
	for _, c := range r.colors {
		c.Dispose()
	}
	r.colors = map[RGB]*swt.Color{}
	r.CTabFolderRenderer.Dispose()
}

// ComputeTrim_ gives a tab the horizontal padding of the Spacing tokens and the body no highlight
// margins; the other parts keep SWT's.
func (r *tabRenderer) ComputeTrim_(part, state, x, y, w, h int32) *swt.Rectangle {
	switch {
	case part >= 0 && part < r.f.GetItemCount():
		pad := int32(Spacing.MD)
		return swt.NewRectangle(x-pad, y, w+2*pad, h)
	case part == swt.CTabFolderRendererPART_BODY:
		var border int32
		if r.f.GetBorderVisible() {
			border = 1
		}
		mw, mh := r.f.MarginWidth, r.f.MarginHeight
		th := r.f.GetTabHeight()
		if r.f.GetMinimized() {
			h = 0
		}
		y0 := y - mh - th
		if r.bottom() {
			y0 = y - mh - border
		}
		return swt.NewRectangle(x-mw-border, y0, w+2*border+2*mw, h+2*border+2*mh+th)
	}
	return r.CTabFolderRenderer.ComputeTrim(part, state, x, y, w, h)
}

// ComputeSize_ makes the chevron a plain glyph button.
func (r *tabRenderer) ComputeSize_(part, state int32, gc *swt.GC, wHint, hHint int32) *swt.Point {
	if part == swt.CTabFolderRendererPART_CHEVRON_BUTTON {
		return swt.NewPoint(tabClose+6, tabClose)
	}
	return r.CTabFolderRenderer.ComputeSize(part, state, gc, wHint, hHint)
}

func (r *tabRenderer) Draw_(part, state int32, b *swt.Rectangle, gc *swt.GC) {
	switch {
	case part == swt.CTabFolderRendererPART_BACKGROUND:
		r.fill(gc, r.palette().strip, b.X, b.Y, b.Width, b.Height)
	case part == swt.CTabFolderRendererPART_BODY:
		r.drawBody(b, gc)
	case part == swt.CTabFolderRendererPART_HEADER:
		r.drawStrip(b, gc)
	case part == swt.CTabFolderRendererPART_CHEVRON_BUTTON:
		r.drawChevron(state, b, gc)
	case part >= 0 && part < r.f.GetItemCount():
		if b.Width > 0 && b.Height > 0 {
			r.drawTab(part, state, b, gc)
		}
	default:
		r.CTabFolderRenderer.Draw(part, state, b, gc)
	}
}

func (r *tabRenderer) drawBody(b *swt.Rectangle, gc *swt.GC) {
	p := r.palette()
	r.fill(gc, p.base, b.X, b.Y, b.Width, b.Height)
	if r.f.GetBorderVisible() {
		gc.SetForeground(r.c(p.line))
		gc.DrawRectangle(b.X, b.Y, b.Width-1, b.Height-1)
	}
}

func (r *tabRenderer) drawStrip(b *swt.Rectangle, gc *swt.GC) {
	p := r.palette()
	th := r.f.GetTabHeight()
	y, line := int32(0), th-1
	if r.bottom() {
		y = b.Height - th
		line = 0
	}
	r.fill(gc, p.strip, 0, y, b.Width, th)
	r.fill(gc, p.line, 0, y+line, b.Width, 1)
}

func (r *tabRenderer) drawTab(i, state int32, b *swt.Rectangle, gc *swt.GC) {
	p := r.palette()
	item := r.f.GetItem(i)
	sel, hot := state&swt.SELECTED != 0, state&swt.HOT != 0
	// The strip's hairline is the last row of a top tab and the first of a bottom one; the selected
	// tab covers it so that it joins the content.
	bodyY, bodyH := b.Y, b.Height-1
	if r.bottom() {
		bodyY++
	}
	switch {
	case sel:
		r.fill(gc, p.base, b.X, b.Y, b.Width, b.Height)
		iy := b.Y
		if r.bottom() {
			iy = b.Y + b.Height - 2
		}
		r.fill(gc, p.accent, b.X, iy, b.Width, 2)
	case hot:
		r.fill(gc, p.hover, b.X, bodyY, b.Width, bodyH)
	}
	x, right := b.X+int32(Spacing.MD), b.X+b.Width-int32(Spacing.MD)
	var closeR *swt.Rectangle
	if r.ShouldAllocateCloseRect(item) {
		closeR = swt.NewRectangle(right-tabClose, b.Y+(b.Height-tabClose)/2, tabClose, tabClose)
		right = closeR.X - swt.CTabFolderRendererINTERNAL_SPACING
	}
	if img := item.GetImage(); img != nil && !img.IsDisposed() {
		ib := img.GetBounds()
		if x+ib.Width <= right {
			gc.DrawImage(img, x, bodyY+(bodyH-ib.Height)/2)
			x += ib.Width + swt.CTabFolderRendererINTERNAL_SPACING
		}
	}
	if w := right - x; w > 0 {
		gc.SetFont(item.GetFont())
		text := r.ShortenText(gc, item.GetText(), w)
		ext := gc.TextExtentStringFlags(text, swt.CTabFolderRendererFLAGS)
		col := p.dim
		if sel || hot {
			col = p.fg
		}
		gc.SetForeground(r.c(col))
		gc.DrawTextStringXYFlags(text, x, bodyY+(bodyH-ext.Y)/2, swt.CTabFolderRendererFLAGS)
	}
	if sel && r.f.IsFocusControl() {
		gc.SetAntialias(swt.ON)
		gc.SetLineWidth(2)
		gc.SetForeground(r.c(p.accent))
		gc.DrawRoundRectangle(b.X+3, bodyY+3, b.Width-7, bodyH-6, 8, 8)
		gc.SetLineWidth(1)
	}
	if closeR != nil && (sel || hot || r.ShouldDrawDirtyIndicator(item)) {
		r.drawClose(p, item, closeR, gc)
	}
}

// overRect reports whether the pointer is inside rect, which is in the folder's coordinates.
func (r *tabRenderer) overRect(rect *swt.Rectangle) bool {
	pt := r.f.ToControlPoint(r.f.GetDisplay().GetCursorLocation())
	return rect.Contains(pt.X, pt.Y)
}

// drawClose draws the cross, with a round background under the pointer; a modified tab shows a dot
// instead of the cross until the pointer is over the button.
func (r *tabRenderer) drawClose(p tabPalette, item *swt.CTabItem, rect *swt.Rectangle, gc *swt.GC) {
	gc.SetAntialias(swt.ON)
	cx, cy := rect.X+rect.Width/2, rect.Y+rect.Height/2
	over := r.overRect(rect)
	if over {
		gc.SetBackground(r.c(mixRGB(p.strip, p.fg, 0.16)))
		gc.FillOval(cx-8, cy-8, 16, 16)
	}
	if r.ShouldDrawDirtyIndicator(item) && !over {
		gc.SetBackground(r.c(p.fg))
		gc.FillOval(cx-4, cy-4, 8, 8)
		return
	}
	col := p.dim
	if over {
		col = p.fg
	}
	gc.SetForeground(r.c(col))
	gc.SetLineWidth(1)
	gc.SetLineCap(swt.CAP_ROUND)
	gc.DrawLine(cx-3, cy-3, cx+3, cy+3)
	gc.DrawLine(cx-3, cy+3, cx+3, cy-3)
	gc.SetLineCap(swt.CAP_FLAT)
}

func (r *tabRenderer) drawChevron(state int32, b *swt.Rectangle, gc *swt.GC) {
	p := r.palette()
	gc.SetAntialias(swt.ON)
	cx, cy := b.X+b.Width/2, b.Y+b.Height/2
	if state&(swt.HOT|swt.SELECTED) != 0 {
		gc.SetBackground(r.c(mixRGB(p.strip, p.fg, 0.12)))
		gc.FillRoundRectangle(b.X, b.Y, b.Width, b.Height, 8, 8)
	}
	gc.SetForeground(r.c(p.dim))
	gc.SetLineWidth(1)
	gc.SetLineCap(swt.CAP_ROUND)
	gc.DrawPolyline([]int32{cx - 4, cy - 2, cx, cy + 2, cx + 4, cy - 2})
	gc.SetLineCap(swt.CAP_FLAT)
}

// redrawOnTheme repaints f when the system switches between light and dark; the App's own listener,
// added before setup, has updated its dark flag by then.
func redrawOnTheme(f *swt.CTabFolder) {
	d := f.GetDisplay()
	l := &tabTheme{f}
	d.AddListener(swt.Settings, l)
	f.AddDisposeListener(disposeFunc(func() { d.RemoveListener(swt.Settings, l) }))
}

// tabTheme is a pointer so that Display.RemoveListener can compare it.
type tabTheme struct{ f *swt.CTabFolder }

func (t *tabTheme) HandleEvent(*swt.Event) { t.f.Redraw() }
