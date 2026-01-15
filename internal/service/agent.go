package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/vineel/vinagents-go/internal/agent"
	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/worker"
)

// AgentRunResponse represents an agent run in API responses
type AgentRunResponse struct {
	AgentRunID    string          `json:"runId"`
	UserID        string          `json:"userId"`
	AgentType     string          `json:"agentType"`
	AgentVersion  string          `json:"agentVersion"`
	InputPayload  json.RawMessage `json:"input"`
	OutputPayload json.RawMessage `json:"output,omitempty"`
	Status        string          `json:"status"`
	CurrentStep   int             `json:"currentStep"`
	TotalSteps    *int            `json:"totalSteps,omitempty"`
	ErrorMessage  *string         `json:"error,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	StartedAt     *time.Time      `json:"startedAt,omitempty"`
	CompletedAt   *time.Time      `json:"completedAt,omitempty"`
}

// AgentRunMessageResponse represents a log message in API responses
type AgentRunMessageResponse struct {
	AgentMessageID string          `json:"messageId"`
	StepNumber     *int            `json:"stepNumber,omitempty"`
	Level          string          `json:"level"`
	Message        string          `json:"message"`
	Details        json.RawMessage `json:"details,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// LaunchRunResponse is the response when launching a run
type LaunchRunResponse struct {
	RunID     string    `json:"runId"`
	Status    string    `json:"status"`
	PollURL   string    `json:"pollUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type AgentService struct {
	runRepo     *repository.AgentRunRepository
	messageRepo *repository.AgentRunMessageRepository
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

func NewAgentService(
	runRepo *repository.AgentRunRepository,
	messageRepo *repository.AgentRunMessageRepository,
	riverClient *river.Client[pgx.Tx],
	pool *pgxpool.Pool,
) *AgentService {
	return &AgentService{
		runRepo:     runRepo,
		messageRepo: messageRepo,
		riverClient: riverClient,
		pool:        pool,
	}
}

func (s *AgentService) LaunchRun(ctx context.Context, userID, agentType string, input json.RawMessage) (*LaunchRunResponse, error) {
	// Validate agent type
	if !agent.Has(agentType) {
		return nil, middleware.NewBadRequestError("Unknown agent type: " + agentType)
	}

	// Ensure input is valid JSON object
	if input == nil {
		input = json.RawMessage("{}")
	}

	// Create run record
	run, err := s.runRepo.Create(ctx, repository.CreateAgentRunInput{
		UserID:       userID,
		AgentType:    agentType,
		InputPayload: input,
	})
	if err != nil {
		return nil, middleware.NewInternalError("Failed to create agent run", err)
	}

	// Enqueue job with River
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to start transaction", err)
	}
	defer tx.Rollback(ctx)

	insertRes, err := s.riverClient.InsertTx(ctx, tx, worker.AgentRunArgs{
		RunID: run.AgentRunID,
	}, nil)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to enqueue job", err)
	}

	// Update run with job ID
	jobID := insertRes.Job.ID
	_, err = s.runRepo.Update(ctx, run.AgentRunID, repository.UpdateAgentRunInput{
		RiverJobID: &jobID,
	})
	if err != nil {
		return nil, middleware.NewInternalError("Failed to update run with job ID", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, middleware.NewInternalError("Failed to commit transaction", err)
	}

	return &LaunchRunResponse{
		RunID:     run.AgentRunID,
		Status:    string(run.Status),
		PollURL:   "/api/v1/agents/runs/" + run.AgentRunID,
		CreatedAt: run.CreatedAt,
	}, nil
}

func (s *AgentService) GetRunStatus(ctx context.Context, runID, userID string, includeMessages bool, messagesSince *time.Time) (*AgentRunResponse, []AgentRunMessageResponse, error) {
	run, err := s.runRepo.FindByIDAndUserID(ctx, runID, userID)
	if err != nil {
		return nil, nil, middleware.NewNotFoundError("Agent run not found")
	}

	response := toAgentRunResponse(run)

	var messages []AgentRunMessageResponse
	if includeMessages {
		msgs, err := s.messageRepo.FindByRunID(ctx, runID, messagesSince)
		if err != nil {
			return nil, nil, middleware.NewInternalError("Failed to fetch messages", err)
		}
		messages = make([]AgentRunMessageResponse, len(msgs))
		for i, msg := range msgs {
			messages[i] = AgentRunMessageResponse{
				AgentMessageID: msg.AgentMessageID,
				StepNumber:     msg.StepNumber,
				Level:          string(msg.Level),
				Message:        msg.Message,
				Details:        msg.Details,
				CreatedAt:      msg.CreatedAt,
			}
		}
	}

	return response, messages, nil
}

func (s *AgentService) CancelRun(ctx context.Context, runID, userID string) (*AgentRunResponse, error) {
	run, err := s.runRepo.FindByIDAndUserID(ctx, runID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Agent run not found")
	}

	// Can only cancel pending or running runs
	if run.Status != repository.AgentRunStatusPending && run.Status != repository.AgentRunStatusRunning {
		return nil, middleware.NewBadRequestError("Cannot cancel run with status '" + string(run.Status) + "'. Only 'pending' or 'running' runs can be cancelled.")
	}

	updatedRun, err := s.runRepo.UpdateStatus(ctx, runID, repository.AgentRunStatusCancelRequested, nil)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to update run status", err)
	}

	// If pending, also cancel the River job
	if run.Status == repository.AgentRunStatusPending && run.RiverJobID != nil {
		// River doesn't have a direct cancel API for pending jobs in the same way Graphile does
		// The job will be picked up and immediately see cancel_requested status
	}

	return toAgentRunResponse(updatedRun), nil
}

func (s *AgentService) ListRuns(ctx context.Context, userID string, filters repository.ListAgentRunsFilters) ([]AgentRunResponse, int, error) {
	runs, err := s.runRepo.FindByUserID(ctx, userID, filters)
	if err != nil {
		return nil, 0, middleware.NewInternalError("Failed to fetch runs", err)
	}

	total, err := s.runRepo.CountByUserID(ctx, userID, filters)
	if err != nil {
		return nil, 0, middleware.NewInternalError("Failed to count runs", err)
	}

	responses := make([]AgentRunResponse, len(runs))
	for i, run := range runs {
		responses[i] = *toAgentRunResponse(&run)
	}

	return responses, total, nil
}

func toAgentRunResponse(run *repository.AgentRun) *AgentRunResponse {
	return &AgentRunResponse{
		AgentRunID:    run.AgentRunID,
		UserID:        run.UserID,
		AgentType:     run.AgentType,
		AgentVersion:  run.AgentVersion,
		InputPayload:  run.InputPayload,
		OutputPayload: run.OutputPayload,
		Status:        string(run.Status),
		CurrentStep:   run.CurrentStep,
		TotalSteps:    run.TotalSteps,
		ErrorMessage:  run.ErrorMessage,
		CreatedAt:     run.CreatedAt,
		StartedAt:     run.StartedAt,
		CompletedAt:   run.CompletedAt,
	}
}
