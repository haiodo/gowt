package qatree

import (
	"encoding/json"
	"fmt"

	"github.com/haiodo/gowt/swt"
)

// drag runs swt.QADrag from e to the control or row args.to picks. A row source is selected first,
// as the mouse press before a drag selects it.
func drag(d *swt.Display, e *entry, args json.RawMessage) error {
	var a struct {
		To   *Selector
		X, Y *int
	}
	if err := parse(args, &a); err != nil {
		return err
	}
	if a.To == nil {
		return fmt.Errorf("qatree: drag needs a target: {\"to\": selector}")
	}
	es, err := resolve(walk(d), *a.To)
	if err != nil {
		return err
	}
	if len(es) != 1 {
		return fmt.Errorf("qatree: drag target %s: %d controls match%s", *a.To, len(es), paths(es))
	}
	to := es[0]
	if err := reachable(d, to); err != nil {
		return fmt.Errorf("qatree: drag target %s: %w", to.n.Path, err)
	}
	if isRow(e) {
		if err := selectItem(e, nil, nil); err != nil {
			return err
		}
	}
	src, sx, sy, err := dragPoint(e, nil, nil)
	if err != nil {
		return err
	}
	dst, dx, dy, err := dragPoint(to, a.X, a.Y)
	if err != nil {
		return err
	}
	if _, err := swt.QADrag(src, sx, sy, dst, dx, dy); err != nil {
		return fmt.Errorf("qatree: drag %s to %s: %w", e.n.Path, to.n.Path, err)
	}
	return nil
}

// dragPoint is the control that owns e and the point in its coordinates a mouse would use: the
// middle of the row, of the control, or of the selected row of a Tree or a Table; x and y override it.
func dragPoint(e *entry, x, y *int) (*swt.Control, int32, int32, error) {
	var b *swt.Rectangle
	var c *swt.Control
	switch o := e.obj.(type) {
	case *swt.TreeItem:
		b, c = o.GetBounds(), o.GetParent().AsControl()
	case *swt.TableItem:
		b, c = o.GetBounds(), o.GetParent().AsControl()
	case *swt.Tree:
		c = o.AsControl()
		if s := o.GetSelection(); len(s) > 0 {
			b = s[0].GetBounds()
		}
	case *swt.Table:
		c = o.AsControl()
		if s := o.GetSelection(); len(s) > 0 {
			b = s[0].GetBounds()
		}
	default:
		if isRow(e) {
			return nil, 0, 0, fmt.Errorf("qatree: drag does not apply to %s %s", e.n.Class, e.n.Path)
		}
		var err error
		if c, err = controlOf(e); err != nil {
			return nil, 0, 0, err
		}
	}
	sz := c.GetSize()
	px, py := sz.X/2, sz.Y/2
	if x != nil && y != nil {
		px, py = int32(*x), int32(*y)
	} else if b != nil {
		px, py = b.X+b.Width/2, b.Y+b.Height/2
	}
	if px < 0 || py < 0 || px >= sz.X || py >= sz.Y {
		return nil, 0, 0, fmt.Errorf("qatree: drag: the point %d,%d is outside %s %s; scroll the row into view", px, py, e.n.Class, e.n.Path)
	}
	return c, px, py, nil
}
