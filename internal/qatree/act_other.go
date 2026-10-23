//go:build !darwin

package qatree

import "github.com/haiodo/gowt/swt"

// clickButton sets the state a click would and fires Selection, as the controlexample -snap
// driver does.
func clickButton(b *swt.Button) {
	switch st := b.GetStyle(); {
	case st&swt.RADIO != 0:
		// A programmatic selection does not clear the others of the group.
		if p := b.GetParent(); p != nil && p.GetStyle()&swt.NO_RADIO_GROUP == 0 {
			for _, k := range p.GetChildren() {
				if o, ok := k.Impl().(*swt.Button); ok && o != b && o.GetStyle()&swt.RADIO != 0 {
					o.SetSelection(false)
				}
			}
		}
		b.SetSelection(true)
	case st&(swt.CHECK|swt.TOGGLE) != 0:
		b.SetSelection(!b.GetSelection())
	}
	b.NotifyListeners(swt.Selection, swt.NewEvent())
}

// selectTab fires the listeners of the tab folder like a click on the tab.
func selectTab(f *swt.TabFolder, i int) { f.SetSelectionIndexNotify(int32(i), true) }
