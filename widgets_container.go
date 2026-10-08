package gowt

import "github.com/haiodo/gowt/swt"

// Group is a container with a titled frame.
type Group struct {
	panel
	g *swt.Group
}

// Group adds a titled container.
func (p *Panel) Group(title string, opts ...Option) *Group {
	g := swt.NewGroup(p.c, resolve(swt.NONE, opts))
	g.SetText(title)
	applyOpts(&g.Control, opts)
	return &Group{Panel{c: &g.Composite}, g}
}

// SetTitle replaces the frame title.
func (g *Group) SetTitle(s string) { g.g.SetText(s) }

// Unwrap returns the underlying swt.Group for the full API.
func (g *Group) Unwrap() *swt.Group { return g.g }

// Tabs is a native tab folder; each Tab returns the page container.
type Tabs struct{ f *swt.TabFolder }

// Tabs adds a tab folder; Bottom() moves the tabs below the pages.
func (p *Panel) Tabs(opts ...Option) *Tabs {
	f := swt.NewTabFolder(p.c, resolve(swt.NONE, opts))
	applyOpts(&f.Control, opts)
	return &Tabs{f}
}

// Tab appends a tab and returns its page.
func (t *Tabs) Tab(title string, opts ...Option) *Panel {
	it := swt.NewTabItem(t.f, resolve(swt.NONE, opts))
	it.SetText(title)
	page := swt.NewCompositeParentStyle(t.f, swt.NONE)
	it.SetControl(page)
	return &Panel{c: page}
}

func (t *Tabs) control() *swt.Control { return &t.f.Control }

// Index is the selected tab, -1 if none.
func (t *Tabs) Index() int { return int(t.f.GetSelectionIndex()) }

// Select selects tab i.
func (t *Tabs) Select(i int) { t.f.SetSelectionIndex(int32(i)) }

// Unwrap returns the underlying swt.TabFolder for the full API.
func (t *Tabs) Unwrap() *swt.TabFolder { return t.f }

// OnSelect runs f with the selected tab index.
func (t *Tabs) OnSelect(f func(index int)) {
	onSelect(t.f, func(*swt.SelectionEvent) { f(t.Index()) })
}

// CTabs is the custom-drawn tab folder: closable tabs (Closable on the folder's Tab) and a flatter look.
type CTabs struct{ f *swt.CTabFolder }

// CTabs adds a custom tab folder; Border() draws a frame.
func (p *Panel) CTabs(opts ...Option) *CTabs {
	f := swt.NewCTabFolder(p.c, resolve(swt.NONE, opts))
	applyOpts(&f.Control, opts)
	return &CTabs{f}
}

// Tab appends a tab and returns its page; Closable() adds a close button to the tab.
func (t *CTabs) Tab(title string, opts ...Option) *Panel {
	it := swt.NewCTabItem(t.f, resolve(swt.NONE, opts))
	it.SetText(title)
	page := swt.NewCompositeParentStyle(t.f, swt.NONE)
	it.SetControl(page)
	return &Panel{c: page}
}

func (t *CTabs) control() *swt.Control { return &t.f.Control }

// Index is the selected tab, -1 if none.
func (t *CTabs) Index() int { return int(t.f.GetSelectionIndex()) }

// Select selects tab i.
func (t *CTabs) Select(i int) { t.f.SetSelectionIndex(int32(i)) }

// Unwrap returns the underlying swt.CTabFolder for the full API.
func (t *CTabs) Unwrap() *swt.CTabFolder { return t.f }

// OnSelect runs f with the selected tab index.
func (t *CTabs) OnSelect(f func(index int)) {
	onSelect(t.f, func(*swt.SelectionEvent) { f(t.Index()) })
}

// OnClose runs f with the index of the tab whose close button was pressed; false keeps the tab.
func (t *CTabs) OnClose(f func(index int) bool) {
	t.f.AddCTabFolder2Listener(swt.CTabFolder2ListenerCloseAdapter(func(e *swt.CTabFolderEvent) {
		for i, it := range t.f.GetItems() {
			if &it.Widget == e.Item && !f(i) {
				e.Doit = false
			}
		}
	}))
}

// Split divides its children with draggable sashes; the children are ordinary Panel children.
type Split struct {
	panel
	s *swt.SashForm
}

// Split adds a sash form: children side by side, or stacked with Vertical().
func (p *Panel) Split(opts ...Option) *Split {
	s := swt.NewSashForm(p.c, resolve(swt.NONE, opts))
	applyOpts(&s.Control, opts)
	return &Split{Panel{c: &s.Composite}, s}
}

// SetWeights sets the relative size of each child, one weight per child.
func (s *Split) SetWeights(w ...int) { s.s.SetWeights(toInt32s(w)) }

// Unwrap returns the underlying swt.SashForm for the full API.
func (s *Split) Unwrap() *swt.SashForm { return s.s }

// Scroll shows content larger than itself with scroll bars. Build the content in Content, then call Fit.
type Scroll struct {
	s    *swt.ScrolledComposite
	body *Panel
}

// Scroll adds a scrolled area; Border() draws a frame.
func (p *Panel) Scroll(opts ...Option) *Scroll {
	s := swt.NewScrolledComposite(p.c, resolve(swt.V_SCROLL|swt.H_SCROLL, opts))
	applyOpts(&s.Control, opts)
	body := swt.NewCompositeParentStyle(s, swt.NONE)
	s.SetContent(body)
	s.SetExpandHorizontal(true)
	s.SetExpandVertical(true)
	return &Scroll{s, &Panel{c: body}}
}

// Content is the container that scrolls.
func (s *Scroll) Content() *Panel { return s.body }

// Fit sets the scrollable size to the content's natural size; call it after the content is built.
func (s *Scroll) Fit() {
	size := s.body.c.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	s.s.SetMinSizeWidthHeight(size.X, size.Y)
}

func (s *Scroll) control() *swt.Control { return &s.s.Control }

// Unwrap returns the underlying swt.ScrolledComposite for the full API.
func (s *Scroll) Unwrap() *swt.ScrolledComposite { return s.s }

// Sash is a bare draggable divider; most code wants Split instead. The sash does not move itself:
// OnMove must lay it out (Unwrap gives the layout data).
type Sash struct{ s *swt.Sash }

// Sash adds a sash; Vertical() makes a vertical bar that moves left and right.
func (p *Panel) Sash(opts ...Option) *Sash {
	s := swt.NewSash(p.c, resolve(swt.NONE, opts))
	applyOpts(&s.Control, opts)
	return &Sash{s}
}

// OnMove runs f with the position the user dragged the sash to, in parent coordinates.
func (s *Sash) OnMove(f func(x, y int)) {
	onSelect(s.s, func(e *swt.SelectionEvent) { f(int(e.X), int(e.Y)) })
}

func (s *Sash) control() *swt.Control { return &s.s.Control }

// Unwrap returns the underlying swt.Sash for the full API.
func (s *Sash) Unwrap() *swt.Sash { return s.s }

// ToolBar is a row of tool buttons.
type ToolBar struct{ t *swt.ToolBar }

// ToolBar adds a tool bar; Flat() and Wrap() change its look.
func (p *Panel) ToolBar(opts ...Option) *ToolBar {
	t := swt.NewToolBar(p.c, resolve(swt.NONE, opts))
	applyOpts(&t.Control, opts)
	return &ToolBar{t}
}

// Item appends a push button (Check, Radio or DropDown for other kinds). onClick may be nil.
func (t *ToolBar) Item(text string, onClick func(), opts ...Option) *ToolItem {
	it := swt.NewToolItem(t.t, resolve(swt.NONE, opts))
	it.SetText(text)
	w := &ToolItem{it}
	if onClick != nil {
		w.OnClick(onClick)
	}
	return w
}

// Separator appends a divider.
func (t *ToolBar) Separator() { swt.NewToolItem(t.t, swt.SEPARATOR) }

func (t *ToolBar) control() *swt.Control { return &t.t.Control }

// Unwrap returns the underlying swt.ToolBar for the full API.
func (t *ToolBar) Unwrap() *swt.ToolBar { return t.t }

// ToolItem is a tool bar button.
type ToolItem struct{ i *swt.ToolItem }

// SetText replaces the item text.
func (i *ToolItem) SetText(s string) { i.i.SetText(s) }

// SetImage sets the icon; the item does not own img.
func (i *ToolItem) SetImage(img *Image) { i.i.SetImage(img.i) }

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

// CoolBar is a tool bar area whose bands the user can rearrange.
type CoolBar struct {
	panel
	b *swt.CoolBar
}

// CoolBar adds a cool bar; create the band contents as its children, then Add them.
func (p *Panel) CoolBar(opts ...Option) *CoolBar {
	b := swt.NewCoolBar(p.c, resolve(swt.NONE, opts))
	applyOpts(&b.Control, opts)
	return &CoolBar{Panel{c: &b.Composite}, b}
}

// Add makes w (a child of the cool bar) a draggable band of its natural size.
func (b *CoolBar) Add(w Widget) {
	it := swt.NewCoolItem(b.b, swt.NONE)
	c := w.control()
	it.SetControl(c)
	size := c.ComputeSize(swt.DEFAULT, swt.DEFAULT)
	it.SetPreferredSizeSize(it.ComputeSize(size.X, size.Y))
}

// Unwrap returns the underlying swt.CoolBar for the full API.
func (b *CoolBar) Unwrap() *swt.CoolBar { return b.b }
