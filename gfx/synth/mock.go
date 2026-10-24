package synth

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/haiodo/gowt/gfx"
)

// Mock screens: 1920 x 1080 application windows made of an app rail, a sidebar of icon and label
// rows, a header with tabs, and a body that is a form, a list of rows or a board of cards, with
// text on nearly every node. The node counts match what the real design screens of a task tracker
// give (about 150 to 540 nodes and 40 to 150 text lines per screen, a third of the nodes vector
// icons); the colors and strings are made up.
const (
	ScreenW = 1920.0
	ScreenH = 1080.0
	// Gap between neighbouring screens on the grid.
	ScreenGap = 160.0
)

// Screen is one mockup screen: a subtree of the scene.
type Screen struct {
	Root          *gfx.Node
	X, Y          float64
	Kind          string
	Nodes, Texts  int
	Paths, Frames int
}

// Mock is a scene of mockup screens on a grid.
type Mock struct {
	Scene      *gfx.Scene
	Screens    []*Screen
	Cols, Rows int
}

// Grid returns the grid for n screens of w x h with the gap between them, wider than tall.
func Grid(n int, w, h float64) (cols, rows int) {
	cols = int(math.Ceil(math.Sqrt(float64(n) * 1.7 * (h + ScreenGap) / (w + ScreenGap) * 1.6)))
	cols = max(1, min(cols, n))
	return cols, (n + cols - 1) / cols
}

// NewMock builds n screens on a grid. Kinds alternate between form, list and board.
func NewMock(n int, seed int64) *Mock {
	cols, rows := Grid(n, ScreenW, ScreenH)
	px, py := ScreenW+ScreenGap, ScreenH+ScreenGap
	m := &Mock{Cols: cols, Rows: rows}
	m.Scene = gfx.NewScene(gfx.R(-ScreenGap, -ScreenGap, float64(cols)*px+ScreenGap, float64(rows)*py+ScreenGap), 256)
	ids := 0
	m.Scene.Update(func(tx *gfx.Tx) {
		for i := 0; i < n; i++ {
			s := BuildScreen(tx, nil, &ids, rand.New(rand.NewSource(seed+int64(i))), i%3)
			s.X, s.Y = float64(i%cols)*px, float64(i/cols)*py
			tx.Move(s.Root, s.X, s.Y)
			m.Screens = append(m.Screens, s)
		}
	})
	m.Scene.Commit()
	return m
}

var (
	cInk     = gfx.Color{R: 0.10, G: 0.12, B: 0.17, A: 1}
	cSecond  = gfx.Color{R: 0.34, G: 0.38, B: 0.46, A: 1}
	cMuted   = gfx.Color{R: 0.54, G: 0.58, B: 0.65, A: 1}
	cLine    = gfx.Color{R: 0.90, G: 0.91, B: 0.93, A: 1}
	cPanel   = gfx.Color{R: 1, G: 1, B: 1, A: 1}
	cRail    = gfx.Color{R: 0.97, G: 0.975, B: 0.98, A: 1}
	cField   = gfx.Color{R: 0.95, G: 0.96, B: 0.975, A: 1}
	cAccent  = gfx.Color{R: 0.05, G: 0.52, B: 0.55, A: 1}
	cAccent2 = gfx.Color{R: 0.90, G: 0.93, B: 0.97, A: 1}
	cWarm    = gfx.Color{R: 0.86, G: 0.42, B: 0.16, A: 1}
	avatars  = []gfx.Color{{R: 0.30, G: 0.55, B: 0.90, A: 1}, {R: 0.55, G: 0.40, B: 0.85, A: 1}, {R: 0.25, G: 0.65, B: 0.45, A: 1}, {R: 0.85, G: 0.45, B: 0.50, A: 1}}
)

var (
	words = []string{"project", "board", "release", "review", "sprint", "design", "backlog", "member", "access", "status",
		"workflow", "archive", "template", "owner", "label", "estimate", "report", "channel", "invite", "settings"}
	fontReg  = "HelveticaNeue"
	fontMed  = "HelveticaNeue-Medium"
	fontBold = "HelveticaNeue-Bold"
)

type mockB struct {
	tx   *gfx.Tx
	ids  *int
	rng  *rand.Rand
	s    *Screen
	meas gfx.TextMeasurer
}

func (b *mockB) node(parent *gfx.Node, x, y float64, pic *gfx.Picture) *gfx.Node {
	*b.ids++
	n := gfx.NewNode(*b.ids)
	b.s.Nodes++
	b.tx.Add(parent, n)
	b.tx.Move(n, x, y)
	if pic != nil {
		b.tx.SetPicture(n, pic)
	}
	return n
}

func rr(w, h, r float64) *gfx.Path {
	p := &gfx.Path{}
	p.RoundRect(0, 0, w, h, r)
	return p
}

// box is a rounded frame with an optional border and shadow; its Lod is the flat fill.
func (b *mockB) box(parent *gfx.Node, x, y, w, h, r float64, fill gfx.Color, border *gfx.Color, sh *gfx.Shadow) *gfx.Node {
	rec := gfx.NewRecorderWith(b.meas)
	if sh != nil {
		rec.FillShadowRRect(0, 0, w, h, r, fill, *sh)
	} else if fill.A > 0 {
		rec.FillPath(rr(w, h, r), fill)
	}
	if border != nil {
		in := &gfx.Path{}
		in.RoundRect(0.5, 0.5, w-1, h-1, math.Max(r-0.5, 0))
		rec.StrokePath(in, gfx.StrokeStyle{Width: 1, MiterLimit: 10}, *border)
	}
	pic := rec.Finish()
	lr := gfx.NewRecorderWith(b.meas)
	lr.FillPath(rr(w, h, 0), fill)
	pic.Lod, pic.LodBelow = lr.Finish(), lodBelow
	b.s.Frames++
	return b.node(parent, x, y, pic)
}

func (b *mockB) text(parent *gfx.Node, x, y float64, s string, size float64, font string, c gfx.Color) *gfx.Node {
	rec := gfx.NewRecorderWith(b.meas)
	rec.Text(s, 0, size*1.0, gfx.Font{Name: font, Size: size}, c)
	b.s.Texts++
	return b.node(parent, x, y, rec.Finish())
}

// icon is a 16 x 16 stroked vector of one of a few shapes.
func (b *mockB) icon(parent *gfx.Node, x, y float64, kind int, c gfx.Color) *gfx.Node {
	p := &gfx.Path{}
	switch kind % 6 {
	case 0: // clock
		p.Ellipse(8, 8, 6.5, 6.5)
		p.MoveTo(8, 4.5)
		p.LineTo(8, 8)
		p.LineTo(10.5, 9.5)
	case 1: // check box
		p.RoundRect(2, 2, 12, 12, 3)
		p.MoveTo(5, 8.2)
		p.LineTo(7.2, 10.4)
		p.LineTo(11, 5.8)
	case 2: // flag
		p.MoveTo(4, 14)
		p.LineTo(4, 2)
		p.LineTo(12, 2)
		p.LineTo(12, 8)
		p.LineTo(4, 8)
	case 3: // star
		for i := 0; i < 5; i++ {
			a := -math.Pi/2 + float64(i)*4*math.Pi/5
			px, py := 8+6.5*math.Cos(a), 8+6.5*math.Sin(a)
			if i == 0 {
				p.MoveTo(px, py)
			} else {
				p.LineTo(px, py)
			}
		}
		p.Close()
	case 4: // person
		p.Ellipse(8, 5, 2.8, 2.8)
		p.MoveTo(2.5, 14)
		p.CubicTo(2.5, 9, 13.5, 9, 13.5, 14)
	default: // bars
		p.MoveTo(3, 3)
		p.LineTo(3, 13)
		p.MoveTo(8, 3)
		p.LineTo(8, 9)
		p.MoveTo(13, 3)
		p.LineTo(13, 11)
	}
	rec := gfx.NewRecorderWith(b.meas)
	rec.StrokePath(p, gfx.StrokeStyle{Width: 1.3, Cap: gfx.CapRound, Join: gfx.JoinRound, MiterLimit: 10}, c)
	b.s.Paths++
	return b.node(parent, x, y, rec.Finish())
}

func (b *mockB) circle(parent *gfx.Node, x, y, d float64, fill gfx.Color) *gfx.Node {
	rec := gfx.NewRecorderWith(b.meas)
	p := &gfx.Path{}
	p.Ellipse(d/2, d/2, d/2, d/2)
	rec.FillPath(p, fill)
	b.s.Frames++
	return b.node(parent, x, y, rec.Finish())
}

func (b *mockB) avatar(parent *gfx.Node, x, y, d float64) {
	c := avatars[b.rng.Intn(len(avatars))]
	n := b.circle(parent, x, y, d, c)
	b.text(n, d/2-5, d/2-6, string(rune('A'+b.rng.Intn(26)))+string(rune('A'+b.rng.Intn(26))), 9, fontMed, gfx.Color{R: 1, G: 1, B: 1, A: 1})
}

func (b *mockB) phrase(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += " "
		}
		s += words[b.rng.Intn(len(words))]
	}
	return s
}

// paragraph puts wrapped lines of text; the line count is fixed, the breaks need no measuring.
func (b *mockB) paragraph(parent *gfx.Node, x, y float64, lines int, size float64, c gfx.Color) {
	for i := 0; i < lines; i++ {
		b.text(parent, x, y+float64(i)*size*1.35, b.phrase(7+b.rng.Intn(3)), size, fontReg, c)
	}
}

func (b *mockB) button(parent *gfx.Node, x, y, w, h float64, label string, primary bool) {
	fill, ink := cField, cInk
	var border *gfx.Color
	if primary {
		fill, ink = cAccent, gfx.Color{R: 1, G: 1, B: 1, A: 1}
	} else {
		border = &cLine
	}
	n := b.box(parent, x, y, w, h, 6, fill, border, nil)
	b.icon(n, 10, h/2-8, b.rng.Intn(6), ink)
	b.text(n, 32, h/2-8, label, 13, fontMed, ink)
}

// BuildScreen adds one screen under parent (the scene root when nil) with ids counted from *ids,
// of kind 0 (form), 1 (list) or 2 (board). The root sits at the origin.
func BuildScreen(tx *gfx.Tx, parent *gfx.Node, ids *int, rng *rand.Rand, kind int) *Screen {
	s := &Screen{Kind: []string{"form", "list", "board"}[kind%3]}
	b := &mockB{tx: tx, ids: ids, rng: rng, s: s, meas: gfx.DefaultMeasurer}
	root := b.box(parent, 0, 0, ScreenW, ScreenH, 0, cPanel, nil, nil)
	s.Root = root
	tx.ClipRect(root, gfx.R(0, 0, ScreenW, ScreenH))

	// top bar: tab, search and status texts
	bar := b.box(root, 0, 0, ScreenW, 36, 0, cRail, nil, nil)
	b.icon(bar, 14, 10, 1, cSecond)
	b.text(bar, 36, 10, b.phrase(2), 12, fontReg, cSecond)
	for i := 0; i < 5; i++ {
		b.icon(bar, ScreenW-260+float64(i)*28, 10, i, cSecond)
	}
	b.text(bar, ScreenW-100, 10, "16:56", 12, fontReg, cSecond)

	// app rail: 22 icons
	rail := b.box(root, 0, 36, 68, ScreenH-36, 0, cRail, nil, nil)
	b.avatar(rail, 18, 14, 32)
	for i := 0; i < 22; i++ {
		b.icon(rail, 26, 70+float64(i)*40, i, cMuted)
	}

	// sidebar: title, primary button, 20 rows of icon and label, two project groups
	side := b.box(root, 68, 36, 280, ScreenH-36, 0, cRail, nil, nil)
	b.text(side, 20, 18, "Tracker", 20, fontBold, cInk)
	b.button(side, 14, 56, 252, 32, "New item", true)
	for i := 0; i < 20; i++ {
		y := 106 + float64(i)*32
		if i == 2 {
			row := b.box(side, 12, y-6, 256, 30, 6, cAccent2, nil, nil)
			_ = row
		}
		b.icon(side, 24, y, i, cInk)
		b.text(side, 50, y, b.phrase(1+i%2), 13, fontReg, cInk)
	}

	// main panel with a header, a tab strip and the body
	panel := b.box(root, 348, 36, ScreenW-348-60, ScreenH-36, 8, cPanel, &cLine, nil)
	tx.ClipRRect(panel, gfx.R(0, 0, ScreenW-348-60, ScreenH-36), 8)
	b.icon(panel, 24, 22, 2, cSecond)
	b.text(panel, 48, 20, b.phrase(2), 14, fontReg, cInk)
	b.text(panel, 160, 18, b.phrase(2), 16, fontBold, cInk)
	for i := 0; i < 6; i++ {
		b.icon(panel, ScreenW-348-60-40-float64(i)*0, 22, 0, cMuted)
		break
	}
	b.box(panel, 0, 56, ScreenW-348-60, 1, 0, cLine, nil, nil)
	for i := 0; i < 5; i++ {
		y := 76 + float64(i)*34
		sel := i == 0
		var t *gfx.Node
		if sel {
			t = b.box(panel, 12, y, 220, 32, 6, cAccent2, nil, nil)
		} else {
			t = b.box(panel, 12, y, 220, 32, 6, gfx.Color{}, nil, nil)
		}
		b.icon(t, 10, 8, i, cInk)
		b.text(t, 34, 8, b.phrase(1+i%2), 13, fontReg, cInk)
	}
	switch kind % 3 {
	case 0:
		b.form(panel)
	case 1:
		b.list(panel)
	default:
		b.board(panel)
	}

	// right sidebar
	for i := 0; i < 5; i++ {
		b.icon(root, ScreenW-40, 60+float64(i)*44, i, cMuted)
	}
	return s
}

func (b *mockB) form(panel *gfx.Node) {
	x0 := 260.0
	b.text(panel, x0, 80, b.phrase(1), 16, fontBold, cInk)
	b.text(panel, x0, 106, b.phrase(4), 12, fontReg, cMuted)
	y := 150.0
	for i := 0; i < 4; i++ {
		b.text(panel, x0, y+8, b.phrase(2), 13, fontReg, cSecond)
		if i < 2 {
			b.text(panel, x0, y+28, b.phrase(5), 11, fontReg, cMuted)
		}
		w := 420.0
		if i == 1 {
			w = 160
		}
		f := b.box(panel, x0+270, y, w, 32, 6, cField, nil, nil)
		b.text(f, 10, 8, b.phrase(2), 13, fontReg, cInk)
		y += 50
	}
	b.text(panel, x0, y+8, b.phrase(1), 13, fontReg, cSecond)
	area := b.box(panel, x0+270, y-6, 420, 64, 6, cField, nil, nil)
	b.paragraph(area, 10, 8, 3, 13, cInk)
	y += 90
	b.text(panel, x0, y+6, b.phrase(1), 13, fontReg, cSecond)
	for i := 0; i < 5; i++ {
		c := b.box(panel, x0+270+float64(i)*38, y, 32, 32, 6, cPanel, &cLine, nil)
		b.icon(c, 8, 8, i, cInk)
	}
	b.button(panel, x0+270, y+80, 120, 32, "Save", true)
	b.button(panel, x0+400, y+80, 120, 32, "Cancel", false)
	// a dialog card with a shadow and steps, as in the creation flow
	d := b.box(panel, 640, 330, 520, 420, 12, cPanel, &cLine, &gfx.Shadow{DY: 12, Blur: 24, Color: gfx.Color{A: 0.18}})
	b.text(d, 28, 24, b.phrase(2), 18, fontMed, cInk)
	for i := 0; i < 4; i++ {
		b.circle(d, 28, 90+float64(i)*44, 14, cAccent2)
		b.text(d, 54, 88+float64(i)*44, fmt.Sprintf("%d. %s", i+1, b.phrase(2)), 12, fontReg, cInk)
	}
	for i := 0; i < 4; i++ {
		b.text(d, 240, 90+float64(i)*54, b.phrase(2), 12, fontReg, cSecond)
		f := b.box(d, 240, 108+float64(i)*54, 250, 28, 6, cField, nil, nil)
		b.text(f, 8, 6, b.phrase(2), 12, fontReg, cInk)
	}
	b.button(d, 380, 360, 110, 32, "Next", true)
}

func (b *mockB) list(panel *gfx.Node) {
	w := ScreenW - 348 - 60
	for g := 0; g < 3; g++ {
		y0 := 90 + float64(g)*270
		hd := b.box(panel, 252, y0, w-264, 36, 4, cAccent2, nil, nil)
		b.icon(hd, 12, 10, g, cInk)
		b.text(hd, 38, 10, b.phrase(1), 13, fontMed, cInk)
		b.text(hd, w-340, 10, fmt.Sprintf("%dh", 4+g*3), 13, fontReg, cSecond)
		for r := 0; r < 4; r++ {
			y := y0 + 44 + float64(r)*44
			b.box(panel, 252, y, 14, 14, 3, cPanel, &cLine, nil)
			b.text(panel, 280, y, fmt.Sprintf("ITM-%d", g*5+r+1), 12, fontReg, cMuted)
			b.icon(panel, 352, y-1, r+g, cWarm)
			b.text(panel, 380, y, b.phrase(3+r%3), 13, fontReg, cInk)
			for c := 0; c < 3; c++ {
				chip := b.box(panel, w-720+float64(c)*100, y-4, 90, 22, 11, cField, &cLine, nil)
				b.text(chip, 10, 4, words[(g+r+c)%len(words)], 11, fontReg, cSecond)
			}
			b.text(panel, w-390, y, "2 days ago", 12, fontReg, cMuted)
			b.text(panel, w-300, y, "Oct 21", 12, fontReg, cMuted)
			b.avatar(panel, w-240, y-6, 24)
		}
	}
}

func (b *mockB) board(panel *gfx.Node) {
	w := (ScreenW - 348 - 60 - 300) / 4
	for c := 0; c < 4; c++ {
		x := 252 + float64(c)*(w+12)
		col := b.box(panel, x, 90, w, 900, 8, cField, nil, nil)
		b.icon(col, 12, 14, c, cAccent)
		b.text(col, 36, 14, b.phrase(1), 13, fontMed, cInk)
		b.text(col, w-30, 14, fmt.Sprint(3+c), 12, fontReg, cMuted)
		for r := 0; r < 5; r++ {
			y := 50 + float64(r)*170
			card := b.box(col, 10, y, w-20, 156, 8, cPanel, &cLine, &gfx.Shadow{DY: 1, Blur: 3, Color: gfx.Color{A: 0.10}})
			b.text(card, 12, 12, fmt.Sprintf("ITM-%d", c*5+r+10), 11, fontReg, cMuted)
			b.paragraph(card, 12, 34, 1+r%2, 13, cInk)
			for k := 0; k < 2; k++ {
				chip := b.box(card, 12+float64(k)*84, 100, 76, 22, 11, cField, &cLine, nil)
				b.text(chip, 10, 4, words[(c+r+k)%len(words)], 11, fontReg, cSecond)
			}
			b.icon(card, 12, 130, r, cMuted)
			b.text(card, 36, 130, "Oct 21", 11, fontReg, cMuted)
			b.avatar(card, w-56, 122, 24)
		}
	}
}
