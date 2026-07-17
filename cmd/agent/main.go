package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/3011/cfscan/internal/config"
	"github.com/3011/cfscan/internal/model"
	"github.com/3011/cfscan/internal/probe"
)

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.LoadAgent()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	apiClient := &client{
		baseURL: strings.TrimRight(cfg.CenterURL, "/"), token: cfg.Token,
		http: &http.Client{Timeout: 45 * time.Second},
	}
	agent, err := apiClient.register(ctx, model.AgentRegistration{
		Name: cfg.Name, Region: cfg.Region, Continent: cfg.Continent, Concurrency: cfg.Concurrency,
	})
	if err != nil {
		logger.Error("register agent", "error", err)
		os.Exit(1)
	}
	logger.Info("agent registered", "agent_id", agent.ID, "name", agent.Name, "region", agent.Region)

	heartbeatTicker := time.NewTicker(cfg.HeartbeatInterval)
	pollTicker := time.NewTicker(cfg.PollInterval)
	defer heartbeatTicker.Stop()
	defer pollTicker.Stop()

	// Claim immediately so a newly started Agent does not wait for the first tick.
	processAvailable(ctx, apiClient, agent.ID, cfg.Concurrency, logger)
	for {
		select {
		case <-ctx.Done():
			logger.Info("agent stopped")
			return
		case <-heartbeatTicker.C:
			if err := apiClient.heartbeat(ctx, agent.ID); err != nil {
				logger.Warn("heartbeat failed", "error", err)
			}
		case <-pollTicker.C:
			processAvailable(ctx, apiClient, agent.ID, cfg.Concurrency, logger)
		}
	}
}

func processAvailable(ctx context.Context, apiClient *client, agentID string, concurrency int, logger *slog.Logger) {
	for {
		batch, err := apiClient.claim(ctx, agentID, concurrency)
		if err != nil {
			logger.Warn("claim task failed", "error", err)
			return
		}
		if batch == nil || len(batch.Tasks) == 0 {
			return
		}
		logger.Info("scan batch claimed", "job_id", batch.JobID, "job", batch.JobName, "targets", len(batch.Tasks))
		results := runBatch(ctx, *batch, concurrency)
		if err := apiClient.submit(ctx, model.ResultBatch{AgentID: agentID, JobID: batch.JobID, Results: results}); err != nil {
			logger.Warn("submit scan results failed", "job_id", batch.JobID, "error", err)
			return
		}
		available := 0
		for _, result := range results {
			if result.Available {
				available++
			}
		}
		logger.Info("scan batch submitted", "job_id", batch.JobID, "available", available, "failed", len(results)-available)
	}
}

func runBatch(ctx context.Context, batch model.TaskBatch, concurrency int) []model.ProbeResult {
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(batch.Tasks) {
		concurrency = len(batch.Tasks)
	}
	jobs := make(chan model.ScanTask)
	results := make(chan model.ProbeResult, len(batch.Tasks))
	var wg sync.WaitGroup
	cfg := probe.Config{Scheme: batch.Scheme, Hostname: batch.Hostname, Path: batch.Path, Port: batch.Port, Attempts: batch.Attempts, TimeoutMS: batch.TimeoutMS}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				results <- probe.Run(ctx, task, cfg)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, task := range batch.Tasks {
			select {
			case jobs <- task:
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	close(results)
	items := make([]model.ProbeResult, 0, len(batch.Tasks))
	for result := range results {
		items = append(items, result)
	}
	return items
}

func (c *client) register(ctx context.Context, input model.AgentRegistration) (model.Agent, error) {
	var result model.Agent
	status, err := c.postJSON(ctx, "/api/v1/agent/register", input, &result)
	if err != nil {
		return model.Agent{}, err
	}
	if status != http.StatusOK {
		return model.Agent{}, fmt.Errorf("unexpected register status %d", status)
	}
	return result, nil
}

func (c *client) heartbeat(ctx context.Context, agentID string) error {
	status, err := c.postJSON(ctx, "/api/v1/agent/heartbeat", model.AgentHeartbeat{AgentID: agentID}, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("unexpected heartbeat status %d", status)
	}
	return nil
}

func (c *client) claim(ctx context.Context, agentID string, limit int) (*model.TaskBatch, error) {
	var result model.TaskBatch
	status, err := c.postJSON(ctx, "/api/v1/agent/tasks/claim", model.TaskClaimRequest{AgentID: agentID, Limit: limit}, &result)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNoContent {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected claim status %d", status)
	}
	return &result, nil
}

func (c *client) submit(ctx context.Context, input model.ResultBatch) error {
	status, err := c.postJSON(ctx, "/api/v1/agent/tasks/results", input, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("unexpected submit status %d", status)
	}
	return nil
}

func (c *client) postJSON(ctx context.Context, path string, input, output any) (int, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	if output != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil && !errors.Is(err, io.EOF) {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}
