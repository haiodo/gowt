package qatree

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/haiodo/gowt/swt"
)

// controlOf is the control behind an entry: a row belongs to the control that lists it.
func controlOf(e *entry) (*swt.Control, error) {
	for isRow(e) && e.parent != nil {
		e = e.parent
	}
	if c, ok := e.obj.(interface{ AsControl() *swt.Control }); ok {
		return c.AsControl(), nil
	}
	return nil, fmt.Errorf("%s %s is not a control", e.n.Class, e.n.Path)
}

// owner is the List, Table or Tree that holds the row e.
func owner(e *entry) *entry {
	ctl := e.parent
	for ctl.n.Class == "TreeItem" {
		ctl = ctl.parent
	}
	return ctl
}

// activate sends DefaultSelection; a row is selected first, as the click before a double click does.
func activate(e *entry) error {
	ctl, row := e, (*entry)(nil)
	if isRow(e) {
		ctl, row = owner(e), e
		if c := ctl.n.Class; c != "List" && c != "Table" && c != "Tree" {
			return fmt.Errorf("qatree: activate does not apply to %s %s", e.n.Class, e.n.Path)
		}
		if err := selectItem(e, nil, nil); err != nil {
			return err
		}
	}
	ev := swt.NewEvent()
	switch o := ctl.obj.(type) {
	case *swt.Button:
		if o.GetStyle()&swt.PUSH == 0 {
			return fmt.Errorf("qatree: activate: %s is not a push button, use check", e.n.Path)
		}
		clickButton(o)
		return nil
	case *swt.Shell:
		b := o.GetDefaultButton()
		if b == nil || !b.GetEnabled() {
			return fmt.Errorf("qatree: activate: %s has no enabled default button", e.n.Path)
		}
		clickButton(b)
		return nil
	case *swt.Text:
		if o.GetStyle()&swt.SINGLE == 0 {
			return fmt.Errorf("qatree: activate: Enter in the multi-line %s adds a line, use key or type", e.n.Path)
		}
		swt.QAFocus(&o.Control)
		o.NotifyListeners(swt.DefaultSelection, ev)
		return nil
	case *swt.List:
		if row == nil && len(ctl.n.Selected) == 0 {
			return fmt.Errorf("qatree: activate: nothing is selected in %s", e.n.Path)
		}
		o.NotifyListeners(swt.DefaultSelection, ev)
	case *swt.Table:
		item, err := activeRow(ctl, row)
		if err != nil {
			return err
		}
		ev.Item = &item.obj.(*swt.TableItem).Widget
		o.NotifyListeners(swt.DefaultSelection, ev)
	case *swt.Tree:
		item, err := activeRow(ctl, row)
		if err != nil {
			return err
		}
		ev.Item = &item.obj.(*swt.TreeItem).Widget
		o.NotifyListeners(swt.DefaultSelection, ev)
	default:
		return fmt.Errorf("qatree: activate does not apply to %s %s", ctl.n.Class, ctl.n.Path)
	}
	return nil
}

// activeRow is row, or the first selected row of ctl.
func activeRow(ctl, row *entry) (*entry, error) {
	if row != nil {
		return row, nil
	}
	if len(ctl.n.Selected) == 0 {
		return nil, fmt.Errorf("qatree: activate: nothing is selected in %s", ctl.n.Path)
	}
	return rows(ctl)[ctl.n.Selected[0]], nil
}

// setExpanded sends Expand (Collapse) first, as SWT does before it shows (hides) the children, so
// a handler can build them, then changes the state, which sends no event.
func setExpanded(e *entry, on bool) error {
	it, ok := e.obj.(*swt.TreeItem)
	if !ok {
		return fmt.Errorf("qatree: expand and collapse apply to a TreeItem, not %s %s", e.n.Class, e.n.Path)
	}
	if it.GetExpanded() == on {
		return nil
	}
	if on && len(it.GetItems()) == 0 {
		return fmt.Errorf("qatree: expand: %s has no children; a lazy node needs a placeholder child", e.n.Path)
	}
	ev := swt.NewEvent()
	ev.Item = &it.Widget
	t := it.GetParent()
	if on {
		t.NotifyListeners(swt.Expand, ev)
	} else {
		t.NotifyListeners(swt.Collapse, ev)
	}
	if !it.IsDisposed() {
		it.SetExpanded(on)
	}
	return nil
}

func focus(d *swt.Display, e *entry) error {
	c, err := controlOf(e)
	if err != nil {
		return err
	}
	if _, ok := e.obj.(*swt.Shell); ok {
		return fmt.Errorf("qatree: focus applies to a control, not the window %s", e.n.Path)
	}
	swt.QAFocus(c)
	flush(d)
	if d.GetFocusControl() != c {
		return fmt.Errorf("qatree: focus: %s did not take the focus", e.n.Path)
	}
	return nil
}

func closeShell(e *entry) error {
	s, ok := e.obj.(*swt.Shell)
	if !ok {
		return fmt.Errorf("qatree: close applies to a window, not %s %s", e.n.Class, e.n.Path)
	}
	s.Close()
	return nil
}

// scroll shows a control inside its ScrollView, or moves the origin of a ScrollView by dx, dy.
func scroll(d *swt.Display, e *entry, to *Selector, dx, dy int) error {
	c, err := controlOf(e)
	if err != nil {
		return err
	}
	sc, isScroll := e.obj.(*swt.ScrolledComposite)
	if !isScroll {
		if sc = scrollParent(c); sc == nil {
			return fmt.Errorf("qatree: scroll: %s %s is not in a ScrollView", e.n.Class, e.n.Path)
		}
		sc.ShowControl(c)
		flush(d)
		return nil
	}
	switch {
	case to != nil:
		es, err := resolve(walk(d), *to)
		if err != nil {
			return err
		}
		if len(es) != 1 {
			return fmt.Errorf("qatree: scroll to %s: %d controls match%s", to, len(es), paths(es))
		}
		if !es[0].n.Visible {
			return fmt.Errorf("qatree: scroll to %s: not visible", es[0].n.Path)
		}
		tc, err := controlOf(es[0])
		if err != nil {
			return err
		}
		if scrollParent(tc) != sc {
			return fmt.Errorf("qatree: scroll: %s is not inside %s", es[0].n.Path, e.n.Path)
		}
		sc.ShowControl(tc)
	case dx != 0 || dy != 0:
		o := sc.GetOrigin()
		sc.SetOriginXY(o.X+int32(dx), o.Y+int32(dy))
	default:
		return fmt.Errorf(`qatree: scroll on %s needs "to", "dx" or "dy"`, e.n.Path)
	}
	flush(d)
	return nil
}

// scrollParent is the nearest ScrolledComposite above c.
func scrollParent(c *swt.Control) *swt.ScrolledComposite {
	for p := c.GetParent(); p != nil; p = p.GetParent() {
		if sc, ok := p.Impl().(*swt.ScrolledComposite); ok {
			return sc
		}
	}
	return nil
}

// keyTarget is the control that gets the keys when no selector names one: the focus control, else
// the active window, else the one modal window or the only window.
func keyTarget(d *swt.Display) (*swt.Control, error) {
	if c := d.GetFocusControl(); c != nil {
		return c, nil
	}
	if s := d.GetActiveShell(); s != nil {
		return &s.Control, nil
	}
	var open []*swt.Shell
	for _, s := range d.GetShells() {
		if s.GetVisible() && !s.IsDisposed() {
			open = append(open, s)
		}
	}
	for _, s := range open {
		if blocker(d, s) == nil && len(open) > 1 {
			continue
		}
		return &s.Control, nil
	}
	return nil, fmt.Errorf("qatree: key: no focus control and no single window to send to, name a control")
}

// keyFocused sends the keys to the focus control, refusing one a modal window blocks.
func keyFocused(d *swt.Display, args json.RawMessage) error {
	var a struct{ Keys string }
	if err := parse(args, &a); err != nil {
		return err
	}
	c, err := keyTarget(d)
	if err != nil {
		return err
	}
	if m := blocker(d, c.GetShell()); m != nil {
		return fmt.Errorf("qatree: key: blocked by the modal window %q", m.GetText())
	}
	return key(d, c, a.Keys)
}

// keyEvent reads "Mod+K", "Shift+Tab", "Enter": modifiers Mod (Cmd on macOS, Ctrl elsewhere), Cmd
// (macOS only), Ctrl, Alt, Shift, then one key.
func keyEvent(keys string) (state, code int32, char uint16, err error) {
	parts := strings.Split(keys, "+")
	if strings.HasSuffix(keys, "++") {
		parts = append(parts[:len(parts)-2], "+")
	}
	for _, p := range parts[:len(parts)-1] {
		switch strings.ToLower(p) {
		case "mod":
			if runtime.GOOS == "darwin" {
				state |= swt.COMMAND
			} else {
				state |= swt.CTRL
			}
		case "cmd":
			if runtime.GOOS != "darwin" {
				return 0, 0, 0, fmt.Errorf("qatree: key %q: Cmd exists on macOS only, use Mod", keys)
			}
			state |= swt.COMMAND
		case "ctrl":
			state |= swt.CTRL
		case "alt":
			state |= swt.ALT
		case "shift":
			state |= swt.SHIFT
		default:
			return 0, 0, 0, fmt.Errorf("qatree: key %q: unknown modifier %q, want Mod, Cmd, Ctrl, Alt or Shift", keys, p)
		}
	}
	k := parts[len(parts)-1]
	code = namedKey(k)
	if r, n := utf8.DecodeRuneInString(k); code == 0 && n == len(k) && r > ' ' && r < 0x7f {
		code = int32(unicode.ToLower(r)) // SWT reports a letter in lower case
	}
	if code == 0 {
		return 0, 0, 0, fmt.Errorf("qatree: key %q: unknown key %q", keys, k)
	}
	if code < 0x80 && state&(swt.COMMAND|swt.CTRL|swt.ALT) == 0 {
		char = uint16(code)
		if state&swt.SHIFT != 0 {
			char = uint16(unicode.ToUpper(rune(code)))
		}
	}
	return state, code, char, nil
}

func namedKey(k string) int32 {
	for _, nk := range []struct {
		name string
		code int32
	}{
		{"Left", swt.ARROW_LEFT}, {"Right", swt.ARROW_RIGHT}, {"Up", swt.ARROW_UP}, {"Down", swt.ARROW_DOWN},
		{"Home", swt.HOME}, {"End", swt.END}, {"PageUp", swt.PAGE_UP}, {"PageDown", swt.PAGE_DOWN},
		{"Enter", int32(swt.CR)}, {"Esc", int32(swt.ESC)}, {"Tab", int32(swt.TAB)}, {"Space", int32(swt.SPACE)},
		{"Backspace", int32(swt.BS)}, {"Delete", int32(swt.DEL)}, {"Insert", swt.INSERT},
	} {
		if strings.EqualFold(k, nk.name) {
			return nk.code
		}
	}
	if num, ok := strings.CutPrefix(strings.ToUpper(k), "F"); ok {
		if n, err := strconv.Atoi(num); err == nil && n >= 1 && n <= 12 {
			return swt.F1 + int32(n-1)
		}
	}
	return 0
}

// key runs the menu item with this accelerator, if any: macOS key equivalents see the key before
// the control, so the control gets nothing. Otherwise KeyDown (filters, listeners, Doit veto), KeyUp.
func key(d *swt.Display, c *swt.Control, keys string) error {
	state, code, char, err := keyEvent(keys)
	if err != nil {
		return err
	}
	if bar := c.GetShell().GetMenuBar(); bar != nil {
		if it := accelerated(bar, state, code); it != nil {
			return click(&entry{obj: it, n: Node{Path: "accelerator " + keys}})
		}
	}
	mk := func() *swt.Event {
		ev := swt.NewEvent()
		ev.KeyCode, ev.Character, ev.StateMask = code, char, state
		return ev
	}
	down := mk()
	c.NotifyListeners(swt.KeyDown, down)
	if !c.IsDisposed() {
		if down.Doit && state == 0 && code == int32(swt.CR) {
			defaultSelection(c)
		}
		c.NotifyListeners(swt.KeyUp, mk())
	}
	flush(d)
	return nil
}

// accelerated finds the enabled item of m, submenus included, that has the accelerator state|code.
func accelerated(m *swt.Menu, state, code int32) *swt.MenuItem {
	for _, it := range m.GetItems() {
		if !it.GetEnabled() {
			continue
		}
		if sub := it.GetMenu(); sub != nil {
			if f := accelerated(sub, state, code); f != nil {
				return f
			}
		}
		a := it.GetAccelerator()
		if a != 0 && it.GetStyle()&swt.CASCADE == 0 && a&swt.MODIFIER_MASK == state &&
			unicode.ToLower(rune(a&^swt.MODIFIER_MASK)) == unicode.ToLower(rune(code)) {
			return it
		}
	}
	return nil
}

// defaultSelection is what Enter does in c itself, where the control has such a thing.
func defaultSelection(c *swt.Control) {
	ev := swt.NewEvent()
	switch o := c.Impl().(type) {
	case *swt.Text:
		if o.GetStyle()&swt.SINGLE == 0 {
			return
		}
	case *swt.List:
	case *swt.Table:
		if sel := o.GetSelection(); len(sel) > 0 {
			ev.Item = &sel[0].Widget
		}
	case *swt.Tree:
		if sel := o.GetSelection(); len(sel) > 0 {
			ev.Item = &sel[0].Widget
		}
	default:
		return
	}
	c.NotifyListeners(swt.DefaultSelection, ev)
}
