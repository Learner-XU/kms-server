package issue

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"kms-server/internal/gitea"
)

type Handler struct {
	client *gitea.Client
}

func NewHandler(client *gitea.Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	issues := r.Group("/issues")
	{
		issues.GET("", h.List)
		issues.POST("", h.Create)
		issues.GET("/labels", h.ListLabels)
		issues.POST("/labels", h.CreateLabel)
		issues.GET("/:index", h.Get)
		issues.PATCH("/:index", h.Update)
		issues.GET("/:index/comments", h.ListComments)
		issues.POST("/:index/comments", h.AddComment)
	}
}

func (h *Handler) List(c *gin.Context) {
	state := c.DefaultQuery("state", "open")
	labels := c.Query("labels")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	issues, err := h.client.ListIssuesDetailed(c.Request.Context(), state, labels, page, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to list issues")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list issues"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"issues": issues})
}

func (h *Handler) Create(c *gin.Context) {
	var req gitea.CreateIssueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	issue, err := h.client.CreateIssueDetailed(c.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to create issue")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create issue"})
		return
	}
	c.JSON(http.StatusCreated, issue)
}

func (h *Handler) Get(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue index"})
		return
	}

	issue, err := h.client.GetIssue(c.Request.Context(), index)
	if err != nil {
		log.Error().Err(err).Int("index", index).Msg("failed to get issue")
		c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
		return
	}
	c.JSON(http.StatusOK, issue)
}

func (h *Handler) Update(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue index"})
		return
	}

	var req gitea.UpdateIssueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	issue, err := h.client.UpdateIssue(c.Request.Context(), index, req)
	if err != nil {
		log.Error().Err(err).Int("index", index).Msg("failed to update issue")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update issue"})
		return
	}
	c.JSON(http.StatusOK, issue)
}

func (h *Handler) ListLabels(c *gin.Context) {
	labels, err := h.client.ListLabels(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list labels")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list labels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"labels": labels})
}

func (h *Handler) CreateLabel(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" || req.Color == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and color are required"})
		return
	}

	label, err := h.client.CreateLabel(c.Request.Context(), req.Name, req.Color)
	if err != nil {
		log.Error().Err(err).Msg("failed to create label")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create label"})
		return
	}
	c.JSON(http.StatusCreated, label)
}

func (h *Handler) ListComments(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue index"})
		return
	}

	comments, err := h.client.ListComments(c.Request.Context(), index)
	if err != nil {
		log.Error().Err(err).Int("index", index).Msg("failed to list comments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (h *Handler) AddComment(c *gin.Context) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue index"})
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body is required"})
		return
	}

	comment, err := h.client.AddComment(c.Request.Context(), index, req.Body)
	if err != nil {
		log.Error().Err(err).Int("index", index).Msg("failed to add comment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add comment"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}
