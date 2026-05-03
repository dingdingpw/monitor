package agent

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"vps-agent/internal/config"
)

type Collector struct {
	cfg config.Config

	mu         sync.Mutex
	lastCPU    cpuTimes
	lastNet    netCounters
	lastDiskIO diskCounters
	lastTime   time.Time

	disks                []Disk
	conns                *Connections
	lastDisk             time.Time
	lastConn             time.Time
	staticHost           string
	staticCores          int
	staticPhysicalCores  int
	staticOS             string
	staticOSName         string
	staticArch           string
	staticKernel         string
	staticVirtualization string
	staticCPUModel       string
}

func NewCollector(cfg config.Config) *Collector {
	host, _ := os.Hostname()
	hostInfo := readHostInfo()
	return &Collector{
		cfg:                  cfg,
		staticHost:           host,
		staticCores:          runtime.NumCPU(),
		staticPhysicalCores:  hostInfo.PhysicalCores,
		staticOS:             runtime.GOOS,
		staticOSName:         hostInfo.OSName,
		staticArch:           runtime.GOARCH,
		staticKernel:         hostInfo.Kernel,
		staticVirtualization: hostInfo.Virtualization,
		staticCPUModel:       hostInfo.CPUModel,
	}
}

func (c *Collector) Collect(ctx context.Context) (Metrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	cpuNow, err := readCPUTimes()
	if err != nil {
		return Metrics{}, err
	}
	mem, swap, err := readMemory()
	if err != nil {
		return Metrics{}, err
	}
	load, _ := readLoad()
	uptime, _ := readUptime()
	netNow, _ := readNetwork(c.cfg.NetworkExclude)
	diskIONow, _ := readDiskCounters()

	if c.lastDisk.IsZero() || now.Sub(c.lastDisk) >= c.cfg.DiskInterval {
		if disks, err := readDisks(c.cfg.Mounts, c.cfg.DiskExcludeFS); err == nil {
			c.disks = disks
			c.lastDisk = now
		}
	}
	if c.lastConn.IsZero() || now.Sub(c.lastConn) >= c.cfg.ConnectionInterval {
		if conns, err := readConnections(); err == nil {
			c.conns = &conns
			c.lastConn = now
		}
	}

	cpuUsage := 0.0
	rxRate := uint64(0)
	txRate := uint64(0)
	diskReadRate := uint64(0)
	diskWriteRate := uint64(0)
	if !c.lastTime.IsZero() {
		elapsed := now.Sub(c.lastTime).Seconds()
		cpuUsage = cpuNow.usageSince(c.lastCPU)
		if elapsed > 0 {
			if netNow.rx >= c.lastNet.rx {
				rxRate = uint64(float64(netNow.rx-c.lastNet.rx) / elapsed)
			}
			if netNow.tx >= c.lastNet.tx {
				txRate = uint64(float64(netNow.tx-c.lastNet.tx) / elapsed)
			}
			if diskIONow.read >= c.lastDiskIO.read {
				diskReadRate = uint64(float64(diskIONow.read-c.lastDiskIO.read) / elapsed)
			}
			if diskIONow.write >= c.lastDiskIO.write {
				diskWriteRate = uint64(float64(diskIONow.write-c.lastDiskIO.write) / elapsed)
			}
		}
	}

	c.lastCPU = cpuNow
	c.lastNet = netNow
	c.lastDiskIO = diskIONow
	c.lastTime = now

	select {
	case <-ctx.Done():
		return Metrics{}, ctx.Err()
	default:
	}

	return Metrics{
		NodeID:         c.cfg.NodeID,
		Timestamp:      now.Unix(),
		OS:             c.staticOS,
		Arch:           c.staticArch,
		Hostname:       c.staticHost,
		Kernel:         c.staticKernel,
		OSName:         c.staticOSName,
		Virtualization: c.staticVirtualization,
		CPU:            CPU{UsagePercent: round2(cpuUsage), Cores: c.staticCores, PhysicalCores: c.staticPhysicalCores, ModelName: c.staticCPUModel},
		Memory:         mem,
		Swap:           swap,
		Load:           load,
		Uptime:         uptime,
		Disks:          c.disks,
		Network:        Network{RxBytes: netNow.rx, TxBytes: netNow.tx, RxRate: rxRate, TxRate: txRate},
		DiskIO:         DiskIO{ReadRate: diskReadRate, WriteRate: diskWriteRate},
		Conns:          c.conns,
		Processes:      readProcessCount(),
	}, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

type cpuTimes struct {
	idle  uint64
	total uint64
}

func (c cpuTimes) usageSince(prev cpuTimes) float64 {
	if c.total <= prev.total || c.idle < prev.idle {
		return 0
	}
	total := c.total - prev.total
	idle := c.idle - prev.idle
	if total == 0 || idle > total {
		return 0
	}
	return (1 - float64(idle)/float64(total)) * 100
}

type netCounters struct {
	rx uint64
	tx uint64
}

type diskCounters struct {
	read  uint64
	write uint64
}

type HostStaticInfo struct {
	Kernel         string
	OSName         string
	Virtualization string
	CPUModel       string
	PhysicalCores  int
}
