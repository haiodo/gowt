package gowt

import (
	"github.com/haiodo/gowt/internal/gtk"
	"github.com/haiodo/gowt/swt"
)

// setFieldSmall gives the entry the compact height; the caption font is set already.
func setFieldSmall(t *swt.Text) {
	gtk.SetEntrySmall(t.Handle, restyled(t))
	releaseEntryLook(t)
}

// setFieldAffix widens the entry's CSS padding by the affixes and draws them in it after GTK has drawn
// the entry, so frame, focus ring and caret stay the theme's.
func setFieldAffix(t *swt.Text, a *affix) {
	ea := gtk.EntryAffix{Prefix: a.prefix, Suffix: a.suffix}
	if a.image != nil {
		ea.Image = a.image.i.Surface
		b := a.image.i.GetBounds()
		ea.ImageW, ea.ImageH = float64(b.Width), float64(b.Height)
	}
	// Until GTK applies the padding, SWT measures the entry without the affixes: a width hint stands in.
	var hint int32
	extra := gtk.SetEntryAffix(t.Handle, ea, func() {
		if w, _, ok := sizeHints(&t.Control); ok && hint != 0 && *w == hint {
			*w = swt.DEFAULT
		}
		hint = 0
		restyled(t)()
	})
	if w, _, ok := sizeHints(&t.Control); ok && *w == swt.DEFAULT {
		hint = preferred(&t.Control).X + extra
		*w = hint
	}
	releaseEntryLook(t)
	if a.drag != nil {
		scrubEntry(t, a.drag)
	}
}

// restyled lays the parents of t out again once GTK has applied its CSS: SWT measures the entry from
// the padding GTK reports, which is the old one until then.
func restyled(t *swt.Text) func() {
	return func() {
		if a := appOf(&t.Control); a != nil && !t.IsDisposed() {
			a.relayout(&t.Control)
		}
	}
}

const entryLookKey = "gowt.entryLook"

func releaseEntryLook(t *swt.Text) {
	if t.GetDataKey(entryLookKey) != nil {
		return
	}
	t.SetDataKeyValue(entryLookKey, true)
	h := t.Handle
	t.AddDisposeListener(disposeFunc(func() { gtk.ReleaseEntryLook(h) }))
}

// scrubEntry turns a drag on the prefix into drag calls. The press there is kept from GTK, so the
// entry neither takes the focus nor starts a selection.
func scrubEntry(t *swt.Text, drag func(dx float64, shift bool)) {
	var last *int32
	moved, resize := false, false
	over := func(e *swt.Event) bool { return e.X < gtk.EntryValueStart(t.Handle) }
	t.AddListener(swt.MouseDown, eventFunc(func(e *swt.Event) {
		if e.Button != 1 || !over(e) {
			return
		}
		e.Doit = false
		x := e.X
		last, moved = &x, false
	}))
	t.AddListener(swt.MouseMove, eventFunc(func(e *swt.Event) {
		if last == nil {
			if o := over(e); o != resize {
				resize = o
				var c *swt.Cursor
				if o {
					c = t.GetDisplay().GetSystemCursor(swt.CURSOR_SIZEWE)
				}
				t.SetCursor(c)
			}
			return
		}
		if dx := e.X - *last; dx != 0 {
			*last = e.X
			moved = true
			drag(float64(dx), e.StateMask&swt.SHIFT != 0)
		}
	}))
	t.AddListener(swt.MouseUp, eventFunc(func(e *swt.Event) {
		if last == nil {
			return
		}
		last = nil
		if !moved {
			t.SetFocus()
			t.SelectAll()
		}
	}))
}
