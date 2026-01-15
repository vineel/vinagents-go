package repository

import (
	"encoding/json"
	"time"
)

// AgentRunStatus represents the status of an agent run
type AgentRunStatus string

const (
	AgentRunStatusPending         AgentRunStatus = "pending"
	AgentRunStatusRunning         AgentRunStatus = "running"
	AgentRunStatusCompleted       AgentRunStatus = "completed"
	AgentRunStatusFailed          AgentRunStatus = "failed"
	AgentRunStatusCancelled       AgentRunStatus = "cancelled"
	AgentRunStatusCancelRequested AgentRunStatus = "cancel_requested"
)

// MessageLevel represents the severity level of a log message
type MessageLevel string

const (
	MessageLevelDebug MessageLevel = "debug"
	MessageLevelInfo  MessageLevel = "info"
	MessageLevelWarn  MessageLevel = "warn"
	MessageLevelError MessageLevel = "error"
)

// User represents a user in the system
type User struct {
	UserID    string     `json:"userId"`
	Email     string     `json:"email"`
	Password  string     `json:"-"` // Never serialize password
	FirstName *string    `json:"firstName"`
	LastName  *string    `json:"lastName"`
	IsActive  bool       `json:"isActive"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	RefreshTokenID string    `json:"refreshTokenId"`
	Token          string    `json:"token"`
	UserID         string    `json:"userId"`
	ExpiresAt      time.Time `json:"expiresAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AgentRun represents an agent execution
type AgentRun struct {
	AgentRunID    string            `json:"agentRunId"`
	UserID        string            `json:"userId"`
	AgentType     string            `json:"agentType"`
	AgentVersion  string            `json:"agentVersion"`
	InputPayload  json.RawMessage   `json:"inputPayload"`
	OutputPayload json.RawMessage   `json:"outputPayload,omitempty"`
	Status        AgentRunStatus    `json:"status"`
	CurrentStep   int               `json:"currentStep"`
	TotalSteps    *int              `json:"totalSteps"`
	ErrorMessage  *string           `json:"errorMessage,omitempty"`
	ErrorDetails  json.RawMessage   `json:"errorDetails,omitempty"`
	RetryCount    int               `json:"retryCount"`
	MaxRetries    int               `json:"maxRetries"`
	CreatedAt     time.Time         `json:"createdAt"`
	StartedAt     *time.Time        `json:"startedAt,omitempty"`
	CompletedAt   *time.Time        `json:"completedAt,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	RiverJobID    *int64            `json:"riverJobId,omitempty"`
}

// AgentRunMessage represents a log message for an agent run
type AgentRunMessage struct {
	AgentMessageID string          `json:"agentMessageId"`
	AgentRunID     string          `json:"agentRunId"`
	StepNumber     *int            `json:"stepNumber,omitempty"`
	Level          MessageLevel    `json:"level"`
	Message        string          `json:"message"`
	Details        json.RawMessage `json:"details,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// CreateUserInput is the input for creating a new user
type CreateUserInput struct {
	Email     string
	Password  string // Already hashed
	FirstName *string
	LastName  *string
}

// UpdateUserInput is the input for updating a user
type UpdateUserInput struct {
	Email     *string
	Password  *string
	FirstName *string
	LastName  *string
	IsActive  *bool
}

// CreateRefreshTokenInput is the input for creating a refresh token
type CreateRefreshTokenInput struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

// CreateAgentRunInput is the input for creating an agent run
type CreateAgentRunInput struct {
	UserID       string
	AgentType    string
	InputPayload json.RawMessage
}

// UpdateAgentRunInput is the input for updating an agent run
type UpdateAgentRunInput struct {
	Status        *AgentRunStatus
	CurrentStep   *int
	TotalSteps    *int
	OutputPayload json.RawMessage
	ErrorMessage  *string
	ErrorDetails  json.RawMessage
	StartedAt     *time.Time
	CompletedAt   *time.Time
	RiverJobID    *int64
	RetryCount    *int
}

// CreateAgentRunMessageInput is the input for creating a log message
type CreateAgentRunMessageInput struct {
	AgentRunID string
	StepNumber *int
	Level      MessageLevel
	Message    string
	Details    json.RawMessage
}

// ListAgentRunsFilters filters for listing agent runs
type ListAgentRunsFilters struct {
	Status    *AgentRunStatus
	AgentType *string
	Limit     int
	Offset    int
}
