package main

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/icons"
	"github.com/haiodo/gowt/swt"
)

var (
	bgApp      = gowt.Adaptive{Light: gowt.RGB{R: 244, G: 245, B: 247}, Dark: gowt.RGB{R: 17, G: 20, B: 22}}
	bgSidebar  = gowt.Adaptive{Light: gowt.RGB{R: 238, G: 240, B: 242}, Dark: gowt.RGB{R: 21, G: 24, B: 26}}
	bgCard     = gowt.Adaptive{Light: gowt.RGB{R: 255, G: 255, B: 255}, Dark: gowt.RGB{R: 40, G: 42, B: 53}}
	bgShell    = gowt.Adaptive{Light: gowt.RGB{R: 248, G: 249, B: 250}, Dark: gowt.RGB{R: 30, G: 32, B: 41}}
	cardBorder = gowt.Adaptive{Light: gowt.RGB{R: 205, G: 208, B: 212}, Dark: gowt.RGB{R: 61, G: 63, B: 73}}
	bgField    = gowt.Adaptive{Light: gowt.RGB{R: 225, G: 228, B: 232}, Dark: gowt.RGB{R: 37, G: 40, B: 42}}
	bgSelected = gowt.Adaptive{Light: gowt.RGB{R: 214, G: 220, B: 226}, Dark: gowt.RGB{R: 56, G: 60, B: 63}}
	fg         = gowt.Adaptive{Light: gowt.RGB{R: 32, G: 35, B: 38}, Dark: gowt.RGB{R: 202, G: 211, B: 211}}
	fgRow      = gowt.Adaptive{Light: gowt.RGB{R: 70, G: 76, B: 82}, Dark: gowt.RGB{R: 146, G: 154, B: 155}}
	shellFG    = gowt.Adaptive{Light: gowt.RGB{R: 70, G: 76, B: 82}, Dark: gowt.RGB{R: 146, G: 154, B: 155}}
	selectedFG = gowt.Adaptive{Light: gowt.RGB{R: 25, G: 28, B: 31}, Dark: gowt.RGB{R: 201, G: 211, B: 211}}
	fgDim      = gowt.Adaptive{Light: gowt.RGB{R: 100, G: 108, B: 114}, Dark: gowt.RGB{R: 136, G: 143, B: 144}}
	teal       = gowt.Adaptive{Light: gowt.RGB{R: 0, G: 120, B: 105}, Dark: gowt.RGB{R: 135, G: 217, B: 197}}
	green      = gowt.Adaptive{Light: gowt.RGB{R: 30, G: 130, B: 55}, Dark: gowt.RGB{R: 120, G: 200, B: 140}}
	red        = gowt.Adaptive{Light: gowt.RGB{R: 190, G: 45, B: 45}, Dark: gowt.RGB{R: 230, G: 120, B: 120}}
	blue       = gowt.Adaptive{Light: gowt.RGB{R: 30, G: 95, B: 165}, Dark: gowt.RGB{R: 104, G: 154, B: 204}}
	purple     = gowt.Adaptive{Light: gowt.RGB{R: 105, G: 65, B: 165}, Dark: gowt.RGB{R: 158, G: 128, B: 208}}
	amber      = gowt.Adaptive{Light: gowt.RGB{R: 140, G: 95, B: 15}, Dark: gowt.RGB{R: 200, G: 170, B: 90}}
	tileBlue   = gowt.RGB{R: 10, G: 132, B: 255}
	tilePurple = gowt.RGB{R: 130, G: 80, B: 230}
	tileGreen  = gowt.RGB{R: 48, G: 176, B: 78}
	tileGray   = gowt.RGB{R: 142, G: 142, B: 147}
)

var rexDark = true

func colour(c gowt.Adaptive) gowt.RGB {
	if rexDark {
		return c.Dark
	}
	return c.Light
}

func watchTheme(app *gowt.App) {
	rexDark = app.Dark()
	app.OnThemeChange(func(dark bool) { rexDark = dark })
}

func setAppearance(app *gowt.App, mode gowt.Appearance) {
	app.SetAppearance(mode)
}

type listItem struct {
	title, detail string
	icon          *gowt.Image
	value         int
	group         bool
	space         int
}

type drawnList struct {
	app      *gowt.App
	items    []listItem
	selected int
	rowH     int
	bg       gowt.Adaptive
	canvas   *gowt.CanvasView
	onSelect func(int)
	closable bool
	onClose  func(int)
	hover    int
	id       string
	focus    bool
}

type drawnRow struct {
	item, y, h int
	separator  bool
}

// Row insets: the icon and group header start here, the title past the icon, and the close X or More
// dots keep the right strip free.
const (
	drawnIconX   = 20
	drawnTextX   = 48
	drawnPillX   = 10
	drawnEndRoom = 30
)

func newDrawnList(app *gowt.App, bg gowt.Adaptive, rowH int, items []listItem, selected int, onSelect func(int)) *drawnList {
	return &drawnList{app: app, items: items, selected: selected, rowH: rowH, bg: bg, onSelect: onSelect, hover: -1}
}

func (l *drawnList) Option() gowt.Option {
	opts := []gowt.Option{gowt.Grow(), gowt.Ref(&l.canvas),
		gowt.AccessibleLabel("Sessions"), gowt.OnMouseDown(l.click),
		gowt.OnMouseMove(l.move), gowt.OnMouseLeave(l.leave),
		gowt.Init(func(c *gowt.CanvasView) {
			c.Unwrap().AddListener(swt.KeyDown, keyMove{l})
			l.app.OnThemeChange(func(bool) { c.Redraw() })
		})}
	if l.id != "" {
		opts = append(opts, gowt.ID(l.id))
	}
	if l.focus {
		opts = append(opts, gowt.Focused())
	}
	return gowt.Canvas(l.draw, opts...)
}

func (l *drawnList) SetItems(items []listItem) {
	l.items = items
	l.hover = -1
	l.redraw()
}

// step selects the row delta steps from the selected one and reports whether it moved: group, space
// and caption rows are skipped, and the first and last rows do not wrap.
func (l *drawnList) step(delta int) bool {
	at := 0
	for i, it := range l.items {
		if it.value == l.selected {
			at = i
			break
		}
	}
	for i := at + delta; i >= 0 && i < len(l.items); i += delta {
		it := l.items[i]
		if it.group || it.title == "" || it.value < 0 {
			continue
		}
		l.Select(it.value)
		if l.onSelect != nil {
			l.onSelect(it.value)
		}
		return true
	}
	return false
}

func (l *drawnList) Select(value int) {
	l.selected = value
	l.redraw()
}

func (l *drawnList) redraw() {
	if l.canvas != nil {
		l.canvas.Redraw()
	}
}

func (l *drawnList) layoutRows() []drawnRow {
	y := 0
	rows := make([]drawnRow, 0, len(l.items))
	for i, it := range l.items {
		if it.space > 0 {
			if it.title == "" && it.icon == nil && !it.group && it.space == 9 {
				rows = append(rows, drawnRow{item: i, y: y, h: it.space, separator: true})
			}
			y += it.space
		}
		if it.title == "" && !it.group {
			continue
		}
		h := l.rowH
		if it.group {
			h = 26
		} else if it.detail != "" {
			h = l.rowH + 4
		}
		rows = append(rows, drawnRow{item: i, y: y, h: h})
		y += h
	}
	return rows
}

func (l *drawnList) draw(g *gowt.GC) {
	w, h := g.Size()
	g.SetFill(colour(l.bg))
	g.FillRect(0, 0, w, h)
	for _, row := range l.layoutRows() {
		it := l.items[row.item]
		if row.separator {
			g.SetFill(colour(bgField))
			g.FillRect(drawnIconX, row.y+row.h/2, w-2*drawnIconX, 1)
			continue
		}
		g.SetFill(colour(l.bg))
		if it.group {
			x := drawnIconX
			if it.icon != nil {
				g.Image(currentSymbol(l.app, it.icon), drawnIconX, row.y+(row.h-14)/2)
				x = drawnIconX + 20
			}
			g.SetFont(semibold(gowt.Type.Caption))
			g.SetColor(colour(fgRow))
			_, th := g.TextSize(it.title)
			g.Text(it.title, x, row.y+(row.h-th)/2)
			continue
		}
		selected := it.value >= 0 && it.value == l.selected
		hovered := l.closable && row.item == l.hover && it.value >= 0
		if selected {
			fillRound(g, drawnPillX, row.y+2, w-2*drawnPillX, row.h-4, 7, bgSelected)
			g.SetFill(colour(bgSelected))
		}
		if it.icon != nil {
			g.Image(currentSymbol(l.app, it.icon), drawnIconX, row.y+(row.h-16)/2)
		}
		title := fgRow
		if selected {
			title = selectedFG
		}
		g.SetColor(colour(title))
		if it.detail != "" {
			g.SetFont(gowt.Type.Body)
			g.Text(it.title, drawnTextX, row.y+3)
			g.SetFont(gowt.Type.Caption)
			g.SetColor(colour(fgDim))
			g.Text(it.detail, drawnTextX, row.y+19)
			continue
		}
		g.SetFont(gowt.Type.Body)
		text := it.title
		if l.closable {
			text = fitText(g, text, w-drawnEndRoom-drawnTextX)
		}
		_, th := g.TextSize(text)
		g.Text(text, drawnTextX, row.y+(row.h-th)/2)
		if l.closable {
			switch {
			case hovered:
				cx, cy := w-15, row.y+row.h/2
				g.SetColor(colour(fgDim))
				g.Line(cx-4, cy-4, cx+4, cy+4)
				g.Line(cx-4, cy+4, cx+4, cy-4)
			case selected:
				cy := row.y + row.h/2
				g.SetFill(colour(fgDim))
				for i := 0; i < 3; i++ {
					g.FillOval(w-22+i*4, cy-1, 2, 2)
				}
			}
		}
	}
}

// fitText cuts s with an ellipsis so a title never runs under the close X or the More dots.
func fitText(g *gowt.GC, s string, maxW int) string {
	if w, _ := g.TextSize(s); w <= maxW {
		return s
	}
	r := []rune(s)
	for n := len(r) - 1; n > 0; n-- {
		t := string(r[:n]) + "..."
		if w, _ := g.TextSize(t); w <= maxW {
			return t
		}
	}
	return "..."
}

func (l *drawnList) move(m gowt.Mouse) {
	if !l.closable {
		return
	}
	hover := -1
	for _, row := range l.layoutRows() {
		if row.separator || m.Y < row.y || m.Y >= row.y+row.h {
			continue
		}
		it := l.items[row.item]
		if !it.group && it.title != "" && it.value >= 0 {
			hover = row.item
		}
		break
	}
	if hover != l.hover {
		l.hover = hover
		l.redraw()
	}
}

// keyMove sends the arrow keys the canvas gets while it holds the focus to move.
type keyMove struct{ l *drawnList }

func (k keyMove) HandleEvent(e *swt.Event) {
	delta := 0
	switch e.KeyCode {
	case swt.ARROW_DOWN:
		delta = 1
	case swt.ARROW_UP:
		delta = -1
	default:
		return
	}
	if k.l.step(delta) {
		e.Doit = false
	}
}

func (l *drawnList) leave(gowt.Mouse) {
	if l.hover == -1 {
		return
	}
	l.hover = -1
	l.redraw()
}

func (l *drawnList) click(m gowt.Mouse) {
	for _, row := range l.layoutRows() {
		if row.separator || m.Y < row.y || m.Y >= row.y+row.h {
			continue
		}
		it := l.items[row.item]
		if it.group || it.title == "" || it.value < 0 {
			return
		}
		hovered := row.item == l.hover
		if l.closable && hovered && l.canvas != nil {
			if w, _ := l.canvas.Size(); m.X >= w-26 {
				l.close(row, it.value)
				return
			}
		}
		if l.canvas != nil {
			l.canvas.Focus()
		}
		l.Select(it.value)
		if l.onSelect != nil {
			l.onSelect(it.value)
		}
		return
	}
}

func (l *drawnList) close(row drawnRow, value int) {
	if !l.closable {
		return
	}
	rows := 0
	for _, it := range l.items {
		if !it.group && it.title != "" && it.value >= 0 {
			rows++
		}
	}
	if rows <= 1 {
		return
	}
	neighbour := -1
	for i := row.item + 1; i < len(l.items); i++ {
		if it := l.items[i]; !it.group && it.title != "" && it.value >= 0 {
			neighbour = it.value
			break
		}
	}
	if neighbour < 0 {
		for i := row.item - 1; i >= 0; i-- {
			if it := l.items[i]; !it.group && it.title != "" && it.value >= 0 {
				neighbour = it.value
				break
			}
		}
	}
	l.items = append(l.items[:row.item], l.items[row.item+1:]...)
	l.hover = -1
	if l.selected == value {
		l.selected = neighbour
	}
	l.redraw()
	if l.onClose != nil {
		l.onClose(value)
	}
}

const (
	pathMonitor  = `<rect width="20" height="14" x="2" y="3" rx="2"/><line x1="8" x2="16" y1="21" y2="21"/><line x1="12" x2="12" y1="17" y2="21"/>`
	pathSettings = `<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.09a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>`
	pathGlobe    = `<circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/>`
	pathLayout   = `<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>`
	pathPalette  = `<circle cx="13.5" cy="6.5" r=".5"/><circle cx="17.5" cy="10.5" r=".5"/><circle cx="8.5" cy="7.5" r=".5"/><circle cx="6.5" cy="12.5" r=".5"/><path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.93 0 1.5-.67 1.5-1.5 0-.4-.15-.72-.4-.97-.24-.26-.38-.6-.38-.98 0-.83.67-1.5 1.5-1.5H16c3.1 0 5.5-2.4 5.5-5.5C21.5 6 17.5 2 12 2z"/>`
	pathTerminal = `<path d="m4 17 6-6-6-6"/><line x1="12" x2="20" y1="19" y2="19"/>`
	pathCommand  = `<path d="M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3"/>`
	pathBlocks   = `<rect width="7" height="7" x="14" y="3" rx="1"/><path d="M10 21V8a1 1 0 0 0-1-1H4a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-5a1 1 0 0 0-1-1H3"/>`
	pathBell     = `<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>`
	pathWrench   = `<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>`
	pathDot      = `<circle cx="12" cy="12" r="7" fill="currentColor" stroke="none"/>`
	pathKeyboard = `<rect width="20" height="16" x="2" y="4" rx="2"/><path d="M6 8h.01"/><path d="M10 8h.01"/><path d="M14 8h.01"/><path d="M18 8h.01"/><path d="M8 12h.01"/><path d="M12 12h.01"/><path d="M16 12h.01"/><path d="M7 16h10"/>`
	pathSidebar  = `<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/>`
	pathSplit    = `<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M12 3v18"/>`
	pathPlus     = `<path d="M5 12h14"/><path d="M12 5v14"/>`
	pathClose    = `<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`
	pathSend     = `<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>`
	pathAttach   = `<path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>`

	glyphDiamond = `<path d="M12 4l7 8-7 8-7-8z" fill="#fff"/>`
	glyphSpark   = `<path d="M12 3.5l2.3 6.2 6.2 2.3-6.2 2.3-2.3 6.2-2.3-6.2-6.2-2.3 6.2-2.3z" fill="#fff"/>`
	glyphGrid    = `<rect x="6" y="6" width="5" height="5" rx="1" fill="#fff"/><rect x="13" y="6" width="5" height="5" rx="1" fill="#fff"/><rect x="6" y="13" width="5" height="5" rx="1" fill="#fff"/><rect x="13" y="13" width="5" height="5" rx="1" fill="#fff"/>`
	glyphBars    = `<path d="M6 18h3V7H6zm4.5 0h3V11h-3zm4.5 0h3V4h-3z" fill="#fff"/>`
)

func faint() gowt.Option { return gowt.Foreground(fgDim) }

func label(text string, opts ...gowt.Option) gowt.Option {
	return gowt.Label(text, append([]gowt.Option{gowt.Foreground(fg)}, opts...)...)
}

func heading(text string) gowt.Option {
	return label(text, gowt.Typeface(gowt.Type.BodyStrong))
}

func card(children ...gowt.Option) gowt.Option {
	return gowt.Section(append([]gowt.Option{gowt.GrowX(), gowt.Background(bgCard)}, children...)...)
}

type symbolSource struct {
	path string
	size int
	col  gowt.Adaptive
}

var symbolSources = map[*gowt.Image]symbolSource{}

func symbol(app *gowt.App, path string, size int, c gowt.Adaptive) *gowt.Image {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">%s</svg>`, path)
	img := icons.FromSVG(app, []byte(svg), size, colour(c))
	symbolSources[img] = symbolSource{path: path, size: size, col: c}
	return img
}

func currentSymbol(app *gowt.App, img *gowt.Image) *gowt.Image {
	src, ok := symbolSources[img]
	if !ok {
		return img
	}
	return symbol(app, src.path, src.size, src.col)
}

// themedSymbol replaces an SVG image when the appearance changes.
func themedSymbol(app *gowt.App, path string, size int, c gowt.Adaptive, set func(*gowt.Image)) {
	update := func() { set(symbol(app, path, size, c)) }
	update()
	app.OnThemeChange(func(bool) { update() })
}

func themedIcon(app *gowt.App, name string, size int, set func(*gowt.Image)) {
	update := func() { set(icons.Get(app, name, icons.Size(size))) }
	update()
	app.OnThemeChange(func(bool) { update() })
}

func appTile(app *gowt.App, bg gowt.RGB, glyph string, size int, selected bool) gowt.Option {
	stroke := "none"
	if selected {
		stroke = "#0A84FF"
	}
	doc := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M6 1h12a5 5 0 0 1 5 5v12a5 5 0 0 1-5 5H6a5 5 0 0 1-5-5V6a5 5 0 0 1 5-5z" fill="#%02x%02x%02x" stroke="%s" stroke-width="1.5"/>%s</svg>`,
		bg.R, bg.G, bg.B, stroke, glyph)
	return gowt.Label("", gowt.Size(size, size), gowt.Place(gowt.AlignStart, gowt.AlignCenter),
		gowt.Init(func(l *gowt.LabelView) {
			l.SetImage(icons.FromSVG(app, []byte(doc), size, gowt.RGB{}))
		}))
}

func iconButton(app *gowt.App, path string, box, size int, c gowt.Adaptive, tip string, onClick func(), extra ...gowt.Option) gowt.Option {
	opts := []gowt.Option{gowt.Borderless(), gowt.Size(box, box),
		gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Tooltip(tip), gowt.OnClick(onClick),
		gowt.Init(func(b *gowt.ButtonView) {
			themedSymbol(app, path, size, c, b.SetImage)
		})}
	return gowt.Button("", append(opts, extra...)...)
}

func infoButton(app *gowt.App) gowt.Option {
	return gowt.Button("", gowt.Borderless(), gowt.Size(gowt.Scaled(24), gowt.Scaled(24)),
		gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Tooltip("More info"),
		gowt.Init(func(b *gowt.ButtonView) {
			themedIcon(app, "info", 14, b.SetImage)
		}))
}

func setting(title, detail string, control gowt.Option) gowt.Option {
	labels := []gowt.Option{gowt.Gap(1), gowt.GrowX(), gowt.Place(gowt.AlignFill, gowt.AlignCenter),
		label(title)}
	if detail != "" {
		labels = append(labels, label(detail, faint(), gowt.Typeface(gowt.Type.Caption),
			gowt.Wrap(), gowt.GrowX()))
	}
	return gowt.HBox(gowt.GrowX(), gowt.Background(bgCard), gowt.Gap(gowt.Spacing.MD),
		gowt.VBox(labels...),
		gowt.VBox(gowt.Padding(0), gowt.Place(gowt.AlignEnd, gowt.AlignCenter), control))
}

func labeled(text string, control gowt.Option) gowt.Option {
	return gowt.VBox(gowt.Gap(1), gowt.GrowX(), gowt.Background(bgCard),
		label(text, faint(), gowt.Typeface(gowt.Type.Caption)),
		control)
}

func page(children ...gowt.Option) gowt.Option {
	return gowt.Scroll(gowt.Grow(), gowt.Background(bgApp),
		gowt.VBox(append([]gowt.Option{
			gowt.GrowX(), gowt.Padding(28), gowt.Gap(20), gowt.Background(bgApp), gowt.Foreground(fg),
		}, children...)...))
}

func progressBar(app *gowt.App, value int) gowt.Option {
	return gowt.Canvas(func(g *gowt.GC) {
		w, h := g.Size()
		g.SetFill(colour(bgField))
		g.FillRect(0, (h-6)/2, w, 6)
		g.SetFill(colour(teal))
		g.FillRect(0, (h-6)/2, w*value/100, 6)
	}, gowt.Size(gowt.Scaled(120), gowt.Scaled(6)), gowt.Place(gowt.AlignStart, gowt.AlignCenter),
		gowt.Init(func(c *gowt.CanvasView) { app.OnThemeChange(func(bool) { c.Redraw() }) }))
}

func pill(text string, w, h int, bg, fg gowt.RGB) gowt.Option {
	return gowt.Canvas(func(g *gowt.GC) {
		width, height := g.Size()
		g.SetFill(bg)
		g.FillOval(0, 0, height, height)
		g.FillOval(width-height, 0, height, height)
		g.FillRect(height/2, 0, width-height, height)
		g.SetColor(fg)
		tw, th := g.TextSize(text)
		g.Text(text, (width-tw)/2, (height-th)/2)
	}, gowt.Size(w, h), gowt.Typeface(gowt.Type.Caption))
}

func fillRound(g *gowt.GC, x, y, w, h, r int, c gowt.Adaptive) {
	g.SetFill(colour(c))
	g.FillOval(x, y, 2*r, 2*r)
	g.FillOval(x+w-2*r, y, 2*r, 2*r)
	g.FillOval(x, y+h-2*r, 2*r, 2*r)
	g.FillOval(x+w-2*r, y+h-2*r, 2*r, 2*r)
	g.FillRect(x+r, y, w-2*r, h)
	g.FillRect(x, y+r, r, h-2*r)
	g.FillRect(x+w-r, y+r, r, h-2*r)
}

func semibold(f gowt.Font) gowt.Font {
	f.Weight = gowt.Semibold
	return f
}
