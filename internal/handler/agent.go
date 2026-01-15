package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/service"
)

type AgentHandler struct {
	agentService *service.AgentService
}

func NewAgentHandler(agentService *service.AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

type launchRunRequest struct {
	Input json.RawMessage `json:"input"`
}

func (h *AgentHandler) LaunchRun(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	agentType := c.Param("agentType")

	var req launchRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body - input is optional
		req.Input = json.RawMessage("{}")
	}

	result, err := h.agentService.LaunchRun(c.Request.Context(), authUser.UserID, agentType, req.Input)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *AgentHandler) GetRunStatus(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	runID := c.Param("runId")

	includeMessages := c.Query("includeMessages") == "true"

	var messagesSince *time.Time
	if sinceStr := c.Query("messagesSince"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			messagesSince = &t
		}
	}

	run, messages, err := h.agentService.GetRunStatus(c.Request.Context(), runID, authUser.UserID, includeMessages, messagesSince)
	if err != nil {
		c.Error(err)
		return
	}

	response := gin.H{
		"status": "success",
		"data": gin.H{
			"runId":       run.AgentRunID,
			"agentType":   run.AgentType,
			"status":      run.Status,
			"currentStep": run.CurrentStep,
			"totalSteps":  run.TotalSteps,
			"input":       run.InputPayload,
			"output":      run.OutputPayload,
			"error":       run.ErrorMessage,
			"createdAt":   run.CreatedAt,
			"startedAt":   run.StartedAt,
			"completedAt": run.CompletedAt,
		},
	}

	if includeMessages {
		response["data"].(gin.H)["messages"] = messages
	}

	c.JSON(http.StatusOK, response)
}

func (h *AgentHandler) CancelRun(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	runID := c.Param("runId")

	run, err := h.agentService.CancelRun(c.Request.Context(), runID, authUser.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"runId":   run.AgentRunID,
			"status":  run.Status,
			"message": "Cancellation requested. The run will be cancelled shortly if still in progress.",
		},
	})
}

func (h *AgentHandler) ListRuns(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)

	// Parse query params
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

	filters := repository.ListAgentRunsFilters{
		Limit:  limit,
		Offset: offset,
	}

	if status := c.Query("status"); status != "" {
		s := repository.AgentRunStatus(status)
		filters.Status = &s
	}

	if agentType := c.Query("agentType"); agentType != "" {
		filters.AgentType = &agentType
	}

	runs, total, err := h.agentService.ListRuns(c.Request.Context(), authUser.UserID, filters)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"runs": runs,
			"pagination": gin.H{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

// RegisterRoutes registers agent routes (all protected)
func (h *AgentHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	agents := rg.Group("/agents")
	agents.Use(authMiddleware)
	{
		agents.POST("/:agentType/run", h.LaunchRun)
		agents.GET("/runs", h.ListRuns)
		agents.GET("/runs/:runId", h.GetRunStatus)
		agents.POST("/runs/:runId/cancel", h.CancelRun)
	}
}
