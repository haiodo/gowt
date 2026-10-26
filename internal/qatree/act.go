package qatree

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"

	"github.com/haiodo/gowt/internal/qaview"
	"github.com/haiodo/gowt/swt"
)

// Act runs click, type {"text"}, check {"on"}, select {"index"} / {"text"}, activate, expand,
// collapse, focus, close, scroll {"to": selector} / {"dx", "dy"}, scrollevent {"x", "y", "dx", "dy",
// "precise", "phase", "momentum", "mods"}, pinch {"x", "y", "magnification", "phase", "mods"}, pointer
// {"kind", "x", "y", "button", "count", "mods", "tox", "toy", "steps"}, drag {"to": selector, "x", "y"}
// or key {"keys"} on the one control s picks; key with an empty s goes to the focus control. Hidden,
// disabled, in a closed window or under a modal window: refused.
func Act(d *swt.Display, s Selector, action string, args json.RawMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("qatree: %s on %s: %v", action, s, r)
		}
	}()
	if action == "key" && s == (Selector{}) {
		return keyFocused(d, args)
	}
	actGen++ // a context menu shows again, so OnShow refreshes it for this action
	es, err := resolve(walk(d), s)
	if err != nil {
		return err
	}
	if len(es) != 1 {
		return fmt.Errorf("qatree: %s on %s: %d controls match%s", action, s, len(es), paths(es))
	}
	e := es[0]
	if it, ok := e.obj.(*swt.MenuItem); ok && action == "click" {
		if root := popupRoot(it); root != nil {
			// As a right click: the app refreshes its items in OnShow before the user sees the menu.
			showPopup(root)
			defer func() {
				if !root.IsDisposed() {
					root.NotifyListeners(swt.Hide, swt.NewEvent())
				}
			}()
			if es, err = resolve(walk(d), s); err != nil {
				return err
			}
			if len(es) != 1 {
				return fmt.Errorf("qatree: %s on %s: %d controls match after the menu was shown%s", action, s, len(es), paths(es))
			}
			e = es[0]
		}
	}
	if err := reachable(d, e); err != nil {
		return fmt.Errorf("qatree: %s on %s: %w", action, e.n.Path, err)
	}
	switch action {
	case "click":
		return click(e)
	case "type":
		var a struct{ Text string }
		if err := parse(args, &a); err != nil {
			return err
		}
		return typeText(d, e, a.Text)
	case "check":
		a := struct{ On bool }{true}
		if err := parse(args, &a); err != nil {
			return err
		}
		return check(e, a.On)
	case "select":
		var a struct {
			Index *int
			Text  *string
		}
		if err := parse(args, &a); err != nil {
			return err
		}
		return selectItem(e, a.Index, a.Text)
	case "activate":
		return activate(d, e)
	case "expand", "collapse":
		return setExpanded(e, action == "expand")
	case "focus":
		return focus(d, e)
	case "close":
		return closeShell(e)
	case "scroll":
		var a struct {
			To     *Selector
			DX, DY int
		}
		if err := parse(args, &a); err != nil {
			return err
		}
		return scroll(d, e, a.To, a.DX, a.DY)
	case "scrollevent", "pinch":
		return gesture(e, action, args)
	case "pointer":
		return pointer(d, e, args)
	case "drag":
		return drag(d, e, args)
	case "key":
		var a struct{ Keys string }
		if err := parse(args, &a); err != nil {
			return err
		}
		c, err := controlOf(e)
		if err != nil {
			return err
		}
		ensureFocus(d, c)
		return key(d, c, a.Keys)
	}
	return fmt.Errorf("qatree: unknown action %q", action)
}

func parse(args json.RawMessage, v any) error {
	if len(args) == 0 {
		return nil
	}
	return json.Unmarshal(args, v)
}

func reachable(d *swt.Display, e *entry) error {
	if e.shell.IsDisposed() || !e.shell.GetVisible() && !qaview.OnScreen(e.shell) {
		return fmt.Errorf("its window is closed")
	}
	if !e.n.Visible {
		return fmt.Errorf("not visible")
	}
	if !e.n.Enabled {
		return fmt.Errorf("disabled")
	}
	if m := blocker(d, e.shell); m != nil {
		return fmt.Errorf("blocked by the modal window %q", m.GetText())
	}
	return nil
}

// blocker returns a modal shell that stops input to s: an application or system modal one that s is
// not part of, or a primary modal one that s is the parent of.
func blocker(d *swt.Display, s *swt.Shell) *swt.Shell {
	for _, m := range d.GetShells() {
		if m == s || m.IsDisposed() || !m.GetVisible() {
			continue
		}
		st := m.GetStyle()
		if st&(swt.APPLICATION_MODAL|swt.SYSTEM_MODAL) != 0 && !below(s, m) ||
			st&swt.PRIMARY_MODAL != 0 && below(m, s) {
			return m
		}
	}
	return nil
}

// below reports whether s is top or one of its descendant shells.
func below(s, top *swt.Shell) bool {
	for s != nil {
		if s == top {
			return true
		}
		p := s.GetParent()
		if p == nil {
			return false
		}
		s = p.GetShell()
	}
	return false
}

var href = regexp.MustCompile(`href\s*=\s*"([^"]*)"`)

func click(e *entry) error {
	switch o := e.obj.(type) {
	case *swt.Button:
		clickButton(o)
	case *swt.Link:
		ev := swt.NewEvent()
		if m := href.FindStringSubmatch(o.GetText()); m != nil {
			ev.Text = m[1]
		}
		o.NotifyListeners(swt.Selection, ev)
	case *swt.ToolItem:
		if o.GetStyle()&(swt.CHECK|swt.RADIO) != 0 {
			o.SetSelection(o.GetStyle()&swt.RADIO != 0 || !o.GetSelection())
		}
		o.NotifyListeners(swt.Selection, swt.NewEvent())
	case *swt.MenuItem:
		st := o.GetStyle()
		if st&swt.CASCADE != 0 {
			return fmt.Errorf("qatree: click on %s: a submenu has nothing to run", e.n.Path)
		}
		if st&(swt.CHECK|swt.RADIO) != 0 {
			o.SetSelection(st&swt.RADIO != 0 || !o.GetSelection())
		}
		o.NotifyListeners(swt.Selection, swt.NewEvent())
	case *swt.TableItem, *swt.TreeItem, *swt.TabItem, *swt.CTabItem, listItem, comboItem:
		return selectItem(e, nil, nil)
	default:
		return fmt.Errorf("qatree: click does not apply to %s %s", e.n.Class, e.n.Path)
	}
	return nil
}

func check(e *entry, on bool) error {
	if e.n.Checked == nil {
		return fmt.Errorf("qatree: check: %s %s has no checked state", e.n.Class, e.n.Path)
	}
	if *e.n.Checked == on {
		return nil
	}
	ev := swt.NewEvent()
	ev.Detail = swt.CHECK
	switch o := e.obj.(type) {
	case *swt.TableItem:
		o.SetChecked(on)
		ev.Item = &o.Widget
		o.GetParent().NotifyListeners(swt.Selection, ev)
		return nil
	case *swt.TreeItem:
		o.SetChecked(on)
		ev.Item = &o.Widget
		o.GetParent().NotifyListeners(swt.Selection, ev)
		return nil
	case *swt.Button:
		if !on && o.GetStyle()&swt.RADIO != 0 {
			return fmt.Errorf("qatree: check: a radio button %s cannot be cleared, select another", e.n.Path)
		}
	}
	return click(e)
}

// typeText replaces the text one character at a time: KeyDown (a listener may veto the character),
// the text with the character added, which fires Modify, then KeyUp.
func typeText(d *swt.Display, e *entry, text string) error {
	var c *swt.Control
	var get func() string
	var set func(string)
	switch o := e.obj.(type) {
	case *swt.Text:
		if !o.GetEditable() {
			return fmt.Errorf("qatree: type: %s is read-only", e.n.Path)
		}
		c, get = &o.Control, o.GetText
		set = func(s string) { o.SetText(s); o.SetSelection(o.GetCharCount()) }
	case *swt.StyledText:
		if !o.GetEditable() {
			return fmt.Errorf("qatree: type: %s is read-only", e.n.Path)
		}
		c, get, set = &o.Control, o.GetText, o.SetText
	case *swt.Combo:
		if o.GetStyle()&swt.READ_ONLY != 0 {
			return fmt.Errorf("qatree: type: %s is read-only", e.n.Path)
		}
		c, get, set = &o.Control, o.GetText, o.SetText
	default:
		return fmt.Errorf("qatree: type does not apply to %s %s", e.n.Class, e.n.Path)
	}
	ensureFocus(d, c)
	if get() != "" {
		set("")
	}
	cur := ""
	for _, r := range text {
		down := swt.NewEvent()
		down.Character, down.KeyCode = uint16(r), int32(r)
		c.NotifyListeners(swt.KeyDown, down)
		if down.Doit {
			cur += string(r)
			set(cur)
		}
		up := swt.NewEvent()
		up.Character, up.KeyCode = uint16(r), int32(r)
		c.NotifyListeners(swt.KeyUp, up)
	}
	return nil
}

func isRow(e *entry) bool {
	switch e.n.Class {
	case "ListItem", "ComboItem", "TableItem", "TreeItem", "TabItem", "CTabItem":
		return true
	}
	return false
}

// rows are the entries a control selects from; a Tree's are all its items in pre-order.
func rows(e *entry) []*entry {
	var out []*entry
	var add func(*entry)
	add = func(p *entry) {
		for _, k := range p.kids {
			if isRow(k) {
				out = append(out, k)
				add(k)
			}
		}
	}
	add(e)
	return out
}

// selectItem selects a row, a tab or a list entry of a control by index or text. On such an entry
// itself it selects that entry and reads no arguments.
func selectItem(e *entry, index *int, text *string) error {
	ctl, i := e, -1
	if isRow(e) {
		ctl = e.parent
		for ctl.n.Class == "TreeItem" {
			ctl = ctl.parent
		}
		i = slices.Index(rows(ctl), e)
	} else {
		items := rows(e)
		switch {
		case len(items) == 0 && e.n.Class != "Table" && e.n.Class != "Tree":
			return fmt.Errorf("qatree: select does not apply to %s %s", e.n.Class, e.n.Path)
		case index != nil:
			i = *index
		case text != nil:
			for j, k := range items {
				if k.n.Text == *text {
					if i >= 0 {
						return fmt.Errorf("qatree: select: %q is in %s twice", *text, e.n.Path)
					}
					i = j
				}
			}
			if i < 0 {
				return fmt.Errorf("qatree: select: %q is not in %s", *text, e.n.Path)
			}
		default:
			return fmt.Errorf(`qatree: select needs "index" or "text"`)
		}
		if i < 0 || i >= len(items) {
			return fmt.Errorf("qatree: select: index %d out of %d in %s", i, len(items), e.n.Path)
		}
	}
	item := rows(ctl)[i]
	switch o := ctl.obj.(type) {
	case *swt.List:
		o.SetSelection(int32(i))
		o.NotifyListeners(swt.Selection, swt.NewEvent())
	case *swt.Combo:
		o.Select(int32(i))
		o.NotifyListeners(swt.Selection, swt.NewEvent())
	case *swt.Table:
		o.SetSelection(int32(i))
		ev := swt.NewEvent()
		ev.Item = &item.obj.(*swt.TableItem).Widget
		o.NotifyListeners(swt.Selection, ev)
	case *swt.Tree:
		ti := item.obj.(*swt.TreeItem)
		o.SetSelection(ti)
		ev := swt.NewEvent()
		ev.Item = &ti.Widget
		o.NotifyListeners(swt.Selection, ev)
	case *swt.TabFolder:
		selectTab(o, i)
	case *swt.CTabFolder:
		o.SetSelectionIndexNotify(int32(i), true)
	default:
		return fmt.Errorf("qatree: select does not apply to %s %s", ctl.n.Class, ctl.n.Path)
	}
	return nil
}
