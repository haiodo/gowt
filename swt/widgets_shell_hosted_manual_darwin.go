// Shell.SetHosted: a Shell whose window is never shown but whose body is on screen elsewhere (the
// NSPopover of gowt.Popover) must report itself visible, or SWT and the QA walker skip its controls.

package swt

func (this *Shell) SetHosted(on bool) {
	if on {
		this.state &^= WidgetHIDDEN
	} else {
		this.state |= WidgetHIDDEN
	}
}
