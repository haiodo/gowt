package swt

// getNameText_ is ToolItem.getNameText: the win32 ToolItem.java translation emits no
// name accessor, so it lives here manually for the accessibility role name.
func (this *ToolItem) getNameText_() string {
	if this.text != "" {
		return this.text
	}
	return this.toolTipText
}
