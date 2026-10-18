package swt

import "github.com/haiodo/gowt/internal/cocoa"

// remeasureLabel makes a shown NSTabView measure the tab again: SWT's label is always empty, so
// setting it once more does not, and only a changed label does (tooling/port.sh adds the calls).
func (this *TabItem) remeasureLabel() {
	this.nsItem.SetLabel(cocoa.NSStringStringWith(" "))
	this.nsItem.SetLabel(cocoa.NSStringString())
}
