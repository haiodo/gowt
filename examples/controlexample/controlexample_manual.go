// Hand-written glue for the translated ControlExample (tooling/j2go/manual.txt, README
// "Round 10 controlexample").
package controlexample

import (
	"embed"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/swt"
	// The Set/Get API dialog looks widget methods up by Java name (Tab.java's getMethod).
	_ "github.com/haiodo/gowt/swt/swtreflect"
)

// Registered from a var initializer: those run before every init(), including the generated
// one that calls ResourceBundle.getBundle.
//
//go:embed *.png *.gif *.bmp examples_control.properties
var resources embed.FS

var _ = registerResources()

func registerResources() bool {
	jrt.RegisterResources(resources)
	return true
}

// CreateTabs replaces ControlExample.createTabs(): only the tabs translated so far.
func (this *ControlExample) CreateTabs() []*Tab {
	return []*Tab{
		&newButtonTab(this).Tab,
		&newCanvasTab(this).Tab,
		&newGroupTab(this).Tab,
		&newLabelTab(this).Tab,
		&newMenuTab(this).Tab,
		&newTextTab(this).Tab,
	}
}

// ShellTab is not translated yet; ControlExample only keeps a nil field of this type.
type ShellTab struct{}

func (t *ShellTab) CloseAllShells() {}

// TabFolder exposes the example's tab folder to a driver (cmd/controlexample -snap).
func (this *ControlExample) TabFolder() *swt.TabFolder { return this.tabFolder }
