package webview

import (
	"github.com/haiodo/gowt/swt"
)

// pageDialog shows a dialog of the page itself and reports whether it was accepted, with the
// prompt's text. A variable so that tests can answer without a modal loop.
var pageDialog = showPageDialog

func showPageDialog(parent *swt.Shell, kind int32, msg, def string) (bool, string) {
	switch kind {
	case wv2DialogPrompt:
		return promptBox(parent, msg, def)
	case wv2DialogAlert:
		return messageBox(parent, swt.ICON_INFORMATION|swt.OK, msg), ""
	case wv2DialogBeforeUnload:
		if msg == "" {
			msg = "Leave this page? Changes you made may not be saved."
		}
	}
	return messageBox(parent, swt.ICON_QUESTION|swt.OK|swt.CANCEL, msg), ""
}

func messageBox(parent *swt.Shell, style int32, msg string) bool {
	b := swt.NewMessageBoxParentStyle(parent, style)
	b.SetText("This page says")
	b.SetMessage(msg)
	return b.Open()&swt.OK != 0
}

// promptBox is a modal input box; closing it by the title bar cancels.
func promptBox(parent *swt.Shell, msg, def string) (bool, string) {
	s := swt.NewShellParentStyle(parent, swt.DIALOG_TRIM|swt.APPLICATION_MODAL)
	s.SetText("This page says")
	s.SetLayout(swt.NewGridLayoutNumColumnsMakeColumnsEqualWidth(2, true))
	l := swt.NewLabel(s, swt.WRAP)
	l.SetText(msg)
	ld := swt.NewGridData()
	ld.HorizontalSpan, ld.WidthHint = 2, 320
	l.SetLayoutData(ld)
	t := swt.NewText(s, swt.SINGLE|swt.BORDER)
	t.SetText(def)
	t.SelectAll()
	td := swt.NewGridData()
	td.HorizontalSpan, td.HorizontalAlignment, td.GrabExcessHorizontalSpace = 2, swt.FILL, true
	t.SetLayoutData(td)
	var text string
	var ok bool
	button := func(label string, accept bool) *swt.Button {
		b := swt.NewButton(s, swt.PUSH)
		b.SetText(label)
		b.SetLayoutData(swt.NewGridDataStyle(swt.GridDataFILL_HORIZONTAL))
		b.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(*swt.SelectionEvent) {
			if accept {
				text, ok = t.GetText(), true
			}
			s.Close()
		}))
		return b
	}
	s.SetDefaultButton(button("OK", true))
	button("Cancel", false)
	s.Pack()
	s.Open()
	d := s.GetDisplay()
	for !s.IsDisposed() {
		if !d.ReadAndDispatch() {
			d.Sleep()
		}
	}
	return ok, text
}
