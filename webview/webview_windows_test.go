package webview

import "testing"

// A transient render (kind 1) or frame render (kind 3) process exit must not kill the engine:
// WebView2 recovers with the next navigation, so only the loads in flight abort. A sticky e.err
// here would block every later Navigate/Eval after one recovered crash.
func TestProcessFailedRecovery(t *testing.T) {
	aborted := 0
	e := &wv2Engine{w: &WebView{}, isReady: true, lastURL: "app://host/index.html"}
	e.pending = []op{{run: func() {}, abort: func(error) { aborted++ }}}
	e.loads, e.cancelled = 2, true // two API loads in flight, the last one cancelled
	var failedURL string
	e.w.OnNavigationFailed = func(u string, _ error) { failedURL = u }
	e.processFailed(1)
	if e.err != nil {
		t.Fatalf("kind 1 left a permanent error: %v", e.err)
	}
	if aborted != 1 {
		t.Errorf("loads in flight aborted %d times, want 1", aborted)
	}
	if e.loads != 0 || e.cancelled {
		t.Errorf("in-flight navigation state loads=%d cancelled=%v, want reset", e.loads, e.cancelled)
	}
	if failedURL != "app://host/index.html" {
		t.Errorf("navigation failure reported for %q, want the loading URL", failedURL)
	}
	ran := false
	e.do(func() { ran = true }, func(error) { t.Error("a recovered engine aborted a new op") })
	if !ran {
		t.Error("an op after a recovered crash did not run")
	}
	e.processFailed(3)
	if e.err != nil {
		t.Fatalf("kind 3 left a permanent error: %v", e.err)
	}

	e.processFailed(6) // GPU process: the page stays alive, not a failure
	if e.err != nil {
		t.Fatalf("kind 6 left an error: %v", e.err)
	}

	browser := &wv2Engine{w: &WebView{}}
	browser.processFailed(0) // the browser process is gone: fatal, like a broken runtime
	if browser.err == nil {
		t.Error("browser process exit must kill the engine")
	}
}
