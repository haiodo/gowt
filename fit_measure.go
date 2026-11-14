//go:build linux || darwin

package gowt

import "github.com/haiodo/gowt/swt"

// textMeasure sizes cell texts with the font of a control, for the packColumn of a port whose own
// autosize does not.
type textMeasure struct{ gc *swt.GC }

func newTextMeasure(c *swt.Control) textMeasure {
	gc := swt.NewGCDrawable(c)
	gc.SetFont(c.GetFont())
	return textMeasure{gc}
}

func (m textMeasure) done() { m.gc.Dispose() }

func (m textMeasure) text(s string) int {
	if s == "" {
		return 0
	}
	return int(m.gc.TextExtent(s).X)
}

func (m textMeasure) cell(s string, img *swt.Image, level int) int {
	w := m.text(s) + fitCellPad + level*fitIndent
	if img != nil {
		w += int(img.GetBounds().Width) + fitImageGap
	}
	return w
}
