package sampler

import (
	"encoding/binary"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")
	iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	pGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	pGetTickCount64       = kernel32.NewProc("GetTickCount64")
	pGetPerformanceInfo   = kernel32.NewProc("K32GetPerformanceInfo")
	pGetProcessMemoryInfo = kernel32.NewProc("K32GetProcessMemoryInfo")
	pNtQuerySystemInfo    = ntdll.NewProc("NtQuerySystemInformation")
	pRtlGetVersion        = ntdll.NewProc("RtlGetVersion")
	pGetIfTable2          = iphlpapi.NewProc("GetIfTable2")
	pFreeMibTable         = iphlpapi.NewProc("FreeMibTable")
)

const (
	systemProcessorPerformanceInformation = 8
	processQueryLimitedInformation        = 0x1000
	ifRow2Size                            = 1352 // MIB_IF_ROW2
	ifTypeLoopback                        = 24
	ifOperUp                              = 1
)

type windowsSource struct {
	h    Host
	self uint32
}

func newSource() (source, error) {
	s := &windowsSource{self: uint32(os.Getpid())}
	s.h.Name, _ = os.Hostname()
	buf := make([]uint32, 5+64) // OSVERSIONINFOW: size, major, minor, build, platform, 128 wchars
	buf[0] = uint32(len(buf) * 4)
	if r, _, _ := pRtlGetVersion.Call(uintptr(unsafe.Pointer(&buf[0]))); r == 0 {
		name := "Windows 10"
		if buf[3] >= 22000 {
			name = "Windows 11"
		}
		s.h.OS = name + " (build " + strconv.Itoa(int(buf[3])) + ")"
	} else {
		s.h.OS = "Windows"
	}
	s.h.CPU = regString(`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "ProcessorNameString")
	var ms memoryStatus
	ms.length = uint32(unsafe.Sizeof(ms))
	if r, _, err := pGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))); r == 0 {
		return nil, err
	}
	s.h.Memory = ms.totalPhys
	return s, nil
}

func regString(key, name string) string {
	k, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return ""
	}
	var h syscall.Handle
	if syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, k, 0, syscall.KEY_READ, &h) != nil {
		return ""
	}
	defer syscall.RegCloseKey(h)
	n, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return ""
	}
	var typ, size uint32 = 0, 512
	buf := make([]uint16, size/2)
	if syscall.RegQueryValueEx(h, n, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &size) != nil {
		return ""
	}
	return trimNul(utf16.Decode(buf[:size/2]))
}

func trimNul(r []rune) string {
	for len(r) > 0 && (r[len(r)-1] == 0 || r[len(r)-1] == ' ') {
		r = r[:len(r)-1]
	}
	return string(r)
}

func (s *windowsSource) host() Host { return s.h }
func (s *windowsSource) close()     {}

type memoryStatus struct {
	length, load                             uint32
	totalPhys, availPhys                     uint64
	totalPageFile, availPageFile             uint64
	totalVirtual, availVirtual, availExtVirt uint64
}

type perfInfo struct {
	size                                   uint32
	commitTotal, commitLimit, commitPeak   uintptr
	physTotal, physAvail, systemCache      uintptr
	kernelTotal, kernelPaged, kernelNonpag uintptr
	pageSize                               uintptr
	handles, processes, threads            uint32
}

func (s *windowsSource) read(procs bool) (raw, error) {
	var r raw
	var err error
	if r.cores, err = cpuTimesNT(); err != nil {
		return r, err
	}
	var ms memoryStatus
	ms.length = uint32(unsafe.Sizeof(ms))
	if x, _, e := pGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))); x == 0 {
		return r, e
	}
	r.mem.Total = ms.totalPhys
	r.mem.Free = ms.availPhys
	r.mem.Used = ms.totalPhys - min(ms.availPhys, ms.totalPhys)
	r.mem.App = r.mem.Used
	var pi perfInfo
	pi.size = uint32(unsafe.Sizeof(pi))
	if x, _, _ := pGetPerformanceInfo.Call(uintptr(unsafe.Pointer(&pi)), uintptr(pi.size)); x != 0 {
		// the cache is partly in use: only what is also available counts as reclaimable
		r.mem.Cached = min(uint64(pi.systemCache*pi.pageSize), ms.availPhys)
		// the commit charge beyond RAM is what lives in the page file
		if total := uint64(pi.commitLimit * pi.pageSize); total > ms.totalPhys {
			r.mem.SwapTotal = total - ms.totalPhys
			if used := uint64(pi.commitTotal * pi.pageSize); used > ms.totalPhys {
				r.mem.SwapUsed = min(used-ms.totalPhys, r.mem.SwapTotal)
			}
		}
	}
	r.rx, r.tx = ifOctets()
	ms64, _, _ := pGetTickCount64.Call()
	r.uptime = time.Duration(ms64) * time.Millisecond
	if procs {
		r.procs = processes()
	}
	return r, nil
}

// cpuTimesNT reads the per processor counters, 100 ns units; the kernel time includes the idle time.
func cpuTimesNT() ([]cpuTimes, error) {
	const rec = 48
	buf := make([]byte, rec*(runtime.NumCPU()+64))
	var n uint32
	st, _, _ := pNtQuerySystemInfo.Call(systemProcessorPerformanceInformation, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)))
	if st != 0 {
		return nil, syscall.Errno(st)
	}
	out := make([]cpuTimes, int(n)/rec)
	for i := range out {
		b := buf[i*rec:]
		idle := binary.LittleEndian.Uint64(b[0:])
		kernel := binary.LittleEndian.Uint64(b[8:])
		user := binary.LittleEndian.Uint64(b[16:])
		out[i] = cpuTimes{User: user, System: kernel - min(idle, kernel), Idle: idle}
	}
	return out, nil
}

// ifOctets sums the octet counters of the interfaces that are up, loopback excepted.
func ifOctets() (rx, tx uint64) {
	var tab uintptr
	if x, _, _ := pGetIfTable2.Call(uintptr(unsafe.Pointer(&tab))); x != 0 || tab == 0 {
		return
	}
	defer pFreeMibTable.Call(tab)
	base := deref(tab)
	n := int(*(*uint32)(base))
	rows := unsafe.Slice((*byte)(unsafe.Add(base, 8)), n*ifRow2Size)
	for i := 0; i < n; i++ {
		b := rows[i*ifRow2Size : (i+1)*ifRow2Size]
		typ := binary.LittleEndian.Uint32(b[1128:])
		oper := binary.LittleEndian.Uint32(b[1156:])
		if typ == ifTypeLoopback || oper != ifOperUp {
			continue
		}
		rx += binary.LittleEndian.Uint64(b[1208:])
		tx += binary.LittleEndian.Uint64(b[1280:])
	}
	return
}

// deref turns an address from the OS into a pointer; go vet cannot tell it is not Go memory.
func deref(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }

func processes() map[int]rawProc {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snap)
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	out := map[int]rawProc{}
	for err = syscall.Process32First(snap, &e); err == nil; err = syscall.Process32Next(snap, &e) {
		pid := int(e.ProcessID)
		if pid == 0 {
			continue
		}
		p := rawProc{Name: syscall.UTF16ToString(e.ExeFile[:])}
		h, err := syscall.OpenProcess(processQueryLimitedInformation, false, e.ProcessID)
		if err == nil {
			var c, x, k, u syscall.Filetime
			if syscall.GetProcessTimes(h, &c, &x, &k, &u) == nil {
				p.CPU = (ft(k) + ft(u)) * 100
			}
			var mc [9]uintptr // PROCESS_MEMORY_COUNTERS: cb, faults, peak, working set, ...
			mc[0] = uintptr(len(mc) * 8)
			if r, _, _ := pGetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&mc[0])), mc[0]); r != 0 {
				p.RSS = uint64(mc[2])
			}
			syscall.CloseHandle(h)
		}
		out[pid] = p
	}
	return out
}

func ft(f syscall.Filetime) uint64 { return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime) }
