// Package jrt hand-writes small Go stand-ins for the java.lang/java.util base types that some
// translated SWT classes extend (SWTException/SWTError extend RuntimeException/Error, TypedEvent
// extends EventObject) but that have no Java source for j2go to translate - see
// tooling/j2go/README.md "Manual superclass embedding". Registered via the same
// Manual.isManual/manual.txt mechanism as Monitor/RoundingMode.
package jrt

import (
	"fmt"
	"os"
)

// RuntimeException and JavaError carry their own Message/Cause instead of sharing an embedded
// Throwable - see README "Manual superclass embedding" for why, and the ceiling this leaves.
type RuntimeException struct {
	Message string
	Cause   error
}

func NewRuntimeException(message string) RuntimeException {
	return RuntimeException{Message: message}
}

func (e *RuntimeException) Error() string      { return e.Message }
func (e *RuntimeException) GetMessage() string { return e.Message }
func (e *RuntimeException) Unwrap() error      { return e.Cause }
func (e *RuntimeException) PrintStackTrace()   { fmt.Fprintln(os.Stderr, e.Message) }

// Named JavaError, not Error: an embedded field literally named "Error" would shadow its own
// promoted Error() method, so *SWTError would silently stop satisfying Go's error interface.
type JavaError struct {
	Message string
	Cause   error
}

func NewJavaError(message string) JavaError { return JavaError{Message: message} }

func (e *JavaError) Error() string      { return e.Message }
func (e *JavaError) GetMessage() string { return e.Message }
func (e *JavaError) Unwrap() error      { return e.Cause }
func (e *JavaError) PrintStackTrace()   { fmt.Fprintln(os.Stderr, e.Message) }

type IllegalArgumentException struct {
	Message string
}

func NewIllegalArgumentException(message string) IllegalArgumentException {
	return IllegalArgumentException{Message: message}
}

func (e *IllegalArgumentException) Error() string { return e.Message }

// EventObject mirrors java.util.EventObject's one field: the object that fired the event.
type EventObject struct {
	Source any
}

func NewEventObject(source any) EventObject { return EventObject{Source: source} }

func (e *EventObject) GetSource() any { return e.Source }
