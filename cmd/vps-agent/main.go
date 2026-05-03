package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"vps-agent/internal/agent"
	"vps-agent/internal/config"
)

const version = "0.1.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		if err := run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "once":
		if err := once(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "test":
		if err := test(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "version":
		fmt.Printf("vps-agent %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	default:
		usage()
		os.Exit(2)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "config file path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		return runWindowsService(*configPath)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runAgentLoop(ctx, *configPath)
}

func once(args []string) error {
	cfg, err := loadForUtility("once", args)
	if err != nil {
		return err
	}

	metrics, err := agent.NewCollector(cfg).Collect(context.Background())
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(metrics)
}

func test(args []string) error {
	cfg, err := loadForUtility("test", args)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.Server, "/")+"/api/agent/ping", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("X-Node-ID", cfg.NodeID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	fmt.Printf("server reachable: yes\nauth: ok\nlatency: %s\n", time.Since(start).Round(time.Millisecond))
	return nil
}

func loadForUtility(name string, args []string) (config.Config, error) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	configPath := fs.String("config", config.DefaultPath(), "config file path")
	if err := fs.Parse(args); err != nil {
		return config.Config{}, err
	}
	cfg, err := config.Load(*configPath)
	if errors.Is(err, os.ErrNotExist) {
		return config.Default(), nil
	}
	return cfg, err
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: vps-agent <run|once|test|version> [flags]")
}
