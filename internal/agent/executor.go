package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/vineel/vinagents-go/internal/repository"
)

// Executor orchestrates the execution of an agent run
type Executor struct {
	runID       string
	runRepo     *repository.AgentRunRepository
	messageRepo *repository.AgentRunMessageRepository
	anthropic   *anthropic.Client
}

// NewExecutor creates a new executor
func NewExecutor(
	runID string,
	runRepo *repository.AgentRunRepository,
	messageRepo *repository.AgentRunMessageRepository,
	anthropicClient *anthropic.Client,
) *Executor {
	return &Executor{
		runID:       runID,
		runRepo:     runRepo,
		messageRepo: messageRepo,
		anthropic:   anthropicClient,
	}
}

// Execute runs the agent
func (e *Executor) Execute(ctx context.Context) error {
	run, err := e.runRepo.FindByID(ctx, e.runID)
	if err != nil {
		return fmt.Errorf("failed to find run: %w", err)
	}

	// Check for cancellation before starting
	if run.Status == repository.AgentRunStatusCancelRequested {
		return e.handleCancellation(ctx)
	}

	// Update status to running
	now := time.Now()
	_, err = e.runRepo.Update(ctx, e.runID, repository.UpdateAgentRunInput{
		Status:    ptr(repository.AgentRunStatusRunning),
		StartedAt: &now,
	})
	if err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	if err := e.log(ctx, "info", "Agent run started", nil); err != nil {
		slog.Warn("failed to log message", "error", err)
	}

	// Build agent context
	agentCtx := e.buildContext(ctx, run)

	// Get agent factory
	factory, err := Get(run.AgentType)
	if err != nil {
		return e.handleError(ctx, err)
	}

	agent := factory(agentCtx)

	// Run agent
	if err := e.runAgent(ctx, agent, agentCtx, run); err != nil {
		return err
	}

	return nil
}

func (e *Executor) runAgent(ctx context.Context, agent Agent, agentCtx *Context, run *repository.AgentRun) error {
	// Initialize
	if err := agent.Initialize(ctx); err != nil {
		return e.handleError(ctx, fmt.Errorf("agent initialization failed: %w", err))
	}

	defer func() {
		if err := agent.Cleanup(ctx); err != nil {
			slog.Warn("agent cleanup failed", "error", err, "runId", e.runID)
		}
	}()

	steps := agent.DefineSteps()

	// Update total steps
	totalSteps := len(steps)
	_, err := e.runRepo.Update(ctx, e.runID, repository.UpdateAgentRunInput{
		TotalSteps: &totalSteps,
	})
	if err != nil {
		slog.Warn("failed to update total steps", "error", err)
	}

	// Parse input payload
	var inputPayload map[string]interface{}
	if err := json.Unmarshal(run.InputPayload, &inputPayload); err != nil {
		inputPayload = make(map[string]interface{})
	}

	var lastOutput interface{} = inputPayload

	for i, step := range steps {
		agentCtx.CurrentStep = i + 1

		// Check for cancellation
		cancelled, err := agentCtx.CheckCancel(ctx)
		if err != nil {
			slog.Warn("failed to check cancellation", "error", err)
		}
		if cancelled {
			return e.handleCancellation(ctx)
		}

		// Check if step should be skipped
		if step.ShouldSkip != nil && step.ShouldSkip(lastOutput, agentCtx) {
			if err := e.log(ctx, "info", fmt.Sprintf("Skipping step: %s", step.Name), nil); err != nil {
				slog.Warn("failed to log", "error", err)
			}
			continue
		}

		// Transform input if needed
		stepInput := lastOutput
		if step.TransformInput != nil {
			stepInput = step.TransformInput(lastOutput)
		}

		// Log step start
		if err := e.log(ctx, "info", fmt.Sprintf("Starting step: %s", step.Name), nil); err != nil {
			slog.Warn("failed to log", "error", err)
		}

		// Execute step
		result, err := step.Execute(ctx, stepInput, agentCtx)
		if err != nil {
			return e.handleError(ctx, fmt.Errorf("step %s failed: %w", step.Name, err))
		}

		lastOutput = result.Output
		agentCtx.StepOutputs[step.Name] = result.Output

		// Update current step
		currentStep := i + 1
		_, err = e.runRepo.Update(ctx, e.runID, repository.UpdateAgentRunInput{
			CurrentStep: &currentStep,
		})
		if err != nil {
			slog.Warn("failed to update current step", "error", err)
		}
	}

	// Mark as completed
	outputJSON, err := json.Marshal(lastOutput)
	if err != nil {
		outputJSON = []byte("{}")
	}

	completedAt := time.Now()
	_, err = e.runRepo.Update(ctx, e.runID, repository.UpdateAgentRunInput{
		Status:        ptr(repository.AgentRunStatusCompleted),
		OutputPayload: outputJSON,
		CompletedAt:   &completedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to mark run as completed: %w", err)
	}

	if err := e.log(ctx, "info", "Agent run completed", nil); err != nil {
		slog.Warn("failed to log", "error", err)
	}

	return nil
}

func (e *Executor) buildContext(ctx context.Context, run *repository.AgentRun) *Context {
	var inputPayload map[string]interface{}
	if err := json.Unmarshal(run.InputPayload, &inputPayload); err != nil {
		inputPayload = make(map[string]interface{})
	}

	return &Context{
		RunID:       run.AgentRunID,
		UserID:      run.UserID,
		Input:       inputPayload,
		CurrentStep: 0,
		Anthropic:   e.anthropic,
		StepOutputs: make(map[string]interface{}),
		Log: func(ctx context.Context, message string, level string, details json.RawMessage) error {
			return e.log(ctx, level, message, details)
		},
		CheckCancel: func(ctx context.Context) (bool, error) {
			return e.checkCancellation(ctx)
		},
	}
}

func (e *Executor) log(ctx context.Context, level, message string, details json.RawMessage) error {
	_, err := e.messageRepo.Create(ctx, repository.CreateAgentRunMessageInput{
		AgentRunID: e.runID,
		Level:      repository.MessageLevel(level),
		Message:    message,
		Details:    details,
	})
	slog.Info(message, "runId", e.runID, "level", level)
	return err
}

func (e *Executor) checkCancellation(ctx context.Context) (bool, error) {
	run, err := e.runRepo.FindByID(ctx, e.runID)
	if err != nil {
		return false, err
	}
	return run.Status == repository.AgentRunStatusCancelRequested, nil
}

func (e *Executor) handleCancellation(ctx context.Context) error {
	completedAt := time.Now()
	_, err := e.runRepo.Update(ctx, e.runID, repository.UpdateAgentRunInput{
		Status:      ptr(repository.AgentRunStatusCancelled),
		CompletedAt: &completedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to mark run as cancelled: %w", err)
	}

	if err := e.log(ctx, "info", "Run cancelled by user request", nil); err != nil {
		slog.Warn("failed to log cancellation", "error", err)
	}

	return nil
}

func (e *Executor) handleError(ctx context.Context, execErr error) error {
	completedAt := time.Now()
	errMsg := execErr.Error()

	_, err := e.runRepo.Update(ctx, e.runID, repository.UpdateAgentRunInput{
		Status:       ptr(repository.AgentRunStatusFailed),
		ErrorMessage: &errMsg,
		CompletedAt:  &completedAt,
	})
	if err != nil {
		slog.Error("failed to mark run as failed", "error", err)
	}

	if err := e.log(ctx, "error", fmt.Sprintf("Agent run failed: %s", execErr.Error()), nil); err != nil {
		slog.Warn("failed to log error", "error", err)
	}

	return execErr
}

func ptr[T any](v T) *T {
	return &v
}
