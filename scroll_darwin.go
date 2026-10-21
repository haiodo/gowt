package gowt

import (
	"sort"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// rowClip hides the content rows far from the visible part natively: on each scroll frame AppKit updated
// the tracking areas of every view in the content, each answered from Go, but it skips a hidden subtree.
// SWT state stays: forceFocus and print show a hidden row first (swt/widgets_reveal_manual_darwin.go).
type rowClip struct {
	s             *ScrollView
	rows          []*swt.Control // the shown children of the content in layout order, as of the last layout
	tops, bottoms []int32
	lo, hi        int  // rows[lo:hi] are in the band; the others are hidden unless they hold the focus
	all           bool // VoiceOver is on: every row is shown, hidden views leave the accessibility tree
}

// clips are the live rowClips, for the VoiceOver observer.
var clips = map[*rowClip]bool{}

func (s *ScrollView) watchRows() {
	r := &rowClip{s: s}
	clip := func(*swt.ControlEvent) { r.scroll() }
	s.body.c.AddControlListener(swt.ControlListenerControlMovedAdapter(clip))
	s.s.AddControlListener(swt.ControlListenerControlResizedAdapter(clip))
	if l, ok := s.body.c.GetLayout().Impl().(*boxLayout); ok {
		l.laidOut = r.measure
	}
	clips[r] = true
	s.s.AddDisposeListener(disposeFunc(func() { delete(clips, r) }))
	d := s.s.GetDisplay()
	cocoa.ObserveVoiceOver(func() {
		d.AsyncExec(jrt.NewRunnable(func() {
			for r := range clips {
				r.measure()
			}
		}))
	})
}

// measure takes the rows' places after a layout of the content and hides or shows every row.
func (r *rowClip) measure() {
	body := r.s.body.c
	if body.IsDisposed() {
		return
	}
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
	focus := r.focusRow()
	for i, k := range r.rows {
		setHidden(k, !r.all && k != focus && (r.bottoms[i] < from || r.tops[i] > to))
	}
	r.lo, r.hi = r.find(from, to)
}

// scroll hides the rows that left the band and shows those that entered it, so a frame costs the rows
// that change, not all of them.
func (r *rowClip) scroll() {
	if r.s.body.c.IsDisposed() || r.s.s.IsDisposed() || r.all {
		return
	}
	lo, hi := r.find(r.band())
	focus := r.focusRow()
	for i := r.lo; i < r.hi; i++ {
		if (i < lo || i >= hi) && r.rows[i] != focus {
			setHidden(r.rows[i], true)
		}
	}
	for i := lo; i < hi; i++ {
		if i < r.lo || i >= r.hi {
			setHidden(r.rows[i], false)
		}
	}
	r.lo, r.hi = lo, hi
}

// focusRow is the row holding the first responder of the ScrollView's window, key or not: a hidden
// view gives the focus up, and AppKit would not let it come back.
func (r *rowClip) focusRow() *swt.Control {
	body := r.s.body.c
	win := body.View.Window()
	if win == nil {
		return nil
	}
	fr := win.FirstResponder()
	if fr == nil || !fr.IsKindOfClass(cocoa.OSClass_NSView) {
		return nil
	}
	content := body.ContentView()
	v := cocoa.NewNSViewOverload1(fr.Id)
	for v != nil && v.Id != content.Id {
		if p := v.Superview(); p != nil && p.Id == content.Id {
			for _, k := range r.rows {
				if k.TopView().Id == v.Id {
					return k
				}
			}
			return nil
		}
		v = v.Superview()
	}
	return nil
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
