package swttests

import (
	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/junit"
)

// Hand-written parts of Test_org_eclipse_swt_browser_Browser (manual.txt).

// The diagnostics print JVM state or list /proc/self/fd; nothing to report here.
func Test_org_eclipse_swt_browser_BrowserPrintSystemEnv()                 {}
func Test_org_eclipse_swt_browser_BrowserPrintMemoryUse()                 {}
func Test_org_eclipse_swt_browser_BrowserPrintThreadsInfo()               {}
func Test_org_eclipse_swt_browser_BrowserGetOpenedDescriptors() *jrt.List { return jrt.NewList() }
func Test_org_eclipse_swt_browser_BrowserGetPropertiesSafe() *jrt.List    { return jrt.NewList() }

// The translated test assigns a null Boolean, which Go's bool cannot hold, and package webview has
// no per-page JavaScript switch for setJavascriptEnabled to drive.
func (this *Test_org_eclipse_swt_browser_Browser) Test_setJavascriptEnabled() {
	junit.AssumeTrue(false, "setJavascriptEnabled is not supported over package webview")
}
