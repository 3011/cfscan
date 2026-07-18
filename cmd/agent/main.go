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
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/3011/cfscan/internal/config"
	"github.com/3011/cfscan/internal/model"
	"github.com/3011/cfscan/internal/probe"
)

var version = "dev"

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("unexpected HTTP status %d", e.Status)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := runCLI(ctx, os.Args[1:], logger); err != nil {
		logger.Error("agent stopped with error", "error", err)
		os.Exit(1)
	}
}

func runSavedIdentity(ctx context.Context, logger *slog.Logger) error {
	identityPath := defaultIdentityPath()
	identity, err := loadIdentity(identityPath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Agent identity not found at %s; run connect or join first", identityPath)
	}
	if err != nil {
		return fmt.Errorf("load Agent identity: %w", err)
	}
	return runAgentLoop(ctx, &client{baseURL: identity.ServerURL, token: identity.Token, http: defaultHTTPClient()}, identity.Concurrency, logger)
}

func runAgentLoop(ctx context.Context, apiClient *client, concurrency int, logger *slog.Logger) error {
	cfg := config.LoadAgent()
	heartbeatTicker := time.NewTicker(cfg.HeartbeatInterval)
	pollTicker := time.NewTicker(cfg.PollInterval)
	defer heartbeatTicker.Stop()
	defer pollTicker.Stop()

	if err := apiClient.heartbeat(ctx); err != nil {
		logger.Warn("initial heartbeat failed", "error", err)
	}
	processAvailable(ctx, apiClient, concurrency, logger)
	for {
		select {
		case <-ctx.Done():
			logger.Info("agent stopped")
			return nil
		case <-heartbeatTicker.C:
			if err := apiClient.heartbeat(ctx); err != nil {
				logger.Warn("heartbeat failed", "error", err)
			}
		case <-pollTicker.C:
			processAvailable(ctx, apiClient, concurrency, logger)
		}
	}
}

func processAvailable(ctx context.Context, apiClient *client, concurrency int, logger *slog.Logger) {
	for {
		batch, err := apiClient.claim(ctx, concurrency)
		if err != nil {
			logger.Warn("claim task failed", "error", err)
			return
		}
		if batch == nil || len(batch.Tasks) == 0 {
			return
		}
		logger.Info("scan batch claimed", "job_id", batch.JobID, "job", batch.JobName, "targets", len(batch.Tasks))
		results := runBatch(ctx, *batch, concurrency)
		if err := apiClient.submit(ctx, model.ResultBatch{JobID: batch.JobID, Results: results}); err != nil {
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

func (c *client) heartbeat(ctx context.Context) error {
	status, err := c.postJSON(ctx, "/api/v1/agent/heartbeat", struct{}{}, nil)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("unexpected heartbeat status %d", status)
	}
	return nil
}

func (c *client) claim(ctx context.Context, limit int) (*model.TaskBatch, error) {
	var result model.TaskBatch
	status, err := c.postJSON(ctx, "/api/v1/agent/tasks/claim", model.TaskClaimRequest{Limit: limit}, &result)
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
	return c.doJSON(ctx, http.MethodPost, path, input, output, true)
}

func (c *client) postPublicJSON(ctx context.Context, path string, input, output any) (int, error) {
	return c.doJSON(ctx, http.MethodPost, path, input, output, false)
}

func (c *client) doJSON(ctx context.Context, method, path string, input, output any, authenticated bool) (int, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = json.Unmarshal(data, &payload)
		message := payload.Error.Message
		if message == "" {
			message = strings.TrimSpace(string(data))
		}
		return resp.StatusCode, &apiError{Status: resp.StatusCode, Code: payload.Error.Code, Message: message}
	}
	if output != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(output); err != nil && !errors.Is(err, io.EOF) {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func defaultHTTPClient() *http.Client { return &http.Client{Timeout: 45 * time.Second} }

func agentVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
