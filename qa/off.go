//go:build !gowtqa

package qa

// Handle registers fn under name for qatest's State; fn runs on the UI thread. The name "reset" is
// the one qatest calls before each test. Without -tags gowtqa it does nothing.
func Handle(name string, fn func(args Args) error) {}

// HandleValue is Handle for a handler that returns a value to the test, JSON encoded.
func HandleValue(name string, fn func(args Args) (any, error)) {}
