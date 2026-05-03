//go:build !windows

package main

import "context"

func runWindowsService(configPath string) error {
	return runAgentLoop(context.Background(), configPath)
}
