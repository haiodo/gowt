package gowt

import (
	"github.com/haiodo/gowt/internal/fluent"
	"github.com/haiodo/gowt/swt"
)

// smallTextHeightDips is the height of a compact WinUI 3 TextBox.
const smallTextHeightDips = 24

// setFieldSmall has nothing to add to the caption font: fluentText takes the compact height.
func setFieldSmall(*swt.Text) {}

// setFieldAffix paints the affixes in the WinUI frame, the non-client area of the edit.
func setFieldAffix(t *swt.Text, a *affix) {
	fa := fluent.Affix{Prefix: a.prefix, Suffix: a.suffix, Drag: a.drag}
	if a.image != nil {
		fa.Image = a.image.i
	}
	extra := fluent.SetAffix(t, fa)
	if w, _, ok := sizeHints(&t.Control); ok && *w == swt.DEFAULT && extra > 0 {
		*w = preferred(&t.Control).X + int32(extra)
	}
}
