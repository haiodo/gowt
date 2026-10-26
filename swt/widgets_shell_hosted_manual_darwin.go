// Shell.SetHosted: a Shell whose window is never shown but whose body is on screen elsewhere (the
// NSPopover of gowt.Popover) must report itself visible, or SWT and the QA walker skip its controls.

package swt

// A Shell that was never shown has a hidden view, which draws nothing, children included.
func (this *Shell) SetHosted(on bool) {
	if on {
		this.state &^= WidgetHIDDEN
	} else {
		this.state |= WidgetHIDDEN
	}
	this.View.SetHidden(!on)
}
