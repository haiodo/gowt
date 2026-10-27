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

// barSymbols are the SF Symbols the bar draws itself, as strokes in a 24-point box (fill="currentColor"
// where the symbol is solid); an item with another symbol shows its title as text.
var barSymbols = map[string]string{
	"chevron.backward":    `<path d="M15 6l-6 6 6 6"/>`,
	"chevron.forward":     `<path d="M9 6l6 6-6 6"/>`,
	"plus":                `<path d="M12 5v14M5 12h14"/>`,
	"square.and.arrow.up": `<path d="M12 3v12M8 7l4-4 4 4M6 11v8a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2v-8"/>`,

	"square.and.arrow.down":      `<path d="M12 3v12M8 11l4 4 4-4M5 15v4a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-4"/>`,
	"terminal":                   `<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M7 10l3 2-3 2M12 15h5"/>`,
	"shippingbox":                `<path d="M12 3l8 4v10l-8 4-8-4V7zM4 7l8 4 8-4M12 11v10"/>`,
	"folder":                     `<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>`,
	"doc":                        `<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8zM14 3v5h5"/>`,
	"house":                      `<path d="M4 11l8-7 8 7M6 10v10h12V10M10 20v-6h4v6"/>`,
	"sidebar.left":               `<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/>`,
	"sidebar.right":              `<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M15 4v16"/>`,
	"gearshape":                  `<circle cx="12" cy="12" r="3"/><circle cx="12" cy="12" r="6.5"/><path d="M12 2.5v3M12 18.5v3M2.5 12h3M18.5 12h3M5.3 5.3l2.1 2.1M16.6 16.6l2.1 2.1M18.7 5.3l-2.1 2.1M7.4 16.6l-2.1 2.1"/>`,
	"magnifyingglass":            `<circle cx="10.5" cy="10.5" r="6.5"/><path d="M15.5 15.5L21 21"/>`,
	"trash":                      `<path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"/>`,
	"arrow.clockwise":            `<path d="M20 12a8 8 0 1 1-2.34-5.66M17.7 2.5v4.5h-4.5"/>`,
	"arrow.up":                   `<path d="M12 20V5M6 11l6-6 6 6"/>`,
	"arrow.down":                 `<path d="M12 4v15M6 13l6 6 6-6"/>`,
	"checkmark":                  `<path d="M5 12.5l4.5 4.5L19 7"/>`,
	"xmark":                      `<path d="M6 6l12 12M18 6L6 18"/>`,
	"minus":                      `<path d="M5 12h14"/>`,
	"play.fill":                  `<path d="M7 4.5v15l12-7.5z" fill="currentColor"/>`,
	"stop.fill":                  `<rect x="6" y="6" width="12" height="12" rx="2" fill="currentColor"/>`,
	"pause.fill":                 `<rect x="6" y="5" width="4" height="14" rx="1" fill="currentColor"/><rect x="14" y="5" width="4" height="14" rx="1" fill="currentColor"/>`,
	"info.circle":                `<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>`,
	"ellipsis.circle":            `<circle cx="12" cy="12" r="9"/><path d="M8 12h.01M12 12h.01M16 12h.01"/>`,
	"line.3.horizontal.decrease": `<path d="M4 7h16M7 12h10M10 17h4"/>`,
	"list.bullet":                `<path d="M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01"/>`,
	"square.grid.2x2":            `<rect x="4" y="4" width="6.5" height="6.5" rx="1.5"/><rect x="13.5" y="4" width="6.5" height="6.5" rx="1.5"/><rect x="4" y="13.5" width="6.5" height="6.5" rx="1.5"/><rect x="13.5" y="13.5" width="6.5" height="6.5" rx="1.5"/>`,
}

const barSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</svg>`

// toolbarBar is the toolbar of a window that has no native one: a strip of gowt widgets at the top of
// the SetSidebar or SetInspector content column, in the content's colour.
type toolbarBar struct {
	w       *gowt.Window
	items   []ToolbarItem
	content *swt.Control // the SetSidebar or SetInspector content; nil leaves the bar unbuilt
	sidebar Container    // the SetSidebar sidebar, which gets the Sidebar items in a row of its own
	panel   *gowt.Panel
	side    *gowt.Panel // the row of Sidebar items at the top of the sidebar
	// lead holds copies of the Sidebar items at the start of the bar, shown while the sidebar is hidden
	lead  []gowt.Widget
	title *gowt.LabelView
	icons []func(on, off string) []*swt.Image // set the images of one tool item in the given colours
	// sideIcons are the icons of the side row, coloured for the sidebar background
	sideIcons []func(on, off string) []*swt.Image
	// strip is the row of Inspector items: over the inspector, or at the end of the bar while it is hidden
	strip      *gowt.Panel
	stripIcons []func(on, off string) []*swt.Image
	images     []*swt.Image
	bg         swt.RGB // content colour the bar was styled for
	stripBG    swt.RGB // colour the strip was styled for: the inspector's, or the content's while it is hidden
	line       *swt.Color
	stripLine  *swt.Color
	search     *gowt.TextView // the search field of an OnSearch item
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
		b.panel, b.title, b.icons, b.lead, b.search = nil, nil, nil, nil, nil
		disposeImages(b.images)
		b.images = nil
	}
	if b.side != nil {
		b.side.Unwrap().Dispose()
		b.side, b.sideIcons = nil, nil
	}
	if b.strip != nil {
		b.strip.Unwrap().Dispose()
		b.strip, b.stripIcons = nil, nil
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

// inspectorPane is the SetInspector pane of the window, nil without one.
func (b *toolbarBar) inspectorPane() *pane { return inspectorOf(b.w) }

// fill lays the items out as macOS does: Nav items, the window title, then the rest, and the Inspector
// items in a strip of their own. The title takes the free width unless a Space item does.
func (b *toolbarBar) fill() {
	var lead, trail, over, insp []ToolbarItem
	spaced := false
	side := b.sideAdder()
	inspector := b.inspectorPane() != nil
	for _, it := range b.items {
		if it.Sidebar && !it.Space && side != nil {
			over = append(over, it)
			continue
		}
		if it.Inspector && inspector {
			insp = append(insp, it)
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
	b.w.Add(gowt.Grid(len(over)+len(lead)+1+len(trail), gowt.Padding(gowt.Scaled(8)), gowt.Gap(gowt.Scaled(6)), gowt.Ref(&p)))
	p.Unwrap().SetBackgroundMode(swt.INHERIT_DEFAULT)
	p.Unwrap().AddPaintListener(painter(func(e *swt.PaintEvent) {
		r := p.Unwrap().GetClientArea()
		e.Gc.SetForeground(b.line)
		e.Gc.DrawLine(0, r.Height-1, r.Width, r.Height-1)
	}))
	b.panel = p
	shown := b.sidebar != nil && b.sidebar.AsComposite().GetVisible()
	for _, it := range over {
		it.ID = "" // the copy is hidden while the sidebar shows; the ID belongs to the item in the sidebar row
		if c := b.item(p, it, &b.icons); c != nil {
			c.SetVisible(!shown)
			b.lead = append(b.lead, c)
		}
	}
	for _, it := range lead {
		b.item(p, it, &b.icons)
	}
	p.Add(gowt.Label(b.w.Unwrap().GetText(), gowt.Cell(gowt.GridCell{GrowX: !spaced, Align: gowt.AlignFill, VAlign: gowt.AlignCenter}), gowt.Ref(&b.title)))
	if gowt.ChosenDensity().Legacy() {
		size := 0
		if fd := b.title.Unwrap().GetFont().GetFontData(); len(fd) > 0 {
			size = int(fd[0].GetHeight()) + 2
		}
		b.title.SetFont(gowt.Font{Size: size, Weight: gowt.Semibold})
	} else {
		b.title.SetFont(gowt.Type.Subtitle)
	}
	for _, it := range trail {
		b.item(p, it, &b.icons)
	}
	if len(over) > 0 {
		// The row goes first in the sidebar, its items at the right end, as over the macOS sidebar.
		side.Add(gowt.Grid(len(over)+1, gowt.Padding(gowt.Scaled(8)), gowt.Gap(gowt.Scaled(6)), gowt.GrowX(), gowt.Ref(&b.side)))
		b.side.Unwrap().MoveAbove(nil)
		b.side.Unwrap().SetBackgroundMode(swt.INHERIT_DEFAULT)
		b.side.Add(gowt.Spacer())
		for _, it := range over {
			b.item(b.side, it, &b.sideIcons)
		}
	}
	if len(insp) > 0 {
		// The strip is a window child that barTop places, as the bar is: inside the inspector it would
		// take the inspector's padding and miss the bar's height.
		b.w.Add(gowt.Grid(len(insp), gowt.Padding(gowt.Scaled(8)), gowt.Gap(gowt.Scaled(6)), gowt.Ref(&b.strip)))
		st := b.strip.Unwrap()
		st.SetBackgroundMode(swt.INHERIT_DEFAULT)
		st.AddPaintListener(painter(func(e *swt.PaintEvent) {
			r := st.GetClientArea()
			e.Gc.SetForeground(b.stripLine)
			e.Gc.DrawLine(0, r.Height-1, r.Width, r.Height-1)
		}))
		for _, it := range insp {
			b.item(b.strip, it, &b.stripIcons)
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
		opts := []gowt.Option{gowt.Search(), gowt.Size(gowt.Scaled(240), 0), gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Ref(&t)}
		if it.ID != "" {
			opts = append(opts, gowt.ID(it.ID))
		}
		if it.Small {
			opts = append(opts, gowt.Typeface(smallFont))
		}
		p.Add(gowt.Text(opts...))
		t.SetHint(it.Title)
		t.OnChange(it.OnSearch)
		t.Unwrap().AddSelectionListener(swt.SelectionListenerWidgetDefaultSelectedAdapter(func(*swt.SelectionEvent) { it.OnSearch(t.Text()) }))
		if b.search == nil {
			b.search = t
		}
		return t
	case it.Text != nil && len(it.Group) == 0:
		var bt *gowt.ButtonView
		opts := []gowt.Option{gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Tooltip(it.Title), gowt.Ref(&bt)}
		if it.OnClick != nil {
			opts = append(opts, gowt.OnClick(it.OnClick))
		}
		if it.ID != "" {
			opts = append(opts, gowt.ID(it.ID))
		}
		if it.Small {
			opts = append(opts, gowt.Typeface(smallFont))
		}
		p.Add(gowt.Button(it.Text.Peek(), opts...))
		gowt.Watch(bt, it.Text, bt.SetText)
		if it.Enabled != nil {
			gowt.Watch(bt, it.Enabled, bt.SetEnabled)
		}
		return bt
	default:
		var tb *gowt.ToolBarView
		p.Add(gowt.ToolBar(gowt.Flat(), gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Ref(&tb)))
		segs := it.Group
		if len(segs) == 0 {
			segs = []ToolbarItem{it}
		}
		size := 16
		if it.Small {
			size = 13
			tb.SetFont(smallFont)
		}
		for _, g := range segs {
			toolItem(tb, g, size, icons)
		}
		return tb
	}
}

// smallFont is the text of a Small item: 11 to 13, as the small and regular control fonts of macOS.
var smallFont = gowt.Font{Scale: 11.0 / 13}

// sidebarShown moves the Sidebar items to the start of the bar while the sidebar is hidden; the row in
// the sidebar hides with it.
func (b *toolbarBar) sidebarShown(on bool) {
	for _, c := range b.lead {
		c.SetVisible(!on)
	}
}

// toolItem adds a flat button whose symbol, size points square, icons sets in the bar's text colour,
// dimmed while Enabled is false.
func toolItem(tb *gowt.ToolBarView, it ToolbarItem, size int, icons *[]func(on, off string) []*swt.Image) {
	shape, ok := barSymbols[it.Symbol]
	text := ""
	if !ok {
		text = it.Title
	}
	ti := tb.Item(text, it.OnClick)
	ti.SetTooltip(it.Title)
	if it.ID != "" {
		ti.Unwrap().SetDataKeyValue("gowt.id", it.ID) // qatree.IDKey
	}
	if ok {
		data := fmt.Sprintf(barSVG, shape)
		*icons = append(*icons, func(on, off string) []*swt.Image {
			d := tb.Unwrap().GetDisplay()
			img := swt.NewImageDeviceImageDataProvider(d, svg.NewImageDataProvider([]byte(data), size, size, on))
			dim := swt.NewImageDeviceImageDataProvider(d, svg.NewImageDataProvider([]byte(data), size, size, off))
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
	sbg := *bg
	if p := b.inspectorPane(); b.strip != nil && p.visible() {
		sbg = *p.c.GetBackground().GetRGB()
	}
	if !force && *bg == b.bg && sbg == b.stripBG {
		return
	}
	b.bg, b.stripBG = *bg, sbg
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
	if b.strip != nil {
		tfg := barForeground(*b.w.Unwrap().GetDisplay().GetSystemColor(swt.COLOR_WIDGET_FOREGROUND).GetRGB(), sbg)
		b.strip.Unwrap().SetBackgroundWithColor(swt.NewColorRgb(&sbg))
		b.stripLine = swt.NewColorRgb(blend(sbg, tfg, 12))
		for _, set := range b.stripIcons {
			b.images = append(b.images, set(hexRGB(tfg), hexRGB(*blend(sbg, tfg, 40)))...)
		}
		b.strip.Unwrap().Redraw()
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

// barTop places the bar of s at the top of the content frame ct and its strip of Inspector items at
// the top of the inspector frame ib, or at the end of the bar where ib is 0 wide (a hidden inspector),
// and returns the frames left below them.
func barTop(s *swt.Shell, ct, ib [4]int32) ([4]int32, [4]int32) {
	b := bars[s]
	if b == nil || b.panel == nil {
		return ct, ib
	}
	h := min(b.height(ct[2]), ct[3])
	w := ct[2]
	if b.strip != nil {
		st := b.strip.Unwrap()
		if ib[2] > 0 {
			st.SetBounds(ib[0], ib[1], ib[2], h)
			ib = [4]int32{ib[0], ib[1] + h, ib[2], max(ib[3]-h, 0)}
		} else {
			sw := min(st.ComputeSize(swt.DEFAULT, h).X, w)
			w -= sw
			st.SetBounds(ct[0]+w, ct[1], sw, h)
		}
	}
	b.panel.Unwrap().SetBounds(ct[0], ct[1], w, h)
	b.restyle(false)
	return [4]int32{ct[0], ct[1] + h, ct[2], ct[3] - h}, ib
}

// height is the height of the bar width points wide, as tall as the strip if that is taller.
func (b *toolbarBar) height(width int32) int32 {
	h := b.panel.Unwrap().ComputeSize(width, swt.DEFAULT).Y
	if b.strip != nil {
		h = max(h, b.strip.Unwrap().ComputeSize(swt.DEFAULT, swt.DEFAULT).Y)
	}
	return h
}

// barHeight is the height the bar of s asks for, 0 without one.
func barHeight(s *swt.Shell) int32 {
	if b := bars[s]; b != nil && b.panel != nil {
		return b.height(swt.DEFAULT)
	}
	return 0
}
