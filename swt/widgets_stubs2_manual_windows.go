// Accessible for win32: MSAA/COM accessibility is not ported; a Control with no screen reader attached
// gets no answer to WM_GETOBJECT (0 means "not handled").
package swt

type Accessible struct{}

func AccessibleInternal_new_Accessible(control *Control) *Accessible { return &Accessible{} }

func (a *Accessible) Internal_dispose_Accessible() {}

func (a *Accessible) Internal_WM_GETOBJECT(wParam int64, lParam int64) int64 { return 0 }

// The listeners are not stored: no accessibility tree is built (custom widgets register them at creation).
func (a *Accessible) AddAccessibleListener(l AccessibleListener)               {}
func (a *Accessible) AddAccessibleControlListener(l AccessibleControlListener) {}
func (a *Accessible) AddAccessibleTextListener(l AccessibleTextListener)       {}
func (a *Accessible) SetFocus(childID int32)                                   {}
