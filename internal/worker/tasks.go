package worker

import (
	"context"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/riverqueue/river"
	"github.com/vineel/vinagents-go/internal/agent"
	"github.com/vineel/vinagents-go/internal/repository"
)

// AgentRunArgs are the arguments for an agent run job
type AgentRunArgs struct {
	RunID string `json:"run_id"`
}

// Kind returns the job kind for River
func (AgentRunArgs) Kind() string {
	return "agent_run"
}

// ClauserWriteArgs are the arguments for a clause write job
type ClauserWriteArgs struct {
	ClauserID    string `json:"clauser_id"`
	AgentRunID   string `json:"agent_run_id"`
	Instructions string `json:"instructions,omitempty"`
}

// Kind returns the job kind for River
func (ClauserWriteArgs) Kind() string {
	return "clauser_write"
}

// ClauserLensArgs are the arguments for a lens analysis job
type ClauserLensArgs struct {
	ClauserID  string   `json:"clauser_id"`
	AgentRunID string   `json:"agent_run_id"`
	Lenses     []string `json:"lenses"`
}

// Kind returns the job kind for River
func (ClauserLensArgs) Kind() string {
	return "clauser_lens"
}

// AgentRunWorker handles agent run jobs
type AgentRunWorker struct {
	river.WorkerDefaults[AgentRunArgs]
	runRepo         *repository.AgentRunRepository
	messageRepo     *repository.AgentRunMessageRepository
	anthropicClient *anthropic.Client
}

// NewAgentRunWorker creates a new agent run worker
func NewAgentRunWorker(
	runRepo *repository.AgentRunRepository,
	messageRepo *repository.AgentRunMessageRepository,
	anthropicClient *anthropic.Client,
) *AgentRunWorker {
	return &AgentRunWorker{
		runRepo:         runRepo,
		messageRepo:     messageRepo,
		anthropicClient: anthropicClient,
	}
}

// Work executes an agent run job
func (w *AgentRunWorker) Work(ctx context.Context, job *river.Job[AgentRunArgs]) error {
	slog.Info("starting agent run job", "runId", job.Args.RunID, "jobId", job.ID)

	executor := agent.NewExecutor(
		job.Args.RunID,
		w.runRepo,
		w.messageRepo,
		w.anthropicClient,
	)

	if err := executor.Execute(ctx); err != nil {
		slog.Error("agent run failed", "runId", job.Args.RunID, "error", err)
		// Return nil to mark job as complete (we've already marked the run as failed in the executor)
		// Returning an error would cause River to retry, which we don't want
		return nil
	}

	slog.Info("agent run completed", "runId", job.Args.RunID)
	return nil
}
