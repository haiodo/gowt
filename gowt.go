// Package gowt is an idiomatic Go facade over the generated swt package: no Java-shaped
// overloads, listeners are plain funcs, errors come back from Run. Unwrap on any widget
// returns the underlying swt object for the full API.
package gowt

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread; pinning in init keeps the main goroutine there.
func init() { runtime.LockOSThread() }

// App is the handle to the UI thread, valid inside Run.
type App struct {
	display *swt.Display
	quit    atomic.Bool
	images  []*Image
	icons   map[iconKey]*Image
}

// Run creates the display, calls setup on the UI thread, then runs the event loop until the
// last window is closed or App.Quit is called. SWT exceptions raised on the UI thread
// (panics carrying an error, e.g. *swt.SWTException) are returned; runtime errors and other panics propagate.
func Run(setup func(*App)) (err error) {
	// The display belongs to its OS thread (Win32 message queue, GTK); a goroutine must not migrate off it.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer catch(&err)
	a := &App{display: swt.NewDisplay()}
	defer a.display.Dispose()
	defer a.disposeImages()
	a.display.FollowSystemTheme()
	setup(a)
	for !a.quit.Load() && len(a.display.GetShells()) > 0 {
		if !a.display.ReadAndDispatch() {
			a.display.Sleep()
		}
	}
	return nil
}

// catch is deferred directly: it turns a panic carrying an SWT error into *err.
func catch(err *error) {
	if r := recover(); r != nil {
		e, ok := r.(error)
		if _, rt := r.(runtime.Error); !ok || rt {
			panic(r)
		}
		*err = fmt.Errorf("gowt: %w", e)
	}
}

// Async queues f on the UI thread; safe from any goroutine.
func (a *App) Async(f func()) { a.display.AsyncExec(jrt.NewRunnable(f)) }

// Sync runs f on the UI thread and waits; safe from any goroutine except while the UI thread is blocked on you.
func (a *App) Sync(f func()) { a.display.SyncExec(jrt.NewRunnable(f)) }

// After runs f on the UI thread once, after d.
func (a *App) After(d time.Duration, f func()) {
	a.display.TimerExec(int32(d.Milliseconds()), jrt.NewRunnable(f))
}

// Quit ends the event loop after the current event; safe from any goroutine.
func (a *App) Quit() {
	a.quit.Store(true)
	a.display.Wake()
}

// Unwrap returns the underlying display.
func (a *App) Unwrap() *swt.Display { return a.display }

// Option adjusts a widget at construction: style bits are OR-ed before creation, apply runs after.
type Option struct {
	style int32
	apply func(*swt.Control)
}

// Cell sets the GridData of a widget laid out by Grid.
func Cell(c GridCell) Option {
	return Option{apply: func(w *swt.Control) { w.SetLayoutData(c.data()) }}
}

// Disabled creates the widget disabled.
func Disabled() Option { return Option{apply: func(w *swt.Control) { w.SetEnabled(false) }} }

// Tooltip sets the hover text.
func Tooltip(s string) Option {
	return Option{apply: func(w *swt.Control) { w.SetToolTipText(s) }}
}

// Check makes a Button a check box, and Table and Tree rows and ToolBar items checkable.
// Radio, Multiline, Password, ReadOnly and Border follow below.
func Check() Option { return Option{style: swt.CHECK} }

// Radio makes a Button a radio button.
func Radio() Option { return Option{style: swt.RADIO} }

// Multiline makes a Text multi-line, wrapped, with a vertical scroll bar.
func Multiline() Option { return Option{style: swt.MULTI | swt.WRAP | swt.V_SCROLL} }

// Password masks the characters typed into a Text.
func Password() Option { return Option{style: swt.PASSWORD} }

// ReadOnly makes a Text or Combo non-editable.
func ReadOnly() Option { return Option{style: swt.READ_ONLY} }

// Border draws a frame; Table and Tree have one by default.
func Border() Option { return Option{style: swt.BORDER} }

func resolve(base int32, opts []Option) int32 {
	for _, o := range opts {
		base |= o.style
	}
	return base
}

func applyOpts(c *swt.Control, opts []Option) {
	for _, o := range opts {
		if o.apply != nil {
			o.apply(c)
		}
	}
}

// Panel is a plain container; Window, Group, Split and CoolBar embed its methods.
type Panel struct{ *panel }

// panel holds the container methods. They are declared on an unexported type so that godoc lists
// them for every embedding type while no exported field named Panel hides the Panel method.
type panel struct {
	c     *swt.Composite
	stack *swt.StackLayout
}

// Unwrap returns the underlying composite.
func (p *panel) Unwrap() *swt.Composite { return p.c }

// SetLayout installs l and returns p.
func (p *panel) SetLayout(l Layout) *Panel {
	sl := l.layout()
	p.stack, _ = sl.(*swt.StackLayout)
	p.c.SetLayout(sl)
	return &Panel{p}
}

// Panel adds a nested container.
func (p *panel) Panel(opts ...Option) *Panel {
	c := swt.NewCompositeParentStyle(p.c, resolve(swt.NONE, opts))
	applyOpts(&c.Control, opts)
	return &Panel{&panel{c: c}}
}

// Label adds a static text.
func (p *panel) Label(text string, opts ...Option) *Label {
	l := swt.NewLabel(p.c, resolve(swt.NONE, opts))
	l.SetText(text)
	applyOpts(&l.Control, opts)
	return &Label{l}
}

// Button adds a push button (Check or Radio for other kinds). onClick may be nil.
func (p *panel) Button(text string, onClick func(), opts ...Option) *Button {
	b := swt.NewButton(p.c, resolve(swt.NONE, opts))
	b.SetText(text)
	applyOpts(&b.Control, opts)
	w := &Button{b: b}
	if onClick != nil {
		w.OnClick(onClick)
	}
	return w
}

// Text adds an edit field.
func (p *panel) Text(opts ...Option) *Text {
	t := swt.NewText(p.c, resolve(swt.NONE, opts))
	applyOpts(&t.Control, opts)
	return &Text{t}
}

// Window is a top-level window.
type Window struct {
	*panel
	shell *swt.Shell
	sized bool
}

// Window creates a hidden window; call Show once its content is built.
func (a *App) Window(title string, opts ...Option) *Window {
	s := swt.NewShellDisplayStyle(a.display, resolve(swt.SHELL_TRIM, opts))
	s.SetText(title)
	return &Window{panel: &panel{c: &s.Composite}, shell: s}
}

// Show packs the window to its content size (unless SetSize was called) and opens it.
func (w *Window) Show() {
	if !w.sized {
		w.shell.Pack()
	}
	w.shell.Open()
}

// SetSize sets the window size in points.
func (w *Window) SetSize(width, height int) {
	w.sized = true
	w.shell.SetSize(int32(width), int32(height))
}

// Close closes the window as the user would, running OnClose.
func (w *Window) Close() { w.shell.Close() }

// OnClose registers f; returning false keeps the window open.
func (w *Window) OnClose(f func() bool) {
	w.shell.AddShellListener(swt.ShellListenerShellClosedAdapter(func(e *swt.ShellEvent) {
		if !f() {
			e.Doit = false
		}
	}))
}

// Unwrap returns the underlying shell.
func (w *Window) Unwrap() *swt.Shell { return w.shell }

// Label is a static text.
type Label struct{ l *swt.Label }

// SetText replaces the label text.
func (l *Label) SetText(s string) { l.l.SetText(s) }

// Unwrap returns the underlying swt.Label for the full API.
func (l *Label) Unwrap() *swt.Label { return l.l }

// Button is a push, check or radio button.
type Button struct{ b *swt.Button }

// SetText replaces the button text.
func (b *Button) SetText(s string) { b.b.SetText(s) }

// Checked reports the state of a Check or Radio button.
func (b *Button) Checked() bool { return b.b.GetSelection() }

// SetChecked sets the state of a Check or Radio button.
func (b *Button) SetChecked(v bool) { b.b.SetSelection(v) }

// Unwrap returns the underlying swt.Button for the full API.
func (b *Button) Unwrap() *swt.Button { return b.b }

// OnClick runs f when the button is pressed (or toggled).
func (b *Button) OnClick(f func()) {
	b.b.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) { f() }))
}

// Text is an edit field.
type Text struct{ t *swt.Text }

// Text returns the current content.
func (t *Text) Text() string { return t.t.GetText() }

// SetText replaces the content.
func (t *Text) SetText(s string) { t.t.SetText(s) }

// SetHint sets the placeholder shown while the field is empty.
func (t *Text) SetHint(s string) { t.t.SetMessage(s) }

// Unwrap returns the underlying swt.Text for the full API.
func (t *Text) Unwrap() *swt.Text { return t.t }

// OnChange runs f with the new content after every edit.
func (t *Text) OnChange(f func(text string)) {
	t.t.AddModifyListener(&modifier{func() { f(t.t.GetText()) }})
}

// OnActivate runs f when the user presses Enter in a single-line field.
func (t *Text) OnActivate(f func()) {
	onActivate(t.t, func(*swt.SelectionEvent) { f() })
}

type modifier struct{ f func() }

func (m *modifier) ModifyText(*swt.ModifyEvent) { m.f() }
