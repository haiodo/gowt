package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// setFieldSmall sets NSControlSizeSmall: the bezel, corners and focus ring of a small field.
func setFieldSmall(t *swt.Text) { cocoa.SetControlSize(t.View, 1) }

// setFieldAffix draws the affixes in the field's own cell: bezel, focus ring and caret stay AppKit's,
// and the field editor keeps clear of them.
func setFieldAffix(t *swt.Text, a *affix) {
	fa := cocoa.FieldAffix{Prefix: a.prefix, Suffix: a.suffix, Drag: a.drag}
	if a.image != nil {
		fa.Image = a.image.i.Handle
	}
	cocoa.SetFieldAffix(t.View, fa)
	v := t.View
	t.AddDisposeListener(disposeFunc(func() { cocoa.ReleaseFieldAffix(v) }))
}
