package sampler

import (
	"encoding/binary"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// libSystem calls, bound once. Mach and libproc have no stable syscall numbers, so they go through
// the library like any C program would.
var (
	machHostSelf       func() uint32
	hostProcessorInfo  func(host uint32, flavor int32, count *uint32, info *uintptr, infoCnt *uint32) int32
	hostStatistics64   func(host uint32, flavor int32, info unsafe.Pointer, count *uint32) int32
	vmDeallocate       func(task uint32, addr, size uintptr) int32
	machTimebaseInfo   func(info *[2]uint32) int32
	sysctlByName       func(name string, oldp unsafe.Pointer, oldlen *uintptr, newp unsafe.Pointer, newlen uintptr) int32
	sysctlMIB          func(mib *int32, n uint32, oldp unsafe.Pointer, oldlen *uintptr, newp unsafe.Pointer, newlen uintptr) int32
	procListAllPIDs    func(buf unsafe.Pointer, size int32) int32
	procPIDInfo        func(pid, flavor int32, arg uint64, buf unsafe.Pointer, size int32) int32
	procName           func(pid int32, buf unsafe.Pointer, size uint32) int32
	machTaskSelfSymbol uintptr
)

const (
	processorCPULoadInfo = 2
	hostVMInfo64         = 4
	procPIDTaskInfo      = 4
)

type darwinSource struct {
	h         Host
	mach      uint32
	numer     uint64
	denom     uint64
	pageSize  uint64
	bootTime  time.Time
	ownUID    uint32
	pidBuf    []int32
	nameBuf   [256]byte
	taskInfo  [96]byte
	cpuInfoSz uintptr
}

func newSource() (source, error) {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, err
	}
	purego.RegisterLibFunc(&machHostSelf, lib, "mach_host_self")
	purego.RegisterLibFunc(&hostProcessorInfo, lib, "host_processor_info")
	purego.RegisterLibFunc(&hostStatistics64, lib, "host_statistics64")
	purego.RegisterLibFunc(&vmDeallocate, lib, "vm_deallocate")
	purego.RegisterLibFunc(&machTimebaseInfo, lib, "mach_timebase_info")
	purego.RegisterLibFunc(&sysctlByName, lib, "sysctlbyname")
	purego.RegisterLibFunc(&sysctlMIB, lib, "sysctl")
	purego.RegisterLibFunc(&procListAllPIDs, lib, "proc_listallpids")
	purego.RegisterLibFunc(&procPIDInfo, lib, "proc_pidinfo")
	purego.RegisterLibFunc(&procName, lib, "proc_name")
	if machTaskSelfSymbol, err = purego.Dlsym(lib, "mach_task_self_"); err != nil {
		return nil, err
	}
	s := &darwinSource{mach: machHostSelf(), pageSize: uint64(os.Getpagesize())}
	var tb [2]uint32
	machTimebaseInfo(&tb)
	s.numer, s.denom = uint64(tb[0]), uint64(tb[1])
	if s.denom == 0 {
		s.numer, s.denom = 1, 1
	}
	s.h.Name, _ = os.Hostname()
	s.h.Name = strings.TrimSuffix(s.h.Name, ".local")
	s.h.CPU = sysctlString("machdep.cpu.brand_string")
	if v := sysctlString("kern.osproductversion"); v != "" {
		s.h.OS = "macOS " + v
	} else {
		s.h.OS = "macOS"
	}
	if b := sysctlBytes("hw.memsize", 8); len(b) == 8 {
		s.h.Memory = binary.LittleEndian.Uint64(b)
	}
	if b := sysctlBytes("kern.boottime", 16); len(b) == 16 {
		s.bootTime = time.Unix(int64(binary.LittleEndian.Uint64(b)), 0)
	}
	s.ownUID = uint32(os.Getuid())
	return s, nil
}

func (s *darwinSource) host() Host { return s.h }
func (s *darwinSource) close()     {}

// deref turns an address from the kernel into a pointer; go vet cannot tell it is not Go memory.
func deref(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }

func sysctlBytes(name string, size uintptr) []byte {
	buf := make([]byte, size)
	if sysctlByName(name, unsafe.Pointer(&buf[0]), &size, nil, 0) != 0 {
		return nil
	}
	return buf[:size]
}

func sysctlString(name string) string {
	var n uintptr
	if sysctlByName(name, nil, &n, nil, 0) != 0 || n == 0 {
		return ""
	}
	b := sysctlBytes(name, n)
	return strings.TrimRight(string(b), "\x00")
}

func (s *darwinSource) read(procs bool) (raw, error) {
	var r raw
	var err error
	if r.cores, err = s.cpu(); err != nil {
		return r, err
	}
	s.memory(&r.mem)
	r.rx, r.tx = netBytes()
	if b := sysctlBytes("vm.loadavg", 24); len(b) == 24 {
		scale := float64(binary.LittleEndian.Uint64(b[16:]))
		for i := range r.load {
			r.load[i] = float64(binary.LittleEndian.Uint32(b[i*4:])) / scale
		}
		r.hasLoad = true
	}
	r.uptime = time.Since(s.bootTime)
	if procs {
		r.procs = s.processes()
	}
	return r, nil
}

func (s *darwinSource) cpu() ([]cpuTimes, error) {
	var n, cnt uint32
	var info uintptr
	if kr := hostProcessorInfo(s.mach, processorCPULoadInfo, &n, &info, &cnt); kr != 0 {
		return nil, os.ErrInvalid
	}
	ticks := unsafe.Slice((*uint32)(deref(info)), int(cnt))
	out := make([]cpuTimes, n)
	// per core: user, system, idle, nice
	for i := range out {
		t := ticks[i*4 : i*4+4]
		out[i] = cpuTimes{User: uint64(t[0]) + uint64(t[3]), System: uint64(t[1]), Idle: uint64(t[2])}
	}
	task := *(*uint32)(deref(machTaskSelfSymbol))
	vmDeallocate(task, info, uintptr(cnt)*4)
	return out, nil
}

// memory fills the figures the way Activity Monitor adds them: app memory is the anonymous pages
// minus purgeable ones, cached files are the file-backed pages plus purgeable ones.
func (s *darwinSource) memory(m *Memory) {
	var st [38]int32
	cnt := uint32(len(st))
	m.Total = s.h.Memory
	if hostStatistics64(s.mach, hostVMInfo64, unsafe.Pointer(&st[0]), &cnt) == 0 {
		u32 := func(off int) uint64 { return uint64(uint32(st[off/4])) }
		page := s.pageSize
		wired := u32(12)
		purgeable, comp := u32(88), u32(128)
		external, internal := u32(136), u32(140)
		app := uint64(0)
		if internal > purgeable {
			app = internal - purgeable
		}
		m.App, m.Wired, m.Compressed = app*page, wired*page, comp*page
		m.Used = min(m.App+m.Wired+m.Compressed, m.Total)
		m.Cached = min((external+purgeable)*page, m.Total-m.Used)
		m.Free = m.Total - m.Used - m.Cached
	}
	if b := sysctlBytes("vm.swapusage", 32); len(b) >= 24 {
		m.SwapTotal = binary.LittleEndian.Uint64(b[0:])
		m.SwapUsed = binary.LittleEndian.Uint64(b[16:])
	}
}

// netBytes sums the 64 bit counters of the interfaces that are up and not loopback, read from the
// routing table dump (getifaddrs has 32 bit counters that wrap at 4 GB).
func netBytes() (rx, tx uint64) {
	mib := [6]int32{4, 17, 0, 0, 6, 0} // CTL_NET, PF_ROUTE, 0, 0, NET_RT_IFLIST2, 0
	var n uintptr
	if sysctlMIB(&mib[0], 6, nil, &n, nil, 0) != 0 || n == 0 {
		return
	}
	buf := make([]byte, n+n/8)
	n = uintptr(len(buf))
	if sysctlMIB(&mib[0], 6, unsafe.Pointer(&buf[0]), &n, nil, 0) != 0 {
		return
	}
	buf = buf[:n]
	const rtmIfinfo2, iffUp, iffLoopback = 0x12, 0x1, 0x8
	for len(buf) >= 4 {
		l := int(binary.LittleEndian.Uint16(buf))
		if l < 4 || l > len(buf) {
			break
		}
		if buf[3] == rtmIfinfo2 && l >= 112 {
			flags := binary.LittleEndian.Uint32(buf[8:])
			if flags&iffUp != 0 && flags&iffLoopback == 0 {
				rx += binary.LittleEndian.Uint64(buf[96:])
				tx += binary.LittleEndian.Uint64(buf[104:])
			}
		}
		buf = buf[l:]
	}
	return
}

// processes reads task info of the pids it may look at; the rest refuse and are left out.
func (s *darwinSource) processes() map[int]rawProc {
	n := int(procListAllPIDs(nil, 0))
	if n <= 0 {
		return nil
	}
	if need := n + n/4 + 16; len(s.pidBuf) < need {
		s.pidBuf = make([]int32, need)
	}
	got := int(procListAllPIDs(unsafe.Pointer(&s.pidBuf[0]), int32(len(s.pidBuf)*4)))
	if got > len(s.pidBuf) { // some releases return bytes
		got /= 4
	}
	out := make(map[int]rawProc, got)
	for _, pid := range s.pidBuf[:max(got, 0)] {
		if pid <= 0 || procPIDInfo(pid, procPIDTaskInfo, 0, unsafe.Pointer(&s.taskInfo[0]), int32(len(s.taskInfo))) < 96 {
			continue
		}
		nl := procName(pid, unsafe.Pointer(&s.nameBuf[0]), uint32(len(s.nameBuf)))
		if nl <= 0 {
			continue
		}
		rss := binary.LittleEndian.Uint64(s.taskInfo[8:])
		ticks := binary.LittleEndian.Uint64(s.taskInfo[16:]) + binary.LittleEndian.Uint64(s.taskInfo[24:])
		out[int(pid)] = rawProc{Name: string(s.nameBuf[:nl]), RSS: rss, CPU: ticks * s.numer / s.denom}
	}
	return out
}
