package note

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kms-server/internal/search"
)

type Handler struct {
	svc     *Service
	indexer *search.Indexer
}

func NewHandler(svc *Service, indexer *search.Indexer) *Handler {
	return &Handler{svc: svc, indexer: indexer}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	notes := r.Group("/notes")
	{
		notes.GET("", h.List)
		notes.POST("", h.Create)
		notes.GET("/*path", h.Get)
		notes.PUT("/*path", h.Update)
		notes.DELETE("/*path", h.Delete)
	}
	r.GET("/history/*path", h.History)
}

func (h *Handler) List(c *gin.Context) {
	dir := c.Query("dir")
	notes, err := h.svc.List(c.Request.Context(), dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Auto-index
	h.indexNote(n)
	c.JSON(http.StatusCreated, n)
}

func (h *Handler) Get(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:] // remove leading /
	note, err := h.svc.Get(c.Request.Context(), path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.JSON(http.StatusOK, note)
}

func (h *Handler) Update(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:]
	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	n, err := h.svc.Update(c.Request.Context(), path, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Auto-index
	h.indexNote(n)
	c.JSON(http.StatusOK, n)
}

func (h *Handler) Delete(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:]
	if err := h.svc.Delete(c.Request.Context(), path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) History(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:] // remove leading /
	commits, err := h.svc.GetHistory(c.Request.Context(), path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commits": commits})
}

func (h *Handler) indexNote(n *Note) {
	if h.indexer == nil || n == nil {
		return
	}
	idx := &search.IndexedNote{
		ID:      n.ID,
		Path:    n.Path,
		Title:   n.Title,
		Content: n.Content,
		Type:    string(n.Type),
		Status:  string(n.Status),
		Tags:    n.Tags,
		Summary: n.Summary,
		Source:  n.Source,
		SHA:     n.SHA,
		Created: n.Created,
		Updated: n.Updated,
	}
	if err := h.indexer.UpsertNote(idx); err != nil {
		// Log but don't fail the request
		return
	}
}
