package gowt

import (
	"sync"
	"sync/atomic"

	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/swt"
)

var (
	terminateApp  atomic.Pointer[App]
	terminateOnce sync.Once
	terminateCb   *swt.Callback
	willEndCb     *swt.Callback
	willEndAdded  bool
)

// hookTerminate replaces SWT's applicationShouldTerminate:, which disposes the display inside
// ReadAndDispatch and crashes the pool pop. An allowed quit ends the loop; terminate: gets
// NSTerminateNow.
func (a *App) hookTerminate() {
	terminateApp.Store(a)
	terminateOnce.Do(func() {
		cls := cocoa.OSObjc_lookUpClass("SWTApplicationDelegate")
		m := cocoa.OSClass_getInstanceMethod(cls, cocoa.OSSel_applicationShouldTerminate_)
		if cls == 0 || m == 0 {
			return
		}
		terminateCb = swt.NewCallbackFn(func([]int64) int64 {
			app := terminateApp.Load()
			if app == nil || app.display.IsDisposed() {
				return int64(cocoa.OSNSTerminateCancel)
			}
			e := swt.NewEvent()
			app.display.SendEvent(swt.Close, e)
			if !e.Doit {
				return int64(cocoa.OSNSTerminateCancel)
			}
			app.approved = true // AppKit now closes the windows: their OnClose must not ask or open a modal
			app.Quit()
			return int64(cocoa.OSNSTerminateNow)
		}, 3)
		cocoa.OSMethod_setImplementation(m, terminateCb.GetAddress())
		// terminate: exits the process after NSTerminateNow, so Run's deferred dispose never runs: dispose here.
		willEndCb = swt.NewCallbackFn(func([]int64) int64 {
			if app := terminateApp.Load(); app != nil && !app.display.IsDisposed() {
				app.display.Dispose()
			}
			return 0
		}, 3)
		willEndAdded = cocoa.OSClass_addMethod(cls, cocoa.OSSel_registerName("applicationWillTerminate:"), willEndCb.GetAddress(), "v@:@")
		// NSApplication reads which delegate methods exist when the delegate is set, so set it again.
		app := cocoa.NSApplicationSharedApplication()
		d := cocoa.OSObjc_msgSend(app.Id, cocoa.OSSel_registerName("delegate"))
		cocoa.OSObjc_msgSendOverload44(app.Id, cocoa.OSSel_registerName("setDelegate:"), d)
	})
}
