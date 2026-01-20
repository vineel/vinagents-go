package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/riverqueue/river"

	"github.com/vineel/vinagents-go/internal/prompt"
	"github.com/vineel/vinagents-go/internal/repository"
)

// ClauserWriteWorker handles clause writing jobs
type ClauserWriteWorker struct {
	river.WorkerDefaults[ClauserWriteArgs]
	clauserRepo     *repository.ClauserRepository
	outputRepo      *repository.ClauserOutputRepository
	runRepo         *repository.AgentRunRepository
	messageRepo     *repository.AgentRunMessageRepository
	anthropicClient *anthropic.Client
	promptLoader    *prompt.Loader
}

// NewClauserWriteWorker creates a new clause write worker
func NewClauserWriteWorker(
	clauserRepo *repository.ClauserRepository,
	outputRepo *repository.ClauserOutputRepository,
	runRepo *repository.AgentRunRepository,
	messageRepo *repository.AgentRunMessageRepository,
	anthropicClient *anthropic.Client,
	promptLoader *prompt.Loader,
) *ClauserWriteWorker {
	return &ClauserWriteWorker{
		clauserRepo:     clauserRepo,
		outputRepo:      outputRepo,
		runRepo:         runRepo,
		messageRepo:     messageRepo,
		anthropicClient: anthropicClient,
		promptLoader:    promptLoader,
	}
}

// Timeout returns the timeout for clause write jobs
func (w *ClauserWriteWorker) Timeout(job *river.Job[ClauserWriteArgs]) time.Duration {
	return 5 * time.Minute
}

// Work executes a clause write job
func (w *ClauserWriteWorker) Work(ctx context.Context, job *river.Job[ClauserWriteArgs]) error {
	slog.Info("starting clauser write job", "clauserId", job.Args.ClauserID, "agentRunId", job.Args.AgentRunID)

	// Mark run as started
	now := time.Now()
	_, err := w.runRepo.Update(ctx, job.Args.AgentRunID, repository.UpdateAgentRunInput{
		Status:    ptr(repository.AgentRunStatusRunning),
		StartedAt: &now,
	})
	if err != nil {
		slog.Error("failed to update run status", "error", err)
		return nil
	}

	// Log start
	w.logMessage(ctx, job.Args.AgentRunID, "info", "Starting clause write job", nil)

	// Execute the work
	if err := w.execute(ctx, job.Args); err != nil {
		slog.Error("clauser write job failed", "error", err)
		w.markFailed(ctx, job.Args.AgentRunID, err)
		return nil
	}

	// Mark run as completed
	completedAt := time.Now()
	_, err = w.runRepo.Update(ctx, job.Args.AgentRunID, repository.UpdateAgentRunInput{
		Status:      ptr(repository.AgentRunStatusCompleted),
		CompletedAt: &completedAt,
	})
	if err != nil {
		slog.Error("failed to mark run as completed", "error", err)
	}

	// Clear agent_run_id on clauser
	_, err = w.clauserRepo.SetAgentRunID(ctx, job.Args.ClauserID, nil)
	if err != nil {
		slog.Error("failed to clear agent_run_id on clauser", "error", err)
	}

	slog.Info("clauser write job completed", "clauserId", job.Args.ClauserID)
	return nil
}

func (w *ClauserWriteWorker) execute(ctx context.Context, args ClauserWriteArgs) error {
	// Load clauser
	clauser, err := w.clauserRepo.FindByID(ctx, args.ClauserID)
	if err != nil {
		return fmt.Errorf("failed to load clauser: %w", err)
	}

	// Load favorites
	favorites, err := w.loadFavorites(ctx, args.ClauserID)
	if err != nil {
		return fmt.Errorf("failed to load favorites: %w", err)
	}

	w.logMessage(ctx, args.AgentRunID, "info", fmt.Sprintf("Loaded %d favorites", len(favorites)), nil)

	// Build prompt data
	data := prompt.ClauserWriteData{
		AgreementA:            deref(clauser.AgreementA),
		AgreementB:            deref(clauser.AgreementB),
		ClauseA:               deref(clauser.ClauseA),
		ClauseB:               deref(clauser.ClauseB),
		RepresentedParty:      deref(clauser.RepresentedParty),
		DraftingApproach:      deref(clauser.DraftingApproach),
		Playbook:              deref(clauser.Playbook),
		CounterpartyRationale: deref(clauser.CounterpartyRationale),
		BusinessContext:       deref(clauser.BusinessContext),
		Favorites:             favorites,
		Instructions:          args.Instructions,
	}

	// Execute prompt template
	promptText, err := w.promptLoader.Execute("clauser-write-clause", data)
	if err != nil {
		return fmt.Errorf("failed to execute prompt template: %w", err)
	}

	// Dump hydrated prompt to file for debugging
	timestamp := time.Now().Format("20060102_150405")
	promptFilename := fmt.Sprintf("write_%s_1_prompt.txt", timestamp)
	promptPath := filepath.Join("output", promptFilename)
	if err := os.WriteFile(promptPath, []byte(promptText), 0644); err != nil {
		slog.Warn("failed to dump prompt to file", "error", err, "path", promptPath)
	} else {
		slog.Info("dumped hydrated prompt", "path", promptPath)
	}

	w.logMessage(ctx, args.AgentRunID, "info", "Calling Claude API", nil)

	// Call Claude
	response, err := w.anthropicClient.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptText)),
		},
	})
	if err != nil {
		return fmt.Errorf("Claude API call failed: %w", err)
	}

	// Extract clause text from response
	var clauseText string
	for _, block := range response.Content {
		if block.Type == "text" {
			clauseText = block.Text
			break
		}
	}

	// Parse JSON response and extract clause_c
	jsonText := extractJSON(clauseText)
	var parsed struct {
		ClauseC string `json:"clause_c"`
	}
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return fmt.Errorf("failed to parse clause response JSON: %w", err)
	}
	clauseText = parsed.ClauseC

	// Dump raw response to file for debugging
	responseFilename := fmt.Sprintf("write_%s_2_response.txt", timestamp)
	responsePath := filepath.Join("output", responseFilename)
	if err := os.WriteFile(responsePath, []byte(clauseText), 0644); err != nil {
		slog.Warn("failed to dump response to file", "error", err, "path", responsePath)
	} else {
		slog.Info("dumped Claude response", "path", responsePath)
	}

	// Log token usage
	details, _ := json.Marshal(map[string]interface{}{
		"inputTokens":  response.Usage.InputTokens,
		"outputTokens": response.Usage.OutputTokens,
		"model":        response.Model,
	})
	w.logMessage(ctx, args.AgentRunID, "info", "Claude API call completed", details)

	// Append to clause_c_history
	historyEntry := map[string]interface{}{
		"text":       clauseText,
		"createdAt":  time.Now().Format(time.RFC3339),
		"agentRunId": args.AgentRunID,
	}
	_, err = w.clauserRepo.AppendClauseCHistory(ctx, args.ClauserID, historyEntry)
	if err != nil {
		return fmt.Errorf("failed to append to clause_c_history: %w", err)
	}

	// Get next ordinal and create output row
	ordinal, err := w.outputRepo.GetNextOrdinal(ctx, args.ClauserID)
	if err != nil {
		return fmt.Errorf("failed to get next ordinal: %w", err)
	}

	content, _ := json.Marshal(map[string]interface{}{
		"text": clauseText,
	})

	_, err = w.outputRepo.Create(ctx, repository.CreateClauserOutputInput{
		ClauserID:    args.ClauserID,
		AgentRunID:   &args.AgentRunID,
		Ordinal:      ordinal,
		GroupName:    "clause_c",
		GroupOrdinal: 0,
		Title:        "Generated Clause",
		Kind:         "clause_draft",
		Content:      content,
	})
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}

	w.logMessage(ctx, args.AgentRunID, "info", "Clause C generated and saved", nil)

	return nil
}

func (w *ClauserWriteWorker) loadFavorites(ctx context.Context, clauserID string) ([]prompt.FavoriteItem, error) {
	favoritesRow, err := w.outputRepo.FindFavorites(ctx, clauserID)
	if err != nil {
		return nil, err
	}
	if favoritesRow == nil {
		return []prompt.FavoriteItem{}, nil
	}

	var content map[string]interface{}
	if err := json.Unmarshal(favoritesRow.Content, &content); err != nil {
		return nil, err
	}

	items, ok := content["items"].([]interface{})
	if !ok {
		return []prompt.FavoriteItem{}, nil
	}

	favorites := make([]prompt.FavoriteItem, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := itemMap["title"].(string)
		body, _ := itemMap["body"].(string)
		favorites = append(favorites, prompt.FavoriteItem{
			Title: title,
			Body:  body,
		})
	}

	return favorites, nil
}

func (w *ClauserWriteWorker) markFailed(ctx context.Context, agentRunID string, err error) {
	errMsg := err.Error()
	completedAt := time.Now()
	_, updateErr := w.runRepo.Update(ctx, agentRunID, repository.UpdateAgentRunInput{
		Status:       ptr(repository.AgentRunStatusFailed),
		ErrorMessage: &errMsg,
		CompletedAt:  &completedAt,
	})
	if updateErr != nil {
		slog.Error("failed to mark run as failed", "error", updateErr)
	}

	w.logMessage(ctx, agentRunID, "error", err.Error(), nil)
}

func (w *ClauserWriteWorker) logMessage(ctx context.Context, agentRunID, level, message string, details json.RawMessage) {
	msgLevel := repository.MessageLevelInfo
	switch level {
	case "debug":
		msgLevel = repository.MessageLevelDebug
	case "warn":
		msgLevel = repository.MessageLevelWarn
	case "error":
		msgLevel = repository.MessageLevelError
	}

	_, err := w.messageRepo.Create(ctx, repository.CreateAgentRunMessageInput{
		AgentRunID: agentRunID,
		Level:      msgLevel,
		Message:    message,
		Details:    details,
	})
	if err != nil {
		slog.Error("failed to create log message", "error", err)
	}
}

