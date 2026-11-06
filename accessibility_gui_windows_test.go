package gowt

import (
	"testing"
	"time"
	"unsafe"

	"github.com/haiodo/gowt/internal/win32"
)

// TestAccessibilityCommonControls queries MSAA through AccessibleObjectFromWindow - the same
// WM_GETOBJECT path a screen reader takes - for name, role, value, state and default action.
// Focus goes through UI Automation, the current Windows assistive-technology API.
func TestAccessibilityCommonControls(t *testing.T) {
	guiRun(t, func(app *App, w *Window) {
		pressed := false
		btn := put[*ButtonView](w, Button("Save", OnClick(func() { pressed = true })))
		icon := put[*ButtonView](w, Button("", AccessibleLabel("Attach")))
		check := put[*ButtonView](w, Button("Wrap", Check()))
		text := put[*TextView](w, Text(Value("hello")))
		scale := put[*ScaleView](w, Scale(0, 100, Value(30)))
		w.Unwrap().Layout()

		acc := msaa(t, btn.Unwrap().Handle)
		if acc == nil {
			return
		}
		if got := msaaRole(acc); got != win32.COMROLE_SYSTEM_PUSHBUTTON {
			t.Errorf("button role %#x, want ROLE_SYSTEM_PUSHBUTTON", got)
		}
		if got := msaaName(acc); got != "Save" {
			t.Errorf("button name %q, want Save", got)
		}
		if got := msaaDefaultAction(acc); got == "" {
			t.Error("button has no default action")
		}

		iconAcc := msaa(t, icon.Unwrap().Handle)
		if iconAcc == nil {
			return
		}
		if got := msaaName(iconAcc); got != "Attach" {
			t.Errorf("icon button name %q, want Attach (AccessibleLabel)", got)
		}

		check.SetChecked(true)
		checkAcc := msaa(t, check.Unwrap().Handle)
		if checkAcc == nil {
			return
		}
		if got := msaaRole(checkAcc); got != win32.COMROLE_SYSTEM_CHECKBUTTON {
			t.Errorf("check role %#x, want ROLE_SYSTEM_CHECKBUTTON", got)
		}
		if msaaState(checkAcc)&win32.COMSTATE_SYSTEM_CHECKED == 0 {
			t.Error("checked state missing on checked check box")
		}

		textAcc := msaa(t, text.Unwrap().Handle)
		if textAcc == nil {
			return
		}
		if got := msaaRole(textAcc); got != win32.COMROLE_SYSTEM_TEXT {
			t.Errorf("text role %#x, want ROLE_SYSTEM_TEXT", got)
		}
		if got := msaaValue(textAcc); got != "hello" {
			t.Errorf("text value %q, want hello", got)
		}

		scaleAcc := msaa(t, scale.Unwrap().Handle)
		if scaleAcc == nil {
			return
		}
		if got := msaaRole(scaleAcc); got != win32.COMROLE_SYSTEM_SLIDER {
			t.Errorf("scale role %#x, want ROLE_SYSTEM_SLIDER", got)
		}
		if got := msaaValue(scaleAcc); got != "30" {
			t.Errorf("scale value %q, want 30", got)
		}

		app.Async(func() {
			w.Unwrap().ForceActive()
			if code := acc.AccDoDefaultAction(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF)))); code != 0 {
				t.Errorf("AccDoDefaultAction(button) = %#x", code)
			}
			if code := checkAcc.AccDoDefaultAction(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF)))); code != 0 {
				t.Errorf("AccDoDefaultAction(check) = %#x", code)
			}
			app.After(20*time.Millisecond, func() {
				if !pressed {
					t.Error("MSAA default action did not press the button")
				}
				if check.Checked() {
					t.Error("MSAA default action did not uncheck the check box")
				}
				w.Unwrap().ForceActive()
				d := text.Unwrap().GetDisplay()
				for range 10 {
					for d.ReadAndDispatch() {
					}
				}
				if !text.Unwrap().SetFocus() {
					t.Error("SetFocus failed on the text control")
				}
				app.After(20*time.Millisecond, func() {
					if focused, ok := win32.UIAHasKeyboardFocus(); !ok || !focused {
						t.Errorf("UIA HasKeyboardFocus = %v, %v, want true, true", focused, ok)
					}
				})
			})
		})
	})
}

func msaa(t *testing.T, hwnd int64) *win32.IAccessible {
	t.Helper()
	acc, hr := win32.AccessibleObjectFromWindow(hwnd, win32.OSOBJID_CLIENT)
	if acc == nil {
		t.Errorf("AccessibleObjectFromWindow(%x) = nil, HRESULT %#x", hwnd, hr)
		return nil
	}
	t.Cleanup(func() { acc.Release() })
	return acc
}

// msaaVariant is a VT_I4 VARIANT for a child id, the form MSAA takes.
func msaaVariant(child int32) unsafe.Pointer {
	b := make([]byte, 16)
	*(*int16)(unsafe.Pointer(&b[0])) = win32.COMVT_I4
	*(*int32)(unsafe.Pointer(&b[8])) = child
	return unsafe.Pointer(&b[0])
}

func msaaName(acc *win32.IAccessible) string {
	var bstr int64
	if code := acc.Get_accName(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF))), int64(uintptr(unsafe.Pointer(&bstr)))); code != 0 {
		return ""
	}
	return win32.AccBSTR(bstr)
}

func msaaDefaultAction(acc *win32.IAccessible) string {
	var bstr int64
	if code := acc.Get_accDefaultAction(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF))), int64(uintptr(unsafe.Pointer(&bstr)))); code != 0 {
		return ""
	}
	return win32.AccBSTR(bstr)
}

func msaaValue(acc *win32.IAccessible) string {
	var bstr int64
	if code := acc.Get_accValue(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF))), int64(uintptr(unsafe.Pointer(&bstr)))); code != 0 {
		return ""
	}
	return win32.AccBSTR(bstr)
}

func msaaRole(acc *win32.IAccessible) int32 {
	var out [2]uint64
	code := acc.Get_accRole(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF))), int64(uintptr(unsafe.Pointer(&out[0]))))
	if code != 0 || *(*int16)(unsafe.Pointer(&out[0])) != win32.COMVT_I4 {
		return -1
	}
	return *(*int32)(unsafe.Add(unsafe.Pointer(&out[0]), 8))
}

func msaaState(acc *win32.IAccessible) int32 {
	var out [2]uint64
	code := acc.Get_accState(int64(uintptr(msaaVariant(win32.COMCHILDID_SELF))), int64(uintptr(unsafe.Pointer(&out[0]))))
	if code != 0 || *(*int16)(unsafe.Pointer(&out[0])) != win32.COMVT_I4 {
		return -1
	}
	return *(*int32)(unsafe.Add(unsafe.Pointer(&out[0]), 8))
}
