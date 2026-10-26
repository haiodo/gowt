// Package printing draws gfx pictures and scenes onto the pages of a gowt.PrintJob.
//
//	job, ok := win.PrintDialog(gowt.PrintOptions{Title: "Map", Margin: 36})
//	if !ok { return }
//	err := job.Print(1, func(p *gowt.PrintPage) {
//		printing.Scene(p, scene, nil, gfx.Rect{}, printing.Options{})
//	})
//
// A page is a gfx.Page in points. On Linux the scene is drawn with cairo and Pango and on macOS
// with CoreGraphics and CoreText straight into the print context, so the paper gets vectors and
// text as the PDF export does. On Windows Direct2D has no printer target: the page is drawn at
// the printer's resolution in bands and sent as bitmaps, so it is as exact as the screen but not
// vector, and a PDF made by the Microsoft Print to PDF driver holds images.
package printing

import (
	"errors"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/gfx"
)

// Options say how a scene sits on the page area (gowt.PrintPage.Area).
type Options struct {
	// Scale is points per scene unit, or the cap of the fitted scale; 0 fits the area to the page
	// as large as it goes, as gfx.PDFOptions does.
	Scale float64
	// Background, when its A is above zero, is filled over the whole scene area behind the content.
	Background gfx.Color
}

// Page returns the gfx.Page of p: points, y down, the origin at the top left corner of the
// printable area.
func Page(p *gowt.PrintPage) (gfx.Page, error) { return nativePage(p) }

// Scene draws the subtree of root (the whole scene when root is nil) fitted into the area of p, as
// gfx.WriteSubtreePDF does for a PDF page. area is the scene rectangle, or the zero Rect for the
// subtree bounds.
func Scene(p *gowt.PrintPage, s *gfx.Scene, root *gfx.Node, area gfx.Rect, o Options) error {
	s.Commit()
	if root == nil {
		root = s.Root()
	}
	if area.Empty() || area == (gfx.Rect{}) {
		s.Read(func() { area = root.SubtreeBounds() })
	}
	var b gfx.Items
	s.CollectSubtree(root, area, &b)
	return draw(p, area, b.List, o)
}

// Picture draws pic, the whole of its bounds, fitted into the area of p.
func Picture(p *gowt.PrintPage, pic *gfx.Picture, o Options) error {
	return draw(p, pic.Bounds, []gfx.Item{{Pic: pic, M: gfx.Identity(), Alpha: 1}}, o)
}

func draw(p *gowt.PrintPage, area gfx.Rect, items []gfx.Item, o Options) error {
	page, err := nativePage(p)
	if err != nil {
		return err
	}
	l, err := gfx.PDFOptions{PageW: p.Area.W, PageH: p.Area.H, Scale: o.Scale}.Layout(area)
	if err != nil {
		return errors.Join(errors.New("printing: nothing to print on the page"), err)
	}
	page.Draw(l.Scale, gfx.Point{X: l.Origin.X - p.Area.X/l.Scale, Y: l.Origin.Y - p.Area.Y/l.Scale}, gfx.ClipItems(items, area, o.Background))
	return nil
}
