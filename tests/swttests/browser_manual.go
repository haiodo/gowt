package swttests

import "github.com/haiodo/gowt/internal/jrt"

// Hand-written parts of Test_org_eclipse_swt_browser_Browser (manual.txt).

// The diagnostics print JVM state or list /proc/self/fd; nothing to report here.
func Test_org_eclipse_swt_browser_BrowserPrintSystemEnv()                 {}
func Test_org_eclipse_swt_browser_BrowserPrintMemoryUse()                 {}
func Test_org_eclipse_swt_browser_BrowserPrintThreadsInfo()               {}
func Test_org_eclipse_swt_browser_BrowserGetOpenedDescriptors() *jrt.List { return jrt.NewList() }
func Test_org_eclipse_swt_browser_BrowserGetPropertiesSafe() *jrt.List    { return jrt.NewList() }
