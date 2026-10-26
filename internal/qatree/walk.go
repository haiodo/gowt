package qatree

import (
	"slices"
	"strconv"
	"strings"

	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/swt"
)

// IDKey is the data key gowt.ID stores the id under.
const IDKey = "gowt.id"

// idOf reads the id of a control or an item; empty when gowt.ID was not given.
func idOf(w *swt.Widget) string {
	id, _ := w.GetDataKey(IDKey).(string)
	return id
}

// entry is a Node with the swt object behind it, so Find and Act work on the same walk as Dump.
type entry struct {
	n      Node
	kids   []*entry
	parent *entry
	obj    any // the concrete swt control or item; listItem and comboItem for the rows of a List and a Combo
	shell  *swt.Shell
}

// listItem and comboItem are the rows of a List and of a Combo: swt has no item objects for them.
type listItem struct {
	l *swt.List
	i int
}

type comboItem struct {
	c *swt.Combo
	i int
}

type walker struct {
	focus  *swt.Control
	shell  *swt.Shell
	pop    bool // shell is a popover on screen; it counts as visible even where its own window is not
	ox, oy int  // window origin on the screen
}

// walk lists the open shells with everything in them: the windows and the popovers on screen.
func walk(d *swt.Display) []*entry {
	w := &walker{focus: d.GetFocusControl()}
	var out []*entry
	for _, s := range d.GetShells() {
		w.pop = qaview.OnScreen(s)
		if s.IsDisposed() || !s.GetVisible() && !w.pop {
			continue
		}
		w.shell = s
		o := s.ToDisplay(0, 0)
		w.ox, w.oy = int(o.X), int(o.Y)
		e := w.control(&s.Control, true, true)
		if w.pop {
			e.n.Class = "Popover"
		}
		out = append(out, e)
	}
	for i, e := range out {
		setPaths(e, strconv.Itoa(i))
	}
	return out
}

func setPaths(e *entry, path string) {
	e.n.Path = path
	for i, k := range e.kids {
		k.parent = e
		setPaths(k, path+"/"+k.n.Class+"."+strconv.Itoa(i))
	}
}

// item is a row or an entry of a control; wd is nil where swt has no widget for it.
func (w *walker) item(wd *swt.Widget, class, text string, vis, en bool, obj any) *entry {
	e := &entry{
		n:     Node{Class: class, Text: text, Visible: vis, Enabled: en},
		obj:   obj,
		shell: w.shell,
	}
	if wd != nil {
		e.n.ID = idOf(wd)
	}
	return e
}

func (w *walker) bounds(e *entry, r *swt.Rectangle, parent *entry) {
	e.n.Bounds = [4]int{parent.n.Bounds[0] + int(r.X), parent.n.Bounds[1] + int(r.Y), int(r.Width), int(r.Height)}
}

// control is vis and en of the parents, so a node says whether the user can reach it.
func (w *walker) control(c *swt.Control, vis, en bool) *entry {
	impl := c.Impl()
	e := &entry{obj: impl, shell: w.shell}
	shown := c.GetVisible() || w.pop && c == &w.shell.Control
	e.n = Node{ID: idOf(&c.Widget), Class: c.GetName(), Visible: vis && shown, Enabled: en && c.GetEnabled(), Focused: c == w.focus}
	p, sz := c.ToDisplay(0, 0), c.GetSize()
	e.n.Bounds = [4]int{int(p.X) - w.ox, int(p.Y) - w.oy, int(sz.X), int(sz.Y)}
	var pages map[*swt.Control]bool // a page of a TabFolder that is not selected
	switch v := impl.(type) {
	case *swt.Shell:
		e.n.Text = v.GetText()
		if w.pop {
			// On macOS the content view lives in the NSPopover and the shell's own client area is gone.
			if kids := v.GetChildren(); len(kids) > 0 {
				sz := kids[0].GetSize()
				e.n.Bounds = [4]int{0, 0, int(sz.X), int(sz.Y)}
			}
			break
		}
		ca := v.GetClientArea()
		e.n.Bounds = [4]int{0, 0, int(ca.Width), int(ca.Height)}
		if bar := v.GetMenuBar(); bar != nil {
			e.kids = w.menu(bar, e.n.Visible, e.n.Enabled)
		}
	case *swt.Button:
		st := v.GetStyle()
		if st&swt.ARROW == 0 {
			e.n.Text = v.GetText()
		}
		if st&(swt.CHECK|swt.RADIO|swt.TOGGLE) != 0 {
			e.n.Checked = ptr(v.GetSelection())
		}
	case *swt.Label:
		e.n.Text = v.GetText()
	case *swt.CLabel:
		e.n.Text = v.GetText()
	case *swt.Link:
		e.n.Text = v.GetText()
	case *swt.Group:
		e.n.Text = v.GetText()
	case *swt.Text:
		e.n.Value, e.n.Text = v.GetText(), v.GetMessage()
	case *swt.StyledText:
		e.n.Value = v.GetText()
	case *swt.Combo:
		e.n.Value = v.GetText()
		if i := int(v.GetSelectionIndex()); i >= 0 {
			e.n.Selected = []int{i}
		}
		for i, s := range v.GetItems() {
			e.kids = append(e.kids, w.item(nil, "ComboItem", s, e.n.Visible, e.n.Enabled, comboItem{v, i}))
		}
	case *swt.List:
		items := v.GetItems()
		for _, i := range v.GetSelectionIndices() {
			e.n.Selected = append(e.n.Selected, int(i))
		}
		if len(e.n.Selected) > 0 {
			e.n.Value = items[e.n.Selected[0]]
		}
		for i, s := range items {
			e.kids = append(e.kids, w.item(nil, "ListItem", s, e.n.Visible, e.n.Enabled, listItem{v, i}))
		}
	case *swt.Scale:
		e.n.Value = strconv.Itoa(int(v.GetSelection()))
	case *swt.Slider:
		e.n.Value = strconv.Itoa(int(v.GetSelection()))
	case *swt.Spinner:
		e.n.Value = strconv.Itoa(int(v.GetSelection()))
	case *swt.ProgressBar:
		e.n.Value = strconv.Itoa(int(v.GetSelection()))
	case *swt.Table:
		w.table(e, v)
	case *swt.Tree:
		w.tree(e, v)
	case *swt.TabFolder:
		pages = map[*swt.Control]bool{}
		sel := int(v.GetSelectionIndex())
		e.n.Selected = []int{sel}
		for i, it := range v.GetItems() {
			e.kids = append(e.kids, w.item(&it.Widget, "TabItem", it.GetText(), e.n.Visible, e.n.Enabled, it))
			if pg := it.GetControl(); pg != nil {
				pages[pg] = i == sel
			}
		}
	case *swt.CTabFolder:
		pages = map[*swt.Control]bool{}
		sel := int(v.GetSelectionIndex())
		e.n.Selected = []int{sel}
		for i, it := range v.GetItems() {
			e.kids = append(e.kids, w.item(&it.Widget, "CTabItem", it.GetText(), e.n.Visible, e.n.Enabled, it))
			if pg := it.GetControl(); pg != nil {
				pages[pg] = i == sel
			}
		}
	case *swt.ToolBar:
		for _, it := range v.GetItems() {
			te := w.item(&it.Widget, "ToolItem", it.GetText(), e.n.Visible, e.n.Enabled && it.GetEnabled(), it)
			if it.GetStyle()&(swt.CHECK|swt.RADIO) != 0 {
				te.n.Checked = ptr(it.GetSelection())
			}
			w.bounds(te, it.GetBounds(), e)
			e.kids = append(e.kids, te)
		}
	}
	if comp, ok := impl.(interface{ AsComposite() *swt.Composite }); ok {
		for _, k := range comp.AsComposite().GetChildren() {
			kvis := e.n.Visible
			if shown, isPage := pages[k]; isPage && !shown {
				kvis = false
			}
			e.kids = append(e.kids, w.control(k, kvis, e.n.Enabled))
		}
	}
	// The context menu is listed as if open, so its items can be run; it has no place on the screen.
	if m := c.GetMenu(); m != nil && m.GetStyle()&swt.POP_UP != 0 {
		if m.GetItemCount() == 0 {
			showPopup(m) // only an empty menu: an OnShow handler fills it
		}
		me := w.item(&m.Widget, "Menu", "", e.n.Visible, e.n.Enabled, m)
		me.kids = w.menu(m, e.n.Visible, e.n.Enabled)
		e.kids = append(e.kids, me)
	}
	return e
}

// shownKey marks a context menu with the action generation it was shown in.
const shownKey = "gowt.qa.shown"

// actGen counts the actions; the dump and Find run on the same UI thread.
var actGen int

// showPopup sends Show to a context menu once per action, as a right click does before the menu opens,
// so items an OnShow handler adds are in the dump. No native menu opens.
func showPopup(m *swt.Menu) {
	if g, ok := m.GetDataKey(shownKey).(int); ok && g == actGen {
		return
	}
	m.SetDataKeyValue(shownKey, actGen)
	m.NotifyListeners(swt.Show, swt.NewEvent())
}

func (w *walker) menu(m *swt.Menu, vis, en bool) []*entry {
	var out []*entry
	for _, it := range m.GetItems() {
		title, accel, _ := strings.Cut(it.GetText(), "\t")
		e := w.item(&it.Widget, "MenuItem", unmnemonic(title), vis, en && it.GetEnabled(), it)
		e.n.Value = accel
		if it.GetStyle()&(swt.CHECK|swt.RADIO) != 0 {
			e.n.Checked = ptr(it.GetSelection())
		}
		if sub := it.GetMenu(); sub != nil {
			e.kids = w.menu(sub, vis, e.n.Enabled)
		}
		out = append(out, e)
	}
	return out
}

// unmnemonic drops the "&" that marks a mnemonic and keeps "&&" as one "&".
func unmnemonic(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			if i+1 < len(s) && s[i+1] == '&' {
				b.WriteByte('&')
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// table lists the rows; the cells after the first go to Value, tab separated.
func (w *walker) table(e *entry, t *swt.Table) {
	cols, check := int(t.GetColumnCount()), t.GetStyle()&swt.CHECK != 0
	for _, i := range t.GetSelectionIndices() {
		e.n.Selected = append(e.n.Selected, int(i))
	}
	for _, it := range t.GetItems() {
		r := w.item(&it.Widget, "TableItem", it.GetTextIndex(0), e.n.Visible, e.n.Enabled, it)
		var rest []string
		for c := 1; c < cols; c++ {
			rest = append(rest, it.GetTextIndex(int32(c)))
		}
		r.n.Value = strings.Join(rest, "\t")
		if check {
			r.n.Checked = ptr(it.GetChecked())
		}
		w.bounds(r, it.GetBounds(), e)
		e.kids = append(e.kids, r)
	}
}

// tree nests the items. Selected holds positions in the pre-order of all items, collapsed ones
// included.
func (w *walker) tree(e *entry, t *swt.Tree) {
	check := t.GetStyle()&swt.CHECK != 0
	var flat []*swt.TreeItem
	var add func(parent *entry, items []*swt.TreeItem, vis bool)
	add = func(parent *entry, items []*swt.TreeItem, vis bool) {
		for _, it := range items {
			r := w.item(&it.Widget, "TreeItem", it.GetTextIndex(0), vis, e.n.Enabled, it)
			if check {
				r.n.Checked = ptr(it.GetChecked())
			}
			w.bounds(r, it.GetBounds(), e)
			parent.kids = append(parent.kids, r)
			flat = append(flat, it)
			add(r, it.GetItems(), vis && it.GetExpanded())
		}
	}
	add(e, t.GetItems(), e.n.Visible)
	for _, it := range t.GetSelection() {
		if i := slices.Index(flat, it); i >= 0 {
			e.n.Selected = append(e.n.Selected, i)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func toNode(e *entry) Node {
	n := e.n
	for _, k := range e.kids {
		n.Children = append(n.Children, toNode(k))
	}
	return n
}

// Dump lists the open shells with all their controls and items.
func Dump(d *swt.Display) []Node {
	var out []Node
	for _, e := range walk(d) {
		out = append(out, toNode(e))
	}
	return out
}
