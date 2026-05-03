//go:build windows

package agent

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	getSystemTimes       = kernel32.NewProc("GetSystemTimes")
	globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	getTickCount64       = kernel32.NewProc("GetTickCount64")
	getDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
	getLogicalDrives     = kernel32.NewProc("GetLogicalDrives")
	iphlpapi             = syscall.NewLazyDLL("iphlpapi.dll")
	getIfTable           = iphlpapi.NewProc("GetIfTable")
)

type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type mibIfRow struct {
	Name            [256]uint16
	Index           uint32
	Type            uint32
	Mtu             uint32
	Speed           uint32
	PhysAddrLen     uint32
	PhysAddr        [8]byte
	AdminStatus     uint32
	OperStatus      uint32
	LastChange      uint32
	InOctets        uint32
	InUcastPkts     uint32
	InNUcastPkts    uint32
	InDiscards      uint32
	InErrors        uint32
	InUnknownProtos uint32
	OutOctets       uint32
	OutUcastPkts    uint32
	OutNUcastPkts   uint32
	OutDiscards     uint32
	OutErrors       uint32
	OutQLen         uint32
	DescrLen        uint32
	Descr           [256]byte
}

func readCPUTimes() (cpuTimes, error) {
	var idle, kernel, user filetime
	r1, _, err := getSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	if r1 == 0 {
		return cpuTimes{}, err
	}
	idleTicks := filetimeToUint64(idle)
	kernelTicks := filetimeToUint64(kernel)
	userTicks := filetimeToUint64(user)
	return cpuTimes{idle: idleTicks, total: kernelTicks + userTicks}, nil
}

func readMemory() (Memory, Memory, error) {
	var stat memoryStatusEx
	stat.Length = uint32(unsafe.Sizeof(stat))
	r1, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&stat)))
	if r1 == 0 {
		return Memory{}, Memory{}, err
	}
	memUsed := stat.TotalPhys - stat.AvailPhys
	swapTotal := uint64(0)
	swapFree := uint64(0)
	if stat.TotalPageFile > stat.TotalPhys {
		swapTotal = stat.TotalPageFile - stat.TotalPhys
	}
	if stat.AvailPageFile > stat.AvailPhys {
		swapFree = stat.AvailPageFile - stat.AvailPhys
	}
	swapUsed := uint64(0)
	if swapTotal > swapFree {
		swapUsed = swapTotal - swapFree
	}
	return Memory{Total: stat.TotalPhys, Used: memUsed, Free: stat.AvailPhys}, Memory{Total: swapTotal, Used: swapUsed, Free: swapFree}, nil
}

func readLoad() (Load, error) {
	return Load{}, nil
}

func readUptime() (uint64, error) {
	r1, _, _ := getTickCount64.Call()
	return uint64(r1) / 1000, nil
}

func readNetwork(exclude []string) (netCounters, error) {
	var size uint32
	getIfTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
	if size == 0 {
		return netCounters{}, nil
	}

	buf := make([]byte, size)
	r1, _, err := getIfTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0)
	if r1 != 0 {
		return netCounters{}, err
	}

	count := *(*uint32)(unsafe.Pointer(&buf[0]))
	rowBase := uintptr(unsafe.Pointer(&buf[0])) + unsafe.Sizeof(count)
	rowSize := unsafe.Sizeof(mibIfRow{})
	var total netCounters
	for i := uint32(0); i < count; i++ {
		row := (*mibIfRow)(unsafe.Pointer(rowBase + uintptr(i)*rowSize))
		total.rx += uint64(row.InOctets)
		total.tx += uint64(row.OutOctets)
	}
	return total, nil
}

func readDisks(mounts []string, excludeFS []string) ([]Disk, error) {
	if autoMounts(mounts) {
		mounts = windowsAutoMounts()
	}
	disks := make([]Disk, 0, len(mounts))
	for _, mount := range mounts {
		path, err := syscall.UTF16PtrFromString(mount)
		if err != nil {
			continue
		}
		var freeAvail, total, free uint64
		r1, _, _ := getDiskFreeSpaceExW.Call(uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(&freeAvail)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
		if r1 == 0 {
			continue
		}
		used := total - freeAvail
		percent := 0.0
		if total > 0 {
			percent = round2(float64(used) / float64(total) * 100)
		}
		disks = append(disks, Disk{Mount: mount, Total: total, Used: used, Free: freeAvail, UsedPercent: percent})
	}
	return disks, nil
}

func autoMounts(mounts []string) bool {
	return len(mounts) == 0 || (len(mounts) == 1 && strings.EqualFold(mounts[0], "auto"))
}

func windowsAutoMounts() []string {
	r1, _, _ := getLogicalDrives.Call()
	mask := uint32(r1)
	if mask == 0 {
		return []string{`C:\`}
	}
	mounts := []string{}
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		mounts = append(mounts, string(rune('A'+i))+`:\`)
	}
	return mounts
}

func readConnections() (Connections, error) {
	return Connections{TCP: windowsTCPCount(), UDP: windowsUDPCount()}, nil
}

func readDiskCounters() (diskCounters, error) {
	return diskCounters{}, nil
}

func readHostInfo() HostStaticInfo {
	return HostStaticInfo{OSName: "Windows"}
}

func readProcessCount() int {
	return powershellCount("Get-Process | Measure-Object | Select-Object -ExpandProperty Count")
}

func filetimeToUint64(ft filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

func windowsTCPCount() int {
	return powershellCount("Get-NetTCPConnection | Measure-Object | Select-Object -ExpandProperty Count")
}

func windowsUDPCount() int {
	return powershellCount("Get-NetUDPEndpoint | Measure-Object | Select-Object -ExpandProperty Count")
}

func powershellCount(script string) int {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

func matchAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == value {
			return true
		}
	}
	return false
}
