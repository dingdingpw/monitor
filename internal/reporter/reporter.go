package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"vps-agent/internal/agent"
	"vps-agent/internal/config"
)

type Reporter struct {
	cfg    config.Config
	client *http.Client
}

func New(cfg config.Config) *Reporter {
	return &Reporter{
		cfg: cfg,
		client: &http.Client{
			Timeout: 8 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        2,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (r *Reporter) Send(ctx context.Context, metrics agent.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	url := strings.TrimRight(r.cfg.Server, "/") + "/api/agent/report"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.cfg.Token)
	req.Header.Set("X-Node-ID", r.cfg.NodeID)

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}
