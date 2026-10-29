package gowt

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread; pinning in init keeps the main goroutine there.
func init() { runtime.LockOSThread() }

// App is the handle to the UI thread, valid inside Run.
type App struct {
	display    *swt.Display
	quit       atomic.Bool
	ended      atomic.Bool // the event loop is over: Display release still runs queued callbacks, on disposed widgets
	images     map[*Image]struct{}
	fonts      map[Font]*sharedFont
	pxFonts    map[*swt.Font]float64 // the Px of the native fonts made from one, to keep their size on a later SetFont
	bodyPx     float64               // the body size SetDensity chose; 0 keeps the system font
	body       *Font                 // the font new widgets get when it differs from the system font
	clip       *Clipboard
	onError    func(error)
	onQuit     []func() bool
	open       openBox
	urls       openBox
	openPend   []string  // paths and URLs of this turn of the event loop, see openIn
	windows    []*Window // in creation order, asked in turn when the app quits
	asking     bool      // a quit question is open: a second request is refused
	approved   bool      // the quit was answered yes: windows closing now are not asked again
	dark       bool      // the theme the Adaptive colours were last applied for
	appearance Appearance
	paints     map[*swt.Control]*painted // controls given an Adaptive colour
	// controls whose preferred size changed in this event-loop turn; see sizing
	relayouts []*swt.Control
	badge     string
	link      linkStyle
	jobs      jobSet // Job, Track and short After timers, see jobs.go
}

// SetAppName sets the name the system shows for the process (macOS menu bar and Dock, Linux
// WM class). Call it before Run.
func SetAppName(name string) { swt.DisplaySetAppName(name) }

// Run creates the display, calls setup on the UI thread, then runs the event loop until the
// last window is closed or App.Quit is called. An SWT exception raised by setup (a panic carrying an
// error, e.g. *swt.SWTException) is returned; a runtime error in setup propagates. Errors and panics
// in event handlers, Async and After callbacks do not end the loop: they go to App.OnError.
//
// With GOWT_SNAP=<file.png> in the environment Run writes the first window as a PNG after the
// loop has run for GOWT_SNAP_DELAY milliseconds (default 800), then returns; see docs/screenshots.md.
func Run(setup func(*App)) (err error) {
	// The display belongs to its OS thread (Win32 message queue, GTK); a goroutine must not migrate off it.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer catch(&err)
	a := &App{display: swt.NewDisplay(), images: map[*Image]struct{}{}, fonts: map[Font]*sharedFont{}, pxFonts: map[*swt.Font]float64{}}
	defer a.display.Dispose()
	defer a.ended.Store(true)
	a.jobs.init()
	defer a.jobs.close()
	toOnError := func(err error) { a.fail(err) }
	a.display.SetRuntimeExceptionHandler(toOnError)
	a.display.SetErrorHandler(toOnError)
	defer a.disposeImages()
	defer a.disposeClipboard()
	apps.Store(a.display, a)
	a.display.AddListener(swt.Close, quitListener(func() bool { return a.askQuit(nil) }))
	a.hookTerminate()
	a.display.AddListener(swt.OpenDocument, openListener(a.openIn))
	a.display.AddListener(swt.OpenUrl, openListener(a.openIn))
	defer a.attachInbox()()
	defer apps.Delete(a.display)
	// Deferred last, so rx stops waking the loop before the display is disposed.
	defer rx.Attach(func(tick func()) func() {
		r := jrt.NewRunnable(tick)
		return func() { a.display.AsyncExec(r) }
	})()
	installStyle()
	a.display.FollowSystemTheme()
	a.followTheme()
	densityChosen, densityFactor = Density{}, 1
	Type, _, _, _ = resolveTokens(a.systemInfo(), DensityNative)
	setup(a)
	if args := argOpen(os.Args[1:]); len(args) > 0 {
		a.openIn(args...)
	}
	snapTick := func() {}
	if p := os.Getenv("GOWT_SNAP"); p != "" {
		snapTick = a.armSnap(p, &err)
	}
	qaStart(a)
	// The display can dispose itself inside ReadAndDispatch (Windows WM_ENDSESSION); a macOS quit sets a.quit.
	for !a.quit.Load() && !a.display.IsDisposed() && len(a.display.GetShells()) > 0 {
		snapTick()
		if !a.dispatch(a.display.ReadAndDispatch) {
			a.display.Sleep()
		}
	}
	return err
}

// apps lets a widget find its App from its display (drop callbacks create App-owned images).
var apps sync.Map // *swt.Display -> *App

// appOf returns the App running c's display, nil outside Run.
func appOf(c displayed) *App {
	if a, ok := apps.Load(c.GetDisplay()); ok {
		return a.(*App)
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

// dispatch runs one step of the event loop. SWT hands errors raised in listeners, Async and After
// to the display's handlers (fail); dispatch catches the panics that get past them, such as a string.
func (a *App) dispatch(step func() bool) (busy bool) {
	defer func() {
		if r := recover(); r != nil {
			a.fail(r)
			busy = true
		}
	}()
	return step()
}

// OnError sets f to receive the errors and panics raised on the UI thread by event handlers, Async,
// Sync and After callbacks, and placement options that do not fit the parent's layout. The event loop
// goes on after f returns. Without OnError each one is logged with the stack where it was raised.
func (a *App) OnError(f func(error)) { a.onError = f }

// fail reports a panic value caught on the UI thread, wrapped like the error Run returns.
func (a *App) fail(r any) { a.report(panicError(r)) }

func (a *App) report(err error) {
	if a.onError != nil {
		a.onError(err)
		return
	}
	log.Printf("%v\n%s", err, debug.Stack())
}

// displayed is any swt widget: it knows its display, and so its App.
type displayed interface{ GetDisplay() *swt.Display }

// reportOn hands err to the App that owns c, see App.OnError, and to the Boundary creating c.
func reportOn(c displayed, err error) {
	a, ok := apps.Load(c.GetDisplay())
	if !ok {
		panic(err)
	}
	a.(*App).report(err)
	if ctl, ok := c.(*swt.Control); ok {
		if b := buildingAt(ctl); b != nil {
			b.record(err)
		}
	}
}

// Async queues f on the UI thread; safe from any goroutine. Once Run has returned, Async does nothing
// and f never runs; f queued while Run is ending may not run either.
func (a *App) Async(f func()) {
	defer ignoreDisposed()
	a.display.AsyncExec(jrt.NewRunnable(a.live(f)))
}

// Sync runs f on the UI thread and waits; safe from any goroutine except while the UI thread is blocked on you.
// Once Run has returned, Sync returns at once without running f.
func (a *App) Sync(f func()) {
	defer ignoreDisposed()
	a.display.SyncExec(jrt.NewRunnable(a.live(f)))
}

// live skips f once the event loop is over.
func (a *App) live(f func()) func() {
	return func() {
		if !a.ended.Load() {
			f()
		}
	}
}

// ignoreDisposed swallows the exception that a call on the display raises after Run disposed it.
func ignoreDisposed() {
	r := recover()
	if r == nil {
		return
	}
	if e, ok := r.(*swt.SWTException); !ok || e.Code != swt.ERROR_DEVICE_DISPOSED {
		panic(r)
	}
}

// After runs f on the UI thread once, after d. A timer under one second counts in Pending until it
// fires or is cancelled; a longer one does not. The returned cancel stops f from running; it is safe from
// any goroutine, safe to call after f ran, and calling it again does nothing.
func (a *App) After(d time.Duration, f func()) (cancel func()) {
	var done atomic.Bool
	short := d < shortTimer
	if short {
		a.jobs.timers.Add(1)
	}
	r := jrt.NewRunnable(nil)
	r.Fn = a.live(func() {
		if !done.CompareAndSwap(false, true) {
			return
		}
		if short {
			a.jobs.timers.Add(-1)
		}
		f()
	})
	a.display.TimerExec(int32(d.Milliseconds()), r)
	return func() {
		if !done.CompareAndSwap(false, true) {
			return
		}
		if short {
			a.jobs.timers.Add(-1)
		}
		a.Async(func() { a.display.TimerExec(-1, r) })
	}
}

// Every runs f on the UI thread every d, the first time after d, until stop is called. stop is safe
// from any goroutine; called on the UI thread, f included, it keeps f from running again.
func (a *App) Every(d time.Duration, f func()) (stop func()) {
	var stopped atomic.Bool
	ms := int32(d.Milliseconds())
	r := jrt.NewRunnable(nil)
	r.Fn = func() {
		if stopped.Load() {
			return
		}
		a.display.TimerExec(ms, r) // before f, so that a panic in f does not end the ticks
		if !a.ended.Load() {
			f()
		}
	}
	a.display.TimerExec(ms, r)
	return func() { stopped.Store(true) }
}

// Flush applies pending changes now instead of on the next event-loop turn: it runs the stale rx
// effects and the layouts that bindings put off, such as BindText's. For timing one update or measuring
// widgets right after a change. UI thread only.
func (a *App) Flush() {
	rx.Flush()
	a.runRelayouts()
	a.display.RunDeferredLayouts()
}

// Pending counts the work the app has not finished: stale rx effects, posted values and layouts that
// bindings put off (Flush applies those), running Jobs and Tracks that are not BackgroundJob, and
// After timers under one second. Zero means nothing is left to wait for. UI thread only.
func (a *App) Pending() int { return rx.Pending() + len(a.relayouts) + a.outstanding() }

// ScreenSize returns the size in points of the primary monitor's area for windows, without the
// menu bar, Dock or taskbar.
func (a *App) ScreenSize() (width, height int) {
	r := a.display.GetPrimaryMonitor().GetClientArea()
	return int(r.Width), int(r.Height)
}

// Quit ends the event loop after the current event; safe from any goroutine, a no-op after Run.
func (a *App) Quit() {
	defer ignoreDisposed()
	a.quit.Store(true)
	a.display.Wake()
}

// OnQuit registers f, asked before the app quits in every way but Quit: Cmd+Q, the app menu and the
// Dock on macOS, the end of the session (logout, shutdown) on macOS and Windows, and the close of
// the last window. false cancels the quit. Several calls add several handlers, asked in order;
// the first false stops the rest.
//
// On a quit that is not the close of the last window, the open windows are asked first, in creation
// order, through their OnClose handlers: the first refusal cancels the quit and no window or OnQuit
// handler after it is asked. No window closes before every answer is yes. Closing the last window
// asks that window's OnClose, then OnQuit.
//
// f and the OnClose handlers may show a dialog (Window.MessageBox, Confirm): the question runs
// on the UI thread inside the system's quit request. To ask from your own Quit item call
// app.Unwrap().Close(), which does the same; app.Quit ends the loop without asking.
func (a *App) OnQuit(f func() bool) { a.onQuit = append(a.onQuit, f) }

// OnOpenFiles registers f, called on the UI thread with the files the system or another process asks
// the app to open: Finder (double click, "Open With", a drop on the Dock icon) on macOS, the command
// line on every OS (Explorer and file managers start the program with the file as argument), and,
// with SingleInstance, a second launch forwarding its arguments to the running app. Paths are
// absolute and without duplicates within one call; URLs other than file: go to OnOpenURLs. Files that
// arrive before the first handler (a launch by double click) are kept and delivered to it once it is
// set. Several calls add several handlers. Users of Unwrap can still add an swt.OpenDocument
// listener. See docs/events.md and docs/packaging.md.
func (a *App) OnOpenFiles(f func(paths []string)) {
	a.open.handlers = append(a.open.handlers, f)
	if q := a.open.queue; len(q) > 0 {
		a.open.queue = nil
		a.openIn(q...)
	}
}

// OnOpenURLs registers f, called on the UI thread with the URLs the system or another process asks
// the app to open, of any scheme but file: (those go to OnOpenFiles): a link to a scheme the app
// registers, such as myapp://, on macOS (CFBundleURLTypes in Info.plist, at launch and while running),
// command-line arguments of the form scheme:... on every OS (a .desktop file with Exec=app %U and
// MimeType=x-scheme-handler/myapp on Linux), and, with SingleInstance, the URLs of a second launch
// forwarded to the running app. URLs come as given, without duplicates within one call. URLs that
// arrive before the first handler (a launch by link) are kept and delivered to it once it is set.
// Several calls add several handlers. Users of Unwrap can still add an swt.OpenUrl listener on macOS.
// See docs/events.md.
func (a *App) OnOpenURLs(f func(urls []string)) {
	a.urls.handlers = append(a.urls.handlers, f)
	if q := a.urls.queue; len(q) > 0 {
		a.urls.queue = nil
		a.openIn(q...)
	}
}

type quitListener func() bool

func (l quitListener) HandleEvent(e *swt.Event) {
	if !l() {
		e.Doit = false
	}
}

// askQuit asks every window and then the OnQuit handlers, skipping except (the window whose close
// started the question).
func (a *App) askQuit(except *Window) bool {
	if a.asking {
		return false
	}
	a.asking = true
	defer func() { a.asking = false }()
	live := a.windows[:0]
	for _, w := range a.windows {
		if !w.shell.IsDisposed() {
			live = append(live, w)
		}
	}
	a.windows = live
	for _, w := range append([]*Window(nil), live...) {
		if w == except || w.shell.IsDisposed() {
			continue
		}
		e := swt.NewEvent()
		w.shell.NotifyListeners(swt.Close, e)
		if !e.Doit {
			return false
		}
	}
	for _, f := range a.onQuit {
		if !f() {
			return false
		}
	}
	return true
}

// Unwrap returns the underlying display.
func (a *App) Unwrap() *swt.Display { return a.display }

// Disabled creates the widget disabled.
func Disabled() Option { return apply(func(w *swt.Control) { w.SetEnabled(false) }) }

// Tooltip sets the hover text.
func Tooltip(s string) Option {
	return apply(func(w *swt.Control) { w.SetToolTipText(s) })
}

// Focused gives the widget the keyboard focus when its window is first shown, in place of the first
// control that SWT and the OS pick (with Full Keyboard Access on, macOS draws the focus ring on that
// control, often a button). Created into a window already on screen, the widget takes the focus on
// the next event-loop turn, or when that window is next activated. With several Focused in a window
// the last one created wins.
func Focused() Option {
	return apply(func(c *swt.Control) {
		s := c.GetShell()
		if c == &s.Control {
			reportOn(c, fmt.Errorf("gowt: Focused applies to a widget in a window, not to Window"))
			return
		}
		if !s.GetVisible() {
			s.SetDataKeyValue(focusedKey, c)
			return
		}
		// On the next turn the widget is laid out; focusing an inactive window would bring it to the front.
		c.GetDisplay().AsyncExec(jrt.NewRunnable(func() {
			if c.IsDisposed() {
				return
			}
			if c.GetDisplay().GetActiveShell() == s {
				c.SetFocus()
				return
			}
			s.SetSavedFocus(c)
		}))
	})
}

// Check makes a Button a check box, and Table and Tree rows and ToolBar items checkable.
// Radio, Multiline, Password, ReadOnly and Border follow below.
func Check() Option { return style{id: sCheck, set: swt.CHECK} }

// Radio makes a Button a radio button.
func Radio() Option { return style{id: sRadio, set: swt.RADIO} }

// Toggle makes a Button a toggle button, which stays pressed until it is clicked again (Wrap, Clip):
// Checked, Bind (bool) and OnChange (bool) read and set its state. macOS: a push-on-push-off NSButton,
// with Borderless a symbol in the accent colour while on; GTK: GtkToggleButton; Windows: the WinUI
// ToggleButton, with the accent fill while it is on.
func Toggle() Option { return style{id: sToggle, set: swt.TOGGLE} }

// Multiline makes a Text multi-line and wrapped. macOS and GTK scroll with an overlay bar that shows on overflow;
// Windows has no scroll bar unless Scrollbars() asks for one.
func Multiline() Option { return style{id: sMultiline, set: swt.MULTI | swt.WRAP | multilineScroll} }

// Password masks the characters typed into a Text.
func Password() Option { return style{id: sPassword, set: swt.PASSWORD} }

// ReadOnly makes a Text or Combo non-editable.
func ReadOnly() Option { return style{id: sReadOnly, set: swt.READ_ONLY} }

// Border draws a frame; Table and Tree have one by default.
func Border() Option { return style{id: sBorder, set: swt.BORDER} }

// CellSelect makes a Tree select only the text of a node, as v0.1.0 did on Windows and Linux; then
// CellAt and OnCellClick find a node only where its text or icon is.
func CellSelect() Option { return style{id: sCellSelect, clear: swt.FULL_SELECTION} }

// NoBorder removes the frame that List, Table and Tree have by default, and the one Border gives a
// Text or StyledText; it wins over Border in either order.
func NoBorder() Option { return style{id: sNoBorder, clear: swt.BORDER} }

// Search makes a Text a search field with a magnifier icon and a clear button.
func Search() Option {
	return style{id: sSearch, set: swt.SEARCH | swt.ICON_SEARCH | swt.ICON_CANCEL}
}

// FieldFill gives a single-line Text or Number a rounded fill instead of the native bezel: the text,
// caret, focus ring and Prefix stay native, and Radius rounds the fill. Pair it with Inset for the
// padding of the text.
func FieldFill(c Paint) Option {
	return hook(func(w Widget) error {
		t, err := fieldOf(w, "FieldFill")
		if err != nil {
			return err
		}
		t.fill = c
		return nil
	})
}

// FieldStroke draws a border of paint around the FieldFill of a single-line field; RGB.Alpha sets a
// translucent border such as the 25% black of an inspector field.
func FieldStroke(c Paint) Option {
	return hook(func(w Widget) error {
		t, err := fieldOf(w, "FieldStroke")
		if err != nil {
			return err
		}
		t.stroke = c
		return nil
	})
}

// Radius is the corner radius in points of the fill FieldFill gives a field.
func Radius(points int) Option {
	return hook(func(w Widget) error {
		t, err := fieldOf(w, "Radius")
		if err != nil {
			return err
		}
		t.radius = points
		return nil
	})
}

// fieldOf is the single-line field of a Text or Number.
func fieldOf(w Widget, name string) (*TextView, error) {
	switch v := w.(type) {
	case *TextView:
		return v, nil
	case *NumberView:
		return v.f, nil
	}
	return nil, fmt.Errorf("gowt: %s applies to Text or Number, not to %s", name, nameOf(w.control()))
}

// Background sets the background colour of a widget, as SetBackground does.
func Background(c Paint) Option { return apply(func(w *swt.Control) { setPaint(w, c, false) }) }

// Foreground sets the text colour of a widget, as SetForeground does.
func Foreground(c Paint) Option { return apply(func(w *swt.Control) { setPaint(w, c, true) }) }

// Typeface gives a widget the font f, as SetFont does.
func Typeface(f Font) Option { return hook(func(w Widget) error { w.SetFont(f); return nil }) }

// Custom runs f on the widget right after it is created and sets no style bits. It lets a package outside gowt adjust the swt control.
func Custom(f func(*swt.Control)) Option { return apply(f) }

// Panel is a container: VBox, HBox, Grid, Flow, Fill, Form and Tab create one. Window, Group, Split
// and CoolBar embed its methods.
type Panel struct{ *panel }

// panel holds the container methods. They are declared on an unexported type so that godoc lists
// them for every embedding type while no exported field named Panel hides the Panel method.
type panel struct {
	base
	c *swt.Composite
}

// Unwrap returns the underlying composite.
func (p *panel) Unwrap() *swt.Composite { return p.c }

// AsComposite returns the underlying composite; unlike Unwrap it has the same type on Panel,
// Window, Group, Split and CoolBar, so one function can take any of them.
func (p *panel) AsComposite() *swt.Composite { return p.c }

// Add creates children at the end of p, in order, and lays p out again when it is on screen,
// measuring only the new children. It is the way to fill a container after its creation, for
// content that changes; see Clear.
func (p *panel) Add(children ...Option) {
	old := p.c.GetChildren()
	add(p, children)
	if p.c.IsDisposed() {
		return
	}
	flushUp(&p.c.Control)
	if p.c.IsVisible() {
		layoutAdded(p.c, old)
	}
}

// layoutAdded lays c out after Add: only the children missing from old are measured, the others
// keep the sizes their layouts cached, so adding to a long list costs the new rows only.
func layoutAdded(c *swt.Composite, old []*swt.Control) {
	// A setter's size change is laid out on the next turn: drop its stale cached size now.
	if a := appOf(&c.Control); a != nil {
		for _, k := range a.relayouts {
			if k != nil && !k.IsDisposed() {
				flushUp(k)
			}
		}
	}
	had := make(map[*swt.Control]bool, len(old))
	for _, k := range old {
		had[k] = true
	}
	var added []*swt.Control
	for _, k := range c.GetChildren() {
		if !had[k] {
			added = append(added, k)
		}
	}
	c.LayoutOverload4(added, swt.NONE)
}

// flushUp drops the sizes the layouts of c's ancestors cached for c and for each other, after a change
// of c's content; their next layout, partial ones included, measures c again.
func flushUp(c *swt.Control) {
	sh := &c.GetShell().Control
	for c != sh {
		p := c.GetParent()
		if l := p.GetLayout(); l != nil {
			l.FlushCache(c)
		}
		c = &p.Control
	}
}

func (p *panel) into() *swt.Composite { return p.c }
func (p *panel) added(Widget)         {}

// Relayout lays the children out again and recomputes their sizes, as needed after a change made
// through Unwrap, such as swt.Label.SetText. The facade setters that change a size (SetText, SetImage,
// SetFont, SetTitle of a Group, SetItems of a Combo) lay out by themselves. p keeps its own size: to
// let it grow too, Relayout the Window.
func (p *panel) Relayout() {
	refitUnder(p.c)
	flushUp(&p.c.Control)
	p.c.LayoutOverload2(true, true)
}

// Clear disposes every child and lays p out again, ready for new content.
func (p *panel) Clear() {
	for _, ch := range p.c.GetChildren() {
		ch.Dispose()
	}
	flushUp(&p.c.Control)
	p.c.Layout()
}

func newPanel(c *swt.Composite) *panel { return &panel{base: base{&c.Control}, c: c} }

// Label creates a static text.
// Options: Wrap, Center, Right, Border, BindText.
func Label(text string, opts ...Option) Option {
	return leaf("Label", sWrap|sCenter|sRight|sBorder, opts, func(p *swt.Composite) Widget {
		l := swt.NewLabel(p, resolve(swt.NONE, opts))
		l.SetText(text)
		return &LabelView{base{&l.Control}, l}
	})
}

// Spacer creates an empty cell that takes the free space along a VBox or an HBox, or the free width
// of a Grid row, pushing the cells after it to the end. Elsewhere it is an empty Label.
// Options: common only.
func Spacer(opts ...Option) Option {
	return &node{"Spacer", func(h host) Widget {
		l := swt.NewLabel(h.into(), swt.NONE)
		w := &LabelView{base{&l.Control}, l}
		switch layoutName(h.into()) {
		case "VBox":
			finish(w, 0, append([]Option{GrowY()}, opts...))
		case "HBox", "Grid":
			finish(w, 0, append([]Option{GrowX()}, opts...))
		default:
			finish(w, 0, opts)
		}
		return w
	}}
}

// Separator creates a line across the width of its cell, or along its height with Vertical().
// Span(n) stretches it over a Grid row of n columns.
// Options: Vertical.
func Separator(opts ...Option) Option {
	return &node{"Separator", func(h host) Widget {
		style := resolve(swt.SEPARATOR, opts)
		grow := GrowY()
		if style&swt.VERTICAL == 0 {
			style |= swt.HORIZONTAL
			grow = GrowX()
		}
		l := swt.NewLabel(h.into(), style)
		w := &LabelView{base{&l.Control}, l}
		o := opts
		if isGrid(layoutName(h.into())) {
			o = append([]Option{grow}, opts...)
		}
		finish(w, sVertical, o)
		return w
	}}
}

// Button creates a push button (Check, Radio or Toggle for other kinds); OnClick gives it an action.
// Options: Check, Radio, Toggle, Small, OnClick, OnChange and Checked (Check, Radio, Toggle), Bind (bool), BindText.
func Button(text string, opts ...Option) Option {
	opts = controlHeight(opts)
	return leaf("Button", sCheck|sRadio|sToggle, opts, func(p *swt.Composite) Widget {
		b := swt.NewButton(p, resolve(swt.NONE, opts))
		b.SetText(text)
		if b.GetStyle()&swt.TOGGLE != 0 {
			styleToggle(b)
		}
		return &ButtonView{base{&b.Control}, b}
	})
}

// Switch creates an on/off toggle for a setting that takes effect at once. It is a check box on every
// OS, the same control as Button(text, Check()), which reads as one choice of a form applied later:
// SWT has no switch control, and NSSwitch on macOS or GtkSwitch on Linux would need a view that the
// toolkit does not create.
// Options: OnChange, Checked, Bind (bool), BindText.
func Switch(text string, opts ...Option) Option {
	return leaf("Switch", 0, opts, func(p *swt.Composite) Widget {
		b := swt.NewButton(p, resolve(swt.CHECK, opts))
		b.SetText(text)
		return &SwitchView{base{&b.Control}, b}
	})
}

// Text creates an edit field.
// Options: Multiline, Password, ReadOnly, Search, Border, NoBorder, Scrollbars, Hint, Prefix, PrefixImage, Suffix, Small, Value, OnChange, OnActivate, Bind (string).
// Windows 11: a Text has the WinUI 3 text box frame (4 px radius, hairline, accent line on focus) unless NoBorder.
func Text(opts ...Option) Option {
	noBorder := hasStyle(opts, sNoBorder)
	opts = append(opts[:len(opts):len(opts)], lateHook(func(w Widget) error {
		fieldLook(w.(*TextView), noBorder)
		return nil
	}))
	return leaf("Text", sMultiline|sPassword|sReadOnly|sSearch|sBorder|sNoBorder|sScrollbars, opts, func(p *swt.Composite) Widget {
		t := swt.NewText(p, resolve(swt.NONE, opts))
		textScrollOnOverflow(t)
		textThemeBackground(t)
		return &TextView{base: base{&t.Control}, t: t}
	})
}

// Window is a top-level window. Its children stack as in a VBox, with Padding(Spacing.PageMargin)
// unless Padding says otherwise.
type Window struct {
	*panel
	shell *swt.Shell
	sized bool
	inset int // top inset requested by SetTopInset
	done  int // part of inset already in the top margin

	onClose []func() bool
}

// SetTopInset adds points to the top margin of the window layout on top of its Padding.
// look.SetFullSizeContent uses it to keep the content below the title bar and the toolbar.
func (w *Window) SetTopInset(points int) {
	w.inset = points
	d := int32(points - w.done)
	if w.shell.GetLayout() == nil {
		return
	}
	switch l := w.shell.GetLayout().Impl().(type) {
	case swt.GridLayoutLike:
		l.AsGridLayout().MarginTop += d
	case *swt.RowLayout:
		l.MarginTop += d
	case *swt.FormLayout:
		l.MarginTop += d
	default:
		return
	}
	w.done = points
	if d != 0 {
		w.shell.Layout()
	}
}

// Window creates a hidden window with the children in opts; call Show once its content is built.
// Children whose callbacks use the window go in a w.Add after this call, when w is set.
// Options: Gap, Padding, children; the placement options do not apply.
func (a *App) Window(title string, opts ...Option) *Window {
	s := swt.NewShellDisplayStyle(a.display, swt.SHELL_TRIM)
	s.SetText(title)
	fluentShell(s)
	a.badgeWindow(s)
	gap, pad := spacingOf(opts, Spacing.PageMargin)
	s.SetLayout(newBox(false, gap, pad))
	w := &Window{panel: newPanel(&s.Composite), shell: s}
	a.windows = append(a.windows, w)
	s.AddShellListener(swt.ShellListenerShellClosedAdapter(func(e *swt.ShellEvent) { a.windowClosing(w, e) }))
	finish(w, 0, opts)
	return w
}

// Show packs the window to its content size (unless SetSize was called), opens it and, the first
// time, gives the keyboard focus to the widget created with Focused.
func (w *Window) Show() {
	if !w.sized {
		w.shell.Pack()
	}
	c := w.initialFocus()
	if c != nil {
		// Shell.Open, and windowDidBecomeKey on macOS, give the focus to the saved control first.
		w.shell.SetSavedFocus(c)
	}
	// Before the window shows: moved while on screen, the window did not come back on the Retina screen.
	shot.PlaceOnRetina(shotHandle(w.shell))
	if qaQuiet() {
		w.shell.SetVisible(true) // orderFront without key on macOS, no activation on Windows
	} else {
		w.shell.Open()
	}
	// Open passes over a push button on Windows and Linux.
	if c != nil && !c.IsDisposed() {
		swt.QAFocus(c) // SetFocus, except in a QA run on macOS
	}
}

// initialFocus returns the control that Focused named or, without one, the default that package look
// set, once: a window shown again keeps the focus SWT saved for it.
func (w *Window) initialFocus() *swt.Control {
	c, _ := w.shell.GetDataKey(focusedKey).(*swt.Control)
	if c == nil {
		if f, ok := w.shell.GetDataKey(focusDefaultKey).(func() *swt.Control); ok {
			// The layout hides the pages of a Pages that are not shown; f skips hidden controls.
			w.shell.Layout()
			c = f()
		}
	}
	w.shell.SetDataKeyValue(focusedKey, nil)
	w.shell.SetDataKeyValue(focusDefaultKey, nil)
	if c == nil || c.IsDisposed() {
		return nil
	}
	return c
}

// focusedKey holds the control given Focused in a window not shown yet. focusDefaultKey holds a
// func() *swt.Control that package look sets for a sidebar window; look has the same key.
const (
	focusedKey      = "gowt.focused"
	focusDefaultKey = "gowt.look.focus"
)

// SetTitle changes the window title.
func (w *Window) SetTitle(s string) {
	w.shell.SetText(s)
	if f, ok := w.shell.GetDataKey(titleHookKey).(func(string)); ok {
		f(s)
	}
}

// titleHookKey holds a func(string) that SetTitle calls with the new title: the toolbar bar of package
// look shows it, and SWT reports no title change. look has the same key.
const titleHookKey = "gowt.look.title"

// SetSize sets the window size in points.
func (w *Window) SetSize(width, height int) {
	w.sized = true
	w.shell.SetSize(int32(width), int32(height))
}

// SetMinSize keeps the user from resizing the window below width x height points. A window laid out
// by look.SetSidebar or look.SetInspector takes the larger of this and the room its panes and content
// need.
func (w *Window) SetMinSize(width, height int) {
	if f, ok := w.shell.GetDataKey(minSizeHookKey).(func(int, int)); ok {
		f(width, height)
		return
	}
	w.shell.SetMinimumSize(int32(width), int32(height))
}

// minSizeHookKey holds a func(width, height int) that SetMinSize calls instead of setting the shell
// minimum: look.SetSidebar keeps it in step with the sidebar. look has the same key.
const minSizeHookKey = "gowt.look.minSize"

// Close closes the window as the user would, running OnClose.
func (w *Window) Close() { w.shell.Close() }

// OnClose registers f; returning false keeps the window open. Several handlers are asked in order
// and the first false stops the rest. Closing the last window then asks App.OnQuit.
func (w *Window) OnClose(f func() bool) { w.onClose = append(w.onClose, f) }

// windowClosing runs the OnClose handlers, then, for the last window, the OnQuit ones. Window listeners
// of swt added through Unwrap run after this one; during a quit question the window is not the last
// one asked, so OnQuit is left to askQuit.
func (a *App) windowClosing(w *Window, e *swt.ShellEvent) {
	if a.approved {
		return
	}
	for _, f := range w.onClose {
		if !f() {
			e.Doit = false
			return
		}
	}
	if a.asking || len(a.display.GetShells()) != 1 {
		return
	}
	if !a.askQuit(w) {
		e.Doit = false
	}
}

// Unwrap returns the underlying shell.
func (w *Window) Unwrap() *swt.Shell { return w.shell }

// LabelView is a static text.
type LabelView struct {
	base
	l *swt.Label
}

// SetText replaces the label text. Lays the parents out when the size changes.
func (l *LabelView) SetText(s string) { sizing(&l.l.Control, func() { l.l.SetText(s) }) }

// Unwrap returns the underlying swt.Label for the full API.
func (l *LabelView) Unwrap() *swt.Label { return l.l }

// ButtonView is a push, check, radio or toggle button.
type ButtonView struct {
	base
	b *swt.Button
}

// SetText replaces the button text. Lays the parents out when the size changes.
func (b *ButtonView) SetText(s string) { sizing(&b.b.Control, func() { b.b.SetText(s) }) }

// Checked reports the state of a Check, Radio or Toggle button.
func (b *ButtonView) Checked() bool { return b.b.GetSelection() }

// SetChecked sets the state of a Check, Radio or Toggle button.
func (b *ButtonView) SetChecked(v bool) {
	b.b.SetSelection(v)
	syncToggle(b.b)
}

// Unwrap returns the underlying swt.Button for the full API.
func (b *ButtonView) Unwrap() *swt.Button { return b.b }

// OnClick runs f when the button is pressed (or toggled).
func (b *ButtonView) OnClick(f func()) {
	b.b.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) { f() }))
}

// SwitchView is an on/off toggle, see panel.SwitchView.
type SwitchView struct {
	base
	b *swt.Button
}

// Checked reports whether the switch is on.
func (s *SwitchView) Checked() bool { return s.b.GetSelection() }

// SetChecked sets the state without calling OnChange.
func (s *SwitchView) SetChecked(v bool) { s.b.SetSelection(v) }

// SetText replaces the label. Lays the parents out when the size changes.
func (s *SwitchView) SetText(t string) { sizing(&s.b.Control, func() { s.b.SetText(t) }) }

// Unwrap returns the underlying swt.Button for the full API.
func (s *SwitchView) Unwrap() *swt.Button { return s.b }

// OnChange runs f with the new state when the user toggles the switch.
func (s *SwitchView) OnChange(f func(on bool)) {
	onSelect(s.b, func(*swt.SelectionEvent) { f(s.b.GetSelection()) })
}

// TextView is an edit field.
type TextView struct {
	base
	t      *swt.Text
	inset  [2]int
	grow   *autoGrow
	aff    *affix // Prefix, Suffix, Small; nil without them
	fill   Paint  // FieldFill; nil for the native background
	stroke Paint  // FieldStroke; nil for no border
	radius int    // FieldFill corner radius, points
	padX   int    // Inset of a single-line field, points
}

// Text returns the current content; line breaks are \n on every system (Windows stores \r\n).
func (t *TextView) Text() string { return lf(t.t.GetText()) }

// lf turns the \r\n of Windows into \n.
func lf(s string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(s, "\r\n", "\n")
	}
	return s
}

// SetText replaces the content.
func (t *TextView) SetText(s string) { t.t.SetText(s) }

// SetHint sets the placeholder shown while the field is empty.
func (t *TextView) SetHint(s string) {
	if t.aff != nil && t.aff.nativeHint {
		t.aff.hint = s
		setFieldAffix(t.t, t.aff)
		return
	}
	t.t.SetMessage(s)
	if t.t.GetStyle()&swt.MULTI != 0 {
		multiHint(t.t)
	}
}

// Unwrap returns the underlying swt.Text for the full API.
func (t *TextView) Unwrap() *swt.Text { return t.t }

// OnChange runs f with the new content after every edit.
func (t *TextView) OnChange(f func(text string)) {
	t.t.AddModifyListener(&modifier{func() { f(lf(t.t.GetText())) }})
}

// OnActivate runs f when the user presses Enter in a single-line field.
func (t *TextView) OnActivate(f func()) {
	onActivate(t.t, func(*swt.SelectionEvent) { f() })
}

// Append adds s at the end without rewriting the content.
func (t *TextView) Append(s string) { t.t.Append(s) }

// Lines returns the number of lines; a trailing newline starts an empty last line.
func (t *TextView) Lines() int { return int(t.t.GetLineCount()) }

// ScrollToEnd moves the caret to the end and scrolls it into view.
func (t *TextView) ScrollToEnd() {
	t.t.SetSelection(t.t.GetCharCount())
	t.t.ShowSelection()
}

// TrimHead deletes the oldest lines so that at most keep lines remain; TrimHead(0) clears the text.
// It reads the content to find the cut, so call it when Lines exceeds a cap by a margin, not after
// every Append. It works on a read-only Text.
func (t *TextView) TrimHead(keep int) {
	if int(t.t.GetLineCount()) <= keep {
		return
	}
	drop := headUnits(t.t.GetText(), keep)
	// Insert is refused on a read-only field on some systems, so lift the flag for the edit.
	ro := !t.t.GetEditable()
	if ro {
		t.t.SetEditable(true)
	}
	t.t.SetSelectionStartEnd(0, int32(drop))
	t.t.Insert("")
	if ro {
		t.t.SetEditable(false)
	}
}

// headUnits returns the UTF-16 length of the part of s before its last keep lines.
func headUnits(s string, keep int) int {
	lines := strings.Count(s, "\n") + 1
	n := 0
	for _, c := range s {
		if lines <= keep {
			break
		}
		n++
		if c >= 0x10000 {
			n++
		}
		if c == '\n' {
			lines--
		}
	}
	return n
}

type modifier struct{ f func() }

func (m *modifier) ModifyText(*swt.ModifyEvent) { m.f() }
