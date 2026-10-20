//go:build !darwin

package gowt

import "github.com/haiodo/gowt/swt"

// multiHint draws the hint of a multi-line Text where the first character goes: a Win32
// multi-line edit ignores EM_SETCUEBANNER and a GtkTextView has no placeholder.
func multiHint(t *swt.Text, _ string) {
	if t.GetDataKey("gowt.hint") == nil {
		t.SetDataKeyValue("gowt.hint", true)
		empty := t.GetCharCount() == 0
		t.AddModifyListener(&modifier{func() {
			if now := t.GetCharCount() == 0; now != empty {
				empty = now
				t.Redraw()
			}
		}})
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
	t.Redraw()
}
