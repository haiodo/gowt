package webview

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const (
	sOK          = 0
	eNoInterface = 0x80004002
	eInvalidArg  = 0x80070057
	ptrSize      = unsafe.Sizeof(uintptr(0))
)

var (
	ole32           = syscall.NewLazyDLL("ole32.dll")
	procCoTaskAlloc = ole32.NewProc("CoTaskMemAlloc")
	procCoTaskFree  = ole32.NewProc("CoTaskMemFree")
	procMemStream   = syscall.NewLazyDLL("shlwapi.dll").NewProc("SHCreateMemStream")
	iidUnknown      = parseGUID("00000000-0000-0000-C000-000000000046")
	iidEnvOptions   = parseGUID("2fde08a8-1e9a-4766-8c05-95a9ceb9d1c5")
	iidEnvOptions4  = parseGUID("ac52d13f-0d38-475a-9dca-876580d6793e")
	sink            any
	objsMu          sync.Mutex
	objs            = map[uintptr]*comObj{}
	vtables         struct{ handler, options, options4, scheme uintptr }
	vtablesOnce     sync.Once
	keepVtables     [][]uintptr
)

// heap forces v onto the heap: its address goes to native code as a uintptr, and a stack object can move.
func heap[T any]() *T {
	p := new(T)
	sink = p
	return p
}

// ptr converts an address coming from native code back to a pointer.
func ptr(u uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&u)) }

func addr[T any](p *T) uintptr { return uintptr(unsafe.Pointer(p)) }

func hrErr(what string, hr uintptr) error {
	return fmt.Errorf("webview: %s failed: HRESULT 0x%08x", what, uint32(hr))
}

func failed(hr uintptr) bool { return int32(hr) < 0 }

// vcall calls slot of the COM object obj.
func vcall(obj uintptr, slot int, args ...uintptr) uintptr {
	vt := *(*uintptr)(ptr(obj))
	fn := *(*uintptr)(unsafe.Add(ptr(vt), uintptr(slot)*ptrSize))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func release(obj uintptr) {
	if obj != 0 {
		vcall(obj, 2)
	}
}

func coAlloc(n uintptr) uintptr {
	r, _, _ := procCoTaskAlloc.Call(n)
	return r
}

// coString is s in CoTaskMem, as the out parameters of the options getters want it.
func coString(s string) uintptr {
	u := utf16Z(s)
	p := coAlloc(uintptr(len(u)) * 2)
	copy(unsafe.Slice((*uint16)(ptr(p)), len(u)), u)
	return p
}

// takeString reads and frees a CoTaskMem string the runtime returned.
func takeString(p uintptr) string {
	s := goString(p)
	if p != 0 {
		procCoTaskFree.Call(p)
	}
	return s
}

func store(out, v uintptr) { *(*uintptr)(ptr(out)) = v }

// comObj is a COM object implemented in Go: vtbl must stay the first field. The registry keeps it alive for
// as long as native code holds a reference.
type comObj struct {
	vtbl uintptr
	refs int32
	// invoke serves the handler vtable's only method; the arguments are (sender, args) or (HRESULT, result).
	invoke func(a, b uintptr) uintptr
	// qi answers QueryInterface; nil means every interface is this object, which is what event handlers need.
	qi   func(iid [16]byte) uintptr
	data any
}

func newObj(vtbl uintptr) *comObj {
	initVtables()
	o := &comObj{vtbl: vtbl, refs: 1}
	objsMu.Lock()
	objs[addr(o)] = o
	objsMu.Unlock()
	return o
}

func lookup(this uintptr) *comObj {
	objsMu.Lock()
	defer objsMu.Unlock()
	return objs[this]
}

func newHandler(fn func(a, b uintptr) uintptr) *comObj {
	initVtables()
	o := newObj(vtables.handler)
	o.invoke = fn
	return o
}

func (o *comObj) ptr() uintptr { return addr(o) }

func cbUnknownQI(this, iid, out uintptr) uintptr {
	o := lookup(this)
	if o == nil {
		return eNoInterface
	}
	if o.qi == nil {
		o.refs++
		store(out, this)
		return sOK
	}
	r := o.qi(*(*[16]byte)(ptr(iid)))
	store(out, r)
	if r == 0 {
		return eNoInterface
	}
	return sOK
}

func cbAddRef(this uintptr) uintptr {
	o := lookup(this)
	if o == nil {
		return 0
	}
	o.refs++
	return uintptr(o.refs)
}

func cbRelease(this uintptr) uintptr {
	objsMu.Lock()
	defer objsMu.Unlock()
	o := objs[this]
	if o == nil {
		return 0
	}
	o.refs--
	if o.refs <= 0 {
		delete(objs, this)
		return 0
	}
	return uintptr(o.refs)
}

func cbInvoke(this, a, b uintptr) uintptr {
	if o := lookup(this); o != nil && o.invoke != nil {
		return o.invoke(a, b)
	}
	return sOK
}

// Environment options (ICoreWebView2EnvironmentOptions): everything unset, except the compatible browser
// version, which the loader compares with the runtime.
func cbOptGetString(this, out uintptr) uintptr { store(out, 0); return sOK }
func cbOptGetVersion(this, out uintptr) uintptr {
	store(out, coString("86.0.616.0"))
	return sOK
}
func cbOptPutString(this, v uintptr) uintptr { return sOK }
func cbOptGetBool(this, out uintptr) uintptr { *(*int32)(ptr(out)) = 0; return sOK }
func cbOptPutBool(this, v uintptr) uintptr   { return sOK }

// ICoreWebView2EnvironmentOptions4.
func cbOpt4Get(this, count, regs uintptr) uintptr {
	o := lookup(this)
	schemes, _ := o.data.([]string)
	*(*uint32)(ptr(count)) = uint32(len(schemes))
	if len(schemes) == 0 {
		store(regs, 0)
		return sOK
	}
	arr := coAlloc(uintptr(len(schemes)) * ptrSize)
	for i, s := range schemes {
		r := newObj(vtables.scheme)
		r.data = s
		store(arr+uintptr(i)*ptrSize, r.ptr())
	}
	store(regs, arr)
	return sOK
}
func cbOpt4Set(this, count, regs uintptr) uintptr { return sOK }

// ICoreWebView2CustomSchemeRegistration.
func cbSchemeName(this, out uintptr) uintptr {
	s, _ := lookup(this).data.(string)
	store(out, coString(s))
	return sOK
}
func cbSchemeTrue(this, out uintptr) uintptr { *(*int32)(ptr(out)) = 1; return sOK }
func cbSchemeOrigins(this, count, out uintptr) uintptr {
	*(*uint32)(ptr(count)) = 0
	store(out, 0)
	return sOK
}
func cbSchemeSet(this, a, b uintptr) uintptr { return sOK }
func cbSchemePut(this, v uintptr) uintptr    { return sOK }

func initVtables() {
	vtablesOnce.Do(func() {
		unk := []uintptr{syscall.NewCallback(cbUnknownQI), syscall.NewCallback(cbAddRef), syscall.NewCallback(cbRelease)}
		mk := func(extra ...uintptr) uintptr {
			t, keep := vtable(append(append([]uintptr(nil), unk...), extra...))
			keepVtables = append(keepVtables, keep)
			return t
		}
		cb := syscall.NewCallback
		vtables.handler = mk(cb(cbInvoke))
		vtables.options = mk(
			cb(cbOptGetString), cb(cbOptPutString), // AdditionalBrowserArguments
			cb(cbOptGetString), cb(cbOptPutString), // Language
			cb(cbOptGetVersion), cb(cbOptPutString), // TargetCompatibleBrowserVersion
			cb(cbOptGetBool), cb(cbOptPutBool), // AllowSingleSignOnUsingOSPrimaryAccount
		)
		vtables.options4 = mk(cb(cbOpt4Get), cb(cbOpt4Set))
		vtables.scheme = mk(
			cb(cbSchemeName),
			cb(cbSchemeTrue), cb(cbSchemePut), // TreatAsSecure
			cb(cbSchemeOrigins), cb(cbSchemeSet), // AllowedOrigins
			cb(cbSchemeTrue), cb(cbSchemePut), // HasAuthorityComponent
		)
	})
}

// newOptions is the environment options object announcing the custom schemes.
func newOptions(schemes []string) *comObj {
	initVtables()
	o := newObj(vtables.options)
	o.qi = func(iid [16]byte) uintptr {
		switch iid {
		case iidUnknown, iidEnvOptions:
			o.refs++
			return o.ptr()
		case iidEnvOptions4:
			o4 := newObj(vtables.options4)
			o4.data = schemes
			return o4.ptr()
		}
		return 0
	}
	return o
}

func realRuntimeEnv() wv2Env {
	return wv2Env{
		reg: func(root, key, name string) (string, bool) {
			h := syscall.Handle(syscall.HKEY_LOCAL_MACHINE)
			if root == "HKCU" {
				h = syscall.Handle(syscall.HKEY_CURRENT_USER)
			}
			k16, _ := syscall.UTF16PtrFromString(`SOFTWARE\` + key)
			var hk syscall.Handle
			if syscall.RegOpenKeyEx(h, k16, 0, syscall.KEY_READ, &hk) != nil {
				return "", false
			}
			defer syscall.RegCloseKey(hk)
			n16, _ := syscall.UTF16PtrFromString(name)
			var typ, n uint32
			if syscall.RegQueryValueEx(hk, n16, nil, &typ, nil, &n) != nil || (typ != syscall.REG_SZ && typ != syscall.REG_EXPAND_SZ) || n == 0 {
				return "", false
			}
			buf := make([]uint16, n/2+1)
			n = uint32(len(buf)) * 2
			if syscall.RegQueryValueEx(hk, n16, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n) != nil {
				return "", false
			}
			return syscall.UTF16ToString(buf), true
		},
		exists: func(p string) bool { _, err := os.Stat(p); return err == nil },
		glob:   func(p string) []string { m, _ := filepath.Glob(p); return m },
		getenv: os.Getenv,
	}
}

func runtimeArch() string {
	if isARM64 {
		return "arm64"
	}
	return "x64"
}

// createEnvironment calls CreateWebViewEnvironmentWithOptionsInternal of the installed runtime's own
// EmbeddedBrowserWebView.dll, what WebView2Loader.dll does after it has located that library.
func createEnvironment(dll, userData string, opts, handler uintptr) error {
	lib, err := syscall.LoadLibrary(dll)
	if err != nil {
		return fmt.Errorf("webview: loading %s: %w", dll, err)
	}
	proc, err := syscall.GetProcAddress(lib, "CreateWebViewEnvironmentWithOptionsInternal")
	if err != nil {
		return fmt.Errorf("webview: %s has no CreateWebViewEnvironmentWithOptionsInternal: %w", dll, err)
	}
	var ud []uint16
	var udp uintptr
	if userData != "" {
		ud = utf16Z(userData)
		udp = addr(&ud[0])
	}
	// (bool unknown, runtime type 0 = installed, user data folder, options, completion handler)
	hr, _, _ := syscall.SyscallN(proc, 1, 0, udp, opts, handler)
	if failed(hr) {
		return hrErr("CreateWebViewEnvironmentWithOptionsInternal", hr)
	}
	runtime.KeepAlive(ud)
	return nil
}
