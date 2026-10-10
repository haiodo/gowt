// Hand-written opaque stubs for Accessible and WidgetSpy (manual.txt): no input method and no
// accessibility tree are built. Methods are added as the translated gtk widgets call them.
package swt

// Accessible: what Control.java calls on a control that has no screen-reader Accessible attached.
type Accessible struct{ control *Control }

func AccessibleInternal_new_Accessible(control *Control) *Accessible {
	return &Accessible{control: control}
}

func (a *Accessible) GetControl() *Control { return a.control }

// The listeners are not stored: no accessibility tree is built (custom widgets register them at creation).
func (a *Accessible) AddAccessibleListener(l AccessibleListener)                            {}
func (a *Accessible) AddAccessibleControlListener(l AccessibleControlListener)              {}
func (a *Accessible) AddAccessibleTextListener(l AccessibleTextListener)                    {}
func (a *Accessible) RemoveAccessibleListener(l AccessibleListener)                         {}
func (a *Accessible) RemoveAccessibleControlListener(l AccessibleControlListener)           {}
func (a *Accessible) RemoveAccessibleTextListener(l AccessibleTextListener)                 {}
func (a *Accessible) AddAccessibleEditableTextListener(l AccessibleEditableTextListener)    {}
func (a *Accessible) RemoveAccessibleEditableTextListener(l AccessibleEditableTextListener) {}
func (a *Accessible) AddAccessibleAttributeListener(l AccessibleAttributeListener)          {}
func (a *Accessible) RemoveAccessibleAttributeListener(l AccessibleAttributeListener)       {}
func (a *Accessible) SetFocus(childID int32)                                                {}

// StyledText's text notifications go nowhere without an accessibility tree.
func (a *Accessible) TextChanged(typ int32, startIndex int32, length int32) {}
func (a *Accessible) TextCaretMoved(index int32)                            {}
func (a *Accessible) TextSelectionChanged()                                 {}

// Relations and lifetime: nothing to maintain without an accessibility tree.
func (a *Accessible) AddRelation(relation int32, target *Accessible)    {}
func (a *Accessible) RemoveRelation(relation int32, target *Accessible) {}
func (a *Accessible) Internal_dispose_Accessible()                      {}

// WidgetSpy: creation/disposal tracking, off by default (as the real class).
var WidgetSpyIsEnabled bool

type widgetSpy struct{}

func (widgetSpy) WidgetCreated(w *Widget)  {}
func (widgetSpy) WidgetDisposed(w *Widget) {}

func WidgetSpyGetInstance() widgetSpy { return widgetSpy{} }
