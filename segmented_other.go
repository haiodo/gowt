//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

// segments is a row of toggle buttons in equal columns with no gap, styled per OS by styleSegment;
// the facade keeps one of them on.
type segments struct {
	buttons   []*swt.Button
	momentary bool
}

func newSegments(s *SegmentedView, segs []Segment) *segments {
	l := swt.NewGridLayout()
	l.NumColumns = int32(max(len(segs), 1))
	l.MakeColumnsEqualWidth = true
	l.MarginWidth, l.MarginHeight, l.HorizontalSpacing = 0, 0, 0
	s.c.SetLayout(l)
	g := &segments{}
	a := appOf(&s.c.Control)
	for i, sg := range segs {
		b := swt.NewButton(s.c, swt.TOGGLE)
		b.SetText(sg.Text)
		if sg.Image != nil {
			b.SetImage(sg.Image.i)
		}
		if sg.Tooltip != "" {
			b.SetToolTipText(sg.Tooltip)
			if sg.Text == "" {
				setAccessibleLabel(&b.Control, sg.Tooltip)
			}
		}
		if a != nil && a.body != nil {
			setFont(&b.Control, *a.body, func(_ *swt.Control, set func()) { set() })
		}
		b.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false))
		styleSegment(b, i, len(segs))
		onSelect(b, func(*swt.SelectionEvent) {
			g.clicked(i)
			s.pressed(i)
		})
		g.buttons = append(g.buttons, b)
	}
	return g
}

// clicked keeps button i on and the others off, or turns i off again when the control is momentary.
func (g *segments) clicked(i int) {
	if g.momentary {
		g.buttons[i].SetSelection(false)
		return
	}
	g.selectAt(i)
}

func (g *segments) count() int { return len(g.buttons) }

func (g *segments) selectAt(i int) {
	for j, b := range g.buttons {
		b.SetSelection(j == i)
	}
}

func (g *segments) setMomentary() { g.momentary = true }

func (g *segments) setEnabled(on bool) {
	for _, b := range g.buttons {
		b.SetEnabled(on)
	}
}

func (g *segments) setSmall() {
	for _, b := range g.buttons {
		setFont(&b.Control, smallFont, sizing)
	}
}
