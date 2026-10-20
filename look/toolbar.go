package look

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
)

// titleHookKey is the shell data key of the func(string) that gowt.Window.SetTitle calls;
// gowt has the same key.
const titleHookKey = "gowt.look.title"

// barSymbols are the SF Symbols the bar draws itself, as strokes in a 24-point box; an item
// with another symbol shows its title.
var barSymbols = map[string]string{
	"chevron.backward":    `<path d="M15 6l-6 6 6 6"/>`,
	"chevron.forward":     `<path d="M9 6l6 6-6 6"/>`,
	"plus":                `<path d="M12 5v14M5 12h14"/>`,
	"sidebar.left":        `<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/>`,
	"square.and.arrow.up": `<path d="M12 3v12M8 7l4-4 4 4M6 11v8a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-8"/>`,
}

const barSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</svg>`

// toolbarBar is the toolbar of a window that has no native one: a strip of gowt widgets at the top of
// the SetSidebar content column, in the content's colour.
type toolbarBar struct {
	w       *gowt.Window
	items   []ToolbarItem
	content *swt.Control // the SetSidebar content; nil leaves the bar unbuilt
	sidebar Container    // the SetSidebar sidebar, which gets the Sidebar items in a row of its own
	panel   *gowt.Panel
	side    *gowt.Panel // the row of Sidebar items at the top of the sidebar
	// lead holds copies of the Sidebar items at the start of the bar, shown while the sidebar is hidden
	lead  []gowt.Widget
	title *gowt.LabelView
	icons []func(on, off string) []*swt.Image // set the images of one tool item in the given colours
	// sideIcons are the icons of the side row, coloured for the sidebar background
	sideIcons []func(on, off string) []*swt.Image
	images    []*swt.Image
	bg        swt.RGB // content colour the bar was styled for
	line      *swt.Color
}

var bars = map[*swt.Shell]*toolbarBar{}

func barOf(w *gowt.Window) *toolbarBar {
	s := w.Unwrap()
	if b := bars[s]; b != nil {
		return b
	}
	b := &toolbarBar{w: w}
	bars[s] = b
	s.AddListener(swt.Dispose, b)
	s.GetDisplay().AddListener(swt.Settings, b)
	s.SetDataKeyValue(titleHookKey, func(t string) {
		if b.title != nil {
			b.title.SetText(t)
			b.panel.Unwrap().Layout()
		}
	})
	return b
}

func (b *toolbarBar) HandleEvent(e *swt.Event) {
	s := b.w.Unwrap()
	if e.Type == swt.Dispose {
		s.GetDisplay().RemoveListener(swt.Settings, b)
		disposeImages(b.images)
		delete(bars, s)
		return
	}
	// The app recolours its content on this same event; take the new colour after it has.
	s.GetDisplay().AsyncExec(jrt.NewRunnable(func() {
		if !s.IsDisposed() {
			b.restyle(true)
		}
	}))
}

// build makes the bar again from the items, or removes it when there are none or no content column.
func (b *toolbarBar) build() {
	if b.panel != nil {
		b.panel.Unwrap().Dispose()
		b.panel, b.title, b.icons, b.lead = nil, nil, nil, nil
		disposeImages(b.images)
		b.images = nil
	}
	if b.side != nil {
		b.side.Unwrap().Dispose()
		b.side, b.sideIcons = nil, nil
	}
	if b.content != nil && len(b.items) > 0 {
		b.fill()
	}
	if b.sidebar != nil {
		b.sidebar.AsComposite().Layout()
	}
	b.w.Unwrap().Layout()
}

// sideAdder is the sidebar as a gowt container the side row can be added to; nil if it is not one.
func (b *toolbarBar) sideAdder() interface{ Add(...gowt.Option) } {
	a, _ := b.sidebar.(interface{ Add(...gowt.Option) })
	return a
}

// fill lays the items out as macOS does: Nav items, the window title, then the rest. The title takes
// the free width unless a Space item does.
func (b *toolbarBar) fill() {
	var lead, trail, over []ToolbarItem
	spaced := false
	side := b.sideAdder()
	for _, it := range b.items {
		if it.Sidebar && !it.Space && side != nil {
			over = append(over, it)
			continue
		}
		if it.Nav {
			lead = append(lead, it)
			continue
		}
		trail = append(trail, it)
		spaced = spaced || it.Space
	}
	var p *gowt.Panel
	b.w.Add(gowt.Grid(len(over)+len(lead)+1+len(trail), gowt.Padding(8), gowt.Gap(6), gowt.Ref(&p)))
	p.Unwrap().SetBackgroundMode(swt.INHERIT_DEFAULT)
	p.Unwrap().AddPaintListener(painter(func(e *swt.PaintEvent) {
		r := p.Unwrap().GetClientArea()
		e.Gc.SetForeground(b.line)
		e.Gc.DrawLine(0, r.Height-1, r.Width, r.Height-1)
	}))
	b.panel = p
	shown := b.sidebar.AsComposite().GetVisible()
	for _, it := range over {
		if c := b.item(p, it, &b.icons); c != nil {
			c.SetVisible(!shown)
			b.lead = append(b.lead, c)
		}
	}
	for _, it := range lead {
		b.item(p, it, &b.icons)
	}
	p.Add(gowt.Label(b.w.Unwrap().GetText(), gowt.Cell(gowt.GridCell{GrowX: !spaced, Align: gowt.AlignFill, VAlign: gowt.AlignCenter}), gowt.Ref(&b.title)))
	size := 0
	if fd := b.title.Unwrap().GetFont().GetFontData(); len(fd) > 0 {
		size = int(fd[0].GetHeight()) + 2
	}
	b.title.SetFont(gowt.Font{Size: size, Weight: gowt.Semibold})
	for _, it := range trail {
		b.item(p, it, &b.icons)
	}
	if len(over) > 0 {
		// The row goes first in the sidebar, its items at the right end, as over the macOS sidebar.
		side.Add(gowt.Grid(len(over)+1, gowt.Padding(8), gowt.Gap(6), gowt.GrowX(), gowt.Ref(&b.side)))
		b.side.Unwrap().MoveAbove(nil)
		b.side.Unwrap().SetBackgroundMode(swt.INHERIT_DEFAULT)
		b.side.Add(gowt.Spacer())
		for _, it := range over {
			b.item(b.side, it, &b.sideIcons)
		}
	}
	b.restyle(true)
}

// item adds it to p and returns its widget, nil for a Space.
func (b *toolbarBar) item(p *gowt.Panel, it ToolbarItem, icons *[]func(on, off string) []*swt.Image) gowt.Widget {
	switch {
	case it.Space:
		p.Add(gowt.Spacer())
		return nil
	case it.OnSearch != nil:
		var t *gowt.TextView
		p.Add(gowt.Text(gowt.Search(), gowt.Size(240, 0), gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Ref(&t)))
		t.SetHint(it.Title)
		t.OnChange(it.OnSearch)
		return t
	default:
		var tb *gowt.ToolBarView
		p.Add(gowt.ToolBar(gowt.Flat(), gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Ref(&tb)))
		segs := it.Group
		if len(segs) == 0 {
			segs = []ToolbarItem{it}
		}
		for _, g := range segs {
			toolItem(tb, g, icons)
		}
		return tb
	}
}

// sidebarShown moves the Sidebar items to the start of the bar while the sidebar is hidden; the row in
// the sidebar hides with it.
func (b *toolbarBar) sidebarShown(on bool) {
	for _, c := range b.lead {
		c.SetVisible(!on)
	}
}

// toolItem adds a flat button whose symbol icons sets in the bar's text colour, dimmed while Enabled
// is false.
func toolItem(tb *gowt.ToolBarView, it ToolbarItem, icons *[]func(on, off string) []*swt.Image) {
	shape, ok := barSymbols[it.Symbol]
	text := ""
	if !ok {
		text = it.Title
	}
	ti := tb.Item(text, it.OnClick)
	ti.SetTooltip(it.Title)
	if ok {
		data := fmt.Sprintf(barSVG, shape)
		*icons = append(*icons, func(on, off string) []*swt.Image {
			d := tb.Unwrap().GetDisplay()
			img := swt.NewImageDeviceImageDataProvider(d, svg.NewImageDataProvider([]byte(data), 16, 16, on))
			dim := swt.NewImageDeviceImageDataProvider(d, svg.NewImageDataProvider([]byte(data), 16, 16, off))
			ti.Unwrap().SetDisabledImage(dim)
			ti.Unwrap().SetImage(img)
			return []*swt.Image{img, dim}
		})
	}
	if it.Enabled != nil {
		gowt.Watch(tb, it.Enabled, ti.SetEnabled)
	}
}

// restyle gives the bar the content's background and a text colour that reads on it; unless forced
// it does nothing while the content colour stays the same.
func (b *toolbarBar) restyle(force bool) {
	if b.panel == nil {
		return
	}
	bg := b.content.GetBackground().GetRGB()
	if !force && *bg == b.bg {
		return
	}
	b.bg = *bg
	fg := barForeground(*b.w.Unwrap().GetDisplay().GetSystemColor(swt.COLOR_WIDGET_FOREGROUND).GetRGB(), *bg)
	p := b.panel.Unwrap()
	p.SetBackgroundWithColor(swt.NewColorRgb(bg))
	b.title.Unwrap().SetForeground(swt.NewColorRgb(&fg))
	b.line = swt.NewColorRgb(blend(*bg, fg, 12))
	old := b.images
	b.images = nil
	for _, set := range b.icons {
		b.images = append(b.images, set(hexRGB(fg), hexRGB(*blend(*bg, fg, 40)))...)
	}
	if b.side != nil {
		sbg := b.side.Unwrap().GetBackground().GetRGB()
		sfg := barForeground(*b.w.Unwrap().GetDisplay().GetSystemColor(swt.COLOR_WIDGET_FOREGROUND).GetRGB(), *sbg)
		for _, set := range b.sideIcons {
			b.images = append(b.images, set(hexRGB(sfg), hexRGB(*blend(*sbg, sfg, 40)))...)
		}
	}
	disposeImages(old)
	p.Redraw()
}

// barForeground is the system text colour fg, or near black or near white where fg does not read on
// bg: the app may paint its content dark while the system content stays light (Windows).
func barForeground(fg, bg swt.RGB) swt.RGB {
	if (luma(fg) > 128) != (luma(bg) > 128) {
		return fg
	}
	if luma(bg) > 128 {
		return swt.RGB{Red: 28, Green: 28, Blue: 30}
	}
	return swt.RGB{Red: 236, Green: 236, Blue: 238}
}

type painter func(*swt.PaintEvent)

func (f painter) PaintControl(e *swt.PaintEvent) { f(e) }

func luma(c swt.RGB) int32 { return (299*c.Red + 587*c.Green + 114*c.Blue) / 1000 }

// blend moves pct percent of the way from a to b.
func blend(a, b swt.RGB, pct int32) *swt.RGB {
	m := func(x, y int32) int32 { return x + (y-x)*pct/100 }
	return &swt.RGB{Red: m(a.Red, b.Red), Green: m(a.Green, b.Green), Blue: m(a.Blue, b.Blue)}
}

func hexRGB(c swt.RGB) string { return fmt.Sprintf("#%02x%02x%02x", c.Red, c.Green, c.Blue) }

func disposeImages(imgs []*swt.Image) {
	for _, i := range imgs {
		i.Dispose()
	}
}

// barTop places the bar of s at the top of the content frame ct and returns the frame left below it.
func barTop(s *swt.Shell, ct [4]int32) [4]int32 {
	b := bars[s]
	if b == nil || b.panel == nil {
		return ct
	}
	c := b.panel.Unwrap()
	h := min(c.ComputeSize(ct[2], swt.DEFAULT).Y, ct[3])
	c.SetBounds(ct[0], ct[1], ct[2], h)
	b.restyle(false)
	return [4]int32{ct[0], ct[1] + h, ct[2], ct[3] - h}
}

// barHeight is the height the bar of s asks for, 0 without one.
func barHeight(s *swt.Shell) int32 {
	if b := bars[s]; b != nil && b.panel != nil {
		return b.panel.Unwrap().ComputeSize(swt.DEFAULT, swt.DEFAULT).Y
	}
	return 0
}
