//go:build darwin

package cocoa

import (
	"fmt"
	"os"

	"github.com/ebitengine/purego"
)

// Kinds of the page's own dialogs, as WKView.Dialog gets them.
const (
	WKDialogAlert = iota
	WKDialogConfirm
	WKDialogPrompt
)

const nsAlertFirstButtonReturn = 1000

// wkDialog is a sheet on the view's window with the WebKit completion handler it must answer once.
type wkDialog struct {
	kind         int
	alert, field uintptr
	handler      uintptr // copied: the sheet outlives the delegate call
	done         bool
}

var blockCopyFn, blockReleaseFn uintptr

func blockCopy(b uintptr) uintptr {
	if blockCopyFn == 0 {
		blockCopyFn, _ = purego.Dlsym(purego.RTLD_DEFAULT, "_Block_copy")
		blockReleaseFn, _ = purego.Dlsym(purego.RTLD_DEFAULT, "_Block_release")
	}
	r, _, _ := purego.SyscallN(blockCopyFn, b)
	return r
}

func blockRelease(b uintptr) { purego.SyscallN(blockReleaseFn, b) }

// answerDialog calls the WebKit handler: alert takes nothing, confirm a BOOL, prompt a string or nil.
func answerDialog(kind int, handler uintptr, ok bool, text string) {
	if wkTrace {
		fmt.Fprintf(os.Stderr, "wk: dialog answer kind=%d ok=%v\n", kind, ok)
	}
	switch {
	case kind == WKDialogAlert:
		callBlock(handler)
	case kind == WKDialogConfirm && ok:
		callBlock(handler, 1)
	case kind == WKDialogPrompt && ok:
		callBlock(handler, nsString(text))
	default:
		callBlock(handler, 0)
	}
}

// wkPageDialog answers the handler exactly once: at once without a view, window or with the
// Dialog hook; otherwise when the sheet ends or the view is disposed.
func wkPageDialog(v *WKView, kind int, frame uintptr, message, def string, handler uintptr) {
	if v == nil || v.disposed {
		answerDialog(kind, handler, false, "")
		return
	}
	if v.Dialog != nil {
		ok, text := false, ""
		func() {
			defer func() { _ = recover() }()
			ok, text = v.Dialog(kind, message, def)
		}()
		answerDialog(kind, handler, ok, text)
		return
	}
	win := msg(v.View, "window")
	if win == 0 {
		answerDialog(kind, handler, false, "")
		return
	}
	title := "This page says"
	if host := goString(msg(msg(msg(frame, "request"), "URL"), "host")); host != "" {
		title = "The page at " + host + " says:"
	}
	d := &wkDialog{kind: kind, handler: blockCopy(handler)}
	d.alert = msg(msg(class("NSAlert"), "alloc"), "init")
	msg(d.alert, "setMessageText:", nsString(title))
	msg(d.alert, "setInformativeText:", nsString(message))
	msg(d.alert, "addButtonWithTitle:", nsString("OK"))
	if kind != WKDialogAlert {
		msg(d.alert, "addButtonWithTitle:", nsString("Cancel"))
	}
	if kind == WKDialogPrompt {
		d.field = msg(msg(class("NSTextField"), "alloc"), "init")
		msgRectOnly(d.field, sel("setFrame:"), NSRect{Width: 300, Height: 24})
		msg(d.field, "setStringValue:", nsString(def))
		msg(d.alert, "setAccessoryView:", d.field)
		msg(d.alert, "layout")
		msg(msg(d.alert, "window"), "setInitialFirstResponder:", d.field)
	}
	if v.dialogs == nil {
		v.dialogs = map[*wkDialog]struct{}{}
	}
	v.dialogs[d] = struct{}{}
	msg(d.alert, "beginSheetModalForWindow:completionHandler:", win, NewBlock(1, func(a []uintptr) {
		v.endDialog(d, a[0] == nsAlertFirstButtonReturn)
	}))
}

// endDialog answers d once; the sheet's own completion after Dispose finds it done and does nothing.
func (v *WKView) endDialog(d *wkDialog, ok bool) {
	if d.done {
		return
	}
	d.done = true
	delete(v.dialogs, d)
	text := ""
	if d.field != 0 {
		text = goString(msg(d.field, "stringValue"))
	}
	answerDialog(d.kind, d.handler, ok, text)
	blockRelease(d.handler)
	if d.field != 0 {
		msg(d.field, "release")
	}
	msg(d.alert, "autorelease")
}

// endDialogs cancels the sheets still up, so no WebKit handler is dropped with the view.
func (v *WKView) endDialogs() {
	for d := range v.dialogs {
		if sheet := msg(d.alert, "window"); sheet != 0 {
			if parent := msg(sheet, "sheetParent"); parent != 0 {
				msg(parent, "endSheet:returnCode:", sheet, 0)
			}
		}
		v.endDialog(d, false)
	}
}
