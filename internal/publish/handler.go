package publish

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"kms-server/internal/note"
)

type Handler struct {
	store   *Store
	noteSvc *note.Service
}

func NewHandler(store *Store, noteSvc *note.Service) *Handler {
	return &Handler{store: store, noteSvc: noteSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	pub := r.Group("/published")
	{
		pub.GET("", h.List)
		pub.GET("/:slug", h.Get)
	}
	// Publish/unpublish — use *path to match paths with slashes
	r.GET("/publish/*path", h.Check)
	r.POST("/publish/*path", h.Publish)
	r.DELETE("/publish/*path", h.Unpublish)
}

// List returns all published notes. Public.
func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.store.List(limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list published")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": items, "total": total})
}

// Check returns the publish status of a note. Public.
func (h *Handler) Check(c *gin.Context) {
	notePath := c.Param("path")
	if notePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	notePath = notePath[1:] // remove leading /

	pub, err := h.store.GetByNotePath(notePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if pub == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"slug": pub.Slug})
}

// Get returns a single published note with full content. Public.
func (h *Handler) Get(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug required"})
		return
	}

	pub, err := h.store.GetBySlug(slug)
	if err != nil {
		log.Error().Err(err).Str("slug", slug).Msg("failed to get published")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if pub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Fetch full note content from Gitea
	n, err := h.noteSvc.Get(c.Request.Context(), pub.NotePath)
	if err != nil {
		log.Error().Err(err).Str("path", pub.NotePath).Msg("failed to get note content")
		c.JSON(http.StatusNotFound, gin.H{"error": "note content not available"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"slug":         pub.Slug,
		"title":        pub.Title,
		"summary":      pub.Summary,
		"tags":         pub.Tags,
		"username":     pub.Username,
		"nickname":     pub.Nickname,
		"content":      n.Content,
		"published_at": pub.PublishedAt,
		"updated_at":   n.Updated,
	})
}

// Publish makes a note publicly accessible. Auth required.
func (h *Handler) Publish(c *gin.Context) {
	notePath := c.Param("path")
	if notePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	notePath = notePath[1:] // remove leading /

	// Must be authenticated
	username, exists := c.Get("username")
	if !exists || username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	nickname, _ := c.Get("nickname")

	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug required"})
		return
	}
	// Sanitize slug
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, slug)
	// Collapse consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid slug"})
		return
	}

	// Check slug uniqueness (allow same note to re-publish with same slug)
	existing, _ := h.store.GetBySlug(slug)
	if existing != nil && existing.NotePath != notePath {
		c.JSON(http.StatusConflict, gin.H{"error": "slug already taken"})
		return
	}

	// Get note metadata
	n, err := h.noteSvc.Get(c.Request.Context(), notePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}

	nick := ""
	if nickname != nil {
		nick = nickname.(string)
	}
	if err := h.store.Publish(slug, notePath, username.(string), nick, n.Title, n.Summary, n.Tags); err != nil {
		log.Error().Err(err).Msg("failed to publish note")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"slug": slug, "url": "/p/" + slug})
}

// Unpublish removes a note from public access. Auth required.
func (h *Handler) Unpublish(c *gin.Context) {
	notePath := c.Param("path")
	if notePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	notePath = notePath[1:] // remove leading /

	username, exists := c.Get("username")
	if !exists || username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}

	// Verify ownership
	pub, _ := h.store.GetByNotePath(notePath)
	if pub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not published"})
		return
	}
	if pub.Username != username.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your note"})
		return
	}

	if err := h.store.Unpublish(notePath); err != nil {
		log.Error().Err(err).Msg("failed to unpublish")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unpublished"})
}
