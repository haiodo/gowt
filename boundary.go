package gowt

import (
	"errors"
	"fmt"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// Boundary creates its children in a column, as a VBox does, and shows a fallback in their place
// when they fail: Svelte's <svelte:boundary>. Caught are panics and reported errors while the
// children are made (constructors, Ref, Init, misused options, Add on a container inside) and panics
// in their bindings and effects after a state change (Bind, BindText, Watch, Rows fill, ...). The
// children go with their effects. Event handlers, Async, After and Every are not caught. Every caught
// error also goes to App.OnError; the innermost boundary catches first, and an error in a fallback
// goes to the boundary above. Without Failed the fallback is the message and a Retry button.
// Options: Failed, Gap, Padding, Border, children.
func Boundary(opts ...Option) Option {
	return &node{"Boundary", func(h host) Widget {
		c := swt.NewCompositeParentStyle(h.into(), resolve(swt.NONE, opts))
		gap, pad := spacingOf(opts, 0)
		c.SetLayout(newBox(false, gap, pad))
		p := &Panel{newPanel(c)}
		b := &boundary{p: p, fallback: defaultFailed}
		var own []Option
		for _, o := range opts {
			switch o := o.(type) {
			case *node:
				b.children = append(b.children, o)
			case failedOpt:
				if o != nil {
					b.fallback = o
				}
			default:
				own = append(own, o)
			}
		}
		c.SetDataKeyValue(boundaryKey, b)
		c.AddDisposeListener(b)
		boundaries++
		b.build()
		// The options of the boundary itself belong to its parent: an error in them is not caught here.
		finish(p, sBorder, own)
		return p
	}}
}

// Failed sets what a Boundary shows when its children fail: f gets the error and reset, which
// disposes the fallback and creates the children again on the next event-loop turn. f returns one
// widget description, nil for nothing.
func Failed(f func(err error, reset func()) Option) Option { return failedOpt(f) }

type failedOpt func(err error, reset func()) Option

func (failedOpt) option() {}

const boundaryKey = "gowt.boundary"

// boundaries counts the live boundaries, so that bindings look for one only when there is one.
var boundaries int

type boundary struct {
	p        *Panel
	children []Option
	fallback failedOpt
	stop     func() // the rx scope of the children
	fbStop   func() // the rx scope of the fallback
	building bool   // its children are being created: errors are collected in err
	failed   bool   // the fallback is shown
	err      error
}

// build creates the children; the first error collected on the way replaces them with the fallback.
func (b *boundary) build() {
	b.err = nil
	b.stop = rx.Catch(func() { b.run(func() { addNodes(b.p, b.children) }) }, b.caught)
	if b.err != nil {
		b.fail(b.err)
	}
}

// run calls f with b collecting the errors reported in it and the panic that ends it.
func (b *boundary) run(f func()) {
	b.building = true
	defer func() {
		b.building = false
		if r := recover(); r != nil {
			err := panicError(r)
			b.report(err) // here the stack still shows where the panic was raised
			b.record(err)
		}
	}()
	f()
}

func (b *boundary) record(err error) {
	if b.err == nil {
		b.err = err
	}
}

func (b *boundary) report(err error) {
	if a := appOf(b.p.c); a != nil {
		a.report(err)
	}
}

// caught takes a panic of an effect inside b, handed over by rx during a flush.
func (b *boundary) caught(r any) {
	err := panicError(r)
	b.report(err)
	b.fail(err)
}

// fail replaces the children with the fallback; while they are being created it only collects err.
func (b *boundary) fail(err error) {
	if b.building {
		b.record(err)
		return
	}
	if b.failed || b.p.c.IsDisposed() {
		return
	}
	b.failed = true
	b.clear(&b.stop)
	b.showFallback(err)
	b.relayout()
}

// clear disposes the rx scope in *stop and the native children. A panic in an OnCleanup on the way
// is reported and does not stop the rest.
func (b *boundary) clear(stop *func()) {
	defer func() {
		if r := recover(); r != nil {
			b.report(panicError(r))
		}
	}()
	if f := *stop; f != nil {
		*stop = nil
		f()
	}
	for _, ch := range b.p.c.GetChildren() {
		ch.Dispose()
	}
}

func (b *boundary) showFallback(err error) {
	defer func() {
		if r := recover(); r != nil {
			e := panicError(r)
			b.report(e)
			b.escalate(e)
		}
	}()
	b.fbStop = rx.Scope(func() {
		o := b.fallback(err, b.resetLater)
		switch o := o.(type) {
		case nil:
		case *node:
			if w := o.build(b.p); w != nil {
				b.p.added(w)
			}
		default:
			panic(errors.New("gowt: Failed returns a widget, not an option"))
		}
	})
}

// escalate hands an error of the fallback to the boundary above b.
func (b *boundary) escalate(err error) {
	if o := boundaryOf(&b.p.c.Control); o != nil {
		o.fail(err)
	}
}

// resetLater is the reset given to Failed. It waits for the next event-loop turn, as the Retry button
// that calls it is disposed by the reset.
func (b *boundary) resetLater() {
	if b.p.c.IsDisposed() {
		return
	}
	defer ignoreDisposed()
	b.p.c.GetDisplay().AsyncExec(jrt.NewRunnable(b.reset))
}

func (b *boundary) reset() {
	if !b.failed || b.p.c.IsDisposed() {
		return
	}
	b.clear(&b.fbStop)
	b.failed = false
	b.build()
	b.relayout()
}

// relayout lays the window out again on the next turn after the content changed on screen.
func (b *boundary) relayout() {
	c := b.p.c
	if c.IsDisposed() || !c.IsVisible() {
		return
	}
	changed := c.GetChildren()
	if len(changed) == 0 {
		changed = []*swt.Control{&c.Control}
	}
	c.GetShell().LayoutOverload4(changed, swt.DEFER)
}

func (b *boundary) WidgetDisposed(*swt.DisposeEvent) {
	boundaries--
	for _, f := range []func(){b.stop, b.fbStop} {
		if f != nil {
			f()
		}
	}
	b.stop, b.fbStop = nil, nil
}

// boundaryOf returns the boundary whose children c is among, at any depth, nil outside every
// boundary. A boundary that shows its fallback is skipped: the fallback belongs to the one above.
func boundaryOf(c *swt.Control) *boundary {
	if boundaries == 0 {
		return nil
	}
	for p := c.GetParent(); p != nil; p = p.GetParent() {
		if b, ok := p.GetDataKey(boundaryKey).(*boundary); ok && !b.failed {
			return b
		}
	}
	return nil
}

// buildingAt returns the boundary creating its children around c, c included, nil when there is none.
func buildingAt(c *swt.Control) *boundary {
	if boundaries == 0 {
		return nil
	}
	if b, ok := c.GetDataKey(boundaryKey).(*boundary); ok && b.building {
		return b
	}
	for p := c.GetParent(); p != nil; p = p.GetParent() {
		if b, ok := p.GetDataKey(boundaryKey).(*boundary); ok && b.building {
			return b
		}
	}
	return nil
}

// addIn creates children in h, a container that exists already; inside a boundary it is caught as
// the first build of the boundary's children is.
func addIn(h host, children []Option) {
	c := h.control()
	b, _ := c.GetDataKey(boundaryKey).(*boundary)
	if b == nil || b.failed {
		b = boundaryOf(c)
	}
	if b == nil || b.building {
		addNodes(h, children)
		return
	}
	b.err = nil
	b.run(func() { addNodes(h, children) })
	if b.err != nil {
		b.fail(b.err)
	}
}

var failedText = Adaptive{Light: RGB{R: 180, G: 30, B: 30}, Dark: RGB{R: 255, G: 120, B: 110}}

func defaultFailed(err error, reset func()) Option {
	return VBox(GrowX(),
		Label(err.Error(), Wrap(), GrowX(), Size(320, 0), Foreground(failedText)),
		Button("Retry", OnClick(reset)))
}

// panicError turns a recovered value into the error OnError gets: "gowt: panic: v" for a non-error.
func panicError(r any) error {
	err, ok := r.(error)
	if !ok {
		err = fmt.Errorf("panic: %v", r)
	}
	return fmt.Errorf("gowt: %w", err)
}
