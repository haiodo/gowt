package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// An edit control cannot hide its scroll bar when there is nothing to scroll, so Multiline asks for none
// and Scrollbars() brings the always-visible one back.
const multilineScroll = 0

// textHeightDips is the height of a WinUI 3 TextBox.
const textHeightDips = 32

// fluentText gives a Text the Windows 11 text box look; noBorder (NoBorder) keeps the plain edit.
func fluentText(t *TextView, noBorder bool) {
	if noBorder {
		return
	}
	fluent.Text(t.t)
	if t.multi() {
		if t.inset == [2]int{} {
			t.SetInset(0, 5)
		}
		return
	}
	if _, h, ok := sizeHints(&t.t.Control); ok && *h == swt.DEFAULT {
		*h = max(textHeightDips, preferred(&t.t.Control).Y)
	}
}

// fluentBorder replaces the 3D edge of a container, list or table with a flat hairline.
func fluentBorder(c *swt.Control) {
	if c.GetStyle()&swt.BORDER == 0 {
		return
	}
	switch c.GetName() {
	case "Composite", "ScrolledComposite", "Table", "Tree", "List", "StyledText":
		fluent.FlatBorder(c)
	}
}

func fluentTable(t *swt.Table) { fluent.Table(t) }
