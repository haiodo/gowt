// Hand-written stub for org.eclipse.swt.internal.ExceptionStash (manual.txt): insulates
// EventTable from panics raised inside a user Listener. Like the real class it offers each
// exception to the current Display's handler first; it keeps the first one that handler rethrows
// (no addSuppressed) and re-raises it on Close.
package swt

import (
	"fmt"

	"github.com/haiodo/gowt/internal/jrt"
)

type ExceptionStash struct {
	stored error
}

func NewExceptionStash() *ExceptionStash {
	return &ExceptionStash{}
}

func (s *ExceptionStash) Stash(err error) {
	if d := DisplayGetCurrent(); d != nil {
		handler := d.GetRuntimeExceptionHandler()
		switch err.(type) {
		case *jrt.JavaError, *SWTError:
			handler = d.GetErrorHandler()
		}
		if thrown := callHandler(handler, err); thrown == nil {
			return
		} else {
			err = thrown
		}
	}
	if s.stored == nil {
		s.stored = err
	}
}

// callHandler returns what the handler panicked with, nil if it handled err.
func callHandler(handler func(error), err error) (thrown error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				thrown = e
			} else {
				thrown = &jrt.RuntimeException{Message: "panic: " + fmt.Sprint(r)}
			}
		}
	}()
	handler(err)
	return nil
}

func (s *ExceptionStash) Close() {
	if s.stored != nil {
		panic(s.stored)
	}
}
