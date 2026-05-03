//go:build windows

package main

import (
	"context"
	"log"
	"time"

	"golang.org/x/sys/windows/svc"
)

func runWindowsService(configPath string) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !isService {
		return runAgentLoop(context.Background(), configPath)
	}
	return svc.Run("vps-agent", windowsService{configPath: configPath})
}

type windowsService struct {
	configPath string
}

func (s windowsService) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runAgentLoop(ctx, s.configPath)
	}()
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case req := <-requests:
			switch req.Cmd {
			case svc.Interrogate:
				changes <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case err := <-done:
					if err != nil {
						log.Printf("agent stopped with error: %v", err)
					}
				case <-time.After(10 * time.Second):
					log.Print("agent stop timed out")
				}
				return false, 0
			}
		case err := <-done:
			if err != nil {
				log.Printf("agent exited with error: %v", err)
				return false, 1
			}
			return false, 0
		}
	}
}
