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
			app.Quit()
			return int64(cocoa.OSNSTerminateNow)
		}, 3)
		cocoa.OSMethod_setImplementation(m, terminateCb.GetAddress())
	})
}
