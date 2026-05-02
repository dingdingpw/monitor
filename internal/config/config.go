package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server             string
	Token              string
	NodeID             string
	BasicInterval      time.Duration
	DiskInterval       time.Duration
	ConnectionInterval time.Duration
	Mounts             []string
	NetworkExclude     []string
	DiskExcludeFS      []string
}

func Default() Config {
	host, _ := os.Hostname()
	if host == "" {
		host = runtime.GOOS + "-" + runtime.GOARCH
	}
	return Config{
		NodeID:             host,
		BasicInterval:      2 * time.Second,
		DiskInterval:       30 * time.Second,
		ConnectionInterval: 60 * time.Second,
		Mounts:             []string{"auto"},
		NetworkExclude:     []string{"lo", "docker*", "veth*", "br-*"},
		DiskExcludeFS:      []string{"tmpfs", "devtmpfs", "overlay", "squashfs", "proc", "sysfs", "cgroup", "cgroup2"},
	}
}

func DefaultPath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\vps-agent\config.env`
	}
	return "/etc/vps-agent/config.env"
}

func Load(path string) (Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("invalid config line %d", lineNo)
		}
		key = strings.TrimSpace(key)
		value = trimValue(value)
		if err := apply(&cfg, key, value); err != nil {
			return Config{}, fmt.Errorf("invalid config line %d: %w", lineNo, err)
		}
	}
	return cfg, scanner.Err()
}

func (c Config) Validate() error {
	if c.Server == "" {
		return errors.New("SERVER is required")
	}
	if c.Token == "" {
		return errors.New("TOKEN is required")
	}
	if c.NodeID == "" {
		return errors.New("NODE_ID is required")
	}
	if c.BasicInterval < time.Second {
		return errors.New("BASIC_INTERVAL must be >= 1s")
	}
	return nil
}

func apply(c *Config, key, value string) error {
	switch strings.ToUpper(key) {
	case "SERVER":
		c.Server = strings.TrimRight(value, "/")
	case "TOKEN":
		c.Token = value
	case "NODE_ID":
		c.NodeID = value
	case "BASIC_INTERVAL":
		d, err := parseDuration(value)
		if err != nil {
			return err
		}
		c.BasicInterval = d
	case "DISK_INTERVAL":
		d, err := parseDuration(value)
		if err != nil {
			return err
		}
		c.DiskInterval = d
	case "CONNECTION_INTERVAL":
		d, err := parseDuration(value)
		if err != nil {
			return err
		}
		c.ConnectionInterval = d
	case "MOUNTS":
		c.Mounts = splitList(value)
	case "NETWORK_EXCLUDE":
		c.NetworkExclude = splitList(value)
	case "DISK_EXCLUDE_FS":
		c.DiskExcludeFS = splitList(value)
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}

func parseDuration(value string) (time.Duration, error) {
	if _, err := strconv.Atoi(value); err == nil {
		value += "s"
	}
	return time.ParseDuration(value)
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func trimValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}
