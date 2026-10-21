package gowt

import (
	"fmt"
	"time"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

// GroupView is a container with a titled frame.
type GroupView struct {
	*panel
	g *swt.Group
}

// Group creates a titled frame whose children stack as in a VBox, Padding(Spacing.SM) from the frame.
// Options: Gap, Padding, children.
func Group(title string, opts ...Option) Option {
	return &node{"Group", func(h host) Widget {
		g := swt.NewGroup(h.into(), swt.NONE)
		g.SetText(title)
		gap, pad := spacingOf(opts, Spacing.SM)
		g.SetLayout(newBox(false, gap, pad))
		w := &GroupView{newPanel(&g.Composite), g}
		finish(w, 0, opts)
		return w
	}}
}

// SetTitle replaces the frame title. Lays the parents out when the size changes.
func (g *GroupView) SetTitle(s string) { sizing(&g.g.Control, func() { g.g.SetText(s) }) }

// Unwrap returns the underlying swt.Group for the full API.
func (g *GroupView) Unwrap() *swt.Group { return g.g }

// TabsView is a native tab folder; each Tab returns the page container.
type TabsView struct {
	base
	f *swt.TabFolder
}

// Tabs creates a tab folder whose children are Tab pages; Bottom() moves the tabs below the pages.
// Options: Bottom, Selected, OnSelect, children (Tab only).
func Tabs(opts ...Option) Option {
	return leaf("Tabs", sBottom, opts, func(p *swt.Composite) Widget {
		f := swt.NewTabFolder(p, resolve(swt.NONE, opts))
		return &TabsView{base{&f.Control}, f}
	})
}

// Add appends Tab pages.
func (t *TabsView) Add(tabs ...Option) { add(t, tabs) }

func (t *TabsView) into() *swt.Composite { return &t.f.Composite }
func (t *TabsView) added(Widget)         {}
func (t *TabsView) only() string         { return "Tab" }

// Tab creates a tab page of a Tabs or CTabs; its children stack as in a VBox,
// Padding(Spacing.PageMargin) from the edge. Ref gives the page as a *Panel; its Dispose removes the
// tab too, and a CTabs tab closed by its button disposes the page.
// Options: Closable (CTabs), Gap, Padding, children.
func Tab(title string, opts ...Option) Option {
	return &node{"Tab", func(h host) Widget {
		var page *swt.Composite
		switch t := h.(type) {
		case *TabsView:
			it := swt.NewTabItem(t.f, resolve(swt.NONE, opts))
			it.SetText(title)
			page = swt.NewCompositeParentStyle(t.f, swt.NONE)
			it.SetControl(page)
			// A disposed page takes its tab along; when the folder goes, its items are released first.
			page.AddDisposeListener(disposeFunc(func() {
				if !it.IsDisposed() {
					it.Dispose()
				}
			}))
		case *CTabsView:
			it := swt.NewCTabItem(t.f, resolve(swt.NONE, opts))
			it.SetText(title)
			page = swt.NewCompositeParentStyle(t.f, swt.NONE)
			it.SetControl(page)
			// SWT disposes only the item of a closed tab; whichever of item and page goes first takes the other.
			gone := false
			page.AddDisposeListener(disposeFunc(func() {
				if !gone {
					gone = true
					it.Dispose()
				}
			}))
			it.AddDisposeListener(disposeFunc(func() {
				if !gone {
					gone = true
					page.Dispose()
				}
			}))
		default:
			reportOn(h.control(), fmt.Errorf("gowt: Tab needs a Tabs or CTabs parent, not %s", nameOf(h.control())))
			return nil
		}
		gap, pad := spacingOf(opts, Spacing.PageMargin)
		page.SetLayout(newBox(false, gap, pad))
		p := &Panel{newPanel(page)}
		finish(p, sClosable, opts)
		return p
	}}
}

// Index is the selected tab, -1 if none.
func (t *TabsView) Index() int { return int(t.f.GetSelectionIndex()) }

// Select selects tab i.
func (t *TabsView) Select(i int) { t.f.SetSelectionIndex(int32(i)) }

// Unwrap returns the underlying swt.TabFolder for the full API.
func (t *TabsView) Unwrap() *swt.TabFolder { return t.f }

// OnSelect runs f with the selected tab index.
func (t *TabsView) OnSelect(f func(index int)) {
	onSelect(t.f, func(*swt.SelectionEvent) { f(t.Index()) })
}

// CTabsView is the custom-drawn tab folder: closable tabs (Closable on the folder's Tab) and a flatter look.
type CTabsView struct {
	base
	f *swt.CTabFolder
}

// CTabs creates a custom tab folder whose children are Tab pages; Border() draws a frame and
// Closable() on a Tab adds a close button to it.
// Options: Bottom, Border, Selected, OnSelect, children (Tab only).
func CTabs(opts ...Option) Option {
	return leaf("CTabs", sBottom|sBorder, opts, func(p *swt.Composite) Widget {
		f := swt.NewCTabFolder(p, resolve(swt.NONE, opts))
		return &CTabsView{base{&f.Control}, f}
	})
}

// Add appends Tab pages.
func (t *CTabsView) Add(tabs ...Option) { add(t, tabs) }

func (t *CTabsView) into() *swt.Composite { return &t.f.Composite }
func (t *CTabsView) added(Widget)         {}
func (t *CTabsView) only() string         { return "Tab" }

// Index is the selected tab, -1 if none.
func (t *CTabsView) Index() int { return int(t.f.GetSelectionIndex()) }

// Select selects tab i.
func (t *CTabsView) Select(i int) { t.f.SetSelectionIndex(int32(i)) }

// Unwrap returns the underlying swt.CTabFolder for the full API.
func (t *CTabsView) Unwrap() *swt.CTabFolder { return t.f }

// OnSelect runs f with the selected tab index.
func (t *CTabsView) OnSelect(f func(index int)) {
	onSelect(t.f, func(*swt.SelectionEvent) { f(t.Index()) })
}

// OnClose runs f with the index of the tab whose close button was pressed; false keeps the tab.
func (t *CTabsView) OnClose(f func(index int) bool) {
	t.f.AddCTabFolder2Listener(swt.CTabFolder2ListenerCloseAdapter(func(e *swt.CTabFolderEvent) {
		for i, it := range t.f.GetItems() {
			if &it.Widget == e.Item && !f(i) {
				e.Doit = false
			}
		}
	}))
}

// SplitView divides its children with draggable sashes; the children are ordinary Panel children.
type SplitView struct {
	*panel
	s *swt.SashForm
}

// Split creates a sash form: its children side by side, or stacked with Vertical(), with draggable
// sashes between them. Every pane fills its part, sized by SetWeights; Grow on a pane is accepted
// and changes nothing, Size, Span and the other placement options are reported to App.OnError.
// Options: Vertical, children.
func Split(opts ...Option) Option {
	return &node{"Split", func(h host) Widget {
		s := swt.NewSashForm(h.into(), resolve(swt.NONE, opts))
		w := &SplitView{newPanel(&s.Composite), s}
		finish(w, sVertical, opts)
		return w
	}}
}

// SetWeights sets the relative size of each child, one weight per child.
func (s *SplitView) SetWeights(w ...int) { s.s.SetWeights(toInt32s(w)) }

// Unwrap returns the underlying swt.SashForm for the full API.
func (s *SplitView) Unwrap() *swt.SashForm { return s.s }

// ScrollView shows content larger than itself with scroll bars.
type ScrollView struct {
	base
	s    *swt.ScrolledComposite
	body *Panel
}

// Scroll creates a scrolled area whose children stack as in a VBox; Border() draws a frame.
// Options: Border, Gap, Padding, children.
func Scroll(opts ...Option) Option {
	return &node{"Scroll", func(h host) Widget {
		s := swt.NewScrolledComposite(h.into(), resolve(swt.V_SCROLL|swt.H_SCROLL, opts))
		body := swt.NewCompositeParentStyle(s, swt.NONE)
		gap, pad := spacingOf(opts, 0)
		body.SetLayout(newBox(false, gap, pad))
		s.SetContent(body)
		s.SetExpandHorizontal(true)
		s.SetExpandVertical(true)
		w := &ScrollView{base{&s.Control}, s, &Panel{newPanel(body)}}
		w.watchRows()
		finish(w, sBorder, opts)
		w.Fit()
		return w
	}}
}

// Add creates children at the end of the content, lays them out and fits the scrollable size as Fit
// does; the children already there keep their measured sizes.
func (s *ScrollView) Add(children ...Option) {
	old := s.body.c.GetChildren()
	add(s, children)
	if s.body.c.IsDisposed() {
		return
	}
	layoutAdded(s.body.c, old)
	s.fit(false)
}

func (s *ScrollView) into() *swt.Composite { return s.body.c }
func (s *ScrollView) added(Widget)         {}

// Content is the container that scrolls.
func (s *ScrollView) Content() *Panel { return s.body }

// Fit sets the scrollable size to the content's natural size; Scroll and Add call it, call it again
// after the content changes otherwise.
func (s *ScrollView) Fit() { s.fit(true) }

// fit without flush reuses the sizes the content's layouts cached.
func (s *ScrollView) fit(flush bool) {
	size := s.body.c.ComputeSizeWHintHHintChanged(swt.DEFAULT, swt.DEFAULT, flush)
	s.s.SetMinSizeWidthHeight(size.X, size.Y)
}

// Origin returns the point of the content shown at the top left corner: 0, 0 before any scroll.
func (s *ScrollView) Origin() (x, y int) {
	o := s.s.GetOrigin()
	return int(o.X), int(o.Y)
}

// SetOrigin scrolls so that the content's point x, y is at the top left corner, as far as the
// content reaches.
func (s *ScrollView) SetOrigin(x, y int) { s.s.SetOriginXY(int32(x), int32(y)) }

// scrollFrame caps OnScroll at 60 calls a second; a 120 Hz display gets one every second frame.
const scrollFrame = time.Second / 60

// OnScroll runs f with the new Origin after any scroll: the mouse wheel, a trackpad gesture and its
// momentum, the scroll bars, SetOrigin, a resize that moves the content. Scrolls that come faster than
// a frame are coalesced into one call; the last call reports where the scroll stopped.
func (s *ScrollView) OnScroll(f func(x, y int)) {
	lx, ly := s.Origin()
	var last time.Time
	pending := false
	fire := jrt.NewRunnable(func() {
		pending = false
		if s.s.IsDisposed() {
			return
		}
		x, y := s.Origin()
		if x == lx && y == ly {
			return
		}
		lx, ly, last = x, y, time.Now()
		f(x, y)
	})
	// Every scroll of a ScrolledComposite ends in moving its content; the scroll bars' Selection
	// misses SetOrigin and the clamp after a resize.
	s.body.c.AddControlListener(swt.ControlListenerControlMovedAdapter(func(*swt.ControlEvent) {
		if pending {
			return
		}
		pending = true
		d := s.s.GetDisplay()
		if wait := scrollFrame - time.Since(last); wait > 0 {
			d.TimerExec(int32((wait+time.Millisecond-1)/time.Millisecond), fire)
		} else {
			d.AsyncExec(fire)
		}
	}))
}

// Unwrap returns the underlying swt.ScrolledComposite for the full API.
func (s *ScrollView) Unwrap() *swt.ScrolledComposite { return s.s }

// SashView is a bare draggable divider; most code wants Split instead. The sash does not move itself:
// OnMove must lay it out (Unwrap gives the layout data).
type SashView struct {
	base
	s *swt.Sash
}

// Sash creates a sash; Vertical() makes a vertical bar that moves left and right.
// Options: Vertical.
func Sash(opts ...Option) Option {
	return leaf("Sash", sVertical, opts, func(p *swt.Composite) Widget {
		s := swt.NewSash(p, resolve(swt.NONE, opts))
		return &SashView{base{&s.Control}, s}
	})
}

// OnMove runs f with the position the user dragged the sash to, in parent coordinates.
func (s *SashView) OnMove(f func(x, y int)) {
	onSelect(s.s, func(e *swt.SelectionEvent) { f(int(e.X), int(e.Y)) })
}

// Unwrap returns the underlying swt.Sash for the full API.
func (s *SashView) Unwrap() *swt.Sash { return s.s }

// ToolBarView is a row of tool buttons.
type ToolBarView struct {
	base
	t *swt.ToolBar
}

// ToolBar creates a tool bar; Flat() and Wrap() change its look. Its items come from Item.
// Options: Flat, Wrap.
func ToolBar(opts ...Option) Option {
	return leaf("ToolBar", sFlat|sWrap, opts, func(p *swt.Composite) Widget {
		t := swt.NewToolBar(p, resolve(swt.NONE, opts))
		return &ToolBarView{base{&t.Control}, t}
	})
}

// Item appends a push button (Check, Radio or DropDown for other kinds). onClick may be nil.
// Options: Check, Radio, DropDown; for a tooltip or a disabled item use SetTooltip and SetEnabled.
func (t *ToolBarView) Item(text string, onClick func(), opts ...Option) *ToolItem {
	itemOpts(t.t, "ToolBar.Item", sCheck|sRadio|sDropDown, false, opts)
	it := swt.NewToolItem(t.t, resolve(swt.NONE, opts))
	it.SetText(text)
	w := &ToolItem{it}
	if onClick != nil {
		w.OnClick(onClick)
	}
	return w
}

// Separator appends a divider.
func (t *ToolBarView) Separator() { swt.NewToolItem(t.t, swt.SEPARATOR) }

// Unwrap returns the underlying swt.ToolBar for the full API.
func (t *ToolBarView) Unwrap() *swt.ToolBar { return t.t }

// ToolItem is a tool bar button.
type ToolItem struct{ i *swt.ToolItem }

// SetText replaces the item text. Lays the tool bar's parents out when its size changes.
func (i *ToolItem) SetText(s string) { sizing(&i.i.GetParent().Control, func() { i.i.SetText(s) }) }

// SetImage sets the icon; the item does not own img. Lays the tool bar's parents out when its size
// changes.
func (i *ToolItem) SetImage(img *Image) {
	sizing(&i.i.GetParent().Control, func() { i.i.SetImage(img.i) })
}

// SetTooltip sets the text shown on hover.
func (i *ToolItem) SetTooltip(s string) { i.i.SetToolTipText(s) }

// SetEnabled enables or greys out the item.
func (i *ToolItem) SetEnabled(v bool) { i.i.SetEnabled(v) }

// Checked reports the state of a Check or Radio item.
func (i *ToolItem) Checked() bool { return i.i.GetSelection() }

// SetChecked sets the state of a Check or Radio item.
func (i *ToolItem) SetChecked(v bool) { i.i.SetSelection(v) }

// Unwrap returns the underlying swt.ToolItem for the full API.
func (i *ToolItem) Unwrap() *swt.ToolItem { return i.i }

// OnClick runs f when the item is pressed (or toggled).
func (i *ToolItem) OnClick(f func()) {
	onSelect(i.i, func(*swt.SelectionEvent) { f() })
}

// CoolBarView is a tool bar area whose bands the user can rearrange.
type CoolBarView struct {
	*panel
	b *swt.CoolBar
}

// CoolBar creates a cool bar: each child, usually a ToolBar, becomes a band of its natural size.
// Options: common only, children.
func CoolBar(opts ...Option) Option {
	return &node{"CoolBar", func(h host) Widget {
		b := swt.NewCoolBar(h.into(), resolve(swt.NONE, opts))
		w := &CoolBarView{newPanel(&b.Composite), b}
		finish(w, 0, opts)
		return w
	}}
}

// Add appends bands, one per child.
func (b *CoolBarView) Add(bands ...Option) { add(b, bands) }

// added makes w a draggable band of its natural size.
func (b *CoolBarView) added(w Widget) {
	it := swt.NewCoolItem(b.b, swt.NONE)
	c := w.control()
	it.SetControl(c)
	size := c.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	it.SetPreferredSizeSize(it.ComputeSize(size.X, size.Y))
}

// Unwrap returns the underlying swt.CoolBar for the full API.
func (b *CoolBarView) Unwrap() *swt.CoolBar { return b.b }
