//go:build gowtqa

package qa

import "sync"

var (
	regMu    sync.Mutex
	handlers = map[string]func(Args) (any, error){}
)

// Handle registers fn under name for qatest's State; fn runs on the UI thread. The name "reset" is
// the one qatest calls before each test. Without -tags gowtqa it does nothing.
func Handle(name string, fn func(args Args) error) {
	HandleValue(name, func(a Args) (any, error) { return nil, fn(a) })
}

// HandleValue is Handle for a handler that returns a value to the test, JSON encoded.
func HandleValue(name string, fn func(args Args) (any, error)) {
	regMu.Lock()
	defer regMu.Unlock()
	handlers[name] = fn
}

func handler(name string) func(Args) (any, error) {
	regMu.Lock()
	defer regMu.Unlock()
	return handlers[name]
}
