//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

// installCard gives c the card colour, inherited by labels and rows, and paints the corners outside the
// rounded rectangle in the parent's colour.
func installCard(c *swt.Composite) {
	d := c.GetDisplay()
	bg, fg := d.GetSystemColor(swt.COLOR_WIDGET_BACKGROUND), d.GetSystemColor(swt.COLOR_WIDGET_FOREGROUND)
	card := swt.NewColorRedGreenBlue(mix(bg.GetRed(), fg.GetRed()), mix(bg.GetGreen(), fg.GetGreen()), mix(bg.GetBlue(), fg.GetBlue()))
	c.SetBackgroundWithColor(card)
	c.SetBackgroundMode(swt.INHERIT_DEFAULT)
	c.AddPaintListener(painter(func(e *swt.PaintEvent) {
		r := c.GetClientArea()
		g := e.Gc
		g.SetAntialias(swt.ON)
		g.SetBackground(c.GetParent().GetBackground())
		g.FillRectangle(0, 0, r.Width, r.Height)
		g.SetBackground(card)
		g.FillRoundRectangle(0, 0, r.Width, r.Height, 2*sectionRadius, 2*sectionRadius)
	}))
}

// mix moves a 6% of the way from a to b.
func mix(a, b int32) int32 { return a + (b-a)*6/100 }
