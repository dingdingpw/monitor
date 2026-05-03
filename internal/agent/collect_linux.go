//go:build linux

package agent

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func readCPUTimes() (cpuTimes, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	fields := strings.Fields(string(line))
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, errors.New("invalid /proc/stat")
	}

	var values [10]uint64
	for i := 1; i < len(fields) && i <= len(values); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return cpuTimes{}, err
		}
		values[i-1] = v
	}
	var total uint64
	for _, v := range values {
		total += v
	}
	idle := values[3] + values[4]
	return cpuTimes{idle: idle, total: total}, nil
}

func readMemory() (Memory, Memory, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return Memory{}, Memory{}, err
	}
	values := map[string]uint64{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		values[key] = v * 1024
	}
	if err := scanner.Err(); err != nil {
		return Memory{}, Memory{}, err
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	if available == 0 {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	used := uint64(0)
	if total > available {
		used = total - available
	}
	swapTotal := values["SwapTotal"]
	swapFree := values["SwapFree"]
	swapUsed := uint64(0)
	if swapTotal > swapFree {
		swapUsed = swapTotal - swapFree
	}
	return Memory{Total: total, Used: used, Free: available}, Memory{Total: swapTotal, Used: swapUsed, Free: swapFree}, nil
}

func readLoad() (Load, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return Load{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return Load{}, errors.New("invalid /proc/loadavg")
	}
	l1, _ := strconv.ParseFloat(fields[0], 64)
	l5, _ := strconv.ParseFloat(fields[1], 64)
	l15, _ := strconv.ParseFloat(fields[2], 64)
	return Load{Load1: l1, Load5: l5, Load15: l15}, nil
}

func readUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, errors.New("invalid /proc/uptime")
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	return uint64(v), err
}

func readNetwork(exclude []string) (netCounters, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return netCounters{}, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var total netCounters
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if lineNo <= 2 {
			continue
		}
		namePart, valuePart, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		name := strings.TrimSpace(namePart)
		if matchAny(name, exclude) {
			continue
		}
		fields := strings.Fields(valuePart)
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		total.rx += rx
		total.tx += tx
	}
	return total, scanner.Err()
}

func readDisks(mounts []string, excludeFS []string) ([]Disk, error) {
	mountFS := linuxMountTypes()
	if autoMounts(mounts) {
		mounts = linuxAutoMounts(mountFS, excludeFS)
	}
	disks := make([]Disk, 0, len(mounts))
	seen := map[string]bool{}
	for _, mount := range mounts {
		if seen[mount] {
			continue
		}
		seen[mount] = true
		fsType := mountFS[mount]
		if matchAny(fsType, excludeFS) {
			continue
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount, &stat); err != nil {
			continue
		}
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		used := total - free
		percent := 0.0
		if total > 0 {
			percent = round2(float64(used) / float64(total) * 100)
		}
		disks = append(disks, Disk{Mount: mount, FSType: fsType, Total: total, Used: used, Free: free, UsedPercent: percent})
	}
	return disks, nil
}

func autoMounts(mounts []string) bool {
	return len(mounts) == 0 || (len(mounts) == 1 && strings.EqualFold(mounts[0], "auto"))
}

func linuxAutoMounts(mountFS map[string]string, excludeFS []string) []string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return []string{"/"}
	}
	out := []string{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		device, mount, fsType := fields[0], fields[1], fields[2]
		if seen[mount] || matchAny(fsType, excludeFS) || !linuxLikelyRealDisk(device, fsType, mount) {
			continue
		}
		seen[mount] = true
		mountFS[mount] = fsType
		out = append(out, mount)
	}
	if len(out) == 0 {
		return []string{"/"}
	}
	return out
}

func linuxLikelyRealDisk(device, fsType, mount string) bool {
	if strings.HasPrefix(mount, "/snap/") || strings.HasPrefix(mount, "/var/lib/docker/") || strings.HasPrefix(mount, "/var/lib/containerd/") {
		return false
	}
	if strings.HasPrefix(device, "/dev/") {
		return true
	}
	switch fsType {
	case "zfs", "btrfs", "xfs", "ext2", "ext3", "ext4", "ntfs", "exfat", "vfat":
		return true
	default:
		return false
	}
}

func readConnections() (Connections, error) {
	tcp := countProcNet("/proc/net/tcp") + countProcNet("/proc/net/tcp6")
	udp := countProcNet("/proc/net/udp") + countProcNet("/proc/net/udp6")
	return Connections{TCP: tcp, UDP: udp}, nil
}

func readDiskCounters() (diskCounters, error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return diskCounters{}, err
	}
	var total diskCounters
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 || !linuxLikelyBlockDevice(fields[2]) {
			continue
		}
		sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
		sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)
		total.read += sectorsRead * 512
		total.write += sectorsWritten * 512
	}
	return total, scanner.Err()
}

func readProcessCount() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && isDigits(entry.Name()) {
			count++
		}
	}
	return count
}

func readHostInfo() HostStaticInfo {
	info := HostStaticInfo{Kernel: readTrimmed("/proc/sys/kernel/osrelease"), OSName: readOSName(), Virtualization: readVirtualization()}
	info.CPUModel, info.PhysicalCores = readCPUDetails()
	return info
}

func readOSName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	values := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok {
			values[key] = strings.Trim(value, "\"")
		}
	}
	if pretty := values["PRETTY_NAME"]; pretty != "" {
		return "Linux (" + pretty + ")"
	}
	if name := values["NAME"]; name != "" {
		return "Linux (" + name + ")"
	}
	return "Linux"
}

func readCPUDetails() (string, int) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", runtime.NumCPU()
	}
	model := ""
	physicalIDs := map[string]bool{}
	coreIDs := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	currentPhysicalID := "0"
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "model name":
			if model == "" {
				model = value
			}
		case "physical id":
			currentPhysicalID = value
			physicalIDs[value] = true
		case "core id":
			coreIDs[currentPhysicalID+":"+value] = true
		}
	}
	physicalCores := len(coreIDs)
	if physicalCores == 0 {
		physicalCores = len(physicalIDs)
	}
	if physicalCores == 0 {
		physicalCores = runtime.NumCPU()
	}
	return model, physicalCores
}

func readVirtualization() string {
	if product := strings.ToLower(readTrimmed("/sys/class/dmi/id/product_name")); product != "" {
		switch {
		case strings.Contains(product, "kvm") || strings.Contains(product, "qemu"):
			return "qemu"
		case strings.Contains(product, "vmware"):
			return "vmware"
		case strings.Contains(product, "virtualbox"):
			return "virtualbox"
		case strings.Contains(product, "hyper-v") || strings.Contains(product, "virtual machine"):
			return "hyper-v"
		}
	}
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil && bytes.Contains(bytes.ToLower(data), []byte("hypervisor")) {
		return "virtualized"
	}
	return ""
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func linuxLikelyBlockDevice(name string) bool {
	return strings.HasPrefix(name, "sd") || strings.HasPrefix(name, "vd") || strings.HasPrefix(name, "xvd") || strings.HasPrefix(name, "nvme")
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func linuxMountTypes() map[string]string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}
	out := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 {
			out[fields[1]] = fields[2]
		}
	}
	return out
}

func countProcNet(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := -1
	for scanner.Scan() {
		count++
	}
	if count < 0 {
		return 0
	}
	return count
}

func matchAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		matched, _ := filepath.Match(pattern, value)
		if matched || value == pattern {
			return true
		}
	}
	return false
}
