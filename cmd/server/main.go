package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/3011/cfscan/internal/api"
	authservice "github.com/3011/cfscan/internal/auth"
	"github.com/3011/cfscan/internal/automation"
	"github.com/3011/cfscan/internal/cloudflare"
	"github.com/3011/cfscan/internal/config"
	"github.com/3011/cfscan/internal/scans"
	"github.com/3011/cfscan/internal/scheduling"
	"github.com/3011/cfscan/internal/store/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.LoadServer()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dataStore, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer dataStore.Close()
	syncer := cloudflare.NewSyncer(dataStore)
	scanService := scans.NewService(dataStore, syncer)
	scheduleRunner := scheduling.NewRunner(dataStore, scanService, logger)
	automationService := automation.NewService(dataStore, syncer, logger)
	authService := authservice.New(dataStore, authservice.Config{SessionTTL: cfg.SessionTTL, CookieSecure: cfg.CookieSecure})
	if err := authService.EnsureBootstrapAdmin(ctx, cfg.BootstrapAdminUser, cfg.BootstrapAdminPassword); err != nil {
		logger.Error("bootstrap administrator", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: api.New(dataStore, syncer, automationService, authService, cfg.AgentToken, logger),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second,
		WriteTimeout: 45 * time.Second, IdleTimeout: 60 * time.Second,
	}

	go runSchedulers(ctx, scheduleRunner, automationService, logger)
	go func() {
		logger.Info("center server listening", "address", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve http", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown server", "error", err)
	}
}

func runSchedulers(ctx context.Context, scheduleRunner *scheduling.Runner, automationService *automation.Service, logger *slog.Logger) {
	runAll := func() {
		runCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
		defer cancel()
		if count, err := scheduleRunner.RunDue(runCtx); err != nil {
			logger.Warn("scheduled scan dispatch failed", "error", err)
		} else if count > 0 {
			logger.Info("scheduled scans dispatched", "schedules", count)
		}
		if count, err := automationService.RunDue(runCtx); err != nil {
			logger.Warn("automation dispatch failed", "error", err)
		} else if count > 0 {
			logger.Info("automations dispatched", "automations", count)
		}
	}
	go func() {
		startupCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
		defer cancel()
		if count, err := automationService.RunStartup(startupCtx); err != nil {
			logger.Warn("startup automations failed", "error", err)
		} else if count > 0 {
			logger.Info("startup automations completed", "automations", count)
		}
		runAll()
	}()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAll()
		}
	}
}
