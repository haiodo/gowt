//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

// paintHint draws the hint where the first character goes: a Win32 multi-line edit ignores
// EM_SETCUEBANNER and a GtkTextView has no placeholder.
func paintHint(t *swt.Text) {
	paintOver(t, func(g *swt.GC) {
		hint := t.GetMessage()
		if hint == "" || t.GetCharCount() != 0 {
			return
		}
		x, y := hintOrigin(t)
		g.SetFont(t.GetFont())
		g.SetForeground(hintColor(t))
		g.DrawTextStringXYFlags(hint, x, y, swt.DRAW_DELIMITER|swt.DRAW_TAB|swt.DRAW_TRANSPARENT)
	})
}
