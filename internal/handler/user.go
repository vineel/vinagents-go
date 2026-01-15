package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/service"
	"github.com/vineel/vinagents-go/pkg/jwt"
)

type UserHandler struct {
	userRepo   *repository.UserRepository
	jwtManager *jwt.Manager
}

func NewUserHandler(userRepo *repository.UserRepository, jwtManager *jwt.Manager) *UserHandler {
	return &UserHandler{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)

	user, err := h.userRepo.FindByID(c.Request.Context(), authUser.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": service.UserResponse{
			UserID:    user.UserID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	users, err := h.userRepo.FindAll(c.Request.Context(), limit, offset)
	if err != nil {
		c.Error(middleware.NewInternalError("Failed to fetch users", err))
		return
	}

	// Sanitize users (remove passwords)
	sanitizedUsers := make([]service.UserResponse, len(users))
	for i, user := range users {
		sanitizedUsers[i] = service.UserResponse{
			UserID:    user.UserID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   sanitizedUsers,
	})
}

// RegisterRoutes registers user routes (all protected)
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	users := rg.Group("/users")
	users.Use(authMiddleware)
	{
		users.GET("/me", h.GetMe)
		users.GET("", h.ListUsers)
	}
}
