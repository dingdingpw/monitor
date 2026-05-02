//go:build !linux && !windows

package agent

import "errors"

func readCPUTimes() (cpuTimes, error)                               { return cpuTimes{}, errors.New("unsupported OS") }
func readMemory() (Memory, Memory, error)                           { return Memory{}, Memory{}, errors.New("unsupported OS") }
func readLoad() (Load, error)                                       { return Load{}, nil }
func readUptime() (uint64, error)                                   { return 0, nil }
func readNetwork(exclude []string) (netCounters, error)             { return netCounters{}, nil }
func readDisks(mounts []string, excludeFS []string) ([]Disk, error) { return nil, nil }
func readConnections() (Connections, error)                         { return Connections{}, nil }
