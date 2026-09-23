// Hand-written stub for org.eclipse.swt.internal.ExceptionStash (manual.txt): insulates
// EventTable from panics raised inside a user Listener. Simplified from the real class - it does
// not forward to Display's global exception handler first (Display isn't translated yet, see
// README "Known gaps"); it only stores the first panic and re-raises it on Close.
package swt

type ExceptionStash struct {
	stored error
}

func NewExceptionStash() *ExceptionStash {
	return &ExceptionStash{}
}

func (s *ExceptionStash) Stash(err error) {
	if s.stored == nil {
		s.stored = err
	}
}

func (s *ExceptionStash) Close() {
	if s.stored != nil {
		panic(s.stored)
	}
}
