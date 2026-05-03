package main

import (
	"context"
	"log"
	"time"

	"vps-agent/internal/agent"
	"vps-agent/internal/config"
	"vps-agent/internal/reporter"
)

func runAgentLoop(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	rep := reporter.New(cfg)
	collector := agent.NewCollector(cfg)
	log.Printf("agent started node_id=%s server=%s interval=%s", cfg.NodeID, cfg.Server, cfg.BasicInterval)
	ticker := time.NewTicker(cfg.BasicInterval)
	defer ticker.Stop()
	for {
		metrics, err := collector.Collect(ctx)
		if err != nil {
			log.Printf("collect failed: %v", err)
		} else if err := rep.Send(ctx, metrics); err != nil {
			log.Printf("report failed: %v", err)
		}
		select {
		case <-ctx.Done():
			log.Print("agent stopped")
			return nil
		case <-ticker.C:
		}
	}
}
