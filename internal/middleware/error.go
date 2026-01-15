package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vineel/vinagents-go/internal/repository"
)

// AppError represents an application error with status code
type AppError struct {
	StatusCode int
	Message    string
	Internal   error
}

func (e *AppError) Error() string {
	return e.Message
}

// Common error constructors
func NewBadRequestError(message string) *AppError {
	return &AppError{StatusCode: http.StatusBadRequest, Message: message}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{StatusCode: http.StatusUnauthorized, Message: message}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{StatusCode: http.StatusNotFound, Message: message}
}

func NewConflictError(message string) *AppError {
	return &AppError{StatusCode: http.StatusConflict, Message: message}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{StatusCode: http.StatusInternalServerError, Message: message, Internal: err}
}

// ErrorHandler handles errors and converts them to JSON responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Check for AppError
			var appErr *AppError
			if errors.As(err, &appErr) {
				if appErr.Internal != nil {
					slog.Error("internal error",
						"message", appErr.Message,
						"error", appErr.Internal,
						"path", c.Request.URL.Path,
					)
				}
				c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
				return
			}

			// Check for known repository errors
			if errors.Is(err, repository.ErrUserNotFound) ||
				errors.Is(err, repository.ErrAgentRunNotFound) ||
				errors.Is(err, repository.ErrRefreshTokenNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
				return
			}

			// Unknown error - log it and return generic message
			slog.Error("unhandled error",
				"error", err,
				"path", c.Request.URL.Path,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
	}
}
