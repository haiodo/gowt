package look

import (
	"fmt"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/svg"
	"github.com/haiodo/gowt/swt"
)

// The toolbar pills follow the DeepSeek Harness style: one rounded capsule per item or group with a
// fill that is almost invisible at rest and darkens under the pointer and when pressed, the segments
// of a group split by a hairline. The native ToolBar cannot give these fill states, so the pills are
// drawn on canvases; each segment is its own canvas, which keeps its ID, tooltip and enabled state.

type pillState int

const (
	pillRest pillState = iota
	pillHover
	pillPressed
)

// pillSeparatorAlpha is the opacity of the hairline between the segments of a group.
const pillSeparatorAlpha = 0x1F

// pillInk is the fill colour over a bar of bg: black on a light bar, white on a dark one, so one
// alpha scheme serves both themes.
func pillInk(bg swt.RGB) swt.RGB {
	if luma(bg) > 128 {
		return swt.RGB{}
	}
	return swt.RGB{Red: 255, Green: 255, Blue: 255}
}

// pillAlpha is the opacity of the pill fill: almost invisible at rest, clearly darker under the
// pointer, darkest pressed.
func pillAlpha(s pillState) int32 {
	switch s {
	case pillHover:
		return 0x14
	case pillPressed:
		return 0x1F
	default:
		return 0x0A
	}
}

// pillRadius is the corner radius of every pill of height h points: about 11, capped at a half
// circle, so all buttons of one size share one radius.
func pillRadius(h int32) int32 { return min(h/2, int32(gowt.Scaled(11))) }

// pillSeg is one segment of a toolbar pill: the button of a single item or one cell of a Group,
// drawn on its own canvas.
type pillSeg struct {
	it      ToolbarItem
	index   int // within the pill
	segs    int
	size    int // icon size in points
	canvas  *gowt.CanvasView
	image   *swt.Image
	fill    *swt.Color // the ink of the fills and the hairline, set by the bar restyle
	enabled bool
	hover   bool
	pressed bool
	focused bool
}

// pillItem adds it, or one segment per item of its Group, to p as pill buttons: one capsule, the
// segments of a group split by a hairline. icons collects the restyle closures, as for toolItem.
func pillItem(p *gowt.Panel, it ToolbarItem, size int, icons *[]func(on, off string) []*swt.Image) gowt.Widget {
	segs := it.Group
	if len(segs) == 0 {
		segs = []ToolbarItem{it}
	}
	if len(segs) == 1 {
		return newPillSeg(p, segs[0], 0, 1, size, icons).canvas
	}
	var group *gowt.Panel
	p.Add(gowt.HBox(gowt.Gap(0), gowt.Padding(0), gowt.Place(gowt.AlignStart, gowt.AlignCenter), gowt.Ref(&group)))
	for i, g := range segs {
		newPillSeg(group, g, i, len(segs), size, icons)
	}
	return group
}

// newPillSeg adds the canvas of one segment to p and returns the segment.
func newPillSeg(p *gowt.Panel, it ToolbarItem, index, segs, size int, icons *[]func(on, off string) []*swt.Image) *pillSeg {
	s := &pillSeg{it: it, index: index, segs: segs, size: size, enabled: it.Enabled == nil || it.Enabled.Peek()}
	opts := []gowt.Option{
		gowt.Size(gowt.Spacing.ControlHeight, gowt.Spacing.ControlHeight),
		gowt.Place(gowt.AlignStart, gowt.AlignCenter),
		gowt.Tooltip(it.Title),
		gowt.AccessibleLabel(it.Title),
		gowt.Ref(&s.canvas),
	}
	if it.ID != "" {
		opts = append(opts, gowt.ID(it.ID))
	}
	p.Add(gowt.Canvas(s.paint, opts...))
	s.fill = swt.NewColorRgb(&swt.RGB{})
	s.canvas.OnDispose(func() {
		s.fill.Dispose()
		s.fill = nil
	})
	if !s.enabled {
		s.canvas.Unwrap().SetEnabled(false)
	}
	pillTabStop(s.canvas.Unwrap())
	s.wire()
	if it.Enabled != nil {
		gowt.Watch(s.canvas, it.Enabled, func(on bool) {
			s.enabled = on
			s.canvas.Unwrap().SetEnabled(on)
			s.canvas.Redraw()
		})
	}
	shape, symbol := barSymbols[it.Symbol]
	var data string
	switch {
	case it.Icon != nil:
		s.image = it.Icon.Unwrap()
	case len(it.IconSVG) > 0:
		data = string(it.IconSVG)
	case symbol:
		data = fmt.Sprintf(barSVG, shape)
	}
	*icons = append(*icons, func(on, _ string) []*swt.Image {
		// The icon colour is the bar text colour; its luma picks the fill ink, as pillInk would.
		s.setInk(pillInk(*rgbOfHex(on)))
		if data == "" {
			return nil
		}
		img := swt.NewImageDeviceImageDataProvider(s.canvas.Unwrap().GetDisplay(), svg.NewImageDataProvider([]byte(data), size, size, on))
		s.image = img
		s.canvas.Redraw()
		return []*swt.Image{img}
	})
	return s
}

// rgbOfHex parses the "#rrggbb" of hexRGB.
func rgbOfHex(s string) *swt.RGB {
	var c swt.RGB
	fmt.Sscanf(s, "#%02x%02x%02x", &c.Red, &c.Green, &c.Blue)
	return &c
}

// setInk changes the fill colour and repaints; the bar restyle calls it with the text colour.
func (s *pillSeg) setInk(c swt.RGB) {
	if s.fill != nil {
		s.fill.Dispose()
	}
	s.fill = swt.NewColorRgb(&c)
	if s.canvas != nil && !s.canvas.Unwrap().IsDisposed() {
		s.canvas.Redraw()
	}
}

func (s *pillSeg) wire() {
	c := s.canvas
	c.OnMouseDown(func(gowt.Mouse) {
		if s.enabled {
			c.Unwrap().SetFocus()
			s.pressed = true
			c.Redraw()
		}
	})
	c.OnMouseUp(func(gowt.Mouse) {
		if !s.pressed {
			return
		}
		s.pressed = false
		c.Redraw()
		if s.enabled && s.it.OnClick != nil {
			s.it.OnClick()
		}
	})
	c.Unwrap().AddMouseTrackListener(swt.MouseTrackListenerMouseEnterAdapter(func(*swt.MouseEvent) {
		s.hover = true
		c.Redraw()
	}))
	c.Unwrap().AddMouseTrackListener(swt.MouseTrackListenerMouseExitAdapter(func(*swt.MouseEvent) {
		s.hover, s.pressed = false, false
		c.Redraw()
	}))
	// Keyboard parity with the Button and ToolItem the pill replaces: the canvas takes the focus (a
	// Tab stop on Windows, a tab group elsewhere), Space or Enter presses it, a ring shows the focus.
	key := func(down bool) func(*swt.KeyEvent) {
		return func(e *swt.KeyEvent) {
			if e.KeyCode != int32(swt.SPACE) && e.KeyCode != int32(swt.CR) && e.KeyCode != swt.KEYPAD_CR {
				return
			}
			if down {
				if s.enabled && !s.pressed {
					s.pressed = true
					c.Redraw()
				}
				return
			}
			was := s.pressed
			s.pressed = false
			c.Redraw()
			if was && s.enabled && s.it.OnClick != nil {
				s.it.OnClick()
			}
		}
	}
	c.Unwrap().AddKeyListener(swt.KeyListenerKeyPressedAdapter(key(true)))
	c.Unwrap().AddKeyListener(swt.KeyListenerKeyReleasedAdapter(key(false)))
	c.Unwrap().AddFocusListener(swt.FocusListenerFocusGainedAdapter(func(*swt.FocusEvent) {
		s.focused = true
		c.Redraw()
	}))
	c.Unwrap().AddFocusListener(swt.FocusListenerFocusLostAdapter(func(*swt.FocusEvent) {
		s.focused, s.pressed = false, false
		c.Redraw()
	}))
}

// state is how the segment draws now: pressed beats hover, a disabled segment stays at rest.
func (s *pillSeg) state() pillState {
	if !s.enabled {
		return pillRest
	}
	switch {
	case s.pressed:
		return pillPressed
	case s.hover:
		return pillHover
	default:
		return pillRest
	}
}

// paint draws the segment: its slice of the shared fill, the hover or press fill, the hairline
// before a later segment and the icon.
func (s *pillSeg) paint(g *gowt.GC) {
	w, h := g.Size()
	if w <= 0 || h <= 0 || s.fill == nil {
		return
	}
	iw, ih := int32(w), int32(h)
	gc := g.Unwrap()
	s.fillSlice(gc, iw, ih, pillAlpha(pillRest))
	if st := s.state(); st != pillRest {
		r := max(pillRadius(ih)-1, 0)
		gc.SetAlpha(pillAlpha(st))
		gc.SetBackground(s.fill)
		gc.FillRoundRectangle(1, 1, iw-2, ih-2, r, r)
		gc.SetAlpha(0xFF)
	}
	if s.index > 0 {
		gc.SetAlpha(pillSeparatorAlpha)
		gc.SetForeground(s.fill)
		y0, y1 := ih/4, ih-ih/4
		gc.DrawLine(0, y0, 0, y1)
		gc.SetAlpha(0xFF)
	}
	if s.focused {
		gc.SetAlpha(0x66)
		gc.SetForeground(s.fill)
		r := max(pillRadius(ih)-1, 0)
		gc.DrawRoundRectangle(1, 1, iw-2, ih-2, r, r)
		gc.SetAlpha(0xFF)
	}
	if s.image != nil {
		b := s.image.GetBounds()
		if !s.enabled {
			gc.SetAlpha(0x66)
		}
		gc.DrawImage(s.image, (iw-b.Width)/2, (ih-b.Height)/2)
		gc.SetAlpha(0xFF)
	}
}

// fillSlice draws the rest fill of this segment's slice of the capsule: rounded on the outer ends
// of the pill, square where segments meet.
func (s *pillSeg) fillSlice(gc *swt.GC, w, h, alpha int32) {
	gc.SetAlpha(alpha)
	gc.SetBackground(s.fill)
	r := pillRadius(h)
	switch {
	case s.segs <= 1:
		gc.FillRoundRectangle(0, 0, w, h, r, r)
	case s.index == 0:
		s.fillSide(gc, w, h, r, true)
	case s.index == s.segs-1:
		s.fillSide(gc, w, h, r, false)
	default:
		gc.FillRectangle(0, 0, w, h)
	}
	gc.SetAlpha(0xFF)
}

// fillSide fills a rectangle that has both corners rounded on one side: the outer end of the first
// or the last segment of a group.
func (s *pillSeg) fillSide(gc *swt.GC, w, h, r int32, left bool) {
	p := swt.NewPath(s.canvas.Unwrap().GetDisplay())
	defer p.Dispose()
	f := func(v int32) float32 { return float32(v) }
	if left {
		p.MoveTo(f(r), 0)
		p.LineTo(f(w), 0)
		p.LineTo(f(w), f(h))
		p.LineTo(f(r), f(h))
		p.AddArc(0, f(h-2*r), f(2*r), f(2*r), 180, 90)
		p.LineTo(0, f(r))
		p.AddArc(0, 0, f(2*r), f(2*r), 90, 90)
	} else {
		p.MoveTo(0, 0)
		p.LineTo(f(w-r), 0)
		p.AddArc(f(w-2*r), 0, f(2*r), f(2*r), 270, 90)
		p.LineTo(f(w), f(h-r))
		p.AddArc(f(w-2*r), f(h-2*r), f(2*r), f(2*r), 0, 90)
		p.LineTo(0, f(h))
	}
	p.Close()
	gc.FillPath(p)
}
