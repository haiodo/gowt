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
	fa := cocoa.FieldAffix{Inset: cocoa.FieldInset(t.View), Prefix: a.prefix, Suffix: a.suffix, Hint: a.hint, Drag: a.drag}
	if a.image != nil {
		fa.Image = a.image.i.Handle
	}
	cocoa.SetFieldAffix(t.View, fa)
	if t.GetDataKey("gowt.fieldAffix") != nil {
		return
	}
	t.SetDataKeyValue("gowt.fieldAffix", true)
	v := t.View
	t.AddDisposeListener(disposeFunc(func() { cocoa.ReleaseFieldAffix(v) }))
}
