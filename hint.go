package gowt

import "github.com/haiodo/gowt/swt"

// multiHint shows the hint of a multi-line Text, which SWT sets on single-line fields only: no
// platform's multi-line editor has a public placeholder. paintHint draws it on each platform.
func multiHint(t *swt.Text) {
	if t.GetDataKey("gowt.hint") == nil {
		t.SetDataKeyValue("gowt.hint", true)
		empty := t.GetCharCount() == 0
		t.AddModifyListener(&modifier{func() {
			if now := t.GetCharCount() == 0; now != empty {
				empty = now
				t.Redraw()
			}
		}})
		paintHint(t)
		r := &hintTheme{t}
		t.GetDisplay().AddListener(swt.Settings, r)
		t.AddListener(swt.Dispose, r)
	}
	t.Redraw()
}

// hintTheme repaints the hint in the placeholder colour of a new system theme.
type hintTheme struct{ t *swt.Text }

func (h *hintTheme) HandleEvent(e *swt.Event) {
	if e.Type == swt.Dispose {
		h.t.GetDisplay().RemoveListener(swt.Settings, h)
		return
	}
	h.t.Redraw()
}
