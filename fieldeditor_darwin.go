package gowt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

// centreField keeps the text of a borderless field on its line: AppKit puts it at the top of the
// frame, which a tree editor as tall as its row shows. The cell only gets a centred title rect;
// nothing is drawn in the frame and no inset or scrub handle is added.
func centreField(t *swt.Text) {
	cocoa.CentreFieldText(t.View)
	if t.GetDataKey("gowt.fieldAffix") != nil {
		return
	}
	t.SetDataKeyValue("gowt.fieldAffix", true)
	v := t.View
	t.AddDisposeListener(disposeFunc(func() { cocoa.ReleaseFieldAffix(v) }))
}

// placeFieldEditor puts the caret of a styled field on the line of its text, where AppKit does not.
func placeFieldEditor(t *swt.Text) { cocoa.PlaceFieldEditor(t.View) }
