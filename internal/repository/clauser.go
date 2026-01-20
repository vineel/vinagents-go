package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrClauserNotFound = errors.New("clauser not found")

type ClauserRepository struct {
	pool *pgxpool.Pool
}

func NewClauserRepository(pool *pgxpool.Pool) *ClauserRepository {
	return &ClauserRepository{pool: pool}
}

func (r *ClauserRepository) Create(ctx context.Context, input CreateClauserInput) (*Clauser, error) {
	query := `
		INSERT INTO app.clausers (user_id, title, represented_party, drafting_approach)
		VALUES ($1, $2, 'Customer', 'Draft from scratch - Optimize for alignment')
		RETURNING clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		          represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		          agent_run_id, clause_c_history, created_at, updated_at
	`
	return r.scanClauser(r.pool.QueryRow(ctx, query, input.UserID, input.Title))
}

func (r *ClauserRepository) FindByID(ctx context.Context, clauserID string) (*Clauser, error) {
	query := `
		SELECT clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		       represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		       agent_run_id, clause_c_history, created_at, updated_at
		FROM app.clausers
		WHERE clauser_id = $1
	`
	return r.scanClauser(r.pool.QueryRow(ctx, query, clauserID))
}

func (r *ClauserRepository) FindByIDAndUserID(ctx context.Context, clauserID, userID string) (*Clauser, error) {
	query := `
		SELECT clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		       represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		       agent_run_id, clause_c_history, created_at, updated_at
		FROM app.clausers
		WHERE clauser_id = $1 AND user_id = $2
	`
	return r.scanClauser(r.pool.QueryRow(ctx, query, clauserID, userID))
}

func (r *ClauserRepository) FindByUserID(ctx context.Context, userID string, filters ListClausersFilters) ([]Clauser, error) {
	limit := filters.Limit
	if limit == 0 {
		limit = 20
	}

	query := `
		SELECT clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		       represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		       agent_run_id, clause_c_history, created_at, updated_at
		FROM app.clausers
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, filters.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clausers []Clauser
	for rows.Next() {
		clauser, err := r.scanClauserFromRows(rows)
		if err != nil {
			return nil, err
		}
		clausers = append(clausers, *clauser)
	}

	return clausers, rows.Err()
}

func (r *ClauserRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM app.clausers WHERE user_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *ClauserRepository) Update(ctx context.Context, clauserID string, input UpdateClauserInput) (*Clauser, error) {
	fields := []string{}
	params := []interface{}{}
	paramIndex := 1

	if input.Title != nil {
		fields = append(fields, fmt.Sprintf("title = $%d", paramIndex))
		params = append(params, *input.Title)
		paramIndex++
	}
	if input.AgreementA != nil {
		fields = append(fields, fmt.Sprintf("agreement_a = $%d", paramIndex))
		params = append(params, *input.AgreementA)
		paramIndex++
	}
	if input.AgreementB != nil {
		fields = append(fields, fmt.Sprintf("agreement_b = $%d", paramIndex))
		params = append(params, *input.AgreementB)
		paramIndex++
	}
	if input.ClauseA != nil {
		fields = append(fields, fmt.Sprintf("clause_a = $%d", paramIndex))
		params = append(params, *input.ClauseA)
		paramIndex++
	}
	if input.ClauseB != nil {
		fields = append(fields, fmt.Sprintf("clause_b = $%d", paramIndex))
		params = append(params, *input.ClauseB)
		paramIndex++
	}
	if input.RepresentedParty != nil {
		fields = append(fields, fmt.Sprintf("represented_party = $%d", paramIndex))
		params = append(params, *input.RepresentedParty)
		paramIndex++
	}
	if input.DraftingApproach != nil {
		fields = append(fields, fmt.Sprintf("drafting_approach = $%d", paramIndex))
		params = append(params, *input.DraftingApproach)
		paramIndex++
	}
	if input.Playbook != nil {
		fields = append(fields, fmt.Sprintf("playbook = $%d", paramIndex))
		params = append(params, *input.Playbook)
		paramIndex++
	}
	if input.CounterpartyRationale != nil {
		fields = append(fields, fmt.Sprintf("counterparty_rationale = $%d", paramIndex))
		params = append(params, *input.CounterpartyRationale)
		paramIndex++
	}
	if input.BusinessContext != nil {
		fields = append(fields, fmt.Sprintf("business_context = $%d", paramIndex))
		params = append(params, *input.BusinessContext)
		paramIndex++
	}
	if input.AgentRunID != nil {
		fields = append(fields, fmt.Sprintf("agent_run_id = $%d", paramIndex))
		params = append(params, *input.AgentRunID)
		paramIndex++
	}

	if len(fields) == 0 {
		return r.FindByID(ctx, clauserID)
	}

	params = append(params, clauserID)
	query := fmt.Sprintf(`
		UPDATE app.clausers
		SET %s
		WHERE clauser_id = $%d
		RETURNING clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		          represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		          agent_run_id, clause_c_history, created_at, updated_at
	`, strings.Join(fields, ", "), paramIndex)

	return r.scanClauser(r.pool.QueryRow(ctx, query, params...))
}

// UpdateField updates a single text field on a clauser.
// Empty string values are converted to NULL.
func (r *ClauserRepository) UpdateField(ctx context.Context, clauserID, fieldName, value string) (*Clauser, error) {
	allowedFields := map[string]bool{
		"title":                  true,
		"agreement_a":            true,
		"agreement_b":            true,
		"clause_a":               true,
		"clause_b":               true,
		"represented_party":      true,
		"drafting_approach":      true,
		"playbook":               true,
		"counterparty_rationale": true,
		"business_context":       true,
	}

	if !allowedFields[fieldName] {
		return nil, fmt.Errorf("invalid field name: %s", fieldName)
	}

	// Convert empty string to NULL
	var valueParam interface{} = value
	if value == "" {
		valueParam = nil
	}

	query := fmt.Sprintf(`
		UPDATE app.clausers
		SET %s = $1
		WHERE clauser_id = $2
		RETURNING clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		          represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		          agent_run_id, clause_c_history, created_at, updated_at
	`, fieldName)

	return r.scanClauser(r.pool.QueryRow(ctx, query, valueParam, clauserID))
}

// AppendClauseCHistory appends a new clause C version to the history
func (r *ClauserRepository) AppendClauseCHistory(ctx context.Context, clauserID string, entry map[string]interface{}) (*Clauser, error) {
	query := `
		UPDATE app.clausers
		SET clause_c_history = clause_c_history || $1::jsonb
		WHERE clauser_id = $2
		RETURNING clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		          represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		          agent_run_id, clause_c_history, created_at, updated_at
	`
	return r.scanClauser(r.pool.QueryRow(ctx, query, entry, clauserID))
}

// SetAgentRunID sets or clears the agent_run_id on a clauser
func (r *ClauserRepository) SetAgentRunID(ctx context.Context, clauserID string, agentRunID *string) (*Clauser, error) {
	query := `
		UPDATE app.clausers
		SET agent_run_id = $1
		WHERE clauser_id = $2
		RETURNING clauser_id, user_id, title, agreement_a, agreement_b, clause_a, clause_b,
		          represented_party, drafting_approach, playbook, counterparty_rationale, business_context,
		          agent_run_id, clause_c_history, created_at, updated_at
	`
	return r.scanClauser(r.pool.QueryRow(ctx, query, agentRunID, clauserID))
}

func (r *ClauserRepository) Delete(ctx context.Context, clauserID string) error {
	query := `DELETE FROM app.clausers WHERE clauser_id = $1`
	result, err := r.pool.Exec(ctx, query, clauserID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrClauserNotFound
	}
	return nil
}

func (r *ClauserRepository) scanClauser(row pgx.Row) (*Clauser, error) {
	var c Clauser
	err := row.Scan(
		&c.ClauserID,
		&c.UserID,
		&c.Title,
		&c.AgreementA,
		&c.AgreementB,
		&c.ClauseA,
		&c.ClauseB,
		&c.RepresentedParty,
		&c.DraftingApproach,
		&c.Playbook,
		&c.CounterpartyRationale,
		&c.BusinessContext,
		&c.AgentRunID,
		&c.ClauseCHistory,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClauserNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *ClauserRepository) scanClauserFromRows(rows pgx.Rows) (*Clauser, error) {
	var c Clauser
	err := rows.Scan(
		&c.ClauserID,
		&c.UserID,
		&c.Title,
		&c.AgreementA,
		&c.AgreementB,
		&c.ClauseA,
		&c.ClauseB,
		&c.RepresentedParty,
		&c.DraftingApproach,
		&c.Playbook,
		&c.CounterpartyRationale,
		&c.BusinessContext,
		&c.AgentRunID,
		&c.ClauseCHistory,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
