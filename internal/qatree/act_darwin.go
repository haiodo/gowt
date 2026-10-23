package qatree

import (
	"github.com/ebitengine/purego/objc"
	"github.com/haiodo/gowt/swt"
)

// clickButton sends performClick:, the real NSButton target/action path, as the controlexample
// -snap driver does.
func clickButton(b *swt.Button) {
	objc.ID(b.View.Id).Send(objc.RegisterName("performClick:"), 0)
}

// selectTab sends selectTabViewItemAtIndex:, which runs TabFolder's own delegate path like a click
// on the tab.
func selectTab(f *swt.TabFolder, i int) {
	objc.ID(f.View.Id).Send(objc.RegisterName("selectTabViewItemAtIndex:"), i)
}
