package simple

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/vineel/vinagents-go/internal/agent"
)

// SimpleAgent is a basic agent that makes a single LLM call
type SimpleAgent struct {
	agent.BaseAgent
}

// New creates a new SimpleAgent
func New(ctx *agent.Context) agent.Agent {
	return &SimpleAgent{
		BaseAgent: agent.NewBaseAgent(ctx),
	}
}

// DefineSteps returns the steps for this agent
func (a *SimpleAgent) DefineSteps() []agent.Step {
	return []agent.Step{
		{
			Name: "llm_call",
			Type: agent.StepTypeLLM,
			Execute: func(ctx context.Context, input interface{}, agentCtx *agent.Context) (*agent.StepResult, error) {
				// Extract prompt from input
				inputMap, ok := input.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid input type: expected map")
				}

				prompt, ok := inputMap["prompt"].(string)
				if !ok {
					return nil, fmt.Errorf("missing or invalid 'prompt' field")
				}

				// Log start
				if err := agentCtx.Log(ctx, "Starting Claude API call", "info", nil); err != nil {
					// Non-fatal, continue
				}

				// Call Claude
				response, err := agentCtx.Anthropic.Messages.New(ctx, anthropic.MessageNewParams{
					Model:     anthropic.ModelClaudeSonnet4_20250514,
					MaxTokens: 1024,
					Messages: []anthropic.MessageParam{
						anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
					},
				})
				if err != nil {
					return nil, fmt.Errorf("Claude API call failed: %w", err)
				}

				// Extract text from response
				var outputText string
				for _, block := range response.Content {
					if block.Type == "text" {
						outputText = block.Text
						break
					}
				}

				// Log completion
				details, _ := json.Marshal(map[string]interface{}{
					"inputTokens":  response.Usage.InputTokens,
					"outputTokens": response.Usage.OutputTokens,
				})
				if err := agentCtx.Log(ctx, "Claude API call completed", "info", details); err != nil {
					// Non-fatal, continue
				}

				return &agent.StepResult{
					Output: map[string]interface{}{
						"text": outputText,
					},
					Metadata: &agent.StepMetadata{
						Model:        string(response.Model),
						InputTokens:  response.Usage.InputTokens,
						OutputTokens: response.Usage.OutputTokens,
					},
				}, nil
			},
		},
	}
}

func init() {
	agent.Register("simple", New)
}
