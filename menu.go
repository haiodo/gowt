package gowt

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/haiodo/gowt/swt"
)

// Menu is a menu bar, a popup menu or a submenu.
type Menu struct{ m *swt.Menu }

// MenuBar creates the window's menu bar. On macOS it becomes the application menu bar.
func (w *Window) MenuBar() *Menu {
	m := swt.NewMenuParentStyle(w.shell, swt.BAR)
	w.shell.SetMenuBar(m)
	return &Menu{m}
}

// OnAbout runs f when the About item of the macOS application menu is chosen and reports true. Elsewhere
// there is no such menu: it reports false and the app puts its own About item in a normal menu. UI thread only.
//
//	if !app.OnAbout(showAbout) {
//		help.Item("About", showAbout)
//	}
func (a *App) OnAbout(f func()) bool { return a.onSystemItem(swt.ID_ABOUT, f) }

// OnPreferences is OnAbout for the Preferences item (Cmd+,), which macOS keeps in the application menu;
// elsewhere put the item in Edit or Tools.
func (a *App) OnPreferences(f func()) bool { return a.onSystemItem(swt.ID_PREFERENCES, f) }

func (a *App) onSystemItem(id int32, f func()) bool {
	m := a.display.GetSystemMenu()
	if m == nil {
		return false
	}
	for _, it := range m.GetItems() {
		if it.GetID() == id {
			onSelect(it, func(*swt.SelectionEvent) { f() })
			return true
		}
	}
	return false
}

// PopupMenu creates the context menu of w. It is a function, not a method, because every
// wrapper would need its own copy; Show opens it by hand, right click opens it by itself.
func PopupMenu(w Widget) *Menu {
	c := w.control()
	m := swt.NewMenu(c)
	c.SetMenu(m)
	return &Menu{m}
}

// Item appends a push item (Check or Radio for other kinds). onClick may be nil.
// A "&" in text marks a mnemonic. Shortcut binds a key and shows it; text after "\t" is only shown.
// Options: Check, Radio, Shortcut.
func (m *Menu) Item(text string, onClick func(), opts ...Option) *MenuItem {
	it := swt.NewMenuItem(m.m, resolve(swt.NONE, opts))
	it.SetText(text)
	for _, o := range opts {
		if f, ok := o.(menuOpt); ok {
			f(it)
		}
	}
	itemOpts(it, "Menu.Item", sCheck|sRadio, true, opts)
	w := &MenuItem{it}
	if onClick != nil {
		w.OnClick(onClick)
	}
	return w
}

// Submenu appends an item that opens and returns a nested menu.
func (m *Menu) Submenu(text string) *Menu {
	it := swt.NewMenuItem(m.m, swt.CASCADE)
	it.SetText(text)
	sub := swt.NewMenuParentMenu(m.m)
	it.SetMenu(sub)
	return &Menu{sub}
}

// Separator appends a divider.
func (m *Menu) Separator() { swt.NewMenuItem(m.m, swt.SEPARATOR) }

// Show opens the popup menu at the pointer.
func (m *Menu) Show() { m.m.SetVisible(true) }

// OnShow runs f just before the menu opens, to refresh enabled states.
func (m *Menu) OnShow(f func()) {
	m.m.AddMenuListener(swt.MenuListenerMenuShownAdapter(func(*swt.MenuEvent) { f() }))
}

// Unwrap returns the underlying swt.Menu for the full API.
func (m *Menu) Unwrap() *swt.Menu { return m.m }

// MenuItem is a menu entry.
type MenuItem struct{ i *swt.MenuItem }

// SetText replaces the item text; "&" marks the mnemonic. The key of a Shortcut stays shown unless s
// has a "\t" part of its own.
func (i *MenuItem) SetText(s string) {
	if _, hint, ok := strings.Cut(i.i.GetText(), "\t"); ok && i.i.GetAccelerator() != 0 && !strings.Contains(s, "\t") {
		s += "\t" + hint
	}
	i.i.SetText(s)
}

// SetEnabled enables or greys out the item.
func (i *MenuItem) SetEnabled(v bool) { i.i.SetEnabled(v) }

// Checked reports the state of a check or radio item.
func (i *MenuItem) Checked() bool { return i.i.GetSelection() }

// SetChecked sets the state of a check or radio item.
func (i *MenuItem) SetChecked(v bool) { i.i.SetSelection(v) }

// Unwrap returns the underlying swt.MenuItem for the full API.
func (i *MenuItem) Unwrap() *swt.MenuItem { return i.i }

// OnClick runs f when the item is chosen (or toggled).
func (i *MenuItem) OnClick(f func()) {
	onSelect(i.i, func(*swt.SelectionEvent) { f() })
}

// Shortcut binds keys to a menu item and shows them next to its text: Shortcut("Cmd+O"),
// Shortcut("Cmd+Shift+S"), Shortcut("Alt+Left"). Cmd is Command on macOS and Ctrl elsewhere; Ctrl,
// Alt and Shift are the same keys on every OS. The last part is a letter, digit or punctuation mark,
// or one of Left, Right, Up, Down, Home, End, PageUp, PageDown, Enter, Esc, Tab, Space, Backspace,
// Delete, Insert, F1-F12, in any case. Shortcut panics on a name it does not know.
func Shortcut(keys string) Option {
	accel, label, err := parseShortcut(keys, runtime.GOOS)
	if err != nil {
		panic(fmt.Errorf("gowt: Shortcut(%q): %w", keys, err))
	}
	return menuOpt(func(it *swt.MenuItem) {
		it.SetAccelerator(accel)
		text, _, _ := strings.Cut(it.GetText(), "\t")
		it.SetText(text + "\t" + label) // macOS shows the bound key itself and drops the text after "\t"
	})
}

var namedKeys = []struct {
	name string
	code int32
}{
	{"Left", swt.ARROW_LEFT}, {"Right", swt.ARROW_RIGHT}, {"Up", swt.ARROW_UP}, {"Down", swt.ARROW_DOWN},
	{"Home", swt.HOME}, {"End", swt.END}, {"PageUp", swt.PAGE_UP}, {"PageDown", swt.PAGE_DOWN},
	{"Enter", int32(swt.CR)}, {"Esc", int32(swt.ESC)}, {"Tab", int32(swt.TAB)}, {"Space", int32(swt.SPACE)},
	{"Backspace", int32(swt.BS)}, {"Delete", int32(swt.DEL)}, {"Insert", swt.INSERT},
}

// parseShortcut returns the SWT accelerator of keys on goos and the text that shows it.
func parseShortcut(keys, goos string) (accel int32, label string, err error) {
	parts := strings.Split(keys, "+")
	if strings.HasSuffix(keys, "++") {
		parts = append(parts[:len(parts)-2], "+") // "Cmd++": the key is the plus sign
	}
	words := make([]string, len(parts))
	for i, p := range parts[:len(parts)-1] {
		switch strings.ToLower(p) {
		case "cmd":
			m, w := swt.CTRL, "Ctrl"
			if goos == "darwin" {
				m, w = swt.COMMAND, "Cmd"
			}
			accel, words[i] = accel|m, w
		case "ctrl":
			accel, words[i] = accel|swt.CTRL, "Ctrl"
		case "alt":
			accel, words[i] = accel|swt.ALT, "Alt"
		case "shift":
			accel, words[i] = accel|swt.SHIFT, "Shift"
		default:
			return 0, "", fmt.Errorf("unknown modifier %q, want Cmd, Ctrl, Alt or Shift", p)
		}
	}
	code, name := shortcutKey(parts[len(parts)-1])
	if code == 0 {
		return 0, "", fmt.Errorf("unknown key %q", parts[len(parts)-1])
	}
	words[len(words)-1] = name
	return accel | code, strings.Join(words, "+"), nil
}

// shortcutKey returns the SWT key code and the shown name of one key, 0 if it has none.
func shortcutKey(k string) (int32, string) {
	if r, n := utf8.DecodeRuneInString(k); n == len(k) && r > ' ' && r < 0x7f {
		r = unicode.ToUpper(r)
		return r, string(r)
	}
	for _, nk := range namedKeys {
		if strings.EqualFold(k, nk.name) {
			return nk.code, nk.name
		}
	}
	if num, ok := strings.CutPrefix(strings.ToUpper(k), "F"); ok {
		if n, err := strconv.Atoi(num); err == nil && n >= 1 && n <= 12 {
			return swt.F1 + int32(n-1), "F" + strconv.Itoa(n)
		}
	}
	return 0, ""
}

// TrayIcon is an icon in the system tray / status area.
type TrayIcon struct{ i *swt.TrayItem }

// Tray adds a tray icon, or returns nil where the platform has no system tray. onClick may be nil.
func (a *App) Tray(tooltip string, onClick func()) *TrayIcon {
	t := a.display.GetSystemTray()
	if t == nil {
		return nil
	}
	it := swt.NewTrayItem(t, swt.NONE)
	it.SetToolTipText(tooltip)
	w := &TrayIcon{it}
	if onClick != nil {
		w.OnClick(onClick)
	}
	return w
}

// SetImage sets the icon; the tray does not own img.
func (t *TrayIcon) SetImage(img *Image) { t.i.SetImage(img.i) }

// SetTooltip sets the text shown on hover.
func (t *TrayIcon) SetTooltip(s string) { t.i.SetToolTipText(s) }

// Remove deletes the icon from the tray.
func (t *TrayIcon) Remove() { t.i.Dispose() }

// Unwrap returns the underlying swt.TrayItem for the full API.
func (t *TrayIcon) Unwrap() *swt.TrayItem { return t.i }

// OnClick runs f when the icon is clicked.
func (t *TrayIcon) OnClick(f func()) {
	onSelect(t.i, func(*swt.SelectionEvent) { f() })
}

// OnKey runs f when keys are pressed anywhere in the window: w.OnKey("Cmd+[", back). keys is written
// as for Shortcut. A focused text field (Text, StyledText, Spinner, an editable Combo) keeps the
// combinations it types or edits with, and f does not run for them there; see editsText. OnKey panics
// on a name it does not know.
func (w *Window) OnKey(keys string, f func()) {
	accel, _, err := parseShortcut(keys, runtime.GOOS)
	if err != nil {
		panic(fmt.Errorf("gowt: OnKey(%q): %w", keys, err))
	}
	d := w.shell.GetDisplay()
	k := &keyBinding{w.shell, accel, editsText(accel, runtime.GOOS), f}
	d.AddFilter(swt.KeyDown, k)
	w.shell.AddDisposeListener(disposeFunc(func() { d.RemoveFilter(swt.KeyDown, k) }))
}

type keyBinding struct {
	shell *swt.Shell
	accel int32
	edits bool // a text field keeps the keys
	f     func()
}

func (k *keyBinding) HandleEvent(e *swt.Event) {
	if !keyMatches(k.accel, e.StateMask, e.KeyCode) || e.Widget == nil {
		return
	}
	c, ok := e.Widget.Impl().(interface{ GetShell() *swt.Shell })
	if !ok || c.GetShell() != k.shell || k.edits && isTextField(e.Widget) {
		return
	}
	e.Doit = false
	k.f()
}

// keyMatches reports whether a key press with the modifiers in state and the key code matches accel.
// SWT reports a letter key in lower case whatever the modifiers.
func keyMatches(accel, state, code int32) bool {
	if state&swt.MODIFIER_MASK != accel&swt.MODIFIER_MASK {
		return false
	}
	if code >= 'a' && code <= 'z' {
		code -= 'a' - 'A'
	}
	return code == accel&swt.KEY_MASK
}

func isTextField(w *swt.Widget) bool {
	switch c := w.Impl().(type) {
	case *swt.Text, *swt.StyledText, *swt.Spinner:
		return true
	case *swt.Combo:
		return c.GetStyle()&swt.READ_ONLY == 0
	case *swt.CCombo:
		return c.GetStyle()&swt.READ_ONLY == 0
	}
	return false
}

// editsText reports whether a text field on goos types or edits with accel: a character, an arrow, Home,
// End, PageUp, PageDown, Enter or Tab alone; Backspace or Delete with any modifiers; Cmd+A, C, V, X, Y
// or Z; on macOS the movement keys with any modifiers and a character with Option or Ctrl; elsewhere the
// movement keys with Ctrl and a character with Ctrl+Alt (AltGr). Shift does not count.
func editsText(accel int32, goos string) bool {
	mods := accel & swt.MODIFIER_MASK &^ swt.SHIFT
	key := accel & swt.KEY_MASK
	char := key >= ' ' && key < 0x7f
	move := key == swt.ARROW_LEFT || key == swt.ARROW_RIGHT || key == swt.ARROW_UP || key == swt.ARROW_DOWN ||
		key == swt.HOME || key == swt.END || key == swt.PAGE_UP || key == swt.PAGE_DOWN
	mac := goos == "darwin"
	switch {
	case key == int32(swt.BS) || key == int32(swt.DEL):
		return true
	case mods == 0:
		return char || move || key == int32(swt.CR) || key == int32(swt.TAB)
	case move:
		return mac || mods&swt.CTRL != 0
	case !char:
		return false
	}
	clip := int32(swt.CTRL)
	if mac {
		clip = swt.COMMAND
	}
	switch {
	case mods == clip:
		return strings.ContainsRune("ACVXYZ", key)
	case mac:
		return mods == swt.ALT || mods == swt.CTRL
	}
	return mods == swt.CTRL|swt.ALT
}
