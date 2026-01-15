package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/riverqueue/river"
	"github.com/vineel/vinagents-go/internal/prompt"
	"github.com/vineel/vinagents-go/internal/repository"
)

// ClauserLensWorker handles lens analysis jobs
type ClauserLensWorker struct {
	river.WorkerDefaults[ClauserLensArgs]
	clauserRepo     *repository.ClauserRepository
	outputRepo      *repository.ClauserOutputRepository
	runRepo         *repository.AgentRunRepository
	messageRepo     *repository.AgentRunMessageRepository
	anthropicClient *anthropic.Client
	promptLoader    *prompt.Loader
}

// NewClauserLensWorker creates a new lens analysis worker
func NewClauserLensWorker(
	clauserRepo *repository.ClauserRepository,
	outputRepo *repository.ClauserOutputRepository,
	runRepo *repository.AgentRunRepository,
	messageRepo *repository.AgentRunMessageRepository,
	anthropicClient *anthropic.Client,
	promptLoader *prompt.Loader,
) *ClauserLensWorker {
	return &ClauserLensWorker{
		clauserRepo:     clauserRepo,
		outputRepo:      outputRepo,
		runRepo:         runRepo,
		messageRepo:     messageRepo,
		anthropicClient: anthropicClient,
		promptLoader:    promptLoader,
	}
}

// Work executes a lens analysis job
func (w *ClauserLensWorker) Work(ctx context.Context, job *river.Job[ClauserLensArgs]) error {
	slog.Info("starting clauser lens job", "clauserId", job.Args.ClauserID, "lenses", job.Args.Lenses)

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
	w.logMessage(ctx, job.Args.AgentRunID, "info", fmt.Sprintf("Starting lens analysis for: %v", job.Args.Lenses), nil)

	// Execute the work
	if err := w.execute(ctx, job.Args); err != nil {
		slog.Error("clauser lens job failed", "error", err)
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

	slog.Info("clauser lens job completed", "clauserId", job.Args.ClauserID)
	return nil
}

func (w *ClauserLensWorker) execute(ctx context.Context, args ClauserLensArgs) error {
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
	data := prompt.ClauserLensData{
		AgreementA: deref(clauser.AgreementA),
		AgreementB: deref(clauser.AgreementB),
		ClauseA:    deref(clauser.ClauseA),
		ClauseB:    deref(clauser.ClauseB),
		Favorites:  favorites,
		Lenses:     args.Lenses,
	}

	// Execute prompt template
	promptText, err := w.promptLoader.Execute("clauser-lenses", data)
	if err != nil {
		return fmt.Errorf("failed to execute prompt template: %w", err)
	}

	w.logMessage(ctx, args.AgentRunID, "info", "Calling Claude API for lens analysis", nil)

	// Call Claude
	response, err := w.anthropicClient.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_20250514,
		MaxTokens: 8192,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptText)),
		},
	})
	if err != nil {
		return fmt.Errorf("Claude API call failed: %w", err)
	}

	// Extract response text
	var responseText string
	for _, block := range response.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	// Log token usage
	details, _ := json.Marshal(map[string]interface{}{
		"inputTokens":  response.Usage.InputTokens,
		"outputTokens": response.Usage.OutputTokens,
		"model":        response.Model,
	})
	w.logMessage(ctx, args.AgentRunID, "info", "Claude API call completed", details)

	// Parse the JSON response
	lensResults, err := w.parseLensResponse(responseText)
	if err != nil {
		return fmt.Errorf("failed to parse lens response: %w", err)
	}

	// Create output rows for each lens
	ordinal, err := w.outputRepo.GetNextOrdinal(ctx, args.ClauserID)
	if err != nil {
		return fmt.Errorf("failed to get next ordinal: %w", err)
	}

	for i, lensName := range args.Lenses {
		lensResult, ok := lensResults[lensName]
		if !ok {
			w.logMessage(ctx, args.AgentRunID, "warn", fmt.Sprintf("No results for lens: %s", lensName), nil)
			continue
		}

		content, _ := json.Marshal(lensResult)

		title := lensResult.Title
		if title == "" {
			title = lensName
		}

		_, err = w.outputRepo.Create(ctx, repository.CreateClauserOutputInput{
			ClauserID:    args.ClauserID,
			AgentRunID:   &args.AgentRunID,
			Ordinal:      ordinal + i,
			GroupName:    lensName,
			GroupOrdinal: 0,
			Title:        title,
			Kind:         "lens_output",
			Content:      content,
		})
		if err != nil {
			return fmt.Errorf("failed to create output for lens %s: %w", lensName, err)
		}
	}

	w.logMessage(ctx, args.AgentRunID, "info", fmt.Sprintf("Created %d lens outputs", len(lensResults)), nil)

	return nil
}

type lensResult struct {
	Title string       `json:"title"`
	Items []lensItem   `json:"items"`
}

type lensItem struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Severity string `json:"severity"`
}

func (w *ClauserLensWorker) parseLensResponse(responseText string) (map[string]lensResult, error) {
	// Try to extract JSON from the response (it might have markdown code blocks)
	jsonText := extractJSON(responseText)

	var parsed struct {
		Lenses map[string]lensResult `json:"lenses"`
	}

	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (response: %s)", err, truncate(jsonText, 500))
	}

	return parsed.Lenses, nil
}

func (w *ClauserLensWorker) loadFavorites(ctx context.Context, clauserID string) ([]prompt.FavoriteItem, error) {
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

func (w *ClauserLensWorker) markFailed(ctx context.Context, agentRunID string, err error) {
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

func (w *ClauserLensWorker) logMessage(ctx context.Context, agentRunID, level, message string, details json.RawMessage) {
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

// extractJSON attempts to extract JSON from text that might have markdown code blocks
func extractJSON(text string) string {
	// Look for JSON code block
	start := 0
	if idx := findIndex(text, "```json"); idx >= 0 {
		start = idx + 7
	} else if idx := findIndex(text, "```"); idx >= 0 {
		start = idx + 3
	}

	end := len(text)
	if start > 0 {
		if idx := findIndex(text[start:], "```"); idx >= 0 {
			end = start + idx
		}
	}

	result := text[start:end]

	// Find the first { and last }
	firstBrace := findIndex(result, "{")
	lastBrace := findLastIndex(result, "}")

	if firstBrace >= 0 && lastBrace > firstBrace {
		return result[firstBrace : lastBrace+1]
	}

	return result
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
