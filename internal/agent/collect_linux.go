//go:build linux

package agent

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
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
