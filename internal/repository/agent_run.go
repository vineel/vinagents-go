package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAgentRunNotFound = errors.New("agent run not found")

type AgentRunRepository struct {
	pool *pgxpool.Pool
}

func NewAgentRunRepository(pool *pgxpool.Pool) *AgentRunRepository {
	return &AgentRunRepository{pool: pool}
}

func (r *AgentRunRepository) Create(ctx context.Context, input CreateAgentRunInput) (*AgentRun, error) {
	query := `
		INSERT INTO app.agent_runs (user_id, agent_type, input_payload)
		VALUES ($1, $2, $3)
		RETURNING agent_run_id, user_id, agent_type, agent_version, input_payload, output_payload,
		          status, current_step, total_steps, error_message, error_details, retry_count,
		          max_retries, created_at, started_at, completed_at, updated_at, river_job_id
	`
	return r.scanRun(r.pool.QueryRow(ctx, query, input.UserID, input.AgentType, input.InputPayload))
}

func (r *AgentRunRepository) FindByID(ctx context.Context, agentRunID string) (*AgentRun, error) {
	query := `
		SELECT agent_run_id, user_id, agent_type, agent_version, input_payload, output_payload,
		       status, current_step, total_steps, error_message, error_details, retry_count,
		       max_retries, created_at, started_at, completed_at, updated_at, river_job_id
		FROM app.agent_runs
		WHERE agent_run_id = $1
	`
	return r.scanRun(r.pool.QueryRow(ctx, query, agentRunID))
}

func (r *AgentRunRepository) FindByIDAndUserID(ctx context.Context, agentRunID, userID string) (*AgentRun, error) {
	query := `
		SELECT agent_run_id, user_id, agent_type, agent_version, input_payload, output_payload,
		       status, current_step, total_steps, error_message, error_details, retry_count,
		       max_retries, created_at, started_at, completed_at, updated_at, river_job_id
		FROM app.agent_runs
		WHERE agent_run_id = $1 AND user_id = $2
	`
	return r.scanRun(r.pool.QueryRow(ctx, query, agentRunID, userID))
}

func (r *AgentRunRepository) FindByUserID(ctx context.Context, userID string, filters ListAgentRunsFilters) ([]AgentRun, error) {
	conditions := []string{"user_id = $1"}
	params := []interface{}{userID}
	paramIndex := 2

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", paramIndex))
		params = append(params, *filters.Status)
		paramIndex++
	}

	if filters.AgentType != nil {
		conditions = append(conditions, fmt.Sprintf("agent_type = $%d", paramIndex))
		params = append(params, *filters.AgentType)
		paramIndex++
	}

	limit := filters.Limit
	if limit == 0 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT agent_run_id, user_id, agent_type, agent_version, input_payload, output_payload,
		       status, current_step, total_steps, error_message, error_details, retry_count,
		       max_retries, created_at, started_at, completed_at, updated_at, river_job_id
		FROM app.agent_runs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), paramIndex, paramIndex+1)

	params = append(params, limit, filters.Offset)

	rows, err := r.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []AgentRun
	for rows.Next() {
		run, err := r.scanRunFromRows(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}

	return runs, rows.Err()
}

func (r *AgentRunRepository) CountByUserID(ctx context.Context, userID string, filters ListAgentRunsFilters) (int, error) {
	conditions := []string{"user_id = $1"}
	params := []interface{}{userID}
	paramIndex := 2

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", paramIndex))
		params = append(params, *filters.Status)
		paramIndex++
	}

	if filters.AgentType != nil {
		conditions = append(conditions, fmt.Sprintf("agent_type = $%d", paramIndex))
		params = append(params, *filters.AgentType)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM app.agent_runs
		WHERE %s
	`, strings.Join(conditions, " AND "))

	var count int
	err := r.pool.QueryRow(ctx, query, params...).Scan(&count)
	return count, err
}

func (r *AgentRunRepository) Update(ctx context.Context, agentRunID string, input UpdateAgentRunInput) (*AgentRun, error) {
	fields := []string{}
	params := []interface{}{}
	paramIndex := 1

	if input.Status != nil {
		fields = append(fields, fmt.Sprintf("status = $%d", paramIndex))
		params = append(params, *input.Status)
		paramIndex++
	}
	if input.CurrentStep != nil {
		fields = append(fields, fmt.Sprintf("current_step = $%d", paramIndex))
		params = append(params, *input.CurrentStep)
		paramIndex++
	}
	if input.TotalSteps != nil {
		fields = append(fields, fmt.Sprintf("total_steps = $%d", paramIndex))
		params = append(params, *input.TotalSteps)
		paramIndex++
	}
	if input.OutputPayload != nil {
		fields = append(fields, fmt.Sprintf("output_payload = $%d", paramIndex))
		params = append(params, input.OutputPayload)
		paramIndex++
	}
	if input.ErrorMessage != nil {
		fields = append(fields, fmt.Sprintf("error_message = $%d", paramIndex))
		params = append(params, *input.ErrorMessage)
		paramIndex++
	}
	if input.ErrorDetails != nil {
		fields = append(fields, fmt.Sprintf("error_details = $%d", paramIndex))
		params = append(params, input.ErrorDetails)
		paramIndex++
	}
	if input.StartedAt != nil {
		fields = append(fields, fmt.Sprintf("started_at = $%d", paramIndex))
		params = append(params, *input.StartedAt)
		paramIndex++
	}
	if input.CompletedAt != nil {
		fields = append(fields, fmt.Sprintf("completed_at = $%d", paramIndex))
		params = append(params, *input.CompletedAt)
		paramIndex++
	}
	if input.RiverJobID != nil {
		fields = append(fields, fmt.Sprintf("river_job_id = $%d", paramIndex))
		params = append(params, *input.RiverJobID)
		paramIndex++
	}
	if input.RetryCount != nil {
		fields = append(fields, fmt.Sprintf("retry_count = $%d", paramIndex))
		params = append(params, *input.RetryCount)
		paramIndex++
	}

	if len(fields) == 0 {
		return r.FindByID(ctx, agentRunID)
	}

	// Always update updated_at
	fields = append(fields, "updated_at = NOW()")

	params = append(params, agentRunID)
	query := fmt.Sprintf(`
		UPDATE app.agent_runs
		SET %s
		WHERE agent_run_id = $%d
		RETURNING agent_run_id, user_id, agent_type, agent_version, input_payload, output_payload,
		          status, current_step, total_steps, error_message, error_details, retry_count,
		          max_retries, created_at, started_at, completed_at, updated_at, river_job_id
	`, strings.Join(fields, ", "), paramIndex)

	return r.scanRun(r.pool.QueryRow(ctx, query, params...))
}

func (r *AgentRunRepository) UpdateStatus(ctx context.Context, agentRunID string, status AgentRunStatus, additional *UpdateAgentRunInput) (*AgentRun, error) {
	input := UpdateAgentRunInput{Status: &status}
	if additional != nil {
		if additional.CurrentStep != nil {
			input.CurrentStep = additional.CurrentStep
		}
		if additional.TotalSteps != nil {
			input.TotalSteps = additional.TotalSteps
		}
		if additional.OutputPayload != nil {
			input.OutputPayload = additional.OutputPayload
		}
		if additional.ErrorMessage != nil {
			input.ErrorMessage = additional.ErrorMessage
		}
		if additional.ErrorDetails != nil {
			input.ErrorDetails = additional.ErrorDetails
		}
		if additional.StartedAt != nil {
			input.StartedAt = additional.StartedAt
		}
		if additional.CompletedAt != nil {
			input.CompletedAt = additional.CompletedAt
		}
		if additional.RiverJobID != nil {
			input.RiverJobID = additional.RiverJobID
		}
		if additional.RetryCount != nil {
			input.RetryCount = additional.RetryCount
		}
	}
	return r.Update(ctx, agentRunID, input)
}

func (r *AgentRunRepository) scanRun(row pgx.Row) (*AgentRun, error) {
	var run AgentRun
	err := row.Scan(
		&run.AgentRunID,
		&run.UserID,
		&run.AgentType,
		&run.AgentVersion,
		&run.InputPayload,
		&run.OutputPayload,
		&run.Status,
		&run.CurrentStep,
		&run.TotalSteps,
		&run.ErrorMessage,
		&run.ErrorDetails,
		&run.RetryCount,
		&run.MaxRetries,
		&run.CreatedAt,
		&run.StartedAt,
		&run.CompletedAt,
		&run.UpdatedAt,
		&run.RiverJobID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAgentRunNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *AgentRunRepository) scanRunFromRows(rows pgx.Rows) (*AgentRun, error) {
	var run AgentRun
	err := rows.Scan(
		&run.AgentRunID,
		&run.UserID,
		&run.AgentType,
		&run.AgentVersion,
		&run.InputPayload,
		&run.OutputPayload,
		&run.Status,
		&run.CurrentStep,
		&run.TotalSteps,
		&run.ErrorMessage,
		&run.ErrorDetails,
		&run.RetryCount,
		&run.MaxRetries,
		&run.CreatedAt,
		&run.StartedAt,
		&run.CompletedAt,
		&run.UpdatedAt,
		&run.RiverJobID,
	)
	if err != nil {
		return nil, err
	}
	return &run, nil
}
