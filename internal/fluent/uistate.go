//go:build windows

package fluent

import (
	"unsafe"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
)

const (
	wmChangeUIState = 0x127
	uisSet          = 1
	uisClear        = 2
	uisfHideFocus   = 1
)

var uiHooked = map[*swt.Display]bool{}

// Shell hides the focus rectangles of a window until the keyboard is used, as Windows 11 apps do: a
// click on a Link, Button, check box, Table or Tree leaves no dotted rectangle. The rule is one display
// filter, not a change of each widget: a mouse press sets UISF_HIDEFOCUS on the window of the control,
// Tab, an arrow key or Space clears it.
func Shell(s *swt.Shell) {
	setHideFocus(uintptr(s.Handle), true)
	// The window takes its UI state from the last input again when it is shown.
	var once bool
	s.AddListener(swt.Show, newListener(func(*swt.Event) {
		if !once {
			once = true
			setHideFocus(uintptr(s.Handle), true)
		}
	}))
	d := s.GetDisplay()
	if uiHooked[d] {
		return
	}
	uiHooked[d] = true
	d.AddFilter(swt.MouseDown, newListener(func(e *swt.Event) {
		if c := d.GetCursorControl(); c != nil {
			setHideFocus(uintptr(c.GetShell().Handle), true)
		}
	}))
	d.AddFilter(swt.MouseUp, newListener(func(e *swt.Event) {
		if c := d.GetCursorControl(); c != nil && c.GetName() == "Link" {
			// SysLink draws its focused item whatever the UI state says; take the focus off the item.
			d.AsyncExec(jrt.NewRunnable(func() {
				if !c.IsDisposed() {
					clearLinkFocus(uintptr(c.Handle))
				}
			}))
		}
	}))
	d.AddFilter(swt.KeyDown, newListener(func(e *swt.Event) {
		switch e.KeyCode {
		case swt.ARROW_UP, swt.ARROW_DOWN, swt.ARROW_LEFT, swt.ARROW_RIGHT:
		default:
			if e.Character != '\t' && e.Character != ' ' {
				return
			}
		}
		if c := d.GetFocusControl(); c != nil {
			setHideFocus(uintptr(c.GetShell().Handle), false)
		}
	}))
}

func setHideFocus(hwnd uintptr, hide bool) {
	op := uintptr(uisClear)
	if hide {
		op = uisSet
	}
	call(pSendMessage, hwnd, wmChangeUIState, op|uisfHideFocus<<16, 0)
}

// clearLinkFocus clears LIS_FOCUSED of the items of a SysLink (LM_SETITEM) and repaints it.
func clearLinkFocus(hwnd uintptr) {
	const lmSetItem, lifItemIndex, lifState, lisFocused = 0x400 + 0x302, 1, 2, 1
	var item [4400]byte // LITEM: mask, iLink, state, stateMask, szID[48], szUrl[2084]
	for i := int32(0); i < 16; i++ {
		*(*uint32)(unsafe.Pointer(&item[0])) = lifItemIndex | lifState
		*(*int32)(unsafe.Pointer(&item[4])) = i
		*(*uint32)(unsafe.Pointer(&item[8])) = 0
		*(*uint32)(unsafe.Pointer(&item[12])) = lisFocused
		if call(pSendMessage, hwnd, lmSetItem, 0, uintptr(unsafe.Pointer(&item[0]))) == 0 {
			break
		}
	}
	call(pInvalidateRect, hwnd, 0, 1)
}
