package jrt

import "sync"

// The org.eclipse.core.runtime subset JFace needs (no OSGi: no Bundle, FrameworkUtil or Platform).
// Hand-written over Java stubs in tooling/j2go/stubs, mapped in j2go's Manual.

// AssertionFailedException is what Assert.isTrue/isNotNull throw.
type AssertionFailedException struct{ RuntimeException }

func newAssertionFailed(msg string) *AssertionFailedException {
	return &AssertionFailedException{RuntimeException{Message: "assertion failed: " + msg}}
}

func msgOf(msg []string) string {
	if len(msg) > 0 {
		return msg[0]
	}
	return ""
}

// Assert mirrors org.eclipse.core.runtime.Assert; msg is Java's optional second argument.
type Assert struct{}

func AssertIsTrue(expr bool, msg ...string) bool {
	if !expr {
		panic(newAssertionFailed(msgOf(msg)))
	}
	return true
}

func AssertIsLegal(expr bool, msg ...string) bool {
	if !expr {
		panic(NewIllegalArgumentException(msgOf(msg)))
	}
	return true
}

func AssertIsNotNull(object any, msg ...string) {
	if IsNil(object) {
		panic(newAssertionFailed("null argument:" + msgOf(msg)))
	}
}

const (
	StatusOK, StatusInfo, StatusWarning, StatusError, StatusCancel = 0, 1, 2, 4, 8
)

// IStatus is org.eclipse.core.runtime.IStatus; Status below implements it.
type IStatus interface {
	GetSeverity() int32
	GetMessage() string
	GetPlugin() string
	GetCode() int32
	GetException() error
	IsOK() bool
}

type Status struct {
	Severity  int32
	Plugin    string
	Code      int32
	Message   string
	Exception error
}

func NewStatus(severity int32, plugin string, message string, exception ...error) *Status {
	s := &Status{Severity: severity, Plugin: plugin, Message: message}
	if len(exception) > 0 {
		s.Exception = exception[0]
	}
	return s
}

func (s *Status) GetSeverity() int32  { return s.Severity }
func (s *Status) GetMessage() string  { return s.Message }
func (s *Status) GetPlugin() string   { return s.Plugin }
func (s *Status) GetCode() int32      { return s.Code }
func (s *Status) GetException() error { return s.Exception }
func (s *Status) IsOK() bool          { return s.Severity == StatusOK }

// CoreException wraps an IStatus like org.eclipse.core.runtime.CoreException.
type CoreException struct {
	RuntimeException
	Status IStatus
}

func NewCoreException(status IStatus) *CoreException {
	return &CoreException{RuntimeException{Message: status.GetMessage(), Cause: status.GetException()}, status}
}

func (e *CoreException) GetStatus() IStatus { return e.Status }

type IProgressMonitor interface {
	BeginTask(name string, totalWork int32)
	Done()
	InternalWorked(work float64)
	IsCanceled() bool
	SetCanceled(value bool)
	SetTaskName(name string)
	SubTask(name string)
	Worked(work int32)
}

const UNKNOWN = -1

// NullProgressMonitor ignores progress but remembers cancellation.
type NullProgressMonitor struct{ canceled bool }

func NewNullProgressMonitor() *NullProgressMonitor { return &NullProgressMonitor{} }

func (m *NullProgressMonitor) BeginTask(string, int32) {}
func (m *NullProgressMonitor) Done()                   {}
func (m *NullProgressMonitor) InternalWorked(float64)  {}
func (m *NullProgressMonitor) IsCanceled() bool        { return m.canceled }
func (m *NullProgressMonitor) SetCanceled(v bool)      { m.canceled = v }
func (m *NullProgressMonitor) SetTaskName(string)      {}
func (m *NullProgressMonitor) SubTask(string)          {}
func (m *NullProgressMonitor) Worked(int32)            {}

// ProgressMonitorWrapper forwards every call to the wrapped monitor.
type ProgressMonitorWrapper struct{ IProgressMonitor }

func NewProgressMonitorWrapper(m IProgressMonitor) *ProgressMonitorWrapper {
	return &ProgressMonitorWrapper{m}
}

func (w *ProgressMonitorWrapper) GetWrappedProgressMonitor() IProgressMonitor {
	return w.IProgressMonitor
}

// ISafeRunnable is org.eclipse.core.runtime.ISafeRunnable.
type ISafeRunnable interface {
	HandleException(exception error)
	Run()
}

// SafeRunnerRun runs r, routing a panic that is an error to HandleException (SafeRunner.run).
func SafeRunnerRun(r ISafeRunnable) {
	defer func() {
		if e := recover(); e != nil {
			err, ok := e.(error)
			if !ok {
				panic(e)
			}
			r.HandleException(err)
		}
	}()
	r.Run()
}

// ListenerList is org.eclipse.core.runtime.ListenerList: identity-compared, copy-on-write so
// GetListeners can be iterated while a listener removes itself.
type ListenerList struct {
	mu        sync.Mutex
	listeners []any
}

func NewListenerList() *ListenerList { return &ListenerList{} }

func (l *ListenerList) Add(x any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.listeners {
		if e == x {
			return
		}
	}
	l.listeners = append(append([]any(nil), l.listeners...), x)
}

func (l *ListenerList) Remove(x any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, e := range l.listeners {
		if e == x {
			l.listeners = append(append([]any(nil), l.listeners[:i]...), l.listeners[i+1:]...)
			return
		}
	}
}

func (l *ListenerList) GetListeners() []any {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.listeners
}

func (l *ListenerList) Size() int32   { return int32(len(l.GetListeners())) }
func (l *ListenerList) IsEmpty() bool { return l.Size() == 0 }
func (l *ListenerList) Clear() {
	l.mu.Lock()
	l.listeners = nil
	l.mu.Unlock()
}
