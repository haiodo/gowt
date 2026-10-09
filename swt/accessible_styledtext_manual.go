package swt

// What StyledText calls on its Accessible: no accessibility tree is built, so listeners are not stored and
// notifications go nowhere (the per-OS Accessible stubs have the rest).
func (a *Accessible) AddAccessibleEditableTextListener(l AccessibleEditableTextListener) {}
func (a *Accessible) AddAccessibleAttributeListener(l AccessibleAttributeListener)       {}
func (a *Accessible) RemoveAccessibleListener(l AccessibleListener)                      {}
func (a *Accessible) RemoveAccessibleControlListener(l AccessibleControlListener)        {}
func (a *Accessible) RemoveAccessibleTextListener(l AccessibleTextListener)              {}
func (a *Accessible) RemoveAccessibleAttributeListener(l AccessibleAttributeListener)    {}
func (a *Accessible) RemoveAccessibleEditableTextListener(l AccessibleEditableTextListener) {
}
func (a *Accessible) TextChanged(typ int32, startIndex int32, length int32) {}
func (a *Accessible) TextCaretMoved(index int32)                            {}
func (a *Accessible) TextSelectionChanged()                                 {}
