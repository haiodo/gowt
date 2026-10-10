package swttests

import (
	"github.com/haiodo/gowt/browser"
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

// Test_setJavascriptEnabled is SWT's test with the result of evaluate kept as an any: the Java one
// assigns a null Boolean, which a Go bool cannot hold.
func (this *Test_org_eclipse_swt_browser_Browser) Test_setJavascriptEnabled() {
	var pageLoadCount *jrt.AtomicInteger = jrt.NewAtomicInteger(0)
	var testFinished *jrt.AtomicBoolean = jrt.NewAtomicBoolean(false)
	var testPassed *jrt.AtomicBoolean = jrt.NewAtomicBoolean(false)
	this.browser.AddProgressListener(browser.ProgressListenerCompletedAdapter(func(event *browser.ProgressEvent) {
		switch pageLoadCount.IncrementAndGet() {
		case 1:
			this.browser.SetJavascriptEnabled(false)
			this.browser.SetText("Second page with javascript disabled")
		case 2:
			var expectedNull any
			func() {
				defer func() {
					if r := recover(); r != nil {
						junit.Fail("1) if javascript is disabled, browser.evaluate() should return null. But an Exception was thrown")
					}
				}()
				expectedNull = this.browser.Evaluate("return true")
			}()
			junit.AssertNull(expectedNull)
			testPassed.Set(true)
			testFinished.Set(true)
		}
	}))
	this.shell.Open()
	this.browser.SetText("First page with javascript enabled. This should not be visible as a second page should load")
	this.WaitForPassCondition(testFinished.Get)
	junit.AssertTrue(testPassed.Get())
}
