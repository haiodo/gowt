// Package gowt is an idiomatic Go facade over the generated swt package: no Java-shaped
// overloads, listeners are plain funcs, errors come back from Run. Unwrap on any widget
// returns the underlying swt object for the full API.
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
	"github.com/haiodo/gowt/rx"
	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread; pinning in init keeps the main goroutine there.
func init() { runtime.LockOSThread() }

// App is the handle to the UI thread, valid inside Run.
type App struct {
	display *swt.Display
	quit    atomic.Bool
	images  map[*Image]struct{}
	fonts   map[Font]*sharedFont
	clip    *Clipboard
	onError func(error)
	dark    bool                      // the theme the Adaptive colours were last applied for
	paints  map[*swt.Control]*painted // controls given an Adaptive colour
	// controls whose preferred size changed in this event-loop turn; see sizing
	relayouts []*swt.Control
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
	a := &App{display: swt.NewDisplay(), images: map[*Image]struct{}{}, fonts: map[Font]*sharedFont{}}
	defer a.display.Dispose()
	toOnError := func(err error) { a.fail(err) }
	a.display.SetRuntimeExceptionHandler(toOnError)
	a.display.SetErrorHandler(toOnError)
	defer a.disposeImages()
	defer a.disposeClipboard()
	apps.Store(a.display, a)
	defer apps.Delete(a.display)
	// Deferred last, so rx stops waking the loop before the display is disposed.
	defer rx.Attach(func(tick func()) func() {
		r := jrt.NewRunnable(tick)
		return func() { a.display.AsyncExec(r) }
	})()
	a.display.FollowSystemTheme()
	a.followTheme()
	setup(a)
	snapTick := func() {}
	if p := os.Getenv("GOWT_SNAP"); p != "" {
		snapTick = a.armSnap(p, &err)
	}
	// The display can dispose itself inside ReadAndDispatch: macOS Quit, Windows WM_ENDSESSION.
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
	a.display.AsyncExec(jrt.NewRunnable(f))
}

// Sync runs f on the UI thread and waits; safe from any goroutine except while the UI thread is blocked on you.
// Once Run has returned, Sync returns at once without running f.
func (a *App) Sync(f func()) {
	defer ignoreDisposed()
	a.display.SyncExec(jrt.NewRunnable(f))
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

// After runs f on the UI thread once, after d.
func (a *App) After(d time.Duration, f func()) {
	a.display.TimerExec(int32(d.Milliseconds()), jrt.NewRunnable(f))
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
		f()
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

// Multiline makes a Text multi-line, wrapped, with a vertical scroll bar.
func Multiline() Option { return style{id: sMultiline, set: swt.MULTI | swt.WRAP | swt.V_SCROLL} }

// Password masks the characters typed into a Text.
func Password() Option { return style{id: sPassword, set: swt.PASSWORD} }

// ReadOnly makes a Text or Combo non-editable.
func ReadOnly() Option { return style{id: sReadOnly, set: swt.READ_ONLY} }

// Border draws a frame; Table and Tree have one by default.
func Border() Option { return style{id: sBorder, set: swt.BORDER} }

// CellSelect makes a Tree select only the text of a node, as v0.1.0 did on Windows and Linux; then
// CellAt and OnCellClick find a node only where its text or icon is.
func CellSelect() Option { return style{id: sCellSelect, clear: swt.FULL_SELECTION} }

// NoBorder removes the frame that List, Table and Tree have by default.
func NoBorder() Option { return style{id: sNoBorder, clear: swt.BORDER} }

// Search makes a Text a search field with a magnifier icon and a clear button.
func Search() Option {
	return style{id: sSearch, set: swt.SEARCH | swt.ICON_SEARCH | swt.ICON_CANCEL}
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

// Add creates children at the end of p, in order, and lays p out again when it is on screen.
// It is the way to fill a container after its creation, for content that changes; see Clear.
func (p *panel) Add(children ...Option) {
	add(p, children)
	if !p.c.IsDisposed() && p.c.IsVisible() {
		p.c.LayoutOverload2(true, true)
	}
}

func (p *panel) into() *swt.Composite { return p.c }
func (p *panel) added(Widget)         {}

// Relayout lays the children out again and recomputes their sizes, as needed after a change made
// through Unwrap, such as swt.Label.SetText. The facade setters that change a size (SetText, SetImage,
// SetFont, SetTitle of a Group, SetItems of a Combo) lay out by themselves. p keeps its own size: to
// let it grow too, Relayout the Window.
func (p *panel) Relayout() { p.c.LayoutOverload2(true, true) }

// Clear disposes every child and lays p out again, ready for new content.
func (p *panel) Clear() {
	for _, ch := range p.c.GetChildren() {
		ch.Dispose()
	}
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

// Button creates a push button (Check or Radio for other kinds); OnClick gives it an action.
// Options: Check, Radio, OnClick, OnChange and Checked (Check, Radio), Bind (bool), BindText.
func Button(text string, opts ...Option) Option {
	return leaf("Button", sCheck|sRadio, opts, func(p *swt.Composite) Widget {
		b := swt.NewButton(p, resolve(swt.NONE, opts))
		b.SetText(text)
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
// Options: Multiline, Password, ReadOnly, Search, Border, Hint, Value, OnChange, OnActivate, Bind (string).
func Text(opts ...Option) Option {
	return leaf("Text", sMultiline|sPassword|sReadOnly|sSearch|sBorder, opts, func(p *swt.Composite) Widget {
		t := swt.NewText(p, resolve(swt.NONE, opts))
		return &TextView{base{&t.Control}, t}
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
	gap, pad := spacingOf(opts, Spacing.PageMargin)
	s.SetLayout(newBox(false, gap, pad))
	w := &Window{panel: newPanel(&s.Composite), shell: s}
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
	w.shell.Open()
	// Open passes over a push button on Windows and Linux.
	if c != nil && !c.IsDisposed() {
		c.SetFocus()
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

// LabelView is a static text.
type LabelView struct {
	base
	l *swt.Label
}

// SetText replaces the label text. Lays the parents out when the size changes.
func (l *LabelView) SetText(s string) { sizing(&l.l.Control, func() { l.l.SetText(s) }) }

// Unwrap returns the underlying swt.Label for the full API.
func (l *LabelView) Unwrap() *swt.Label { return l.l }

// ButtonView is a push, check or radio button.
type ButtonView struct {
	base
	b *swt.Button
}

// SetText replaces the button text. Lays the parents out when the size changes.
func (b *ButtonView) SetText(s string) { sizing(&b.b.Control, func() { b.b.SetText(s) }) }

// Checked reports the state of a Check or Radio button.
func (b *ButtonView) Checked() bool { return b.b.GetSelection() }

// SetChecked sets the state of a Check or Radio button.
func (b *ButtonView) SetChecked(v bool) { b.b.SetSelection(v) }

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
	t *swt.Text
}

// Text returns the current content.
func (t *TextView) Text() string { return t.t.GetText() }

// SetText replaces the content.
func (t *TextView) SetText(s string) { t.t.SetText(s) }

// SetHint sets the placeholder shown while the field is empty.
func (t *TextView) SetHint(s string) { t.t.SetMessage(s) }

// Unwrap returns the underlying swt.Text for the full API.
func (t *TextView) Unwrap() *swt.Text { return t.t }

// OnChange runs f with the new content after every edit.
func (t *TextView) OnChange(f func(text string)) {
	t.t.AddModifyListener(&modifier{func() { f(t.t.GetText()) }})
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
