package swt

// hiddenNatively: SWT has c visible but its view is hidden. gowt's ScrollView hides the rows far off
// screen so; SWT itself hides a control's view only together with its state.
func hiddenNatively(c *Control) bool {
	if _, shell := isControlToShell(c); shell {
		return false
	}
	return (c.state&WidgetHIDDEN) == 0 && c.impl.topView_().IsHidden()
}

func showNatively(c *Control) {
	c.impl.topView_().SetHidden(false)
	c.impl.invalidateVisibleRegion_()
}

// revealForFocus shows the natively hidden views up from c before c takes the focus: AppKit gives no
// key view status to a view in a hidden subtree, so Tab and setFocus skipped such a row.
func revealForFocus(c *Control) {
	for ; c != nil; c = upcastCompositeToControl(c.parent) {
		if _, shell := isControlToShell(c); shell {
			return
		}
		if hiddenNatively(c) {
			showNatively(c)
		}
	}
}

// revealForPrint shows the natively hidden views inside c for a print and returns the call that hides
// them again.
func revealForPrint(c *Control) (restore func()) {
	var shown []*Control
	var walk func(c *Control)
	walk = func(c *Control) {
		if hiddenNatively(c) {
			showNatively(c)
			shown = append(shown, c)
		}
		if comp, ok := isControlToComposite(c); ok {
			for _, k := range comp._getChildren() {
				if _, shell := isControlToShell(k); !shell {
					walk(k)
				}
			}
		}
	}
	walk(c)
	return func() {
		for _, k := range shown {
			if !k.IsDisposed() {
				k.impl.topView_().SetHidden(true)
				k.impl.invalidateVisibleRegion_()
			}
		}
	}
}
