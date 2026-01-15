package agent

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
)

// StepType represents the type of step
type StepType string

const (
	StepTypeLLM         StepType = "llm"
	StepTypeCode        StepType = "code"
	StepTypeExternalAPI StepType = "external_api"
)

// StepResult represents the result of executing a step
type StepResult struct {
	Output   interface{}   `json:"output"`
	Metadata *StepMetadata `json:"metadata,omitempty"`
}

// StepMetadata contains metadata about step execution
type StepMetadata struct {
	Model        string `json:"model,omitempty"`
	InputTokens  int64  `json:"inputTokens,omitempty"`
	OutputTokens int64  `json:"outputTokens,omitempty"`
}

// Step represents a single step in an agent's workflow
type Step struct {
	Name           string
	Type           StepType
	Execute        func(ctx context.Context, input interface{}, agentCtx *Context) (*StepResult, error)
	TransformInput func(previousOutput interface{}) interface{}
	ShouldSkip     func(previousOutput interface{}, agentCtx *Context) bool
}

// Context provides the execution context for an agent
type Context struct {
	RunID       string
	UserID      string
	Input       map[string]interface{}
	CurrentStep int
	Anthropic   *anthropic.Client
	StepOutputs map[string]interface{}
	Log         func(ctx context.Context, message string, level string, details json.RawMessage) error
	CheckCancel func(ctx context.Context) (bool, error)
}

// Agent is the interface that all agents must implement
type Agent interface {
	// DefineSteps returns the list of steps this agent will execute
	DefineSteps() []Step
	// Initialize is called before step execution begins
	Initialize(ctx context.Context) error
	// Cleanup is called after step execution completes (success or failure)
	Cleanup(ctx context.Context) error
}

// BaseAgent provides a default implementation that can be embedded
type BaseAgent struct {
	Ctx *Context
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(agentCtx *Context) BaseAgent {
	return BaseAgent{Ctx: agentCtx}
}

// Initialize is a no-op by default
func (b *BaseAgent) Initialize(ctx context.Context) error {
	return nil
}

// Cleanup is a no-op by default
func (b *BaseAgent) Cleanup(ctx context.Context) error {
	return nil
}
