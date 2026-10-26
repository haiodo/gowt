package gfx

import (
	"errors"
	"fmt"
	"io"
	"math"
)

// Document is a paged vector output, a PDF file. A PDFRenderer makes one.
type Document interface {
	// NewPage ends the page before it and starts one of wpt by hpt points (1/72 inch).
	NewPage(wpt, hpt float64) Page
	// Close ends the last page and finishes the output. The Document is not used afterwards.
	Close() error
}

// Page is drawn in points, y down, the origin at the top left corner. Text stays text and shapes
// stay vector: nothing is rasterized except what the picture holds (images, shadows on some backends).
type Page interface {
	// Draw paints the items in order, as Target.Draw does: scale is points per scene unit and origin
	// the scene point at the top left corner of the page.
	Draw(scale float64, origin Point, items []Item)
}

// PDFRenderer is a Renderer that can also write PDF: CoreGraphics on macOS, cairo on Linux; on
// Windows the PDF writer of package gfx/pdf with Direct2D's text engine for font lookup.
type PDFRenderer interface {
	NewPDF(w io.Writer) (Document, error)
}

// PDFOptions place an area of a scene on a page. The zero value makes a page of the size of the
// area at one point per scene unit.
type PDFOptions struct {
	// PageW and PageH are the page size in points. Both zero: the page fits the area at Scale
	// plus Margin on every side. Both set: the area is centred in the page inside Margin and
	// scaled to the largest scale that fits, but not past Scale when that is set.
	PageW, PageH float64
	Margin       float64
	// Scale is points per scene unit, or the cap of the fitted scale on a fixed page.
	Scale float64
	// Background, when its A is above zero, is filled over the whole area behind the content.
	Background Color
}

// PageLayout says where an area of the scene lies on a page.
type PageLayout struct {
	W, H   float64 // page size in points
	Scale  float64 // points per scene unit
	Origin Point   // scene point at the top left corner of the page
}

// Layout computes the layout of area under o, or an error when nothing is left to draw into.
func (o PDFOptions) Layout(area Rect) (PageLayout, error) {
	if noArea(area) || area.W() <= 0 || area.H() <= 0 {
		return PageLayout{}, errors.New("gfx: nothing to write: the area is empty")
	}
	m := math.Max(o.Margin, 0)
	l := PageLayout{W: o.PageW, H: o.PageH, Scale: o.Scale}
	switch {
	case o.PageW == 0 && o.PageH == 0:
		if l.Scale <= 0 {
			l.Scale = 1
		}
		l.W, l.H = area.W()*l.Scale+2*m, area.H()*l.Scale+2*m
	case o.PageW > 0 && o.PageH > 0:
		bw, bh := o.PageW-2*m, o.PageH-2*m
		if !(bw > 0 && bh > 0) {
			return PageLayout{}, fmt.Errorf("gfx: margin %v leaves nothing of the %vx%v page", m, o.PageW, o.PageH)
		}
		fit := math.Min(bw/area.W(), bh/area.H())
		if l.Scale <= 0 || fit < l.Scale {
			l.Scale = fit
		}
	default:
		return PageLayout{}, fmt.Errorf("gfx: page size %vx%v: set both sides or neither", o.PageW, o.PageH)
	}
	left := (l.W - area.W()*l.Scale) / 2
	top := (l.H - area.H()*l.Scale) / 2
	l.Origin = Point{area.Min.X - left/l.Scale, area.Min.Y - top/l.Scale}
	return l, nil
}

// ClipItems returns items clipped to area, with a background of the whole area first when bg is
// not transparent: what a page of a fixed size gets, so that nothing lies outside the area.
func ClipItems(items []Item, area Rect, bg Color) []Item {
	clip := Clip{M: Identity(), Kind: ClipRect, Rect: area}
	out := make([]Item, 0, len(items)+1)
	if bg.A > 0 {
		rec := NewRecorderWith(EstimateText{})
		rec.FillRect(area, bg)
		out = append(out, Item{Pic: rec.Finish(), M: Identity(), Alpha: 1})
	}
	for _, it := range items {
		it.Clips = append([]Clip{clip}, it.Clips...)
		out = append(out, it)
	}
	return out
}

// WriteSubtreePDF writes the subtree of root, as RenderSubtree draws it, as a one-page PDF; a nil
// root writes the whole scene. area is the scene rectangle to write, or the zero Rect for the
// subtree bounds. It commits the scene's pending changes first and needs no window.
func WriteSubtreePDF(r PDFRenderer, w io.Writer, s *Scene, root *Node, area Rect, o PDFOptions) error {
	s.Commit()
	if root == nil {
		root = s.Root()
	}
	if noArea(area) {
		s.Read(func() { area = root.SubtreeBounds() })
	}
	l, err := o.Layout(area)
	if err != nil {
		return err
	}
	var b Items
	s.CollectSubtree(root, area, &b)
	return writePDF(r, w, l, ClipItems(b.List, area, o.Background))
}

// WritePicturePDF writes one picture, the whole of its bounds, as a one-page PDF.
func WritePicturePDF(r PDFRenderer, w io.Writer, p *Picture, o PDFOptions) error {
	l, err := o.Layout(p.Bounds)
	if err != nil {
		return err
	}
	return writePDF(r, w, l, ClipItems([]Item{{Pic: p, M: Identity(), Alpha: 1}}, p.Bounds, o.Background))
}

func writePDF(r PDFRenderer, w io.Writer, l PageLayout, items []Item) error {
	doc, err := r.NewPDF(w)
	if err != nil {
		return err
	}
	doc.NewPage(l.W, l.H).Draw(l.Scale, l.Origin, items)
	return doc.Close()
}
