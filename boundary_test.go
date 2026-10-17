package gowt

import (
	"strings"
	"testing"

	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// shown is what a test Failed records: the error, and reset to call later.
type shown struct {
	errs  []string
	reset func()
}

func (s *shown) failed(err error, reset func()) Option {
	s.errs = append(s.errs, err.Error())
	s.reset = reset
	return Label("fallback: " + err.Error())
}

// texts lists the texts of the labels and buttons directly in p.
func texts(p *Panel) []string {
	var out []string
	for _, c := range p.c.GetChildren() {
		switch w := c.Impl().(type) {
		case *swt.Label:
			out = append(out, w.GetText())
		case *swt.Button:
			out = append(out, w.GetText())
		default:
			out = append(out, nameOf(c))
		}
	}
	return out
}

// pump runs the queued events as the loop would, panics of handlers going to OnError.
func pump(app *App) {
	for app.dispatch(app.display.ReadAndDispatch) {
	}
}

func catchErrors(app *App) *[]string {
	var errs []string
	app.OnError(func(err error) { errs = append(errs, err.Error()) })
	return &errs
}

func TestBoundaryBuildPanic(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		var s shown
		p := put[*Panel](w, Boundary(Failed(s.failed),
			Label("a"),
			Button("b", Init(func(*ButtonView) { panic("init failed") })),
			Label("never made")))
		want := "gowt: panic: init failed"
		if strings.Join(s.errs, "|") != want || strings.Join(*errs, "|") != want {
			t.Errorf("fallback got %q, OnError got %q", s.errs, *errs)
		}
		if got := strings.Join(texts(p), "|"); got != "fallback: "+want {
			t.Errorf("boundary shows %q", got)
		}
		put[*LabelView](w, Label("after")) // the window goes on
	})
}

func TestBoundaryMisfitOption(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		var s shown
		p := put[*Panel](w, Boundary(Failed(s.failed), GrowX(),
			Label("x", Multiline()),
			Label("y", Label("child"))))
		if len(s.errs) != 1 || s.errs[0] != "gowt: Multiline does not apply to Label" {
			t.Errorf("fallback got %q", s.errs)
		}
		if len(*errs) != 2 { // both reach OnError; the fallback shows the first
			t.Errorf("OnError got %q", *errs)
		}
		if got := texts(p); len(got) != 1 {
			t.Errorf("boundary shows %q", got)
		}
		// The boundary's own options belong to its parent and are not caught by it.
		var s2 shown
		p2 := put[*Panel](w, Boundary(Failed(s2.failed), Wrap(), Label("kept")))
		if len(s2.errs) != 0 || strings.Join(texts(p2), "|") != "kept" {
			t.Errorf("own option caught: %q %q", s2.errs, texts(p2))
		}
	})
}

func TestBoundaryEffectPanic(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		n := rx.New(0)
		text := rx.Compute(func() string {
			if n.Get() == 1 {
				panic("bad n")
			}
			return "n ok"
		})
		var s shown
		p := put[*Panel](w, Boundary(Failed(s.failed), Label("", BindText(text))))
		other := put[*LabelView](w, Label("", BindText(rx.Compute(func() string { return "other " + string(rune('0'+n.Get())) }))))
		if got := strings.Join(texts(p), "|"); got != "n ok" {
			t.Fatalf("before: %q", got)
		}
		n.Set(1)
		rx.Flush() // does not panic: the boundary took it
		if got := strings.Join(texts(p), "|"); got != "fallback: gowt: panic: bad n" {
			t.Errorf("after: %q", got)
		}
		if strings.Join(*errs, "|") != "gowt: panic: bad n" || other.l.GetText() != "other 1" {
			t.Errorf("OnError %q, other label %q", *errs, other.l.GetText())
		}
		// reset rebuilds on the next turn, with the state as it is then.
		n.Set(2)
		s.reset()
		pump(app)
		if got := strings.Join(texts(p), "|"); got != "n ok" {
			t.Errorf("after reset: %q", got)
		}
		n.Set(1)
		rx.Flush()
		if len(s.errs) != 2 {
			t.Errorf("the rebuilt children are not caught again: %q", s.errs)
		}
	})
}

func TestBoundaryRowsFill(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		catchErrors(app)
		list := rx.NewFunc([]proc{{1, "a", "0"}}, nil)
		var s shown
		p := put[*Panel](w, Boundary(Failed(s.failed),
			Table(Init(columns), Rows(list, procKey, func(r *TableRow, p proc) {
				if p.name == "bad" {
					panic("fill failed")
				}
				fillProc(r, p)
			}))))
		list.Set([]proc{{1, "a", "0"}, {2, "bad", "0"}})
		rx.Flush()
		if len(s.errs) != 1 || s.errs[0] != "gowt: panic: fill failed" || len(texts(p)) != 1 {
			t.Errorf("fallback %q, shows %q", s.errs, texts(p))
		}
		list.Set([]proc{{3, "c", "0"}})
		s.reset()
		pump(app)
		tbl, ok := p.c.GetChildren()[0].Impl().(*swt.Table)
		if !ok || tbl.GetItemCount() != 1 || tbl.GetItem(0).GetText() != "c" {
			t.Errorf("rebuilt table: %v", texts(p))
		}
	})
}

func TestBoundaryNested(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		catchErrors(app)
		n := rx.New(0)
		var outer, inner shown
		var in *Panel
		p := put[*Panel](w, Boundary(Failed(outer.failed),
			Label("outer child"),
			Boundary(Ref(&in), Failed(inner.failed),
				Label("", BindText(rx.Compute(func() string {
					if n.Get() > 0 {
						panic("inner failed")
					}
					return "inner child"
				}))))))
		n.Set(1)
		rx.Flush()
		if len(inner.errs) != 1 || len(outer.errs) != 0 || texts(p)[0] != "outer child" {
			t.Errorf("inner %q outer %q, outer shows %q", inner.errs, outer.errs, texts(p))
		}
		// A fallback that fails goes to the boundary above.
		var s shown
		put[*Panel](w, Boundary(Failed(s.failed),
			Boundary(Failed(func(error, func()) Option { panic("fallback failed") }),
				Label("x", Init(func(*LabelView) { panic("child failed") })))))
		if len(s.errs) != 1 || s.errs[0] != "gowt: panic: fallback failed" {
			t.Errorf("outer of a failed fallback got %q", s.errs)
		}
		if in == nil {
			t.Error("Ref on a Boundary did not set the panel")
		}
	})
}

// After resets, one Set runs one binding and the effects of Init live once.
func TestBoundaryResetDoesNotLeak(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		catchErrors(app)
		n := rx.New(0)
		live, applied := 0, 0
		var s shown
		p := put[*Panel](w, Boundary(Failed(s.failed),
			Label("", Init(func(l *LabelView) {
				rx.Effect(func() {
					n.Get()
					live++
					rx.OnCleanup(func() { live-- })
				})
				Watch(l, n, func(v int) {
					applied++
					if v < 0 {
						panic("negative")
					}
				})
			}))))
		for i := range 3 {
			n.Set(-1 - i)
			rx.Flush()
			n.Set(i)
			s.reset()
			pump(app)
		}
		if len(s.errs) != 3 || live != 1 || len(texts(p)) != 1 {
			t.Fatalf("errors %d, live effects %d, shows %q", len(s.errs), live, texts(p))
		}
		applied = 0
		n.Set(10)
		rx.Flush()
		if applied != 1 || live != 1 {
			t.Errorf("one Set ran the binding %d times, live effects %d", applied, live)
		}
		p.Dispose()
		if live != 0 || boundaries != 0 {
			t.Errorf("after dispose: live effects %d, boundaries %d", live, boundaries)
		}
	})
}

// Add on a container inside a boundary is caught; an event handler is not.
func TestBoundaryAddAndHandler(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		errs := catchErrors(app)
		var s shown
		var box *Panel
		var btn *ButtonView
		p := put[*Panel](w, Boundary(Failed(s.failed),
			VBox(Ref(&box)),
			Button("b", Ref(&btn), OnClick(func() { panic("click failed") }))))
		app.Async(func() { notify(btn.Unwrap(), swt.Selection) }) // in the loop, as a click comes
		pump(app)
		if len(s.errs) != 0 || len(*errs) != 1 || len(texts(p)) != 2 {
			t.Errorf("handler: fallback %q, OnError %q", s.errs, *errs)
		}
		box.Add(Label("x", Init(func(*LabelView) { panic("add failed") })))
		if len(s.errs) != 1 || s.errs[0] != "gowt: panic: add failed" || len(texts(p)) != 1 {
			t.Errorf("Add: fallback %q, shows %q", s.errs, texts(p))
		}
	})
}

// The default fallback is the message and a Retry button that rebuilds.
func TestBoundaryDefaultFallback(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		catchErrors(app)
		ok := false
		p := put[*Panel](w, Boundary(Label("x", Init(func(*LabelView) {
			if !ok {
				panic("not yet")
			}
		}))))
		box, isBox := p.c.GetChildren()[0].Impl().(*swt.Composite)
		if !isBox {
			t.Fatalf("default fallback %q", texts(p))
		}
		got := texts(&Panel{newPanel(box)})
		if strings.Join(got, "|") != "gowt: panic: not yet|Retry" {
			t.Fatalf("default fallback shows %q", got)
		}
		ok = true
		notify(box.GetChildren()[1].Impl().(*swt.Button), swt.Selection)
		pump(app)
		if strings.Join(texts(p), "|") != "x" {
			t.Errorf("after Retry: %q", texts(p))
		}
	})
}
