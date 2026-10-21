package gowt

import (
	"sort"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// rowClip hides the content rows far from the visible part natively: on each scroll frame AppKit updated
// the tracking areas of every view in the content, each answered from Go, but it skips a hidden subtree.
// SWT still sees the rows visible: layout, focus, traversal and Visible do not change.
type rowClip struct {
	s             *ScrollView
	rows          []*swt.Control // the shown children of the content in layout order, as of the last layout
	tops, bottoms []int32
	lo, hi        int  // rows[lo:hi] are natively shown, the others hidden
	all           bool // VoiceOver is on: every row is shown, hidden views leave the accessibility tree
}

func (s *ScrollView) watchRows() {
	r := &rowClip{s: s}
	clip := func(*swt.ControlEvent) { r.scroll() }
	s.body.c.AddControlListener(swt.ControlListenerControlMovedAdapter(clip))
	s.s.AddControlListener(swt.ControlListenerControlResizedAdapter(clip))
	if l, ok := s.body.c.GetLayout().Impl().(*boxLayout); ok {
		l.laidOut = r.measure
	}
}

// measure takes the rows' places after a layout of the content and hides or shows every row.
func (r *rowClip) measure() {
	body := r.s.body.c
	r.rows, r.tops, r.bottoms = r.rows[:0], r.tops[:0], r.bottoms[:0]
	for _, k := range body.GetChildren() {
		if !k.GetVisible() {
			continue // hidden by SetVisible and excluded: SWT owns its native state and its bounds are stale
		}
		b := k.GetBounds()
		r.rows = append(r.rows, k)
		r.tops = append(r.tops, b.Y)
		r.bottoms = append(r.bottoms, b.Y+b.Height)
	}
	r.all = cocoa.VoiceOverEnabled()
	from, to := r.band()
	for i, k := range r.rows {
		setHidden(k, !r.all && (r.bottoms[i] < from || r.tops[i] > to))
	}
	r.lo, r.hi = r.find(from, to)
}

// scroll hides the rows that left the band and shows those that entered it, so a frame costs the rows
// that change, not all of them.
func (r *rowClip) scroll() {
	body := r.s.body.c
	if body.IsDisposed() || r.s.s.IsDisposed() {
		return
	}
	if r.all != cocoa.VoiceOverEnabled() {
		r.measure()
		return
	}
	if r.all {
		return
	}
	lo, hi := r.find(r.band())
	for i := r.lo; i < r.hi; i++ {
		if i < lo || i >= hi {
			setHidden(r.rows[i], true)
		}
	}
	for i := lo; i < hi; i++ {
		if i < r.lo || i >= r.hi {
			setHidden(r.rows[i], false)
		}
	}
	r.lo, r.hi = lo, hi
	// A hidden view gives up the keyboard focus: the row holding it stays shown wherever it is.
	for f := body.GetDisplay().GetFocusControl(); f != nil && f != &body.Control; f = &f.GetParent().Control {
		if f.GetParent() == body {
			setHidden(f, false)
			break
		}
		if &f.GetShell().Control == f {
			break
		}
	}
}

// band is the part of the content kept shown: the visible part and a quarter of it above and below.
func (r *rowClip) band() (from, to int32) {
	h := r.s.s.GetClientArea().Height
	top := -r.s.body.c.GetLocation().Y
	return top - h/4, top + h + h/4
}

// find returns the rows that reach into from..to; a VBox places its rows top to bottom.
func (r *rowClip) find(from, to int32) (lo, hi int) {
	lo = sort.Search(len(r.bottoms), func(i int) bool { return r.bottoms[i] >= from })
	hi = sort.Search(len(r.tops), func(i int) bool { return r.tops[i] > to })
	return lo, max(lo, hi)
}

func setHidden(k *swt.Control, hide bool) {
	if k.IsDisposed() || !k.GetVisible() {
		return
	}
	if v := k.TopView(); v.IsHidden() != hide {
		v.SetHidden(hide)
		k.InvalidateVisibleRegion()
	}
}
