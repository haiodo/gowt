package gowt

import (
	"fmt"
	"slices"

	"github.com/haiodo/gowt/swt"
)

// DisclosureView is a collapsible section: a header row and a body that folds.
type DisclosureView struct {
	base
	c, body, trailing *swt.Composite
	chevron           *swt.Button
	title             *swt.Label
	expanded          bool
	handlers          []*func(bool)
}

// Disclosure creates a collapsible section of an inspector: a header of a disclosure chevron, the
// title and the Trailing widgets, over a body of the children stacked as in a VBox, which a click on
// the chevron or the title folds. It starts expanded unless Collapsed; Gap and Padding apply to the
// body. macOS: the chevron is an NSButton with the disclosure bezel; GTK a flat arrow button; Windows
// the WinUI expander chevron.
// Options: Collapsed, Trailing, OnChange (bool: expanded), Bind (bool), Gap, Padding, children.
func Disclosure(title string, opts ...Option) Option {
	return &node{"Disclosure", func(h host) Widget {
		c := swt.NewCompositeParentStyle(h.into(), swt.NONE)
		c.SetLayout(newBox(false, Spacing.XS, 0))
		head := swt.NewCompositeParentStyle(c, swt.NONE)
		head.SetLayout(newBox(true, Spacing.XS, 0))
		head.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false))
		d := &DisclosureView{base: base{&c.Control}, c: c, expanded: true}
		d.chevron = newChevron(head)
		d.chevron.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.CENTER, swt.CENTER, false, false))
		setAccessibleLabel(&d.chevron.Control, title)
		setChevron(d.chevron, true)
		onSelect(d.chevron, func(*swt.SelectionEvent) { d.toggle() })
		d.title = swt.NewLabel(head, swt.NONE)
		d.title.SetText(title)
		td := swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.CENTER, true, false)
		td.MinimumWidth = swt.DEFAULT // the title keeps its width in a narrow inspector
		d.title.SetLayoutData(td)
		setFont(&d.title.Control, Type.BodyStrong, func(_ *swt.Control, set func()) { set() })
		d.title.AddListener(swt.MouseUp, listenerFunc(func(*swt.Event) { d.toggle() }))
		d.trailing = swt.NewCompositeParentStyle(head, swt.NONE)
		d.trailing.SetLayout(newBox(true, 0, 0))
		d.trailing.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.END, swt.CENTER, false, false))
		gap, pad := spacingOf(opts, 0)
		d.body = swt.NewCompositeParentStyle(c, swt.NONE)
		d.body.SetLayout(newBox(false, gap, pad))
		d.body.SetLayoutData(swt.NewGridDataHorizontalAlignmentVerticalAlignmentGrabExcessHorizontalSpaceGrabExcessVerticalSpace(swt.FILL, swt.FILL, true, true))
		finish(d, 0, opts)
		return d
	}}
}

// SetTitle replaces the title of the header and the name of the chevron for VoiceOver, without
// folding or unfolding the body.
func (d *DisclosureView) SetTitle(s string) {
	d.title.SetText(s)
	setAccessibleLabel(&d.chevron.Control, s)
	if !d.c.IsDisposed() {
		flushUp(&d.title.Control)
		d.c.Layout()
	}
}

// Expanded reports whether the body shows.
func (d *DisclosureView) Expanded() bool { return d.expanded }

// SetExpanded shows or folds the body without running OnChange; the parents are laid out again on
// the next event-loop turn.
func (d *DisclosureView) SetExpanded(v bool) {
	if v == d.expanded {
		return
	}
	d.expanded = v
	setChevron(d.chevron, v)
	sizing(&d.c.Control, func() {
		d.body.SetVisible(v)
		d.body.GetLayoutData().(*swt.GridData).Exclude = !v
		d.c.Layout()
	})
}

// OnChange runs f with the new state after the user folded or unfolded the body.
func (d *DisclosureView) OnChange(f func(expanded bool)) { d.handlers = append(d.handlers, &f) }

// Add appends children to the body.
func (d *DisclosureView) Add(children ...Option) {
	add(d, children)
	if !d.c.IsDisposed() {
		flushUp(&d.body.Control)
		d.c.Layout()
	}
}

// Unwrap returns the composite that holds the header and the body.
func (d *DisclosureView) Unwrap() *swt.Composite { return d.c }

// AsComposite returns the body, as Panel does, for the look package.
func (d *DisclosureView) AsComposite() *swt.Composite { return d.body }

func (d *DisclosureView) into() *swt.Composite { return d.body }
func (d *DisclosureView) added(Widget)         {}

// toggle is a click on the chevron or the title.
func (d *DisclosureView) toggle() {
	d.SetExpanded(!d.expanded)
	for _, f := range slices.Clone(d.handlers) {
		(*f)(d.expanded)
	}
}

// listen is OnChange with a removal, for Bind.
func (d *DisclosureView) listen(f func()) func() {
	g := func(bool) { f() }
	p := &g
	d.handlers = append(d.handlers, p)
	return func() { d.handlers = slices.DeleteFunc(d.handlers, func(h *func(bool)) bool { return h == p }) }
}

// Collapsed starts a Disclosure folded.
func Collapsed() Option {
	return hook(func(w Widget) error {
		d, ok := w.(*DisclosureView)
		if !ok {
			return fmt.Errorf("gowt: Collapsed applies to Disclosure, not to %s", nameOf(w.control()))
		}
		d.SetExpanded(false)
		return nil
	})
}

// Trailing puts widgets at the right end of a Disclosure header, such as Borderless + and - buttons.
func Trailing(children ...Option) Option {
	return hook(func(w Widget) error {
		d, ok := w.(*DisclosureView)
		if !ok {
			return fmt.Errorf("gowt: Trailing applies to Disclosure, not to %s", nameOf(w.control()))
		}
		add(&Panel{newPanel(d.trailing)}, children)
		return nil
	})
}

type listenerFunc func(*swt.Event)

func (f listenerFunc) HandleEvent(e *swt.Event) { f(e) }
