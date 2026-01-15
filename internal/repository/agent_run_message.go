package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAgentRunMessageNotFound = errors.New("agent run message not found")

type AgentRunMessageRepository struct {
	pool *pgxpool.Pool
}

func NewAgentRunMessageRepository(pool *pgxpool.Pool) *AgentRunMessageRepository {
	return &AgentRunMessageRepository{pool: pool}
}

func (r *AgentRunMessageRepository) Create(ctx context.Context, input CreateAgentRunMessageInput) (*AgentRunMessage, error) {
	level := input.Level
	if level == "" {
		level = MessageLevelInfo
	}

	query := `
		INSERT INTO app.agent_run_messages (agent_run_id, step_number, level, message, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING agent_message_id, agent_run_id, step_number, level, message, details, created_at
	`
	return r.scanMessage(r.pool.QueryRow(ctx, query,
		input.AgentRunID,
		input.StepNumber,
		level,
		input.Message,
		input.Details,
	))
}

func (r *AgentRunMessageRepository) FindByRunID(ctx context.Context, agentRunID string, since *time.Time) ([]AgentRunMessage, error) {
	var query string
	var params []interface{}

	if since != nil {
		query = `
			SELECT agent_message_id, agent_run_id, step_number, level, message, details, created_at
			FROM app.agent_run_messages
			WHERE agent_run_id = $1 AND created_at > $2
			ORDER BY created_at ASC
		`
		params = []interface{}{agentRunID, *since}
	} else {
		query = `
			SELECT agent_message_id, agent_run_id, step_number, level, message, details, created_at
			FROM app.agent_run_messages
			WHERE agent_run_id = $1
			ORDER BY created_at ASC
		`
		params = []interface{}{agentRunID}
	}

	rows, err := r.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []AgentRunMessage
	for rows.Next() {
		msg, err := r.scanMessageFromRows(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}

	return messages, rows.Err()
}

func (r *AgentRunMessageRepository) scanMessage(row pgx.Row) (*AgentRunMessage, error) {
	var msg AgentRunMessage
	err := row.Scan(
		&msg.AgentMessageID,
		&msg.AgentRunID,
		&msg.StepNumber,
		&msg.Level,
		&msg.Message,
		&msg.Details,
		&msg.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAgentRunMessageNotFound
		}
		return nil, err
	}
	return &msg, nil
}

func (r *AgentRunMessageRepository) scanMessageFromRows(rows pgx.Rows) (*AgentRunMessage, error) {
	var msg AgentRunMessage
	err := rows.Scan(
		&msg.AgentMessageID,
		&msg.AgentRunID,
		&msg.StepNumber,
		&msg.Level,
		&msg.Message,
		&msg.Details,
		&msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
