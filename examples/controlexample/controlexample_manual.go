// Hand-written glue for the translated ControlExample (tooling/j2go/manual.txt, README
// "Round 10 controlexample").
package controlexample

import (
	// Registers the resource FS; an imported package initializes before this package's vars.
	_ "github.com/haiodo/gowt/examples/controlexample/res"
	"github.com/haiodo/gowt/swt"
	// The Set/Get API dialog looks widget methods up by Java name (Tab.java's getMethod).
	_ "github.com/haiodo/gowt/swt/swtreflect"
)

// CreateTabs replaces ControlExample.createTabs(): only the tabs translated so far.
func (this *ControlExample) CreateTabs() []*Tab {
	this.shellTab = newShellTab(this)
	return []*Tab{
		&newButtonTab(this).Tab,
		&newCanvasTab(this).Tab,
		&newComboTab(this).Tab,
		&newCoolBarTab(this).Tab,
		&newDateTimeTab(this).Tab,
		&newDialogTab(this).Tab,
		&newExpandBarTab(this).Tab,
		&newGroupTab(this).Tab,
		&newLabelTab(this).Tab,
		&newLinkTab(this).Tab,
		&newListTab(this).Tab,
		&newMenuTab(this).Tab,
		&newProgressBarTab(this).Tab,
		&newSashTab(this).Tab,
		&newScaleTab(this).Tab,
		&this.shellTab.Tab,
		&newSliderTab(this).Tab,
		&newSpinnerTab(this).Tab,
		&newSystemTab(this).Tab,
		&newTabFolderTab(this).Tab,
		&newTableTab(this).Tab,
		&newTextTab(this).Tab,
		&newToolBarTab(this).Tab,
		&newToolTipTab(this).Tab,
		&newTreeTab(this).Tab,
	}
}

// TabFolder exposes the example's tab folder to a driver (cmd/controlexample -snap).
func (this *ControlExample) TabFolder() *swt.TabFolder { return this.tabFolder }
