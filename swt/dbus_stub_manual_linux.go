package swt

// SWT's D-Bus helpers GDBus and SessionManagerDBus are LGPL upstream and are not translated. On Linux the swt level
// therefore does not raise OpenDocument/OpenUrl from D-Bus nor QueryEndSession/EndSession: these are no-ops, nothing is registered.

type GDBus_GDBusMethod struct{}

func NewGDBusGDBusMethod(name string, in [][]string, out [][]string, fn func(args []any) []any) *GDBus_GDBusMethod {
	return &GDBus_GDBusMethod{}
}

func GDBusInit(methods []*GDBus_GDBusMethod, appName string) {}

type SessionManagerDBus_IListener interface {
	IsReadyToExit() bool
	Stop()
}

type SessionManagerDBus struct{}

func NewSessionManagerDBus() *SessionManagerDBus { return &SessionManagerDBus{} }

func (this *SessionManagerDBus) AddListener(listener SessionManagerDBus_IListener) {}

func (this *SessionManagerDBus) Dispose() {}
