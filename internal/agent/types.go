package agent

type Metrics struct {
	NodeID         string       `json:"node_id"`
	Timestamp      int64        `json:"ts"`
	OS             string       `json:"os"`
	Arch           string       `json:"arch"`
	Hostname       string       `json:"hostname"`
	Kernel         string       `json:"kernel"`
	OSName         string       `json:"os_name"`
	Virtualization string       `json:"virtualization"`
	CPU            CPU          `json:"cpu"`
	Memory         Memory       `json:"memory"`
	Swap           Memory       `json:"swap"`
	Load           Load         `json:"load"`
	Uptime         uint64       `json:"uptime"`
	Disks          []Disk       `json:"disks"`
	Network        Network      `json:"network"`
	DiskIO         DiskIO       `json:"disk_io"`
	Conns          *Connections `json:"connections,omitempty"`
	Processes      int          `json:"processes"`
}

type CPU struct {
	UsagePercent  float64 `json:"usage_percent"`
	Cores         int     `json:"cores"`
	PhysicalCores int     `json:"physical_cores"`
	ModelName     string  `json:"model_name"`
}

type Memory struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type Load struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type Disk struct {
	Mount       string  `json:"mount"`
	FSType      string  `json:"fs_type,omitempty"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type Network struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	RxRate  uint64 `json:"rx_rate"`
	TxRate  uint64 `json:"tx_rate"`
}

type DiskIO struct {
	ReadRate  uint64 `json:"read_rate"`
	WriteRate uint64 `json:"write_rate"`
}

type Connections struct {
	TCP int `json:"tcp"`
	UDP int `json:"udp"`
}
