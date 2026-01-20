package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrClauserOutputNotFound = errors.New("clauser output not found")

const (
	FavoritesGroupName = "favorites"
	FavoritesKind      = "favorites"
)

type ClauserOutputRepository struct {
	pool *pgxpool.Pool
}

func NewClauserOutputRepository(pool *pgxpool.Pool) *ClauserOutputRepository {
	return &ClauserOutputRepository{pool: pool}
}

func (r *ClauserOutputRepository) Create(ctx context.Context, input CreateClauserOutputInput) (*ClauserOutput, error) {
	query := `
		INSERT INTO app.clauser_outputs (clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
	`
	return r.scanOutput(r.pool.QueryRow(ctx, query,
		input.ClauserID, input.AgentRunID, input.Ordinal, input.GroupName,
		input.GroupOrdinal, input.Title, input.Kind, input.Content,
	))
}

func (r *ClauserOutputRepository) CreateBatch(ctx context.Context, inputs []CreateClauserOutputInput) ([]ClauserOutput, error) {
	if len(inputs) == 0 {
		return []ClauserOutput{}, nil
	}

	outputs := make([]ClauserOutput, 0, len(inputs))
	for _, input := range inputs {
		output, err := r.Create(ctx, input)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, *output)
	}
	return outputs, nil
}

func (r *ClauserOutputRepository) FindByID(ctx context.Context, outputID string) (*ClauserOutput, error) {
	query := `
		SELECT clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
		FROM app.clauser_outputs
		WHERE clauser_output_id = $1
	`
	return r.scanOutput(r.pool.QueryRow(ctx, query, outputID))
}

func (r *ClauserOutputRepository) FindByClauserID(ctx context.Context, clauserID string) ([]ClauserOutput, error) {
	query := `
		SELECT clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
		FROM app.clauser_outputs
		WHERE clauser_id = $1
		ORDER BY ordinal ASC
	`

	rows, err := r.pool.Query(ctx, query, clauserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outputs []ClauserOutput
	for rows.Next() {
		output, err := r.scanOutputFromRows(rows)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, *output)
	}

	return outputs, rows.Err()
}

// FindByClauserIDExcludingFavorites returns all outputs except the favorites row
func (r *ClauserOutputRepository) FindByClauserIDExcludingFavorites(ctx context.Context, clauserID string) ([]ClauserOutput, error) {
	query := `
		SELECT clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
		FROM app.clauser_outputs
		WHERE clauser_id = $1 AND group_name != $2
		ORDER BY ordinal ASC
	`

	rows, err := r.pool.Query(ctx, query, clauserID, FavoritesGroupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outputs []ClauserOutput
	for rows.Next() {
		output, err := r.scanOutputFromRows(rows)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, *output)
	}

	return outputs, rows.Err()
}

// FindFavorites returns the favorites row for a clauser, or nil if none exists
func (r *ClauserOutputRepository) FindFavorites(ctx context.Context, clauserID string) (*ClauserOutput, error) {
	query := `
		SELECT clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
		FROM app.clauser_outputs
		WHERE clauser_id = $1 AND group_name = $2
	`
	output, err := r.scanOutput(r.pool.QueryRow(ctx, query, clauserID, FavoritesGroupName))
	if err != nil {
		if errors.Is(err, ErrClauserOutputNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return output, nil
}

// GetOrCreateFavorites ensures a favorites row exists and returns it
func (r *ClauserOutputRepository) GetOrCreateFavorites(ctx context.Context, clauserID string) (*ClauserOutput, error) {
	favorites, err := r.FindFavorites(ctx, clauserID)
	if err != nil {
		return nil, err
	}
	if favorites != nil {
		return favorites, nil
	}

	// Create the favorites row
	emptyItems, _ := json.Marshal(map[string]interface{}{"items": []interface{}{}})
	return r.Create(ctx, CreateClauserOutputInput{
		ClauserID:    clauserID,
		Ordinal:      0, // Favorites always at ordinal 0
		GroupName:    FavoritesGroupName,
		GroupOrdinal: 0,
		Title:        "Favorites",
		Kind:         FavoritesKind,
		Content:      emptyItems,
	})
}

// AppendFavorite adds an item to the favorites content
func (r *ClauserOutputRepository) AppendFavorite(ctx context.Context, clauserID string, item map[string]interface{}) (*ClauserOutput, error) {
	favorites, err := r.GetOrCreateFavorites(ctx, clauserID)
	if err != nil {
		return nil, err
	}

	// Parse existing content
	var content map[string]interface{}
	if err := json.Unmarshal(favorites.Content, &content); err != nil {
		return nil, err
	}

	items, ok := content["items"].([]interface{})
	if !ok {
		items = []interface{}{}
	}

	items = append(items, item)
	content["items"] = items

	newContent, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE app.clauser_outputs
		SET content = $1
		WHERE clauser_output_id = $2
		RETURNING clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
	`
	return r.scanOutput(r.pool.QueryRow(ctx, query, newContent, favorites.ClauserOutputID))
}

// RemoveFavorite removes an item from the favorites content by index
func (r *ClauserOutputRepository) RemoveFavorite(ctx context.Context, clauserID string, index int) (*ClauserOutput, error) {
	favorites, err := r.FindFavorites(ctx, clauserID)
	if err != nil {
		return nil, err
	}
	if favorites == nil {
		return nil, ErrClauserOutputNotFound
	}

	// Parse existing content
	var content map[string]interface{}
	if err := json.Unmarshal(favorites.Content, &content); err != nil {
		return nil, err
	}

	items, ok := content["items"].([]interface{})
	if !ok || index < 0 || index >= len(items) {
		return nil, errors.New("invalid favorite index")
	}

	// Remove the item at index
	items = append(items[:index], items[index+1:]...)
	content["items"] = items

	newContent, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE app.clauser_outputs
		SET content = $1
		WHERE clauser_output_id = $2
		RETURNING clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
	`
	return r.scanOutput(r.pool.QueryRow(ctx, query, newContent, favorites.ClauserOutputID))
}

// RemoveFavoriteByItemID removes an item from favorites by its itemId
func (r *ClauserOutputRepository) RemoveFavoriteByItemID(ctx context.Context, clauserID, itemID string) (*ClauserOutput, error) {
	favorites, err := r.FindFavorites(ctx, clauserID)
	if err != nil {
		return nil, err
	}
	if favorites == nil {
		return nil, ErrClauserOutputNotFound
	}

	// Parse existing content
	var content map[string]interface{}
	if err := json.Unmarshal(favorites.Content, &content); err != nil {
		return nil, err
	}

	items, ok := content["items"].([]interface{})
	if !ok {
		return nil, errors.New("invalid favorites content")
	}

	// Find and remove the item with matching itemId
	found := false
	newItems := make([]interface{}, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["itemId"] == itemID {
			found = true
			continue // Skip this item (remove it)
		}
		newItems = append(newItems, item)
	}

	if !found {
		return nil, errors.New("favorite not found")
	}

	content["items"] = newItems

	newContent, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE app.clauser_outputs
		SET content = $1
		WHERE clauser_output_id = $2
		RETURNING clauser_output_id, clauser_id, agent_run_id, ordinal, group_name, group_ordinal, title, kind, content, created_at
	`
	return r.scanOutput(r.pool.QueryRow(ctx, query, newContent, favorites.ClauserOutputID))
}

// GetNextOrdinal returns the next ordinal value for a clauser's outputs
func (r *ClauserOutputRepository) GetNextOrdinal(ctx context.Context, clauserID string) (int, error) {
	query := `SELECT COALESCE(MAX(ordinal), 0) + 1 FROM app.clauser_outputs WHERE clauser_id = $1`
	var nextOrdinal int
	err := r.pool.QueryRow(ctx, query, clauserID).Scan(&nextOrdinal)
	return nextOrdinal, err
}

// DeleteByClauserID deletes all outputs for a clauser
func (r *ClauserOutputRepository) DeleteByClauserID(ctx context.Context, clauserID string) error {
	query := `DELETE FROM app.clauser_outputs WHERE clauser_id = $1`
	_, err := r.pool.Exec(ctx, query, clauserID)
	return err
}

func (r *ClauserOutputRepository) scanOutput(row pgx.Row) (*ClauserOutput, error) {
	var o ClauserOutput
	err := row.Scan(
		&o.ClauserOutputID,
		&o.ClauserID,
		&o.AgentRunID,
		&o.Ordinal,
		&o.GroupName,
		&o.GroupOrdinal,
		&o.Title,
		&o.Kind,
		&o.Content,
		&o.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClauserOutputNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (r *ClauserOutputRepository) scanOutputFromRows(rows pgx.Rows) (*ClauserOutput, error) {
	var o ClauserOutput
	err := rows.Scan(
		&o.ClauserOutputID,
		&o.ClauserID,
		&o.AgentRunID,
		&o.Ordinal,
		&o.GroupName,
		&o.GroupOrdinal,
		&o.Title,
		&o.Kind,
		&o.Content,
		&o.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
