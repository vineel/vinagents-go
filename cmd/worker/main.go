package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/vineel/vinagents-go/internal/config"
	"github.com/vineel/vinagents-go/internal/db"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/worker"

	// Import agents to register them
	_ "github.com/vineel/vinagents-go/internal/agent/simple"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up logging
	setupLogging(cfg)

	ctx := context.Background()

	// Create database pool
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("connected to database")

	// Create Anthropic client
	anthropicClient := anthropic.NewClient()
	anthropicClientPtr := &anthropicClient

	// Create repositories
	agentRunRepo := repository.NewAgentRunRepository(pool)
	agentRunMessageRepo := repository.NewAgentRunMessageRepository(pool)

	// Create worker
	agentRunWorker := worker.NewAgentRunWorker(agentRunRepo, agentRunMessageRepo, anthropicClientPtr)

	// Create River workers
	workers := river.NewWorkers()
	river.AddWorker(workers, agentRunWorker)

	// Create River client with workers
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 5},
		},
		Workers: workers,
	})
	if err != nil {
		slog.Error("failed to create river client", "error", err)
		os.Exit(1)
	}

	// Start worker
	if err := riverClient.Start(ctx); err != nil {
		slog.Error("failed to start river worker", "error", err)
		os.Exit(1)
	}

	slog.Info("worker started", "concurrency", 5)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker...")

	// Stop worker gracefully
	if err := riverClient.Stop(ctx); err != nil {
		slog.Error("error stopping worker", "error", err)
	}

	slog.Info("worker stopped")
}

func setupLogging(cfg *config.Config) {
	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	slog.SetDefault(slog.New(handler))
}
