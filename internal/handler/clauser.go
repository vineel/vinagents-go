package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/service"
)

type ClauserHandler struct {
	clauserService *service.ClauserService
}

func NewClauserHandler(clauserService *service.ClauserService) *ClauserHandler {
	return &ClauserHandler{clauserService: clauserService}
}

// Request types

type createClauserRequest struct {
	Title *string `json:"title"`
}

type updateFieldRequest struct {
	Value string `json:"value" binding:"required"`
}

type runLensesRequest struct {
	Lenses []string `json:"lenses" binding:"required,min=1"`
}

type rewriteRequest struct {
	Instructions string `json:"instructions"`
}

type addFavoriteRequest struct {
	ItemID string `json:"itemId" binding:"required"`
}

type appendClauseCRequest struct {
	Text string `json:"text" binding:"required"`
}

// Handlers

func (h *ClauserHandler) Create(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)

	var req createClauserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
	}

	result, err := h.clauserService.Create(c.Request.Context(), authUser.UserID, req.Title)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) Get(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	result, err := h.clauserService.GetByID(c.Request.Context(), clauserID, authUser.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) List(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)

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

	clausers, total, err := h.clauserService.List(c.Request.Context(), authUser.UserID, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"clausers": clausers,
			"pagination": gin.H{
				"total":  total,
				"limit":  limit,
				"offset": offset,
			},
		},
	})
}

func (h *ClauserHandler) Delete(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	if err := h.clauserService.Delete(c.Request.Context(), clauserID, authUser.UserID); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Clauser deleted",
	})
}

func (h *ClauserHandler) UpdateAgreementA(c *gin.Context) {
	h.updateField(c, "agreement_a")
}

func (h *ClauserHandler) UpdateAgreementB(c *gin.Context) {
	h.updateField(c, "agreement_b")
}

func (h *ClauserHandler) UpdateClauseA(c *gin.Context) {
	h.updateField(c, "clause_a")
}

func (h *ClauserHandler) UpdateClauseB(c *gin.Context) {
	h.updateField(c, "clause_b")
}

func (h *ClauserHandler) UpdateTitle(c *gin.Context) {
	h.updateField(c, "title")
}

func (h *ClauserHandler) UpdateRepresentedParty(c *gin.Context) {
	h.updateField(c, "represented_party")
}

func (h *ClauserHandler) UpdateDraftingApproach(c *gin.Context) {
	h.updateField(c, "drafting_approach")
}

func (h *ClauserHandler) UpdatePlaybook(c *gin.Context) {
	h.updateField(c, "playbook")
}

func (h *ClauserHandler) UpdateCounterpartyRationale(c *gin.Context) {
	h.updateField(c, "counterparty_rationale")
}

func (h *ClauserHandler) UpdateBusinessContext(c *gin.Context) {
	h.updateField(c, "business_context")
}

func (h *ClauserHandler) updateField(c *gin.Context, fieldName string) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	var req updateFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(middleware.NewBadRequestError("Invalid request body: " + err.Error()))
		return
	}

	result, err := h.clauserService.UpdateField(c.Request.Context(), clauserID, authUser.UserID, fieldName, req.Value)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) GetScreen(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	result, err := h.clauserService.GetScreenState(c.Request.Context(), clauserID, authUser.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) RunLenses(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	var req runLensesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(middleware.NewBadRequestError("Invalid request body: " + err.Error()))
		return
	}

	result, err := h.clauserService.RunLenses(c.Request.Context(), clauserID, authUser.UserID, req.Lenses)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) Rewrite(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	var req rewriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body - instructions are optional
	}

	result, err := h.clauserService.Rewrite(c.Request.Context(), clauserID, authUser.UserID, req.Instructions)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) AddFavorite(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")
	outputID := c.Param("outputId")

	var req addFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(middleware.NewBadRequestError("Invalid request body: " + err.Error()))
		return
	}

	result, err := h.clauserService.AddToFavorites(c.Request.Context(), clauserID, authUser.UserID, outputID, req.ItemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) RemoveFavorite(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")
	itemID := c.Param("itemId")

	if itemID == "" {
		c.Error(middleware.NewBadRequestError("Item ID is required"))
		return
	}

	result, err := h.clauserService.RemoveFromFavorites(c.Request.Context(), clauserID, authUser.UserID, itemID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) GetOutputs(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	result, err := h.clauserService.GetOutputs(c.Request.Context(), clauserID, authUser.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) AppendClauseC(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	var req appendClauseCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(middleware.NewBadRequestError("Invalid request body: " + err.Error()))
		return
	}

	result, err := h.clauserService.AppendClauseC(c.Request.Context(), clauserID, authUser.UserID, req.Text)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *ClauserHandler) ClearRun(c *gin.Context) {
	authUser := middleware.MustGetAuthUser(c)
	clauserID := c.Param("id")

	result, err := h.clauserService.ClearRun(c.Request.Context(), clauserID, authUser.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    result,
		"message": "Run cleared",
	})
}

// RegisterRoutes registers clauser routes (all protected)
func (h *ClauserHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	clausers := rg.Group("/clausers")
	clausers.Use(authMiddleware)
	{
		clausers.POST("", h.Create)
		clausers.GET("", h.List)
		clausers.GET("/:id", h.Get)
		clausers.DELETE("/:id", h.Delete)

		clausers.PUT("/:id/agreement-a", h.UpdateAgreementA)
		clausers.PUT("/:id/agreement-b", h.UpdateAgreementB)
		clausers.PUT("/:id/clause-a", h.UpdateClauseA)
		clausers.PUT("/:id/clause-b", h.UpdateClauseB)
		clausers.PUT("/:id/title", h.UpdateTitle)
		clausers.PUT("/:id/represented-party", h.UpdateRepresentedParty)
		clausers.PUT("/:id/drafting-approach", h.UpdateDraftingApproach)
		clausers.PUT("/:id/playbook", h.UpdatePlaybook)
		clausers.PUT("/:id/counterparty-rationale", h.UpdateCounterpartyRationale)
		clausers.PUT("/:id/business-context", h.UpdateBusinessContext)

		clausers.GET("/:id/screen", h.GetScreen)
		clausers.POST("/:id/run-lenses", h.RunLenses)
		clausers.POST("/:id/rewrite", h.Rewrite)
		clausers.POST("/:id/clause-c", h.AppendClauseC)
		clausers.POST("/:id/clear-run", h.ClearRun)

		clausers.GET("/:id/outputs", h.GetOutputs)
		clausers.POST("/:id/outputs/:outputId/favorite", h.AddFavorite)
		clausers.DELETE("/:id/favorites/:itemId", h.RemoveFavorite)
	}
}
