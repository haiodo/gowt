package swt

import "github.com/haiodo/gowt/internal/jrt"

// SyncCall is Display.syncCall<T, E>: generic, so hand-written. A panic in the callable (Java's
// thrown exception) is re-raised in the calling goroutine.
func (this *Display) SyncCall(callable func() any) any {
	var result, raised any
	this.SyncExec(jrt.NewRunnable(func() {
		defer func() { raised = recover() }()
		result = callable()
	}))
	if raised != nil {
		panic(raised)
	}
	return result
}
