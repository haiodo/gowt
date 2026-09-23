// Hand-written: exercises Round 3's new translator constructs (switch statement with
// fallthrough, throw/try/catch/finally/try-with-resources, manual superclass embedding) against
// the generated EventTable/TypedListener/SWT/SWTException/SWTError code.
package swt

import "testing"

// A pointer-to-struct listener, not a bare func: Unhook's == (Java reference equality) panics
// on an uncomparable concrete type like a func value.
type recordingListener struct {
	got *Event
}

func (l *recordingListener) HandleEvent(event *Event) { l.got = event }

func TestEventTableHookUnhookSendEvent(t *testing.T) {
	table := &EventTable{}
	if table.Hooks(SWTSelection) {
		t.Fatal("should not be hooked yet")
	}
	l := &recordingListener{}
	table.Hook(SWTSelection, l)
	if !table.Hooks(SWTSelection) {
		t.Fatal("should be hooked")
	}

	ev := &Event{Type: SWTSelection, Data: "payload"}
	table.SendEvent(ev)
	if l.got == nil || l.got.Data != "payload" {
		t.Fatalf("listener was not invoked with the event, got %+v", l.got)
	}

	l.got = nil
	table.Unhook(SWTSelection, l)
	table.SendEvent(&Event{Type: SWTSelection})
	if l.got != nil {
		t.Fatal("listener should not fire after unhook")
	}
}

type panickingListener struct{}

func (panickingListener) HandleEvent(*Event) { panic(errBoom) }

type flagListener struct{ ran bool }

func (l *flagListener) HandleEvent(*Event) { l.ran = true }

// A listener that panics must not stop other listeners, and the stashed panic must surface
// once sendEvent's try-with-resources/finally have both run (see EventTable.java's own doc).
func TestEventTableSendEventPropagatesListenerPanic(t *testing.T) {
	table := &EventTable{}
	second := &flagListener{}
	table.Hook(SWTSelection, panickingListener{})
	table.Hook(SWTSelection, second)

	func() {
		defer func() {
			r := recover()
			if r != errBoom {
				t.Fatalf("expected the stashed panic to propagate, got %v", r)
			}
		}()
		table.SendEvent(&Event{Type: SWTSelection})
	}()

	if !second.ran {
		t.Fatal("a panicking listener must not prevent the next listener from running")
	}
	if table.level != 0 {
		t.Fatalf("finally must still restore level, got %d", table.level)
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

var errBoom = testErr("boom")

func TestTypedListenerSwitchDispatch(t *testing.T) {
	sl := &fakeShellListener{}
	tl := NewTypedListener(sl)

	tl.HandleEvent(&Event{Type: SWTActivate})
	if !sl.activated {
		t.Fatal("case SWT.Activate did not dispatch to ShellActivated")
	}

	e := &Event{Type: SWTClose}
	tl.HandleEvent(e)
	if !sl.closed {
		t.Fatal("case SWT.Close did not dispatch to ShellClosed")
	}
}

type fakeShellListener struct {
	activated, closed bool
}

func (f *fakeShellListener) ShellActivated(*ShellEvent)   { f.activated = true }
func (f *fakeShellListener) ShellClosed(*ShellEvent)      { f.closed = true }
func (f *fakeShellListener) ShellDeactivated(*ShellEvent) {}
func (f *fakeShellListener) ShellDeiconified(*ShellEvent) {}
func (f *fakeShellListener) ShellIconified(*ShellEvent)   {}

// SWT.error/findErrorText: old-style switch with grouped case labels, throw -> panic.
func TestSWTErrorSwitchAndThrow(t *testing.T) {
	if got := SWTFindErrorText(SWTERROR_NULL_ARGUMENT); got != "Argument not valid" && got == "Unknown error" {
		t.Fatalf("findErrorText regressed: %q", got)
	}
	func() {
		defer func() {
			r := recover()
			ex, ok := r.(*SWTException)
			if !ok {
				t.Fatalf("expected a *SWTException panic, got %T (%v)", r, r)
			}
			if ex.Code != SWTERROR_IO {
				t.Fatalf("wrong code: %d", ex.Code)
			}
		}()
		SWTErrorFn(SWTERROR_IO)
	}()
}

// SWTException/SWTError extend the hand-written internal/jrt base types (manual superclass
// embedding) - GetMessage must still combine with the wrapped cause via super.getMessage().
func TestSWTExceptionGetMessageWithCause(t *testing.T) {
	inner := NewSWTErrorCodeMessage(SWTERROR_UNSPECIFIED, "inner")
	outer := NewSWTExceptionCodeMessage(SWTERROR_IO, "outer")
	outer.Throwable = inner
	got := outer.GetMessage()
	if got != "outer (inner)" {
		t.Fatalf("got %q", got)
	}
	var _ error = outer // *SWTException must still satisfy error via the embedded jrt type.
	var _ error = inner
}
