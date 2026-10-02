package gowt

import "github.com/haiodo/gowt/swt"

// Menu is a menu bar, a popup menu or a submenu.
type Menu struct{ m *swt.Menu }

// MenuBar creates the window's menu bar. On macOS it becomes the application menu bar.
func (w *Window) MenuBar() *Menu {
	m := swt.NewMenuParentStyle(w.shell, swt.BAR)
	w.shell.SetMenuBar(m)
	return &Menu{m}
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
// A "&" in text marks a mnemonic; text after "\t" is shown as the shortcut hint (it does not bind a key).
func (m *Menu) Item(text string, onClick func(), opts ...Option) *MenuItem {
	it := swt.NewMenuItem(m.m, resolve(swt.NONE, opts))
	it.SetText(text)
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

func (m *Menu) Unwrap() *swt.Menu { return m.m }

// MenuItem is a menu entry.
type MenuItem struct{ i *swt.MenuItem }

func (i *MenuItem) SetText(s string)      { i.i.SetText(s) }
func (i *MenuItem) SetEnabled(v bool)     { i.i.SetEnabled(v) }
func (i *MenuItem) Checked() bool         { return i.i.GetSelection() }
func (i *MenuItem) SetChecked(v bool)     { i.i.SetSelection(v) }
func (i *MenuItem) Unwrap() *swt.MenuItem { return i.i }

// OnClick runs f when the item is chosen (or toggled).
func (i *MenuItem) OnClick(f func()) {
	onSelect(i.i, func(*swt.SelectionEvent) { f() })
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

func (t *TrayIcon) SetImage(img *Image)   { t.i.SetImage(img.i) }
func (t *TrayIcon) SetTooltip(s string)   { t.i.SetToolTipText(s) }
func (t *TrayIcon) Remove()               { t.i.Dispose() }
func (t *TrayIcon) Unwrap() *swt.TrayItem { return t.i }

// OnClick runs f when the icon is clicked.
func (t *TrayIcon) OnClick(f func()) {
	onSelect(t.i, func(*swt.SelectionEvent) { f() })
}
