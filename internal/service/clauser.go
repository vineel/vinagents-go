package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/worker"
)

// ClauserResponse represents a clauser in API responses
type ClauserResponse struct {
	ClauserID             string          `json:"clauserId"`
	UserID                string          `json:"userId"`
	Title                 *string         `json:"title"`
	AgreementA            *string         `json:"agreementA"`
	AgreementB            *string         `json:"agreementB"`
	ClauseA               *string         `json:"clauseA"`
	ClauseB               *string         `json:"clauseB"`
	RepresentedParty      *string         `json:"representedParty"`
	DraftingApproach      *string         `json:"draftingApproach"`
	Playbook              *string         `json:"playbook"`
	CounterpartyRationale *string         `json:"counterpartyRationale"`
	BusinessContext       *string         `json:"businessContext"`
	AgentRunID            *string         `json:"agentRunId,omitempty"`
	ClauseCHistory        json.RawMessage `json:"clauseCHistory"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
}

// ClauserOutputResponse represents a clauser output in API responses
type ClauserOutputResponse struct {
	ClauserOutputID string          `json:"clauserOutputId"`
	AgentRunID      *string         `json:"agentRunId,omitempty"`
	Ordinal         int             `json:"ordinal"`
	GroupName       string          `json:"groupName"`
	GroupOrdinal    int             `json:"groupOrdinal"`
	Title           string          `json:"title"`
	Kind            string          `json:"kind"`
	Content         json.RawMessage `json:"content"`
	CreatedAt       time.Time       `json:"createdAt"`
}

// ClauserListItem represents a clauser in list responses
type ClauserListItem struct {
	ClauserID  string    `json:"clauserId"`
	Title      *string   `json:"title"`
	HasClauseC bool      `json:"hasClauseC"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// ScreenStateResponse represents the full screen state for a clauser
type ScreenStateResponse struct {
	Clauser   ClauserResponse         `json:"clauser"`
	Outputs   []ClauserOutputResponse `json:"outputs"`
	Favorites *ClauserOutputResponse  `json:"favorites"`
	ActiveRun *ActiveRunResponse      `json:"activeRun"`
}

// ActiveRunResponse represents an active agent run
type ActiveRunResponse struct {
	RunID       string `json:"runId"`
	Status      string `json:"status"`
	CurrentStep int    `json:"currentStep"`
	TotalSteps  *int   `json:"totalSteps"`
}

// RunJobResponse is returned when a job is launched
type RunJobResponse struct {
	RunID   string `json:"runId"`
	Status  string `json:"status"`
	PollURL string `json:"pollUrl"`
}

type ClauserService struct {
	clauserRepo *repository.ClauserRepository
	outputRepo  *repository.ClauserOutputRepository
	runRepo     *repository.AgentRunRepository
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

func NewClauserService(
	clauserRepo *repository.ClauserRepository,
	outputRepo *repository.ClauserOutputRepository,
	runRepo *repository.AgentRunRepository,
	riverClient *river.Client[pgx.Tx],
	pool *pgxpool.Pool,
) *ClauserService {
	return &ClauserService{
		clauserRepo: clauserRepo,
		outputRepo:  outputRepo,
		runRepo:     runRepo,
		riverClient: riverClient,
		pool:        pool,
	}
}

// Create creates a new clauser
func (s *ClauserService) Create(ctx context.Context, userID string, title *string) (*ClauserResponse, error) {
	clauser, err := s.clauserRepo.Create(ctx, repository.CreateClauserInput{
		UserID: userID,
		Title:  title,
	})
	if err != nil {
		return nil, middleware.NewInternalError("Failed to create clauser", err)
	}
	return toClauserResponse(clauser), nil
}

// GetByID gets a clauser by ID, verifying ownership
func (s *ClauserService) GetByID(ctx context.Context, clauserID, userID string) (*ClauserResponse, error) {
	clauser, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}
	return toClauserResponse(clauser), nil
}

// List lists clausers for a user
func (s *ClauserService) List(ctx context.Context, userID string, limit, offset int) ([]ClauserListItem, int, error) {
	clausers, err := s.clauserRepo.FindByUserID(ctx, userID, repository.ListClausersFilters{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, middleware.NewInternalError("Failed to fetch clausers", err)
	}

	total, err := s.clauserRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, middleware.NewInternalError("Failed to count clausers", err)
	}

	items := make([]ClauserListItem, len(clausers))
	for i, c := range clausers {
		hasClauseC := len(c.ClauseCHistory) > 2 // "[]" is empty
		items[i] = ClauserListItem{
			ClauserID:  c.ClauserID,
			Title:      c.Title,
			HasClauseC: hasClauseC,
			CreatedAt:  c.CreatedAt,
			UpdatedAt:  c.UpdatedAt,
		}
	}

	return items, total, nil
}

// Delete deletes a clauser
func (s *ClauserService) Delete(ctx context.Context, clauserID, userID string) error {
	// Verify ownership
	_, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return middleware.NewNotFoundError("Clauser not found")
	}

	if err := s.clauserRepo.Delete(ctx, clauserID); err != nil {
		return middleware.NewInternalError("Failed to delete clauser", err)
	}
	return nil
}

// UpdateField updates a single field on a clauser
func (s *ClauserService) UpdateField(ctx context.Context, clauserID, userID, fieldName, value string) (*ClauserResponse, error) {
	// Verify ownership
	_, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	clauser, err := s.clauserRepo.UpdateField(ctx, clauserID, fieldName, value)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to update clauser", err)
	}
	return toClauserResponse(clauser), nil
}

// ClearRun cancels any active run and clears the agent_run_id on the clauser
func (s *ClauserService) ClearRun(ctx context.Context, clauserID, userID string) (*ClauserResponse, error) {
	// Verify ownership and get clauser
	clauser, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	// If there's an active run, mark it as cancelled
	if clauser.AgentRunID != nil {
		run, err := s.runRepo.FindByID(ctx, *clauser.AgentRunID)
		if err == nil {
			// Only cancel if still pending or running
			if run.Status == repository.AgentRunStatusPending || run.Status == repository.AgentRunStatusRunning {
				now := time.Now()
				_, _ = s.runRepo.Update(ctx, run.AgentRunID, repository.UpdateAgentRunInput{
					Status:      ptr(repository.AgentRunStatusCancelled),
					CompletedAt: &now,
				})
			}
		}
	}

	// Clear agent_run_id on clauser
	clauser, err = s.clauserRepo.SetAgentRunID(ctx, clauserID, nil)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to clear run", err)
	}

	return toClauserResponse(clauser), nil
}

// AppendClauseC appends a new clause C version to the history
func (s *ClauserService) AppendClauseC(ctx context.Context, clauserID, userID, text string) (*ClauserResponse, error) {
	// Verify ownership
	_, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	entry := map[string]interface{}{
		"text":      text,
		"createdAt": time.Now().Format(time.RFC3339),
	}

	clauser, err := s.clauserRepo.AppendClauseCHistory(ctx, clauserID, entry)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to append clause C", err)
	}
	return toClauserResponse(clauser), nil
}

// GetScreenState returns the full screen state for a clauser
func (s *ClauserService) GetScreenState(ctx context.Context, clauserID, userID string) (*ScreenStateResponse, error) {
	// Get clauser
	clauser, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	// Get outputs (excluding favorites)
	outputs, err := s.outputRepo.FindByClauserIDExcludingFavorites(ctx, clauserID)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to fetch outputs", err)
	}

	// Get favorites
	favorites, err := s.outputRepo.FindFavorites(ctx, clauserID)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to fetch favorites", err)
	}

	// Get active run status if there is one
	var activeRun *ActiveRunResponse
	if clauser.AgentRunID != nil {
		run, err := s.runRepo.FindByID(ctx, *clauser.AgentRunID)
		if err == nil {
			activeRun = &ActiveRunResponse{
				RunID:       run.AgentRunID,
				Status:      string(run.Status),
				CurrentStep: run.CurrentStep,
				TotalSteps:  run.TotalSteps,
			}
		}
	}

	// Build response
	outputResponses := make([]ClauserOutputResponse, len(outputs))
	for i, o := range outputs {
		outputResponses[i] = toClauserOutputResponse(&o)
	}

	var favoritesResponse *ClauserOutputResponse
	if favorites != nil {
		resp := toClauserOutputResponse(favorites)
		favoritesResponse = &resp
	}

	return &ScreenStateResponse{
		Clauser:   *toClauserResponse(clauser),
		Outputs:   outputResponses,
		Favorites: favoritesResponse,
		ActiveRun: activeRun,
	}, nil
}

// RunLenses triggers a lens analysis job
func (s *ClauserService) RunLenses(ctx context.Context, clauserID, userID string, lenses []string) (*RunJobResponse, error) {
	// Verify ownership and get clauser
	clauser, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	// Check if there's already an active run
	if clauser.AgentRunID != nil {
		return nil, middleware.NewConflictError("A job is already running for this clauser")
	}

	// Validate lenses
	if len(lenses) == 0 {
		return nil, middleware.NewBadRequestError("At least one lens must be specified")
	}

	// Create agent run
	inputPayload, _ := json.Marshal(map[string]interface{}{
		"clauserId": clauserID,
		"lenses":    lenses,
	})
	run, err := s.runRepo.Create(ctx, repository.CreateAgentRunInput{
		UserID:       userID,
		AgentType:    "clauser_lens",
		InputPayload: inputPayload,
	})
	if err != nil {
		return nil, middleware.NewInternalError("Failed to create agent run", err)
	}

	// Set agent_run_id on clauser
	_, err = s.clauserRepo.SetAgentRunID(ctx, clauserID, &run.AgentRunID)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to update clauser", err)
	}

	// Enqueue job
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to start transaction", err)
	}
	defer tx.Rollback(ctx)

	insertRes, err := s.riverClient.InsertTx(ctx, tx, worker.ClauserLensArgs{
		ClauserID:  clauserID,
		AgentRunID: run.AgentRunID,
		Lenses:     lenses,
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

	return &RunJobResponse{
		RunID:   run.AgentRunID,
		Status:  string(run.Status),
		PollURL: "/api/v1/agents/runs/" + run.AgentRunID,
	}, nil
}

// Rewrite triggers a clause rewrite job
func (s *ClauserService) Rewrite(ctx context.Context, clauserID, userID string, instructions string) (*RunJobResponse, error) {
	// Verify ownership and get clauser
	clauser, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	// Check if there's already an active run
	if clauser.AgentRunID != nil {
		return nil, middleware.NewConflictError("A job is already running for this clauser")
	}

	// Create agent run
	inputPayload, _ := json.Marshal(map[string]interface{}{
		"clauserId":    clauserID,
		"instructions": instructions,
	})
	run, err := s.runRepo.Create(ctx, repository.CreateAgentRunInput{
		UserID:       userID,
		AgentType:    "clauser_write",
		InputPayload: inputPayload,
	})
	if err != nil {
		return nil, middleware.NewInternalError("Failed to create agent run", err)
	}

	// Set agent_run_id on clauser
	_, err = s.clauserRepo.SetAgentRunID(ctx, clauserID, &run.AgentRunID)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to update clauser", err)
	}

	// Enqueue job
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to start transaction", err)
	}
	defer tx.Rollback(ctx)

	insertRes, err := s.riverClient.InsertTx(ctx, tx, worker.ClauserWriteArgs{
		ClauserID:    clauserID,
		AgentRunID:   run.AgentRunID,
		Instructions: instructions,
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

	return &RunJobResponse{
		RunID:   run.AgentRunID,
		Status:  string(run.Status),
		PollURL: "/api/v1/agents/runs/" + run.AgentRunID,
	}, nil
}

// AddToFavorites adds an item to favorites by searching for itemId in the output content
func (s *ClauserService) AddToFavorites(ctx context.Context, clauserID, userID, outputID, itemID string) (*ClauserOutputResponse, error) {
	// Verify ownership
	_, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	// Get the source output
	output, err := s.outputRepo.FindByID(ctx, outputID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Output not found")
	}

	// Verify output belongs to this clauser
	if output.ClauserID != clauserID {
		return nil, middleware.NewNotFoundError("Output not found")
	}

	// Parse the output content and search for the item
	var content map[string]interface{}
	if err := json.Unmarshal(output.Content, &content); err != nil {
		return nil, middleware.NewInternalError("Failed to parse output content", err)
	}

	// Search for item with matching itemId
	item, lensName, found := findItemByID(content, itemID)
	if !found {
		return nil, middleware.NewNotFoundError("Item not found")
	}

	// Create favorite object - copy item data without the itemId
	itemData := make(map[string]interface{})
	for k, v := range item {
		if k != "itemId" {
			itemData[k] = v
		}
	}

	favoriteItem := map[string]interface{}{
		"itemId":         itemID,
		"sourceOutputId": outputID,
		"lens":           lensName,
		"data":           itemData,
	}

	// Add to favorites
	favorites, err := s.outputRepo.AppendFavorite(ctx, clauserID, favoriteItem)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to add to favorites", err)
	}

	resp := toClauserOutputResponse(favorites)
	return &resp, nil
}

// findItemByID searches for an item with the given itemId in lens output content
// Returns the item, the lens name, and whether it was found
func findItemByID(content map[string]interface{}, targetItemID string) (map[string]interface{}, string, bool) {
	for lensName, lensData := range content {
		if lensName == "terminator" {
			continue
		}

		switch v := lensData.(type) {
		case []interface{}:
			// Direct array of items (e.g., differenceSummary, deltaResolutionMap)
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemMap["itemId"] == targetItemID {
						return itemMap, lensName, true
					}
				}
			}
		case map[string]interface{}:
			// Object with nested arrays (e.g., frictionForecast with objections)
			for _, val := range v {
				if arr, ok := val.([]interface{}); ok {
					for _, item := range arr {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if itemMap["itemId"] == targetItemID {
								return itemMap, lensName, true
							}
						}
					}
				}
			}
		}
	}
	return nil, "", false
}

// RemoveFromFavorites removes an item from favorites
func (s *ClauserService) RemoveFromFavorites(ctx context.Context, clauserID, userID string, index int) (*ClauserOutputResponse, error) {
	// Verify ownership
	_, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	favorites, err := s.outputRepo.RemoveFavorite(ctx, clauserID, index)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to remove from favorites", err)
	}

	resp := toClauserOutputResponse(favorites)
	return &resp, nil
}

// GetOutputs returns all outputs for a clauser
func (s *ClauserService) GetOutputs(ctx context.Context, clauserID, userID string) ([]ClauserOutputResponse, error) {
	// Verify ownership
	_, err := s.clauserRepo.FindByIDAndUserID(ctx, clauserID, userID)
	if err != nil {
		return nil, middleware.NewNotFoundError("Clauser not found")
	}

	outputs, err := s.outputRepo.FindByClauserID(ctx, clauserID)
	if err != nil {
		return nil, middleware.NewInternalError("Failed to fetch outputs", err)
	}

	responses := make([]ClauserOutputResponse, len(outputs))
	for i, o := range outputs {
		responses[i] = toClauserOutputResponse(&o)
	}
	return responses, nil
}

func toClauserResponse(c *repository.Clauser) *ClauserResponse {
	return &ClauserResponse{
		ClauserID:             c.ClauserID,
		UserID:                c.UserID,
		Title:                 c.Title,
		AgreementA:            c.AgreementA,
		AgreementB:            c.AgreementB,
		ClauseA:               c.ClauseA,
		ClauseB:               c.ClauseB,
		RepresentedParty:      c.RepresentedParty,
		DraftingApproach:      c.DraftingApproach,
		Playbook:              c.Playbook,
		CounterpartyRationale: c.CounterpartyRationale,
		BusinessContext:       c.BusinessContext,
		AgentRunID:            c.AgentRunID,
		ClauseCHistory:        c.ClauseCHistory,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}

func toClauserOutputResponse(o *repository.ClauserOutput) ClauserOutputResponse {
	return ClauserOutputResponse{
		ClauserOutputID: o.ClauserOutputID,
		AgentRunID:      o.AgentRunID,
		Ordinal:         o.Ordinal,
		GroupName:       o.GroupName,
		GroupOrdinal:    o.GroupOrdinal,
		Title:           o.Title,
		Kind:            o.Kind,
		Content:         o.Content,
		CreatedAt:       o.CreatedAt,
	}
}

func ptr[T any](v T) *T {
	return &v
}
