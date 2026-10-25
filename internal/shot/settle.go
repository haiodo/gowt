package shot

import "sync"

// GOWT_SNAP waits for Settled: a web view that was created must have finished its last navigation.
var (
	settleMu sync.Mutex
	created  bool
	pending  bool
	finished bool
)

// WebViewCreated, NavStarted and NavDone are called by package webview.
func WebViewCreated() { settleMu.Lock(); created = true; settleMu.Unlock() }
func NavStarted()     { settleMu.Lock(); pending = true; settleMu.Unlock() }
func NavDone()        { settleMu.Lock(); pending, finished = false, true; settleMu.Unlock() }

// Settled reports whether a snapshot can be taken: no web view, or one whose last load has completed or failed.
func Settled() bool {
	settleMu.Lock()
	defer settleMu.Unlock()
	return !created || (finished && !pending)
}

// HasWebView reports whether a web view exists; its first paint comes a moment after the load finished.
func HasWebView() bool {
	settleMu.Lock()
	defer settleMu.Unlock()
	return created
}
