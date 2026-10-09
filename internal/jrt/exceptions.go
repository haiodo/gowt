package jrt

// The JDK exceptions the translated sources construct or catch. Each is a RuntimeException by
// shape (checked or not in Java): a catch of a concrete one matches by Go type.

func exceptionOf(args []any) RuntimeException {
	e := RuntimeException{}
	for _, a := range args {
		switch v := a.(type) {
		case string:
			e.Message = v
		case error:
			e.Cause = v
			if e.Message == "" {
				e.Message = v.Error()
			}
		}
	}
	return e
}

type InterruptedException struct{ RuntimeException }

func NewInterruptedException(args ...any) *InterruptedException {
	return &InterruptedException{exceptionOf(args)}
}

type TimeoutException struct{ RuntimeException }

func NewTimeoutException(args ...any) *TimeoutException {
	return &TimeoutException{exceptionOf(args)}
}

type RejectedExecutionException struct{ RuntimeException }

func NewRejectedExecutionException(args ...any) *RejectedExecutionException {
	return &RejectedExecutionException{exceptionOf(args)}
}

type IndexOutOfBoundsException struct{ RuntimeException }

func NewIndexOutOfBoundsException(args ...any) *IndexOutOfBoundsException {
	return &IndexOutOfBoundsException{exceptionOf(args)}
}

type NullPointerException struct{ RuntimeException }

func NewNullPointerException(args ...any) *NullPointerException {
	return &NullPointerException{exceptionOf(args)}
}

// URISyntaxException is new URISyntaxException(input, reason).
type URISyntaxException struct{ RuntimeException }

func NewURISyntaxException(input, reason string) *URISyntaxException {
	return &URISyntaxException{RuntimeException{Message: reason + ": " + input}}
}
